package thinking

import (
	"container/list"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"
)

// defaultMaxEpisodes bounds the number of logical episodes retained per server.
// The least-recently-used episode is evicted when a new episode exceeds it.
const defaultMaxEpisodes = 64

// episode holds the state of one logical reasoning episode. The fields were
// previously flat on SequentialThinkingServer; they now live per-episode so
// independent reasoning episodes within one transport session do not
// contaminate each other's history, confidence, or branches.
type episode struct {
	thoughtHistory []ThoughtData
	branches       map[string][]ThoughtData
	confidenceSum  float64
	confidenceN    int
	branchConfSum  map[string]float64
	branchConfN    map[string]int
	lastAccessed   time.Time
}

func newEpisode() *episode {
	return &episode{
		branches:      make(map[string][]ThoughtData),
		branchConfSum: make(map[string]float64),
		branchConfN:   make(map[string]int),
		lastAccessed:  time.Now(),
	}
}

// SequentialThinkingServer holds the per-session state for one client of the
// criticalthinking tool. Construct exactly one per session: in HTTP mode this
// happens inside the StreamableHTTP factory closure; in stdio mode there is
// one global instance for the process.
//
// The factory-closure pattern is the cross-session isolation invariant. There
// is intentionally no map keyed by session-id anywhere — the closure scope is
// the only addressable path to a session's state.
type SequentialThinkingServer struct {
	mu sync.Mutex
	// episodes partitions state by a client-supplied episodeId tool argument.
	//
	// Compatible divergence from the rubber-ducky design: this map is keyed by
	// a CLIENT-SUPPLIED TOOL ARGUMENT (episodeId), scoped WITHIN this single
	// transport session's closure — NOT by mcp-session-id. One connection still
	// cannot address another connection's state, so the cross-session isolation
	// invariant (TestCrossSessionIsolation) is untouched.
	episodes    map[string]*episode
	lru         *list.List               // front = most-recently-used; Value = episodeId string
	lruIndex    map[string]*list.Element // episodeId -> its lru element
	maxEpisodes int
}

// NewServer returns an empty SequentialThinkingServer bounded to
// defaultMaxEpisodes logical episodes.
func NewServer() *SequentialThinkingServer {
	return newServerWithMax(defaultMaxEpisodes)
}

// newServerWithMax returns an empty server with a custom episode cap. Tests use
// a small cap to exercise LRU eviction deterministically.
func newServerWithMax(max int) *SequentialThinkingServer {
	return &SequentialThinkingServer{
		episodes:    make(map[string]*episode),
		lru:         list.New(),
		lruIndex:    make(map[string]*list.Element),
		maxEpisodes: max,
	}
}

// getOrCreateEpisodeLocked returns the episode for id, creating it if absent.
// Every access (found or created) marks the episode most-recently-used so
// active episodes are not evicted. Creating a new episode that pushes the count
// over maxEpisodes evicts the least-recently-used episode. Caller holds s.mu.
func (s *SequentialThinkingServer) getOrCreateEpisodeLocked(id string) *episode {
	if elem, ok := s.lruIndex[id]; ok {
		s.lru.MoveToFront(elem)
		return s.episodes[id]
	}

	ep := newEpisode()
	s.episodes[id] = ep
	s.lruIndex[id] = s.lru.PushFront(id)

	if len(s.episodes) > s.maxEpisodes {
		s.evictLRULocked()
	}
	return ep
}

// evictLRULocked removes the least-recently-used episode (the lru back element).
// Caller holds s.mu.
func (s *SequentialThinkingServer) evictLRULocked() {
	back := s.lru.Back()
	if back == nil {
		return
	}
	evictedKey := back.Value.(string)
	evictedLen := len(s.episodes[evictedKey].thoughtHistory)

	s.lru.Remove(back)
	delete(s.episodes, evictedKey)
	delete(s.lruIndex, evictedKey)

	slog.Warn("evicted least-recently-used thinking episode",
		"episodeId", evictedKey,
		"thoughtHistoryLength", evictedLen,
		"maxEpisodes", s.maxEpisodes)
}

// defaultEpisodeLocked returns the "default" episode without creating it or
// touching the LRU order, so informational accessors stay side-effect-free.
// Returns nil when the default episode does not exist yet. Caller holds s.mu.
func (s *SequentialThinkingServer) defaultEpisodeLocked() *episode {
	return s.episodes["default"]
}

// HistoryLength returns the number of thoughts in the default episode's trunk +
// branches (a single append-only log). Side-effect-free read.
func (s *SequentialThinkingServer) HistoryLength() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	ep := s.defaultEpisodeLocked()
	if ep == nil {
		return 0
	}
	return len(ep.thoughtHistory)
}

// SessionConfidence returns the running mean confidence over the default
// episode's trunk thoughts. Returns 0 when no trunk thoughts have been
// recorded. Side-effect-free read.
func (s *SequentialThinkingServer) SessionConfidence() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	ep := s.defaultEpisodeLocked()
	if ep == nil || ep.confidenceN == 0 {
		return 0
	}
	return ep.confidenceSum / float64(ep.confidenceN)
}

// LastAccessed returns the time of the last successful ProcessThought call on
// the default episode. Informational only — HTTP idle-session lifecycle is
// driven by the SDK's SessionTimeout, not by this value. Side-effect-free read.
func (s *SequentialThinkingServer) LastAccessed() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	ep := s.defaultEpisodeLocked()
	if ep == nil {
		return time.Time{}
	}
	return ep.lastAccessed
}

// ToolResult is the package-internal return type from ProcessThought. main.go
// adapts it into a *mcp.CallToolResult — keeping mcp imports out of this
// package preserves its testability.
type ToolResult struct {
	Text           string // the thinking-out-loud transcript (or error JSON when IsError)
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

	// Resolve the logical episode. Empty means the "default" episode. This is
	// done here, not in Validate(), so Validate() stays state-free.
	episodeID := td.EpisodeID
	if episodeID == "" {
		episodeID = "default"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ep := s.getOrCreateEpisodeLocked(episodeID)

	// Cross-field validation against the EPISODE's state.
	if td.RevisesThought != nil && *td.RevisesThought > len(ep.thoughtHistory) {
		return errorResult(fmt.Errorf("revisesThought %d out of range (history length %d)",
			*td.RevisesThought, len(ep.thoughtHistory))), nil
	}
	if td.BranchFromThought != nil && *td.BranchFromThought > len(ep.thoughtHistory) {
		return errorResult(fmt.Errorf("branchFromThought %d out of range (history length %d)",
			*td.BranchFromThought, len(ep.thoughtHistory))), nil
	}

	if td.ThoughtNumber > td.TotalThoughts {
		td.TotalThoughts = td.ThoughtNumber
	}

	ep.thoughtHistory = append(ep.thoughtHistory, td)
	ep.lastAccessed = time.Now()

	if td.BranchFromThought != nil && td.BranchID != "" {
		ep.branches[td.BranchID] = append(ep.branches[td.BranchID], td)
	}

	onBranch := td.BranchFromThought != nil && td.BranchID != ""
	if onBranch {
		ep.branchConfSum[td.BranchID] += td.Confidence
		ep.branchConfN[td.BranchID]++
	} else {
		ep.confidenceSum += td.Confidence
		ep.confidenceN++
	}

	var branchConf map[string]float64
	if len(ep.branchConfN) > 0 {
		branchConf = make(map[string]float64, len(ep.branchConfN))
		for k, n := range ep.branchConfN {
			branchConf[k] = ep.branchConfSum[k] / float64(n)
		}
	}

	sessionConf := 0.0
	if ep.confidenceN > 0 {
		sessionConf = ep.confidenceSum / float64(ep.confidenceN)
	}

	resp := ThoughtResponse{
		ThoughtNumber:        td.ThoughtNumber,
		TotalThoughts:        td.TotalThoughts,
		NextThoughtNeeded:    *td.NextThoughtNeeded,
		Branches:             sortedKeys(ep.branches),
		ThoughtHistoryLength: len(ep.thoughtHistory),
		SessionConfidence:    sessionConf,
		BranchConfidences:    branchConf,
		EpisodeID:            episodeID,
	}

	structured, err := json.Marshal(resp)
	if err != nil {
		// Should be impossible for fixed-shape struct.
		return errorResult(fmt.Errorf("marshal response: %w", err)), nil
	}

	return ToolResult{
		Text:           ep.renderTranscript(td, sessionConf),
		StructuredJSON: string(structured),
		IsError:        false,
	}, nil
}

// renderTranscript builds the narrated transcript text for one thought.
// Caller must hold s.mu (the episode is reached only through the locked server).
func (e *episode) renderTranscript(td ThoughtData, sessionConf float64) string {
	var b strings.Builder

	header := e.headerLine(td)
	fmt.Fprintf(&b, "%s\n\n", header)
	fmt.Fprintln(&b, td.Thought)
	fmt.Fprintln(&b)

	if len(td.Assumptions) == 0 {
		fmt.Fprintln(&b, "  Assumptions: (none claimed)")
	} else {
		fmt.Fprintln(&b, "  Assumptions:")
		for _, a := range td.Assumptions {
			fmt.Fprintf(&b, "    - %s\n", a)
		}
	}
	fmt.Fprintln(&b)

	fmt.Fprintln(&b, "  Critique:")
	fmt.Fprintf(&b, "    %s\n\n", td.Critique)

	fmt.Fprintln(&b, "  Counter-argument:")
	fmt.Fprintf(&b, "    %s\n\n", td.CounterArgument)

	if *td.NextThoughtNeeded {
		fmt.Fprintf(&b, "  Next, I want to: %s\n\n", td.NextStepRationale)
	}

	e.renderFooter(&b, td, sessionConf)
	return b.String()
}

// headerLine picks one of four header forms based on revision/branch state.
// Caller must hold s.mu.
func (e *episode) headerLine(td ThoughtData) string {
	switch {
	case td.IsRevision != nil && *td.IsRevision && td.RevisesThought != nil:
		return fmt.Sprintf("Revision of thought %d (now thought %d) · confidence %.2f",
			*td.RevisesThought, td.ThoughtNumber, td.Confidence)
	case td.BranchFromThought != nil && td.BranchID != "":
		// First-in-branch vs subsequent: count the current branch's depth.
		// At this point the new thought has already been appended to e.branches[BranchID].
		depth := len(e.branches[td.BranchID])
		if depth <= 1 {
			return fmt.Sprintf("Branch '%s' from thought %d · confidence %.2f",
				td.BranchID, *td.BranchFromThought, td.Confidence)
		}
		return fmt.Sprintf("Branch '%s' · thought %d · confidence %.2f",
			td.BranchID, td.ThoughtNumber, td.Confidence)
	default:
		return fmt.Sprintf("Thought %d of %d · confidence %.2f",
			td.ThoughtNumber, td.TotalThoughts, td.Confidence)
	}
}

// renderFooter writes either the trunk or branch+trunk footer.
// Caller must hold s.mu.
func (e *episode) renderFooter(b *strings.Builder, td ThoughtData, sessionConf float64) {
	onBranch := td.BranchFromThought != nil && td.BranchID != ""
	if onBranch {
		bn := e.branchConfN[td.BranchID]
		bc := 0.0
		if bn > 0 {
			bc = e.branchConfSum[td.BranchID] / float64(bn)
		}
		bnoun := "thought"
		if bn != 1 {
			bnoun = "thoughts"
		}
		fmt.Fprintf(b, "— branch '%s' confidence %.2f across %d %s\n",
			td.BranchID, bc, bn, bnoun)
		tnoun := "thought"
		if e.confidenceN != 1 {
			tnoun = "thoughts"
		}
		fmt.Fprintf(b, "— session confidence (trunk) %.2f across %d %s",
			sessionConf, e.confidenceN, tnoun)
		return
	}
	noun := "thought"
	if e.confidenceN != 1 {
		noun = "thoughts"
	}
	fmt.Fprintf(b, "— session confidence %.2f across %d %s",
		sessionConf, e.confidenceN, noun)
}

// errorResult formats a validation/runtime error in the JS-compatible
// {error, status: "failed"} shape.
func errorResult(err error) ToolResult {
	// Marshaling a fixed-shape {string,string} struct cannot fail.
	body, _ := json.Marshal(struct {
		Error  string `json:"error"`
		Status string `json:"status"`
	}{Error: err.Error(), Status: "failed"})
	return ToolResult{Text: string(body), IsError: true}
}

func sortedKeys(m map[string][]ThoughtData) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// HistorySnapshot returns a deep copy of the trunk + branch thought history,
// safe to marshal and ship to a resource consumer. Branches are keyed by id.
type HistorySnapshot struct {
	Thoughts []ThoughtData            `json:"thoughts"`
	Branches map[string][]ThoughtData `json:"branches,omitempty"`
}

// Snapshot returns the current state for the thinking://current resource.
// The returned slices and map are safe to mutate without affecting the server.
//
// The resource intentionally exposes only the "default" episode: surfacing
// per-episode state through this resource would risk the same cross-exposure
// the resource handler already guards against. Side-effect-free read.
func (s *SequentialThinkingServer) Snapshot() HistorySnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	ep := s.defaultEpisodeLocked()
	if ep == nil {
		return HistorySnapshot{Thoughts: []ThoughtData{}}
	}

	thoughts := make([]ThoughtData, len(ep.thoughtHistory))
	copy(thoughts, ep.thoughtHistory)

	var branches map[string][]ThoughtData
	if len(ep.branches) > 0 {
		branches = make(map[string][]ThoughtData, len(ep.branches))
		for k, v := range ep.branches {
			cp := make([]ThoughtData, len(v))
			copy(cp, v)
			branches[k] = cp
		}
	}
	return HistorySnapshot{Thoughts: thoughts, Branches: branches}
}
