package stats

import (
	"fmt"
	"math"
	"sync"
	"testing"
)

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestStore_Increment(t *testing.T) {
	s := New()
	s.Increment("a", 1)
	s.Increment("a", 2.5)
	s.Increment("b", 10)
	s.Increment("a", -0.5)

	all := s.GetAll()
	if got := all["a"]; got != 3 {
		t.Errorf("a = %v, want 3", got)
	}
	if got := all["b"]; got != 10 {
		t.Errorf("b = %v, want 10", got)
	}
	if _, ok := all["c"]; ok {
		t.Error("c should be absent, was never incremented")
	}
}

func TestStore_Set(t *testing.T) {
	s := New()
	s.Increment("g", 5)
	s.Set("g", 100)

	all := s.GetAll()
	if got := all["g"]; got != 100 {
		t.Errorf("g = %v, want 100 (Set should overwrite)", got)
	}

	// Increment after Set resumes from the set value (shared metrics map).
	s.Increment("g", 1)
	all = s.GetAll()
	if got := all["g"]; got != 101 {
		t.Errorf("g = %v, want 101", got)
	}
}

func TestStore_Observe_Summary(t *testing.T) {
	s := New()
	// 1..10
	for i := 1; i <= 10; i++ {
		s.Observe("d", float64(i))
	}

	all := s.GetAll()
	if got := all["d_count"]; got != 10 {
		t.Errorf("d_count = %v, want 10", got)
	}
	// Note: Go's GetAll also emits d_sum; the TS reference (createStatsStore
	// in src/context/stats.tsx) does not expose a sum key. Locking Go's
	// current behavior here — see TASKS.md TEST-09.
	if got := all["d_sum"]; got != 55 {
		t.Errorf("d_sum = %v, want 55", got)
	}
	if got := all["d_min"]; got != 1 {
		t.Errorf("d_min = %v, want 1", got)
	}
	if got := all["d_max"]; got != 10 {
		t.Errorf("d_max = %v, want 10", got)
	}
	if got := all["d_avg"]; got != 5.5 {
		t.Errorf("d_avg = %v, want 5.5", got)
	}
	// sorted reservoir is exactly [1..10]; hand-computed percentiles.
	// (float tolerance: percentile() interpolation introduces rounding error)
	if got := all["d_p50"]; !almostEqual(got, 5.5) {
		t.Errorf("d_p50 = %v, want 5.5", got)
	}
	if got := all["d_p95"]; !almostEqual(got, 9.55) {
		t.Errorf("d_p95 = %v, want 9.55", got)
	}
	if got := all["d_p99"]; !almostEqual(got, 9.91) {
		t.Errorf("d_p99 = %v, want 9.91", got)
	}
}

func TestStore_Observe_Single(t *testing.T) {
	s := New()
	s.Observe("solo", 42)

	all := s.GetAll()
	for _, key := range []string{"solo_min", "solo_max", "solo_avg", "solo_p50", "solo_p95", "solo_p99"} {
		if got := all[key]; got != 42 {
			t.Errorf("%s = %v, want 42 (single observation)", key, got)
		}
	}
}

func TestStore_Observe_Negative(t *testing.T) {
	s := New()
	s.Observe("n", -5)
	s.Observe("n", -1)
	s.Observe("n", -10)

	all := s.GetAll()
	if got := all["n_min"]; got != -10 {
		t.Errorf("n_min = %v, want -10", got)
	}
	if got := all["n_max"]; got != -1 {
		t.Errorf("n_max = %v, want -1", got)
	}
}

func TestStore_Observe_Reservoir(t *testing.T) {
	s := New()
	const n = ReservoirSize + 500

	observed := make(map[float64]bool, n)
	var sum float64
	for i := 0; i < n; i++ {
		v := float64(i)
		observed[v] = true
		sum += v
		s.Observe("r", v)
	}

	all := s.GetAll()
	if got := all["r_count"]; got != float64(n) {
		t.Errorf("r_count = %v, want %d (count must not be capped by reservoir size)", got, n)
	}
	if got := all["r_sum"]; got != sum {
		t.Errorf("r_sum = %v, want %v", got, sum)
	}
	if got := all["r_min"]; got != 0 {
		t.Errorf("r_min = %v, want 0", got)
	}
	if got := all["r_max"]; got != float64(n-1) {
		t.Errorf("r_max = %v, want %v", got, float64(n-1))
	}

	s.mu.Lock()
	h := s.histograms["r"]
	reservoirLen := len(h.Reservoir)
	reservoirCopy := make([]float64, len(h.Reservoir))
	copy(reservoirCopy, h.Reservoir)
	s.mu.Unlock()

	if reservoirLen != ReservoirSize {
		t.Fatalf("reservoir length = %d, want %d", reservoirLen, ReservoirSize)
	}
	// Randomized sampling: assert invariants, not exact contents.
	for _, v := range reservoirCopy {
		if !observed[v] {
			t.Errorf("reservoir contains value %v that was never observed", v)
		}
	}
}

func TestStore_Add(t *testing.T) {
	s := New()
	s.Add("tags", "x")
	s.Add("tags", "y")
	s.Add("tags", "x") // duplicate
	s.Add("tags", "")  // empty string counts as a value
	s.Add("other", "z")

	all := s.GetAll()
	// Note: Go's GetAll emits sets as "<name>_unique"; the TS reference
	// (createStatsStore) emits the plain "<name>" key. Locking Go's current
	// behavior here — see TASKS.md TEST-09.
	if got := all["tags_unique"]; got != 3 {
		t.Errorf("tags_unique = %v, want 3 (x, y, \"\")", got)
	}
	if got := all["other_unique"]; got != 1 {
		t.Errorf("other_unique = %v, want 1", got)
	}
}

func TestPercentile(t *testing.T) {
	tests := []struct {
		name   string
		sorted []float64
		p      float64
		want   float64
	}{
		{"empty", nil, 50, 0},
		{"single element", []float64{7}, 50, 7},
		{"single element p0", []float64{7}, 0, 7},
		{"single element p100", []float64{7}, 100, 7},
		{"p0 returns first", []float64{1, 2, 3, 4, 5}, 0, 1},
		{"p100 returns last (upper>=len guard)", []float64{1, 2, 3, 4, 5}, 100, 5},
		{"interpolation branch", []float64{1, 2, 3, 4, 5}, 50, 3},
		// index = 30/100*(5-1) = 1.2 -> lower=1 upper=2 -> 2 + (3-2)*0.2 = 2.2
		{"interpolation fractional", []float64{1, 2, 3, 4, 5}, 30, 2.2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := percentile(tt.sorted, tt.p)
			if !almostEqual(got, tt.want) {
				t.Errorf("percentile(%v, %v) = %v, want %v", tt.sorted, tt.p, got, tt.want)
			}
		})
	}
}

func TestStore_GetAll_Isolation(t *testing.T) {
	s := New()

	// Fresh store returns an empty, non-nil map.
	fresh := s.GetAll()
	if fresh == nil {
		t.Fatal("GetAll() on fresh store returned nil, want empty map")
	}
	if len(fresh) != 0 {
		t.Errorf("GetAll() on fresh store returned %d entries, want 0", len(fresh))
	}

	s.Increment("k", 1)

	first := s.GetAll()
	first["k"] = 999 // mutate returned map
	first["injected"] = 1

	second := s.GetAll()
	if got := second["k"]; got != 1 {
		t.Errorf("k = %v after mutating a prior GetAll() result, want 1 (store must be isolated)", got)
	}
	if _, ok := second["injected"]; ok {
		t.Error("mutation of a prior GetAll() result leaked into the store")
	}
}

func TestStore_Concurrent(t *testing.T) {
	s := New()
	const goroutines = 50
	const opsPerGoroutine = 200

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				s.Increment("counter", 1)
				s.Set(fmt.Sprintf("gauge-%d", id), float64(i))
				s.Observe("hist", float64(i))
				s.Add("set", fmt.Sprintf("v-%d", i%10))
				s.GetAll()
			}
		}(g)
	}
	wg.Wait()

	all := s.GetAll()
	want := float64(goroutines * opsPerGoroutine)
	if got := all["counter"]; got != want {
		t.Errorf("counter = %v, want %v", got, want)
	}
	if got := all["hist_count"]; got != want {
		t.Errorf("hist_count = %v, want %v", got, want)
	}
	if got := all["set_unique"]; got != 10 {
		t.Errorf("set_unique = %v, want 10", got)
	}
}
