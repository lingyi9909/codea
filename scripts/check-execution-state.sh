#!/usr/bin/env bash
set -euo pipefail

STATE_FILE="${1:-docs/execution-state.yaml}"

if ! command -v python3 >/dev/null 2>&1; then
  echo "BLOCKED: python3 is required to validate execution state" >&2
  exit 2
fi

if ! python3 -c 'import yaml' >/dev/null 2>&1; then
  echo "BLOCKED: PyYAML is required to validate execution state" >&2
  exit 2
fi

python3 - "$STATE_FILE" <<'PY'
import pathlib
import sys
import yaml

path = pathlib.Path(sys.argv[1])
if not path.is_file():
    raise SystemExit(f"FAIL: state file not found: {path}")

try:
    state = yaml.safe_load(path.read_text())
except Exception as exc:
    raise SystemExit(f"FAIL: invalid YAML: {exc}")

task_states = {"pending", "in_progress", "blocked", "awaiting_acceptance", "completed"}
verification_states = {"not_run", "pass", "fail", "unable_to_run"}
task_gate_states = {"not_evaluated", "pass", "fail", "unable_to_evaluate"}

if state.get("schemaVersion") != 1:
    raise SystemExit("FAIL: schemaVersion must be 1")

tasks = state.get("tasks")
if not isinstance(tasks, dict) or set(tasks) != {str(i) for i in range(22)}:
    raise SystemExit("FAIL: tasks must contain exactly Task 0 through Task 21")

for task_id, task in tasks.items():
    if task.get("status") not in task_states:
        raise SystemExit(f"FAIL: Task {task_id} has invalid status")
    if task.get("verificationStatus") not in verification_states:
        raise SystemExit(f"FAIL: Task {task_id} has invalid verificationStatus")
    if task.get("taskGateStatus") not in task_gate_states:
        raise SystemExit(f"FAIL: Task {task_id} has invalid taskGateStatus")
    if task.get("status") == "completed":
        if task.get("verificationStatus") != "pass" or task.get("taskGateStatus") != "pass":
            raise SystemExit(f"FAIL: completed Task {task_id} must pass verification and Task Gate")
        if task.get("humanAccepted") is not True:
            raise SystemExit(f"FAIL: completed Task {task_id} requires human acceptance")
        if not pathlib.Path(task.get("report", "")).is_file():
            raise SystemExit(f"FAIL: completed Task {task_id} report is missing")

active = [task_id for task_id, task in tasks.items() if task["status"] in {"in_progress", "blocked", "awaiting_acceptance"}]
if len(active) > 1:
    raise SystemExit("FAIL: more than one Task is active")

current = state.get("current", {})
current_id = str(current.get("task"))
if current_id not in tasks:
    raise SystemExit("FAIL: current.task is invalid")
if current.get("status") != tasks[current_id]["status"]:
    raise SystemExit("FAIL: current.status does not match current Task")
if current.get("status") != "pending" and active != [current_id]:
    raise SystemExit("FAIL: current.task must be the unique active Task")

verification = state.get("verification", {})
task_gate = state.get("taskGate", {})
human_acceptance = state.get("humanAcceptance", {})
if verification.get("status") != tasks[current_id]["verificationStatus"]:
    raise SystemExit("FAIL: current verification does not match current Task")
if task_gate.get("status") != tasks[current_id]["taskGateStatus"]:
    raise SystemExit("FAIL: current task gate does not match current Task")
if human_acceptance.get("accepted") != tasks[current_id]["humanAccepted"]:
    raise SystemExit("FAIL: current acceptance does not match current Task")

seen_incomplete = False
for task_id in map(str, range(22)):
    status = tasks[task_id]["status"]
    if status != "completed":
        seen_incomplete = True
    elif seen_incomplete:
        raise SystemExit(f"FAIL: Task {task_id} completed before an earlier Task")

print(f"Execution state is valid: Task {current_id} Step {current.get('step')} ({current.get('status')})")
PY
