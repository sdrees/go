// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reflectdata

import (
	"cmd/compile/internal/base"
	"cmd/compile/internal/ir"
	"cmd/compile/internal/rttype"
	"cmd/compile/internal/types"
	"cmd/internal/obj"
	"cmd/internal/objabi"
	"cmd/internal/src"
	"internal/abi"
)

// SwissMapGroupType makes the map slot group type given the type of the map.
func SwissMapGroupType(t *types.Type) *types.Type {
	if t.MapType().SwissGroup != nil {
		return t.MapType().SwissGroup
	}

	// Builds a type representing a group structure for the given map type.
	// This type is not visible to users, we include it so we can generate
	// a correct GC program for it.
	//
	// Make sure this stays in sync with internal/runtime/maps/group.go.
	//
	// type group struct {
	//     ctrl uint64
	//     slots [abi.SwissMapGroupSlots]struct {
	//         key  keyType
	//         elem elemType
	//     }
	// }
	slotFields := []*types.Field{
		makefield("key", t.Key()),
		makefield("elem", t.Elem()),
	}
	slot := types.NewStruct(slotFields)
	slot.SetNoalg(true)

	slotArr := types.NewArray(slot, abi.SwissMapGroupSlots)
	slotArr.SetNoalg(true)

	fields := []*types.Field{
		makefield("ctrl", types.Types[types.TUINT64]),
		makefield("slots", slotArr),
	}

	group := types.NewStruct(fields)
	group.SetNoalg(true)
	types.CalcSize(group)

	// Check invariants that map code depends on.
	if !types.IsComparable(t.Key()) {
		base.Fatalf("unsupported map key type for %v", t)
	}
	if group.Size() <= 8 {
		// internal/runtime/maps creates pointers to slots, even if
		// both key and elem are size zero. In this case, each slot is
		// size 0, but group should still reserve a word of padding at
		// the end to ensure pointers are valid.
		base.Fatalf("bad group size for %v", t)
	}

	t.MapType().SwissGroup = group
	group.StructType().Map = t
	return group
}

var cachedSwissTableType *types.Type

// swissTableType returns a type interchangeable with internal/runtime/maps.table.
// Make sure this stays in sync with internal/runtime/maps/table.go.
func swissTableType() *types.Type {
	if cachedSwissTableType != nil {
		return cachedSwissTableType
	}

	// type table struct {
	//     used       uint16
	//     capacity   uint16
	//     growthLeft uint16
	//     localDepth uint8
	//     // N.B Padding
	//
	//     typ  unsafe.Pointer // *abi.SwissMapType
	//     seed uintptr
	//
	//     index int
	//
	//     // From groups.
	//     groups_typ        unsafe.Pointer // *abi.SwissMapType
	//     groups_data       unsafe.Pointer
	//     groups_lengthMask uint64
	// }
	// must match internal/runtime/maps/table.go:table.
	fields := []*types.Field{
		makefield("used", types.Types[types.TUINT16]),
		makefield("capacity", types.Types[types.TUINT16]),
		makefield("growthLeft", types.Types[types.TUINT16]),
		makefield("localDepth", types.Types[types.TUINT8]),
		makefield("typ", types.Types[types.TUNSAFEPTR]),
		makefield("seed", types.Types[types.TUINTPTR]),
		makefield("index", types.Types[types.TINT]),
		makefield("groups_typ", types.Types[types.TUNSAFEPTR]),
		makefield("groups_data", types.Types[types.TUNSAFEPTR]),
		makefield("groups_lengthMask", types.Types[types.TUINT64]),
	}

	n := ir.NewDeclNameAt(src.NoXPos, ir.OTYPE, ir.Pkgs.InternalMaps.Lookup("table"))
	table := types.NewNamed(n)
	n.SetType(table)
	n.SetTypecheck(1)

	table.SetUnderlying(types.NewStruct(fields))
	types.CalcSize(table)

	// The size of table should be 56 bytes on 64 bit
	// and 36 bytes on 32 bit platforms.
	if size := int64(3*2 + 2*1 /* one extra for padding */ + 1*8 + 5*types.PtrSize); table.Size() != size {
		base.Fatalf("internal/runtime/maps.table size not correct: got %d, want %d", table.Size(), size)
	}

	cachedSwissTableType = table
	return table
}

var cachedSwissMapType *types.Type

// SwissMapType returns a type interchangeable with internal/runtime/maps.Map.
// Make sure this stays in sync with internal/runtime/maps/map.go.
func SwissMapType() *types.Type {
	if cachedSwissMapType != nil {
		return cachedSwissMapType
	}

	// type Map struct {
	//     used uint64
	//     typ  unsafe.Pointer // *abi.SwissMapType
	//     seed uintptr
	//
	//     directory []*table
	//
	//     globalDepth uint8
	//     // N.B Padding
	//
	//     clearSeq uint64
	// }
	// must match internal/runtime/maps/map.go:Map.
	fields := []*types.Field{
		makefield("used", types.Types[types.TUINT64]),
		makefield("typ", types.Types[types.TUNSAFEPTR]),
		makefield("seed", types.Types[types.TUINTPTR]),
		makefield("directory", types.NewSlice(types.NewPtr(swissTableType()))),
		makefield("globalDepth", types.Types[types.TUINT8]),
		makefield("clearSeq", types.Types[types.TUINT64]),
	}

	n := ir.NewDeclNameAt(src.NoXPos, ir.OTYPE, ir.Pkgs.InternalMaps.Lookup("Map"))
	m := types.NewNamed(n)
	n.SetType(m)
	n.SetTypecheck(1)

	m.SetUnderlying(types.NewStruct(fields))
	types.CalcSize(m)

	// The size of Map should be 64 bytes on 64 bit
	// and 40 bytes on 32 bit platforms.
	if size := int64(2*8 + 6*types.PtrSize); m.Size() != size {
		base.Fatalf("internal/runtime/maps.Map size not correct: got %d, want %d", m.Size(), size)
	}

	cachedSwissMapType = m
	return m
}

var cachedSwissIterType *types.Type

// SwissMapIterType returns a type interchangeable with runtime.hiter.
// Make sure this stays in sync with runtime/map.go.
func SwissMapIterType() *types.Type {
	if cachedSwissIterType != nil {
		return cachedSwissIterType
	}

	// type Iter struct {
	//    key  unsafe.Pointer // *Key
	//    elem unsafe.Pointer // *Elem
	//    typ  unsafe.Pointer // *SwissMapType
	//    m    *Map
	//
	//    groupSlotOffset uint64
	//    dirOffset       uint64
	//
	//    clearSeq uint64
	//
	//    globalDepth uint8
	//    // N.B. padding
	//
	//    dirIdx int
	//
	//    tab *table
	//
	//    groupIdx uint64
	//    slotIdx  uint32
	//
	//    // 4 bytes of padding on 64-bit arches.
	// }
	// must match internal/runtime/maps/table.go:Iter.
	fields := []*types.Field{
		makefield("key", types.Types[types.TUNSAFEPTR]),  // Used in range.go for TMAP.
		makefield("elem", types.Types[types.TUNSAFEPTR]), // Used in range.go for TMAP.
		makefield("typ", types.Types[types.TUNSAFEPTR]),
		makefield("m", types.NewPtr(SwissMapType())),
		makefield("groupSlotOffset", types.Types[types.TUINT64]),
		makefield("dirOffset", types.Types[types.TUINT64]),
		makefield("clearSeq", types.Types[types.TUINT64]),
		makefield("globalDepth", types.Types[types.TUINT8]),
		makefield("dirIdx", types.Types[types.TINT]),
		makefield("tab", types.NewPtr(swissTableType())),
		makefield("groupIdx", types.Types[types.TUINT64]),
		makefield("slotIdx", types.Types[types.TUINT32]),
	}

	// build iterator struct hswissing the above fields
	n := ir.NewDeclNameAt(src.NoXPos, ir.OTYPE, ir.Pkgs.InternalMaps.Lookup("Iter"))
	iter := types.NewNamed(n)
	n.SetType(iter)
	n.SetTypecheck(1)

	iter.SetUnderlying(types.NewStruct(fields))
	types.CalcSize(iter)
	want := 7*types.PtrSize + 4*8 + 1*4
	if types.PtrSize == 8 {
		want += 4 // tailing padding
	}
	if iter.Size() != int64(want) {
		base.Fatalf("internal/runtime/maps.Iter size not correct: got %d, want %d", iter.Size(), want)
	}

	cachedSwissIterType = iter
	return iter
}

func writeSwissMapType(t *types.Type, lsym *obj.LSym, c rttype.Cursor) {
	// internal/abi.SwissMapType
	gtyp := SwissMapGroupType(t)
	s1 := writeType(t.Key())
	s2 := writeType(t.Elem())
	s3 := writeType(gtyp)
	hasher := genhash(t.Key())

	slotTyp := gtyp.Field(1).Type.Elem()
	elemOff := slotTyp.Field(1).Offset

	c.Field("Key").WritePtr(s1)
	c.Field("Elem").WritePtr(s2)
	c.Field("Group").WritePtr(s3)
	c.Field("Hasher").WritePtr(hasher)
	c.Field("SlotSize").WriteUintptr(uint64(slotTyp.Size()))
	c.Field("ElemOff").WriteUintptr(uint64(elemOff))
	var flags uint32
	if needkeyupdate(t.Key()) {
		flags |= abi.SwissMapNeedKeyUpdate
	}
	if hashMightPanic(t.Key()) {
		flags |= abi.SwissMapHashMightPanic
	}
	c.Field("Flags").WriteUint32(flags)

	if u := t.Underlying(); u != t {
		// If t is a named map type, also keep the underlying map
		// type live in the binary. This is important to make sure that
		// a named map and that same map cast to its underlying type via
		// reflection, use the same hash function. See issue 37716.
		r := obj.Addrel(lsym)
		r.Sym = writeType(u)
		r.Type = objabi.R_KEEP
	}
}
