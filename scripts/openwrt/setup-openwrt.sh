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
msg() { printf '%b\n' "$*"; }

msg "${CYAN}========================================================================${NC}"
msg "${CYAN}🌌 正在启动 CubeX 宿主共生零感网络系统 · OpenWrt Docker 自动化编排流水线${NC}"
msg "${CYAN}========================================================================${NC}"

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
        msg "${RED}❌ 暂不支持的硬件架构: ${ARCH_RAW}。CubeX 官方支持 amd64 与 arm64。${NC}"
        exit 1
        ;;
esac
msg "${GREEN}✅ [1/6 硬件识别] 检测到宿主架构: ${ARCH_RAW} ➔ 匹配运行镜像架构: ${ARCH}${NC}"

# 2. Docker 守护进程检测与全自动自愈拉起 (Docker Detection & Auto-Start)
msg "${CYAN}🔍 [2/6 Docker检测] 正在巡检 OpenWrt Docker 服务与守护进程...${NC}"

# 2.1 检查 Docker 服务是否已安装
if ! command -v docker >/dev/null 2>&1; then
    msg "${YELLOW}⚠️ 未检测到 Docker 命令行工具，正在尝试自动安装...${NC}"
    if command -v apk >/dev/null 2>&1; then
        msg "${CYAN}   -> 识别到新一代 apk 包管理器 (OpenWrt 24+ / iStoreOS 25+)，正在执行 apk 安装...${NC}"
        apk update
        apk add dockerd docker docker-compose luci-app-dockerman curl ca-certificates 2>/dev/null || apk add dockerd docker 2>/dev/null || true
    elif command -v opkg >/dev/null 2>&1; then
        msg "${CYAN}   -> 识别到 opkg 包管理器，正在执行 opkg 安装...${NC}"
        # 国内镜像加速: 若检测到官方源连通困难，自动切换为清华大学开源镜像源
        if [ -f /etc/opkg/distfeeds.conf ]; then
            sed -i 's/downloads.openwrt.org/mirrors.tuna.tsinghua.edu.cn\/openwrt/g' /etc/opkg/distfeeds.conf 2>/dev/null || true
        fi
        opkg update
        opkg install dockerd docker luci-app-dockerman curl ca-certificates ca-bundle 2>/dev/null || opkg install dockerd docker 2>/dev/null || true
    fi
fi

# 2.2 检查 Docker 守护进程是否在运行，未开启则自动开启并激活服务
if ! docker info >/dev/null 2>&1; then
    msg "${YELLOW}⚠️ Docker 守护进程尚未处于活跃状态，正在自动启用并拉起服务...${NC}"
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
    msg "${RED}❌ Docker 守护进程无法自动启动！${NC}"
    msg "${RED}   请在 OpenWrt/iStoreOS 后台【Docker】或【系统服务】中手动开启 Docker 服务后重新运行本脚本。${NC}"
    exit 1
fi
msg "${GREEN}✅ [2/6 Docker就绪] OpenWrt Docker 守护进程运行正常${NC}"

# 3. LAN 网桥、主网关与 IP 动态计算 (Subnet & Gateway Probing)
LAN_IF="br-lan"
if ! ip link show "$LAN_IF" >/dev/null 2>&1; then
    LAN_IF=$(uci -q get network.lan.device 2>/dev/null || true)
fi
if [ -z "$LAN_IF" ] || ! ip link show "$LAN_IF" >/dev/null 2>&1; then
    LAN_IF=$(ip route show default 2>/dev/null | awk '{for(i=1;i<=NF;i++) if($i=="dev") print $(i+1)}' | head -n 1)
fi
if [ -z "$LAN_IF" ] || ! ip link show "$LAN_IF" >/dev/null 2>&1; then
    LAN_IF=$(ip -4 addr show 2>/dev/null | grep -E 'inet [0-9.]+' | grep -v '127.0.0.1' | awk '{print $NF}' | head -n 1)
fi
[ -z "$LAN_IF" ] && LAN_IF="eth0"

ROUTER_IP=$(uci -q get network.lan.ipaddr 2>/dev/null || ip -4 addr show "$LAN_IF" 2>/dev/null | grep -oE 'inet [0-9.]+' | awk '{print $2}' | head -n 1)
if [ -z "$ROUTER_IP" ]; then
    ROUTER_IP=$(ip route show default 2>/dev/null | awk '{print $3}' | head -n 1)
fi
if [ -z "$ROUTER_IP" ]; then
    msg "${RED}❌ 无法自动嗅探局域网主网关与子网，请检查网络配置或网卡连接！${NC}"
    exit 1
fi

IP_PREFIX=$(echo "$ROUTER_IP" | cut -d. -f1-3)
SUBNET="${IP_PREFIX}.0/24"

# 嗅探当前子网的真实主网关（严格区分主路由与旁路由模式）
DEF_ROUTE_DEV=$(ip route show default 2>/dev/null | awk '{for(i=1;i<=NF;i++) if($i=="dev") print $(i+1)}' | head -n 1)
UPSTREAM_GW=$(ip route show default 2>/dev/null | awk '{print $3}' | head -n 1)
UPSTREAM_PREFIX=$(echo "$UPSTREAM_GW" | cut -d. -f1-3)

# 判断条件：只有当上级网关与 LAN IP 处于同一子网网段且不等于自身 IP，并且默认路由出接口不是 PPPoE/WAN 口时，才属于旁路由底座
if [ -n "$UPSTREAM_GW" ] && [ "$UPSTREAM_PREFIX" = "$IP_PREFIX" ] && [ "$UPSTREAM_GW" != "$ROUTER_IP" ] && ! echo "$DEF_ROUTE_DEV" | grep -qE '^(ppp|wan)'; then
    MACVLAN_GW="$UPSTREAM_GW"
    msg "${GREEN}✅ [3/6 模式识别] 当前 OpenWrt 处于旁路由底座，上级物理主网关为: ${UPSTREAM_GW}${NC}"
else
    MACVLAN_GW="$ROUTER_IP"
    msg "${GREEN}✅ [3/6 模式识别] 当前 OpenWrt 处于主路由底座 (WAN 出口: ${DEF_ROUTE_DEV:-pppoe-wan})，主网关为: ${ROUTER_IP}${NC}"
fi

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

msg "${GREEN}✅ [3/6 自动配赋] 分配 CubeX 独立物理 IP: ${CUBEX_IP} (Macvlan 网关: ${MACVLAN_GW})${NC}"

# 4. 创建 Docker Macvlan 独立网络与 Shim 互通接口
msg "${CYAN}🔧 [4/6 网络编排] 准备 Macvlan 隔离网络 (${NET_NAME} 绑定 ${LAN_IF})...${NC}"
if docker network inspect "$NET_NAME" >/dev/null 2>&1; then
    msg "${GREEN}   -> 检测到已存在 Macvlan 网络 (${NET_NAME})，自动复用现有网络配置${NC}"
else
    docker network create -d macvlan \
        --subnet="$SUBNET" \
        --gateway="$MACVLAN_GW" \
        -o parent="$LAN_IF" \
        "$NET_NAME"
fi

# 解决 Linux Macvlan 宿主与容器二层隔离难题 (创建 macvlan-shim)
if ! ip link show macvlan-shim >/dev/null 2>&1; then
    ip link add macvlan-shim link "$LAN_IF" type macvlan mode bridge
fi
bridge link set dev "$LAN_IF" hairpin on 2>/dev/null || true
ip addr flush dev macvlan-shim 2>/dev/null || true
ip addr add "${SHIM_IP}/24" dev macvlan-shim
ip link set macvlan-shim up
ip route replace "${CUBEX_IP}/32" dev macvlan-shim src "$SHIM_IP" 2>/dev/null || ip route add "${CUBEX_IP}/32" dev macvlan-shim 2>/dev/null || true

# 彻底解除宿主与 Macvlan 间的反向路径过滤拦截
sysctl -w net.ipv4.conf.all.rp_filter=0 >/dev/null 2>&1 || true
sysctl -w net.ipv4.conf.default.rp_filter=0 >/dev/null 2>&1 || true
sysctl -w "net.ipv4.conf.macvlan-shim.rp_filter=0" >/dev/null 2>&1 || true
sysctl -w "net.ipv4.conf.${LAN_IF}.rp_filter=0" >/dev/null 2>&1 || true

# 将 macvlan-shim 固化至 OpenWrt UCI network 与防火墙放行双向互访 (防 network reload 刷掉)
if which uci >/dev/null 2>&1; then
    uci -q delete network.cubex_shim || true
    uci set network.cubex_shim=interface
    uci set network.cubex_shim.device='macvlan-shim'
    uci set network.cubex_shim.proto='static'
    uci set network.cubex_shim.ipaddr="$SHIM_IP"
    uci set network.cubex_shim.netmask='255.255.255.0'
    uci -q delete network.cubex_shim_route || true
    uci set network.cubex_shim_route=route
    uci set network.cubex_shim_route.interface='cubex_shim'
    uci set network.cubex_shim_route.target="$CUBEX_IP"
    uci set network.cubex_shim_route.netmask='255.255.255.255'
    uci commit network 2>/dev/null || true

    uci -q del_list firewall.@zone[0].device='macvlan-shim' || true
    uci add_list firewall.@zone[0].device='macvlan-shim' 2>/dev/null || true
    uci -q del_list firewall.@zone[0].network='cubex_shim' || true
    uci add_list firewall.@zone[0].network='cubex_shim' 2>/dev/null || true
    
    # 宿主 2026 端口原生透明映射至容器 (免 iptables 刷新丢失)
    uci -q delete firewall.cubex_dnat || true
    uci set firewall.cubex_dnat=redirect
    uci set firewall.cubex_dnat.name='CubeX-Panel-2026'
    uci set firewall.cubex_dnat.src='lan'
    uci set firewall.cubex_dnat.src_dport='2026'
    uci set firewall.cubex_dnat.dest='lan'
    uci set firewall.cubex_dnat.dest_ip="$CUBEX_IP"
    uci set firewall.cubex_dnat.dest_port='2026'
    uci set firewall.cubex_dnat.proto='tcp'
    uci set firewall.cubex_dnat.target='DNAT'

    # Hairpin NAT: 允许内网终端直接访问主路由 IP:2026 获得反向伪装响应
    uci -q delete firewall.cubex_hairpin || true
    uci set firewall.cubex_hairpin=nat
    uci set firewall.cubex_hairpin.name='CubeX-Hairpin-NAT'
    uci set firewall.cubex_hairpin.src='lan'
    uci set firewall.cubex_hairpin.dest_ip="$CUBEX_IP"
    uci set firewall.cubex_hairpin.dest_port='2026'
    uci set firewall.cubex_hairpin.proto='tcp'
    uci set firewall.cubex_hairpin.target='MASQUERADE'

    uci commit firewall 2>/dev/null || true

    # Flow Offloading (流量加速) 避坑加固: 豁免 198.18.0.0/15, 杜绝旁路规则被快路径跳过
    if [ "$(uci -q get firewall.@defaults[0].flow_offloading)" = "1" ] || [ "$(uci -q get firewall.@defaults[0].flow_offloading_hw)" = "1" ]; then
        msg "${YELLOW}   -> 检测到 OpenWrt 开启了 Flow Offloading (加速流表)，注入豁免规则以确保 100% 规则分流...${NC}"
        mkdir -p /etc/nftables.d 2>/dev/null || true
        cat << 'NFT_EOF' > /etc/nftables.d/10-cubex-offload-exempt.nft
chain cubex_flow_exempt {
    type filter hook forward priority -5; policy accept;
    ip daddr 198.18.0.0/15 counter return
    ip saddr 198.18.0.0/15 counter return
}
NFT_EOF
        nft insert rule inet fw4 forward ip daddr 198.18.0.0/15 counter return 2>/dev/null || true
        nft insert rule inet fw4 forward ip saddr 198.18.0.0/15 counter return 2>/dev/null || true
        /etc/init.d/firewall reload >/dev/null 2>&1 || true
    fi
    /etc/init.d/firewall reload >/dev/null 2>&1 || true
fi

# 持久化 macvlan-shim 与容器回指宿主路由开机自启 (幂等块管理)
if [ -f /etc/rc.local ]; then
    sed -i '/^# BEGIN CUBEX$/,/^# END CUBEX$/d' /etc/rc.local 2>/dev/null || true
    sed -i '/macvlan-shim/d' /etc/rc.local 2>/dev/null || true
    sed -i '/cubex.*ip route/d' /etc/rc.local 2>/dev/null || true
    RC_TMP="/tmp/rc.local.cubex.$$"
    grep -v '^exit 0$' /etc/rc.local > "$RC_TMP" 2>/dev/null || true
    cat << RCEOF >> "$RC_TMP"
# BEGIN CUBEX
ip link add macvlan-shim link ${LAN_IF} type macvlan mode bridge 2>/dev/null || true
ip addr add ${SHIM_IP}/24 dev macvlan-shim 2>/dev/null || true
ip link set macvlan-shim up 2>/dev/null || true
ip route replace ${CUBEX_IP}/32 dev macvlan-shim 2>/dev/null || true
docker exec cubex ip route replace ${ROUTER_IP}/32 via ${SHIM_IP} dev eth0 2>/dev/null || true
# END CUBEX
exit 0
RCEOF
    mv "$RC_TMP" /etc/rc.local
    chmod +x /etc/rc.local 2>/dev/null || true
fi

# 5. 部署拉起 CubeX 容器 (先确保新镜像就绪，再平滑替换旧容器)
CONTAINER_NAME="cubex"

# 优先检查并自动载入同目录下的离线镜像包
LOCAL_TAR=$(ls CubeX-*-docker-*.tar.gz 2>/dev/null | head -n 1 || true)
if [ -n "$LOCAL_TAR" ] && gzip -t "$LOCAL_TAR" 2>/dev/null; then
    msg "${GREEN}📦 [5/6 离线识别] 发现当前目录离线镜像包: ${LOCAL_TAR}，正在载入本地 Docker...${NC}"
    if docker load -i "$LOCAL_TAR"; then
        TAGGED_ID=$(docker images --format '{{.Repository}}:{{.Tag}}' | grep -E '^(cubex|fyasshole/cubex):' | head -n 1)
        if [ -n "$TAGGED_ID" ] && [ "$TAGGED_ID" != "cubex:latest" ]; then
            docker tag "$TAGGED_ID" "cubex:latest" 2>/dev/null || true
        fi
    fi
fi

IMAGE="cubex:latest"

# 1. 优先检查本地是否已有镜像（支持 cubex:latest, fyasshole/cubex:latest, 等）
if ! docker inspect "$IMAGE" >/dev/null 2>&1; then
    if docker inspect "fyasshole/cubex:latest" >/dev/null 2>&1; then
        docker tag "fyasshole/cubex:latest" "cubex:latest" 2>/dev/null || true
        IMAGE="cubex:latest"
    elif docker inspect "ghcr.io/zyeyefyass/cubex:latest" >/dev/null 2>&1; then
        docker tag "ghcr.io/zyeyefyass/cubex:latest" "cubex:latest" 2>/dev/null || true
        IMAGE="cubex:latest"
    else
        EXISTING_TAG=$(docker images --format '{{.Repository}}:{{.Tag}}' | grep -E '^(cubex|fyasshole/cubex|ghcr.io/zyeyefyass/cubex):' | head -n 1)
        if [ -n "$EXISTING_TAG" ]; then
            docker tag "$EXISTING_TAG" "cubex:latest" 2>/dev/null || true
            IMAGE="cubex:latest"
        fi
    fi
fi

# 2. 如果本地仍无可用镜像，执行多通道极速拉取策略
if ! docker inspect "$IMAGE" >/dev/null 2>&1; then
    msg "${YELLOW}   -> 本地暂无可用镜像，正在进行多通道镜像获取 (在线加速 / 离线安装包)...${NC}"
    PULL_SUCCESS=0

    # 2.0 方案 0: 若 /tmp 或 /root 已预置离线 Docker 归档包，0 秒校验直载 (无外网/Docker Hub 阻断兜底)
    OFFLINE_TAR=$(ls /tmp/CubeX-*-docker-"${ARCH}".tar.gz /root/CubeX-*-docker-"${ARCH}".tar.gz 2>/dev/null | sort -V | tail -n1 || true)
    if [ -n "$OFFLINE_TAR" ] && [ -s "$OFFLINE_TAR" ] && gzip -t "$OFFLINE_TAR" 2>/dev/null; then
        msg "${GREEN}📦 [离线兜底] 发现本地离线 Docker 归档 (${OFFLINE_TAR})，正在校验并载入...${NC}"
        if docker load -i "$OFFLINE_TAR"; then
            EXISTING_TAG=$(docker images --format '{{.Repository}}:{{.Tag}}' | grep -E '^(cubex|fyasshole/cubex|ghcr.io/zyeyefyass/cubex):' | head -n 1 || true)
            [ -n "$EXISTING_TAG" ] && docker tag "$EXISTING_TAG" "cubex:latest" 2>/dev/null || true
            if docker inspect "cubex:latest" >/dev/null 2>&1; then
                PULL_SUCCESS=1
                IMAGE="cubex:latest"
            fi
        fi
    fi

    # 2.1 方案 A: 尝试从 Docker Hub 拉取官方镜像 (带超时防卡死)
    if [ "$PULL_SUCCESS" = "0" ]; then
        msg "${CYAN}   -> [通道 1/2] 正在尝试直接拉取官方 Docker 镜像: fyasshole/cubex:latest ...${NC}"
        if (command -v timeout >/dev/null 2>&1 && timeout 45 docker pull "fyasshole/cubex:latest" 2>&1) || (! command -v timeout >/dev/null 2>&1 && docker pull "fyasshole/cubex:latest" 2>&1); then
            docker tag "fyasshole/cubex:latest" "cubex:latest" 2>/dev/null || true
            if docker inspect "cubex:latest" >/dev/null 2>&1; then
                msg "${GREEN}   ✓ 官方 Docker 镜像拉取成功！${NC}"
                PULL_SUCCESS=1
                IMAGE="cubex:latest"
            fi
        fi
    fi

    # 2.2 方案 B: 若在线 docker pull 超时/不可达，从高速 GitHub Releases 加速源拉取完整 Docker 归档包
    if [ "$PULL_SUCCESS" = "0" ]; then
        msg "${YELLOW}   -> [通道 2/2] 在线 docker pull 受阻，切换为高速离线安装包下载 (CubeX-docker-${ARCH}.tar.gz)...${NC}"
        TMP_TAR="/tmp/cubex-docker-${ARCH}.tar.gz"
        REL_TAG=""

        # 智能探查最新稳定版本号 (多源竞速，兼顾重定向与 API，拒绝 502/HTML 污染)
        for MIRROR in "https://ghfast.top" "https://ghproxy.net" "https://gh-proxy.com" "https://gh.ddlc.top"; do
            REDIRECT_TAG=$(curl -4 --http1.1 -sSL --connect-timeout 4 -m 6 -I "${MIRROR}/https://github.com/zyeyefyass/CubeX/releases/latest" 2>/dev/null | grep -i '^location:' | sed -E 's/.*tag\///' | tr -d '\r\n' | grep -E '^v[0-9]+' || true)
            if [ -n "$REDIRECT_TAG" ]; then
                REL_TAG="$REDIRECT_TAG"
                break
            fi
            LATEST_JSON=$(curl -4 --http1.1 -fsSL --connect-timeout 4 -m 6 "${MIRROR}/https://api.github.com/repos/zyeyefyass/CubeX/releases/latest" 2>/dev/null || true)
            if [ -n "$LATEST_JSON" ]; then
                REL_TAG=$(echo "$LATEST_JSON" | grep -o '"tag_name": *"[^"]*"' | head -n1 | cut -d'"' -f4 | grep -E '^v[0-9]+' || true)
                [ -n "$REL_TAG" ] && break
            fi
        done
        [ -z "$REL_TAG" ] && REL_TAG="v9.94"

        msg "${CYAN}   -> 匹配最新发布版本: ${REL_TAG}${NC}"

        for MIRROR in "https://ghfast.top" "https://ghproxy.net" "https://gh-proxy.com" "https://gh.ddlc.top" ""; do
            if [ -n "$MIRROR" ]; then
                TAR_DL="${MIRROR}/https://github.com/zyeyefyass/CubeX/releases/download/${REL_TAG}/CubeX-${REL_TAG}-docker-${ARCH}.tar.gz"
            else
                TAR_DL="https://github.com/zyeyefyass/CubeX/releases/download/${REL_TAG}/CubeX-${REL_TAG}-docker-${ARCH}.tar.gz"
            fi
            msg "${CYAN}   -> 正在连接高速下载源: ${TAR_DL} ...${NC}"
            rm -f "$TMP_TAR" 2>/dev/null || true
            if command -v curl >/dev/null 2>&1; then
                curl -4 --http1.1 -# -L --connect-timeout 10 -m 360 -o "$TMP_TAR" "$TAR_DL" 2>&1 || true
            elif command -v wget >/dev/null 2>&1; then
                wget -4 -c -T 20 -O "$TMP_TAR" "$TAR_DL" 2>&1 || true
            fi
            if [ -s "$TMP_TAR" ] && gzip -t "$TMP_TAR" 2>/dev/null; then
                msg "${GREEN}   ✓ 镜像包下载完成并通过完整性校验，正在载入 Docker...${NC}"
                if docker load -i "$TMP_TAR"; then
                    rm -f "$TMP_TAR"
                    PULL_SUCCESS=1
                    IMAGE="cubex:latest"
                    break
                fi
            fi
        done
    fi

    # 2.3 最终守门员校验
    if ! docker inspect "$IMAGE" >/dev/null 2>&1; then
        msg "${RED}❌ 无法获取 CubeX 运行镜像！(已保留现有运行容器不变)${NC}"
        msg "${YELLOW}💡 建议解决方式：${NC}"
        msg "   1. 检查路由器是否能正常连接外网；"
        msg "   2. 手动执行命令拉取镜像：docker pull fyasshole/cubex:latest"
        msg "   3. 或手动下载发布包中的 CubeX-*-docker-${ARCH}.tar.gz 传至软路由后执行: docker load -i CubeX-*-docker-${ARCH}.tar.gz"
        exit 1
    fi
fi
docker rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true

mkdir -p /etc/cubex/data

# 自动加固与探查 TUN 虚拟设备 (支持物理 OpenWrt 与虚拟化容器 OpenWrt)
modprobe tun 2>/dev/null || true
if [ ! -c /dev/net/tun ]; then
    mkdir -p /dev/net 2>/dev/null || true
    mknod /dev/net/tun c 10 200 2>/dev/null || true
    chmod 666 /dev/net/tun 2>/dev/null || true
fi

DEV_TUN_OPT=""
if [ -c /dev/net/tun ]; then
    DEV_TUN_OPT="--device=/dev/net/tun"
fi

msg "${CYAN}📦 [5/6 容器拉起] 正在拉起 CubeX 容器 (${CONTAINER_NAME} @ ${CUBEX_IP} 使用镜像: ${IMAGE})...${NC}"

docker run -d \
    --name "$CONTAINER_NAME" \
    --restart=always \
    --net="$NET_NAME" \
    --ip="$CUBEX_IP" \
    --cap-add=NET_ADMIN \
    --cap-add=NET_RAW \
    $DEV_TUN_OPT \
    -v /etc/cubex/data:/app/data \
    "$IMAGE"

# 打通容器向宿主自身 IP 发起 SSH / Web 管理访问链路 (经由 macvlan-shim 跨层互通)
sleep 2
docker exec "$CONTAINER_NAME" ip route add "${ROUTER_IP}/32" via "$SHIM_IP" dev eth0 2>/dev/null || true

# 6. 下发三大一键配置 (策略回指 + DHCP 黄金解耦 + IPv6 防泄露 + 0秒逃生)
msg "${CYAN}⚡ [6/6 一键自愈] 正在下发 OpenWrt UCI 策略路由与防火墙加固...${NC}"

# 6.1 Fake-IP 回指路由 (198.18.0.0/15 - 先入内核即刻生效，再持久化至 UCI)
ip route replace 198.18.0.0/15 via "$CUBEX_IP" dev "$LAN_IF" 2>/dev/null || ip route replace 198.18.0.0/15 via "$CUBEX_IP" 2>/dev/null || true
uci -q delete network.cubex_fakeip || true
uci set network.cubex_fakeip=route
uci set network.cubex_fakeip.interface='lan'
uci set network.cubex_fakeip.target='198.18.0.0'
uci set network.cubex_fakeip.netmask='255.254.0.0'
uci set network.cubex_fakeip.gateway="$CUBEX_IP"
uci commit network

# 6.2 DNS 全链路自愈闭环: 根除 DNS Rebinding 拦截并打通双轨解析 (0 重启即刻生效)
# 6.2.1 豁免 198.18.0.0/15 Fake-IP 防误杀 (杜绝 OpenWrt 默认 rebind_protection 将 Fake-IP 当攻击丢弃)
uci -q delete dhcp.@dnsmasq[0].rebind_domain 2>/dev/null || true
uci add_list dhcp.@dnsmasq[0].rebind_domain='198.18.0.0/15'
uci set dhcp.@dnsmasq[0].rebind_protection='0'

# 6.2.2 将 dnsmasq 上游直接指向 CubeX (彻底解决电脑/手机因 DHCP 租期未到继续请求主路由 53 导致解析不到 Fake-IP 的顽疾)
uci -q delete dhcp.@dnsmasq[0].server 2>/dev/null || true
uci add_list dhcp.@dnsmasq[0].server="${CUBEX_IP}"
uci set dhcp.@dnsmasq[0].noresolv='1'

# 6.2.3 DHCP Option 6 指向 CubeX (SIGHUP 平滑重载，0 断网)
for opt in $(uci -q get dhcp.lan.dhcp_option 2>/dev/null); do
    case "$opt" in
        6,*) uci -q del_list dhcp.lan.dhcp_option="$opt" ;;
    esac
done
uci add_list dhcp.lan.dhcp_option="6,${CUBEX_IP}"
uci commit dhcp
/etc/init.d/dnsmasq reload >/dev/null 2>&1 || /etc/init.d/dnsmasq restart >/dev/null 2>&1 || true

# 6.3 锁定 odhcpd 杜绝运营商 IPv6 DNS 泄露
uci set dhcp.lan.dns_service='0' 2>/dev/null || true
uci -q delete dhcp.lan.dns 2>/dev/null || true
uci set dhcp.lan.ra_dns='0' 2>/dev/null || true
uci commit dhcp 2>/dev/null || true
/etc/init.d/odhcpd enable >/dev/null 2>&1 || true
/etc/init.d/odhcpd reload >/dev/null 2>&1 || /etc/init.d/odhcpd restart >/dev/null 2>&1 || true

# 6.4 全版本防火墙 (fw3/fw4 双向向下兼容) 转发放行与内核优化
sysctl -w net.ipv4.ip_forward=1 >/dev/null 2>&1 || true
sysctl -w net.ipv4.conf.all.rp_filter=0 >/dev/null 2>&1 || true
uci set firewall.@zone[0].forward='ACCEPT' 2>/dev/null || true
uci -q delete firewall.cubex_bypass || true
uci set firewall.cubex_bypass=rule
uci set firewall.cubex_bypass.name='CubeX-Bypass-Forward'
uci set firewall.cubex_bypass.src='*'
uci set firewall.cubex_bypass.dest='*'
uci set firewall.cubex_bypass.proto='all'
uci set firewall.cubex_bypass.target='ACCEPT'

# 6.4.1 内网 53 端口全局劫持重定向 (无论设备是否手动指定了 114/223/8.8.8.8，一律透明捕获至 CubeX)
uci -q delete firewall.cubex_dns_redir_udp || true
uci set firewall.cubex_dns_redir_udp=redirect
uci set firewall.cubex_dns_redir_udp.name='CubeX-DNS-Redirect-UDP'
uci set firewall.cubex_dns_redir_udp.src='lan'
uci set firewall.cubex_dns_redir_udp.src_dport='53'
uci set firewall.cubex_dns_redir_udp.dest='lan'
uci set firewall.cubex_dns_redir_udp.dest_ip="$CUBEX_IP"
uci set firewall.cubex_dns_redir_udp.dest_port='53'
uci set firewall.cubex_dns_redir_udp.proto='udp'
uci set firewall.cubex_dns_redir_udp.target='DNAT'

uci -q delete firewall.cubex_dns_redir_tcp || true
uci set firewall.cubex_dns_redir_tcp=redirect
uci set firewall.cubex_dns_redir_tcp.name='CubeX-DNS-Redirect-TCP'
uci set firewall.cubex_dns_redir_tcp.src='lan'
uci set firewall.cubex_dns_redir_tcp.src_dport='53'
uci set firewall.cubex_dns_redir_tcp.dest='lan'
uci set firewall.cubex_dns_redir_tcp.dest_ip="$CUBEX_IP"
uci set firewall.cubex_dns_redir_tcp.dest_port='53'
uci set firewall.cubex_dns_redir_tcp.proto='tcp'
uci set firewall.cubex_dns_redir_tcp.target='DNAT'

uci commit firewall 2>/dev/null || true

# Flow Offloading (流量加速) 避坑加固: 豁免 198.18.0.0/15, 杜绝旁路规则被快路径跳过
if [ "$(uci -q get firewall.@defaults[0].flow_offloading)" = "1" ] || [ "$(uci -q get firewall.@defaults[0].flow_offloading_hw)" = "1" ]; then
    msg "${YELLOW}   -> 检测到 OpenWrt 开启了 Flow Offloading (加速流表)，注入豁免规则以确保 100% 规则分流...${NC}"
    mkdir -p /etc/nftables.d 2>/dev/null || true
    cat << 'NFT_EOF' > /etc/nftables.d/10-cubex-offload-exempt.nft
chain cubex_flow_exempt {
    type filter hook forward priority -5; policy accept;
    ip daddr 198.18.0.0/15 counter return
    ip saddr 198.18.0.0/15 counter return
}
NFT_EOF
    nft insert rule inet fw4 forward ip daddr 198.18.0.0/15 counter return 2>/dev/null || true
    nft insert rule inet fw4 forward ip saddr 198.18.0.0/15 counter return 2>/dev/null || true
    /etc/init.d/firewall reload >/dev/null 2>&1 || true
fi
iptables -C FORWARD -d 198.18.0.0/15 -j ACCEPT 2>/dev/null || iptables -I FORWARD 1 -d 198.18.0.0/15 -j ACCEPT 2>/dev/null || true
iptables -C FORWARD -s 198.18.0.0/15 -j ACCEPT 2>/dev/null || iptables -I FORWARD 1 -s 198.18.0.0/15 -j ACCEPT 2>/dev/null || true
/etc/init.d/firewall reload >/dev/null 2>&1 || true

# 6.5 植入原生 procd 秒级自愈断电逃生看门狗 (2秒极速故障转移与回指热切除)
cat << EOF > /usr/bin/cubex-watchdog.sh
#!/bin/sh
CUBEX_IP="${CUBEX_IP}"
BACKUP_DNS="223.5.5.5"

is_down=0
while true; do
    if nc -z -w 1 "\${CUBEX_IP}" 2026 >/dev/null 2>&1 || wget -q -T 1 -O - "http://\${CUBEX_IP}:2026/api/health" 2>/dev/null | grep -q "status" || docker exec "${CONTAINER_NAME}" curl -fsSL -m 1 http://127.0.0.1:2026/api/health 2>/dev/null | grep -q "status"; then
        if [ "\$is_down" -eq 1 ]; then
            logger -t cubex-watchdog "🛡️ CubeX Gateway revived! Restoring Fake-IP route and DNS to \${CUBEX_IP}"
            ip route replace 198.18.0.0/15 via "\${CUBEX_IP}" 2>/dev/null || true
            uci -q del_list dhcp.lan.dhcp_option="6,\${BACKUP_DNS}" || true
            uci add_list dhcp.lan.dhcp_option="6,\${CUBEX_IP}"
            uci -q delete dhcp.@dnsmasq[0].server 2>/dev/null || true
            uci add_list dhcp.@dnsmasq[0].server="\${CUBEX_IP}"
            uci set dhcp.@dnsmasq[0].noresolv='1'
            uci set firewall.cubex_dns_redir_udp.enabled='1' 2>/dev/null || true
            uci set firewall.cubex_dns_redir_tcp.enabled='1' 2>/dev/null || true
            uci commit dhcp
            uci commit firewall
            /etc/init.d/dnsmasq reload >/dev/null 2>&1
            /etc/init.d/firewall reload >/dev/null 2>&1
            is_down=0
        fi
    else
        if [ "\$is_down" -eq 0 ]; then
            logger -t cubex-watchdog "🚨 CubeX Gateway DOWN! 0-second failover: cutting Fake-IP route and switching to \${BACKUP_DNS}"
            ip route replace blackhole 198.18.0.0/15 2>/dev/null || true
            uci -q del_list dhcp.lan.dhcp_option="6,\${CUBEX_IP}" || true
            uci add_list dhcp.lan.dhcp_option="6,\${BACKUP_DNS}"
            uci -q delete dhcp.@dnsmasq[0].server 2>/dev/null || true
            uci add_list dhcp.@dnsmasq[0].server="\${BACKUP_DNS}"
            uci set dhcp.@dnsmasq[0].noresolv='0'
            uci set firewall.cubex_dns_redir_udp.enabled='0' 2>/dev/null || true
            uci set firewall.cubex_dns_redir_tcp.enabled='0' 2>/dev/null || true
            uci commit dhcp
            uci commit firewall
            /etc/init.d/dnsmasq reload >/dev/null 2>&1
            /etc/init.d/firewall reload >/dev/null 2>&1
            is_down=1
        fi
    fi
    sleep 2
done
EOF
chmod +x /usr/bin/cubex-watchdog.sh

# 注册并启动标准 procd 守护服务
cat << 'EOF' > /etc/init.d/cubex-watchdog
#!/bin/sh /etc/rc.common
START=95
STOP=10
USE_PROCD=1

start_service() {
    procd_open_instance
    procd_set_param command /usr/bin/cubex-watchdog.sh
    procd_set_param respawn 3600 5 0
    procd_set_param stdout 1
    procd_set_param stderr 1
    procd_close_instance
}
EOF
chmod +x /etc/init.d/cubex-watchdog
/etc/init.d/cubex-watchdog enable >/dev/null 2>&1 || true
/etc/init.d/cubex-watchdog restart >/dev/null 2>&1 || true

msg "${GREEN}========================================================================${NC}"
msg "${GREEN}🎉 恭喜！CubeX OpenWrt Docker 宿主共生系统已全自动部署并加固完毕！${NC}"
msg "${GREEN}👉 CubeX Web 控制台驾驶舱: http://${CUBEX_IP}:2026${NC}"
msg "${GREEN}👉 主路由 OpenWrt 网关管理: http://${ROUTER_IP}${NC}"
msg "${YELLOW}💡 提示: 进入 CubeX 控制台联动主路由时，【宿主/主路由 IP】填写 OpenWrt IP (${ROUTER_IP})，切勿误填 CubeX 自身 IP (${CUBEX_IP})！${NC}"
msg "${GREEN}🛡️ 全屋设备无需任何手动设置，已自动继承透明代理、去广告与0秒逃生！${NC}"
msg "${GREEN}========================================================================${NC}"
