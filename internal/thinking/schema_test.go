package thinking

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestThoughtDataJSONRoundTrip(t *testing.T) {
	yes := true
	td := ThoughtData{
		Thought:           "I think we should normalize first",
		ThoughtNumber:     1,
		TotalThoughts:     3,
		NextThoughtNeeded: &yes,
		Confidence:        0.6,
		Assumptions:       []string{"row count is current"},
		Critique:          "drifted into solution mode",
		CounterArgument:   "monolith-first is simpler",
		NextStepRationale: "verify row count next",
	}

	data, err := json.Marshal(td)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got ThoughtData
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Confidence != 0.6 {
		t.Errorf("confidence = %v, want 0.6", got.Confidence)
	}
	if len(got.Assumptions) != 1 || got.Assumptions[0] != "row count is current" {
		t.Errorf("assumptions = %v, want [row count is current]", got.Assumptions)
	}
}

func TestThoughtResponseJSONShape(t *testing.T) {
	resp := ThoughtResponse{
		ThoughtNumber:        1,
		TotalThoughts:        3,
		NextThoughtNeeded:    true,
		Branches:             []string{},
		ThoughtHistoryLength: 1,
		SessionConfidence:    0.6,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, key := range []string{"thoughtNumber", "totalThoughts", "nextThoughtNeeded", "branches", "thoughtHistoryLength", "sessionConfidence"} {
		if _, ok := got[key]; !ok {
			t.Errorf("missing key: %s", key)
		}
	}
	if _, ok := got["branchConfidences"]; ok {
		t.Errorf("branchConfidences should be omitted when nil/empty")
	}
}

func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }

// validBase returns a minimally valid ThoughtData.
// Each test mutates one field to assert that field's rule.
func validBase() ThoughtData {
	return ThoughtData{
		Thought:           "a thought",
		ThoughtNumber:     1,
		TotalThoughts:     1,
		NextThoughtNeeded: boolPtr(false),
		Confidence:        0.5,
		Assumptions:       []string{},
		Critique:          "narrow analysis",
		CounterArgument:   "the opposite case",
	}
}

func TestValidateRequiredFields(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*ThoughtData)
		wantErr string
	}{
		{"empty thought", func(td *ThoughtData) { td.Thought = "" }, "thought must be a non-empty string"},
		{"zero thoughtNumber", func(td *ThoughtData) { td.ThoughtNumber = 0 }, "thoughtNumber must be ≥ 1"},
		{"negative thoughtNumber", func(td *ThoughtData) { td.ThoughtNumber = -1 }, "thoughtNumber must be ≥ 1"},
		{"zero totalThoughts", func(td *ThoughtData) { td.TotalThoughts = 0 }, "totalThoughts must be ≥ 1"},
		{"missing nextThoughtNeeded", func(td *ThoughtData) { td.NextThoughtNeeded = nil }, "nextThoughtNeeded must be present"},
		{"empty critique", func(td *ThoughtData) { td.Critique = "" }, "critique must be a non-empty string"},
		{"empty counterArgument", func(td *ThoughtData) { td.CounterArgument = "" }, "counterArgument must be a non-empty string"},
		{"nil assumptions", func(td *ThoughtData) { td.Assumptions = nil }, "assumptions must be present"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			td := validBase()
			tc.mutate(&td)
			err := td.Validate()
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestValidateAcceptsBase(t *testing.T) {
	td := validBase()
	if err := td.Validate(); err != nil {
		t.Fatalf("base case should validate, got: %v", err)
	}
}

func TestValidateAcceptsEmptyAssumptions(t *testing.T) {
	td := validBase()
	td.Assumptions = []string{} // explicit empty slice is allowed
	if err := td.Validate(); err != nil {
		t.Fatalf("empty assumptions should be allowed, got: %v", err)
	}
}

// contains is a tiny strings.Contains wrapper for table tests; implemented in
// schema_test.go to keep production code free of test-only helpers.
func contains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && stringIndex(s, substr) >= 0)
}

func stringIndex(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func TestValidateConfidenceRange(t *testing.T) {
	cases := []struct {
		name       string
		confidence float64
		wantOK     bool
	}{
		{"below zero", -0.01, false},
		{"zero", 0.0, true},
		{"midpoint", 0.5, true},
		{"one", 1.0, true},
		{"above one", 1.01, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			td := validBase()
			td.Confidence = tc.confidence
			err := td.Validate()
			if tc.wantOK && err != nil {
				t.Errorf("confidence %v should validate, got: %v", tc.confidence, err)
			}
			if !tc.wantOK && err == nil {
				t.Errorf("confidence %v should fail validation", tc.confidence)
			}
		})
	}
}

func TestValidateConditionalNextStepRationale(t *testing.T) {
	t.Run("required when nextThoughtNeeded=true", func(t *testing.T) {
		td := validBase()
		td.NextThoughtNeeded = boolPtr(true)
		td.NextStepRationale = ""
		err := td.Validate()
		if err == nil || !contains(err.Error(), "nextStepRationale required") {
			t.Errorf("expected nextStepRationale error, got %v", err)
		}
	})
	t.Run("ignored when nextThoughtNeeded=false", func(t *testing.T) {
		td := validBase()
		td.NextThoughtNeeded = boolPtr(false)
		td.NextStepRationale = ""
		if err := td.Validate(); err != nil {
			t.Errorf("nextStepRationale empty should be OK when nextThoughtNeeded=false, got: %v", err)
		}
	})
}

func TestValidateLengthCaps(t *testing.T) {
	t.Run("critique at cap is allowed", func(t *testing.T) {
		td := validBase()
		td.Critique = strings.Repeat("x", maxCritiqueLen)
		if err := td.Validate(); err != nil {
			t.Errorf("critique at cap should validate, got: %v", err)
		}
	})
	t.Run("critique over cap rejected", func(t *testing.T) {
		td := validBase()
		td.Critique = strings.Repeat("x", maxCritiqueLen+1)
		err := td.Validate()
		if err == nil || !contains(err.Error(), "critique must be ≤") {
			t.Errorf("expected critique length error, got %v", err)
		}
	})
	t.Run("counterArgument at cap is allowed", func(t *testing.T) {
		td := validBase()
		td.CounterArgument = strings.Repeat("x", maxCounterArgumentLen)
		if err := td.Validate(); err != nil {
			t.Errorf("counterArgument at cap should validate, got: %v", err)
		}
	})
	t.Run("counterArgument over cap rejected", func(t *testing.T) {
		td := validBase()
		td.CounterArgument = strings.Repeat("x", maxCounterArgumentLen+1)
		err := td.Validate()
		if err == nil || !contains(err.Error(), "counterArgument must be ≤") {
			t.Errorf("expected counterArgument length error, got %v", err)
		}
	})
	t.Run("assumption entry at cap is allowed", func(t *testing.T) {
		td := validBase()
		td.Assumptions = []string{strings.Repeat("x", maxAssumptionLen)}
		if err := td.Validate(); err != nil {
			t.Errorf("assumption at cap should validate, got: %v", err)
		}
	})
	t.Run("assumption entry over cap rejected", func(t *testing.T) {
		td := validBase()
		td.Assumptions = []string{"ok", strings.Repeat("x", maxAssumptionLen+1)}
		err := td.Validate()
		if err == nil || !contains(err.Error(), "assumptions[1] must be ≤") {
			t.Errorf("expected assumption length error, got %v", err)
		}
	})
	t.Run("nextStepRationale over cap rejected when needed", func(t *testing.T) {
		td := validBase()
		td.NextThoughtNeeded = boolPtr(true)
		td.NextStepRationale = strings.Repeat("x", maxNextStepRationaleLen+1)
		err := td.Validate()
		if err == nil || !contains(err.Error(), "nextStepRationale must be ≤") {
			t.Errorf("expected nextStepRationale length error, got %v", err)
		}
	})
	t.Run("nextStepRationale length not enforced when not needed", func(t *testing.T) {
		// When nextThoughtNeeded=false, the field is logically absent and any
		// stale value is benign — no length check.
		td := validBase()
		td.NextThoughtNeeded = boolPtr(false)
		td.NextStepRationale = strings.Repeat("x", maxNextStepRationaleLen+500)
		if err := td.Validate(); err != nil {
			t.Errorf("nextStepRationale length should not fire when nextThoughtNeeded=false, got: %v", err)
		}
	})
	t.Run("rune-counted not byte-counted", func(t *testing.T) {
		// 200 em-dashes (each 3 bytes in UTF-8, 1 rune). Should be rejected
		// for nextStepRationale (cap 200) only because we go one rune over.
		td := validBase()
		td.NextThoughtNeeded = boolPtr(true)
		td.NextStepRationale = ""
		for range maxNextStepRationaleLen {
			td.NextStepRationale += "—"
		}
		if err := td.Validate(); err != nil {
			t.Errorf("exactly cap runes should validate, got: %v", err)
		}
		td.NextStepRationale += "—"
		if err := td.Validate(); err == nil {
			t.Errorf("one rune over cap should be rejected")
		}
	})
}

func TestValidateBranchBothOrNeither(t *testing.T) {
	t.Run("both present", func(t *testing.T) {
		td := validBase()
		td.BranchFromThought = intPtr(1)
		td.BranchID = "branch-a"
		if err := td.Validate(); err != nil {
			t.Errorf("both branch fields present should validate, got: %v", err)
		}
	})
	t.Run("both absent", func(t *testing.T) {
		td := validBase()
		td.BranchFromThought = nil
		td.BranchID = ""
		if err := td.Validate(); err != nil {
			t.Errorf("both branch fields absent should validate, got: %v", err)
		}
	})
	t.Run("only BranchFromThought", func(t *testing.T) {
		td := validBase()
		td.BranchFromThought = intPtr(1)
		td.BranchID = ""
		err := td.Validate()
		if err == nil || !contains(err.Error(), "branchFromThought and branchId") {
			t.Errorf("expected both-or-neither error, got %v", err)
		}
	})
	t.Run("only BranchID", func(t *testing.T) {
		td := validBase()
		td.BranchFromThought = nil
		td.BranchID = "branch-a"
		err := td.Validate()
		if err == nil || !contains(err.Error(), "branchFromThought and branchId") {
			t.Errorf("expected both-or-neither error, got %v", err)
		}
	})
}
