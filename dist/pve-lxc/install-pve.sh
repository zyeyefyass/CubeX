#!/usr/bin/env bash
# ════════════════════════════════════════════════════════════════════
# 🌌 CubeX Proxmox VE (PVE) LXC 容器自动创建与 TUN 穿透脚本
# ════════════════════════════════════════════════════════════════════
set -e

CT_ID=${1:-8228}
GATEWAY_IP=${2:-"10.0.0.8"}
ROUTER_IP=${3:-"10.0.0.1"}

echo "🌌 正在为 CubeX 创建 PVE LXC 极简旁路容器 (CT ID: $CT_ID)..."

# 1. 确保 PVE 节点加载 tun 模块
modprobe tun || true
echo "tun" >> /etc/modules-load.d/cubex-tun.conf || true

# 2. 检查 LXC 配置文件并注入 TUN 与 /dev/net/tun 穿透权限
CONF_FILE="/etc/pve/lxc/${CT_ID}.conf"
if [ -f "$CONF_FILE" ]; then
    echo "⚙️ 正在向 $CONF_FILE 注入 TUN 与网络特权权限..."
    grep -q "lxc.cgroup2.devices.allow: c 10:200 rwm" "$CONF_FILE" || cat <<EOF >> "$CONF_FILE"
# ── CubeX 专用 TUN 穿透权限 ──
lxc.cgroup2.devices.allow: c 10:200 rwm
lxc.mount.entry: /dev/net/tun dev/net/tun none bind,create=file
EOF
    echo "✅ LXC 配置已生效，请重启容器: pct reboot $CT_ID"
else
    echo "💡 提示: 容器 $CT_ID 尚未创建，请在创建后运行此脚本完成 TUN 权限打通。"
fi
