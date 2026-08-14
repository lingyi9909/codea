package components

import "testing"

func TestDangerousUnixCommandsDetected(t *testing.T) {
	cases := map[string]string{
		"rm -rf /":                 "rm -rf",
		"git reset --hard HEAD":    "git reset --hard",
		"git clean -fd":            "git clean -fd",
		"git push --force origin":  "git push --force",
		"chmod 777 /etc/passwd":    "chmod 777",
		"echo x > /dev/sda":        "> /dev/sda",
		"mkfs.ext4 /dev/sdb1":      "mkfs.",
		"dd if=/dev/zero of=/dev/sda": "dd if=",
	}
	for input, want := range cases {
		got, matched := IsDangerousCommand(input)
		if !got || matched != want {
			t.Errorf("IsDangerousCommand(%q) = (%v, %q), want (true, %q)", input, got, matched, want)
		}
	}
}

func TestDangerousWindowsCommandsDetected(t *testing.T) {
	cases := map[string]string{
		"del /s /q C:\\*":                       "del /s /q",
		"rmdir /s /q C:\\build":                 "rmdir /s /q",
		"rd /s /q C:\\build":                    "rd /s /q",
		"format C:":                             "format",
		"diskpart":                              "diskpart",
		"Remove-Item -Recurse -Force C:\\tmp":   "remove-item -recurse -force",
		"Stop-Computer":                         "stop-computer",
		"Restart-Computer":                      "restart-computer",
	}
	for input, want := range cases {
		got, matched := IsDangerousCommand(input)
		if !got || matched != want {
			t.Errorf("IsDangerousCommand(%q) = (%v, %q), want (true, %q)", input, got, matched, want)
		}
	}
}

func TestSafeCommandsNotDetected(t *testing.T) {
	cases := []string{
		"git status",
		"git diff",
		"go test ./...",
		"go build ./...",
		"dir",
		"Get-ChildItem",
		"ls -la",
		"cat file.txt",
	}
	for _, input := range cases {
		if got, matched := IsDangerousCommand(input); got {
			t.Errorf("IsDangerousCommand(%q) = (true, %q), want false", input, matched)
		}
	}
}

func TestDangerousDetectionCaseInsensitive(t *testing.T) {
	cases := []string{
		"RM -RF /",
		"Git Reset --Hard",
		"DEL /S /Q C:\\*",
		"Remove-Item -recurse -force C:\\tmp",
		"Format C:",
		"DiskPart",
	}
	for _, input := range cases {
		if got, _ := IsDangerousCommand(input); !got {
			t.Errorf("IsDangerousCommand(%q) = false, want true (case-insensitive)", input)
		}
	}
}

func TestDangerousDetectionWhitespace(t *testing.T) {
	cases := []string{
		"  rm -rf /  ",
		"\tgit reset --hard\n",
		"  del /s /q C:\\*  ",
	}
	for _, input := range cases {
		if got, _ := IsDangerousCommand(input); !got {
			t.Errorf("IsDangerousCommand(%q) = false, want true (whitespace-trimmed)", input)
		}
	}
}
