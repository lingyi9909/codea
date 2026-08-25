package opencode

import "testing"

func TestPluginFileURLWindowsDrive(t *testing.T) {
	got := pluginFileURL(`C:\Users\alice\.codea\current\plugins\index.js`)
	want := "file:///C:/Users/alice/.codea/current/plugins/index.js"
	if got != want {
		t.Fatalf("pluginFileURL() = %q, want %q", got, want)
	}
}

func TestPluginFileURLWindowsDriveEscapesSpaces(t *testing.T) {
	got := pluginFileURL(`C:\Users\Alice Smith\.codea\plugins\index.js`)
	want := "file:///C:/Users/Alice%20Smith/.codea/plugins/index.js"
	if got != want {
		t.Fatalf("pluginFileURL() = %q, want %q", got, want)
	}
}

func TestPluginFileURLPosixPath(t *testing.T) {
	got := pluginFileURL("/opt/codea/plugins/index.js")
	want := "file:///opt/codea/plugins/index.js"
	if got != want {
		t.Fatalf("pluginFileURL() = %q, want %q", got, want)
	}
}
