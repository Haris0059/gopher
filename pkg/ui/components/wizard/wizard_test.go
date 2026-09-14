package wizard

import "testing"

func steps(n int) []Step {
	s := make([]Step, n)
	for i := range s {
		s[i] = Step{ID: "step", Title: "Step"}
	}
	return s
}

func TestNew(t *testing.T) {
	w := New(steps(3))
	if w.CurrentIndex() != 0 {
		t.Errorf("CurrentIndex() = %d, want 0", w.CurrentIndex())
	}
	if w.TotalSteps() != 3 {
		t.Errorf("TotalSteps() = %d, want 3", w.TotalSteps())
	}
	if !w.ShowStepCounter {
		t.Error("ShowStepCounter should default to true")
	}
	if !w.IsFirst() {
		t.Error("new wizard should be on the first step")
	}
}

func TestWizard_NextAdvances(t *testing.T) {
	w := New(steps(3))
	if !w.Next() {
		t.Fatal("Next() should return true when not on the last step")
	}
	if w.CurrentIndex() != 1 {
		t.Errorf("CurrentIndex() = %d, want 1", w.CurrentIndex())
	}
	if w.IsCompleted() {
		t.Error("wizard should not be completed after advancing to step 1 of 3")
	}
}

func TestWizard_NextOnLastStepCompletes(t *testing.T) {
	w := New(steps(2))
	w.Next() // -> index 1, last step
	if w.Next() {
		t.Error("Next() on the last step should return false")
	}
	if !w.IsCompleted() {
		t.Error("Next() on the last step should mark the wizard completed")
	}
	if w.CurrentIndex() != 1 {
		t.Errorf("CurrentIndex() should stay at the last step, got %d", w.CurrentIndex())
	}
}

func TestWizard_PrevGoesBack(t *testing.T) {
	w := New(steps(3))
	w.Next()
	if !w.Prev() {
		t.Fatal("Prev() should return true when not on the first step")
	}
	if w.CurrentIndex() != 0 {
		t.Errorf("CurrentIndex() = %d, want 0", w.CurrentIndex())
	}
}

func TestWizard_PrevOnFirstStepCancels(t *testing.T) {
	w := New(steps(3))
	if w.Prev() {
		t.Error("Prev() on the first step with no history should return false")
	}
	if !w.IsCancelled() {
		t.Error("Prev() on the first step with no history should cancel the wizard")
	}
}

func TestWizard_GoToThenPrevReturnsToJumpOrigin(t *testing.T) {
	// Reference behavior (WizardProvider.tsx goToStep/goBack): a jump pushes
	// the step jumped *from* onto history, and Prev pops back to it rather
	// than just decrementing the index.
	w := New(steps(5))
	w.Next() // index 0 -> 1
	w.GoTo(4)
	if w.CurrentIndex() != 4 {
		t.Fatalf("GoTo(4): CurrentIndex() = %d, want 4", w.CurrentIndex())
	}
	if !w.Prev() {
		t.Fatal("Prev() after GoTo should succeed")
	}
	if w.CurrentIndex() != 1 {
		t.Errorf("Prev() after GoTo(4) from index 1 = %d, want 1 (jump origin, not 3)", w.CurrentIndex())
	}
}

func TestWizard_NextPushesHistoryOnlyWhenNonEmpty(t *testing.T) {
	// Reference: goNext only pushes to navigationHistory when it is already
	// non-empty, so a plain forward walk leaves it empty and a later Prev()
	// falls through to the currentStepIndex-1 branch, not a history pop.
	w := New(steps(4))
	w.Next() // 0 -> 1, history still empty
	w.Next() // 1 -> 2, history still empty
	if len(w.history) != 0 {
		t.Fatalf("history = %v, want empty after plain forward walk", w.history)
	}
	w.Prev()
	if w.CurrentIndex() != 1 {
		t.Errorf("CurrentIndex() = %d, want 1 (plain decrement)", w.CurrentIndex())
	}
}

func TestWizard_GoToIgnoresOutOfRange(t *testing.T) {
	w := New(steps(3))
	w.GoTo(-1)
	if w.CurrentIndex() != 0 {
		t.Errorf("GoTo(-1) should be a no-op, got index %d", w.CurrentIndex())
	}
	w.GoTo(99)
	if w.CurrentIndex() != 0 {
		t.Errorf("GoTo(99) should be a no-op, got index %d", w.CurrentIndex())
	}
}

func TestWizard_CancelClearsHistory(t *testing.T) {
	w := New(steps(5))
	w.GoTo(3)
	w.Cancel()
	if !w.IsCancelled() {
		t.Error("Cancel() should mark the wizard cancelled")
	}
	if len(w.history) != 0 {
		t.Errorf("Cancel() should clear navigation history, got %v", w.history)
	}
}

func TestWizard_SetGetData(t *testing.T) {
	w := New(steps(1))
	w.Set("name", "gopher")
	if got := w.Get("name"); got != "gopher" {
		t.Errorf("Get(%q) = %v, want %q", "name", got, "gopher")
	}
	if w.Get("missing") != nil {
		t.Error("Get on a missing key should return nil")
	}
}

func TestWizard_UpdateMerges(t *testing.T) {
	w := New(steps(1))
	w.Set("a", 1)
	w.Update(map[string]any{"a": 2, "b": 3})
	data := w.Data()
	if data["a"] != 2 {
		t.Errorf("Update should overwrite existing keys: a = %v, want 2", data["a"])
	}
	if data["b"] != 3 {
		t.Errorf("Update should add new keys: b = %v, want 3", data["b"])
	}
}

func TestWizard_Progress(t *testing.T) {
	w := New(steps(3))
	if got, want := w.Progress(), "Step 1 of 3"; got != want {
		t.Errorf("Progress() = %q, want %q", got, want)
	}
	w.Next()
	if got, want := w.Progress(), "Step 2 of 3"; got != want {
		t.Errorf("Progress() = %q, want %q", got, want)
	}
}

func TestWizard_TitleWithCounter(t *testing.T) {
	w := New(steps(3))
	w.Title = "Setup"
	if got, want := w.TitleWithCounter(), "Setup (1/3)"; got != want {
		t.Errorf("TitleWithCounter() = %q, want %q", got, want)
	}
	w.ShowStepCounter = false
	if got, want := w.TitleWithCounter(), "Setup"; got != want {
		t.Errorf("TitleWithCounter() with counter hidden = %q, want %q", got, want)
	}
}

func TestWizard_TitleWithCounterDefaultsToWizard(t *testing.T) {
	w := New(steps(1))
	w.ShowStepCounter = false
	if got, want := w.TitleWithCounter(), "Wizard"; got != want {
		t.Errorf("TitleWithCounter() with no title set = %q, want %q", got, want)
	}
}

func TestWizard_CurrentStepOutOfRange(t *testing.T) {
	w := New(nil)
	if got, want := w.CurrentStep(), (Step{}); got != want {
		t.Errorf("CurrentStep() on an empty wizard = %+v, want zero value", got)
	}
}

func TestWizard_IsFirstIsLast(t *testing.T) {
	w := New(steps(2))
	if !w.IsFirst() || w.IsLast() {
		t.Error("wizard should start first-not-last")
	}
	w.Next()
	if w.IsFirst() || !w.IsLast() {
		t.Error("wizard should be last-not-first on its final step")
	}
}
