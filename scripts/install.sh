#!/usr/bin/env bash
# ═════════════════════════════════════════════════════════════════════════════
# 🌌 CubeX Universal Deployer · 次世代双擎高可用旁路网关一键极速安装程序
# Target Version: latest
# Supported Hosts: 爱快 iKuai (KVM/Docker/主路由联动), 飞牛 fnOS, RouterOS, PVE, OpenWrt, Linux
# ═════════════════════════════════════════════════════════════════════════════
set -e

# Terminal Colors
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
PURPLE='\033[0;35m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'
BOLD='\033[1m'

clear || true
cat << "EOF"
  ██████╗██╗   ██╗██████╗ ███████╗██╗  ██╗
 ██╔════╝██║   ██║██╔══██╗██╔════╝╚██╗██╔╝
 ██║     ██║   ██║██████╔╝█████╗   ╚███╔╝ 
 ██║     ██║   ██║██╔══██╗██╔══╝   ██╔██╗ 
 ╚██████╗╚██████╔╝██████╔╝███████╗██╔╝ ██╗
  ╚═════╝ ╚═════╝ ╚═════╝ ╚══════╝╚═╝  ╚═╝
 🌌 次世代双擎高可用旁路网关系统 · 软路由的天花板
 [Version: latest | Engine: Mihomo × sing-box v1.14+]
EOF
echo ""

# ─────────────────────────────────────────────────────────────────────────────
# 1. 权限与运行环境检查
# ─────────────────────────────────────────────────────────────────────────────
if [ "$(id -u)" -ne 0 ]; then
    echo -e "${RED}❌ 错误: 此安装脚本需要 root 超级用户权限执行。${NC}"
    echo -e "${YELLOW}👉 请使用: sudo bash $0${NC}"
    exit 1
fi

# ─────────────────────────────────────────────────────────────────────────────
# 2. CPU 架构自适应探测
# ─────────────────────────────────────────────────────────────────────────────
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)
        TARGET_ARCH="amd64"
        ;;
    aarch64|arm64)
        TARGET_ARCH="arm64"
        ;;
    *)
        echo -e "${RED}❌ 不支持的 CPU 架构: $ARCH (仅支持 x86_64 / arm64)${NC}"
        exit 1
        ;;
esac
echo -e "${GREEN}🔍 检测到处理器架构: ${BOLD}${TARGET_ARCH}${NC}"
# 自动检测并安装必要依赖 (curl / unzip / ca-certificates / iptables)
if ! command -v curl >/dev/null 2>&1 || ! command -v unzip >/dev/null 2>&1 || ! command -v iptables >/dev/null 2>&1; then
    echo -e "${YELLOW}📦 检测到缺少基础工具 (curl / unzip / iptables)，正在自动安装依赖...${NC}"
    if command -v apt-get >/dev/null 2>&1; then
        apt-get update -y >/dev/null 2>&1 || true
        apt-get install -y curl ca-certificates unzip iptables iproute2 procps >/dev/null 2>&1 || true
    elif command -v apk >/dev/null 2>&1; then
        apk update >/dev/null 2>&1 || true
        apk add --no-cache curl ca-certificates unzip iptables iproute2 procps >/dev/null 2>&1 || true
    elif command -v dnf >/dev/null 2>&1; then
        dnf install -y curl ca-certificates unzip iptables iproute procps-ng >/dev/null 2>&1 || true
    elif command -v yum >/dev/null 2>&1; then
        yum install -y curl ca-certificates unzip iptables iproute procps-ng >/dev/null 2>&1 || true
    fi
fi


# ─────────────────────────────────────────────────────────────────────────────
# 3. 宿主系统全自动智能嗅探与默认安装路径匹配
# ─────────────────────────────────────────────────────────────────────────────
DETECTED_HOST="linux"
DETECTED_HOST_NAME="Linux 通用宿主环境"
DEFAULT_INSTALL_DIR="/opt/cubex"

# 3.1 飞牛 fnOS 专属环境嗅探 (严格适配 /vol1/1000/cubex 规范)
if [ -f "/etc/fnos-release" ] || [ -d "/vol1" ] || grep -qi "fnos" /etc/os-release 2>/dev/null || [ -d "/usr/trim" ]; then
    DETECTED_HOST="fnos"
    DETECTED_HOST_NAME="飞牛 fnOS 专属 NAS 环境"
    DEFAULT_INSTALL_DIR="/vol1/1000/cubex"
# 3.2 MikroTik RouterOS Container 沙盒环境嗅探
elif [ -n "$ROUTEROS" ] || [ -d "/nova" ] || grep -qi "routeros" /etc/os-release 2>/dev/null; then
    DETECTED_HOST="routeros"
    DETECTED_HOST_NAME="MikroTik RouterOS v7.x 容器沙盒"
    DEFAULT_INSTALL_DIR="/app"
# 3.3 Proxmox VE (PVE) 虚拟化环境嗅探
elif [ -d "/etc/pve" ] || command -v pveversion >/dev/null 2>&1; then
    DETECTED_HOST="pve"
    DETECTED_HOST_NAME="Proxmox VE (PVE) 宿主/LXC 环境"
    DEFAULT_INSTALL_DIR="/opt/cubex"
# 3.4 OpenWrt 软路由环境嗅探
elif [ -f "/etc/openwrt_release" ]; then
    DETECTED_HOST="openwrt"
    DETECTED_HOST_NAME="OpenWrt 软路由环境"
    DEFAULT_INSTALL_DIR="/opt/cubex"
fi

# 3.5 局域网上级主路由与宿主底座深度嗅探 (爱快 iKuai / MikroTik RouterOS)
DEFAULT_GW=$(ip route show default 2>/dev/null | awk '/default/ {print $3}' | head -n 1)
IS_IKUAI_GATEWAY=0
IS_IKUAI_HOST=0

if [ -n "$DEFAULT_GW" ]; then
    GW_PROBE=$(curl -s -m 2 -I "http://$DEFAULT_GW/" 2>/dev/null || true)
    GW_BODY=$(curl -s -m 2 "http://$DEFAULT_GW/" 2>/dev/null || true)
    if echo "$GW_PROBE $GW_BODY" | grep -qi "ikuai"; then
        IS_IKUAI_GATEWAY=1
    fi
fi

# 检查当前机器是否本身运行在爱快 KVM 虚拟机或爱快 Docker 插件容器中
if [ "$IS_IKUAI_GATEWAY" -eq 1 ]; then
    if [ -f "/.dockerenv" ] || [ -f "/run/.containerenv" ]; then
        IS_IKUAI_HOST=1
        DETECTED_HOST="ikuai_docker"
        DETECTED_HOST_NAME="爱快 iKuai Docker 插件沙盒容器"
    elif [ "$DETECTED_HOST" = "linux" ]; then
        VIRT_TYPE=""
        if command -v systemd-detect-virt >/dev/null 2>&1; then
            VIRT_TYPE=$(systemd-detect-virt 2>/dev/null || true)
        fi
        if [ "$VIRT_TYPE" = "kvm" ] || [ "$VIRT_TYPE" = "qemu" ] || grep -qi "qemu" /sys/class/dmi/id/sys_vendor 2>/dev/null; then
            IS_IKUAI_HOST=1
            DETECTED_HOST="ikuai_vm"
            DETECTED_HOST_NAME="爱快 iKuai 内置 KVM 虚拟机"
        fi
    fi
fi

echo -e "${CYAN}🖥️  检测到宿主平台: ${BOLD}${DETECTED_HOST_NAME}${NC}"
echo -e "${BLUE}📁 自动匹配推荐部署路径: ${BOLD}${DEFAULT_INSTALL_DIR}${NC}"
if [ "$IS_IKUAI_GATEWAY" -eq 1 ]; then
    echo -e "${PURPLE}🚀 嗅探到上级主路由为: ${BOLD}爱快 iKuai 企业级路由 (${DEFAULT_GW})${NC}"
    echo -e "${CYAN}💡 【爱快宿主共生建议】: 部署完成后，在 CubeX 驾驶舱【宿主共生凭据】中填入爱快 IP (${DEFAULT_GW}) 与管理密码，系统将全自动下发 198.18.0.0/15 静态回指与 DHCP 联动加固！${NC}"
fi

# ─────────────────────────────────────────────────────────────────────────────
# 4. 用户交互与非交互模式判断
# ─────────────────────────────────────────────────────────────────────────────
AUTO_CONFIRM=0
DEPLOY_MODE=""

for arg in "$@"; do
    case "$arg" in
        -y|--yes|--silent)
            AUTO_CONFIRM=1
            ;;
        --docker)
            DEPLOY_MODE="docker"
            ;;
        --native)
            DEPLOY_MODE="native"
            ;;
        --dir=*)
            DEFAULT_INSTALL_DIR="${arg#*=}"
            ;;
    esac
done

INSTALL_DIR="${DEFAULT_INSTALL_DIR}"

if [ "$AUTO_CONFIRM" -eq 0 ]; then
    if [ ! -t 0 ] && [ ! -r /dev/tty ]; then
        AUTO_CONFIRM=1
    fi
fi

if [ "$AUTO_CONFIRM" -eq 0 ]; then
    echo ""
    echo -e "${BOLD}请选择或确认安装目录:${NC} (直接回车保持默认)"
    INPUT_DIR=""
    if [ -r /dev/tty ]; then
        read -r -p "安装路径 [${DEFAULT_INSTALL_DIR}]: " INPUT_DIR </dev/tty || INPUT_DIR=""
    elif [ -t 0 ]; then
        read -r -p "安装路径 [${DEFAULT_INSTALL_DIR}]: " INPUT_DIR || INPUT_DIR=""
    fi
    if [ -n "$INPUT_DIR" ]; then
        CLEAN_DIR=$(echo "$INPUT_DIR" | tr -d '\r\n' | grep -E '^/[a-zA-Z0-9_\.\/-]+$' || true)
        if [ -n "$CLEAN_DIR" ]; then
            INSTALL_DIR="$CLEAN_DIR"
        else
            INSTALL_DIR="$DEFAULT_INSTALL_DIR"
        fi
    fi

    # 飞牛 fnOS 友好提示
    if [ "$DETECTED_HOST" = "fnos" ] && [ "$INSTALL_DIR" = "/vol1/1000/cubex" ]; then
        echo -e "${GREEN}✨ 已应用飞牛官方推荐持久化目录: /vol1/1000/cubex${NC}"
    fi

    if [ -z "$DEPLOY_MODE" ]; then
        echo ""
        echo -e "${BOLD}请选择运行部署模式:${NC}"
        echo -e "  ${CYAN}1) Docker 容器化主旁一体部署 (推荐 · 隔离性好 · 飞牛/PVE/RouterOS/爱快首选)${NC}"
        echo -e "  ${PURPLE}2) 原生单二进制 Systemd 守护部署 (纯粹裸跑 · 极致轻量低开销 · 极客首选)${NC}"
        MODE_CHOICE=""
        if [ -r /dev/tty ]; then
            read -r -p "请输入选项 [1/2, 默认 1]: " MODE_CHOICE </dev/tty || MODE_CHOICE=""
        elif [ -t 0 ]; then
            read -r -p "请输入选项 [1/2, 默认 1]: " MODE_CHOICE || MODE_CHOICE=""
        fi
        case "$MODE_CHOICE" in
            2)
                DEPLOY_MODE="native"
                ;;
            *)
                DEPLOY_MODE="docker"
                ;;
        esac
    fi
else
    # 默认自动非交互执行
    if [ -z "$DEPLOY_MODE" ]; then
        if command -v docker >/dev/null 2>&1; then
            DEPLOY_MODE="docker"
        else
            DEPLOY_MODE="native"
        fi
    fi
fi

# ─────────────────────────────────────────────────────────────────────────────
# 5. Linux 内核网络协议栈加固与爱快 / ROS / Linux 专属调优
# ─────────────────────────────────────────────────────────────────────────────
echo -e "\n${YELLOW}⚙️  [1/4] 执行宿主底层网络协议栈加固 (开启内核转发与网络缓冲区优化)...${NC}"

# 开启 IP 转发
sysctl -w net.ipv4.ip_forward=1 >/dev/null 2>&1 || true
sysctl -w net.ipv6.conf.all.forwarding=1 >/dev/null 2>&1 || true

# 高性能 TCP Socket 缓冲区与并发队列自适应调优 (Mihomo × sing-box 双引擎线速底座)
sysctl -w net.core.somaxconn=16384 >/dev/null 2>&1 || true
sysctl -w net.core.netdev_max_backlog=32768 >/dev/null 2>&1 || true
sysctl -w net.ipv4.ip_local_port_range="10240 65535" >/dev/null 2>&1 || true
sysctl -w net.ipv4.tcp_rmem="16384 1048576 33554432" >/dev/null 2>&1 || true
sysctl -w net.ipv4.tcp_wmem="16384 1048576 33554432" >/dev/null 2>&1 || true
sysctl -w net.ipv4.tcp_window_scaling=1 >/dev/null 2>&1 || true
sysctl -w net.ipv4.tcp_timestamps=1 >/dev/null 2>&1 || true
sysctl -w net.ipv4.tcp_sack=1 >/dev/null 2>&1 || true
sysctl -w net.ipv4.tcp_fastopen=3 >/dev/null 2>&1 || true
sysctl -w net.ipv4.tcp_slow_start_after_idle=0 >/dev/null 2>&1 || true
sysctl -w net.ipv4.tcp_max_syn_backlog=16384 >/dev/null 2>&1 || true
sysctl -w net.ipv4.tcp_max_tw_buckets=1440000 >/dev/null 2>&1 || true
sysctl -w net.ipv4.tcp_fin_timeout=10 >/dev/null 2>&1 || true
sysctl -w net.ipv4.tcp_tw_reuse=1 >/dev/null 2>&1 || true

# 智能探查并尝试启用 BBR 拥塞控制
if modprobe tcp_bbr >/dev/null 2>&1; then
    sysctl -w net.core.default_qdisc=fq >/dev/null 2>&1 || true
    sysctl -w net.ipv4.tcp_congestion_control=bbr >/dev/null 2>&1 || true
fi

# 🌟 爱快 iKuai 宿主机 / 主路由联动专属网络协议栈优化
if [ "$IS_IKUAI_GATEWAY" -eq 1 ] || [ "$IS_IKUAI_HOST" -eq 1 ]; then
    echo -e "${PURPLE}🚀 [爱快专属优化] 检测到爱快环境，正在注入爱快旁路四项底层加固...${NC}"
    
    # 1. 禁用反向路径过滤 (非对称旁路由放行，彻底消除爱快重定向丢包)
    sysctl -w net.ipv4.conf.all.rp_filter=0 >/dev/null 2>&1 || true
    sysctl -w net.ipv4.conf.default.rp_filter=0 >/dev/null 2>&1 || true
    for dev in /proc/sys/net/ipv4/conf/*; do
        [ -d "$dev" ] && echo 0 > "$dev/rp_filter" 2>/dev/null || true
    done

    # 2. 抑制 ARP 广播与防止 IP 漂移 (解决爱快 ARP 绑定紊乱与抢答风暴)
    sysctl -w net.ipv4.conf.all.arp_ignore=1 >/dev/null 2>&1 || true
    sysctl -w net.ipv4.conf.all.arp_announce=2 >/dev/null 2>&1 || true
    sysctl -w net.ipv4.conf.default.arp_ignore=1 >/dev/null 2>&1 || true
    sysctl -w net.ipv4.conf.default.arp_announce=2 >/dev/null 2>&1 || true

    # 3. 禁用 ICMP 重定向 (避免爱快与旁路之间产生次优路由死循环)
    sysctl -w net.ipv4.conf.all.send_redirects=0 >/dev/null 2>&1 || true
    sysctl -w net.ipv4.conf.default.send_redirects=0 >/dev/null 2>&1 || true
    sysctl -w net.ipv4.conf.all.accept_redirects=0 >/dev/null 2>&1 || true
    sysctl -w net.ipv4.conf.default.accept_redirects=0 >/dev/null 2>&1 || true

    # 4. PPPoE 广域网 TCP MSS 自动钳制 (适配爱快 1492 MTU，消除大包分片卡顿)
    if command -v iptables >/dev/null 2>&1; then
        iptables -t mangle -C POSTROUTING -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu 2>/dev/null || \
        iptables -t mangle -A POSTROUTING -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu 2>/dev/null || true
    fi

    echo -e "${GREEN}  ✓ 爱快非对称路由直通 (rp_filter=0) 已生效${NC}"
    echo -e "${GREEN}  ✓ 爱快 ARP 抑风暴与防漂移 (arp_ignore=1/arp_announce=2) 已就绪${NC}"
    echo -e "${GREEN}  ✓ 爱快 PPPoE 自动 MSS 钳制已加载 (解决网页与测速截断卡顿)${NC}"
    echo -e "${GREEN}  ✓ 爱快 ICMP 环路抑制已就绪${NC}"
fi

# 检查并自动创建 /dev/net/tun
if [ ! -c /dev/net/tun ]; then
    mkdir -p /dev/net
    mknod /dev/net/tun c 10 200 2>/dev/null || true
    chmod 666 /dev/net/tun 2>/dev/null || true
fi
modprobe tun 2>/dev/null || true

# 宿主内存自适应分级计算 (支持 512MB / 1GB / 2GB / 4GB+ 动态适配)
TOTAL_MEM_KB=$(grep MemTotal /proc/meminfo 2>/dev/null | awk '{print $2}' || echo "1048576")
if [ "$TOTAL_MEM_KB" -le 600000 ]; then
    HOST_GOMEMLIMIT="120MiB"
    HOST_GOGC="50"
elif [ "$TOTAL_MEM_KB" -le 1500000 ]; then
    HOST_GOMEMLIMIT="256MiB"
    HOST_GOGC="80"
elif [ "$TOTAL_MEM_KB" -le 3000000 ]; then
    HOST_GOMEMLIMIT="512MiB"
    HOST_GOGC="100"
else
    HOST_GOMEMLIMIT="1024MiB"
    HOST_GOGC="100"
fi

# ─────────────────────────────────────────────────────────────────────────────
# 6. 目录结构初始化与数据包释放
# ─────────────────────────────────────────────────────────────────────────────
echo -e "${YELLOW}📁 [2/4] 初始化目录与数据包释放 (${INSTALL_DIR})...${NC}"
mkdir -p "$INSTALL_DIR/bin" "$INSTALL_DIR/etc/rules" "$INSTALL_DIR/data"

# 若当前目录已有 etc 数据包，则直接同步复制
CURRENT_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -d "$CURRENT_SCRIPT_DIR/../etc" ]; then
    cp -r "$CURRENT_SCRIPT_DIR/../etc/"* "$INSTALL_DIR/etc/"
elif [ -d "$CURRENT_SCRIPT_DIR/etc" ]; then
    cp -r "$CURRENT_SCRIPT_DIR/etc/"* "$INSTALL_DIR/etc/"
elif [ -d "./etc" ]; then
    cp -r "./etc/"* "$INSTALL_DIR/etc/"
fi

# 若飞牛 fnOS，赋予 1000:1000 读写权限防止容器访问受阻
if [ "$DETECTED_HOST" = "fnos" ]; then
    chown -R 1000:1000 "$INSTALL_DIR" 2>/dev/null || true
fi

# ─────────────────────────────────────────────────────────────────────────────
# 7. 根据选择的部署模式下发配置与拉起
# ─────────────────────────────────────────────────────────────────────────────
if [ "$DEPLOY_MODE" = "docker" ]; then
    echo -e "${YELLOW}🐳 [3/4] 正在配置 Docker 容器化主旁一体网关...${NC}"
    
    # 确保 docker 已安装
    if ! command -v docker >/dev/null 2>&1; then
        echo -e "${RED}⚠️  检测到当前系统未安装 Docker，正在自动降级为原生单二进制守护部署...${NC}"
        DEPLOY_MODE="native"
    fi
fi

APP_PORT="${PORT:-${CUBEX_PORT:-2026}}"
if command -v ss >/dev/null 2>&1; then
    if ss -tuln | grep -q ":${APP_PORT} "; then
        if docker ps 2>/dev/null | grep -q "cubex" || (command -v systemctl >/dev/null 2>&1 && systemctl is-active --quiet cubexd 2>/dev/null); then
            echo -e "${CYAN}💡 检测到已有 CubeX 运行于端口 ${APP_PORT}，将执行平滑替换升级...${NC}"
        else
            echo -e "${YELLOW}⚠️ 宿主端口 ${APP_PORT} 已被其他应用占用，启动智能空闲端口探查...${NC}"
            while ss -tuln | grep -q ":${APP_PORT} "; do
                APP_PORT=$((APP_PORT + 1))
            done
            echo -e "${GREEN}✅ 自动切换至空闲可用端口: ${APP_PORT}${NC}"
        fi
    fi
fi

if [ "$DEPLOY_MODE" = "docker" ]; then
    cat <<EOF > "$INSTALL_DIR/docker-compose.yml"
services:
  cubex:
    image: docker.io/fyasshole/cubex:latest
    container_name: cubex
    restart: unless-stopped
    network_mode: host
    cap_add:
      - NET_ADMIN
      - NET_BIND_SERVICE
      - SYS_MODULE
    devices:
      - /dev/net/tun:/dev/net/tun
    volumes:
      - ./data:/app/data
      - ./etc:/app/etc
      - /etc/timezone:/etc/timezone:ro
      - /etc/localtime:/etc/localtime:ro
    environment:
      - TZ=Asia/Shanghai
      - CUBEX_PORT=${APP_PORT}
      - CUBEX_LISTEN=:${APP_PORT}
      - CUBEX_HOST_PLATFORM=${DETECTED_HOST}
      - GOMEMLIMIT=${HOST_GOMEMLIMIT}
      - GOGC=${HOST_GOGC}
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
EOF

    echo -e "${GREEN}✅ 已生成 docker-compose.yml 配置文件 (内存配额: ${HOST_GOMEMLIMIT})${NC}"
    cd "$INSTALL_DIR"
    
    if command -v docker-compose >/dev/null 2>&1; then
        docker-compose down 2>/dev/null || true
        docker-compose up -d
    elif docker compose version >/dev/null 2>&1; then
        docker compose down 2>/dev/null || true
        docker compose up -d
    else
        echo -e "${CYAN}💡 正在拉取最新版 Docker 镜像 (docker.io/fyasshole/cubex:latest)...${NC}"
        DOCKER_IMG="docker.io/fyasshole/cubex:latest"
        if ! docker pull "$DOCKER_IMG" 2>/dev/null; then
            echo -e "${YELLOW}⚠️ GHCR 镜像源拉取超时，切换至 Docker Hub: docker.io/fyasshole/cubex:latest...${NC}"
            docker pull "docker.io/fyasshole/cubex:latest" 2>/dev/null || true
            if docker inspect "docker.io/fyasshole/cubex:latest" >/dev/null 2>&1; then
                DOCKER_IMG="docker.io/fyasshole/cubex:latest"
            fi
        fi
        echo -e "${CYAN}💡 正在通过标准 docker run 命令拉起容器...${NC}"
        docker stop cubex 2>/dev/null || true
        docker rm cubex 2>/dev/null || true
        docker run -d \
          --name cubex \
          --restart unless-stopped \
          --network host \
          --cap-add NET_ADMIN \
          --cap-add NET_BIND_SERVICE \
          --device /dev/net/tun:/dev/net/tun \
          -v "$INSTALL_DIR/data:/app/data" \
          -v "$INSTALL_DIR/etc:/app/etc" \
          -e TZ=Asia/Shanghai \
          -e CUBEX_PORT=${APP_PORT} \
          -e CUBEX_LISTEN=:${APP_PORT} \
          -e CUBEX_HOST_PLATFORM=${DETECTED_HOST} \
          -e GOMEMLIMIT=${HOST_GOMEMLIMIT} \
          -e GOGC=${HOST_GOGC} \
          "$DOCKER_IMG" || true
    fi

else
        echo -e "${PURPLE}⚡ [3/4] 正在配置原生单二进制 Systemd 守护进程...${NC}"
    
    # 优先查找本地完整发行归档，或自动从 GitHub Releases 下载全量包 (包含双引擎与101套规则)
    NEED_DOWNLOAD=1
    LOCAL_ZIP=$(ls "$CURRENT_SCRIPT_DIR/../dist/release/CubeX-Test-"*"-linux-$TARGET_ARCH.zip" "$CURRENT_SCRIPT_DIR/../dist/CubeX-Test-"*"-linux-$TARGET_ARCH.zip" "./dist/release/CubeX-Test-"*"-linux-$TARGET_ARCH.zip" "./dist/CubeX-Test-"*"-linux-$TARGET_ARCH.zip" 2>/dev/null | sort -V | tail -n1 || true)
    if [ -n "$LOCAL_ZIP" ] && [ -f "$LOCAL_ZIP" ]; then
        echo -e "${GREEN}📦 发现本地发行归档包 (${LOCAL_ZIP})，直接解压部署...${NC}"
        unzip -q -o "$LOCAL_ZIP" -d "$INSTALL_DIR" 2>/dev/null || true
        NEED_DOWNLOAD=0
    elif [ -f "$CURRENT_SCRIPT_DIR/../dist/cubexd-linux-$TARGET_ARCH" ]; then
        cp "$CURRENT_SCRIPT_DIR/../dist/cubexd-linux-$TARGET_ARCH" "$INSTALL_DIR/bin/cubexd"
        NEED_DOWNLOAD=0
    fi

    if [ "$NEED_DOWNLOAD" -eq 1 ] && [ ! -f "$INSTALL_DIR/bin/cubexd" ]; then
        echo -e "${CYAN}🌐 正在从 GitHub Releases 拉取最新完整发布包 (双内核 + 101套规则集 + 控制面)...${NC}"
        REL_TAG=$(curl -4 -sSL --connect-timeout 3 -m 5 -I "https://gh-proxy.com/https://github.com/zyeyefyass/CubeX/releases/latest" 2>/dev/null | grep -i "location:" | grep -oE 'tag/[^/ \r\n]+' | cut -d/ -f2 | tr -d '\r\n')
        if [ -z "$REL_TAG" ]; then
            REL_TAG=$(curl -4 -sSL --connect-timeout 3 -m 5 -I "https://github.com/zyeyefyass/CubeX/releases/latest" 2>/dev/null | grep -i "location:" | grep -oE 'tag/[^/ \r\n]+' | cut -d/ -f2 | tr -d '\r\n')
        fi
        [ -z "$REL_TAG" ] && REL_TAG="Test-v4.4"
        ZIP_NAME="CubeX-${REL_TAG}-linux-${TARGET_ARCH}.zip"
        ZIP_URL="https://github.com/zyeyefyass/CubeX/releases/download/${REL_TAG}/${ZIP_NAME}"
        ZIP_MIRROR="https://gh-proxy.com/${ZIP_URL}"

        TMP_ZIP="/tmp/${ZIP_NAME}"
        if [ -s "$TMP_ZIP" ] && unzip -t "$TMP_ZIP" >/dev/null 2>&1; then
            echo -e "${GREEN}📦 发现本地已缓存的完整归档: ${TMP_ZIP}，秒级解压部署...${NC}"
        else
            rm -f "$TMP_ZIP"
            echo -e "${CYAN}   -> 正在通过高速镜像源下载 ${ZIP_NAME} (约 84MB，免代理极速拉取)...${NC}"
            DOWNLOAD_OK=0
            for MIRROR in "https://gh-proxy.com" "https://gh.ddlc.top" ""; do
                if [ -n "$MIRROR" ]; then
                    DL_LINK="${MIRROR}/${ZIP_URL}"
                else
                    DL_LINK="${ZIP_URL}"
                fi
                if curl -4 --http1.1 -fsSL --connect-timeout 8 -m 300 -o "$TMP_ZIP" "$DL_LINK" 2>/dev/null && [ -s "$TMP_ZIP" ] && unzip -t "$TMP_ZIP" >/dev/null 2>&1; then
                    DOWNLOAD_OK=1
                    break
                fi
                rm -f "$TMP_ZIP"
            done
        fi

        if [ -s "$TMP_ZIP" ] && unzip -t "$TMP_ZIP" >/dev/null 2>&1; then
            unzip -q -o "$TMP_ZIP" -d "$INSTALL_DIR" 2>/dev/null || true
            echo -e "${GREEN}✅ 完整归档已成功释放至 ${INSTALL_DIR} (包含 Mihomo + sing-box 双核及全量规则)${NC}"
        else
            echo -e "${YELLOW}⚠️ 完整归档拉取失败，尝试降级拉取单二进制核心控制面...${NC}"
            DL_URL="https://github.com/zyeyefyass/CubeX/releases/download/${REL_TAG}/cubexd-linux-${TARGET_ARCH}"
            DL_MIRROR="https://gh-proxy.com/${DL_URL}"
            curl -4 -sSL --connect-timeout 8 -m 120 -o "$INSTALL_DIR/bin/cubexd" "$DL_URL" 2>/dev/null || \
            curl -4 -sSL --connect-timeout 10 -m 120 -o "$INSTALL_DIR/bin/cubexd" "$DL_MIRROR" 2>/dev/null || true
        fi
    fi
    chmod +x "$INSTALL_DIR/bin/"* 2>/dev/null || true

    if [ ! -f "$INSTALL_DIR/bin/cubexd" ] || [ ! -s "$INSTALL_DIR/bin/cubexd" ]; then
        echo -e "${RED}❌ 错误: 未能获取到有效的 cubexd 二进制文件，请检查网络连接或权限后重试。${NC}"
        exit 1
    fi

    mkdir -p /usr/local/bin 2>/dev/null || true
    if [ -f "$INSTALL_DIR/bin/cubex" ]; then
        ln -sf "$INSTALL_DIR/bin/cubex" /usr/local/bin/cubex 2>/dev/null || cp "$INSTALL_DIR/bin/cubex" /usr/local/bin/cubex 2>/dev/null || true
        chmod +x /usr/local/bin/cubex 2>/dev/null || true
    fi

    # 生成 systemd 服务文件
    cat <<EOF > /etc/systemd/system/cubex.service
[Unit]
Description=CubeX Dual-Engine Cyber Gateway Daemon
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/bin/cubexd -dir $INSTALL_DIR -listen :${APP_PORT}
Restart=always
RestartSec=2s
LimitNOFILE=65535
Environment=GOMEMLIMIT=${HOST_GOMEMLIMIT}
Environment=GOGC=${HOST_GOGC}

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable cubex >/dev/null 2>&1 || true
    systemctl restart cubex || true
    echo -e "${GREEN}✅ systemd 服务已注册并启动 (/etc/systemd/system/cubex.service, 软上限: ${HOST_GOMEMLIMIT})${NC}"
fi

# ─────────────────────────────────────────────────────────────────────────────
# 8. 快捷命令软链接与部署完成驾驶舱
# ─────────────────────────────────────────────────────────────────────────────
echo -e "${YELLOW}🔗 [4/4] 注册全局管理快捷指令 (cubex)...${NC}"
mkdir -p /usr/local/bin 2>/dev/null || true

# 如果本地存在编译好的 cubex 运维 CLI，则优先链接或复制
CLI_INSTALLED=0
for cli_path in "$INSTALL_DIR/bin/cubex" "$CURRENT_SCRIPT_DIR/../dist/cubex-linux-$TARGET_ARCH" "$CURRENT_SCRIPT_DIR/dist/cubex-linux-$TARGET_ARCH" "./dist/cubex-linux-$TARGET_ARCH"; do
    if [ -f "$cli_path" ]; then
        ln -sf "$cli_path" /usr/local/bin/cubex 2>/dev/null || cp "$cli_path" /usr/local/bin/cubex 2>/dev/null || true
        chmod +x /usr/local/bin/cubex 2>/dev/null || true
        CLI_INSTALLED=1
        break
    fi
done

if [ "$CLI_INSTALLED" -eq 0 ]; then
    echo -e "${CYAN}🌐 正在尝试拉取最新版原生运维 CLI (cubex-linux-${TARGET_ARCH})...${NC}"
    CLI_URL="https://github.com/zyeyefyass/CubeX/releases/latest/download/cubex-linux-$TARGET_ARCH"
    CLI_MIRROR="https://gh-proxy.com/$CLI_URL"
    if curl -4 -sSL --connect-timeout 5 -m 30 -o /usr/local/bin/cubex "$CLI_URL" 2>/dev/null && [ -s /usr/local/bin/cubex ]; then
        chmod +x /usr/local/bin/cubex 2>/dev/null || true
        CLI_INSTALLED=1
    elif curl -4 -sSL --connect-timeout 6 -m 30 -o /usr/local/bin/cubex "$CLI_MIRROR" 2>/dev/null && [ -s /usr/local/bin/cubex ]; then
        chmod +x /usr/local/bin/cubex 2>/dev/null || true
        CLI_INSTALLED=1
    fi
fi

if [ "$CLI_INSTALLED" -eq 0 ]; then
cat << 'EOF' > /usr/local/bin/cubex
#!/usr/bin/env bash
ACTION="$1"
case "$ACTION" in
    status)
        if command -v docker >/dev/null 2>&1 && docker ps --format '{{.Names}}' | grep -q '^cubex$'; then
            docker ps --filter "name=cubex"
        elif command -v systemctl >/dev/null 2>&1; then
            systemctl status cubex --no-pager
        fi
        curl -s http://127.0.0.1:${APP_PORT}/api/v1/system/status | grep -o '"version":[^,]*' || true
        ;;
    restart)
        echo "🔄 正在重启 CubeX 守护进程..."
        if command -v docker >/dev/null 2>&1 && docker ps -a --format '{{.Names}}' | grep -q '^cubex$'; then
            docker restart cubex
        elif command -v systemctl >/dev/null 2>&1; then
            systemctl restart cubex
        fi
        echo "✅ 重启指令已下发"
        ;;
    log|logs)
        if command -v docker >/dev/null 2>&1 && docker ps -a --format '{{.Names}}' | grep -q '^cubex$'; then
            docker logs -f --tail 100 cubex
        elif command -v journalctl >/dev/null 2>&1; then
            journalctl -u cubex -f -n 100
        fi
        ;;
    reload)
        echo "🔄 热重载分流策略与规则..."
        curl -s -X POST http://127.0.0.1:${APP_PORT}/api/v1/system/reload-config
        echo ""
        ;;
    ikuai)
        echo "🌐 爱快 iKuai 宿主联动中枢"
        curl -s -X POST http://127.0.0.1:${APP_PORT}/api/v1/host/auto-setup
        echo ""
        ;;
    *)
        echo "🌌 CubeX 命令行控制中心"
        echo "用法: cubex {status|restart|log|reload|ikuai}"
        ;;
esac
EOF
chmod +x /usr/local/bin/cubex 2>/dev/null || true
fi

# 获取本机局域网 IP
LOCAL_IP=$(ip route get 1.1.1.1 2>/dev/null | grep -oP 'src \K\S+' || hostname -I 2>/dev/null | awk '{print $1}' || echo "127.0.0.1")

echo ""
echo -e "${GREEN}═════════════════════════════════════════════════════════════════════════${NC}"
echo -e "${BOLD}${GREEN}🎉 恭喜！CubeX · 次世代双擎高可用旁路网关部署成功！${NC}"
echo -e "${GREEN}═════════════════════════════════════════════════════════════════════════${NC}"
echo -e "  🌐 ${BOLD}Web 控制驾驶舱:${NC}   ${CYAN}http://${LOCAL_IP}:${APP_PORT}${NC}"
echo -e "  🖥️  ${BOLD}已识别宿主底座:${NC}   ${PURPLE}${DETECTED_HOST_NAME}${NC}"
echo -e "  📁 ${BOLD}持久化工作目录:${NC}   ${BLUE}${INSTALL_DIR}${NC}"
echo -e "  ⚡ ${BOLD}双核心代理架构:${NC}   ${GREEN}Mihomo Meta ⇄ sing-box v1.14+ (毫秒级零断网热切)${NC}"
echo -e "  🛡️  ${BOLD}权威规则数据库:${NC}   ${CYAN}217heidai MRS 广告拦截 + 隐私反追踪 + 0 明文 DNS 阻断${NC}"
echo -e "  💾 ${BOLD}宿主自适应内存:${NC}   ${YELLOW}GOMEMLIMIT=${HOST_GOMEMLIMIT} (物理内存分级防护)${NC}"
echo ""
echo -e "${BOLD}常用运维指令:${NC}"
echo -e "  • 查看运行状态:   ${YELLOW}cubex status${NC}"
echo -e "  • 追踪实时日志:   ${YELLOW}cubex log${NC}"
echo -e "  • 重启网关服务:   ${YELLOW}cubex restart${NC}"
echo -e "  • 热重载分流配置: ${YELLOW}cubex reload${NC}"
if [ "$IS_IKUAI_GATEWAY" -eq 1 ]; then
    echo ""
    echo -e "${PURPLE}═════════════════════════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}${PURPLE}🚀 【爱快 iKuai 宿主联动中枢就绪】(已嗅探到上级爱快主路由: ${DEFAULT_GW})${NC}"
    echo -e "${PURPLE}═════════════════════════════════════════════════════════════════════════${NC}"
    echo -e "  💡 爱快旁路由拓扑核心：需在爱快中将 ${BOLD}198.18.0.0/15${NC} 静态回指路由指向 CubeX (${LOCAL_IP})"
    echo -e "  🔧 ${BOLD}全自动一键配置步骤:${NC}"
    echo -e "     1. 浏览器打开: ${CYAN}http://${LOCAL_IP}:${APP_PORT}${NC}"
    echo -e "     2. 进入左侧导航【宿主共生凭据】，选择【爱快 iKuai】"
    echo -e "     3. 填入爱快地址 (${DEFAULT_GW}) 与 Web 管理密码"
    echo -e "     4. 点击【一键自愈与规则加固】，系统自动写入 198.18.0.0/15 静态回指与 DHCP 首选 DNS！"
    echo -e "  💻 也可在当前终端执行: ${CYAN}cubex ikuai${NC} 触发自动同步"
fi
echo -e "${GREEN}═════════════════════════════════════════════════════════════════════════${NC}"
