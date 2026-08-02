#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CHECKER="$ROOT_DIR/scripts/check-execution-state.sh"
STATE="$ROOT_DIR/docs/execution-state.yaml"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

expect_fail() {
  local name="$1"
  local file="$2"
  if "$CHECKER" "$file" >/dev/null 2>&1; then
    echo "FAIL: $name unexpectedly passed"
    exit 1
  fi
}

"$CHECKER" "$STATE"

python3 - "$STATE" "$TMP_DIR" <<'PY'
import copy
import pathlib
import sys
import yaml

source = pathlib.Path(sys.argv[1])
target = pathlib.Path(sys.argv[2])
data = yaml.safe_load(source.read_text())

duplicate = copy.deepcopy(data)
duplicate["tasks"]["0"]["status"] = "in_progress"
duplicate["tasks"]["1"]["status"] = "blocked"
(target / "duplicate-active.yaml").write_text(yaml.safe_dump(duplicate, sort_keys=False))

unaccepted = copy.deepcopy(data)
unaccepted["current"]["status"] = "completed"
unaccepted["tasks"]["0"].update({
    "status": "completed",
    "verificationStatus": "pass",
    "taskGateStatus": "pass",
    "humanAccepted": False,
})
(target / "unaccepted-completed.yaml").write_text(yaml.safe_dump(unaccepted, sort_keys=False))

mismatch = copy.deepcopy(data)
mismatch["current"]["status"] = "in_progress"
(target / "current-mismatch.yaml").write_text(yaml.safe_dump(mismatch, sort_keys=False))

acceptance_mismatch = copy.deepcopy(data)
acceptance_mismatch["humanAcceptance"]["accepted"] = True
(target / "acceptance-mismatch.yaml").write_text(yaml.safe_dump(acceptance_mismatch, sort_keys=False))

missing_report = copy.deepcopy(data)
missing_report["current"]["status"] = "completed"
missing_report["verification"]["status"] = "pass"
missing_report["taskGate"]["status"] = "pass"
missing_report["humanAcceptance"]["accepted"] = True
missing_report["tasks"]["0"].update({
    "status": "completed",
    "verificationStatus": "pass",
    "taskGateStatus": "pass",
    "humanAccepted": True,
})
(target / "missing-report.yaml").write_text(yaml.safe_dump(missing_report, sort_keys=False))
PY

expect_fail "duplicate active tasks" "$TMP_DIR/duplicate-active.yaml"
expect_fail "completed without acceptance" "$TMP_DIR/unaccepted-completed.yaml"
expect_fail "current status mismatch" "$TMP_DIR/current-mismatch.yaml"
expect_fail "current acceptance mismatch" "$TMP_DIR/acceptance-mismatch.yaml"
expect_fail "completed without report" "$TMP_DIR/missing-report.yaml"

echo "Execution state validator tests passed."
