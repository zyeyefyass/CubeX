#!/usr/bin/env bash
# ==============================================================================
# scripts/p1_stress/switch_burnin.sh
# CubeX P1 压力测试 - 带载 10 次双核往返热切拷机 (附 100ms 激进防误熔断探针)
# ==============================================================================
set -euo pipefail

SWITCH_API="${CUBEX_SWITCH_API:-http://127.0.0.1:2026/api/v1/engine/switch}"
HEALTH_URL="${CUBEX_HEALTH_URL:-http://127.0.0.1:2026/healthz/dataplane}"
PHASE_FILE="${1:-/tmp/cubex_p1_phase}"
FALSE_TRIP_LOG="/tmp/cubex_p1_false_trips.log"
: > "$FALSE_TRIP_LOG"

echo "step3_loaded_switch" > "$PHASE_FILE"

echo "🔥 [Burnin] 启动 100ms 激进探针 (比 RouterOS Netwatch 严格 5 倍，专抓切核瞬间误熔断)..."
(
  while true; do
    code=$(curl -sS --max-time 0.5 -o /dev/null -w "%{http_code}" "$HEALTH_URL" 2>/dev/null || echo "000")
    if [ "$code" != "200" ]; then
      ts=$(date +"%H:%M:%S")
      echo "${ts} NON-200 DETECTED: $code" >> "$FALSE_TRIP_LOG"
    fi
    sleep 0.1
  done
) &
POLLER_PID=$!

cleanup() {
  kill "$POLLER_PID" 2>/dev/null || true
  echo "idle" > "$PHASE_FILE"
}
trap cleanup EXIT INT TERM

echo "🚀 [Burnin] 开始带载执行 10 次双核连续往返热切拷机..."
targets=("mihomo" "singbox" "mihomo" "singbox" "mihomo" "singbox" "mihomo" "singbox" "mihomo" "singbox")

for i in "${!targets[@]}"; do
  target="${targets[$i]}"
  round=$((i + 1))
  
  t0=$(date +%s)
  # 使用高精度 curl time_total
  resp=$(curl -sS -X POST "$SWITCH_API" \
    -H "Content-Type: application/json" \
    -d "{\"engine\":\"$target\"}" \
    -w "\n%{http_code} %{time_total}" 2>/dev/null || printf '{}\n000 0.000')
  timing=$(echo "$resp" | tail -n1)
  http_code=$(echo "$timing" | awk '{print $1}')
  time_total_s=$(echo "$timing" | awk '{print $2}')
  cost=$(awk -v s="$time_total_s" 'BEGIN {printf "%.0f", s*1000}')

  # 切核后立刻验证国内外双栈连通性与僵尸进程
  cn_code=$(curl -sS --max-time 3 -o /dev/null -w "%{http_code}" https://www.baidu.com 2>/dev/null || echo "000")
  intl_code=$(curl -sS --max-time 5 -o /dev/null -w "%{http_code}" https://github.com 2>/dev/null || echo "000")
  z_cnt=$(ps -o stat 2>/dev/null | grep -c '^Z' | tr -d '\r\n' || true)
  [ -z "$z_cnt" ] && z_cnt=0

  printf "[Round %02d -> %-8s] api=%s cost=%4dms | CN=%s INTL=%s | zombies=%s\n" \
    "$round" "$target" "$http_code" "$cost" "$cn_code" "$intl_code" "$z_cnt"
  sleep 6
done

kill "$POLLER_PID" 2>/dev/null || true
trap - EXIT INT TERM
echo "idle" > "$PHASE_FILE"

false_trip_cnt=$(wc -l < "$FALSE_TRIP_LOG" 2>/dev/null || echo 0)
echo "=================================================================="
echo "📊 [Burnin Summary] 100ms 激进探针误报总数: ${false_trip_cnt} 次"
if [ "$false_trip_cnt" -gt 0 ]; then
  cat "$FALSE_TRIP_LOG"
else
  echo "✅ 零误熔断通过！1500ms 计划内切核宽限态完美护航！"
fi
echo "=================================================================="
