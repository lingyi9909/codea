package opencode

// Task 30 verification evidence crosses the vendor boundary only through these
// three machine-owned, bounded string fields. Raw tool input/output and arbitrary
// vendor metadata remain excluded by safeTaskToolMetadata.
func init() {
	taskToolMetadataAllowlist = append(taskToolMetadataAllowlist,
		"codeaVerification",
		"codeaVerificationResult",
		"codeaVerificationProfile",
	)
}
