#!/usr/bin/env bash
# ==============================================================================
# 🌌 CubeX - PVE LXC (CT) 容器极致轻量一键全自动极速安装程序 (实测强化版)
# 适用环境: Proxmox VE 7.x / 8.x / 9.x (x86_64)
# 特性: 
#   1. 纯原生 Alpine Linux 极致精简底座 (仅占 30~70MB 内存，启动耗时 < 1秒)
#   2. 自动检测局域网活跃 IP 并进行冲突规避 (Zero-Hardcode & Ping Check)
#   3. 优先复用 PVE 本地已有 Alpine 模板，免重复下载，秒级秒开
#   4. 宿主机自动加载 tun / xt_TPROXY 模块，自动注入 /dev/net/tun 直通
#   5. 支持全自动非交互模式 (-y / --silent / --ip=...)
#   6. 注册 OpenRC 守护进程并注入 GOMEMLIMIT 软上限，彻底防御 OOM
# ==============================================================================

set -e

CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
PURPLE='\033[0;35m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

echo -e "${CYAN}${BOLD}"
cat << 'EOF'
   ______      __        _  __
  / ____/_  __/ /_  ___ | |/ /
 / /   / / / / __ \/ _ \|   / 
/ /___/ /_/ / /_/ /  __/   |  
\____/\__,_/_.___/\___/_/|_|  
   PVE LXC 极速原生容器一键部署总线 (实测加固版)
EOF
echo -e "${NC}"

# 1. 宿主环境自检
if ! command -v pveversion >/dev/null 2>&1; then
    echo -e "${RED}[ERROR] 检测到当前系统非 Proxmox VE 宿主机！${NC}"
    echo -e "本脚本专为 PVE 物理宿主机设计，用于秒级编排高性能 Alpine LXC 旁路由沙盒。"
    exit 1
fi

PVE_VER=$(pveversion)
echo -e "${GREEN}✓ Proxmox VE 环境自检通过: ${PVE_VER}${NC}"

# 2. 命令行参数解析 (支持非交互全自动模式)
AUTO_CONFIRM=0
CUSTOM_IP=""
CUSTOM_VMID=""
CUSTOM_GW=""
CUSTOM_BRIDGE=""
CUSTOM_VER="${CUBEX_VERSION:-}"

for arg in "$@"; do
    case "$arg" in
        -y|--yes|--silent)
            AUTO_CONFIRM=1
            ;;
        --ip=*)
            CUSTOM_IP="${arg#*=}"
            ;;
        --vmid=*)
            CUSTOM_VMID="${arg#*=}"
            ;;
        --gw=*)
            CUSTOM_GW="${arg#*=}"
            ;;
        --bridge=*)
            CUSTOM_BRIDGE="${arg#*=}"
            ;;
        --version=*|-v=*)
            CUSTOM_VER="${arg#*=}"
            ;;
    esac
done

# 2.2 自动探查宿主机 CPU 架构 (严格按硬件架构匹配模板与二进制，杜绝 amd64/arm64 错位)
SYS_ARCH=$(uname -m 2>/dev/null || echo "x86_64")
case "$SYS_ARCH" in
    x86_64|amd64)
        PVE_ARCH="amd64"
        ;;
    aarch64|arm64)
        PVE_ARCH="arm64"
        ;;
    *)
        PVE_ARCH="amd64"
        ;;
esac

# 3. 宿主网络与存储池自动智能探查
# 3.1 自动寻找下一个未占用的 VMID
NEXT_ID=107
if [ -n "$CUSTOM_VMID" ]; then
    NEXT_ID="$CUSTOM_VMID"
else
    while pct status "$NEXT_ID" >/dev/null 2>&1 || [ -f "/etc/pve/lxc/${NEXT_ID}.conf" ] || [ -f "/etc/pve/qemu-server/${NEXT_ID}.conf" ]; do
        NEXT_ID=$((NEXT_ID + 1))
    done
fi

# 3.2 探查可用存储池 (智能识别支持 rootdir 容器根分区的存储池，优先 local-lvm, local-zfs, local)
DEFAULT_STORAGE=$(pvesm status -content rootdir 2>/dev/null | awk 'NR>1 {print $1}' | head -n1)
if [ -z "$DEFAULT_STORAGE" ]; then
    if pvesm status -storage local-lvm >/dev/null 2>&1; then
        DEFAULT_STORAGE="local-lvm"
    elif pvesm status -storage local-zfs >/dev/null 2>&1; then
        DEFAULT_STORAGE="local-zfs"
    else
        DEFAULT_STORAGE="local"
    fi
fi

# 3.3 探查宿主网关与局域网段 (严禁写死，全自动动态探查)
GW_IP=$(ip route | grep default | awk '{print $3}' | head -n1)
if [ -z "$GW_IP" ]; then
    GW_IP=$(ip route get 1.1.1.1 2>/dev/null | awk '{print $3}' | head -n1)
fi
if [ -n "$CUSTOM_GW" ]; then
    GW_IP="$CUSTOM_GW"
fi

if [ -n "$CUSTOM_BRIDGE" ]; then
    PVE_BRIDGE="$CUSTOM_BRIDGE"
else
    PVE_BRIDGE=$(ip route show default 2>/dev/null | awk '{print $5}' | head -n1)
fi
if [ -z "$PVE_BRIDGE" ] || ! ip link show "$PVE_BRIDGE" >/dev/null 2>&1; then
    PVE_BRIDGE="vmbr0"
fi

HOST_IP=$(ip -4 addr show "$PVE_BRIDGE" 2>/dev/null | grep -oP '(?<=inet\s)\d+(\.\d+){3}' | head -n1)
if [ -z "$HOST_IP" ]; then
    HOST_IP=$(hostname -I | awk '{print $1}')
fi

SUBNET_PREFIX=$(echo "$GW_IP" | cut -d'.' -f1-3)
if [ -z "$SUBNET_PREFIX" ] && [ -n "$HOST_IP" ]; then
    SUBNET_PREFIX=$(echo "$HOST_IP" | cut -d'.' -f1-3)
    GW_IP="${SUBNET_PREFIX}.1"
fi
if [ -z "$SUBNET_PREFIX" ]; then
    echo -e "${RED}❌ 无法自动探查 PVE 局域网段与物理网关！请通过参数手动指定: -g [网关IP] -i [容器IP/24]${NC}"
    exit 1
fi

# 3.4 动态智能选择未被占用的 IP (自动执行 ping 冲突检测)
DEFAULT_CT_IP=""
if [ -n "$CUSTOM_IP" ]; then
    DEFAULT_CT_IP="$CUSTOM_IP"
else
    # 候选 IP 列表: 7, 77, 88, 107, 207, 227
    for candidate in 7 77 88 107 207 227 17 27 37 47; do
        CHECK_IP="${SUBNET_PREFIX}.${candidate}"
        if [ "$CHECK_IP" != "$GW_IP" ] && [ "$CHECK_IP" != "$HOST_IP" ]; then
            # ping 探测是否已有设备占用
            if ! ping -c 1 -W 1 "$CHECK_IP" >/dev/null 2>&1; then
                DEFAULT_CT_IP="${CHECK_IP}/24"
                break
            fi
        fi
    done
    if [ -z "$DEFAULT_CT_IP" ]; then
        DEFAULT_CT_IP="${SUBNET_PREFIX}.77/24"
    fi
fi

# 4. 交互式确认或非交互跳过
# 智能检测：如果 stdin 为管道 (curl ... | bash) 或无控制终端 /dev/tty，或者指定了 -y，强制切换为非交互极速模式
if [ "$AUTO_CONFIRM" -eq 0 ]; then
    if [ ! -t 0 ] || ! true </dev/tty 2>/dev/null; then
        AUTO_CONFIRM=1
    fi
fi

if [ "$AUTO_CONFIRM" -eq 0 ]; then
    echo -e "\n${YELLOW}${BOLD}⚙️ 请配置 CubeX LXC 容器基础参数 (15秒超时自动保持默认，直接回车确认):${NC}"
    if [ -r /dev/tty ]; then
        read -t 15 -rp "容器 VMID [默认: $NEXT_ID]: " CT_ID </dev/tty || CT_ID=""
        read -t 15 -rp "容器静态 IP (CIDR 格式) [默认: $DEFAULT_CT_IP]: " CT_IP </dev/tty || CT_IP=""
        read -t 15 -rp "上级主路由网关 IP [默认: $GW_IP]: " CT_GW </dev/tty || CT_GW=""
        read -t 15 -rp "LXC 存储池 [默认: $DEFAULT_STORAGE]: " CT_STORAGE </dev/tty || CT_STORAGE=""
        read -t 15 -rp "分配内存 (MB) [推荐 256~512MB, 默认: 256]: " CT_MEM </dev/tty || CT_MEM=""
        read -t 15 -rp "分配磁盘大小 (GB) [推荐 2~4GB, 默认: 2]: " CT_DISK </dev/tty || CT_DISK=""
        read -t 15 -rp "容器主机名 [默认: cubex-gateway]: " CT_NAME </dev/tty || CT_NAME=""
    else
        read -t 15 -rp "容器 VMID [默认: $NEXT_ID]: " CT_ID || CT_ID=""
        read -t 15 -rp "容器静态 IP (CIDR 格式) [默认: $DEFAULT_CT_IP]: " CT_IP || CT_IP=""
        read -t 15 -rp "上级主路由网关 IP [默认: $GW_IP]: " CT_GW || CT_GW=""
        read -t 15 -rp "LXC 存储池 [默认: $DEFAULT_STORAGE]: " CT_STORAGE || CT_STORAGE=""
        read -t 15 -rp "分配内存 (MB) [推荐 256~512MB, 默认: 256]: " CT_MEM || CT_MEM=""
        read -t 15 -rp "分配磁盘大小 (GB) [推荐 2~4GB, 默认: 2]: " CT_DISK || CT_DISK=""
        read -t 15 -rp "容器主机名 [默认: cubex-gateway]: " CT_NAME || CT_NAME=""
    fi

    CT_ID=${CT_ID:-$NEXT_ID}
    CT_IP=${CT_IP:-$DEFAULT_CT_IP}
    CT_GW=${CT_GW:-$GW_IP}
    CT_STORAGE=${CT_STORAGE:-$DEFAULT_STORAGE}
    CT_MEM=${CT_MEM:-256}
    CT_DISK=${CT_DISK:-2}
    CT_NAME=${CT_NAME:-cubex-gateway}
else
    CT_ID="$NEXT_ID"
    CT_IP="$DEFAULT_CT_IP"
    CT_GW="$GW_IP"
    CT_STORAGE="$DEFAULT_STORAGE"
    CT_MEM=256
    CT_DISK=2
    CT_NAME="cubex-gateway"
    echo -e "${GREEN}✓ 已启用非交互极速模式，自动匹配参数: VMID=$CT_ID, IP=$CT_IP, GW=$CT_GW, Storage=$CT_STORAGE${NC}"
fi

# 参数防御性校验与自愈净化 (防止误贴多行代码导致 pct create 参数类型错误)
case "$CT_ID" in
    ''|*[!0-9]*) CT_ID="$NEXT_ID" ;;
esac
case "$CT_MEM" in
    ''|*[!0-9]*) CT_MEM=256 ;;
esac
case "$CT_DISK" in
    ''|*[!0-9]*) CT_DISK=2 ;;
esac
if [[ ! "$CT_IP" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+/[0-9]+$ ]]; then
    CT_IP="$DEFAULT_CT_IP"
fi
if [[ ! "$CT_GW" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    CT_GW="$GW_IP"
fi
case "$CT_NAME" in
    ''|*[!a-zA-Z0-9_-]*) CT_NAME="cubex-gateway" ;;
esac
# 验证存储池是否支持 rootdir 容器目录/镜像
if ! pvesm status -storage "$CT_STORAGE" -content rootdir >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️ 存储池 ${CT_STORAGE} 不支持容器 rootdir，自动修正为可用存储池: ${DEFAULT_STORAGE}${NC}"
    CT_STORAGE="$DEFAULT_STORAGE"
fi

# DNS 参数格式化 (PVE pct create --nameserver 仅接受单个有效 IP，避免传多个 IP 导致 400 校验阻断)
DNS_ARG=""
if [ -n "$CT_GW" ] && [[ "$CT_GW" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    DNS_ARG="--nameserver $CT_GW"
else
    DNS_ARG="--nameserver 223.5.5.5"
fi

# 检查容器 ID 冲突自愈与原地升级判定 (静态磁盘探测，严禁自动 destroy 清空老配置)
IS_UPGRADE=0
if pct status "$CT_ID" >/dev/null 2>&1 || [ -f "/etc/pve/lxc/${CT_ID}.conf" ]; then
    echo -e "${BLUE}🔍 发现已存在容器 CT ${CT_ID}，正在执行静态磁盘挂载安全探测...${NC}"
    CT_CUR_STATUS=$(pct status "$CT_ID" 2>/dev/null | awk '{print $2}')
    MOUNT_DIR=""
    MOUNT_PATH=""
    
    # 架构铁律：通过 trap 确保任何异常退出均执行 pct unmount，防止挂载残留锁死容器
    CLEANUP_LXC_MOUNT() {
        if [ -n "$MOUNT_DIR" ]; then
            pct unmount "$CT_ID" >/dev/null 2>&1 || true
            MOUNT_DIR=""
        fi
    }
    trap CLEANUP_LXC_MOUNT EXIT INT TERM
    
    CUBEX_EXISTS=0
    if [ "$CT_CUR_STATUS" = "running" ]; then
        if pct exec "$CT_ID" -- test -f /opt/cubex/bin/cubexd >/dev/null 2>&1 || \
           pct exec "$CT_ID" -- test -f /opt/CubeX/bin/cubexd >/dev/null 2>&1 || \
           pct exec "$CT_ID" -- test -d /opt/cubex/data >/dev/null 2>&1 || \
           pct exec "$CT_ID" -- test -d /opt/CubeX/data >/dev/null 2>&1; then
            CUBEX_EXISTS=1
        fi
    else
        MOUNT_RAW=$(pct mount "$CT_ID" 2>/dev/null || true)
        if [ -n "$MOUNT_RAW" ]; then
            MOUNT_DIR="1"
            MOUNT_PATH=$(printf '%s\n' "$MOUNT_RAW" | grep -oE "/[^[:space:]'\"]+" | head -n1 || true)
            [ -z "$MOUNT_PATH" ] && [ -d "/var/lib/lxc/${CT_ID}/rootfs" ] && MOUNT_PATH="/var/lib/lxc/${CT_ID}/rootfs"
        fi
        if [ -n "$MOUNT_PATH" ] && [ -d "$MOUNT_PATH" ]; then
            # 兼容 /opt/cubex 与旧版 /opt/CubeX 路径，同时探测核心二进制或用户数据目录
            if [ -f "$MOUNT_PATH/opt/cubex/bin/cubexd" ] || [ -f "$MOUNT_PATH/opt/CubeX/bin/cubexd" ] || \
               [ -d "$MOUNT_PATH/opt/cubex/data" ] || [ -d "$MOUNT_PATH/opt/CubeX/data" ]; then
                CUBEX_EXISTS=1
            fi
        fi
        if [ -n "$MOUNT_DIR" ]; then
            pct unmount "$CT_ID" >/dev/null 2>&1 || true
            MOUNT_DIR=""
        fi
    fi
    
    if [ "$CUBEX_EXISTS" -eq 1 ]; then
        echo -e "${GREEN}💡 检测到容器 CT ${CT_ID} 已运行 CubeX，进入保留用户订阅与规则的平滑升级模式...${NC}"
        IS_UPGRADE=1
        pct start "$CT_ID" >/dev/null 2>&1 || true
    else
        # 架构铁律：彻底禁止无交互自动执行 pct destroy --purge 1，防范误杀老用户容器与业务数据！
        echo -e "${RED}❌ 检测到容器 CT ${CT_ID} 已存在，但未检测到 CubeX 核心文件或配置目录！${NC}"
        echo -e "${YELLOW}🛡️ [安全保护] 为防止误删除您的关键业务数据，已阻断自动销毁。${NC}"
        echo -e "${YELLOW}如确认该容器无用且需完全清空重装，请在 PVE 宿主终端手动执行:${NC}"
        echo -e "${CYAN}   pct stop ${CT_ID} && pct destroy ${CT_ID} --purge 1${NC}"
        echo -e "${YELLOW}然后再重新运行本脚本进行全新部署。${NC}"
        exit 1
    fi
fi

CT_CLEAN_IP=$(echo "$CT_IP" | cut -d'/' -f1)

# 5. 宿主机内核网络与穿透模块加固
echo -e "\n${CYAN}🔧 [1/5] 加固 PVE 宿主机网络与内核模块 (加载 tun / xt_TPROXY)...${NC}"
modprobe tun 2>/dev/null || true
modprobe xt_TPROXY 2>/dev/null || true
modprobe xt_socket 2>/dev/null || true
modprobe xt_mark 2>/dev/null || true
sysctl -w net.ipv4.ip_forward=1 >/dev/null 2>&1 || true

if [ "$IS_UPGRADE" -eq 0 ]; then
    # 6. 模板探查 (优先复用本地已有模板，防止外网断流报错)
    echo -e "\n${CYAN}📦 [2/5] 探查并匹配 Alpine Linux LXC 模板 (${PVE_ARCH})...${NC}"
TEMPLATE=""
# 优先检查本地已经存在的对应架构 Alpine 模板 (严格过滤架构后缀，避免错用 arm64/amd64)
LOCAL_TEMPLATES=$(pveam list local 2>/dev/null | grep -i 'alpine' | grep "${PVE_ARCH}" | awk '{print $1}' | tail -n1 || true)
if [ -n "$LOCAL_TEMPLATES" ]; then
    TEMPLATE=$(basename "$LOCAL_TEMPLATES")
    echo -e "${GREEN}✓ 发现本地已缓存的 Alpine 模板: ${TEMPLATE} (免重复下载，秒级开机)${NC}"
fi

if [ -z "$TEMPLATE" ]; then
    echo -e "${BLUE}正在更新并检索官方 Alpine 模板清单 (${PVE_ARCH})...${NC}"
    pveam update >/dev/null 2>&1 || true
    TEMPLATE=$(pveam available -section system 2>/dev/null | grep -i 'alpine' | grep "${PVE_ARCH}" | tail -n1 | awk '{print $2}' || true)
    if [ -z "$TEMPLATE" ]; then
        TEMPLATE="alpine-3.20-default_20240606_${PVE_ARCH}.tar.xz"
    fi
    if ! pveam list local 2>/dev/null | grep -q "$TEMPLATE"; then
        echo -e "${BLUE}正在下载 Alpine 模板: ${TEMPLATE}...${NC}"
        pveam download local "$TEMPLATE"
    fi
fi

# 7. 创建极速优化 LXC 容器
echo -e "\n${CYAN}🚀 [3/5] 正在创建 CubeX LXC 原生容器 (CT $CT_ID)...${NC}"
pct create "$CT_ID" "local:vztmpl/$TEMPLATE" \
    --ostype alpine \
    --hostname "$CT_NAME" \
    --cores 2 \
    --memory "$CT_MEM" \
    --swap 256 \
    --rootfs "${CT_STORAGE}:${CT_DISK}" \
    --net0 name=eth0,bridge=${PVE_BRIDGE},ip="$CT_IP",gw="$CT_GW",type=veth,firewall=0 \
    $DNS_ARG \
    --features nesting=1,keyctl=1 \
    --unprivileged 0 \
    --onboot 1 \
    --start 0

# 8. 注入 TUN 设备穿透 (确保容器原生支持 /dev/net/tun 与高并发网络)
echo -e "${CYAN}🔧 [4/5] 注入宿主内核 TUN 设备直通与 cgroup 白名单...${NC}"
CONF_FILE="/etc/pve/lxc/${CT_ID}.conf"
# 防御性解除 PVE 网卡防火墙拦截 (确保 2026 控制台端口秒开无阻断)
sed -i 's/firewall=1/firewall=0/g' "$CONF_FILE" 2>/dev/null || true
pve-firewall restart 2>/dev/null || true

cat << 'EOF' >> "$CONF_FILE"

# === CubeX Native TUN & Acceleration Direct Pass ===
lxc.cgroup2.devices.allow: c 10:200 rwm
lxc.mount.entry: /dev/net/tun dev/net/tun none bind,create=file
EOF

# 启动容器
pct start "$CT_ID"

# 优雅等待网络链路就绪 (最多等待 8 秒)
echo -e "   -> ${BLUE}等待容器网卡与链路协商就绪...${NC}"
for i in 1 2 3 4 5 6 7 8; do
    if pct exec "$CT_ID" -- ping -c 1 -W 1 "$CT_GW" >/dev/null 2>&1; then
        break
    fi
    sleep 1
done

# 9. 容器内部初始化与依赖部署
echo -e "\n${CYAN}🔄 [5/5] 容器内部全自动初始化（首次部署约需 30~60 秒，请耐心等待）...${NC}"

echo -e "   -> ${BOLD}[5.1/5.4] 正在配置容器 DNS 与国内极速镜像源 (带自动故障转移与多源轮换)...${NC}"
pct exec "$CT_ID" -- sh -s "$CT_GW" << 'EOF'
set -e
CT_GW="$1"
# 优先注入网关 DNS (内网秒解)，并挂载公共 DNS 容灾兜底
if [ -n "$CT_GW" ]; then
    echo "nameserver $CT_GW" > /etc/resolv.conf
fi
echo "nameserver 223.5.5.5" >> /etc/resolv.conf
echo "nameserver 119.29.29.29" >> /etc/resolv.conf
echo "nameserver 114.114.114.114" >> /etc/resolv.conf
echo "nameserver 8.8.8.8" >> /etc/resolv.conf

# 探查真实 Alpine 发行版分支
ALPINE_VER=$(cat /etc/alpine-release 2>/dev/null | cut -d. -f1,2)
if [ -n "$ALPINE_VER" ]; then
    BRANCH="v${ALPINE_VER}"
else
    BRANCH="latest-stable"
fi

# 首选用 HTTP 协议拉取索引与 ca-certificates，彻底规避新容器无根证书时导致的 HTTPS 握手失败
cat << REPO > /etc/apk/repositories
http://mirrors.aliyun.com/alpine/${BRANCH}/main
http://mirrors.aliyun.com/alpine/${BRANCH}/community
REPO
EOF

echo -e "   -> ${BOLD}[5.2/5.4] 正在安装底层网络与系统依赖 (iptables / iproute2 / curl / ca-certificates)...${NC}"
pct exec "$CT_ID" -- sh -s << 'EOF'
set -e
ALPINE_VER=$(cat /etc/alpine-release 2>/dev/null | cut -d. -f1,2)
BRANCH="v${ALPINE_VER:-latest-stable}"

# 五大镜像源容灾轮换池 (Aliyun -> USTC -> TUNA -> Tencent -> Official)
MIRRORS="mirrors.aliyun.com mirrors.ustc.edu.cn mirrors.tuna.tsinghua.edu.cn mirrors.cloud.tencent.com dl-cdn.alpinelinux.org"

UPDATED=0
for M in $MIRRORS; do
    cat << REPO > /etc/apk/repositories
http://${M}/alpine/${BRANCH}/main
http://${M}/alpine/${BRANCH}/community
REPO
    if (command -v timeout >/dev/null 2>&1 && { timeout 10 apk update --quiet 2>/dev/null || timeout -t 10 apk update --quiet 2>/dev/null; }) || (! command -v timeout >/dev/null 2>&1 && apk update --quiet 2>/dev/null); then
        UPDATED=1
        break
    fi
    sleep 1
done

# 依赖安装自愈重试 (最多 2 次，带超时防断网宿主机长时间卡死)
INSTALLED=0
for attempt in 1 2; do
    if (command -v timeout >/dev/null 2>&1 && { timeout 25 apk add --no-cache curl ca-certificates iptables ip6tables iproute2 tzdata unzip 2>/dev/null || timeout -t 25 apk add --no-cache curl ca-certificates iptables ip6tables iproute2 tzdata unzip 2>/dev/null; }) || (! command -v timeout >/dev/null 2>&1 && apk add --no-cache curl ca-certificates iptables ip6tables iproute2 tzdata unzip 2>/dev/null); then
        INSTALLED=1
        break
    fi
    echo "正在重试依赖安装 ($attempt/2)..."
    sleep 1
done

# 安装完成 ca-certificates 后无缝升级回安全 HTTPS 镜像源
sed -i 's|http://|https://|g' /etc/apk/repositories 2>/dev/null || true

mkdir -p /opt/CubeX/bin /opt/CubeX/data /opt/CubeX/etc /opt/CubeX/rules
EOF

echo -e "   -> ${BOLD}[5.3/5.4] 正在配置 OpenRC 系统级守护进程 (含自适应 GOMEMLIMIT 防爆保护)...${NC}"
pct exec "$CT_ID" -- sh -s << 'EOF'
set -e
cat << "INIT" > /etc/init.d/cubexd
#!/sbin/openrc-run
name="cubexd"
description="CubeX System Control Plane Daemon"
command="/opt/CubeX/bin/cubexd"
command_args=""
directory="/opt/CubeX"
supervisor="supervise-daemon"
respawn_max=0
output_log="/var/log/cubexd.log"
error_log="/var/log/cubexd.err"

export GOMEMLIMIT="64MiB"
export GOGC="100"
export GODEBUG="madvdontneed=1"
export TZ="Asia/Shanghai"

depend() {
    need net
    after firewall
}
INIT

chmod 755 /etc/init.d/cubexd
# Shebang hard verification & auto-heal
if ! head -n 1 /etc/init.d/cubexd | grep -q "openrc-run"; then
    sed -i '1s|^|#!/sbin/openrc-run\n|' /etc/init.d/cubexd || true
fi
rc-update add cubexd default >/dev/null 2>&1
EOF
else
    echo -e "\n${CYAN}🔄 [3/5] 原地升级容器内二进制与规则集 (保留 /opt/CubeX/data 与 /opt/CubeX/etc 用户数据并兼容迁移旧 /opt/cubex)...${NC}"
    pct exec "$CT_ID" -- sh -s << 'MIGRATE_EOF'
mkdir -p /opt/CubeX/bin /opt/CubeX/data /opt/CubeX/etc /opt/CubeX/rules
if [ -d /opt/cubex ] && [ ! -L /opt/cubex ]; then
    for sub in data etc rules; do
        if [ -d "/opt/cubex/$sub" ]; then
            cp -rn "/opt/cubex/$sub/." "/opt/CubeX/$sub/" 2>/dev/null || cp -r "/opt/cubex/$sub/." "/opt/CubeX/$sub/" 2>/dev/null || true
        fi
    done
    mv /opt/cubex "/opt/cubex.migrated.$(date +%s)" 2>/dev/null || true
    ln -sf /opt/CubeX /opt/cubex 2>/dev/null || true
fi
if [ -f /etc/init.d/cubexd ] && grep -q "/opt/cubex/" /etc/init.d/cubexd 2>/dev/null; then
    sed -i 's|/opt/cubex/|/opt/CubeX/|g; s|directory="/opt/cubex"|directory="/opt/CubeX"|g' /etc/init.d/cubexd 2>/dev/null || true
fi
MIGRATE_EOF
fi

echo -e "   -> ${BOLD}[5.4/5.4] 正在拉取并释放 CubeX ${CUSTOM_VER:-latest} 双核完整发布包 (${PVE_ARCH})...${NC}"
# 10. 推送或下载 CubeX 完整发布包 (优先检测宿主本地安装包，0 秒直推免下载)
if [ -n "$CUSTOM_VER" ]; then
    LOCAL_PKG=$(ls dist/release/CubeX-${CUSTOM_VER}*-linux-${PVE_ARCH}.zip 2>/dev/null | sort -V | tail -n1 || true)
    CACHE_PKG=$(ls /tmp/CubeX-${CUSTOM_VER}*-linux-${PVE_ARCH}.zip 2>/dev/null | sort -V | tail -n1 || true)
    ROOT_PKG=$(ls /root/CubeX-${CUSTOM_VER}*-linux-${PVE_ARCH}.zip 2>/dev/null | sort -V | tail -n1 || true)
else
    LOCAL_PKG=$(ls dist/release/CubeX-*-linux-${PVE_ARCH}.zip 2>/dev/null | sort -V | tail -n1 || true)
    CACHE_PKG=$(ls /tmp/CubeX-*-linux-${PVE_ARCH}.zip 2>/dev/null | sort -V | tail -n1 || true)
    ROOT_PKG=$(ls /root/CubeX-*-linux-${PVE_ARCH}.zip 2>/dev/null | sort -V | tail -n1 || true)
fi

CHOSEN_ZIP=""
for cand_zip in "$LOCAL_PKG" "$CACHE_PKG" "$ROOT_PKG"; do
    if [ -n "$cand_zip" ] && [ -s "$cand_zip" ]; then
        cand_ok=0
        if command -v unzip >/dev/null 2>&1 && unzip -t "$cand_zip" >/dev/null 2>&1; then
            cand_ok=1
        elif command -v python3 >/dev/null 2>&1 && python3 -c "import sys,zipfile; z=zipfile.ZipFile(sys.argv[1]); bad=z.testzip(); sys.exit(1 if bad else 0)" "$cand_zip" 2>/dev/null; then
            cand_ok=1
        fi
        if [ "$cand_ok" -eq 1 ]; then
            CHOSEN_ZIP="$cand_zip"
            break
        else
            echo -e "${YELLOW}⚠️ 本地缓存包损坏或格式异常，已拦截拒绝使用: ${cand_zip}${NC}"
        fi
    fi
done

# 若宿主机本地无有效缓存包，直接由 PVE 宿主机从国内高速加速源拉取发布包
if [ -z "$CHOSEN_ZIP" ]; then
    echo -e "${BLUE}🌐 正在由 PVE 宿主机拉取完整发布包 (${PVE_ARCH} 双内核 + 105套规则集 + 控制面)...${NC}"
    REL_TAG="$CUSTOM_VER"
    if [ -z "$REL_TAG" ]; then
        for MIRROR in "https://ghfast.top" "https://ghproxy.net" "https://gh-proxy.com" "https://gh.ddlc.top" ""; do
            PREFIX="${MIRROR:+$MIRROR/}"
            LATEST_JSON=$(curl -4 --http1.1 -fsSL --connect-timeout 4 -m 8 "${PREFIX}https://api.github.com/repos/zyeyefyass/CubeX/releases/latest" 2>/dev/null || true)
            if [ -n "$LATEST_JSON" ]; then
                REL_TAG=$(echo "$LATEST_JSON" | grep -o '"tag_name": *"[^"]*"' | head -n1 | cut -d'"' -f4 | grep -E '^v[0-9]+' || true)
                [ -n "$REL_TAG" ] && break
            fi
            REL_TAG=$(curl -4 --http1.1 -sSL --connect-timeout 4 -m 6 -I "${PREFIX}https://github.com/zyeyefyass/CubeX/releases/latest" 2>/dev/null | grep -i "^location:" | grep -oE 'tag/v[0-9][^/ \r\n]*' | cut -d/ -f2 | tr -d '\r\n' | grep -E '^v[0-9]+' || true)
            [ -n "$REL_TAG" ] && break
        done
    fi
    [ -z "$REL_TAG" ] && REL_TAG="v9.94"

    ZIP_NAME="CubeX-${REL_TAG}-linux-${PVE_ARCH}.zip"
    TMP_ZIP="/tmp/${ZIP_NAME}"
    rm -f "$TMP_ZIP"

    DOWNLOAD_URLS="https://ghfast.top/https://github.com/zyeyefyass/CubeX/releases/download/${REL_TAG}/${ZIP_NAME} https://ghproxy.net/https://github.com/zyeyefyass/CubeX/releases/download/${REL_TAG}/${ZIP_NAME} https://gh-proxy.com/https://github.com/zyeyefyass/CubeX/releases/download/${REL_TAG}/${ZIP_NAME} https://gh.ddlc.top/https://github.com/zyeyefyass/CubeX/releases/download/${REL_TAG}/${ZIP_NAME} https://github.com/zyeyefyass/CubeX/releases/download/${REL_TAG}/${ZIP_NAME}"

    for DL_URL in $DOWNLOAD_URLS; do
        echo -e "   -> 尝试节点: ${DL_URL} ..."
        if curl -4 --http1.1 -fL --progress-bar --connect-timeout 8 -m 120 -o "$TMP_ZIP" "$DL_URL" && [ -s "$TMP_ZIP" ]; then
            ZIP_VALID=0
            if command -v unzip >/dev/null 2>&1 && unzip -t "$TMP_ZIP" >/dev/null 2>&1; then
                ZIP_VALID=1
            elif command -v python3 >/dev/null 2>&1 && python3 -c "import sys,zipfile; z=zipfile.ZipFile(sys.argv[1]); bad=z.testzip(); sys.exit(1 if bad else 0)" "$TMP_ZIP" 2>/dev/null; then
                ZIP_VALID=1
            fi
            if [ "$ZIP_VALID" -eq 1 ]; then
                CHOSEN_ZIP="$TMP_ZIP"
                break
            fi
            echo -e "${YELLOW}   ⚠️ 校验失败: ${TMP_ZIP} 压缩包损坏，自动切换下一镜像...${NC}"
        fi
        rm -f "$TMP_ZIP"
    done
fi

UNPACKED_OK=0
if [ -n "$CHOSEN_ZIP" ] && [ -s "$CHOSEN_ZIP" ]; then
    echo -e "${GREEN}✓ 宿主机完整发布包就绪 (${CHOSEN_ZIP})，执行宿主规范化解压与推入...${NC}"
    UNPACK_TMP="/tmp/cubex_unpack_$$"
    rm -rf "$UNPACK_TMP" && mkdir -p "$UNPACK_TMP"
    UNPACKED=0
    # 优先使用 Python3 进行跨平台反斜杠归一化解压 (完美解决 Windows 打包反斜杠与权限问题)
    if command -v python3 >/dev/null 2>&1; then
        if python3 -c '
import sys, zipfile, os
zpath, outdir = sys.argv[1], sys.argv[2]
with zipfile.ZipFile(zpath, "r") as z:
    for m in z.infolist():
        clean_name = m.filename.replace(chr(92), "/").strip("/")
        if not clean_name: continue
        target = os.path.join(outdir, clean_name)
        if m.is_dir() or clean_name.endswith("/"):
            os.makedirs(target, exist_ok=True)
        else:
            os.makedirs(os.path.dirname(target), exist_ok=True)
            with z.open(m) as src, open(target, "wb") as dst:
                dst.write(src.read())
            if m.external_attr > 0:
                os.chmod(target, (m.external_attr >> 16) & 0o777)
' "$CHOSEN_ZIP" "$UNPACK_TMP" 2>/dev/null; then
            UNPACKED=1
        fi
    fi
    if [ "$UNPACKED" -eq 0 ] && command -v unzip >/dev/null 2>&1; then
        if unzip -q -o "$CHOSEN_ZIP" -d "$UNPACK_TMP/"; then
            UNPACKED=1
        fi
    fi

    if [ "$UNPACKED" -eq 1 ] && [ -d "$UNPACK_TMP/bin" ]; then
        echo -e "   -> ${BLUE}正在推送双核引擎、控制面与规则集至容器内部...${NC}"
        pct exec "$CT_ID" -- mkdir -p /opt/CubeX/bin /opt/CubeX/rules /opt/CubeX/etc /opt/CubeX/data /usr/local/bin
        [ -f "$UNPACK_TMP/bin/cubexd" ] && pct push "$CT_ID" "$UNPACK_TMP/bin/cubexd" /opt/CubeX/bin/cubexd -perms 755
        [ -f "$UNPACK_TMP/bin/cubex" ] && pct push "$CT_ID" "$UNPACK_TMP/bin/cubex" /opt/CubeX/bin/cubex -perms 755
        [ -f "$UNPACK_TMP/bin/sing-box" ] && pct push "$CT_ID" "$UNPACK_TMP/bin/sing-box" /opt/CubeX/bin/sing-box -perms 755
        [ -f "$UNPACK_TMP/bin/mihomo" ] && pct push "$CT_ID" "$UNPACK_TMP/bin/mihomo" /opt/CubeX/bin/mihomo -perms 755
        if [ -d "$UNPACK_TMP/rules" ]; then
            tar -C "$UNPACK_TMP/rules" -cf - . | pct exec "$CT_ID" -- tar -xf - -C /opt/CubeX/rules/ 2>/dev/null || true
        fi
        pct exec "$CT_ID" -- ln -sf /opt/CubeX/bin/cubex /usr/local/bin/cubex 2>/dev/null || true
        rm -rf "$UNPACK_TMP"
        UNPACKED_OK=1
        echo -e "${GREEN}✓ 发布包直推完成 (免容器内部网络，零依赖解压)！${NC}"
    fi
fi

# 极端降级兜底：若宿主拉取完整包仍未成功，由宿主机独立拉取控制面与引擎核心直接推送
if [ "$UNPACKED_OK" -eq 0 ]; then
    echo -e "${YELLOW}⚠️ 完整包拉取受阻，启用宿主机独立拉取控制面与内核降级策略...${NC}"
    pct exec "$CT_ID" -- mkdir -p /opt/CubeX/bin /opt/CubeX/rules /opt/CubeX/etc /opt/CubeX/data /usr/local/bin
    REL_TAG="${CUSTOM_VER:-${REL_TAG:-v9.94}}"
    
    # 1. cubexd 控制面
    TMP_BIN="/tmp/cubexd_${PVE_ARCH}_$$"
    for MIRROR in "https://ghfast.top" "https://ghproxy.net" "https://gh-proxy.com" "https://gh.ddlc.top" ""; do
        PREFIX="${MIRROR:+$MIRROR/}"
        BIN_URL="${PREFIX}https://github.com/zyeyefyass/CubeX/releases/download/${REL_TAG}/cubexd-linux-${PVE_ARCH}"
        if curl -4 --http1.1 -fsSL --connect-timeout 8 -m 60 -o "$TMP_BIN" "$BIN_URL" 2>/dev/null && [ -s "$TMP_BIN" ] && head -c 4 "$TMP_BIN" 2>/dev/null | grep -q "ELF"; then
            pct push "$CT_ID" "$TMP_BIN" /opt/CubeX/bin/cubexd -perms 755
            rm -f "$TMP_BIN"
            break
        fi
    done
    rm -f "$TMP_BIN"

    # 2. sing-box 核心
    TMP_SB="/tmp/sb_${PVE_ARCH}_$$"
    for MIRROR in "https://ghfast.top" "https://ghproxy.net" "https://gh-proxy.com" "https://gh.ddlc.top" ""; do
        PREFIX="${MIRROR:+$MIRROR/}"
        SB_URL="${PREFIX}https://github.com/zyeyefyass/CubeX/releases/download/${REL_TAG}/sing-box-linux-${PVE_ARCH}"
        if curl -4 --http1.1 -fsSL --connect-timeout 8 -m 60 -o "$TMP_SB" "$SB_URL" 2>/dev/null && [ -s "$TMP_SB" ] && head -c 4 "$TMP_SB" 2>/dev/null | grep -q "ELF"; then
            pct push "$CT_ID" "$TMP_SB" /opt/CubeX/bin/sing-box -perms 755
            rm -f "$TMP_SB"
            break
        fi
    done
    rm -f "$TMP_SB"
fi

# 增量升级校验：仅当未解压完整发布包时，允许从本地构建目录注入独立控制面
if [ "$UNPACKED_OK" -eq 0 ]; then
    if [ -f "dist/cubexd-linux-${PVE_ARCH}" ]; then
        echo -e "${BLUE}注入本地最新构建的控制面: dist/cubexd-linux-${PVE_ARCH}...${NC}"
        pct push "$CT_ID" "dist/cubexd-linux-${PVE_ARCH}" /opt/CubeX/bin/cubexd -perms 755
    elif [ -f "/opt/cubex/bin/cubexd" ]; then
        pct push "$CT_ID" /opt/cubex/bin/cubexd /opt/CubeX/bin/cubexd -perms 755
    fi
fi

pct exec "$CT_ID" -- sh -s << 'EOF'
if [ ! -f /opt/CubeX/bin/cubexd ] || [ ! -s /opt/CubeX/bin/cubexd ]; then
    echo "❌ 致命错误: 未能获取到有效的 cubexd 控制面二进制文件，请检查网络后重试！"
    exit 1
fi
if [ ! -f /opt/CubeX/bin/sing-box ] && [ ! -f /opt/CubeX/bin/mihomo ]; then
    echo "❌ 致命错误: 双引擎核心文件 (sing-box / mihomo) 均缺失！"
    echo "👉 严重警告: 控制面在没有代理内核的情况下无法转发任何网络流量！请确保网络正常或放置发布包至宿主机 /tmp/ 后重试！"
    exit 1
fi
EOF

# 11. 守护进程就绪与服务启动
pct exec "$CT_ID" -- sh -c '
chmod 755 /opt/CubeX/bin/* 2>/dev/null || true
if [ ! -f /etc/init.d/cubexd ] || ! head -n 1 /etc/init.d/cubexd | grep -q "openrc-run"; then
    sed -i "1s|^|#!/sbin/openrc-run\n|" /etc/init.d/cubexd 2>/dev/null || true
    chmod 755 /etc/init.d/cubexd 2>/dev/null || true
fi
rc-update add cubexd default >/dev/null 2>&1 || true
'
echo -e "${CYAN}🚀 正在启动 CubeX 守护进程...${NC}"
pct exec "$CT_ID" -- rc-service cubexd restart >/dev/null 2>&1 || true
sleep 2

# 12. 运行状态校验
if pct exec "$CT_ID" -- pgrep -f cubexd >/dev/null 2>&1; then
    # 注册 PVE 宿主快捷指令 (一键穿透直达容器)
    cat << PVECLI > /usr/local/bin/cubex 2>/dev/null || true
#!/bin/sh
pct exec "$CT_ID" -- /opt/CubeX/bin/cubex "\$@"
PVECLI
    chmod +x /usr/local/bin/cubex 2>/dev/null || true
    if [ "$IS_UPGRADE" -eq 1 ]; then
        echo -e "\n${GREEN}${BOLD}═════════════════════════════════════════════════════════════════${NC}"
        echo -e "${GREEN}${BOLD}🎉 恭喜！CT ${CT_ID} 原地升级完成！现有订阅、节点与规则配置 100% 完整保留！${NC}"
        echo -e "${GREEN}${BOLD}═════════════════════════════════════════════════════════════════${NC}"
        echo -e "  📌 容器 ID     : ${BOLD}${CT_ID} (${CT_NAME})${NC}"
        echo -e "  🌐 旁路由 IP   : ${BOLD}${CT_CLEAN_IP}${NC}"
        echo -e "  🖥️ Web 控制台  : ${CYAN}${BOLD}http://${CT_CLEAN_IP}:2026${NC}"
        echo -e "${GREEN}${BOLD}═════════════════════════════════════════════════════════════════${NC}\n"
    else
        echo -e "\n${GREEN}${BOLD}═════════════════════════════════════════════════════════════════${NC}"
        echo -e "${GREEN}${BOLD}🎉 恭喜！CubeX 宿主共生透明代理网关已在 PVE LXC 容器成功部署运行！${NC}"
        echo -e "${GREEN}${BOLD}═════════════════════════════════════════════════════════════════${NC}"
        echo -e "  📌 容器 ID     : ${BOLD}${CT_ID} (${CT_NAME})${NC}"
        echo -e "  🌐 旁路由 IP   : ${BOLD}${CT_CLEAN_IP}${NC}"
        echo -e "  🖥️ Web 控制台  : ${CYAN}${BOLD}http://${CT_CLEAN_IP}:2026${NC}"
        echo -e "  ⚡ 底座优势    : 原生共享宿主物理内核，内存仅需 70MB，启动 < 1 秒"
        echo -e "  🛡️ 访问安全    : 开箱即用 · 局域网直连免密 (亦可在控制台设置访问密钥)"
        echo -e "\n${YELLOW}💡 下一步联动指引:${NC}"
        echo -e "  1. 打开浏览器直接访问 Web 控制台: ${CYAN}http://${CT_CLEAN_IP}:2026${NC}"
        echo -e "  2. 进入【⚙️ 宿主共生与系统运维】，输入主路由（如 RouterOS / 爱快）的 IP 与凭据"
        echo -e "  3. 点击【⚡ 一键全栈自愈】，CubeX 将自动完成策略路由表创建、Fake-IP 回指与 0 秒逃生编排！"
        echo -e "${GREEN}${BOLD}═════════════════════════════════════════════════════════════════${NC}\n"
    fi
else
    echo -e "${YELLOW}⚠️ 容器已就绪，但尚未下载到 CubeX 二进制文件。可通过命令手动推送:${NC}"
    echo -e "  pct push ${CT_ID} dist/cubexd-linux-amd64 /opt/CubeX/bin/cubexd -perms 755"
    echo -e "  pct exec ${CT_ID} -- rc-service cubexd start"
fi
