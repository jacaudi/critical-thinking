// Package thinking implements the criticalthinking MCP tool's data model
// and per-session state. It has no MCP SDK dependencies; main.go is the
// adapter that bridges this package to the SDK.
package thinking

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

// Length caps on the critical-thinking fields. Limits are in runes, not bytes,
// so multi-byte text (em-dashes, accented chars) doesn't get unfairly truncated.
// These force one-sentence-per-field discipline; padded prose returns an error.
const (
	maxCritiqueLen          = 280
	maxCounterArgumentLen   = 280
	maxNextStepRationaleLen = 200
	maxAssumptionLen        = 200
)

// ThoughtData is the input to one criticalthinking tool call.
//
// Sent fields whose omission cannot be distinguished from their zero value
// (ThoughtNumber, TotalThoughts, NextThoughtNeeded, IsRevision, RevisesThought,
// BranchFromThought, NeedsMoreThoughts) use pointer types so the validator can
// detect "not sent". For ThoughtNumber/TotalThoughts, "not sent" is a signal
// for server-side auto-assign (next sequential / inherit prior).
type ThoughtData struct {
	Thought           string `json:"thought"`
	ThoughtNumber     *int   `json:"thoughtNumber,omitempty"`
	TotalThoughts     *int   `json:"totalThoughts,omitempty"`
	NextThoughtNeeded *bool  `json:"nextThoughtNeeded"`
	IsRevision        *bool  `json:"isRevision,omitempty"`
	RevisesThought    *int   `json:"revisesThought,omitempty"`
	BranchFromThought *int   `json:"branchFromThought,omitempty"`
	BranchID          string `json:"branchId,omitempty"`
	NeedsMoreThoughts *bool  `json:"needsMoreThoughts,omitempty"`

	Confidence        float64  `json:"confidence"`
	Assumptions       []string `json:"assumptions"`
	Critique          string   `json:"critique"`
	CounterArgument   string   `json:"counterArgument"`
	NextStepRationale string   `json:"nextStepRationale,omitempty"`
}

// ThoughtResponse is the structuredContent of a criticalthinking tool call.
//
// Echo fields the caller already sent (thoughtNumber, totalThoughts,
// nextThoughtNeeded) are deliberately omitted to save tokens. The server
// tracks them internally; callers can read them back from the
// thinking://current resource if needed.
type ThoughtResponse struct {
	Branches             []string           `json:"branches"`
	ThoughtHistoryLength int                `json:"thoughtHistoryLength"`
	SessionConfidence    float64            `json:"sessionConfidence"`
	BranchConfidences    map[string]float64 `json:"branchConfidences,omitempty"`
}

// Validate enforces every wire-format rule for ThoughtData except those that
// require knowing the current session state (RevisesThought / BranchFromThought
// range checks). Those run separately in SequentialThinkingServer.ProcessThought.
//
// Returns the first error encountered; callers that want all errors should
// extend this with a multi-error type later.
func (td ThoughtData) Validate() error {
	if td.Thought == "" {
		return errors.New("thought must be a non-empty string")
	}
	if td.ThoughtNumber != nil && *td.ThoughtNumber < 1 {
		return errors.New("thoughtNumber must be ≥ 1 (omit to auto-assign)")
	}
	if td.TotalThoughts != nil && *td.TotalThoughts < 1 {
		return errors.New("totalThoughts must be ≥ 1 (omit to inherit prior)")
	}
	if td.NextThoughtNeeded == nil {
		return errors.New("nextThoughtNeeded must be present (true or false)")
	}
	if td.Confidence < 0.0 || td.Confidence > 1.0 {
		return fmt.Errorf("confidence must be between 0.0 and 1.0 (got %v)", td.Confidence)
	}
	if td.Assumptions == nil {
		return errors.New("assumptions must be present (use [] if none)")
	}
	if td.Critique == "" {
		return errors.New("critique must be a non-empty string")
	}
	if n := utf8.RuneCountInString(td.Critique); n > maxCritiqueLen {
		return fmt.Errorf("critique must be ≤ %d chars, got %d (one tight sentence; the thought field is for narration)", maxCritiqueLen, n)
	}
	if td.CounterArgument == "" {
		return errors.New("counterArgument must be a non-empty string")
	}
	if n := utf8.RuneCountInString(td.CounterArgument); n > maxCounterArgumentLen {
		return fmt.Errorf("counterArgument must be ≤ %d chars, got %d (one tight sentence; the thought field is for narration)", maxCounterArgumentLen, n)
	}
	for i, a := range td.Assumptions {
		if n := utf8.RuneCountInString(a); n > maxAssumptionLen {
			return fmt.Errorf("assumptions[%d] must be ≤ %d chars, got %d (one fact per entry)", i, maxAssumptionLen, n)
		}
	}
	if *td.NextThoughtNeeded {
		if td.NextStepRationale == "" {
			return errors.New("nextStepRationale required when nextThoughtNeeded is true")
		}
		if n := utf8.RuneCountInString(td.NextStepRationale); n > maxNextStepRationaleLen {
			return fmt.Errorf("nextStepRationale must be ≤ %d chars, got %d (one sentence)", maxNextStepRationaleLen, n)
		}
	}
	// When nextThoughtNeeded=false, NextStepRationale is logically absent — we
	// don't enforce the length cap. Clients are advised to OMIT the field; a
	// stale value here is benign and ignored.

	// Both-or-neither rule for branch fields.
	hasFrom := td.BranchFromThought != nil
	hasID := td.BranchID != ""
	if hasFrom != hasID {
		return errors.New("branchFromThought and branchId must both be present or both omitted")
	}
	if hasFrom && *td.BranchFromThought < 1 {
		return fmt.Errorf("branchFromThought must be ≥ 1 (got %d)", *td.BranchFromThought)
	}
	if td.RevisesThought != nil && *td.RevisesThought < 1 {
		return fmt.Errorf("revisesThought must be ≥ 1 (got %d)", *td.RevisesThought)
	}

	return nil
}
