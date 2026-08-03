#!/bin/bash
set -euo pipefail

# ============================================================
# S1 真实断网启动验证
# 执行方式: ! bash /tmp/s1-network-test.sh
# ============================================================

REPO="/Users/zhangzhanhui/Documents/job/codea"
ARTIFACTS="$REPO/docs/spike-artifacts"
OP="$ARTIFACTS/opencode"

# --- 1. 完全隔离的环境 ---
SANDBOX="$ARTIFACTS/s1-sandbox"
rm -rf "$SANDBOX"
mkdir -p "$SANDBOX"/{home,config,data,cache,state}

export HOME="$SANDBOX/home"
export XDG_CONFIG_HOME="$SANDBOX/config"
export XDG_DATA_HOME="$SANDBOX/data"
export XDG_CACHE_HOME="$SANDBOX/cache"
export XDG_STATE_HOME="$SANDBOX/state"
export OPENCODE_CONFIG_DIR="$SANDBOX/config/opencode"

# 额外的 Bun/Node 隔离
export BUN_INSTALL="$SANDBOX/bun"
export NPM_CONFIG_CACHE="$SANDBOX/npm-cache"

export OPENCODE_OFFLINE_MODE=1
export OPENCODE_DISABLE_AUTO_UPDATE=1
export OPENCODE_DISABLE_LSP_DOWNLOAD=1
export OPENCODE_DISABLE_DEFAULT_PLUGINS=1
export OPENCODE_DISABLE_EXTERNAL_SKILLS=1
export OPENCODE_SKIP_MODEL_FETCH=1
export OPENCODE_SKIP_WEB_UI=1
export OPENCODE_SERVER_USERNAME=codea
export OPENCODE_SERVER_PASSWORD=test-s1-offline

mkdir -p "$OPENCODE_CONFIG_DIR"

echo "=== S1: Real offline startup verification ==="
echo "=== $(date) ==="

# --- 2. 记录当前活跃的网络接口 ---
echo ""
echo "=== Active network interfaces (before) ==="
ifconfig -u | grep -E '^[a-z]' | cut -d: -f1

# --- 3. 启动持续抓包（all interfaces, 60s） ---
PCAP="$ARTIFACTS/s1-offline-real.pcap"
echo ""
echo "=== Starting tcpdump on all non-loopback interfaces ==="
# 抓取所有非 lo0 接口的流量，包括 DNS
sudo tcpdump -i en0 -n -w "$PCAP" \
  not host 127.0.0.1 and not host ::1 \
  > "$ARTIFACTS/s1-tcpdump.log" 2>&1 &
TCPDUMP_PID=$!
sleep 1
echo "tcpdump PID=$TCPDUMP_PID"

# --- 4. 关闭网络 ---
echo ""
echo "=== Disabling external network interfaces ==="
for iface in en0 en1 en2 en3 en4 en5 en6 awdl0 llw0 bridge0 ap1; do
  if ifconfig "$iface" >/dev/null 2>&1; then
    sudo ifconfig "$iface" down 2>/dev/null && echo "  $iface: down" || true
  fi
done
sleep 1

# 确认状态
echo ""
echo "=== Remaining active interfaces ==="
ifconfig -u | grep -E '^[a-z]' | cut -d: -f1

# --- 5. 启动 OpenCode ---
echo ""
echo "=== Starting OpenCode Server ==="
"$OP" serve --hostname 127.0.0.1 --port 49325 \
  > "$ARTIFACTS/s1-offline-real-server.log" 2>&1 &
OP_PID=$!
echo "OpenCode PID=$OP_PID"

# 等待稳定
sleep 5

# --- 6. 健康检查 ---
echo ""
echo "=== Health check ==="
curl -sf -u codea:test-s1-offline http://127.0.0.1:49325/global/health \
  | tee "$ARTIFACTS/s1-offline-real-health.json"
echo

# 再等 15 秒，给够时间观察是否有延迟的对外请求
echo "=== Waiting 15s for any delayed outbound attempts ==="
sleep 15

# --- 7. 停止 OpenCode ---
echo ""
echo "=== Stopping OpenCode ==="
kill $OP_PID 2>/dev/null || true
wait $OP_PID 2>/dev/null || true

# --- 8. 停止抓包 ---
sleep 2
echo "=== Stopping tcpdump ==="
sudo kill $TCPDUMP_PID 2>/dev/null || true
wait $TCPDUMP_PID 2>/dev/null || true

# --- 9. 恢复网络 ---
echo ""
echo "=== Restoring network interfaces ==="
for iface in en0 en1 en2 en3 en4 en5 en6 awdl0 llw0 bridge0 ap1; do
  if ifconfig "$iface" >/dev/null 2>&1; then
    sudo ifconfig "$iface" up 2>/dev/null && echo "  $iface: up" || true
  fi
done
sleep 2

echo ""
echo "=== Network restored ==="

# --- 10. 分析结果 ---
echo ""
echo "============================================"
echo "=== RESULTS ==="
echo "============================================"

echo ""
echo "--- Health check response ---"
cat "$ARTIFACTS/s1-offline-real-health.json"

echo ""
echo "--- Server log ---"
cat "$ARTIFACTS/s1-offline-real-server.log"

echo ""
echo "--- Packet capture summary ---"
PKT_COUNT=$(tcpdump -r "$PCAP" 2>/dev/null | wc -l)
echo "Total packets captured: $PKT_COUNT"

echo ""
echo "--- DNS queries (port 53) ---"
tcpdump -r "$PCAP" -n 'port 53' 2>/dev/null | head -20 || echo "(none)"

echo ""
echo "--- HTTP/HTTPS (port 80, 443) ---"
tcpdump -r "$PCAP" -n 'port 80 or port 443' 2>/dev/null | head -20 || echo "(none)"

echo ""
echo "--- All captured traffic (first 50 packets) ---"
tcpdump -r "$PCAP" -n 2>/dev/null | head -50 || echo "(none)"

echo ""
echo "--- SANDBOX PATH ---"
echo "HOME=$HOME"
echo "XDG_CONFIG_HOME=$XDG_CONFIG_HOME"
echo "XDG_DATA_HOME=$XDG_DATA_HOME"
echo "XDG_CACHE_HOME=$XDG_CACHE_HOME"
echo "XDG_STATE_HOME=$XDG_STATE_HOME"
echo "OPENCODE_CONFIG_DIR=$OPENCODE_CONFIG_DIR"
echo "All data/cache/state was empty at startup."

echo ""
echo "=== DONE ==="
echo "Evidence files:"
echo "  PCAP:        $PCAP"
echo "  Server log:  $ARTIFACTS/s1-offline-real-server.log"
echo "  Health:      $ARTIFACTS/s1-offline-real-health.json"
