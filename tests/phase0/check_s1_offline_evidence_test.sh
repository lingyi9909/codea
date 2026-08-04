#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
fixture_root=$(mktemp -d "${TMPDIR:-/tmp}/codea-s1-validator.XXXXXX")
trap 'find "$fixture_root" -type f -delete 2>/dev/null || true; find "$fixture_root" -depth -type d -exec rmdir {} \; 2>/dev/null || true' EXIT

write_pcap() {
  local path=$1
  local protocol=${2:-none}
  local timestamp=${3:-15}
  python3 - "$path" "$protocol" "$timestamp" <<'PY'
import pathlib
import socket
import struct
import sys

path = pathlib.Path(sys.argv[1])
protocol = sys.argv[2]
timestamp = int(sys.argv[3])
global_header = struct.pack("<IHHIIII", 0xA1B2C3D4, 2, 4, 0, 0, 65535, 1)
if protocol == "none":
    path.write_bytes(global_header)
    raise SystemExit

proto_number = 17 if protocol == "dns" else 6
destination_port = 53 if protocol == "dns" else 443
transport = (
    struct.pack("!HHHH", 12345, destination_port, 8, 0)
    if proto_number == 17
    else struct.pack("!HHIIHHHH", 12345, destination_port, 0, 0, 0x5002, 65535, 0, 0)
)
ip = struct.pack(
    "!BBHHHBBH4s4s",
    0x45, 0, 20 + len(transport), 1, 0, 64, proto_number, 0,
    socket.inet_aton("10.0.0.2"), socket.inet_aton("8.8.8.8"),
)
ethernet = bytes.fromhex("00112233445566778899aabb0800")
packet = ethernet + ip + transport
record = struct.pack("<IIII", timestamp, 0, len(packet), len(packet)) + packet
path.write_bytes(global_header + record)
PY
}

new_fixture() {
  local name=$1
  local dir="$fixture_root/$name"
  mkdir -p "$dir"
  printf '%s\n' '{"healthy":true,"version":"1.18.11"}' >"$dir/health.json"
  printf '%s\n' 'timestamp=x level=INFO message=ready' >"$dir/opencode-internal.log"
  printf '%s\n' '{"startEpoch":10,"endEpoch":20}' >"$dir/validation-window.json"
  printf '%s\n' en0 >"$dir/capture-interfaces.txt"
  write_pcap "$dir/traffic-en0.pcap"
  printf '%s\n' "$dir"
}

expect_pass() {
  local dir=$1
  "$repo_root/scripts/check-s1-offline-evidence.sh" "$dir" >/dev/null
}

expect_fail() {
  local label=$1
  local dir=$2
  if "$repo_root/scripts/check-s1-offline-evidence.sh" "$dir" >"$dir/result.log" 2>&1; then
    echo "expected failure: $label" >&2
    exit 1
  fi
  grep -q '\[FAIL\]' "$dir/result.log"
}

clean=$(new_fixture clean)
expect_pass "$clean"

forbidden=$(new_fixture forbidden-host)
printf '%s\n' 'timestamp=x level=INFO url=https://models.opencode.ai' >"$forbidden/opencode-internal.log"
expect_fail 'forbidden public host' "$forbidden"

forbidden_case=$(new_fixture forbidden-host-case)
printf '%s\n' 'timestamp=x level=INFO url=https://MODELS.OPENCODE.AI' >"$forbidden_case/opencode-internal.log"
expect_fail 'case-insensitive forbidden public host' "$forbidden_case"

error_log=$(new_fixture error-log)
printf '%s\n' 'timestamp=x level=ERROR message=boom' >"$error_log/opencode-internal.log"
expect_fail 'runtime ERROR' "$error_log"

dns=$(new_fixture dns-traffic)
write_pcap "$dns/traffic-en0.pcap" dns
expect_fail 'DNS traffic' "$dns"

web=$(new_fixture web-traffic)
write_pcap "$web/traffic-en0.pcap" web
expect_fail 'HTTP or HTTPS traffic' "$web"

before_window=$(new_fixture before-window)
write_pcap "$before_window/traffic-en0.pcap" dns 5
expect_pass "$before_window"

missing=$(new_fixture missing-pcap)
rm "$missing/traffic-en0.pcap"
expect_fail 'missing pcap evidence' "$missing"

missing_interface=$(new_fixture missing-interface)
printf '%s\n' en0 en1 >"$missing_interface/capture-interfaces.txt"
expect_fail 'missing pcap for a captured interface' "$missing_interface"

unhealthy=$(new_fixture unhealthy)
printf '%s\n' '{"healthy":false}' >"$unhealthy/health.json"
expect_fail 'unhealthy response' "$unhealthy"

echo 'S1 offline evidence validator tests passed'
