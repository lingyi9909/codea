#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

require_literal() {
  local file="$1"
  local text="$2"
  if ! grep -Fq "$text" "$file"; then
    echo "missing required Task 30 verification contract in $file: $text" >&2
    exit 1
  fi
}

require_ordered() {
  local file="$1"
  shift
  local previous=0
  local text line
  for text in "$@"; do
    line="$(grep -nF -m1 "$text" "$file" | cut -d: -f1 || true)"
    if [[ -z "$line" ]]; then
      echo "missing ordered Task 30 contract in $file: $text" >&2
      exit 1
    fi
    if (( line <= previous )); then
      echo "Task 30 contract out of order in $file: $text" >&2
      exit 1
    fi
    previous="$line"
  done
}

for file in \
  distribution/agents/debug/agent.md \
  distribution/skills/debug/SKILL.md \
  distribution/agents/unit-test-generator/agent.md \
  distribution/skills/unit-test/SKILL.md
do
  require_literal "$file" '`verify_project`'
  require_literal "$file" 'machine-observable verification evidence'
  require_literal "$file" '`NOT_CONFIGURED`'
  require_literal "$file" '`TIMEOUT`'
  require_literal "$file" 'unverified'
  require_literal "$file" 'one bounded repair'
  require_literal "$file" '`verification-control`'
  require_literal "$file" 'same root task'
  require_literal "$file" 'not a new user task'
  require_literal "$file" 'must not reset the plan epoch'
done

require_ordered distribution/agents/debug/agent.md \
  '**Collect evidence**' \
  '**Reproduce**' \
  '**Plan or refresh**' \
  '**Root cause**' \
  '**Controlled fix**' \
  '**Machine verification**' \
  '**Bounded repair**' \
  '**Report from machine evidence**'

require_ordered distribution/skills/debug/SKILL.md \
  '**Collect evidence**' \
  '**Reproduce**' \
  '**Plan or refresh**' \
  '**Root cause**' \
  '**Controlled fix**' \
  '**Machine verification**' \
  '**Bounded repair**' \
  '**Report from machine evidence**'

require_ordered distribution/agents/unit-test-generator/agent.md \
  '**Analyze target and risk**' \
  '**Plan or refresh**' \
  '**Protected test write**' \
  '**Machine verification**' \
  '**Report from machine evidence**'

require_ordered distribution/skills/unit-test/SKILL.md \
  '**Analyze target and risk**' \
  '**Plan or refresh**' \
  '**Protected test write**' \
  '**Machine verification**' \
  '**Report from machine evidence**'

if grep -Fq 'verify_project' distribution/agents/code-reviewer/agent.md; then
  echo 'Code Reviewer is read-only and must not be forced through verify_project' >&2
  exit 1
fi
if grep -Fq 'verify_project' distribution/skills/code-review/SKILL.md; then
  echo 'Code Review skill is read-only and must not be forced through verify_project' >&2
  exit 1
fi
require_literal distribution/agents/code-reviewer/agent.md 'Never modify code.'
require_literal distribution/skills/code-review/SKILL.md 'read-only analysis'

printf '%s\n' 'MUTATING_AGENT_VERIFY_PROJECT_CONTRACT PASS'
printf '%s\n' 'BOUNDED_REPAIR_CONTRACT PASS'
printf '%s\n' 'CONTROL_CONTINUATION_AGENT_CONTRACT PASS'
printf '%s\n' 'CODE_REVIEWER_REMAINS_READ_ONLY PASS'
