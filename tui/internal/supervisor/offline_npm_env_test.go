package supervisor

import "testing"

func TestBuildEnvBoundsNPMNetworkRetries(t *testing.T) {
	env := buildEnv(Config{ConfigDir: `C:\codea\runtime-config`}, "u", "p")
	for _, want := range []string{
		"npm_config_fetch_retries=0",
		"npm_config_fetch_timeout=2000",
		"npm_config_fetch_retry_mintimeout=250",
		"npm_config_fetch_retry_maxtimeout=500",
	} {
		if !hasEnv(env, want) {
			t.Fatalf("runtime env missing %s", want)
		}
	}
}
