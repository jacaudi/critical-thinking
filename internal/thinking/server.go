package thinking

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// SequentialThinkingServer holds the per-session state for one client of the
// criticalthinking tool. Construct exactly one per session: in HTTP mode this
// happens inside the StreamableHTTP factory closure; in stdio mode there is
// one global instance for the process.
//
// The factory-closure pattern is the cross-session isolation invariant. There
// is intentionally no map keyed by session-id anywhere — the closure scope is
// the only addressable path to a session's state.
type SequentialThinkingServer struct {
	mu             sync.Mutex
	thoughtHistory []ThoughtData
	branches       map[string][]ThoughtData
	confidenceSum  float64
	confidenceN    int
	branchConfSum  map[string]float64
	branchConfN    map[string]int
	lastAccessed   time.Time
}

// NewServer returns an empty SequentialThinkingServer.
func NewServer() *SequentialThinkingServer {
	return &SequentialThinkingServer{
		branches:      make(map[string][]ThoughtData),
		branchConfSum: make(map[string]float64),
		branchConfN:   make(map[string]int),
		lastAccessed:  time.Now(),
	}
}

// HistoryLength returns the number of thoughts in the trunk + branches
// (a single append-only log).
func (s *SequentialThinkingServer) HistoryLength() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.thoughtHistory)
}

// SessionConfidence returns the running mean confidence over trunk thoughts.
// Returns 0 when no trunk thoughts have been recorded.
func (s *SequentialThinkingServer) SessionConfidence() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.confidenceN == 0 {
		return 0
	}
	return s.confidenceSum / float64(s.confidenceN)
}

// LastAccessed returns the time of the last successful ProcessThought call.
// Used by the HTTP idle-timeout cleanup goroutine in main.go.
func (s *SequentialThinkingServer) LastAccessed() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastAccessed
}

// ToolResult is the package-internal return type from ProcessThought. main.go
// adapts it into a *mcp.CallToolResult — keeping mcp imports out of this
// package preserves its testability.
type ToolResult struct {
	Text           string // the rubber-duck transcript (or error JSON when IsError)
	StructuredJSON string // JSON-encoded ThoughtResponse, "" when IsError
	IsError        bool
}

// ProcessThought validates input, mutates state, and returns either a
// transcript+structured response or an error result. The Go-level error
// return is reserved for unrecoverable internal faults (currently never
// returned); validation failures produce IsError=true results.
func (s *SequentialThinkingServer) ProcessThought(td ThoughtData) (ToolResult, error) {
	if err := td.Validate(); err != nil {
		return errorResult(err), nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Cross-field validation against state.
	if td.RevisesThought != nil && *td.RevisesThought > len(s.thoughtHistory) {
		return errorResult(fmt.Errorf("revisesThought %d out of range (history length %d)",
			*td.RevisesThought, len(s.thoughtHistory))), nil
	}
	if td.BranchFromThought != nil && *td.BranchFromThought > len(s.thoughtHistory) {
		return errorResult(fmt.Errorf("branchFromThought %d out of range (history length %d)",
			*td.BranchFromThought, len(s.thoughtHistory))), nil
	}

	if td.ThoughtNumber > td.TotalThoughts {
		td.TotalThoughts = td.ThoughtNumber
	}

	s.thoughtHistory = append(s.thoughtHistory, td)
	s.lastAccessed = time.Now()

	// Trunk-only confidence aggregation in this task; per-branch added in Task 8.
	s.confidenceSum += td.Confidence
	s.confidenceN++

	resp := ThoughtResponse{
		ThoughtNumber:        td.ThoughtNumber,
		TotalThoughts:        td.TotalThoughts,
		NextThoughtNeeded:    *td.NextThoughtNeeded,
		Branches:             []string{}, // populated in Task 7
		ThoughtHistoryLength: len(s.thoughtHistory),
		SessionConfidence:    s.confidenceSum / float64(s.confidenceN),
	}

	structured, err := json.Marshal(resp)
	if err != nil {
		// Should be impossible for fixed-shape struct.
		return errorResult(fmt.Errorf("marshal response: %w", err)), nil
	}

	return ToolResult{
		Text:           "Thought " + itoaInt(td.ThoughtNumber) + " of " + itoaInt(td.TotalThoughts), // expanded in Task 9
		StructuredJSON: string(structured),
		IsError:        false,
	}, nil
}

// errorResult formats a validation/runtime error in the JS-compatible
// {error, status: "failed"} shape.
func errorResult(err error) ToolResult {
	body, _ := json.Marshal(struct {
		Error  string `json:"error"`
		Status string `json:"status"`
	}{Error: err.Error(), Status: "failed"})
	return ToolResult{Text: string(body), IsError: true}
}

func itoaInt(i int) string {
	return fmt.Sprintf("%d", i)
}
