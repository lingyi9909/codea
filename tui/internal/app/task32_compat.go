package app

// maxVerificationContinuations is the Task 30 legacy/default continuation
// budget retained for compatibility. Task 32 freezes the effective budget per
// root task in TaskExecutionState.VerificationContinuationLimit.
const maxVerificationContinuations = 2
