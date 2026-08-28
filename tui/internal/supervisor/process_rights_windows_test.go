//go:build windows

package supervisor

import "testing"

func TestJobObjectUsesMinimalProcessRights(t *testing.T) {
	const want = uintptr(0x0100 | 0x0001) // PROCESS_SET_QUOTA | PROCESS_TERMINATE
	if processJobAccess != want {
		t.Fatalf("processJobAccess = %#x, want %#x", processJobAccess, want)
	}
	if processJobAccess == uintptr(0x001F0FFF) {
		t.Fatal("processJobAccess must not use PROCESS_ALL_ACCESS")
	}
}
