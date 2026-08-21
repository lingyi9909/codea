#!/usr/bin/env bash
set -euo pipefail

package_dir=${1:?usage: install.sh <extracted-package-dir>}
version=$(tr -d '[:space:]' < "$package_dir/VERSION")
[ -n "$version" ] || { echo "VERSION missing or empty" >&2; exit 1; }
repo_scripts=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/scripts
"$repo_scripts/verify-checksum.sh" "$package_dir"
"$repo_scripts/verify-offline.sh" "$package_dir"

home=${CODEA_HOME:-$HOME/.codea}
versions="$home/versions"
target="$versions/$version"
mkdir -p "$versions" "$home/bin"
[ ! -e "$target" ] || { echo "version already installed: $version" >&2; exit 1; }
tmp="$versions/.install-$version-$$"
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp"
cp -R "$package_dir/." "$tmp/"
chmod +x "$tmp/bin/codea" "$tmp/bin/opencode"
mv "$tmp" "$target"
trap - EXIT
ln -sfn "$target" "$home/current"
ln -sfn "$home/current/bin/codea" "$home/bin/codea"
printf 'Installed Codea %s\nAdd %s/bin to PATH if needed.\n' "$version" "$home"
