package thinking

import (
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
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
		ThoughtNumber:     intPtr(num),
		TotalThoughts:     intPtr(3),
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
	if resp.ThoughtHistoryLength != 1 {
		t.Errorf("response.ThoughtHistoryLength = %d, want 1", resp.ThoughtHistoryLength)
	}
	// The response no longer echoes thoughtNumber/totalThoughts/nextThoughtNeeded
	// (caller already has them). Verify the value via the persisted history.
	snap := s.Snapshot()
	if got := *snap.Thoughts[0].ThoughtNumber; got != 1 {
		t.Errorf("snapshot thoughtNumber = %d, want 1", got)
	}
}

func TestProcessThoughtAutoAssignsThoughtNumber(t *testing.T) {
	s := NewServer()
	for i := 1; i <= 3; i++ {
		td := validInput(i)
		td.ThoughtNumber = nil // omit — server should fill in
		if _, err := s.ProcessThought(td); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	snap := s.Snapshot()
	if got := len(snap.Thoughts); got != 3 {
		t.Fatalf("history length = %d, want 3", got)
	}
	for i, th := range snap.Thoughts {
		if got := *th.ThoughtNumber; got != i+1 {
			t.Errorf("thought %d auto-assigned %d, want %d", i, got, i+1)
		}
	}
}

func TestProcessThoughtInheritsTotalThoughts(t *testing.T) {
	s := NewServer()
	first := validInput(1)
	first.TotalThoughts = intPtr(7)
	if _, err := s.ProcessThought(first); err != nil {
		t.Fatal(err)
	}
	second := validInput(2)
	second.TotalThoughts = nil // omit — should inherit 7
	if _, err := s.ProcessThought(second); err != nil {
		t.Fatal(err)
	}
	snap := s.Snapshot()
	if got := *snap.Thoughts[1].TotalThoughts; got != 7 {
		t.Errorf("totalThoughts not inherited: got %d, want 7", got)
	}
}

func TestProcessThoughtAutoAssignsBranchThoughtNumber(t *testing.T) {
	s := NewServer()
	if _, err := s.ProcessThought(validInput(1)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ProcessThought(validInput(2)); err != nil {
		t.Fatal(err)
	}
	// First branch thought, omitted thoughtNumber → expect within-branch depth = 1.
	first := validInput(0)
	first.ThoughtNumber = nil
	first.BranchFromThought = intPtr(2)
	first.BranchID = "alt"
	if _, err := s.ProcessThought(first); err != nil {
		t.Fatal(err)
	}
	// Second branch thought, also omitted → expect 2.
	second := validInput(0)
	second.ThoughtNumber = nil
	second.BranchFromThought = intPtr(2)
	second.BranchID = "alt"
	if _, err := s.ProcessThought(second); err != nil {
		t.Fatal(err)
	}
	snap := s.Snapshot()
	got := snap.Branches["alt"]
	if len(got) != 2 {
		t.Fatalf("branch alt length = %d, want 2", len(got))
	}
	if n := *got[0].ThoughtNumber; n != 1 {
		t.Errorf("first branch thought auto-assigned %d, want 1", n)
	}
	if n := *got[1].ThoughtNumber; n != 2 {
		t.Errorf("second branch thought auto-assigned %d, want 2", n)
	}
}

func TestProcessThoughtAutoAssignsRevisionThoughtNumber(t *testing.T) {
	s := NewServer()
	for i := 1; i <= 3; i++ {
		if _, err := s.ProcessThought(validInput(i)); err != nil {
			t.Fatal(err)
		}
	}
	// Revising thought 2; omit thoughtNumber → server fills in next trunk slot (4).
	rev := validInput(0)
	rev.ThoughtNumber = nil
	rev.IsRevision = boolPtr(true)
	rev.RevisesThought = intPtr(2)
	if _, err := s.ProcessThought(rev); err != nil {
		t.Fatal(err)
	}
	snap := s.Snapshot()
	if got := *snap.Thoughts[3].ThoughtNumber; got != 4 {
		t.Errorf("revision auto-assigned %d, want 4 (next trunk slot)", got)
	}
}

func TestProcessThoughtInheritanceSkipsBranchThoughts(t *testing.T) {
	// Regression: inheritance must walk back to the last *trunk* thought, not
	// the most recent thought of any kind. A branch thought with an
	// auto-bumped TotalThoughts should not contaminate the trunk's value.
	s := NewServer()
	first := validInput(1)
	first.TotalThoughts = intPtr(5)
	if _, err := s.ProcessThought(first); err != nil {
		t.Fatal(err)
	}
	// Branch thought with explicit thoughtNumber > totalThoughts → server
	// auto-bumps the stored TotalThoughts on the branch thought to 99.
	branch := validInput(99)
	branch.TotalThoughts = intPtr(5)
	branch.BranchFromThought = intPtr(1)
	branch.BranchID = "alt"
	if _, err := s.ProcessThought(branch); err != nil {
		t.Fatal(err)
	}
	// Resume trunk with omitted totalThoughts → must inherit 5 (from the
	// last trunk thought), NOT 99 (from the branch thought).
	resume := validInput(2)
	resume.TotalThoughts = nil
	if _, err := s.ProcessThought(resume); err != nil {
		t.Fatal(err)
	}
	snap := s.Snapshot()
	if got := *snap.Thoughts[2].TotalThoughts; got != 5 {
		t.Errorf("inheritance contaminated by branch: got %d, want 5", got)
	}
}

func TestProcessThoughtFirstThoughtRequiresTotalThoughts(t *testing.T) {
	s := NewServer()
	td := validInput(1)
	td.TotalThoughts = nil // omit on first call — must error
	res, err := s.ProcessThought(td)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Error("expected IsError=true when totalThoughts is omitted on first thought")
	}
	if got := s.HistoryLength(); got != 0 {
		t.Errorf("first-thought failure should not mutate state, HistoryLength=%d", got)
	}
}

func TestProcessThoughtAutoBumpsTotalThoughts(t *testing.T) {
	s := NewServer()
	td := validInput(5)
	td.TotalThoughts = intPtr(3) // less than ThoughtNumber
	if _, err := s.ProcessThought(td); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	snap := s.Snapshot()
	if got := *snap.Thoughts[0].TotalThoughts; got != 5 {
		t.Errorf("totalThoughts not auto-bumped: got %d, want 5", got)
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

func TestRenderTranscriptIncludesAllSections(t *testing.T) {
	s := NewServer()
	td := validInput(1)
	td.Thought = "I think we should normalize first."
	td.Assumptions = []string{"row count is current"}
	td.Critique = "drifted into solution mode"
	td.CounterArgument = "monolith-first is simpler"
	td.NextStepRationale = "verify row count next"

	res, err := s.ProcessThought(td)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("unexpected error: %s", res.Text)
	}

	for _, want := range []string{
		"Thought 1 of 3",
		"confidence 0.50",
		"I think we should normalize first.",
		"Assumptions:",
		"row count is current",
		"Critique:",
		"drifted into solution mode",
		"Counter-argument:",
		"monolith-first is simpler",
		"Next, I want to: verify row count next",
		"session confidence 0.50 across 1 thought",
	} {
		if !strings.Contains(res.Text, want) {
			t.Errorf("transcript missing %q\n--- transcript ---\n%s", want, res.Text)
		}
	}
}

func TestRenderTranscriptEmptyAssumptions(t *testing.T) {
	s := NewServer()
	td := validInput(1)
	td.Assumptions = []string{}
	res, _ := s.ProcessThought(td)
	if !strings.Contains(res.Text, "Assumptions: (none claimed)") {
		t.Errorf("empty assumptions should render as (none claimed); got:\n%s", res.Text)
	}
}

func TestRenderTranscriptOmitsNextOnTerminal(t *testing.T) {
	s := NewServer()
	td := validInput(1)
	td.NextThoughtNeeded = boolPtr(false)
	td.NextStepRationale = ""
	res, _ := s.ProcessThought(td)
	if strings.Contains(res.Text, "Next, I want to:") {
		t.Errorf("terminal thought should omit Next section; got:\n%s", res.Text)
	}
}

func TestProcessThoughtConcurrent(t *testing.T) {
	s := NewServer()
	const goroutines = 100
	const perGoroutine = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(gID int) {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				td := validInput(gID*perGoroutine + i + 1)
				td.TotalThoughts = intPtr(goroutines * perGoroutine)
				if _, err := s.ProcessThought(td); err != nil {
					t.Errorf("goroutine %d iter %d: %v", gID, i, err)
					return
				}
			}
		}(g)
	}
	wg.Wait()

	if got := s.HistoryLength(); got != goroutines*perGoroutine {
		t.Errorf("HistoryLength = %d, want %d", got, goroutines*perGoroutine)
	}
}

func TestRevisionHeader(t *testing.T) {
	s := NewServer()
	if _, err := s.ProcessThought(validInput(1)); err != nil {
		t.Fatal(err)
	}
	td := validInput(2)
	td.IsRevision = boolPtr(true)
	td.RevisesThought = intPtr(1)
	res, _ := s.ProcessThought(td)
	if !strings.Contains(res.Text, "Revision of thought 1 (now thought 2)") {
		t.Errorf("revision header missing; got:\n%s", res.Text)
	}
}

func TestBranchFirstHeader(t *testing.T) {
	s := NewServer()
	if _, err := s.ProcessThought(validInput(1)); err != nil {
		t.Fatal(err)
	}
	td := validInput(2)
	td.BranchFromThought = intPtr(1)
	td.BranchID = "monolith-first"
	res, _ := s.ProcessThought(td)
	if !strings.Contains(res.Text, "Branch 'monolith-first' from thought 1") {
		t.Errorf("branch-first header missing; got:\n%s", res.Text)
	}
}

func TestBranchSubsequentHeader(t *testing.T) {
	s := NewServer()
	if _, err := s.ProcessThought(validInput(1)); err != nil {
		t.Fatal(err)
	}
	td2 := validInput(2)
	td2.BranchFromThought = intPtr(1)
	td2.BranchID = "monolith-first"
	if _, err := s.ProcessThought(td2); err != nil {
		t.Fatal(err)
	}
	td3 := validInput(3)
	td3.BranchFromThought = intPtr(1)
	td3.BranchID = "monolith-first"
	res, _ := s.ProcessThought(td3)
	if !strings.Contains(res.Text, "Branch 'monolith-first' · thought 3") {
		t.Errorf("branch-subsequent header missing; got:\n%s", res.Text)
	}
}

func TestSnapshotDeepCopiesThoughts(t *testing.T) {
	s := NewServer()
	if _, err := s.ProcessThought(validInput(1)); err != nil {
		t.Fatal(err)
	}
	snap := s.Snapshot()
	if len(snap.Thoughts) != 1 {
		t.Fatalf("snapshot length = %d, want 1", len(snap.Thoughts))
	}
	// Mutating the snapshot must not affect the server.
	snap.Thoughts[0].Thought = "MUTATED"
	if _, err := s.ProcessThought(validInput(2)); err != nil {
		t.Fatal(err)
	}
	snap2 := s.Snapshot()
	if snap2.Thoughts[0].Thought == "MUTATED" {
		t.Errorf("snapshot is shallow — mutating snapshot leaked into server state")
	}
}

func TestSnapshotIncludesBranches(t *testing.T) {
	s := NewServer()
	if _, err := s.ProcessThought(validInput(1)); err != nil {
		t.Fatal(err)
	}
	td := validInput(2)
	td.BranchFromThought = intPtr(1)
	td.BranchID = "alt"
	if _, err := s.ProcessThought(td); err != nil {
		t.Fatal(err)
	}
	snap := s.Snapshot()
	if got, ok := snap.Branches["alt"]; !ok || len(got) != 1 {
		t.Errorf("branch 'alt' missing or empty in snapshot: %+v", snap.Branches)
	}
}

func TestBranchDualFooter(t *testing.T) {
	s := NewServer()
	// Trunk seed
	if _, err := s.ProcessThought(validInput(1)); err != nil {
		t.Fatal(err)
	}
	td := validInput(2)
	td.BranchFromThought = intPtr(1)
	td.BranchID = "alt"
	td.Confidence = 0.4
	res, _ := s.ProcessThought(td)
	if !strings.Contains(res.Text, "branch 'alt' confidence 0.40") {
		t.Errorf("branch confidence footer missing; got:\n%s", res.Text)
	}
	if !strings.Contains(res.Text, "session confidence (trunk)") {
		t.Errorf("trunk session footer missing; got:\n%s", res.Text)
	}
}
