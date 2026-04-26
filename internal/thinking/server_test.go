package thinking

import (
	"encoding/json"
	"testing"
)

func TestNewServerStartsEmpty(t *testing.T) {
	s := NewServer()
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
	if got := s.HistoryLength(); got != 0 {
		t.Errorf("HistoryLength = %d, want 0", got)
	}
	if got := s.SessionConfidence(); got != 0 {
		t.Errorf("SessionConfidence = %v, want 0", got)
	}
}

func validInput(num int) ThoughtData {
	return ThoughtData{
		Thought:           "thought number " + itoa(num),
		ThoughtNumber:     num,
		TotalThoughts:     3,
		NextThoughtNeeded: boolPtr(true),
		Confidence:        0.5,
		Assumptions:       []string{},
		Critique:          "narrow",
		CounterArgument:   "alternative",
		NextStepRationale: "next thing",
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	digits := []byte{}
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	if neg {
		return "-" + string(digits)
	}
	return string(digits)
}

func TestProcessThoughtAppendsHistory(t *testing.T) {
	s := NewServer()
	res, err := s.ProcessThought(validInput(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError=true, text=%s", res.Text)
	}
	if got := s.HistoryLength(); got != 1 {
		t.Errorf("HistoryLength = %d, want 1", got)
	}

	var resp ThoughtResponse
	if err := json.Unmarshal([]byte(res.StructuredJSON), &resp); err != nil {
		t.Fatalf("unmarshal structured: %v", err)
	}
	if resp.ThoughtNumber != 1 {
		t.Errorf("response.ThoughtNumber = %d, want 1", resp.ThoughtNumber)
	}
	if resp.ThoughtHistoryLength != 1 {
		t.Errorf("response.ThoughtHistoryLength = %d, want 1", resp.ThoughtHistoryLength)
	}
}

func TestProcessThoughtAutoBumpsTotalThoughts(t *testing.T) {
	s := NewServer()
	td := validInput(5)
	td.TotalThoughts = 3 // less than ThoughtNumber
	res, err := s.ProcessThought(td)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var resp ThoughtResponse
	_ = json.Unmarshal([]byte(res.StructuredJSON), &resp)
	if resp.TotalThoughts != 5 {
		t.Errorf("totalThoughts not auto-bumped: got %d, want 5", resp.TotalThoughts)
	}
}

func TestProcessThoughtValidationError(t *testing.T) {
	s := NewServer()
	td := validInput(1)
	td.Thought = "" // invalid
	res, err := s.ProcessThought(td)
	if err != nil {
		t.Fatalf("ProcessThought should not return Go error on validation failure: %v", err)
	}
	if !res.IsError {
		t.Error("expected IsError=true")
	}
	if got := s.HistoryLength(); got != 0 {
		t.Errorf("validation failure should not mutate state, HistoryLength=%d", got)
	}
}
