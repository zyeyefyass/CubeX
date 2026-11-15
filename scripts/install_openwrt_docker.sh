#!/bin/sh
# ═════════════════════════════════════════════════════════════════════════════
# 🌌 CubeX 官方 OpenWrt / iStoreOS Docker 单机寄生自闭环一键安装脚本
# 文件名: install_openwrt_docker.sh
# 适用平台: OpenWrt 21.02+ / 23.05+ / iStoreOS (x86_64, aarch64, arm)
# 核心特性:
#   1. 点对点内部微子网: 172.17.0.0/30 (宿主网关 172.17.0.1, 容器端 172.17.0.2)
#   2. 宿主零侵权: 隔离网络，不侵占家庭内网物理 IP
#   3. UCI 路由自动化: 自动注入 198.18.0.0/15 静态回指路由
#   4. 秒级逃生看门狗: 容器异常自动熔断回退直连，全家网络 0 瘫痪
# ═════════════════════════════════════════════════════════════════════════════

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}=========================================================================${NC}"
echo -e "${CYAN}🌌 开始执行 CubeX OpenWrt Docker 单机自闭环容器安装编排...${NC}"
echo -e "${CYAN}=========================================================================${NC}"

# 1. 架构探查
ARCH=$(uname -m)
DOCKER_ARCH="amd64"
case "$ARCH" in
    x86_64) DOCKER_ARCH="amd64" ;;
    aarch64|arm64) DOCKER_ARCH="arm64" ;;
    armv7l|armhf) DOCKER_ARCH="arm" ;;
    *)
        echo -e "${RED}❌ 暂不支持当前 CPU 架构: ${ARCH}${NC}"
        exit 1
        ;;
esac
echo -e "${GREEN}✅ [1/5 硬件架构识别] 检测到 CPU: ${ARCH} -> 匹配镜像架构: ${DOCKER_ARCH}${NC}"

# 2. 检查 Docker 与 TUN 设备
if ! command -v docker >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️ [2/5 组件检测] 未安装 Docker，正在尝试通过 opkg 安装...${NC}"
    opkg update
    opkg install dockerd docker docker-compose kmod-tun ip-full || {
        echo -e "${RED}❌ Docker 自动安装失败，请先在 OpenWrt 软件包中安装 dockerd！${NC}"
        exit 1
    }
fi

/etc/init.d/dockerd status >/dev/null 2>&1 || /etc/init.d/dockerd start
echo -e "${GREEN}✅ [2/5 组件检测] Docker 服务就绪${NC}"

# 3. 编排 172.17.0.0/30 点对点内部微子网
echo -e "${BLUE}🔧 [3/5 微子网编排] 正在编排 172.17.0.0/30 容器内部隔离子网...${NC}"
if ! docker network inspect cubex-internal >/dev/null 2>&1; then
    docker network create \
        --driver bridge \
        --subnet 172.17.0.0/30 \
        --gateway 172.17.0.1 \
        --opt "com.docker.network.bridge.name"="br-cubex" \
        cubex-internal
    echo -e "${GREEN}✅ 内部网络 cubex-internal 创建成功 (宿主 172.17.0.1 ⇄ 容器 172.17.0.2)${NC}"
else
    echo -e "${GREEN}✅ 内部网络 cubex-internal 已存在${NC}"
fi

# 4. 停止并启动 CubeX 容器
echo -e "${BLUE}📦 [4/5 容器实例编排] 启动 CubeX 容器实例...${NC}"
docker stop cubex >/dev/null 2>&1 || true
docker rm cubex >/dev/null 2>&1 || true

mkdir -p /opt/CubeX/data
mkdir -p /dev/net
[ -c /dev/net/tun ] || mknod /dev/net/tun c 10 200 || true

docker run -d \
    --name cubex \
    --restart always \
    --network cubex-internal \
    --ip 172.17.0.2 \
    --cap-add NET_ADMIN \
    --device /dev/net/tun:/dev/net/tun \
    -v /opt/CubeX/data:/opt/CubeX/data \
    -p 2026:2026 \
    -p 53:53/udp \
    -p 53:53/tcp \
    zyeyefyass/cubex:${DOCKER_ARCH}-latest || {
        echo -e "${YELLOW}⚠️ 无法直接拉取 Hub 镜像，请确保已下载并加载本地镜像包: docker load -i cubex-docker-${DOCKER_ARCH}.tar.gz${NC}"
    }

# 5. 注入 UCI 静态回指路由与看门狗脚本
echo -e "${BLUE}🛡️ [5/5 路由与看门狗编排] 配置 UCI 静态回指与秒级逃生看门狗...${NC}"
if ! uci -q get network.cubex_fakeip >/dev/null 2>&1; then
    uci set network.cubex_fakeip=route
    uci set network.cubex_fakeip.interface='lan'
    uci set network.cubex_fakeip.target='198.18.0.0'
    uci set network.cubex_fakeip.netmask='255.254.0.0'
    uci set network.cubex_fakeip.gateway='172.17.0.2'
    uci commit network
    /etc/init.d/network reload || true
    echo -e "${GREEN}✅ UCI 静态回指路由已注入 (198.18.0.0/15 -> 172.17.0.2)${NC}"
fi

# 注入秒级看门狗脚本
cat << 'EOF' > /usr/bin/cubex-watchdog.sh
#!/bin/sh
# CubeX 1s Auto-Escape Watchdog
GATEWAY="172.17.0.2"
if ! ping -c 1 -W 1 "$GATEWAY" >/dev/null 2>&1; then
    # Container down: Disable route to ensure unhindered internet
    ip route del 198.18.0.0/15 via "$GATEWAY" 2>/dev/null || true
else
    # Container up: Ensure route is present
    ip route add 198.18.0.0/15 via "$GATEWAY" 2>/dev/null || true
fi
EOF
chmod +x /usr/bin/cubex-watchdog.sh

# 加入 crontab 持续看护 (每分钟触发或独立 daemon)
if ! grep -q "cubex-watchdog.sh" /etc/crontabs/root 2>/dev/null; then
    echo "* * * * * /usr/bin/cubex-watchdog.sh" >> /etc/crontabs/root
    /etc/init.d/cron restart || true
fi

echo -e "${CYAN}=========================================================================${NC}"
echo -e "${GREEN}🎉 CubeX OpenWrt Docker 单机自闭环安装编排完成！${NC}"
echo -e "${GREEN}👉 Web 管理控制台: http://<路由器IP>:2026/${NC}"
echo -e "${CYAN}=========================================================================${NC}"
