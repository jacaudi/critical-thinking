package thinking

import (
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
