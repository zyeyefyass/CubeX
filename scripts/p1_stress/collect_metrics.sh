#!/usr/bin/env bash
# ==============================================================================
# scripts/p1_stress/collect_metrics.sh
# CubeX P1 压力测试 - 1Hz 低开销遥测记录仪 (纯 /proc 精准采样)
# 用法: ./collect_metrics.sh [输出CSV路径] [状态标签文件路径]
# ==============================================================================
set -euo pipefail

OUT_CSV="${1:-/tmp/cubex_p1_telemetry.csv}"
PHASE_FILE="${2:-/tmp/cubex_p1_phase}"
HEALTH_URL="${CUBEX_HEALTH_URL:-http://127.0.0.1:2026/healthz/dataplane}"

mkdir -p "$(dirname "$OUT_CSV")"
echo "ts,phase,cubexd_rss_kb,core_name,core_pid,core_rss_kb,total_rss_mb,open_fds,zombies,conntrack,health_code,health_ms,health_state" > "$OUT_CSV"

get_rss_kb() {
  local pid="$1"
  if [ -n "$pid" ] && [ -r "/proc/$pid/status" ]; then
    awk '/^VmRSS:/ {print $2}' "/proc/$pid/status"
  else
    echo 0
  fi
}

get_fds() {
  local pid="$1"
  if [ -n "$pid" ] && [ -d "/proc/$pid/fd" ]; then
    ls -1 "/proc/$pid/fd" 2>/dev/null | wc -l
  else
    echo 0
  fi
}

echo "🚀 [Collector] 1Hz 低开销遥测记录仪已就绪，正在持续输出到: $OUT_CSV"

while true; do
  ts=$(date +"%Y-%m-%dT%H:%M:%S")
  phase=$(cat "$PHASE_FILE" 2>/dev/null || echo "idle")

  cubexd_pid=$(pidof cubexd 2>/dev/null | awk '{print $1}')
  sb_pid=$(pidof sing-box 2>/dev/null | awk '{print $1}')
  mh_pid=$(pidof mihomo 2>/dev/null | awk '{print $1}')

  core_name="none"
  core_pid=""
  if [ -n "$sb_pid" ]; then
    core_name="sing-box"
    core_pid="$sb_pid"
  elif [ -n "$mh_pid" ]; then
    core_name="mihomo"
    core_pid="$mh_pid"
  fi

  cubexd_rss=$(get_rss_kb "$cubexd_pid")
  [ -z "$cubexd_rss" ] && cubexd_rss=0
  core_rss=$(get_rss_kb "$core_pid")
  [ -z "$core_rss" ] && core_rss=0
  total_mb=$(awk -v a="$cubexd_rss" -v b="$core_rss" 'BEGIN {printf "%.2f", (a+b)/1024}')

  fds_d=$(get_fds "$cubexd_pid")
  [ -z "$fds_d" ] && fds_d=0
  fds_c=$(get_fds "$core_pid")
  [ -z "$fds_c" ] && fds_c=0
  open_fds=$((fds_d + fds_c))

  zombies=$(ps -o stat 2>/dev/null | grep -c '^Z' | tr -d '\r\n' || true)
  [ -z "$zombies" ] && zombies=0
  conntrack=$(cat /proc/sys/net/netfilter/nf_conntrack_count 2>/dev/null | tr -d '\r\n' || true)
  [ -z "$conntrack" ] && conntrack=0

  # 探针状态与耗时(ms) - 优先使用 curl 内置高精度计时器
  resp=$(curl -sS --max-time 1 -w "\n%{http_code} %{time_total}" "$HEALTH_URL" 2>/dev/null || printf '{}\n000 0.000')
  timing_line=$(printf '%s\n' "$resp" | tail -n1)
  health_code=$(echo "$timing_line" | awk '{print $1}')
  time_total_s=$(echo "$timing_line" | awk '{print $2}')
  health_ms=$(awk -v s="$time_total_s" 'BEGIN {printf "%.1f", s*1000}')

  body_json=$(printf '%s\n' "$resp" | sed '$d')
  health_state=$(echo "$body_json" | grep -oE '"state":"[^"]+"' | cut -d'"' -f4 || echo "unknown")
  if [ -z "$health_state" ]; then
    health_state=$(echo "$body_json" | grep -oE '"status":"[^"]+"' | cut -d'"' -f4 || echo "unknown")
  fi

  echo "$ts,$phase,$cubexd_rss,$core_name,$core_pid,$core_rss,$total_mb,$open_fds,$zombies,$conntrack,$health_code,$health_ms,$health_state" >> "$OUT_CSV"
  sleep 1
done
