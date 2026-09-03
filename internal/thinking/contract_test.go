package thinking

import (
	"regexp"
	"strings"
	"testing"
)

// TestHintNamesEveryRequiredField guards the self-correcting hint: a caller
// that fails validation must be able to fix the call from the hint alone.
func TestHintNamesEveryRequiredField(t *testing.T) {
	for _, f := range []string{
		"thought", "thoughtNumber", "totalThoughts", "nextThoughtNeeded",
		"confidence", "assumptions", "critique", "counterArgument", "nextStepRationale",
	} {
		if !regexp.MustCompile(`\b` + f + `\b`).MatchString(requiredFieldsChecklist) {
			t.Errorf("checklist missing required field %q as a whole word: %s", f, requiredFieldsChecklist)
		}
	}
}

// TestToolDescriptionContractGuards pins the parts of the description that
// other components depend on: the shared checklist, the statelessness
// statement, and the absence of the removed episode discipline.
func TestToolDescriptionContractGuards(t *testing.T) {
	for _, want := range []string{
		requiredFieldsChecklist,
		"keeps nothing between calls",
		"Your own context is the record",
		"after the first",                                   // the restatement norm exempts thought 1
		"this line of thinking",                             // nextThoughtNeeded is gate-scoped, matching the skill
		"sit in\n0.8–0.9, the field is telling you nothing", // calibration heuristic shared with the skill
	} {
		if !strings.Contains(ToolDescription, want) {
			t.Errorf("ToolDescription missing %q", want)
		}
	}
	for _, stale := range []string{"episodeId", "session confidence", "sessionConfidence", "thoughtHistoryLength"} {
		if strings.Contains(ToolDescription, stale) {
			t.Errorf("ToolDescription still mentions removed %q", stale)
		}
	}
	if n := strings.Count(ToolDescription, "\n"); n > 80 {
		t.Errorf("ToolDescription is %d lines; keep it under 80 so the contract stays readable", n)
	}
}
