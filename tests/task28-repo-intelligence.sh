#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
TUI="$ROOT/tui"

fail() {
  echo "TASK28_REPO_INTELLIGENCE FAIL: $*" >&2
  exit 1
}

# Architecture boundary: Repo Intelligence and Application remain Codea-owned.
if grep -R -nE '"codea/tui/internal/opencode(/generated|/client)?"|"codea/tui/internal/opencode/generated|"codea/tui/internal/opencode/client' \
  "$TUI/internal/repoctx" "$TUI/internal/app" --include='*.go'; then
  fail "repoctx/application imports vendor OpenCode implementation packages"
fi
if grep -R -nE '"net/http"|"net/url"|"net/rpc"' "$TUI/internal/repoctx" --include='*.go'; then
  fail "repoctx imports network libraries"
fi
if grep -R -nE 'import[[:space:]]+"C"' "$TUI/internal/repoctx" --include='*.go'; then
  fail "repoctx requires CGO"
fi
if grep -Eiq 'tree[-_]?sitter|go-tree-sitter' "$TUI/go.mod"; then
  fail "tree-sitter/parser dependency found in tui/go.mod"
fi

(
  cd "$TUI"
  GOTOOLCHAIN=local go test ./tests/architecture -count=1

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
