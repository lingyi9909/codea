package app

// VerificationDecision is derived only from machine-observed execution state.
// Assistant prose is intentionally absent from this contract.
type VerificationDecision string

const (
	VerifyNotRequired VerificationDecision = "not_required"
	VerifyAccepted    VerificationDecision = "accepted"
	VerifyMissing     VerificationDecision = "missing"
	VerifyFailed      VerificationDecision = "failed"
)

func verificationDecision(state TaskExecutionState) VerificationDecision {
	if !state.MutationSeen {
		return VerifyNotRequired
	}
	if state.VerifyPassed && state.LastVerificationResult == "pass" {
		return VerifyAccepted
	}
	if state.LastVerificationResult == "" {
		return VerifyMissing
	}
	return VerifyFailed
}

func traceStatusForVerification(decision VerificationDecision) TraceStatus {
	switch decision {
	case VerifyNotRequired, VerifyAccepted:
		return TraceSuccess
	default:
		return TraceUnverified
	}
}

func verificationMetricCategory(state TaskExecutionState, decision VerificationDecision) string {
	switch decision {
	case VerifyMissing:
		return "verification_missing"
	case VerifyFailed:
		if state.LastVerificationResult != "" {
			return "verification_" + state.LastVerificationResult
		}
		return "verification_failed"
	default:
		return ""
	}
}

// finishStepWithVerification owns terminal state for Runtime step.finished.
// Step 6 will interpose bounded continuations before calling this terminal path.
func (m *Model) finishStepWithVerification() {
	decision := verificationDecision(m.taskExecution)
	m.finishActiveTurnTrace(traceStatusForVerification(decision))
	if decision == VerifyNotRequired || decision == VerifyAccepted {
		m.finishStreaming()
		return
	}
	m.finishStreamingWithOutcome(MetricStatusFailed, verificationMetricCategory(m.taskExecution, decision), false)
}
