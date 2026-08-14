package components

import "strings"

// dangerousCommands lists high-risk command fragments spanning both Unix-like
// shells and Windows cmd/PowerShell. Detection is a UI/Policy warning only: it
// strengthens the approval prompt and defaults focus to Reject. It is not a
// security engine — no shell parsing or sandboxing is performed here.
var dangerousCommands = []string{
	"rm -rf",
	"git reset --hard",
	"git clean -fd",
	"git push --force",
	"chmod 777",
	"> /dev/sda",
	"mkfs.",
	"dd if=",
	":(){ :|:& };:",
	"del /s /q",
	"rmdir /s /q",
	"rd /s /q",
	"format",
	"diskpart",
	"remove-item -recurse -force",
	"stop-computer",
	"restart-computer",
}

// IsDangerousCommand reports whether input contains a known dangerous command
// fragment, returning the matched fragment. Matching is case-insensitive and
// ignores leading/trailing whitespace.
func IsDangerousCommand(input string) (bool, string) {
	lower := strings.ToLower(strings.TrimSpace(input))
	for _, cmd := range dangerousCommands {
		if strings.Contains(lower, cmd) {
			return true, cmd
		}
	}
	return false, ""
}
