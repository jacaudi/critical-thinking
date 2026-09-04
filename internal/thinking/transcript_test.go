package thinking

import (
	"strings"
	"testing"
)

func TestRenderTranscriptIncludesAllSections(t *testing.T) {
	td := validInput(1)
	td.Assumptions = []string{"first", "second"}
	res := Process(td)
	if res.IsError {
		t.Fatalf("unexpected error: %s", res.Text)
	}
	for _, want := range []string{
		"Thought 1 of 3 · confidence 0.50",
		"thought number 1",
		"  Assumptions:",
		"    - first",
		"    - second",
		"  Critique:\n    narrow",
		"  Counter-argument:\n    alternative",
		"  Next, I want to: next thing",
	} {
		if !strings.Contains(res.Text, want) {
			t.Errorf("transcript missing %q:\n%s", want, res.Text)
		}
	}
	for _, stale := range []string{"session confidence", "across"} {
		if strings.Contains(res.Text, stale) {
			t.Errorf("transcript still carries stateful footer text %q:\n%s", stale, res.Text)
		}
	}
}

func TestRenderTranscriptEmptyAssumptions(t *testing.T) {
	res := Process(validInput(1)) // Assumptions is []
	if !strings.Contains(res.Text, "  Assumptions: (none claimed)") {
		t.Errorf("expected the none-claimed line:\n%s", res.Text)
	}
}

func TestRenderTranscriptOmitsNextOnTerminal(t *testing.T) {
	td := validInput(3)
	td.NextThoughtNeeded = new(false)
	// NextStepRationale is deliberately left set: the gate is nextThoughtNeeded.
	res := Process(td)
	if res.IsError {
		t.Fatalf("unexpected error: %s", res.Text)
	}
	if strings.Contains(res.Text, "Next, I want to") {
		t.Errorf("terminal thought must not carry a next-step line:\n%s", res.Text)
	}
}

// TestRenderTranscriptExact pins the whole transcript, byte for byte: layout,
// blank lines, and the caller's assumption order.
func TestRenderTranscriptExact(t *testing.T) {
	td := validInput(1)
	td.Assumptions = []string{"zebra", "apple"} // deliberately unsorted
	want := "Thought 1 of 3 · confidence 0.50\n\nthought number 1\n\n  Assumptions:\n    - zebra\n    - apple\n\n  Critique:\n    narrow\n\n  Counter-argument:\n    alternative\n\n  Next, I want to: next thing\n"
	if got := Process(td).Text; got != want {
		t.Errorf("transcript =\n%q\nwant\n%q", got, want)
	}
}

func TestHeaderForms(t *testing.T) {
	plain := validInput(2)

	revision := validInput(3)
	revision.IsRevision = true
	revision.RevisesThought = 1

	branch := validInput(1)
	branch.BranchFromThought = 2
	branch.BranchID = "alt"

	// A revision made while on a branch: the revision claim wins the header.
	both := validInput(4)
	both.IsRevision = true
	both.RevisesThought = 2
	both.BranchFromThought = 1
	both.BranchID = "alt"

	cases := []struct {
		name string
		td   ThoughtData
		want string
	}{
		{"plain", plain, "Thought 2 of 3 · confidence 0.50"},
		{"revision", revision, "Revision of thought 1 (now thought 3) · confidence 0.50"},
		{"branch", branch, "Branch 'alt' from thought 2 · thought 1 · confidence 0.50"},
		{"revision on branch", both, "Revision of thought 2 (now thought 4) · confidence 0.50"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := Process(tc.td)
			if res.IsError {
				t.Fatalf("unexpected error: %s", res.Text)
			}
			first, _, _ := strings.Cut(res.Text, "\n")
			if first != tc.want {
				t.Errorf("header = %q, want %q", first, tc.want)
			}
		})
	}
}
