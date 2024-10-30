// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build goexperiment.swissmap

package maps

import (
	"internal/abi"
	"internal/asan"
	"internal/msan"
	//"internal/runtime/sys"
	"unsafe"
)

// Functions below pushed from runtime.

//go:linkname mapKeyError
func mapKeyError(typ *abi.SwissMapType, p unsafe.Pointer) error

// Pushed from runtime in order to use runtime.plainError
//
//go:linkname errNilAssign
var errNilAssign error

// Pull from runtime. It is important that is this the exact same copy as the
// runtime because runtime.mapaccess1_fat compares the returned pointer with
// &runtime.zeroVal[0].
// TODO: move zeroVal to internal/abi?
//
//go:linkname zeroVal runtime.zeroVal
var zeroVal [abi.ZeroValSize]byte

// mapaccess1 returns a pointer to h[key].  Never returns nil, instead
// it will return a reference to the zero object for the elem type if
// the key is not in the map.
// NOTE: The returned pointer may keep the whole map live, so don't
// hold onto it for very long.
//
//go:linkname runtime_mapaccess1 runtime.mapaccess1
func runtime_mapaccess1(typ *abi.SwissMapType, m *Map, key unsafe.Pointer) unsafe.Pointer {
	// TODO: concurrent checks.
	//if raceenabled && m != nil {
	//	callerpc := sys.GetCallerPC()
	//	pc := abi.FuncPCABIInternal(mapaccess1)
	//	racereadpc(unsafe.Pointer(m), callerpc, pc)
	//	raceReadObjectPC(t.Key, key, callerpc, pc)
	//}
	if msan.Enabled && m != nil {
		msan.Read(key, typ.Key.Size_)
	}
	if asan.Enabled && m != nil {
		asan.Read(key, typ.Key.Size_)
	}

	if m == nil || m.Used() == 0 {
		if err := mapKeyError(typ, key); err != nil {
			panic(err) // see issue 23734
		}
		return unsafe.Pointer(&zeroVal[0])
	}

	hash := typ.Hasher(key, m.seed)

	if m.dirLen <= 0 {
		_, elem, ok := m.getWithKeySmall(typ, hash, key)
		if !ok {
			return unsafe.Pointer(&zeroVal[0])
		}
		return elem
	}

	// Select table.
	idx := m.directoryIndex(hash)
	t := m.directoryAt(idx)

	// Probe table.
	seq := makeProbeSeq(h1(hash), t.groups.lengthMask)
	for ; ; seq = seq.next() {
		g := t.groups.group(typ, seq.offset)

		match := g.ctrls().matchH2(h2(hash))

		for match != 0 {
			i := match.first()

			slotKey := g.key(typ, i)
			if typ.Key.Equal(key, slotKey) {
				return g.elem(typ, i)
			}
			match = match.removeFirst()
		}

		match = g.ctrls().matchEmpty()
		if match != 0 {
			// Finding an empty slot means we've reached the end of
			// the probe sequence.
			return unsafe.Pointer(&zeroVal[0])
		}
	}
}

//go:linkname runtime_mapassign runtime.mapassign
func runtime_mapassign(typ *abi.SwissMapType, m *Map, key unsafe.Pointer) unsafe.Pointer {
	// TODO: concurrent checks.
	if m == nil {
		panic(errNilAssign)
	}
	//if raceenabled {
	//	callerpc := sys.GetCallerPC()
	//	pc := abi.FuncPCABIInternal(mapassign)
	//	racewritepc(unsafe.Pointer(m), callerpc, pc)
	//	raceReadObjectPC(t.Key, key, callerpc, pc)
	//}
	if msan.Enabled {
		msan.Read(key, typ.Key.Size_)
	}
	if asan.Enabled {
		asan.Read(key, typ.Key.Size_)
	}

	hash := typ.Hasher(key, m.seed)

	if m.dirLen == 0 {
		if m.used < abi.SwissMapGroupSlots {
			return m.putSlotSmall(typ, hash, key)
		}

		// Can't fit another entry, grow to full size map.
		m.growToTable(typ)
	}

outer:
	for {
		// Select table.
		idx := m.directoryIndex(hash)
		t := m.directoryAt(idx)

		seq := makeProbeSeq(h1(hash), t.groups.lengthMask)

		// As we look for a match, keep track of the first deleted slot
		// we find, which we'll use to insert the new entry if
		// necessary.
		var firstDeletedGroup groupReference
		var firstDeletedSlot uint32

		for ; ; seq = seq.next() {
			g := t.groups.group(typ, seq.offset)
			match := g.ctrls().matchH2(h2(hash))

			// Look for an existing slot containing this key.
			for match != 0 {
				i := match.first()

				slotKey := g.key(typ, i)
				if typ.Key.Equal(key, slotKey) {
					if typ.NeedKeyUpdate() {
						typedmemmove(typ.Key, slotKey, key)
					}

					slotElem := g.elem(typ, i)

					t.checkInvariants(typ)
					return slotElem
				}
				match = match.removeFirst()
			}

			// No existing slot for this key in this group. Is this the end
			// of the probe sequence?
			match = g.ctrls().matchEmpty()
			if match != 0 {
				// Finding an empty slot means we've reached the end of
				// the probe sequence.

				var i uint32

				// If we found a deleted slot along the way, we
				// can replace it without consuming growthLeft.
				if firstDeletedGroup.data != nil {
					g = firstDeletedGroup
					i = firstDeletedSlot
					t.growthLeft++ // will be decremented below to become a no-op.
				} else {
					// Otherwise, use the empty slot.
					i = match.first()
				}

				// If there is room left to grow, just insert the new entry.
				if t.growthLeft > 0 {
					slotKey := g.key(typ, i)
					typedmemmove(typ.Key, slotKey, key)
					slotElem := g.elem(typ, i)

					g.ctrls().set(i, ctrl(h2(hash)))
					t.growthLeft--
					t.used++
					m.used++

					t.checkInvariants(typ)
					return slotElem
				}

				t.rehash(typ, m)
				continue outer
			}

			// No empty slots in this group. Check for a deleted
			// slot, which we'll use if we don't find a match later
			// in the probe sequence.
			//
			// We only need to remember a single deleted slot.
			if firstDeletedGroup.data == nil {
				// Since we already checked for empty slots
				// above, matches here must be deleted slots.
				match = g.ctrls().matchEmptyOrDeleted()
				if match != 0 {
					firstDeletedGroup = g
					firstDeletedSlot = match.first()
				}
			}
		}
	}
}
