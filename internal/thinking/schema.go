// Package thinking implements the criticalthinking tool: the input contract
// (ThoughtData), its validation, and Process, the pure function that turns one
// thought into a narrated transcript plus a structured echo. The package keeps
// nothing between calls and imports neither the MCP SDK nor OpenTelemetry;
// cmd/critical-thinking is the only adapter.
package thinking

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

// ThoughtData is the input to one criticalthinking call.
//
// NextThoughtNeeded is a pointer so a missing field can be told apart from an
// explicit false on the path that has no JSON-schema gate (the cli command).
// Every other optional field uses its zero value as "absent".
type ThoughtData struct {
	Thought           string `json:"thought"`
	ThoughtNumber     int    `json:"thoughtNumber"`
	TotalThoughts     int    `json:"totalThoughts"`
	NextThoughtNeeded *bool  `json:"nextThoughtNeeded"`
	IsRevision        bool   `json:"isRevision,omitempty" jsonschema:"This thought corrects an earlier one. Send together with revisesThought, or omit both."`
	RevisesThought    int    `json:"revisesThought,omitempty" jsonschema:"The thoughtNumber being corrected (≥ 1). Send together with isRevision=true, or omit both."`
	BranchFromThought int    `json:"branchFromThought,omitempty" jsonschema:"The thoughtNumber this alternative branches from (≥ 1). Send together with branchId, or omit both."`
	BranchID          string `json:"branchId,omitempty" jsonschema:"Name of the alternative being explored. Send together with branchFromThought, or omit both."`

	Confidence        float64  `json:"confidence"`
	Assumptions       []string `json:"assumptions"`
	Critique          string   `json:"critique"`
	CounterArgument   string   `json:"counterArgument"`
	NextStepRationale string   `json:"nextStepRationale,omitempty"`

	// Deprecated: accepted and ignored. The generated JSON schema forbids
	// unknown properties, so this stays until the next breaking release so
	// that clients written for v1.15 and earlier keep validating.
	EpisodeID string `json:"episodeId,omitempty" jsonschema:"Deprecated and ignored: the server keeps no state, so there is nothing to isolate. Omit it."`
	// Deprecated: accepted and ignored; see EpisodeID.
	NeedsMoreThoughts bool `json:"needsMoreThoughts,omitempty" jsonschema:"Deprecated and ignored. Adjust totalThoughts instead."`
}

// ThoughtResponse is the structured echo of one call: the routing fields the
// caller needs to plan the next call, with TotalThoughts after the clamp.
type ThoughtResponse struct {
	ThoughtNumber     int     `json:"thoughtNumber"`
	TotalThoughts     int     `json:"totalThoughts"`
	NextThoughtNeeded bool    `json:"nextThoughtNeeded"`
	Confidence        float64 `json:"confidence"`
}

// requiredFieldsChecklist is the one-line input contract shared by the tool
// description's lead-in and the validation-error hint, so the two cannot
// drift. It restates the rules Validate enforces; keep the two in step.
const requiredFieldsChecklist = "Every call requires: thought, thoughtNumber, totalThoughts, nextThoughtNeeded, confidence, assumptions, critique, counterArgument — plus nextStepRationale whenever nextThoughtNeeded=true."

// Validate enforces every input rule; "present" means non-blank for strings.
// It delegates to one helper per validation concern, in the fixed order
// callers and tests depend on: the first violation found is the error
// returned.
func (td ThoughtData) Validate() error {
	if err := td.validateRequiredFields(); err != nil {
		return err
	}
	if err := td.validateAssumptions(); err != nil {
		return err
	}
	if err := td.validateNarrative(); err != nil {
		return err
	}
	if err := td.validateRevisionPair(); err != nil {
		return err
	}
	return td.validateBranchPair()
}

// validateRequiredFields checks the fields every call must send: the thought
// text, the two counters, the routing flag, and confidence's range.
func (td ThoughtData) validateRequiredFields() error {
	if strings.TrimSpace(td.Thought) == "" {
		return errors.New("thought must be a non-blank string")
	}
	if td.ThoughtNumber < 1 {
		return fmt.Errorf("thoughtNumber must be ≥ 1 (got %d)", td.ThoughtNumber)
	}
	if td.TotalThoughts < 1 {
		return fmt.Errorf("totalThoughts must be ≥ 1 (got %d)", td.TotalThoughts)
	}
	if td.NextThoughtNeeded == nil {
		return errors.New("nextThoughtNeeded must be present (true or false)")
	}
	if math.IsNaN(td.Confidence) || td.Confidence < 0.0 || td.Confidence > 1.0 {
		return fmt.Errorf("confidence must be between 0.0 and 1.0 (got %v)", td.Confidence)
	}
	return nil
}

// validateAssumptions checks that the slice is present (possibly empty) and
// that every entry, if any, is non-blank.
func (td ThoughtData) validateAssumptions() error {
	if td.Assumptions == nil {
		return errors.New("assumptions must be present (use [] if none)")
	}
	for i, a := range td.Assumptions {
		if strings.TrimSpace(a) == "" {
			return fmt.Errorf("assumptions[%d] must be a non-blank string (use [] if you claim none)", i)
		}
	}
	return nil
}

// validateNarrative checks the free-text fields that must be non-blank:
// critique and counterArgument always, nextStepRationale only when more
// thinking is needed. NextThoughtNeeded is guaranteed non-nil here because
// validateRequiredFields always runs first.
func (td ThoughtData) validateNarrative() error {
	if strings.TrimSpace(td.Critique) == "" {
		return errors.New("critique must be a non-blank string")
	}
	if strings.TrimSpace(td.CounterArgument) == "" {
		return errors.New("counterArgument must be a non-blank string")
	}
	if *td.NextThoughtNeeded && strings.TrimSpace(td.NextStepRationale) == "" {
		return errors.New("nextStepRationale required when nextThoughtNeeded is true; send one, or set nextThoughtNeeded=false if this line of thinking is done")
	}
	return nil
}

// validateRevisionPair checks that isRevision and revisesThought are sent
// together or not at all, and that revisesThought is in range.
func (td ThoughtData) validateRevisionPair() error {
	if td.IsRevision != (td.RevisesThought != 0) {
		return errors.New("set isRevision=true together with revisesThought ≥ 1, or omit both")
	}
	if td.RevisesThought < 0 {
		return fmt.Errorf("revisesThought must be ≥ 1 (got %d)", td.RevisesThought)
	}
	return nil
}

// validateBranchPair checks that branchFromThought and branchId are sent
// together or not at all, and that branchFromThought is in range.
func (td ThoughtData) validateBranchPair() error {
	if (td.BranchFromThought != 0) != (td.BranchID != "") {
		return errors.New("send branchFromThought ≥ 1 together with branchId, or omit both")
	}
	if td.BranchFromThought < 0 {
		return fmt.Errorf("branchFromThought must be ≥ 1 (got %d)", td.BranchFromThought)
	}
	return nil
}
