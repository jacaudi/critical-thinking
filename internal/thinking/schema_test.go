package thinking

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func TestThoughtDataJSONRoundTrip(t *testing.T) {
	in := `{"thought":"x","thoughtNumber":2,"totalThoughts":3,"nextThoughtNeeded":true,` +
		`"isRevision":true,"revisesThought":1,"branchFromThought":1,"branchId":"b",` +
		`"confidence":0.7,"assumptions":["a"],"critique":"c","counterArgument":"ca",` +
		`"nextStepRationale":"n","episodeId":"ignored","needsMoreThoughts":true}`
	var td ThoughtData
	if err := json.Unmarshal([]byte(in), &td); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !td.IsRevision || td.RevisesThought != 1 || td.BranchFromThought != 1 || td.BranchID != "b" {
		t.Errorf("optional fields not decoded: %+v", td)
	}
	if td.NextThoughtNeeded == nil || !*td.NextThoughtNeeded {
		t.Errorf("nextThoughtNeeded not decoded: %+v", td.NextThoughtNeeded)
	}
	if td.EpisodeID != "ignored" || !td.NeedsMoreThoughts {
		t.Errorf("deprecated fields must still decode (schema forbids unknown keys): %+v", td)
	}
	out, err := json.Marshal(td)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back ThoughtData
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if back.Thought != td.Thought || back.RevisesThought != td.RevisesThought || back.BranchID != td.BranchID {
		t.Errorf("round trip changed data: %+v vs %+v", back, td)
	}
}

// The omitempty tags decide which fields jsonschema-go marks required (design §6); this pins them.
func TestOptionalFieldsOmittedWhenZero(t *testing.T) {
	out, err := json.Marshal(validInput(1))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"isRevision", "revisesThought", "branchFromThought", "branchId", "episodeId", "needsMoreThoughts"} {
		if strings.Contains(string(out), `"`+key+`"`) {
			t.Errorf("zero-valued %s must be omitted: %s", key, out)
		}
	}
}

func TestThoughtResponseJSONShape(t *testing.T) {
	out, err := json.Marshal(ThoughtResponse{ThoughtNumber: 1, TotalThoughts: 2, NextThoughtNeeded: false, Confidence: 0.6})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"thoughtNumber":1,"totalThoughts":2,"nextThoughtNeeded":false,"confidence":0.6}`
	if string(out) != want {
		t.Errorf("response JSON = %s, want %s", out, want)
	}
}

func TestValidateAcceptsBase(t *testing.T) {
	if err := validInput(1).Validate(); err != nil {
		t.Errorf("valid input rejected: %v", err)
	}
}

func TestValidateRules(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ThoughtData)
		want   string // substring of the error; "" means valid
	}{
		{"empty thought", func(td *ThoughtData) { td.Thought = "" }, "thought must be"},
		{"thought whitespace only", func(td *ThoughtData) { td.Thought = " \n\t" }, "thought must be"},
		{"thoughtNumber 0", func(td *ThoughtData) { td.ThoughtNumber = 0 }, "thoughtNumber must be ≥ 1 (got 0)"},
		{"totalThoughts 0", func(td *ThoughtData) { td.TotalThoughts = 0 }, "totalThoughts must be ≥ 1 (got 0)"},
		{"nextThoughtNeeded absent", func(td *ThoughtData) { td.NextThoughtNeeded = nil }, "nextThoughtNeeded must be present"},
		{"confidence below 0", func(td *ThoughtData) { td.Confidence = -0.1 }, "confidence must be between"},
		{"confidence above 1", func(td *ThoughtData) { td.Confidence = 1.1 }, "confidence must be between"},
		{"confidence NaN", func(td *ThoughtData) { td.Confidence = math.NaN() }, "confidence must be between"},
		{"confidence 0 ok", func(td *ThoughtData) { td.Confidence = 0 }, ""},
		{"confidence 1 ok", func(td *ThoughtData) { td.Confidence = 1 }, ""},
		{"assumptions nil", func(td *ThoughtData) { td.Assumptions = nil }, "assumptions must be present"},
		{"assumptions empty ok", func(td *ThoughtData) { td.Assumptions = []string{} }, ""},
		{"assumptions blank entry", func(td *ThoughtData) { td.Assumptions = []string{"ok", " "} }, "assumptions[1] must be"},
		{"critique empty", func(td *ThoughtData) { td.Critique = "" }, "critique must be"},
		{"critique whitespace only", func(td *ThoughtData) { td.Critique = "  " }, "critique must be"},
		{"counterArgument empty", func(td *ThoughtData) { td.CounterArgument = "" }, "counterArgument must be"},
		{"counterArgument whitespace only", func(td *ThoughtData) { td.CounterArgument = "\n" }, "counterArgument must be"},
		{"rationale required when more needed", func(td *ThoughtData) { td.NextStepRationale = "" }, "nextStepRationale required"},
		{"rationale whitespace when more needed", func(td *ThoughtData) { td.NextStepRationale = " " }, "nextStepRationale required"},
		{"rationale optional when done", func(td *ThoughtData) {
			td.NextThoughtNeeded = new(false)
			td.NextStepRationale = ""
		}, ""},
		{"isRevision without revisesThought", func(td *ThoughtData) { td.IsRevision = true }, "isRevision=true together with revisesThought"},
		{"revisesThought without isRevision", func(td *ThoughtData) { td.RevisesThought = 1 }, "isRevision=true together with revisesThought"},
		{"revision pair ok", func(td *ThoughtData) { td.IsRevision = true; td.RevisesThought = 1 }, ""},
		{"revisesThought negative", func(td *ThoughtData) { td.IsRevision = true; td.RevisesThought = -1 }, "revisesThought must be ≥ 1"},
		{"revisesThought beyond thoughtNumber ok", func(td *ThoughtData) { td.IsRevision = true; td.RevisesThought = 999 }, ""},
		{"branchFromThought without branchId", func(td *ThoughtData) { td.BranchFromThought = 1 }, "branchFromThought ≥ 1 together with branchId"},
		{"branchId without branchFromThought", func(td *ThoughtData) { td.BranchID = "b" }, "branchFromThought ≥ 1 together with branchId"},
		{"branch pair ok", func(td *ThoughtData) { td.BranchFromThought = 1; td.BranchID = "b" }, ""},
		{"branchFromThought negative", func(td *ThoughtData) { td.BranchFromThought = -1; td.BranchID = "b" }, "branchFromThought must be ≥ 1"},
		{"deprecated fields ignored", func(td *ThoughtData) { td.EpisodeID = "x"; td.NeedsMoreThoughts = true }, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			td := validInput(1)
			tc.mutate(&td)
			err := td.Validate()
			switch {
			case tc.want == "" && err != nil:
				t.Errorf("unexpected error: %v", err)
			case tc.want != "" && err == nil:
				t.Errorf("expected error containing %q, got nil", tc.want)
			case tc.want != "" && !strings.Contains(err.Error(), tc.want):
				t.Errorf("error = %q, want substring %q", err, tc.want)
			}
		})
	}
}
