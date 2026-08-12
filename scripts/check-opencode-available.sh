#!/usr/bin/env bash
set -euo pipefail

# Check whether OpenCode is reachable for contract/smoke tests.
# Exit 0 if available (tests can run).
# Exit 2 if unavailable (verification should be unable_to_run, not pass).

OPENCODE_URL="${1:-http://127.0.0.1:14242}"

if curl -sf -o /dev/null "$OPENCODE_URL/global/health" 2>/dev/null; then
  echo "OpenCode available at $OPENCODE_URL"
  exit 0
else
  echo "UNAVAILABLE: OpenCode not reachable at $OPENCODE_URL — contract smoke test will be skipped"
  exit 2
fi
