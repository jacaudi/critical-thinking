package thinking

import "encoding/json"

// Result is what one call produces. Text is the narrated transcript, or the
// {error, status, hint} JSON envelope when IsError. Structured is nil when
// IsError.
type Result struct {
	Text       string
	Structured *ThoughtResponse
	IsError    bool
}

// Process validates one thought and renders it. It is a pure function of its
// argument: it reads and writes no package or process state, so identical
// inputs give identical results and concurrent calls need no synchronisation.
func Process(td ThoughtData) Result {
	if err := td.Validate(); err != nil {
		return errorResult(err)
	}
	if td.ThoughtNumber > td.TotalThoughts {
		td.TotalThoughts = td.ThoughtNumber
	}
	return Result{
		Text: render(td),
		Structured: &ThoughtResponse{
			ThoughtNumber:     td.ThoughtNumber,
			TotalThoughts:     td.TotalThoughts,
			NextThoughtNeeded: *td.NextThoughtNeeded,
			Confidence:        td.Confidence,
		},
	}
}

// errorResult encodes the on-wire error envelope. The hint is the
// required-fields checklist so a caller can self-correct without re-reading
// the tool description.
func errorResult(err error) Result {
	// A fixed-shape struct of strings cannot fail to marshal.
	body, _ := json.Marshal(struct {
		Error  string `json:"error"`
		Status string `json:"status"`
		Hint   string `json:"hint"`
	}{Error: err.Error(), Status: "failed", Hint: requiredFieldsChecklist})
	return Result{Text: string(body), IsError: true}
}
