// Package wizard provides a multi-step form/dialog framework.
// Source: components/wizard/ — WizardProvider.tsx, useWizard.ts, WizardDialogLayout.tsx
package wizard

import (
	"fmt"
	"strconv"
)

// Step represents a single step in a wizard flow.
type Step struct {
	ID    string
	Title string
}

// Wizard tracks multi-step form state.
// Go equivalent of TS WizardProvider React context.
type Wizard struct {
	steps           []Step
	currentStep     int
	history         []int // indices to return to on Prev, pushed by GoTo
	data            map[string]any
	completed       bool
	cancelled       bool
	Title           string
	ShowStepCounter bool
}

// New creates a wizard with the given steps. Step counter display defaults
// to on, matching the reference's showStepCounter default of true.
func New(steps []Step) *Wizard {
	return &Wizard{
		steps:           steps,
		data:            make(map[string]any),
		ShowStepCounter: true,
	}
}

// CurrentStep returns the current step.
func (w *Wizard) CurrentStep() Step {
	if w.currentStep >= len(w.steps) {
		return Step{}
	}
	return w.steps[w.currentStep]
}

// CurrentIndex returns the zero-based step index.
func (w *Wizard) CurrentIndex() int { return w.currentStep }

// TotalSteps returns the number of steps.
func (w *Wizard) TotalSteps() int { return len(w.steps) }

// IsFirst returns true if on the first step.
func (w *Wizard) IsFirst() bool { return w.currentStep == 0 }

// IsLast returns true if on the last step.
func (w *Wizard) IsLast() bool { return w.currentStep == len(w.steps)-1 }

// Next advances to the next step. On the last step it marks the wizard
// completed instead of moving, matching goNext in WizardProvider.tsx. A
// pending GoTo history entry is carried forward only while history is
// already non-empty, so a plain forward walk never grows it.
func (w *Wizard) Next() bool {
	if w.currentStep >= len(w.steps)-1 {
		w.completed = true
		return false
	}
	if len(w.history) > 0 {
		w.history = append(w.history, w.currentStep)
	}
	w.currentStep++
	return true
}

// Prev goes back to the previous step. If a GoTo jump pushed history, it
// pops back to the step that was current before the jump; otherwise it
// steps back by one index. On the first step with no history, it cancels
// the wizard instead — matching goBack in WizardProvider.tsx.
func (w *Wizard) Prev() bool {
	if len(w.history) > 0 {
		last := len(w.history) - 1
		w.currentStep = w.history[last]
		w.history = w.history[:last]
		return true
	}
	if w.currentStep > 0 {
		w.currentStep--
		return true
	}
	w.cancelled = true
	return false
}

// GoTo jumps to a specific step index, pushing the current index onto the
// navigation history so a later Prev returns here.
// Source: WizardProvider.tsx — goToStep
func (w *Wizard) GoTo(index int) {
	if index < 0 || index >= len(w.steps) {
		return
	}
	w.history = append(w.history, w.currentStep)
	w.currentStep = index
}

// Set stores a value in the wizard's data map.
func (w *Wizard) Set(key string, value any) { w.data[key] = value }

// Update merges the given fields into the wizard's data map.
// Source: WizardProvider.tsx — updateWizardData
func (w *Wizard) Update(fields map[string]any) {
	for k, v := range fields {
		w.data[k] = v
	}
}

// Get retrieves a value from the wizard's data map.
func (w *Wizard) Get(key string) any { return w.data[key] }

// Data returns all accumulated wizard data.
func (w *Wizard) Data() map[string]any { return w.data }

// Complete marks the wizard as completed.
func (w *Wizard) Complete() { w.completed = true }

// Cancel marks the wizard as cancelled and clears navigation history.
// Source: WizardProvider.tsx — cancel
func (w *Wizard) Cancel() {
	w.history = nil
	w.cancelled = true
}

// IsCompleted returns true if the wizard finished successfully.
func (w *Wizard) IsCompleted() bool { return w.completed }

// IsCancelled returns true if the wizard was cancelled.
func (w *Wizard) IsCancelled() bool { return w.cancelled }

// Progress returns "Step X of Y" text.
func (w *Wizard) Progress() string {
	return "Step " + strconv.Itoa(w.currentStep+1) + " of " + strconv.Itoa(len(w.steps))
}

// TitleWithCounter renders the dialog title the way WizardDialogLayout.tsx
// does: "<Title> (X/Y)", or just "<Title>" when ShowStepCounter is false.
func (w *Wizard) TitleWithCounter() string {
	title := w.Title
	if title == "" {
		title = "Wizard"
	}
	if !w.ShowStepCounter {
		return title
	}
	return fmt.Sprintf("%s (%d/%d)", title, w.currentStep+1, len(w.steps))
}
