package thinking

import (
	"encoding/json"
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
