package async

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	testDelay = 20 * time.Millisecond
	testWait  = 500 * time.Millisecond
)

// waitForCount blocks until got() reaches want or testWait elapses, whichever
// comes first, then returns the final observed value.
func waitForCount(got func() int64, want int64) int64 {
	deadline := time.After(testWait)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		if v := got(); v >= want {
			return v
		}
		select {
		case <-ticker.C:
		case <-deadline:
			return got()
		}
	}
}

func TestDebouncer_CallFires(t *testing.T) {
	d := NewDebouncer(testDelay)
	var n int64
	d.Call(func() { atomic.AddInt64(&n, 1) })

	if got := waitForCount(func() int64 { return atomic.LoadInt64(&n) }, 1); got != 1 {
		t.Errorf("n = %v, want 1", got)
	}
}

func TestDebouncer_CallResetsTimer(t *testing.T) {
	d := NewDebouncer(testDelay)
	var n int64
	var lastFired int64

	// Fire several Calls spaced under the delay; each reset should push out
	// the previous timer, so only the last fn should ever run.
	for i := int64(1); i <= 3; i++ {
		i := i
		d.Call(func() {
			atomic.AddInt64(&n, 1)
			atomic.StoreInt64(&lastFired, i)
		})
		time.Sleep(testDelay / 2)
	}

	if got := waitForCount(func() int64 { return atomic.LoadInt64(&n) }, 1); got != 1 {
		t.Errorf("n = %v, want exactly 1 fire", got)
	}
	if got := atomic.LoadInt64(&lastFired); got != 3 {
		t.Errorf("lastFired = %v, want 3 (the last Call)", got)
	}
}

func TestDebouncer_Cancel(t *testing.T) {
	d := NewDebouncer(testDelay)
	var n int64
	d.Call(func() { atomic.AddInt64(&n, 1) })
	d.Cancel()

	time.Sleep(2 * testDelay)
	if got := atomic.LoadInt64(&n); got != 0 {
		t.Errorf("n = %v, want 0 (cancelled before firing)", got)
	}
}

func TestDebouncer_CancelWithoutPending(t *testing.T) {
	d := NewDebouncer(testDelay)
	// No pending Call: should be a no-op, not panic.
	d.Cancel()
	d.Cancel()
}

func TestDebouncer_CallAfterCancel(t *testing.T) {
	d := NewDebouncer(testDelay)
	var n int64
	d.Call(func() { atomic.AddInt64(&n, 1) })
	d.Cancel()

	d.Call(func() { atomic.AddInt64(&n, 1) })
	if got := waitForCount(func() int64 { return atomic.LoadInt64(&n) }, 1); got != 1 {
		t.Errorf("n = %v, want 1 (Call after Cancel should still fire)", got)
	}
}

func TestDebouncer_ConcurrentCalls(t *testing.T) {
	d := NewDebouncer(testDelay)
	var n int64
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d.Call(func() { atomic.AddInt64(&n, 1) })
		}()
	}
	wg.Wait()

	if got := waitForCount(func() int64 { return atomic.LoadInt64(&n) }, 1); got != 1 {
		t.Errorf("n = %v, want exactly 1 fire from concurrent Calls", got)
	}
}

func TestThrottler_FirstAllow(t *testing.T) {
	th := NewThrottler(time.Hour)
	if !th.Allow() {
		t.Error("first Allow() = false, want true (zero-value last)")
	}
}

func TestThrottler_BlocksWithinInterval(t *testing.T) {
	th := NewThrottler(testWait)
	if !th.Allow() {
		t.Fatal("first Allow() = false, want true")
	}
	if th.Allow() {
		t.Error("second immediate Allow() = true, want false (within interval)")
	}
}

func TestThrottler_AllowsAfterInterval(t *testing.T) {
	th := NewThrottler(testDelay)
	if !th.Allow() {
		t.Fatal("first Allow() = false, want true")
	}

	time.Sleep(testDelay + testDelay/2)
	if !th.Allow() {
		t.Error("Allow() after interval elapsed = false, want true")
	}
	if th.Allow() {
		t.Error("immediate follow-up Allow() = true, want false (window re-armed)")
	}
}

func TestThrottler_ZeroInterval(t *testing.T) {
	th := NewThrottler(0)
	for i := 0; i < 5; i++ {
		if !th.Allow() {
			t.Errorf("Allow() call %d = false, want true (zero interval always allows)", i)
		}
	}
}

func TestThrottler_Concurrent(t *testing.T) {
	th := NewThrottler(time.Hour)
	var allowed int64
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if th.Allow() {
				atomic.AddInt64(&allowed, 1)
			}
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt64(&allowed); got != 1 {
		t.Errorf("allowed = %v, want exactly 1", got)
	}
}
