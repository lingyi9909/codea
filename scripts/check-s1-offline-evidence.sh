#!/usr/bin/env bash
set -u

result_dir=${1:-}
script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
pcap_summary="$script_dir/pcap-port-summary.py"
failures=0

fail() {
  echo "[FAIL] $*" >&2
  failures=$((failures + 1))
}

pass() {
  echo "[PASS] $*"
}

if [ -z "$result_dir" ] || [ ! -d "$result_dir" ]; then
  echo "Usage: $0 <s1-result-directory>" >&2
  exit 2
fi

health_file="$result_dir/health.json"
if [ ! -f "$health_file" ]; then
  fail "health response is missing"
elif python3 - "$health_file" <<'PY'
import json
import pathlib
import sys

try:
    payload = json.loads(pathlib.Path(sys.argv[1]).read_text())
except Exception:
    raise SystemExit(1)
raise SystemExit(0 if payload.get("healthy") is True else 1)
PY
then
  pass "health response is healthy"
else
  fail "health response is invalid or unhealthy"
fi

internal_log="$result_dir/opencode-internal.log"
if [ ! -f "$internal_log" ]; then
  fail "OpenCode internal log is missing"
else
  if grep -q 'models\.opencode\.ai' "$internal_log"; then
    fail "models.opencode.ai request found"
  else
    pass "zero models.opencode.ai requests"
  fi

  if grep -q 'ERROR' "$internal_log"; then
    fail "ERROR entry found in OpenCode internal log"
  else
    pass "zero ERROR entries"
  fi
fi

window_file="$result_dir/validation-window.json"
if [ -f "$window_file" ]; then
  window=$(python3 - "$window_file" <<'PY'
import json
import pathlib
import sys

try:
    payload = json.loads(pathlib.Path(sys.argv[1]).read_text())
    start = float(payload["startEpoch"])
    end = float(payload["endEpoch"])
    if start > end:
        raise ValueError("start is after end")
except Exception:
    raise SystemExit(1)
print(start, end)
PY
  ) || window=""
else
  window=$(python3 - "$internal_log" <<'PY'
import datetime
import pathlib
import re
import sys

try:
    first_line = pathlib.Path(sys.argv[1]).read_text().splitlines()[0]
    value = re.search(r"timestamp=([^ ]+)", first_line).group(1)
    start = datetime.datetime.fromisoformat(value.replace("Z", "+00:00")).timestamp()
except Exception:
    raise SystemExit(1)
print(start, 1e20)
PY
  ) || window=""
fi
if [ -z "$window" ]; then
  fail "validation traffic window is missing or invalid"
  window_start=0
  window_end=0
else
  read -r window_start window_end <<<"$window"
  pass "traffic window is $window_start through $window_end"
fi

if [ ! -f "$pcap_summary" ]; then
  fail "pcap analyzer is missing: $pcap_summary"
else
  found_pcap=0
  for pcap in "$result_dir"/traffic-*.pcap; do
    [ -f "$pcap" ] || continue
    found_pcap=1
    iface=$(basename "$pcap" .pcap | sed 's/^traffic-//')
    if ! read -r dns web packet_count < <(python3 "$pcap_summary" "$pcap" "$window_start" "$window_end"); then
      fail "$iface packet capture could not be parsed"
      continue
    fi
    if [ "$dns" -gt 0 ] || [ "$web" -gt 0 ]; then
      fail "$iface contains abnormal traffic (packets=$packet_count DNS=$dns HTTP/HTTPS=$web)"
    else
      pass "$iface contains no DNS/HTTP/HTTPS traffic (packets=$packet_count)"
    fi
  done
  if [ "$found_pcap" -eq 0 ]; then
    fail "packet capture evidence is missing"
  fi
fi

if [ "$failures" -gt 0 ]; then
  echo "S1 evidence validation failed with $failures issue(s)." >&2
  exit 1
fi

echo "S1 evidence validation passed."
