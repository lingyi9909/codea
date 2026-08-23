package update

import (
	"strings"
	"testing"
)

func TestNewServiceRequiresCandidateChecker(t *testing.T) {
	_, err := NewService(ServiceConfig{HomeDir: t.TempDir()})
	if err == nil {
		t.Fatal("NewService must fail closed when CandidateChecker is missing")
	}
	if !strings.Contains(err.Error(), "checker") {
		t.Fatalf("error = %q, want checker-related failure", err)
	}
}
