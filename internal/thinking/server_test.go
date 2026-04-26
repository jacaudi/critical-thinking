package thinking

import (
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"testing"
	"time"
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
		Thought:           "thought number " + strconv.Itoa(num),
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

func TestProcessThoughtRevisesOutOfRange(t *testing.T) {
	s := NewServer()
	if _, err := s.ProcessThought(validInput(1)); err != nil {
		t.Fatal(err)
	}
	td := validInput(2)
	td.IsRevision = boolPtr(true)
	td.RevisesThought = intPtr(99) // history only has 1 thought
	res, err := s.ProcessThought(td)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Error("expected IsError for out-of-range revisesThought")
	}
	if !contains(res.Text, "revisesThought 99 out of range") {
		t.Errorf("error message: %s", res.Text)
	}
	if got := s.HistoryLength(); got != 1 {
		t.Errorf("range failure should not append; HistoryLength = %d, want 1", got)
	}
}

func TestProcessThoughtAdvancesLastAccessed(t *testing.T) {
	s := NewServer()
	before := s.LastAccessed()
	// Sleep a tiny amount to ensure the timestamp can advance.
	// time.Now() resolution is platform-dependent but consistently >= 1µs on
	// supported targets, so a 1ms sleep is more than enough.
	time.Sleep(time.Millisecond)
	if _, err := s.ProcessThought(validInput(1)); err != nil {
		t.Fatal(err)
	}
	after := s.LastAccessed()
	if !after.After(before) {
		t.Errorf("LastAccessed did not advance: before=%v after=%v", before, after)
	}
}

func TestProcessThoughtRecordsBranch(t *testing.T) {
	s := NewServer()

	// Trunk thought 1.
	if _, err := s.ProcessThought(validInput(1)); err != nil {
		t.Fatal(err)
	}

	// Branch from thought 1.
	td := validInput(2)
	td.BranchFromThought = intPtr(1)
	td.BranchID = "branch-a"
	res, err := s.ProcessThought(td)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("unexpected error: %s", res.Text)
	}

	var resp ThoughtResponse
	_ = json.Unmarshal([]byte(res.StructuredJSON), &resp)
	sort.Strings(resp.Branches)
	if len(resp.Branches) != 1 || resp.Branches[0] != "branch-a" {
		t.Errorf("Branches = %v, want [branch-a]", resp.Branches)
	}
}

func TestProcessThoughtRevisionRangeCheck(t *testing.T) {
	s := NewServer()
	if _, err := s.ProcessThought(validInput(1)); err != nil {
		t.Fatal(err)
	}
	td := validInput(2)
	td.IsRevision = boolPtr(true)
	td.RevisesThought = intPtr(5) // out of range, only 1 in history
	res, err := s.ProcessThought(td)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Error("expected IsError for out-of-range revisesThought")
	}
	if !contains(res.Text, "revisesThought 5 out of range") {
		t.Errorf("error message: %s", res.Text)
	}
}

func TestProcessThoughtBranchFromOutOfRange(t *testing.T) {
	s := NewServer()
	td := validInput(1)
	td.BranchFromThought = intPtr(99)
	td.BranchID = "branch-a"
	res, err := s.ProcessThought(td)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Error("expected IsError for out-of-range branchFromThought")
	}
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestSessionConfidenceTrunkOnly(t *testing.T) {
	s := NewServer()
	confs := []float64{0.5, 0.7, 0.9}
	for i, c := range confs {
		td := validInput(i + 1)
		td.Confidence = c
		if _, err := s.ProcessThought(td); err != nil {
			t.Fatal(err)
		}
	}
	want := (0.5 + 0.7 + 0.9) / 3.0
	if got := s.SessionConfidence(); !almostEqual(got, want) {
		t.Errorf("SessionConfidence = %v, want %v", got, want)
	}
}

func TestPerBranchConfidence(t *testing.T) {
	s := NewServer()

	// Trunk thought 1, conf 0.6.
	td1 := validInput(1)
	td1.Confidence = 0.6
	if _, err := s.ProcessThought(td1); err != nil {
		t.Fatal(err)
	}

	// Branch-a thought, conf 0.4.
	td2 := validInput(2)
	td2.BranchFromThought = intPtr(1)
	td2.BranchID = "branch-a"
	td2.Confidence = 0.4
	if _, err := s.ProcessThought(td2); err != nil {
		t.Fatal(err)
	}

	// Branch-a another thought, conf 0.2.
	td3 := validInput(3)
	td3.BranchFromThought = intPtr(1)
	td3.BranchID = "branch-a"
	td3.Confidence = 0.2
	res, err := s.ProcessThought(td3)
	if err != nil {
		t.Fatal(err)
	}

	var resp ThoughtResponse
	_ = json.Unmarshal([]byte(res.StructuredJSON), &resp)

	// Trunk should still be 0.6 (only 1 trunk thought).
	if !almostEqual(resp.SessionConfidence, 0.6) {
		t.Errorf("SessionConfidence = %v, want 0.6", resp.SessionConfidence)
	}
	// Branch-a should be (0.4 + 0.2) / 2 = 0.3.
	got := resp.BranchConfidences["branch-a"]
	if !almostEqual(got, 0.3) {
		t.Errorf("branch-a confidence = %v, want 0.3", got)
	}
}
