#!/usr/bin/env bash
set -euo pipefail
repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "$repo_root/tui"
GOTOOLCHAIN=local go test ./internal/update -run '^TestUpgradeCommitsVersionAndMigratedConfig$' -count=1 -v
