#!/bin/bash
# S1 真实断网启动验证（修正版 — 正确环境变量 + trap + 全接口抓包）
# 执行方式: ! sudo bash docs/spike-artifacts/s1-network-test.sh

set -euo pipefail

REPO="$(cd "$(dirname "$0")/../.." && pwd)"
ARTIFACTS="$REPO/docs/spike-artifacts"
OP="$ARTIFACTS/opencode"
RUN_ID="s1-$(date +%Y%m%d-%H%M%S)"
RESULT_DIR="$ARTIFACTS/$RUN_ID"
EXEC_LOG="$RESULT_DIR/execution.log"

# --- trap: 无论如何恢复网络 ---
cleanup() {
  echo "=== cleanup: restoring network ===" | tee -a "$EXEC_LOG" 2>/dev/null || true
  for iface in en0 en1 en2 en3 en4 en5 en6 awdl0 llw0 bridge0 ap1; do
    if ifconfig "$iface" >/dev/null 2>&1; then
      sudo ifconfig "$iface" up 2>/dev/null || true
    fi
  done
  echo "=== cleanup: done ==="
}
trap cleanup EXIT INT TERM

# --- 初始化 ---
rm -rf "$RESULT_DIR"
mkdir -p "$RESULT_DIR"

exec > >(tee -a "$EXEC_LOG") 2>&1

echo "=== S1 Offline Startup Verification ==="
echo "=== Run ID: $RUN_ID ==="
echo "=== $(date) ==="

# --- 1. 完全隔离沙箱 ---
SANDBOX="$ARTIFACTS/s1-sandbox"
rm -rf "$SANDBOX"
mkdir -p "$SANDBOX"/{home,config,data,cache,state}

export HOME="$SANDBOX/home"
export XDG_CONFIG_HOME="$SANDBOX/config"
export XDG_DATA_HOME="$SANDBOX/data"
export XDG_CACHE_HOME="$SANDBOX/cache"
export XDG_STATE_HOME="$SANDBOX/state"
export OPENCODE_CONFIG_DIR="$SANDBOX/config/opencode"
export BUN_INSTALL="$SANDBOX/bun"
export NPM_CONFIG_CACHE="$SANDBOX/npm-cache"
mkdir -p "$OPENCODE_CONFIG_DIR"

# --- 2. 正确环境变量（来源: OpenCode v1.18.11 flag.ts / models-dev.ts） ---
export OPENCODE_DISABLE_MODELS_FETCH=1       # 禁用 models.opencode.ai 获取（核心）
export OPENCODE_DISABLE_AUTOUPDATE=1         # 禁用自动更新
export OPENCODE_DISABLE_EMBEDDED_WEB_UI=1    # 禁用内嵌 Web UI
export OPENCODE_DISABLE_LSP_DOWNLOAD=1       # 禁用 LSP 下载
export OPENCODE_DISABLE_DEFAULT_PLUGINS=1    # 禁用默认 Plugin
export OPENCODE_DISABLE_EXTERNAL_SKILLS=1    # 禁用外部 Skill
export OPENCODE_DISABLE_PROJECT_CONFIG=1     # 禁用项目配置
export OPENCODE_DISABLE_CLAUDE_CODE=1        # 禁用 Claude Code 集成
export OPENCODE_SERVER_USERNAME=codea
export OPENCODE_SERVER_PASSWORD=test-s1-offline

echo ""
echo "=== Environment ==="
echo "OPENCODE_DISABLE_MODELS_FETCH=$OPENCODE_DISABLE_MODELS_FETCH"
echo "OPENCODE_DISABLE_AUTOUPDATE=$OPENCODE_DISABLE_AUTOUPDATE"

# --- 3. 断网前接口 ---
echo ""
echo "=== Active interfaces (before) ==="
ifconfig -u | grep -E '^[a-z]' | cut -d: -f1

# --- 4. 全接口抓包 ---
echo ""
echo "=== Starting tcpdump on all external interfaces ==="
TCPDUMP_PIDS=()
for iface in en0 awdl0 llw0 utun0 utun1 utun2 utun3 utun4 utun5; do
  if ifconfig "$iface" >/dev/null 2>&1; then
    sudo tcpdump -i "$iface" -n -w "$RESULT_DIR/traffic-$iface.pcap" \
      > "$RESULT_DIR/tcpdump-$iface.log" 2>&1 &
    TCPDUMP_PIDS+=($!)
    echo "  tcpdump on $iface PID=$!"
  fi
done
sleep 1

# --- 5. 关闭外部网络 ---
echo ""
echo "=== Disabling external interfaces ==="
for iface in en0 en1 en2 en3 en4 en5 en6 awdl0 llw0 bridge0 ap1; do
  if ifconfig "$iface" >/dev/null 2>&1; then
    sudo ifconfig "$iface" down 2>/dev/null && echo "  $iface: down" || true
  fi
done
sleep 1

echo ""
echo "=== Remaining active interfaces ==="
ifconfig -u | grep -E '^[a-z]' | cut -d: -f1

# --- 6. 启动 OpenCode ---
echo ""
echo "=== Starting OpenCode Server ==="
VALIDATION_START=$(python3 -c 'import time; print(time.time())')
"$OP" serve --hostname 127.0.0.1 --port 49325 \
  > "$RESULT_DIR/server-stdout.log" 2>&1 &
OP_PID=$!
echo "OpenCode PID=$OP_PID"
sleep 5

# --- 7. 健康检查 ---
echo ""
echo "=== Health check ==="
curl -sf -u codea:test-s1-offline http://127.0.0.1:49325/global/health \
  | tee "$RESULT_DIR/health.json"
echo

echo "=== Waiting 20s for any delayed outbound attempts ==="
sleep 20
VALIDATION_END=$(python3 -c 'import time; print(time.time())')
python3 - "$RESULT_DIR/validation-window.json" "$VALIDATION_START" "$VALIDATION_END" <<'PY'
import json
import pathlib
import sys

pathlib.Path(sys.argv[1]).write_text(json.dumps({
    "startEpoch": float(sys.argv[2]),
    "endEpoch": float(sys.argv[3]),
}, indent=2) + "\n")
PY

# --- 8. 停止 OpenCode ---
echo ""
echo "=== Stopping OpenCode ==="
kill $OP_PID 2>/dev/null || true
wait $OP_PID 2>/dev/null || true
sleep 2

# --- 9. 停止抓包 ---
echo ""
echo "=== Stopping tcpdump ==="
for pid in "${TCPDUMP_PIDS[@]}"; do
  sudo kill "$pid" 2>/dev/null || true
  wait "$pid" 2>/dev/null || true
done

# --- 10. 收集内部日志 ---
echo ""
echo "=== Collecting OpenCode internal log ==="
if [ -f "$XDG_DATA_HOME/opencode/log/opencode.log" ]; then
  cp "$XDG_DATA_HOME/opencode/log/opencode.log" "$RESULT_DIR/opencode-internal.log"
  echo "  saved ($(wc -l < "$RESULT_DIR/opencode-internal.log") lines)"
else
  echo "  (no internal log found)"
fi

# --- 11. 分析 ---
echo ""
echo "============================================"
echo "=== RESULTS ==="
echo "============================================"

echo ""
echo "--- Health check ---"
cat "$RESULT_DIR/health.json" 2>/dev/null || echo "(missing)"

echo ""
echo "--- Server stdout ---"
cat "$RESULT_DIR/server-stdout.log" 2>/dev/null || echo "(missing)"

echo ""
echo "--- OpenCode internal log ---"
cat "$RESULT_DIR/opencode-internal.log" 2>/dev/null || echo "(missing)"

echo ""
echo "--- Automated evidence verdict ---"
if "$REPO/scripts/check-s1-offline-evidence.sh" "$RESULT_DIR"; then
  validation_status=0
else
  validation_status=$?
fi

echo ""
echo "--- Evidence files ---"
ls -la "$RESULT_DIR"/

echo ""
if [ "$validation_status" -eq 0 ]; then
  echo "=== DONE: PASS ==="
else
  echo "=== DONE: FAIL ==="
fi
exit "$validation_status"
