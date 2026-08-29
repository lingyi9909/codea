#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
TUI="$ROOT/tui"

fail() {
  echo "TASK28_REPO_INTELLIGENCE FAIL: $*" >&2
  exit 1
}

# Architecture boundary applies to shipped Application/Repo Intelligence code.
# Integration tests may intentionally instantiate OpenCode adapters to exercise
# the boundary end-to-end, so *_test.go is excluded from this source-import gate.
if grep -R -nE --include='*.go' --exclude='*_test.go' \
  '"codea/tui/internal/opencode(/generated|/client)?"|"codea/tui/internal/opencode/generated|"codea/tui/internal/opencode/client' \
  "$TUI/internal/repoctx" "$TUI/internal/app"; then
  fail "repoctx/application production code imports vendor OpenCode implementation packages"
fi
if grep -R -nE --include='*.go' --exclude='*_test.go' \
  '"net/http"|"net/url"|"net/rpc"' "$TUI/internal/repoctx"; then
  fail "repoctx production code imports network libraries"
fi
if grep -R -nE --include='*.go' --exclude='*_test.go' \
  'import[[:space:]]+"C"' "$TUI/internal/repoctx"; then
  fail "repoctx production code requires CGO"
fi
if grep -Eiq 'tree[-_]?sitter|go-tree-sitter' "$TUI/go.mod"; then
  fail "tree-sitter/parser dependency found in tui/go.mod"
fi

(
  cd "$TUI"
  GOTOOLCHAIN=local go test ./tests/architecture -count=1

  # Reviewer-requested Task 28 remediation regressions are part of the formal
  # mechanical Gate, not local-only tests.
  GOTOOLCHAIN=local go test ./internal/repoctx \
    -run '^TestTask28Remediation' -count=1
  echo 'TASK28_REMEDIATION_REGRESSIONS PASS'

  # Real Java business-chain fixture must be confirmed from typed evidence.
  GOTOOLCHAIN=local go test ./internal/repoctx \
    -run '^TestIndexGraphConfirmsJavaBusinessChainAndRejectsPackageSimilarity$' -count=1
  echo 'JAVA_CHAIN OrderController#createOrder -> OrderService#createOrder -> OrderRepository#save'

  # Ambiguous overload/call targets must remain unresolved.
  GOTOOLCHAIN=local go test ./internal/repoctx \
    -run '^TestIndexAmbiguousMethodTargetRemainsUnresolved$' -count=1
  echo 'AMBIGUOUS_RELATION_NOT_PROMOTED PASS'

  # Deterministic ranking and bounded rendering are mechanical contracts.
  GOTOOLCHAIN=local go test ./internal/repoctx \
    -run '^(TestRankPriorityAndDeterminism|TestRepoMapBudgetAndDeterminism)$' -count=1
  echo 'REPO_MAP_DETERMINISTIC PASS'
  echo 'REPO_MAP_BUDGET PASS'

  # Full Task 28 focused behavior: walker, Java/Spring, Go, fallback, ranking,
  # map injection, professional commands, async submit and composition root.
  GOTOOLCHAIN=local go test ./internal/repoctx ./internal/app ./cmd/codea \
    -run 'Repo|Task28|Professional|Submit' -count=1

  # V1.2 must remain pure-Go and cross-build for the mandatory Windows target.
  CGO_ENABLED=0 GOOS=windows GOARCH=amd64 GOTOOLCHAIN=local go build ./cmd/codea
  echo 'WINDOWS_CGO0_BUILD PASS'
)

echo 'TASK28_REPO_INTELLIGENCE PASS'
