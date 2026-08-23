#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
out_dir=${1:?usage: build-plugins.sh <output-dir>}
plugin_root="$repo_root/distribution/plugins"
command -v bun >/dev/null 2>&1 || { echo "bun is required" >&2; exit 2; }
command -v python3 >/dev/null 2>&1 || { echo "python3 is required" >&2; exit 2; }

(cd "$plugin_root" && bun test && bun run build)
[ -f "$plugin_root/dist/index.js" ] || { echo "plugin bundle missing" >&2; exit 1; }
mkdir -p "$out_dir"
cp "$plugin_root/dist/index.js" "$out_dir/index.js"
rm -f "$out_dir/package.json" "$out_dir/bun.lock" "$out_dir/bun.lockb"

python3 - "$out_dir/index.js" <<'PY'
import pathlib, re, sys
p=pathlib.Path(sys.argv[1])
text=p.read_text(errors='replace')
patterns=[
    re.compile(r'''(?:from\s*|import\s*\(|require\s*\()\s*["']([^"'\n]+)["']'''),
    re.compile(r'''(?:^|[;\n])\s*import\s*["']([^"'\n]+)["']'''),
]
NODE_BUILTINS = frozenset("""assert async_hooks buffer child_process cluster console constants crypto dgram diagnostics_channel dns domain events fs http http2 https inspector module net os path perf_hooks process punycode querystring readline repl stream string_decoder sys timers tls trace_events tty url util v8 vm wasi worker_threads zlib""".split())
def allowed(spec):
    return spec.startswith(('.', '/', 'node:', 'bun:', 'data:')) or spec.split('/')[0] in NODE_BUILTINS
for pat in patterns:
    for spec in pat.findall(text):
        if not allowed(spec):
            raise SystemExit(f'external dependency import found in plugin bundle: {spec}')
print('plugin import audit passed')
PY

echo "plugin bundle ready: self-contained ESM"
