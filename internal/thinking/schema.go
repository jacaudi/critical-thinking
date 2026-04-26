// Package thinking implements the criticalthinking MCP tool's data model
// and per-session state. It has no MCP SDK dependencies; main.go is the
// adapter that bridges this package to the SDK.
package thinking

// ThoughtData is the input to one criticalthinking tool call.
//
// Sent fields whose omission cannot be distinguished from their zero value
// (NextThoughtNeeded, IsRevision, RevisesThought, BranchFromThought,
// NeedsMoreThoughts) use pointer types so the validator can detect "not sent".
type ThoughtData struct {
	Thought           string  `json:"thought"`
	ThoughtNumber     int     `json:"thoughtNumber"`
	TotalThoughts     int     `json:"totalThoughts"`
	NextThoughtNeeded *bool   `json:"nextThoughtNeeded"`
	IsRevision        *bool   `json:"isRevision,omitempty"`
	RevisesThought    *int    `json:"revisesThought,omitempty"`
	BranchFromThought *int    `json:"branchFromThought,omitempty"`
	BranchID          string  `json:"branchId,omitempty"`
	NeedsMoreThoughts *bool   `json:"needsMoreThoughts,omitempty"`

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
