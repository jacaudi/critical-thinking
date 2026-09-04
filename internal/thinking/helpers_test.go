package thinking

import "strconv"

// validInput is a complete, valid trunk thought; tests mutate one field each.
func validInput(num int) ThoughtData {
	return ThoughtData{
		Thought:           "thought number " + strconv.Itoa(num),
		ThoughtNumber:     num,
		TotalThoughts:     3,
		NextThoughtNeeded: new(true),
		Confidence:        0.5,
		Assumptions:       []string{},
		Critique:          "narrow",
		CounterArgument:   "alternative",
		NextStepRationale: "next thing",
	}
}

// errorPayload is the JSON shape Process emits on an error result.
type errorPayload struct {
	Error  string `json:"error"`
	Status string `json:"status"`
	Hint   string `json:"hint"`
}
