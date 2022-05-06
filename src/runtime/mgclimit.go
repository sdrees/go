// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import "runtime/internal/atomic"

// gcCPULimiter is a mechanism to limit GC CPU utilization in situations
// where it might become excessive and inhibit application progress (e.g.
// a death spiral).
//
// The core of the limiter is a leaky bucket mechanism that fills with GC
// CPU time and drains with mutator time. Because the bucket fills and
// drains with time directly (i.e. without any weighting), this effectively
// sets a very conservative limit of 50%. This limit could be enforced directly,
// however, but the purpose of the bucket is to accomodate spikes in GC CPU
// utilization without hurting throughput.
//
// Note that the bucket in the leaky bucket mechanism can never go negative,
// so the GC never gets credit for a lot of CPU time spent without the GC
// running. This is intentional, as an application that stays idle for, say,
// an entire day, could build up enough credit to fail to prevent a death
// spiral the following day. The bucket's capacity is the GC's only leeway.
//
// The capacity thus also sets the window the limiter considers. For example,
// if the capacity of the bucket is 1 cpu-second, then the limiter will not
// kick in until at least 1 full cpu-second in the last 2 cpu-second window
// is spent on GC CPU time.
var gcCPULimiter gcCPULimiterState

type gcCPULimiterState struct {
	lock atomic.Uint32

	enabled atomic.Bool
	bucket  struct {
		// Invariants:
		// - fill >= 0
		// - capacity >= 0
		// - fill <= capacity
		fill, capacity uint64
	}
	// TODO(mknyszek): Export this as a runtime/metric to provide an estimate of
	// how much GC work is being dropped on the floor.
	overflow uint64

	// gcEnabled is an internal copy of gcBlackenEnabled that determines
	// whether the limiter tracks total assist time.
	//
	// gcBlackenEnabled isn't used directly so as to keep this structure
	// unit-testable.
	gcEnabled bool

	// transitioning is true when the GC is in a STW and transitioning between
	// the mark and sweep phases.
	transitioning bool

	// lastTotalAssistTime is the last value of a monotonically increasing
	// count of GC assist time, like gcController.assistTime.
	lastTotalAssistTime int64

	_ uint32 // Align lastUpdate on 32-bit platforms.

	// lastUpdate is the nanotime timestamp of the last time update was called.
	//
	// Updated under lock, but may be read concurrently.
	lastUpdate atomic.Int64

	// nprocs is an internal copy of gomaxprocs, used to determine total available
	// CPU time.
	//
	// gomaxprocs isn't used directly so as to keep this structure unit-testable.
	nprocs int32
}

// limiting returns true if the CPU limiter is currently enabled, meaning the Go GC
// should take action to limit CPU utilization.
//
// It is safe to call concurrently with other operations.
func (l *gcCPULimiterState) limiting() bool {
	return l.enabled.Load()
}

// startGCTransition notifies the limiter of a GC transition. totalAssistTime
// is the same as described for update. now must be the start of the STW pause
// for the GC transition.
//
// This call takes ownership of the limiter and disables all other means of
// updating the limiter. Release ownership by calling finishGCTransition.
//
// It is safe to call concurrently with other operations.
func (l *gcCPULimiterState) startGCTransition(enableGC bool, totalAssistTime, now int64) {
	if !l.tryLock() {
		// This must happen during a STW, so we can't fail to acquire the lock.
		// If we did, something went wrong. Throw.
		throw("failed to acquire lock to start a GC transition")
	}
	if l.gcEnabled == enableGC {
		throw("transitioning GC to the same state as before?")
	}
	// Flush whatever was left between the last update and now.
	l.updateLocked(totalAssistTime, now)
	if enableGC && totalAssistTime != 0 {
		throw("assist time must be zero on entry to a GC cycle")
	}
	l.gcEnabled = enableGC
	l.transitioning = true
	// N.B. finishGCTransition releases the lock.
	//
	// We don't release here to increase the chance that if there's a failure
	// to finish the transition, that we throw on failing to acquire the lock.
}

// finishGCTransition notifies the limiter that the GC transition is complete
// and releases ownership of it. It also accumulates STW time in the bucket.
// now must be the timestamp from the end of the STW pause.
func (l *gcCPULimiterState) finishGCTransition(now int64) {
	if !l.transitioning {
		throw("finishGCTransition called without starting one?")
	}
	// Count the full nprocs set of CPU time because the world is stopped
	// between startGCTransition and finishGCTransition. Even though the GC
	// isn't running on all CPUs, it is preventing user code from doing so,
	// so it might as well be.
	if lastUpdate := l.lastUpdate.Load(); now >= lastUpdate {
		l.accumulate(0, (now-lastUpdate)*int64(l.nprocs))
	}
	l.lastUpdate.Store(now)
	l.transitioning = false
	// Reset lastTotalAssistTime for the next GC cycle.
	l.lastTotalAssistTime = 0
	l.unlock()
}

// gcCPULimiterUpdatePeriod dictates the maximum amount of wall-clock time
// we can go before updating the limiter.
const gcCPULimiterUpdatePeriod = 10e6 // 10ms

// needUpdate returns true if the limiter's maximum update period has been
// exceeded, and so would benefit from an update.
func (l *gcCPULimiterState) needUpdate(now int64) bool {
	return now-l.lastUpdate.Load() > gcCPULimiterUpdatePeriod
}

// update updates the bucket given runtime-specific information. totalAssistTime must
// be a value that increases monotonically throughout the GC cycle, and is reset
// at the start of a new mark phase. now is the current monotonic time in nanoseconds.
//
// This is safe to call concurrently with other operations, except *GCTransition.
func (l *gcCPULimiterState) update(totalAssistTime int64, now int64) {
	if !l.tryLock() {
		// We failed to acquire the lock, which means something else is currently
		// updating. Just drop our update, the next one to update will include
		// our total assist time.
		return
	}
	if l.transitioning {
		throw("update during transition")
	}
	l.updateLocked(totalAssistTime, now)
	l.unlock()
}

// updatedLocked is the implementation of update. l.lock must be held.
func (l *gcCPULimiterState) updateLocked(totalAssistTime int64, now int64) {
	lastUpdate := l.lastUpdate.Load()
	if now < lastUpdate || totalAssistTime < l.lastTotalAssistTime {
		// Defensively avoid overflow. This isn't even the latest update anyway.
		// This might seem like a lot to back out on, but provided that both
		// totalAssistTime and now are fresh, updaters must've been closely
		// racing. It's close enough that it doesn't matter, and in the long
		// term the result is the same.
		return
	}
	windowTotalTime := (now - lastUpdate) * int64(l.nprocs)
	l.lastUpdate.Store(now)
	if !l.gcEnabled {
		l.accumulate(windowTotalTime, 0)
		return
	}
	windowGCTime := totalAssistTime - l.lastTotalAssistTime
	windowGCTime += int64(float64(windowTotalTime) * gcBackgroundUtilization)
	l.accumulate(windowTotalTime-windowGCTime, windowGCTime)
	l.lastTotalAssistTime = totalAssistTime
}

// accumulate adds time to the bucket and signals whether the limiter is enabled.
//
// This is an internal function that deals just with the bucket. Prefer update.
// l.lock must be held.
func (l *gcCPULimiterState) accumulate(mutatorTime, gcTime int64) {
	headroom := l.bucket.capacity - l.bucket.fill
	enabled := headroom == 0

	// Let's be careful about three things here:
	// 1. The addition and subtraction, for the invariants.
	// 2. Overflow.
	// 3. Excessive mutation of l.enabled, which is accessed
	//    by all assists, potentially more than once.
	change := gcTime - mutatorTime

	// Handle limiting case.
	if change > 0 && headroom <= uint64(change) {
		l.overflow += uint64(change) - headroom
		l.bucket.fill = l.bucket.capacity
		if !enabled {
			l.enabled.Store(true)
		}
		return
	}

	// Handle non-limiting cases.
	if change < 0 && l.bucket.fill <= uint64(-change) {
		// Bucket emptied.
		l.bucket.fill = 0
	} else {
		// All other cases.
		l.bucket.fill -= uint64(-change)
	}
	if change != 0 && enabled {
		l.enabled.Store(false)
	}
}

// tryLock attempts to lock l. Returns true on success.
func (l *gcCPULimiterState) tryLock() bool {
	return l.lock.CompareAndSwap(0, 1)
}

// unlock releases the lock on l. Must be called if tryLock returns true.
func (l *gcCPULimiterState) unlock() {
	old := l.lock.Swap(0)
	if old != 1 {
		throw("double unlock")
	}
}

// capacityPerProc is the limiter's bucket capacity for each P in GOMAXPROCS.
const capacityPerProc = 1e9 // 1 second in nanoseconds

// resetCapacity updates the capacity based on GOMAXPROCS. Must not be called
// while the GC is enabled.
//
// It is safe to call concurrently with other operations.
func (l *gcCPULimiterState) resetCapacity(now int64, nprocs int32) {
	if !l.tryLock() {
		// This must happen during a STW, so we can't fail to acquire the lock.
		// If we did, something went wrong. Throw.
		throw("failed to acquire lock to reset capacity")
	}
	// Flush the rest of the time for this period.
	l.updateLocked(0, now)
	l.nprocs = nprocs

	l.bucket.capacity = uint64(nprocs) * capacityPerProc
	if l.bucket.fill > l.bucket.capacity {
		l.bucket.fill = l.bucket.capacity
		l.enabled.Store(true)
	} else if l.bucket.fill < l.bucket.capacity {
		l.enabled.Store(false)
	}
	l.unlock()
}
