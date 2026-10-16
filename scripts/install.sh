#!/usr/bin/env bash
# ════════════════════════════════════════════════════════════════════
# 🌌 CubeX 官方一键极速安装与部署脚本
# ════════════════════════════════════════════════════════════════════
set -e

INSTALL_DIR="/opt/CubeX"
BIN_DIR="/usr/local/bin"

echo "🌌 欢迎使用 CubeX · 次世代双擎高可用旁路网关 (Mihomo × sing-box v1.14+)..."

# 1. 架构检测
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)
        TARGET_ARCH="amd64"
        ;;
    aarch64|arm64)
        TARGET_ARCH="arm64"
        ;;
    *)
        echo "❌ 不支持的 CPU 架构: $ARCH"
        exit 1
        ;;
esac
echo "🔍 检测到系统架构: $TARGET_ARCH"

# 2. 目录创建
mkdir -p "$INSTALL_DIR/bin" "$INSTALL_DIR/etc" "$INSTALL_DIR/data"

# 3. 安装服务
cat <<EOF > /etc/systemd/system/cubex.service
[Unit]
Description=CubeX Dual-Engine Gateway Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/bin/cubexd -dir $INSTALL_DIR
Restart=always
RestartSec=3s
LimitNOFILE=65535
Environment=GOMEMLIMIT=60MiB
Environment=GOGC=30

[Install]
WantedBy=multi-user.target
EOF

echo "✅ systemd 服务已生成 (/etc/systemd/system/cubex.service)"
echo "💡 启动命令: systemctl daemon-reload && systemctl enable --now cubex"
