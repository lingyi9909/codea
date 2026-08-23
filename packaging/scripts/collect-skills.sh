#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
out_dir=${1:?usage: collect-skills.sh <output-dir>}
src="$repo_root/distribution/skills"
[ -d "$src" ] || { echo "skills source missing: $src" >&2; exit 1; }
rm -rf "$out_dir"
mkdir -p "$out_dir"
cp -R "$src/." "$out_dir/"
for skill in code-review unit-test api-documentation; do
  [ -f "$out_dir/$skill/SKILL.md" ] || { echo "required skill missing: $skill" >&2; exit 1; }
done
echo "skills collected"
