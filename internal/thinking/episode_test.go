package thinking

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// episodeResp runs one thought and returns the decoded structured response.
func episodeResp(t *testing.T, s *SequentialThinkingServer, td ThoughtData) ThoughtResponse {
	t.Helper()
	res, err := s.ProcessThought(td)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError=true, text=%s", res.Text)
	}
	var resp ThoughtResponse
	if err := json.Unmarshal([]byte(res.StructuredJSON), &resp); err != nil {
		t.Fatalf("unmarshal structured: %v", err)
	}
	return resp
}

func TestResponseEchoesEpisodeID(t *testing.T) {
	s := NewServer()

	td := validInput(1)
	td.EpisodeID = "alpha"
	resp := episodeResp(t, s, td)
	if resp.EpisodeID != "alpha" {
		t.Errorf("EpisodeID = %q, want %q", resp.EpisodeID, "alpha")
	}

	td2 := validInput(1)
	// EpisodeID left empty.
	resp2 := episodeResp(t, s, td2)
	if resp2.EpisodeID != "default" {
		t.Errorf("absent episodeId echoed as %q, want %q", resp2.EpisodeID, "default")
	}
}

func TestEpisodesAreIsolated(t *testing.T) {
	s := NewServer()

	// Episode alpha: two trunk thoughts at confidence 0.2.
	for i := range 2 {
		td := validInput(i + 1)
		td.EpisodeID = "alpha"
		td.Confidence = 0.2
		episodeResp(t, s, td)
	}

	// Episode beta: one trunk thought at confidence 0.8, plus a branch.
	tdB := validInput(1)
	tdB.EpisodeID = "beta"
	tdB.Confidence = 0.8
	respB := episodeResp(t, s, tdB)
	if respB.ThoughtHistoryLength != 1 {
		t.Errorf("beta history length = %d, want 1 (not shared with alpha)", respB.ThoughtHistoryLength)
	}

	tdBranch := validInput(2)
	tdBranch.EpisodeID = "beta"
	tdBranch.BranchFromThought = intPtr(1)
	tdBranch.BranchID = "beta-branch"
	tdBranch.Confidence = 0.8
	respBranch := episodeResp(t, s, tdBranch)
	if len(respBranch.Branches) != 1 || respBranch.Branches[0] != "beta-branch" {
		t.Errorf("beta branches = %v, want [beta-branch]", respBranch.Branches)
	}

	// Now run a third alpha thought and confirm alpha's view is unaffected by beta.
	tdA := validInput(3)
	tdA.EpisodeID = "alpha"
	tdA.Confidence = 0.2
	respA := episodeResp(t, s, tdA)
	if respA.ThoughtHistoryLength != 3 {
		t.Errorf("alpha history length = %d, want 3", respA.ThoughtHistoryLength)
	}
	if !almostEqual(respA.SessionConfidence, 0.2) {
		t.Errorf("alpha sessionConfidence = %v, want 0.2 (beta's 0.8 must not leak)", respA.SessionConfidence)
	}
	if len(respA.Branches) != 0 {
		t.Errorf("alpha branches = %v, want [] (beta's branch must not leak)", respA.Branches)
	}
}

func TestDefaultEpisodeWhenAbsent(t *testing.T) {
	s := NewServer()

	// A thought with no episodeId and a thought with episodeId:"default" must
	// land in the same episode.
	resp1 := episodeResp(t, s, validInput(1))
	if resp1.ThoughtHistoryLength != 1 {
		t.Errorf("first default thought length = %d, want 1", resp1.ThoughtHistoryLength)
	}

	td := validInput(2)
	td.EpisodeID = "default"
	resp2 := episodeResp(t, s, td)
	if resp2.ThoughtHistoryLength != 2 {
		t.Errorf("explicit default thought length = %d, want 2 (same episode as absent)", resp2.ThoughtHistoryLength)
	}
	if got := s.HistoryLength(); got != 2 {
		t.Errorf("HistoryLength() = %d, want 2 (reads default episode)", got)
	}
}

func TestRevisePerEpisodeValidation(t *testing.T) {
	s := NewServer()

	// Episode alpha has 2 thoughts; episode beta has 0.
	for i := range 2 {
		td := validInput(i + 1)
		td.EpisodeID = "alpha"
		episodeResp(t, s, td)
	}

	// Revising thought 2 in alpha is in range.
	tdA := validInput(3)
	tdA.EpisodeID = "alpha"
	tdA.IsRevision = boolPtr(true)
	tdA.RevisesThought = intPtr(2)
	if resp, err := s.ProcessThought(tdA); err != nil {
		t.Fatal(err)
	} else if resp.IsError {
		t.Errorf("revising thought 2 in alpha should be in range; got error: %s", resp.Text)
	}

	// Revising thought 2 in beta (empty) is out of range — must validate against
	// beta's history (length 0), not alpha's or the server's.
	tdB := validInput(1)
	tdB.EpisodeID = "beta"
	tdB.IsRevision = boolPtr(true)
	tdB.RevisesThought = intPtr(2)
	res, err := s.ProcessThought(tdB)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Error("revising thought 2 in empty beta should be out of range")
	}
	if !strings.Contains(res.Text, "revisesThought 2 out of range (history length 0)") {
		t.Errorf("error message should reference beta's length 0; got: %s", res.Text)
	}
}

func TestBranchFromPerEpisodeValidation(t *testing.T) {
	s := NewServer()

	// Episode alpha has 1 thought.
	tdSeed := validInput(1)
	tdSeed.EpisodeID = "alpha"
	episodeResp(t, s, tdSeed)

	// Branch from thought 1 in beta (empty) is out of range against beta.
	tdB := validInput(1)
	tdB.EpisodeID = "beta"
	tdB.BranchFromThought = intPtr(1)
	tdB.BranchID = "beta-branch"
	res, err := s.ProcessThought(tdB)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Error("branchFromThought 1 in empty beta should be out of range")
	}
	if !strings.Contains(res.Text, "branchFromThought 1 out of range (history length 0)") {
		t.Errorf("error message should reference beta's length 0; got: %s", res.Text)
	}
}

// episodeHistoryLen reads the trunk length of a named episode without mutating
// LRU order. Test-only helper; reaches into unexported state under the lock.
func episodeHistoryLen(s *SequentialThinkingServer, id string) (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ep, ok := s.episodes[id]
	if !ok {
		return 0, false
	}
	return len(ep.thoughtHistory), true
}

func TestEpisodeLRUEviction(t *testing.T) {
	s := newServerWithMax(2)

	// Create episodes a and b.
	for _, id := range []string{"a", "b"} {
		td := validInput(1)
		td.EpisodeID = id
		episodeResp(t, s, td)
	}

	// Touch a so b becomes least-recently-used.
	tdA := validInput(2)
	tdA.EpisodeID = "a"
	episodeResp(t, s, tdA)

	// Create c — this exceeds the cap of 2 and must evict the LRU episode (b).
	tdC := validInput(1)
	tdC.EpisodeID = "c"
	episodeResp(t, s, tdC)

	if _, ok := episodeHistoryLen(s, "b"); ok {
		t.Error("episode b should have been evicted as least-recently-used")
	}
	if got, ok := episodeHistoryLen(s, "a"); !ok || got != 2 {
		t.Errorf("episode a should retain its history (len 2); got len=%d ok=%v", got, ok)
	}

	// Reusing b restarts it at length 1 (fresh episode).
	tdB := validInput(1)
	tdB.EpisodeID = "b"
	respB := episodeResp(t, s, tdB)
	if respB.ThoughtHistoryLength != 1 {
		t.Errorf("re-created episode b should restart at length 1; got %d", respB.ThoughtHistoryLength)
	}
}

func TestParallelEpisodesIsolation(t *testing.T) {
	// N distinct episodes (<= defaultMaxEpisodes so none are evicted), each
	// submitting M thoughts concurrently. Every episode must end with exactly M
	// thoughts — never N*M — proving cross-episode isolation under concurrency.
	const n = 16
	const m = 25

	s := NewServer()

	var wg sync.WaitGroup
	finalLen := make([]int, n)
	for g := range n {
		wg.Go(func() {
			id := "episode-" + strconv.Itoa(g)
			last := 0
			for i := range m {
				td := validInput(i + 1)
				td.TotalThoughts = m
				td.EpisodeID = id
				res, err := s.ProcessThought(td)
				if err != nil || res.IsError {
					t.Errorf("episode %s thought %d: err=%v isError=%v text=%s", id, i+1, err, res.IsError, res.Text)
					return
				}
				var resp ThoughtResponse
				if err := json.Unmarshal([]byte(res.StructuredJSON), &resp); err != nil {
					t.Errorf("episode %s unmarshal: %v", id, err)
					return
				}
				last = resp.ThoughtHistoryLength
			}
			finalLen[g] = last
		})
	}
	wg.Wait()

	for g, got := range finalLen {
		if got != m {
			t.Errorf("episode %d final thoughtHistoryLength = %d, want %d", g, got, m)
		}
	}
}
