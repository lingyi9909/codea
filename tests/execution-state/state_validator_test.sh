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
  local expected="$3"
  local output
  if output="$("$CHECKER" "$file" 2>&1)"; then
    echo "FAIL: $name unexpectedly passed"
    exit 1
  fi
  if [[ "$output" != *"$expected"* ]]; then
    echo "FAIL: $name returned the wrong error"
    echo "Expected: $expected"
    echo "Actual: $output"
    exit 1
  fi
}

expect_pass() {
  local name="$1"
  local file="$2"
  local expected="$3"
  local output
  if ! output="$("$CHECKER" "$file" 2>&1)"; then
    echo "FAIL: $name was rejected"
    echo "Actual: $output"
    exit 1
  fi
  if [[ "$output" != *"$expected"* ]]; then
    echo "FAIL: $name returned the wrong output"
    echo "Expected: $expected"
    echo "Actual: $output"
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
duplicate["current"]["status"] = "in_progress"
duplicate["tasks"]["0"]["status"] = "in_progress"
duplicate["tasks"]["1"]["status"] = "blocked"
(target / "duplicate-active.yaml").write_text(yaml.safe_dump(duplicate, sort_keys=False))

unaccepted = copy.deepcopy(data)
unaccepted["current"]["status"] = "completed"
unaccepted["verification"]["status"] = "pass"
unaccepted["taskGate"]["status"] = "pass"
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

skipped = copy.deepcopy(data)
skipped["current"].update({"task": 1, "step": 1, "status": "in_progress"})
skipped["tasks"]["1"]["status"] = "in_progress"
(target / "skipped-task.yaml").write_text(yaml.safe_dump(skipped, sort_keys=False))

awaiting_without_gates = copy.deepcopy(data)
awaiting_without_gates["current"]["status"] = "awaiting_acceptance"
awaiting_without_gates["tasks"]["0"]["status"] = "awaiting_acceptance"
(target / "awaiting-without-gates.yaml").write_text(yaml.safe_dump(awaiting_without_gates, sort_keys=False))

verification_unable = copy.deepcopy(data)
verification_unable["verification"]["status"] = "unable_to_run"
verification_unable["tasks"]["0"]["verificationStatus"] = "unable_to_run"
(target / "verification-unable-pending.yaml").write_text(yaml.safe_dump(verification_unable, sort_keys=False))

verification_failed = copy.deepcopy(data)
verification_failed["verification"]["status"] = "fail"
verification_failed["tasks"]["0"]["verificationStatus"] = "fail"
(target / "verification-failed-pending.yaml").write_text(yaml.safe_dump(verification_failed, sort_keys=False))

gate_unable = copy.deepcopy(data)
gate_unable["taskGate"]["status"] = "unable_to_evaluate"
gate_unable["tasks"]["0"]["taskGateStatus"] = "unable_to_evaluate"
(target / "gate-unable-pending.yaml").write_text(yaml.safe_dump(gate_unable, sort_keys=False))

missing_step = copy.deepcopy(data)
missing_step["current"].pop("step")
(target / "missing-step.yaml").write_text(yaml.safe_dump(missing_step, sort_keys=False))

accepted_pending = copy.deepcopy(data)
accepted_pending["humanAcceptance"]["accepted"] = True
accepted_pending["tasks"]["0"]["humanAccepted"] = True
(target / "accepted-pending.yaml").write_text(yaml.safe_dump(accepted_pending, sort_keys=False))

future_active = copy.deepcopy(data)
future_active["tasks"]["1"]["status"] = "in_progress"
(target / "future-active.yaml").write_text(yaml.safe_dump(future_active, sort_keys=False))

terminal_report = target / "terminal-report.md"
terminal_report.write_text("# Terminal Task Report\n")
terminal = copy.deepcopy(data)
for task in terminal["tasks"].values():
    task.update({
        "status": "completed",
        "verificationStatus": "pass",
        "taskGateStatus": "pass",
        "humanAccepted": True,
        "report": str(terminal_report),
    })
terminal["current"].update({"task": 21, "step": 1, "status": "completed"})
terminal["verification"]["status"] = "pass"
terminal["taskGate"]["status"] = "pass"
terminal["humanAcceptance"]["accepted"] = True
(target / "terminal.yaml").write_text(yaml.safe_dump(terminal, sort_keys=False))
PY

expect_fail "duplicate active tasks" "$TMP_DIR/duplicate-active.yaml" "FAIL: more than one Task is active"
expect_fail "completed without acceptance" "$TMP_DIR/unaccepted-completed.yaml" "FAIL: completed Task 0 requires human acceptance"
expect_fail "current status mismatch" "$TMP_DIR/current-mismatch.yaml" "FAIL: current.status does not match current Task"
expect_fail "current acceptance mismatch" "$TMP_DIR/acceptance-mismatch.yaml" "FAIL: current acceptance does not match current Task"
expect_fail "completed without report" "$TMP_DIR/missing-report.yaml" "FAIL: completed Task 0 report is missing"
expect_fail "skipped current task" "$TMP_DIR/skipped-task.yaml" "FAIL: current.task must be the first incomplete Task"
expect_fail "awaiting acceptance without gates" "$TMP_DIR/awaiting-without-gates.yaml" "FAIL: awaiting_acceptance Task 0 must pass verification and Task Gate"
expect_fail "verification unable while pending" "$TMP_DIR/verification-unable-pending.yaml" "FAIL: unable_to_run verification requires Task 0 to be blocked"
expect_fail "verification failed while pending" "$TMP_DIR/verification-failed-pending.yaml" "FAIL: failed verification requires Task 0 to be blocked"
expect_fail "task gate unable while pending" "$TMP_DIR/gate-unable-pending.yaml" "FAIL: unable_to_evaluate Task Gate requires Task 0 to be blocked"
expect_fail "missing current step" "$TMP_DIR/missing-step.yaml" "FAIL: current.step must be a positive integer"
expect_fail "pending task already accepted" "$TMP_DIR/accepted-pending.yaml" "FAIL: human acceptance is only valid for completed Task 0"
expect_pass "all tasks completed terminal state" "$TMP_DIR/terminal.yaml" "Execution state is valid: Task 21 Step 1 (completed)"
expect_fail "future task active while current is pending" "$TMP_DIR/future-active.yaml" "FAIL: pending current Task requires no active Task"

echo "Execution state validator tests passed."
