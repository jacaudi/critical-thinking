package thinking

import (
	"encoding/json"
	"strings"
	"testing"
)

// errorPayload is the JSON shape ProcessThought emits on an error result.
type errorPayload struct {
	Error  string `json:"error"`
	Status string `json:"status"`
	Hint   string `json:"hint"`
}

func TestValidatePathErrorCarriesHint(t *testing.T) {
	s := NewServer()
	td := validInput(1)
	td.NextStepRationale = "" // Validate() fails: required when nextThoughtNeeded=true
	res, err := s.ProcessThought(td)
	if err != nil {
		t.Fatalf("unexpected go error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true")
	}
	var p errorPayload
	if err := json.Unmarshal([]byte(res.Text), &p); err != nil {
		t.Fatalf("error result is not JSON: %v\n%s", err, res.Text)
	}
	if p.Status != "failed" {
		t.Errorf("status = %q, want failed", p.Status)
	}
	if p.Hint != requiredFieldsChecklist {
		t.Errorf("hint = %q, want the shared checklist", p.Hint)
	}
	// The hint must name every required field.
	for _, f := range []string{
		"thought", "thoughtNumber", "totalThoughts", "nextThoughtNeeded",
		"confidence", "assumptions", "critique", "counterArgument", "nextStepRationale",
	} {
		if !strings.Contains(p.Hint, f) {
			t.Errorf("hint missing required field %q: %s", f, p.Hint)
		}
	}
}

func TestRangeErrorOmitsHint(t *testing.T) {
	s := NewServer()
	if _, err := s.ProcessThought(validInput(1)); err != nil {
		t.Fatal(err)
	}
	td := validInput(2)
	td.IsRevision = boolPtr(true)
	td.RevisesThought = intPtr(99) // out of range -> ProcessThought range error, not Validate()
	res, err := s.ProcessThought(td)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true")
	}
	var p errorPayload
	if err := json.Unmarshal([]byte(res.Text), &p); err != nil {
		t.Fatalf("error result is not JSON: %v", err)
	}
	if p.Hint != "" {
		t.Errorf("range error must not carry a hint; got %q", p.Hint)
	}
}

func TestToolDescriptionContractGuards(t *testing.T) {
	for _, want := range []string{
		requiredFieldsChecklist,   // front-loaded checklist (single source)
		"episodeId",               // isolation discipline is documented
		"unrelated problem",       // the episodeId switch guidance
		"current episode's trunk", // corrected per-episode confidence line (substring chosen to survive the line-wrap in the description text)
	} {
		if !strings.Contains(ToolDescription, want) {
			t.Errorf("ToolDescription missing %q", want)
		}
	}
	// The stale connection-wide confidence phrasing must be gone.
	if strings.Contains(ToolDescription, "(mean of trunk thoughts)") {
		t.Error("ToolDescription still has the pre-episode confidence phrasing")
	}
}
