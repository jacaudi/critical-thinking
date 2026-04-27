// Package thinking implements the criticalthinking MCP tool's data model
// and per-session state. It has no MCP SDK dependencies; main.go is the
// adapter that bridges this package to the SDK.
package thinking

import (
	"errors"
	"fmt"
)

// ThoughtData is the input to one criticalthinking tool call.
//
// Sent fields whose omission cannot be distinguished from their zero value
// (NextThoughtNeeded, IsRevision, RevisesThought, BranchFromThought,
// NeedsMoreThoughts) use pointer types so the validator can detect "not sent".
type ThoughtData struct {
	Thought           string `json:"thought"`
	ThoughtNumber     int    `json:"thoughtNumber"`
	TotalThoughts     int    `json:"totalThoughts"`
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
type ThoughtResponse struct {
	ThoughtNumber        int                `json:"thoughtNumber"`
	TotalThoughts        int                `json:"totalThoughts"`
	NextThoughtNeeded    bool               `json:"nextThoughtNeeded"`
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
	if td.ThoughtNumber < 1 {
		return errors.New("thoughtNumber must be ≥ 1")
	}
	if td.TotalThoughts < 1 {
		return errors.New("totalThoughts must be ≥ 1")
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
	if td.CounterArgument == "" {
		return errors.New("counterArgument must be a non-empty string")
	}
	if *td.NextThoughtNeeded && td.NextStepRationale == "" {
		return errors.New("nextStepRationale required when nextThoughtNeeded is true")
	}

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
