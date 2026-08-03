#!/usr/bin/env bash
set -euo pipefail

RESULTS_FILE=${RESULTS_FILE:-docs/spike-results.json}
FAILED=0

echo "=== Phase 0 Spike Gates ==="

if [ ! -f "$RESULTS_FILE" ]; then
    echo "FAIL: $RESULTS_FILE not found."
    echo "Run all S1-S6 spikes and record results in docs/spike-results.json"
    exit 1
fi

check_gate() {
    local gate=$1
    local result
    result=$(jq -r ".${gate} // \"missing\"" "$RESULTS_FILE")
    case "$result" in
        pass)
            echo "  $gate ... PASS"
            ;;
        *)
            echo "  $gate ... FAIL (result: $result)"
            FAILED=1
            ;;
    esac
}

for gate in S1 S2 S3 S4 S5 S6; do
    check_gate "$gate"
done

if [ "$FAILED" -ne 0 ]; then
    echo
    echo "Phase 0 gates FAILED. Fix issues before proceeding to Phase 1."
    exit 1
fi

echo
echo "All Phase 0 gates PASSED."
