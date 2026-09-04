package thinking

import (
	"fmt"
	"strings"
)

// render narrates one thought in the first-person register the tool
// description asks for, followed by its critical-thinking sections. It is
// called only after Validate has passed, so NextThoughtNeeded is non-nil.
func render(td ThoughtData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n%s\n\n", headerLine(td), td.Thought)

	if len(td.Assumptions) == 0 {
		b.WriteString("  Assumptions: (none claimed)\n")
	} else {
		b.WriteString("  Assumptions:\n")
		for _, a := range td.Assumptions {
			fmt.Fprintf(&b, "    - %s\n", a)
		}
	}
	fmt.Fprintf(&b, "\n  Critique:\n    %s\n", td.Critique)
	fmt.Fprintf(&b, "\n  Counter-argument:\n    %s\n", td.CounterArgument)
	if *td.NextThoughtNeeded {
		fmt.Fprintf(&b, "\n  Next, I want to: %s\n", td.NextStepRationale)
	}
	return b.String()
}

// headerLine picks the header by the most specific claim the call makes: a
// revision, else a branch, else a plain numbered thought. Validate has already
// guaranteed each pair of fields is present together.
func headerLine(td ThoughtData) string {
	switch {
	case td.IsRevision:
		return fmt.Sprintf("Revision of thought %d (now thought %d) · confidence %.2f",
			td.RevisesThought, td.ThoughtNumber, td.Confidence)
	case td.BranchID != "":
		return fmt.Sprintf("Branch '%s' from thought %d · thought %d · confidence %.2f",
			td.BranchID, td.BranchFromThought, td.ThoughtNumber, td.Confidence)
	default:
		return fmt.Sprintf("Thought %d of %d · confidence %.2f",
			td.ThoughtNumber, td.TotalThoughts, td.Confidence)
	}
}
