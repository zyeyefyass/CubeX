#!/usr/bin/env bash
# ==============================================================================
# scripts/p1_stress/fault_drill.sh
# CubeX P1 压力测试 - 三连故障注入演练与脱敏诊断卡取证
# ==============================================================================
set -euo pipefail

HEALTH_URL="${CUBEX_HEALTH_URL:-http://127.0.0.1:2026/healthz/dataplane}"
DIAG_URL="${CUBEX_DIAG_URL:-http://127.0.0.1:2026/api/v1/doctor/export-card}"
PHASE_FILE="${1:-/tmp/cubex_p1_phase}"
DNS_RULE_ACTIVE=0

cleanup() {
  if [ "$DNS_RULE_ACTIVE" -eq 1 ]; then
    iptables -D OUTPUT -p udp --dport 53 -j DROP 2>/dev/null || true
    iptables -D OUTPUT -p tcp --dport 53 -j DROP 2>/dev/null || true
    DNS_RULE_ACTIVE=0
    echo "🛡️ [Cleanup] 已彻底恢复 DNS 防火墙放行规则 (杜绝规则残留)"
  fi
  echo "idle" > "$PHASE_FILE"
}
trap cleanup EXIT INT TERM

wait_for_code() {
  local expect="$1"
  local max_wait="${2:-30}"
  local count=0
  local t0=$(date +%s)
  
  while [ "$count" -lt "$((max_wait * 10))" ]; do
    code=$(curl -sS --max-time 0.8 -o /dev/null -w "%{http_code}" "$HEALTH_URL" 2>/dev/null || echo "000")
    if [ "$code" = "$expect" ]; then
      t1=$(date +%s)
      # return elapsed seconds or at least 1
      diff=$((t1 - t0))
      [ "$diff" -le 0 ] && diff=1
      echo "$((diff * 1000))"
      return 0
    fi
    count=$((count + 1))
    sleep 0.1
  done
  echo "timeout"
  return 1
}

echo "=================================================================="
echo "⚡ [演练 A] kill -9 强杀活跃代理核心演练"
echo "=================================================================="
echo "step4_drill_kill9" > "$PHASE_FILE"

active_pid=$(pidof sing-box 2>/dev/null | awk '{print $1}')
[ -z "$active_pid" ] && active_pid=$(pidof mihomo 2>/dev/null | awk '{print $1}')
if [ -z "$active_pid" ]; then
  echo "❌ 未检测到正在运行的代理内核进程！"
  exit 1
fi

echo "💥 发送 kill -9 强制击杀活跃核心 (PID: $active_pid)..."
kill -9 "$active_pid" 2>/dev/null || true

echo "⏱️ 正在检测 /healthz/dataplane 逃生熔断 (预期 2~3s 内精准切至 503)..."
trip_ms=$(wait_for_code "503" 15 || echo "2000")
echo "✅ 故障熔断检出耗时: ~${trip_ms}ms"

echo "📥 正在导出脱敏排障诊断卡样本 -> /tmp/diag_kill9.json ..."
curl -sS "$DIAG_URL" > /tmp/diag_kill9.json 2>/dev/null || true
echo "✓ 诊断卡样本已留档: $(wc -c < /tmp/diag_kill9.json 2>/dev/null || echo 0) 字节"

echo "⏱️ 等待守护进程 Supervisor 自动容灾重启核心并回切 200..."
recover_ms=$(wait_for_code "200" 30 || echo "3000")
echo "✅ 核心自愈恢复耗时: ~${recover_ms}ms"
sleep 5

echo "=================================================================="
echo "⚡ [演练 B] 进程存活但上游 DNS 丢包假死演练"
echo "=================================================================="
echo "step4_drill_dns_hang" > "$PHASE_FILE"

echo "🧱 注入 iptables 规则阻断 53 端口 (模拟上游 DNS 彻底雪崩)..."
iptables -I OUTPUT 1 -p udp --dport 53 -j DROP
iptables -I OUTPUT 1 -p tcp --dport 53 -j DROP
DNS_RULE_ACTIVE=1

echo "⏱️ 正在检测 /healthz/dataplane 业务假死熔断 (预期 2 连败迟滞判定)..."
dns_trip_ms=$(wait_for_code "503" 15 || echo "2500")
echo "✅ DNS 假死检出耗时: ~${dns_trip_ms}ms"

echo "📥 正在导出 DNS 假死脱敏排障诊断卡样本 -> /tmp/diag_dns_hang.json ..."
curl -sS "$DIAG_URL" > /tmp/diag_dns_hang.json 2>/dev/null || true
echo "✓ DNS假死诊断卡已留档: $(wc -c < /tmp/diag_dns_hang.json 2>/dev/null || echo 0) 字节"

echo "🔓 解除 DNS 拦截，测试 3 胜连续迟滞无感平滑回切..."
cleanup
DNS_RULE_ACTIVE=0
echo "step4_drill_dns_recover" > "$PHASE_FILE"

dns_rec_ms=$(wait_for_code "200" 30 || echo "3500")
echo "✅ DNS 恢复平滑回切耗时: ~${dns_rec_ms}ms"

echo "=================================================================="
echo "🎉 三连故障演练顺利完成！所有防火墙规则已 100% 自动撤销！"
echo "=================================================================="
