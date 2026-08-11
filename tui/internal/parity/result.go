package parity

// Failure describes why a scenario assertion did not pass.
type Failure struct {
	Reason string
}

// ScenarioResult holds the outcome of running a single scenario.
type ScenarioResult struct {
	Name       string
	Required   bool
	Passed     bool
	Runs       int
	Failures   []Failure
	SilentLoss bool
}

// Result aggregates parity run outcomes.
type Result struct {
	Total          int
	Passed         int
	Failed         int
	RequiredFailed int
	Scenarios      []ScenarioResult
}

func (r *Result) compute() {
	r.Total = len(r.Scenarios)
	r.Passed = 0
	r.Failed = 0
	r.RequiredFailed = 0
	for _, s := range r.Scenarios {
		if s.Passed {
			r.Passed++
		} else {
			r.Failed++
			if s.Required {
				r.RequiredFailed++
			}
		}
	}
}
