#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
fixture_dir=$(mktemp -d "${TMPDIR:-/tmp}/codea-phase0-gates.XXXXXX")
trap 'find "$fixture_dir" -type f -delete 2>/dev/null || true; rmdir "$fixture_dir" 2>/dev/null || true' EXIT

pass_file="$fixture_dir/pass.json"
printf '%s\n' '{"S1":"pass","S2":"pass","S3":"pass","S4":"pass","S5":"pass","S6":"pass"}' >"$pass_file"
RESULTS_FILE="$pass_file" "$repo_root/scripts/run-phase0-gates.sh" >"$fixture_dir/pass.log"
grep -q 'All Phase 0 gates PASSED.' "$fixture_dir/pass.log"

fail_file="$fixture_dir/fail.json"
printf '%s\n' '{"S1":"pass","S2":"pass","S3":"pass","S4":"pass","S5":"pass"}' >"$fail_file"
if RESULTS_FILE="$fail_file" "$repo_root/scripts/run-phase0-gates.sh" >"$fixture_dir/fail.log" 2>&1; then
  echo "expected missing S6 to fail" >&2
  exit 1
fi
grep -q 'S6 ... FAIL' "$fixture_dir/fail.log"

echo "Phase 0 gate script tests passed"
