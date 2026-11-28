# CubeX OpenWrt / iStoreOS Docker Macvlan 部署模块 (POSIX sh)
run_platform_openwrt_docker() {
    msg "${CYAN}========================================================================${NC}"
    msg "${CYAN}🌌 正在启动 CubeX 宿主共生零感网络系统 · OpenWrt Docker 自动化编排流水线${NC}"
    msg "${CYAN}========================================================================${NC}"

    detect_target_arch
    ARCH="$TARGET_ARCH"
    msg "${GREEN}✅ [1/6 硬件识别] 检测到宿主架构: $(uname -m) ➔ 匹配运行镜像架构: ${ARCH}${NC}"

    msg "${CYAN}🔍 [2/6 Docker检测] 正在巡检 OpenWrt Docker 服务与守护进程...${NC}"
    if ! command -v docker >/dev/null 2>&1; then
        msg "${YELLOW}⚠️ 未检测到 Docker 命令行工具，正在尝试自动安装...${NC}"
        if command -v apk >/dev/null 2>&1; then
            msg "${CYAN}   -> 识别到新一代 apk 包管理器 (OpenWrt 24+ / iStoreOS 25+)，正在执行 apk 安装...${NC}"
            apk update
            apk add dockerd docker docker-compose luci-app-dockerman curl ca-certificates 2>/dev/null || apk add dockerd docker 2>/dev/null || true
        elif command -v opkg >/dev/null 2>&1; then
            msg "${CYAN}   -> 识别到 opkg 包管理器，正在执行 opkg 安装...${NC}"
            if [ -f /etc/opkg/distfeeds.conf ]; then
                sed -i 's/downloads.openwrt.org/mirrors.tuna.tsinghua.edu.cn\/openwrt/g' /etc/opkg/distfeeds.conf 2>/dev/null || true
            fi
            opkg update
            opkg install dockerd docker luci-app-dockerman curl ca-certificates ca-bundle 2>/dev/null || opkg install dockerd docker 2>/dev/null || true
        fi
    fi

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
        for _ow_i in 1 2 3 4 5 6 7 8 9 10; do
            docker info >/dev/null 2>&1 && break
            sleep 1
        done
    fi

    if ! docker info >/dev/null 2>&1; then
        msg "${RED}❌ Docker 守护进程无法自动启动！${NC}"
        msg "${RED}   请在 OpenWrt/iStoreOS 后台【Docker】或【系统服务】中手动开启 Docker 服务后重新运行本脚本。${NC}"
        exit 1
    fi
    msg "${GREEN}✅ [2/6 Docker就绪] OpenWrt Docker 守护进程运行正常${NC}"

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

    IP_PREFIX=$(printf '%s\n' "$ROUTER_IP" | cut -d. -f1-3)
    SUBNET="${IP_PREFIX}.0/24"
    DEF_ROUTE_DEV=$(ip route show default 2>/dev/null | awk '{for(i=1;i<=NF;i++) if($i=="dev") print $(i+1)}' | head -n 1)
    UPSTREAM_GW=$(ip route show default 2>/dev/null | awk '{print $3}' | head -n 1)
    UPSTREAM_PREFIX=$(printf '%s\n' "$UPSTREAM_GW" | cut -d. -f1-3)

    if [ -n "$UPSTREAM_GW" ] && [ "$UPSTREAM_PREFIX" = "$IP_PREFIX" ] && [ "$UPSTREAM_GW" != "$ROUTER_IP" ] && ! printf '%s\n' "$DEF_ROUTE_DEV" | grep -qE '^(ppp|wan)'; then
        MACVLAN_GW="$UPSTREAM_GW"
        msg "${GREEN}✅ [3/6 模式识别] 当前 OpenWrt 处于旁路由底座，上级物理主网关为: ${UPSTREAM_GW}${NC}"
    else
        MACVLAN_GW="$ROUTER_IP"
        msg "${GREEN}✅ [3/6 模式识别] 当前 OpenWrt 处于主路由底座 (WAN 出口: ${DEF_ROUTE_DEV:-pppoe-wan})，主网关为: ${ROUTER_IP}${NC}"
    fi

    NET_NAME="cubex-macvlan"
    CONTAINER_NAME="cubex"
    EXISTING_CUBEX_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$CONTAINER_NAME" 2>/dev/null || true)
    docker rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true
    if docker network inspect "$NET_NAME" >/dev/null 2>&1; then
        docker network rm "$NET_NAME" >/dev/null 2>&1 || true
    fi

    CUBEX_IP=""
    if [ -n "$EXISTING_CUBEX_IP" ] && [ "$(printf '%s\n' "$EXISTING_CUBEX_IP" | cut -d. -f1-3)" = "$IP_PREFIX" ] && [ "$EXISTING_CUBEX_IP" != "$ROUTER_IP" ] && [ "$EXISTING_CUBEX_IP" != "$UPSTREAM_GW" ]; then
        CUBEX_IP="$EXISTING_CUBEX_IP"
    else
        for _ow_cand in 11 12 13 14 15 211 212 213; do
            TEST_IP="${IP_PREFIX}.${_ow_cand}"
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

    if ! ip link show macvlan-shim >/dev/null 2>&1; then
        ip link add macvlan-shim link "$LAN_IF" type macvlan mode bridge
    fi
    bridge link set dev "$LAN_IF" hairpin on 2>/dev/null || true
    ip addr flush dev macvlan-shim 2>/dev/null || true
    ip addr add "${SHIM_IP}/24" dev macvlan-shim
    ip link set macvlan-shim up
    ip route replace "${CUBEX_IP}/32" dev macvlan-shim src "$SHIM_IP" 2>/dev/null || ip route add "${CUBEX_IP}/32" dev macvlan-shim 2>/dev/null || true

    sysctl -w net.ipv4.conf.all.rp_filter=0 >/dev/null 2>&1 || true
    sysctl -w net.ipv4.conf.default.rp_filter=0 >/dev/null 2>&1 || true
    sysctl -w "net.ipv4.conf.macvlan-shim.rp_filter=0" >/dev/null 2>&1 || true
    sysctl -w "net.ipv4.conf.${LAN_IF}.rp_filter=0" >/dev/null 2>&1 || true

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

        uci -q delete firewall.cubex_hairpin || true
        uci set firewall.cubex_hairpin=nat
        uci set firewall.cubex_hairpin.name='CubeX-Hairpin-NAT'
        uci set firewall.cubex_hairpin.src='lan'
        uci set firewall.cubex_hairpin.dest_ip="$CUBEX_IP"
        uci set firewall.cubex_hairpin.dest_port='2026'
        uci set firewall.cubex_hairpin.proto='tcp'
        uci set firewall.cubex_hairpin.target='MASQUERADE'
        uci commit firewall 2>/dev/null || true

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
        fi
        /etc/init.d/firewall reload >/dev/null 2>&1 || true
    fi

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

    LOCAL_TAR=$(ls CubeX-*-docker-*.tar.gz 2>/dev/null | head -n 1 || true)
    if [ -n "$LOCAL_TAR" ] && gzip_is_valid "$LOCAL_TAR"; then
        msg "${GREEN}📦 [5/6 离线识别] 发现当前目录离线镜像包: ${LOCAL_TAR}，正在载入本地 Docker...${NC}"
        if docker load -i "$LOCAL_TAR"; then
            TAGGED_ID=$(docker images --format '{{.Repository}}:{{.Tag}}' | grep -E '^(cubex|fyasshole/cubex):' | head -n 1)
            if [ -n "$TAGGED_ID" ] && [ "$TAGGED_ID" != "cubex:latest" ]; then
                docker tag "$TAGGED_ID" "cubex:latest" 2>/dev/null || true
            fi
        fi
    fi

    IMAGE="cubex:latest"
    if ! docker inspect "$IMAGE" >/dev/null 2>&1; then
        if docker inspect "fyasshole/cubex:latest" >/dev/null 2>&1; then
            docker tag "fyasshole/cubex:latest" "cubex:latest" 2>/dev/null || true
        elif docker inspect "ghcr.io/zyeyefyass/cubex:latest" >/dev/null 2>&1; then
            docker tag "ghcr.io/zyeyefyass/cubex:latest" "cubex:latest" 2>/dev/null || true
        else
            EXISTING_TAG=$(docker images --format '{{.Repository}}:{{.Tag}}' | grep -E '^(cubex|fyasshole/cubex|ghcr.io/zyeyefyass/cubex):' | head -n 1 || true)
            [ -n "$EXISTING_TAG" ] && docker tag "$EXISTING_TAG" "cubex:latest" 2>/dev/null || true
        fi
    fi

    if ! docker inspect "$IMAGE" >/dev/null 2>&1; then
        msg "${YELLOW}   -> 本地暂无可用镜像，正在进行多通道镜像获取 (在线加速 / 离线安装包)...${NC}"
        PULL_SUCCESS=0
        OFFLINE_TAR=$(ls /tmp/CubeX-*-docker-"${ARCH}".tar.gz /root/CubeX-*-docker-"${ARCH}".tar.gz 2>/dev/null | sort -V | tail -n1 || true)
        if [ -n "$OFFLINE_TAR" ] && gzip_is_valid "$OFFLINE_TAR"; then
            msg "${GREEN}📦 [离线兜底] 发现本地离线 Docker 归档 (${OFFLINE_TAR})，正在校验并载入...${NC}"
            if docker load -i "$OFFLINE_TAR"; then
                EXISTING_TAG=$(docker images --format '{{.Repository}}:{{.Tag}}' | grep -E '^(cubex|fyasshole/cubex|ghcr.io/zyeyefyass/cubex):' | head -n 1 || true)
                [ -n "$EXISTING_TAG" ] && docker tag "$EXISTING_TAG" "cubex:latest" 2>/dev/null || true
                docker inspect "cubex:latest" >/dev/null 2>&1 && PULL_SUCCESS=1
            fi
        fi

        if [ "$PULL_SUCCESS" = "0" ]; then
            msg "${CYAN}   -> [通道 1/2] 正在尝试直接拉取官方 Docker 镜像: fyasshole/cubex:latest ...${NC}"
            if (command -v timeout >/dev/null 2>&1 && timeout 45 docker pull "fyasshole/cubex:latest" 2>&1) || (! command -v timeout >/dev/null 2>&1 && docker pull "fyasshole/cubex:latest" 2>&1); then
                docker tag "fyasshole/cubex:latest" "cubex:latest" 2>/dev/null || true
                if docker inspect "cubex:latest" >/dev/null 2>&1; then
                    msg "${GREEN}   ✓ 官方 Docker 镜像拉取成功！${NC}"
                    PULL_SUCCESS=1
                fi
            fi
        fi

        if [ "$PULL_SUCCESS" = "0" ]; then
            msg "${YELLOW}   -> [通道 2/2] 在线 docker pull 受阻，切换为高速离线安装包下载 (CubeX-docker-${ARCH}.tar.gz)...${NC}"
            cubex_resolve_latest_tag "${CUSTOM_VER:-}"
            msg "${CYAN}   -> 匹配最新发布版本: ${REL_TAG}${NC}"
            TMP_TAR="/tmp/cubex-docker-${ARCH}.tar.gz"
            TAR_RAW_URL="https://github.com/zyeyefyass/CubeX/releases/download/${REL_TAG}/CubeX-${REL_TAG}-docker-${ARCH}.tar.gz"
            if cubex_download_with_mirrors "$TAR_RAW_URL" "$TMP_TAR" "gzip_is_valid" "360" "1"; then
                if docker load -i "$TMP_TAR"; then
                    rm -f "$TMP_TAR"
                    EXISTING_TAG=$(docker images --format '{{.Repository}}:{{.Tag}}' | grep -E '^(cubex|fyasshole/cubex|ghcr.io/zyeyefyass/cubex):' | head -n 1 || true)
                    [ -n "$EXISTING_TAG" ] && docker tag "$EXISTING_TAG" "cubex:latest" 2>/dev/null || true
                    PULL_SUCCESS=1
                fi
            fi
        fi

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

    modprobe tun 2>/dev/null || true
    if [ ! -c /dev/net/tun ]; then
        mkdir -p /dev/net 2>/dev/null || true
        mknod /dev/net/tun c 10 200 2>/dev/null || true
        chmod 666 /dev/net/tun 2>/dev/null || true
    fi
    DEV_TUN_OPT=""
    [ -c /dev/net/tun ] && DEV_TUN_OPT="--device=/dev/net/tun"

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

    sleep 2
    docker exec "$CONTAINER_NAME" ip route add "${ROUTER_IP}/32" via "$SHIM_IP" dev eth0 2>/dev/null || true

    msg "${CYAN}⚡ [6/6 一键自愈] 正在下发 OpenWrt UCI 策略路由与防火墙加固...${NC}"
    ip route replace 198.18.0.0/15 via "$CUBEX_IP" dev "$LAN_IF" 2>/dev/null || ip route replace 198.18.0.0/15 via "$CUBEX_IP" 2>/dev/null || true
    uci -q delete network.cubex_fakeip || true
    uci set network.cubex_fakeip=route
    uci set network.cubex_fakeip.interface='lan'
    uci set network.cubex_fakeip.target='198.18.0.0'
    uci set network.cubex_fakeip.netmask='255.254.0.0'
    uci set network.cubex_fakeip.gateway="$CUBEX_IP"
    uci commit network

    uci -q delete dhcp.@dnsmasq[0].rebind_domain 2>/dev/null || true
    uci add_list dhcp.@dnsmasq[0].rebind_domain='198.18.0.0/15'
    uci set dhcp.@dnsmasq[0].rebind_protection='0'
    uci -q delete dhcp.@dnsmasq[0].server 2>/dev/null || true
    uci add_list dhcp.@dnsmasq[0].server="${CUBEX_IP}"
    uci set dhcp.@dnsmasq[0].noresolv='1'
    for _ow_opt in $(uci -q get dhcp.lan.dhcp_option 2>/dev/null); do
        case "$_ow_opt" in
            6,*) uci -q del_list dhcp.lan.dhcp_option="$_ow_opt" ;;
        esac
    done
    uci add_list dhcp.lan.dhcp_option="6,${CUBEX_IP}"
    uci commit dhcp
    /etc/init.d/dnsmasq reload >/dev/null 2>&1 || /etc/init.d/dnsmasq restart >/dev/null 2>&1 || true

    uci set dhcp.lan.dns_service='0' 2>/dev/null || true
    uci -q delete dhcp.lan.dns 2>/dev/null || true
    uci set dhcp.lan.ra_dns='0' 2>/dev/null || true
    uci commit dhcp 2>/dev/null || true
    /etc/init.d/odhcpd enable >/dev/null 2>&1 || true
    /etc/init.d/odhcpd reload >/dev/null 2>&1 || /etc/init.d/odhcpd restart >/dev/null 2>&1 || true

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

    if [ "$(uci -q get firewall.@defaults[0].flow_offloading)" = "1" ] || [ "$(uci -q get firewall.@defaults[0].flow_offloading_hw)" = "1" ]; then
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
    fi
    iptables -C FORWARD -d 198.18.0.0/15 -j ACCEPT 2>/dev/null || iptables -I FORWARD 1 -d 198.18.0.0/15 -j ACCEPT 2>/dev/null || true
    iptables -C FORWARD -s 198.18.0.0/15 -j ACCEPT 2>/dev/null || iptables -I FORWARD 1 -s 198.18.0.0/15 -j ACCEPT 2>/dev/null || true
    /etc/init.d/firewall reload >/dev/null 2>&1 || true

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

    _ow_alive=0
    _ow_wait=0
    while [ "$_ow_wait" -lt 15 ]; do
        if wget -q -T 2 -O - "http://${CUBEX_IP}:2026/api/health" 2>/dev/null | grep -q "status" || \
           docker exec "${CONTAINER_NAME}" sh -c 'curl -fsS -m 2 http://127.0.0.1:2026/api/health >/dev/null 2>&1 || wget -q -T 2 -O - http://127.0.0.1:2026/api/health >/dev/null 2>&1' 2>/dev/null; then
            _ow_alive=1
            break
        fi
        sleep 1
        _ow_wait=$((_ow_wait + 1))
    done
    if [ "$_ow_alive" -ne 1 ]; then
        msg "${RED}========================================================================${NC}"
        msg "${RED}❌ [存活校验失败] 容器 ${CONTAINER_NAME} 的 2026 控制端口未就绪！${NC}"
        msg "${YELLOW}--- 最近 20 行容器日志 ---${NC}"
        docker logs --tail 20 "${CONTAINER_NAME}" 2>&1 || true
        msg "${RED}========================================================================${NC}"
        exit 1
    fi

    msg "${GREEN}========================================================================${NC}"
    msg "${GREEN}🎉 恭喜！CubeX OpenWrt Docker 宿主共生系统已全自动部署并加固完毕！${NC}"
    msg "${GREEN}👉 CubeX Web 控制台驾驶舱: http://${CUBEX_IP}:2026${NC}"
    msg "${GREEN}👉 主路由 OpenWrt 网关管理: http://${ROUTER_IP}${NC}"
    msg "${YELLOW}💡 提示: 进入 CubeX 控制台联动主路由时，【宿主/主路由 IP】填写 OpenWrt IP (${ROUTER_IP})，切勿误填 CubeX 自身 IP (${CUBEX_IP})！${NC}"
    msg "${GREEN}🛡️ 全屋设备无需任何手动设置，已自动继承透明代理、去广告与0秒逃生！${NC}"
    msg "${GREEN}========================================================================${NC}"
}
