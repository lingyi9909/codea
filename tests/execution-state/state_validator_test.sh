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

# Schema v2 inserts Task 2A without renumbering Task 3 through Task 21. Build
# this fixture independently so the test proves the validator understands the
# explicit taskOrder before the repository state itself migrates.
python3 - "$STATE" "$TMP_DIR/task-2a-valid.yaml" "$TMP_DIR/task-2a-missing-order.yaml" "$TMP_DIR/task-2a-order-mismatch.yaml" <<'PY'
import copy
import pathlib
import sys
import yaml

source = pathlib.Path(sys.argv[1])
valid_path = pathlib.Path(sys.argv[2])
missing_order_path = pathlib.Path(sys.argv[3])
order_mismatch_path = pathlib.Path(sys.argv[4])
data = yaml.safe_load(source.read_text())

task_order = ["0", "1", "2", "2A"] + [str(i) for i in range(3, 22)]
data["schemaVersion"] = 2
data["taskOrder"] = task_order
data["current"].update({"task": "2A", "step": 1, "status": "pending"})
data["verification"].update({"status": "not_run", "commands": []})
data["taskGate"]["status"] = "not_evaluated"
data["humanAcceptance"]["accepted"] = False
# Reset all tasks to pending so the fixture is independent of current
# repository state. The real state may have later tasks in_progress or
# completed, which would produce invalid fixtures otherwise.
for tid in data["taskOrder"]:
    data["tasks"][tid] = {
        "status": "pending",
        "completedSteps": [],
        "verificationStatus": "not_run",
        "taskGateStatus": "not_evaluated",
        "humanAccepted": False,
        "checkpoint": None,
        "report": f"docs/task-reports/task-{tid.zfill(2)}.md",
    }
# Mark earlier tasks as completed so current.task=2A is valid
for tid in ["0", "1", "2"]:
    data["tasks"][tid].update({
        "status": "completed",
        "verificationStatus": "pass",
        "taskGateStatus": "pass",
        "humanAccepted": True,
    })
valid_path.write_text(yaml.safe_dump(data, sort_keys=False))

missing_order = copy.deepcopy(data)
missing_order.pop("taskOrder")
missing_order_path.write_text(yaml.safe_dump(missing_order, sort_keys=False))

order_mismatch = copy.deepcopy(data)
order_mismatch["taskOrder"].remove("2A")
order_mismatch_path.write_text(yaml.safe_dump(order_mismatch, sort_keys=False))
PY

expect_pass "Task 2A schema v2 state" "$TMP_DIR/task-2a-valid.yaml" "Execution state is valid: Task 2A Step 1 (pending)"
expect_fail "schema v2 requires taskOrder" "$TMP_DIR/task-2a-missing-order.yaml" "FAIL: taskOrder must match the Codea V1 execution order"
expect_fail "taskOrder must cover every task" "$TMP_DIR/task-2a-order-mismatch.yaml" "FAIL: taskOrder must match the Codea V1 execution order"

python3 - "$STATE" "$TMP_DIR" <<'PY'
import copy
import pathlib
import sys
import yaml

source = pathlib.Path(sys.argv[1])
target = pathlib.Path(sys.argv[2])
data = yaml.safe_load(source.read_text())

# Build every negative fixture from the same known-valid state. The repository
# state may legitimately be pending, in progress, blocked, or awaiting review
# while this suite runs, so using it directly would make failure reasons drift.
data["current"].update({"task": 0, "step": 11, "status": "awaiting_acceptance"})
data["verification"]["status"] = "pass"
data["taskGate"]["status"] = "pass"
data["humanAcceptance"]["accepted"] = False
data["checkpoint"] = {"commit": "0000000000000000000000000000000000000000"}
data["tasks"]["0"].update({
    "status": "awaiting_acceptance",
    "verificationStatus": "pass",
    "taskGateStatus": "pass",
    "humanAccepted": False,
    "checkpoint": "0000000000000000000000000000000000000000",
    "report": str(target / "baseline-task-00.md"),
})
baseline_report = target / "baseline-task-00.md"
baseline_report.write_text("**Checkpoint:** `0000000000000000000000000000000000000000`\n")
for task_id in data["taskOrder"][1:]:
    data["tasks"][task_id].update({
        "status": "pending",
        "completedSteps": [],
        "verificationStatus": "not_run",
        "taskGateStatus": "not_evaluated",
        "humanAccepted": False,
        "checkpoint": None,
    })

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
    "report": str(target / "missing-task-00.md"),
})
(target / "missing-report.yaml").write_text(yaml.safe_dump(missing_report, sort_keys=False))

skipped = copy.deepcopy(data)
skipped["current"].update({"task": 1, "step": 1, "status": "in_progress"})
skipped["tasks"]["0"].update({
    "status": "pending",
    "verificationStatus": "not_run",
    "taskGateStatus": "not_evaluated",
})
skipped["tasks"]["1"]["status"] = "in_progress"
(target / "skipped-task.yaml").write_text(yaml.safe_dump(skipped, sort_keys=False))

awaiting_without_gates = copy.deepcopy(data)
awaiting_without_gates["current"]["status"] = "awaiting_acceptance"
awaiting_without_gates["verification"]["status"] = "not_run"
awaiting_without_gates["taskGate"]["status"] = "not_evaluated"
awaiting_without_gates["tasks"]["0"]["status"] = "awaiting_acceptance"
awaiting_without_gates["tasks"]["0"]["verificationStatus"] = "not_run"
awaiting_without_gates["tasks"]["0"]["taskGateStatus"] = "not_evaluated"
(target / "awaiting-without-gates.yaml").write_text(yaml.safe_dump(awaiting_without_gates, sort_keys=False))

verification_unable = copy.deepcopy(data)
verification_unable["current"]["status"] = "pending"
verification_unable["verification"]["status"] = "unable_to_run"
verification_unable["taskGate"]["status"] = "not_evaluated"
verification_unable["tasks"]["0"]["status"] = "pending"
verification_unable["tasks"]["0"]["verificationStatus"] = "unable_to_run"
verification_unable["tasks"]["0"]["taskGateStatus"] = "not_evaluated"
(target / "verification-unable-pending.yaml").write_text(yaml.safe_dump(verification_unable, sort_keys=False))

verification_failed = copy.deepcopy(data)
verification_failed["current"]["status"] = "pending"
verification_failed["verification"]["status"] = "fail"
verification_failed["taskGate"]["status"] = "not_evaluated"
verification_failed["tasks"]["0"]["status"] = "pending"
verification_failed["tasks"]["0"]["verificationStatus"] = "fail"
verification_failed["tasks"]["0"]["taskGateStatus"] = "not_evaluated"
(target / "verification-failed-pending.yaml").write_text(yaml.safe_dump(verification_failed, sort_keys=False))

gate_unable = copy.deepcopy(data)
gate_unable["current"]["status"] = "pending"
gate_unable["verification"]["status"] = "not_run"
gate_unable["taskGate"]["status"] = "unable_to_evaluate"
gate_unable["tasks"]["0"]["status"] = "pending"
gate_unable["tasks"]["0"]["verificationStatus"] = "not_run"
gate_unable["tasks"]["0"]["taskGateStatus"] = "unable_to_evaluate"
(target / "gate-unable-pending.yaml").write_text(yaml.safe_dump(gate_unable, sort_keys=False))

missing_step = copy.deepcopy(data)
missing_step["current"].pop("step")
(target / "missing-step.yaml").write_text(yaml.safe_dump(missing_step, sort_keys=False))

accepted_pending = copy.deepcopy(data)
accepted_pending["current"]["status"] = "pending"
accepted_pending["verification"]["status"] = "not_run"
accepted_pending["taskGate"]["status"] = "not_evaluated"
accepted_pending["humanAcceptance"]["accepted"] = True
accepted_pending["tasks"]["0"].update({
    "status": "pending",
    "verificationStatus": "not_run",
    "taskGateStatus": "not_evaluated",
})
accepted_pending["tasks"]["0"]["humanAccepted"] = True
(target / "accepted-pending.yaml").write_text(yaml.safe_dump(accepted_pending, sort_keys=False))

future_active = copy.deepcopy(data)
future_active["current"]["status"] = "pending"
future_active["verification"]["status"] = "not_run"
future_active["taskGate"]["status"] = "not_evaluated"
future_active["tasks"]["0"].update({
    "status": "pending",
    "verificationStatus": "not_run",
    "taskGateStatus": "not_evaluated",
})
future_active["tasks"]["1"]["status"] = "in_progress"
(target / "future-active.yaml").write_text(yaml.safe_dump(future_active, sort_keys=False))

# Checkpoint mismatch fixtures
chk_mismatch = copy.deepcopy(data)
chk_mismatch["checkpoint"] = {"commit": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
chk_mismatch["tasks"]["0"]["checkpoint"] = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
(target / "checkpoint-mismatch.yaml").write_text(yaml.safe_dump(chk_mismatch, sort_keys=False))

report_no_chk = copy.deepcopy(data)
report_no_chk["checkpoint"] = {"commit": "cccccccccccccccccccccccccccccccccccccccc"}
report_no_chk["tasks"]["0"]["checkpoint"] = "cccccccccccccccccccccccccccccccccccccccc"
report = target / "report-no-chk.md"
report.write_text("# Task Report\n\n**Checkpoint:** `dddddddddddddddddddddddddddddddddddddddd`\n")
report_no_chk["tasks"]["0"]["report"] = str(report)
(target / "report-checkpoint-mismatch.yaml").write_text(yaml.safe_dump(report_no_chk, sort_keys=False))

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
expect_fail "checkpoint mismatch global vs task" "$TMP_DIR/checkpoint-mismatch.yaml" "FAIL: global checkpoint does not match Task 0 checkpoint"
expect_fail "report checkpoint mismatch" "$TMP_DIR/report-checkpoint-mismatch.yaml" "FAIL: Task 0 report checkpoint does not match task checkpoint"

echo "Execution state validator tests passed."
