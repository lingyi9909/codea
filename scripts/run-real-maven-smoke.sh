#!/usr/bin/env bash
set -euo pipefail

# run-real-maven-smoke.sh
#
# Real Maven integration smoke: compiles and executes the fixture JUnit with a
# real Maven install (NOT the mvnw stub), proving the fixture is a genuine
# compilable Maven project. Network may be required on the first run to populate
# the local ~/.m2 cache; the runtime distribution itself remains offline.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
FIXTURE="$REPO_ROOT/tui/tests/e2e/fixtures/java-maven-project"

if ! command -v mvn >/dev/null 2>&1; then
  echo "SKIP: mvn not installed — cannot run real Maven integration smoke" >&2
  exit 0
fi

run_root=$(mktemp -d "${TMPDIR:-/tmp}/codea-real-maven.XXXXXX")
trap 'rm -rf "$run_root"' EXIT

cp -R "$FIXTURE/." "$run_root/"
rm -f "$run_root/mvnw"   # drop the stub so real Maven is exercised

set +e
(
  cd "$run_root"
  mvn -B test
) >"$run_root/maven.log" 2>&1
mvn_exit=$?
set -e

if [ "$mvn_exit" -ne 0 ]; then
  echo "FAIL: real Maven build exited $mvn_exit" >&2
  tail -60 "$run_root/maven.log" >&2
  exit 1
fi

if ! grep -q "BUILD SUCCESS" "$run_root/maven.log"; then
  echo "FAIL: real Maven build did not report BUILD SUCCESS" >&2
  tail -60 "$run_root/maven.log" >&2
  exit 1
fi

if ! grep -qE "Tests run: [1-9][0-9]*, Failures: 0, Errors: 0, Skipped: 0" "$run_root/maven.log"; then
  echo "FAIL: surefire did not report a green test run" >&2
  tail -60 "$run_root/maven.log" >&2
  exit 1
fi

echo "[PASS] real Maven integration smoke: fixture JUnit compiled and ran green"
