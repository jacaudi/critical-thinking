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
