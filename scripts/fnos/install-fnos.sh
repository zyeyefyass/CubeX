#!/usr/bin/env bash
# ==============================================================================
# 🌌 CubeX Test v5.3 · 飞牛 fnOS 专属极简一键直装脚本
# 专为 飞牛 fnOS (Debian 底座) 量身定制：自动探查网卡、持久化卷编排与开箱即用
# ==============================================================================
set -euo pipefail

echo -e "\033[1;36m=================================================================\033[0m"
echo -e "\033[1;36m  🌌 CubeX Test v5.3 · 飞牛 fnOS 专属宿主共生透明网关部署  \033[0m"
echo -e "\033[1;36m  🏆 Docker Hub 10K+ 里程碑 · 网关+DNS全指主路由 · 0改动躺平  \033[0m"
echo -e "\033[1;36m=================================================================\033[0m"

# 1. 检查 root 权限
if [ "$(id -u)" -ne 0 ]; then
    echo -e "\033[1;31m❌ 请使用 root 权限或 sudo 运行此脚本！\033[0m"
    exit 1
fi

# 2. 检查 Docker 环境
if ! command -v docker &> /dev/null; then
    echo -e "\033[1;31m❌ 未检测到 Docker 环境，请先在飞牛应用中心安装 Docker！\033[0m"
    exit 1
fi

# 3. 动态嗅探宿主 IP 与主路由网关 (严禁硬编码)
DEFAULT_ROUTE=$(ip route show default 2>/dev/null | head -n 1 || true)
GATEWAY_IP=$(echo "$DEFAULT_ROUTE" | awk '{print $3}' || true)
DEFAULT_IFACE=$(echo "$DEFAULT_ROUTE" | awk '{print $5}' || true)

if [ -n "$DEFAULT_IFACE" ]; then
    HOST_IP=$(ip -4 addr show "$DEFAULT_IFACE" 2>/dev/null | grep -oP '(?<=inet\s)\d+(\.\d+){3}' | head -n 1 || true)
else
    HOST_IP=$(hostname -I 2>/dev/null | awk '{print $1}' || true)
fi

echo -e "\033[1;32m✓ 动态网络探查成功:\033[0m"
echo -e "  * 飞牛 NAS IP   : \033[1;33m${HOST_IP:-自动获取}\033[0m"
echo -e "  * 主路由网关 IP : \033[1;33m${GATEWAY_IP:-自动获取}\033[0m"
echo -e "  * 物理网络接口  : \033[1;33m${DEFAULT_IFACE:-默认}\033[0m"

# 4. 探查飞牛持久化存储池 (优先 /vol1, 其次 /vol2, 回退 /var/lib/cubex)
STORAGE_BASE="/var/lib/cubex"
if [ -d "/vol1/1000" ]; then
    STORAGE_BASE="/vol1/1000/docker/cubex"
elif [ -d "/vol1" ]; then
    STORAGE_BASE="/vol1/docker/cubex"
elif [ -d "/vol2" ]; then
    STORAGE_BASE="/vol2/docker/cubex"
fi

echo -e "  * 数据持久化路径: \033[1;33m${STORAGE_BASE}\033[0m"
mkdir -p "${STORAGE_BASE}/etc" "${STORAGE_BASE}/data" "${STORAGE_BASE}/rules"

# 5. 确保 TUN 驱动模块加载
if [ ! -c /dev/net/tun ]; then
    echo -e "\033[1;33m🔧 正在初始化 /dev/net/tun 虚拟网卡通道...\033[0m"
    mkdir -p /dev/net
    mknod /dev/net/tun c 10 200 || true
    chmod 666 /dev/net/tun || true
fi
modprobe tun 2>/dev/null || true

# 6. 停止并清理旧容器
if docker ps -a --format '{{.Names}}' | grep -Eq '^cubex$'; then
    echo -e "\033[1;33m🔄 正在更新现有 CubeX 容器...\033[0m"
    docker stop cubex >/dev/null 2>&1 || true
    docker rm cubex >/dev/null 2>&1 || true
fi

# 7. 拉取最新官方镜像 (Docker Hub 10K+ 官方源)
echo -e "\033[1;32m🚀 正在拉取官方最新镜像 (fyasshole/cubex:latest)...\033[0m"
docker pull fyasshole/cubex:latest

# 8. 启动容器 (host 网络模式，零性能折损)
echo -e "\033[1;32m🚀 正在启动 CubeX 宿主共生容器...\033[0m"
docker run -d \
  --name cubex \
  --restart always \
  --net=host \
  --privileged \
  --cap-add=NET_ADMIN \
  --cap-add=NET_RAW \
  --device /dev/net/tun:/dev/net/tun \
  -v "${STORAGE_BASE}/etc:/opt/CubeX/etc" \
  -v "${STORAGE_BASE}/data:/opt/CubeX/data" \
  -v "${STORAGE_BASE}/rules:/opt/CubeX/rules" \
  -v /etc/localtime:/etc/localtime:ro \
  -e TZ=Asia/Shanghai \
  -e CUBEX_ENV=production \
  -e GOMEMLIMIT=128MiB \
  -e CUBEX_PORT=2026 \
  -e CUBEX_LISTEN=:2026 \
  fyasshole/cubex:latest

echo -e "\n\033[1;32m=================================================================\033[0m"
echo -e "\033[1;32m  🎉 恭喜！CubeX Test v5.3 在飞牛 fnOS 上启动成功！            \033[0m"
echo -e "\033[1;32m=================================================================\033[0m"
echo -e "👉 \033[1;37mWeb 控制台访问入口:\033[0m \033[1;36mhttp://${HOST_IP:-127.0.0.1}:2026\033[0m"
echo -e "👉 \033[1;37m主路由联动两步法:\033[0m"
echo -e "   1. 登录控制台【宿主共生】输入 RouterOS / 爱快 凭据，点击一键自愈；"
echo -e "   2. 全屋终端保持默认 DHCP，网关+DNS全指主路由，永不断线！"
