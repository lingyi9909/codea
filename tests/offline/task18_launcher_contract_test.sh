#!/usr/bin/env bash
set -euo pipefail
repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
mac="$repo_root/packaging/platform/macos/install.sh"
win="$repo_root/packaging/platform/windows/install.ps1"

for key in CODEA_HOME CODEA_RUNTIME_CONFIG_DIR OPENCODE_BIN CODEA_AGENTS_DIR CODEA_SKILLS_DIR CODEA_PLUGIN_BUNDLE; do
  grep -q "$key" "$mac" || { echo "macOS launcher missing $key" >&2; exit 1; }
  grep -q "$key" "$win" || { echo "Windows launcher missing $key" >&2; exit 1; }
done

grep -q 'current.txt' "$win" || { echo "Windows launcher must use current.txt" >&2; exit 1; }
grep -q 'set /p "CODEA_CURRENT="' "$win" || { echo "Windows launcher must resolve current.txt on every start" >&2; exit 1; }
grep -q 'current="$codea_home/current"' "$mac" || { echo "macOS launcher must resolve current symlink" >&2; exit 1; }

echo "task18 launcher contract passed"
