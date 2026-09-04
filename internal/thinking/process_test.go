package thinking

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
)

// TestProcessIsDeterministic: identical inputs give identical results, and an
// unrelated call in between changes nothing.
func TestProcessIsDeterministic(t *testing.T) {
	a := validInput(1)
	b := validInput(7)
	b.Confidence = 0.9

	first := Process(a)
	Process(b) // an unrelated call in between must change nothing
	again := Process(a)

	if !reflect.DeepEqual(first, again) {
		t.Errorf("Process(a) differs across calls:\nfirst=%+v\nagain=%+v", first, again)
	}
	if first.Structured == nil || first.Structured.ThoughtNumber != 1 {
		t.Fatalf("Structured = %+v, want thoughtNumber 1", first.Structured)
	}
}

// TestProcessDoesNotMutateItsArgument: the clamp and the renderer work on
// Process's own copy; the caller's value and its slices are untouched.
func TestProcessDoesNotMutateItsArgument(t *testing.T) {
	td := validInput(9) // ThoughtNumber 9 > TotalThoughts 3 forces the clamp
	td.Assumptions = []string{"zebra", "apple"}
	res := Process(td)
	if res.IsError {
		t.Fatalf("unexpected error: %s", res.Text)
	}
	// By-value receiver today; this assertion guards a future pointer refactor.
	if td.TotalThoughts != 3 {
		t.Errorf("Process clamped the caller's TotalThoughts to %d", td.TotalThoughts)
	}
	if !slices.Equal(td.Assumptions, []string{"zebra", "apple"}) {
		t.Errorf("Process changed the caller's assumptions: %v", td.Assumptions)
	}
}

func TestProcessEchoesRoutingFields(t *testing.T) {
	td := validInput(2)
	td.Confidence = 0.75
	res := Process(td)
	if res.IsError {
		t.Fatalf("unexpected error result: %s", res.Text)
	}
	want := &ThoughtResponse{ThoughtNumber: 2, TotalThoughts: 3, NextThoughtNeeded: true, Confidence: 0.75}
	if !reflect.DeepEqual(res.Structured, want) {
		t.Errorf("Structured = %+v, want %+v", res.Structured, want)
	}
}

func TestProcessClampsTotalThoughts(t *testing.T) {
	td := validInput(5) // TotalThoughts is 3
	res := Process(td)
	if res.IsError {
		t.Fatalf("unexpected error result: %s", res.Text)
	}
	if res.Structured.TotalThoughts != 5 {
		t.Errorf("TotalThoughts = %d, want clamped to 5", res.Structured.TotalThoughts)
	}
	if !strings.Contains(res.Text, "Thought 5 of 5") {
		t.Errorf("header must show the clamped total:\n%s", res.Text)
	}
}

func TestProcessValidationErrorEnvelope(t *testing.T) {
	td := validInput(1)
	td.Critique = ""
	res := Process(td)
	if !res.IsError {
		t.Fatal("expected IsError=true")
	}
	if res.Structured != nil {
		t.Errorf("Structured must be nil on error, got %+v", res.Structured)
	}
	var p errorPayload
	if err := json.Unmarshal([]byte(res.Text), &p); err != nil {
		t.Fatalf("error result is not JSON: %v\n%s", err, res.Text)
	}
	if p.Status != "failed" {
		t.Errorf("status = %q, want failed", p.Status)
	}
	if p.Error != "critique must be a non-blank string" {
		t.Errorf("error = %q", p.Error)
	}
	if p.Hint != requiredFieldsChecklist {
		t.Errorf("hint = %q, want the shared checklist", p.Hint)
	}
}

func TestProcessConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Go(func() {
			td := validInput(i + 1)
			res := Process(td)
			if res.IsError || res.Structured.ThoughtNumber != i+1 {
				t.Errorf("call %d: IsError=%v Structured=%+v", i+1, res.IsError, res.Structured)
			}
		})
	}
	wg.Wait()
}
