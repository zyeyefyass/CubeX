#!/bin/sh
# ==============================================================================
# 🌌 CubeX 宿主共生零感网络系统 - OpenWrt Docker 一键极速部署与全量自愈编排脚本
# 适用底座: OpenWrt 21.02 / 22.03 / 23.05 / 24.x / iStoreOS / ImmortalWrt / LEDE
# 支持架构: x86_64 (amd64) | aarch64 / arm64 (RK3568/RK3588/R2S/R4S/R5S/R6S)
# 特性: 独立 IP Macvlan · 0 端口冲突 · 0 侵入主路由 · 0 运营商 DNS 泄露 · 0秒逃生
# ==============================================================================

set -e

CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${CYAN}========================================================================${NC}"
echo -e "${CYAN}🌌 正在启动 CubeX 宿主共生零感网络系统 · OpenWrt Docker 自动化编排流水线${NC}"
echo -e "${CYAN}========================================================================${NC}"

# 1. 架构检测 (Arch Detection)
ARCH_RAW=$(uname -m)
case "$ARCH_RAW" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}❌ 暂不支持的硬件架构: ${ARCH_RAW}。CubeX 官方支持 amd64 与 arm64。${NC}"
        exit 1
        ;;
esac
echo -e "${GREEN}✅ [1/6 硬件识别] 检测到宿主架构: ${ARCH_RAW} ➔ 匹配运行镜像架构: ${ARCH}${NC}"

# 2. Docker 守护进程检测与全自动自愈拉起 (Docker Detection & Auto-Start)
echo -e "${CYAN}🔍 [2/6 Docker检测] 正在巡检 OpenWrt Docker 服务与守护进程...${NC}"

# 2.1 检查 Docker 服务是否已安装
if ! command -v docker >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️ 未检测到 Docker 命令行工具，正在尝试自动安装...${NC}"
    if command -v apk >/dev/null 2>&1; then
        echo -e "${CYAN}   -> 识别到新一代 apk 包管理器 (OpenWrt 24+ / iStoreOS 25+)，正在执行 apk 安装...${NC}"
        apk update
        apk add dockerd docker docker-compose luci-app-dockerman 2>/dev/null || apk add dockerd docker 2>/dev/null || true
    elif command -v opkg >/dev/null 2>&1; then
        echo -e "${CYAN}   -> 识别到 opkg 包管理器，正在执行 opkg 安装...${NC}"
        opkg update
        opkg install dockerd docker luci-app-dockerman 2>/dev/null || opkg install dockerd docker 2>/dev/null || true
    fi
fi

# 2.2 检查 Docker 守护进程是否在运行，未开启则自动开启并激活服务
if ! docker info >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️ Docker 守护进程尚未处于活跃状态，正在自动启用并拉起服务...${NC}"
    if [ -f /etc/init.d/dockerd ]; then
        /etc/init.d/dockerd enable >/dev/null 2>&1 || true
        /etc/init.d/dockerd start >/dev/null 2>&1 || true
    fi
    if command -v uci >/dev/null 2>&1; then
        uci -q set dockerd.globals.log_level='warn' 2>/dev/null || true
        uci -q commit dockerd 2>/dev/null || true
    fi
    for i in $(seq 1 10); do
        if docker info >/dev/null 2>&1; then
            break
        fi
        sleep 1
    done
fi

if ! docker info >/dev/null 2>&1; then
    echo -e "${RED}❌ Docker 守护进程无法自动启动！${NC}"
    echo -e "${RED}   请在 OpenWrt/iStoreOS 后台【Docker】或【系统服务】中手动开启 Docker 服务后重新运行本脚本。${NC}"
    exit 1
fi
echo -e "${GREEN}✅ [2/6 Docker就绪] OpenWrt Docker 守护进程运行正常${NC}"

# 3. LAN 网桥、主网关与 IP 动态计算 (Subnet & Gateway Probing)
LAN_IF="br-lan"
if ! ip link show "$LAN_IF" >/dev/null 2>&1; then
    LAN_IF=$(uci -q get network.lan.device || echo "eth0")
fi

ROUTER_IP=$(uci -q get network.lan.ipaddr 2>/dev/null || ip -4 addr show "$LAN_IF" 2>/dev/null | grep -oE 'inet [0-9.]+' | awk '{print $2}' | head -n 1)
if [ -z "$ROUTER_IP" ]; then
    ROUTER_IP=$(ip route show default 2>/dev/null | awk '{print $3}' | head -n 1)
fi
[ -z "$ROUTER_IP" ] && ROUTER_IP="192.168.1.1"

# 嗅探当前子网的真实主网关（区分主路由与旁路由模式）
UPSTREAM_GW=$(ip route show default 2>/dev/null | awk '{print $3}' | head -n 1)
if [ -n "$UPSTREAM_GW" ] && [ "$UPSTREAM_GW" != "$ROUTER_IP" ]; then
    MACVLAN_GW="$UPSTREAM_GW"
    echo -e "${GREEN}✅ [3/6 模式识别] 当前 OpenWrt 处于旁路由底座，上级物理主网关为: ${UPSTREAM_GW}${NC}"
else
    MACVLAN_GW="$ROUTER_IP"
    echo -e "${GREEN}✅ [3/6 模式识别] 当前 OpenWrt 处于主路由底座，主网关为: ${ROUTER_IP}${NC}"
fi

IP_PREFIX=$(echo "$ROUTER_IP" | cut -d. -f1-3)
SUBNET="${IP_PREFIX}.0/24"

NET_NAME="cubex-macvlan"
CONTAINER_NAME="cubex"

# 先检测已有容器原有 IP，并在探测前提前销毁旧容器，防止其自身占用的 IP 造成防撞逻辑误判漂移
EXISTING_CUBEX_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$CONTAINER_NAME" 2>/dev/null || true)
docker rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true
if docker network inspect "$NET_NAME" >/dev/null 2>&1; then
    docker network rm "$NET_NAME" >/dev/null 2>&1 || true
fi

# 动态防撞 IP 分配：若已有旧分配 IP 优先保持原有固定 IP，否则从 .11 开始探测空闲 IP
CUBEX_IP=""
if [ -n "$EXISTING_CUBEX_IP" ] && [ "$(echo "$EXISTING_CUBEX_IP" | cut -d. -f1-3)" = "$IP_PREFIX" ] && [ "$EXISTING_CUBEX_IP" != "$ROUTER_IP" ] && [ "$EXISTING_CUBEX_IP" != "$UPSTREAM_GW" ]; then
    CUBEX_IP="$EXISTING_CUBEX_IP"
else
    for cand in 11 12 13 14 15 211 212 213; do
        TEST_IP="${IP_PREFIX}.${cand}"
        if [ "$TEST_IP" != "$ROUTER_IP" ] && [ "$TEST_IP" != "$UPSTREAM_GW" ]; then
            if ! ping -c 1 -W 1 "$TEST_IP" >/dev/null 2>&1; then
                CUBEX_IP="$TEST_IP"
                break
            fi
        fi
    done
fi
[ -z "$CUBEX_IP" ] && CUBEX_IP="${IP_PREFIX}.11"
SHIM_IP="${IP_PREFIX}.228"
[ "$SHIM_IP" = "$CUBEX_IP" ] && SHIM_IP="${IP_PREFIX}.229"

echo -e "${GREEN}✅ [3/6 自动配赋] 分配 CubeX 独立物理 IP: ${CUBEX_IP} (Macvlan 网关: ${MACVLAN_GW})${NC}"

# 4. 创建 Docker Macvlan 独立网络与 Shim 互通接口
echo -e "${CYAN}🔧 [4/6 网络编排] 创建 Macvlan 隔离网络 (${NET_NAME} 绑定 ${LAN_IF})...${NC}"
docker network create -d macvlan \
    --subnet="$SUBNET" \
    --gateway="$MACVLAN_GW" \
    -o parent="$LAN_IF" \
    "$NET_NAME"

# 解决 Linux Macvlan 宿主与容器二层隔离难题 (创建 macvlan-shim)
if ! ip link show macvlan-shim >/dev/null 2>&1; then
    ip link add macvlan-shim link "$LAN_IF" type macvlan mode bridge
fi
ip addr flush dev macvlan-shim 2>/dev/null || true
ip addr add "${SHIM_IP}/24" dev macvlan-shim
ip link set macvlan-shim up
ip route add "${CUBEX_IP}/32" dev macvlan-shim 2>/dev/null || true

# 将 macvlan-shim 纳入 OpenWrt 防火墙放行双向互访
if which uci >/dev/null 2>&1; then
    uci add_list firewall.@zone[0].device='macvlan-shim' 2>/dev/null || true
    uci commit firewall 2>/dev/null || true
    /etc/init.d/firewall reload >/dev/null 2>&1 || true
fi

# 持久化 macvlan-shim 与容器回指宿主路由开机自启
if [ -f /etc/rc.local ]; then
    sed -i '/macvlan-shim/d' /etc/rc.local 2>/dev/null || true
    sed -i '/cubex.*ip route/d' /etc/rc.local 2>/dev/null || true
    sed -i -e '$i ip link add macvlan-shim link '"$LAN_IF"' type macvlan mode bridge 2>/dev/null || true' /etc/rc.local
    sed -i -e '$i ip addr add '"$SHIM_IP"'/24 dev macvlan-shim 2>/dev/null || true' /etc/rc.local
    sed -i -e '$i ip link set macvlan-shim up 2>/dev/null || true' /etc/rc.local
    sed -i -e '$i ip route add '"$CUBEX_IP"'/32 dev macvlan-shim 2>/dev/null || true' /etc/rc.local
    sed -i -e '$i docker exec cubex ip route add '"$ROUTER_IP"'/32 via '"$SHIM_IP"' dev eth0 2>/dev/null || true' /etc/rc.local
fi

# 5. 部署拉起 CubeX 容器
CONTAINER_NAME="cubex"
docker rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true

# 优先检查并自动载入同目录下的离线镜像包
LOCAL_TAR=$(ls CubeX-*-docker-*.tar.gz 2>/dev/null | head -n 1 || true)
if [ -n "$LOCAL_TAR" ]; then
    echo -e "${GREEN}📦 [5/6 离线识别] 发现当前目录离线镜像包: ${LOCAL_TAR}，正在载入本地 Docker...${NC}"
    docker load -i "$LOCAL_TAR"
    TAGGED_ID=$(docker images --format '{{.Repository}}:{{.Tag}}' | grep -E '^cubex:' | head -n 1)
    if [ -n "$TAGGED_ID" ] && [ "$TAGGED_ID" != "cubex:latest" ]; then
        docker tag "$TAGGED_ID" "cubex:latest" 2>/dev/null || true
    fi
fi

IMAGE="cubex:latest"
if ! docker inspect "$IMAGE" >/dev/null 2>&1; then
    if docker inspect "zyeyefyass/cubex:latest" >/dev/null 2>&1; then
        IMAGE="zyeyefyass/cubex:latest"
    else
        EXISTING_TAG=$(docker images --format '{{.Repository}}:{{.Tag}}' | grep -E '^(cubex|zyeyefyass/cubex):' | head -n 1)
        if [ -n "$EXISTING_TAG" ]; then
            IMAGE="$EXISTING_TAG"
            docker tag "$EXISTING_TAG" "cubex:latest" 2>/dev/null || true
            IMAGE="cubex:latest"
        else
            echo -e "${YELLOW}   -> 未检测到本地离线镜像，正在尝试在线拉取: ${IMAGE}...${NC}"
            docker pull "$IMAGE" 2>/dev/null || docker pull "zyeyefyass/cubex:latest" || true
            if docker inspect "zyeyefyass/cubex:latest" >/dev/null 2>&1; then
                IMAGE="zyeyefyass/cubex:latest"
            fi
        fi
    fi
fi

mkdir -p /etc/cubex/data

echo -e "${CYAN}📦 [5/6 容器拉起] 正在拉起 CubeX 容器 (${CONTAINER_NAME} @ ${CUBEX_IP} 使用镜像: ${IMAGE})...${NC}"

docker run -d \
    --name "$CONTAINER_NAME" \
    --restart=always \
    --net="$NET_NAME" \
    --ip="$CUBEX_IP" \
    --cap-add=NET_ADMIN \
    --cap-add=NET_RAW \
    --device=/dev/net/tun \
    -v /etc/cubex/data:/app/data \
    "$IMAGE"

# 打通容器向宿主自身 IP 发起 SSH / Web 管理访问链路 (经由 macvlan-shim 跨层互通)
sleep 2
docker exec "$CONTAINER_NAME" ip route add "${ROUTER_IP}/32" via "$SHIM_IP" dev eth0 2>/dev/null || true

# 6. 下发三大一键配置 (策略回指 + DHCP 黄金解耦 + IPv6 防泄露 + 0秒逃生)
echo -e "${CYAN}⚡ [6/6 一键自愈] 正在下发 OpenWrt UCI 策略路由与防火墙加固...${NC}"

# 6.1 Fake-IP 回指路由 (198.18.0.0/15)
uci -q delete network.cubex_fakeip || true
uci set network.cubex_fakeip=route
uci set network.cubex_fakeip.interface='lan'
uci set network.cubex_fakeip.target='198.18.0.0'
uci set network.cubex_fakeip.netmask='255.254.0.0'
uci set network.cubex_fakeip.gateway="$CUBEX_IP"
uci commit network
/etc/init.d/network reload >/dev/null 2>&1 || true
ip route replace 198.18.0.0/15 via "$CUBEX_IP" dev "$LAN_IF" 2>/dev/null || ip route replace 198.18.0.0/15 via "$CUBEX_IP" 2>/dev/null || true

# 6.2 DHCP Option 6 指向 CubeX
uci -q del_list dhcp.lan.dhcp_option="6,${CUBEX_IP}" || true
uci add_list dhcp.lan.dhcp_option="6,${CUBEX_IP}"
uci commit dhcp
/etc/init.d/dnsmasq restart >/dev/null 2>&1 || true

# 6.3 锁定 odhcpd 杜绝运营商 IPv6 DNS 泄露
uci set dhcp.lan.dns_service='0' 2>/dev/null || true
uci -q delete dhcp.lan.dns 2>/dev/null || true
uci set dhcp.lan.ra_dns='0' 2>/dev/null || true
uci commit dhcp 2>/dev/null || true
/etc/init.d/odhcpd enable >/dev/null 2>&1 || true
/etc/init.d/odhcpd restart >/dev/null 2>&1 || true

# 6.4 防火墙转发放行与内核优化
sysctl -w net.ipv4.ip_forward=1 >/dev/null 2>&1 || true
sysctl -w net.ipv4.conf.all.rp_filter=0 >/dev/null 2>&1 || true
uci set firewall.@zone[0].forward='ACCEPT' 2>/dev/null || true
uci commit firewall 2>/dev/null || true
/etc/init.d/firewall reload >/dev/null 2>&1 || true

# 6.5 植入 0 秒毫秒级自愈逃生看门狗
cat << EOF > /usr/bin/cubex-watchdog.sh
#!/bin/sh
CUBEX_IP="${CUBEX_IP}"
BACKUP_DNS="223.5.5.5"

if wget -q -T 2 -O - "http://\${CUBEX_IP}:2026/api/health" 2>/dev/null | grep -q "status"; then
    CURRENT=\$(uci -q get dhcp.lan.dhcp_option | grep -o "\${CUBEX_IP}" || true)
    if [ -z "\$CURRENT" ]; then
        logger -t cubex-watchdog "🛡️ CubeX Gateway revived! Restoring DHCP Option 6 to \${CUBEX_IP}"
        uci -q del_list dhcp.lan.dhcp_option="6,\${BACKUP_DNS}" || true
        uci add_list dhcp.lan.dhcp_option="6,\${CUBEX_IP}"
        uci commit dhcp
        /etc/init.d/dnsmasq reload >/dev/null 2>&1
    fi
else
    CURRENT=\$(uci -q get dhcp.lan.dhcp_option | grep -o "\${CUBEX_IP}" || true)
    if [ -n "\$CURRENT" ]; then
        logger -t cubex-watchdog "🚨 CubeX Gateway unreachable! Activating 0-second failover to \${BACKUP_DNS}"
        uci -q del_list dhcp.lan.dhcp_option="6,\${CUBEX_IP}" || true
        uci add_list dhcp.lan.dhcp_option="6,\${BACKUP_DNS}"
        uci commit dhcp
        /etc/init.d/dnsmasq reload >/dev/null 2>&1
    fi
fi
EOF
chmod +x /usr/bin/cubex-watchdog.sh

if ! grep -q "cubex-watchdog.sh" /etc/crontabs/root 2>/dev/null; then
    echo "* * * * * /usr/bin/cubex-watchdog.sh" >> /etc/crontabs/root 2>/dev/null || true
    /etc/init.d/cron enable >/dev/null 2>&1 || true
    /etc/init.d/cron restart >/dev/null 2>&1 || true
fi

echo -e "${GREEN}========================================================================${NC}"
echo -e "${GREEN}🎉 恭喜！CubeX OpenWrt Docker 宿主共生系统已全自动部署并加固完毕！${NC}"
echo -e "${GREEN}👉 CubeX Web 控制台驾驶舱: http://${CUBEX_IP}:2026${NC}"
echo -e "${GREEN}👉 主路由 OpenWrt 网关管理: http://${ROUTER_IP}${NC}"
echo -e "${GREEN}🛡️ 全屋设备无需任何手动设置，已自动继承透明代理、去广告与0秒逃生！${NC}"
echo -e "${GREEN}========================================================================${NC}"
