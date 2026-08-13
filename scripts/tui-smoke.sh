#!/usr/bin/env bash
#
# tui-smoke.sh — real TUI smoke: launches the actual `codea` binary under a PTY
# with a deterministic fake OpenCode runtime and drives the full flow
# (start -> healthy -> prompt -> reasoning/answer streaming -> tool -> resize ->
# ctrl+t -> ctrl+c -> runtime stop). Writes a readable transcript to
# docs/task-reports/tui-smoke-transcript.txt.
#
# Opt-in: the underlying test skips unless CODEA_TUI_SMOKE=1.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
transcript="$repo_root/docs/task-reports/tui-smoke-transcript"

cd "$repo_root/tui"

export CODEA_TUI_SMOKE=1
export CODEA_TUI_SMOKE_TRANSCRIPT="$transcript"

GOTOOLCHAIN=local go test ./tests/tui-smoke/ -run TestRealTUISmoke -count=1 -v

# The raw (ANSI) transcript is a debugging artifact; keep only the readable
# .txt in the repo.
rm -f "$transcript"

echo
echo "Readable transcript: ${transcript}.txt"
