#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

require_literal() {
  local file="$1"
  local text="$2"
  if ! grep -Fq "$text" "$file"; then
    echo "missing required Task 29 contract in $file: $text" >&2
    exit 1
  fi
}

for file in \
  distribution/agents/debug/agent.md \
  distribution/agents/unit-test-generator/agent.md \
  distribution/agents/api-documentation/agent.md \
  distribution/skills/debug/SKILL.md \
  distribution/skills/unit-test/SKILL.md \
  distribution/skills/api-documentation/SKILL.md
do
  require_literal "$file" 'task_plan'
  require_literal "$file" 'task_step'
  require_literal "$file" '3–7'
  require_literal "$file" 'before the first mutation or command execution'
  require_literal "$file" 'in_progress'
  require_literal "$file" 'completed'
  require_literal "$file" 'evidence'
  require_literal "$file" 'blocked'
  require_literal "$file" 'Never fabricate'
done

if grep -Fq 'task_plan' distribution/agents/code-reviewer/agent.md; then
  echo 'Code Reviewer must remain plan-free/read-only' >&2
  exit 1
fi
require_literal distribution/agents/code-reviewer/agent.md 'Never modify code.'
if grep -Fq 'task_plan' distribution/skills/code-review/SKILL.md; then
  echo 'Code Review skill must remain plan-free/read-only' >&2
  exit 1
fi
require_literal distribution/skills/code-review/SKILL.md 'read-only analysis'

if [[ -e distribution/agents/planner || -e distribution/agents/planner.md ]]; then
  echo 'Task 29 must not introduce a Planner Agent' >&2
  exit 1
fi

printf '%s\n' 'MUTATING_AGENT_PLAN_CONTRACT PASS'
printf '%s\n' 'CODE_REVIEWER_PLAN_FREE PASS'
printf '%s\n' 'NO_PLANNER_AGENT PASS'
