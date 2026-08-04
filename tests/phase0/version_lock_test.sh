#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)

python3 - "$repo_root/runtime/version.json" "$repo_root/docs/spike-artifacts/s1-release.json" "$repo_root/docs/spike-artifacts/s1-release-checksums.txt" <<'PY'
import json
import pathlib
import re
import sys

version_path, evidence_path, checksums_path = map(pathlib.Path, sys.argv[1:])
version = json.loads(version_path.read_text())
evidence = json.loads(evidence_path.read_text())

required = {"linux-x64", "darwin-arm64", "darwin-x64", "windows-x64"}
platforms = version.get("platforms", {})
assert set(platforms) == required, f"platform lock must contain exactly {sorted(required)}"
assert version.get("openCodeVersion") == "1.18.11"
assert version.get("openCodeCommit") == "012c2f57f976489d88bd4598a056b4bdcdd428ee"
assert version.get("releaseMetadataSource") == "https://api.github.com/repos/anomalyco/opencode/releases/tags/v1.18.11"

assets = []
expected_lines = []
for platform in sorted(required):
    item = platforms[platform]
    assert set(item) == {"asset", "size", "checksum", "url"}, f"incomplete lock for {platform}"
    assert isinstance(item["size"], int) and item["size"] > 0
    assert re.fullmatch(r"sha256:[0-9a-f]{64}", item["checksum"]), f"invalid checksum for {platform}"
    assert "TBD" not in json.dumps(item)
    assert item["url"] == f"https://github.com/anomalyco/opencode/releases/download/v1.18.11/{item['asset']}"
    assets.append(item["asset"])
    expected_lines.append(f"{item['checksum'].removeprefix('sha256:')}  {item['asset']}")
    recorded = evidence["assets"][platform]
    assert recorded["name"] == item["asset"]
    assert recorded["size"] == item["size"]
    assert recorded["sha256"] == item["checksum"].removeprefix("sha256:")
    assert recorded["downloadUrl"] == item["url"]
    assert recorded["verifiedByDownload"] is True

assert len(assets) == len(set(assets)), "platforms must not share an asset"
actual_lines = [line for line in checksums_path.read_text().splitlines() if line]
assert actual_lines == expected_lines, "downloaded checksum evidence does not match version lock"
print("OpenCode v1.18.11 platform version lock is complete and consistent")
PY
