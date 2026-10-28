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
    esac
done

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

# 3.2 探查可用存储池 (优先 local-lvm, local-zfs, local)
DEFAULT_STORAGE="local-lvm"
if ! pvesm status -storage local-lvm >/dev/null 2>&1; then
    if pvesm status -storage local-zfs >/dev/null 2>&1; then
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

HOST_IP=$(ip -4 addr show vmbr0 2>/dev/null | grep -oP '(?<=inet\s)\d+(\.\d+){3}' | head -n1)
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
# 智能检测：如果 stdin 为管道 (curl ... | bash) 且无法读 tty，或者指定了 -y，强制切换为非交互模式
if [ "$AUTO_CONFIRM" -eq 0 ]; then
    if [ ! -t 0 ] && [ ! -r /dev/tty ]; then
        AUTO_CONFIRM=1
    fi
fi

if [ "$AUTO_CONFIRM" -eq 0 ]; then
    echo -e "\n${YELLOW}${BOLD}⚙️ 请配置 CubeX LXC 容器基础参数 (直接回车保持默认):${NC}"
    if [ -r /dev/tty ]; then
        read -rp "容器 VMID [默认: $NEXT_ID]: " CT_ID </dev/tty || CT_ID=""
        read -rp "容器静态 IP (CIDR 格式) [默认: $DEFAULT_CT_IP]: " CT_IP </dev/tty || CT_IP=""
        read -rp "上级主路由网关 IP [默认: $GW_IP]: " CT_GW </dev/tty || CT_GW=""
        read -rp "LXC 存储池 [默认: $DEFAULT_STORAGE]: " CT_STORAGE </dev/tty || CT_STORAGE=""
        read -rp "分配内存 (MB) [推荐 256~512MB, 默认: 256]: " CT_MEM </dev/tty || CT_MEM=""
        read -rp "分配磁盘大小 (GB) [推荐 2~4GB, 默认: 2]: " CT_DISK </dev/tty || CT_DISK=""
        read -rp "容器主机名 [默认: cubex-gateway]: " CT_NAME </dev/tty || CT_NAME=""
    else
        read -rp "容器 VMID [默认: $NEXT_ID]: " CT_ID || CT_ID=""
        read -rp "容器静态 IP (CIDR 格式) [默认: $DEFAULT_CT_IP]: " CT_IP || CT_IP=""
        read -rp "上级主路由网关 IP [默认: $GW_IP]: " CT_GW || CT_GW=""
        read -rp "LXC 存储池 [默认: $DEFAULT_STORAGE]: " CT_STORAGE || CT_STORAGE=""
        read -rp "分配内存 (MB) [推荐 256~512MB, 默认: 256]: " CT_MEM || CT_MEM=""
        read -rp "分配磁盘大小 (GB) [推荐 2~4GB, 默认: 2]: " CT_DISK || CT_DISK=""
        read -rp "容器主机名 [默认: cubex-gateway]: " CT_NAME || CT_NAME=""
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

CT_CLEAN_IP=$(echo "$CT_IP" | cut -d'/' -f1)

# 5. 宿主机内核网络与穿透模块加固
echo -e "\n${CYAN}🔧 [1/5] 加固 PVE 宿主机网络与内核模块 (加载 tun / xt_TPROXY)...${NC}"
modprobe tun 2>/dev/null || true
modprobe xt_TPROXY 2>/dev/null || true
modprobe xt_socket 2>/dev/null || true
modprobe xt_mark 2>/dev/null || true
sysctl -w net.ipv4.ip_forward=1 >/dev/null 2>&1 || true

# 6. 模板探查 (优先复用本地已有模板，防止外网断流报错)
echo -e "\n${CYAN}📦 [2/5] 探查并匹配 Alpine Linux LXC 模板...${NC}"
TEMPLATE=""
# 优先检查本地已经存在的 Alpine 模板
LOCAL_TEMPLATES=$(pveam list local 2>/dev/null | grep -i 'alpine' | awk '{print $1}' | tail -n1 || true)
if [ -n "$LOCAL_TEMPLATES" ]; then
    TEMPLATE=$(basename "$LOCAL_TEMPLATES")
    echo -e "${GREEN}✓ 发现本地已缓存的 Alpine 模板: ${TEMPLATE} (免重复下载，秒级开机)${NC}"
fi

if [ -z "$TEMPLATE" ]; then
    echo -e "${BLUE}正在更新并检索官方 Alpine 模板清单...${NC}"
    pveam update >/dev/null 2>&1 || true
    TEMPLATE=$(pveam available -section system 2>/dev/null | grep -i 'alpine' | tail -n1 | awk '{print $2}' || true)
    if [ -z "$TEMPLATE" ]; then
        TEMPLATE="alpine-3.20-default_20240606_amd64.tar.xz"
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
    --net0 name=eth0,bridge=vmbr0,ip="$CT_IP",gw="$CT_GW",type=veth \
    --nameserver "$CT_GW" \
    --features nesting=1,keyctl=1 \
    --unprivileged 0 \
    --onboot 1 \
    --start 0

# 8. 注入 TUN 设备穿透 (确保容器原生支持 /dev/net/tun 与高并发网络)
echo -e "${CYAN}🔧 [4/5] 注入宿主内核 TUN 设备直通与 cgroup 白名单...${NC}"
CONF_FILE="/etc/pve/lxc/${CT_ID}.conf"
cat << 'EOF' >> "$CONF_FILE"

# === CubeX Native TUN & Acceleration Direct Pass ===
lxc.cgroup2.devices.allow: c 10:200 rwm
lxc.mount.entry: /dev/net/tun dev/net/tun none bind,create=file
EOF

# 启动容器
pct start "$CT_ID"
sleep 2

# 9. 容器内部初始化与依赖部署
echo -e "\n${CYAN}🔄 [5/5] 容器内部全自动初始化（首次部署约需 30~60 秒，请耐心等待）...${NC}"

echo -e "   -> ${BOLD}[5.1/5.4] 正在配置国内极速镜像源 (Aliyun Mirror)...${NC}"
pct exec "$CT_ID" -- sh -s << 'EOF'
set -e
cat << "REPO" > /etc/apk/repositories
https://mirrors.aliyun.com/alpine/latest-stable/main
https://mirrors.aliyun.com/alpine/latest-stable/community
REPO
EOF

echo -e "   -> ${BOLD}[5.2/5.4] 正在安装底层网络与系统依赖 (iptables / iproute2 / curl / ca-certificates)...${NC}"
pct exec "$CT_ID" -- sh -s << 'EOF'
set -e
apk update --quiet || true
apk add --no-cache curl ca-certificates iptables ip6tables iproute2 tzdata unzip
mkdir -p /opt/CubeX/bin /opt/CubeX/data /opt/CubeX/etc /opt/CubeX/rules
EOF

echo -e "   -> ${BOLD}[5.3/5.4] 正在配置 OpenRC 系统级守护进程 (含 180MB GOMEMLIMIT 防爆保护)...${NC}"
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

export GOMEMLIMIT="180MiB"
export GOGC="80"
export TZ="Asia/Shanghai"

depend() {
    need net
    after firewall
}
INIT

chmod +x /etc/init.d/cubexd
rc-update add cubexd default >/dev/null 2>&1
EOF

echo -e "   -> ${BOLD}[5.4/5.4] 正在拉取并释放 CubeX Test v4.1 双核完整发布包...${NC}"
# 10. 推送或下载 CubeX 完整发布包
LOCAL_PKG=$(ls dist/release/CubeX-Test-*-linux-amd64.zip 2>/dev/null | sort -V | tail -n1 || true)
CACHE_PKG=$(ls /tmp/CubeX-Test-*-linux-amd64.zip 2>/dev/null | sort -V | tail -n1 || true)

if [ -n "$LOCAL_PKG" ] && [ -f "$LOCAL_PKG" ]; then
    echo -e "${BLUE}检测到本地发布包 (${LOCAL_PKG})，直接推送至容器: /opt/CubeX...${NC}"
    pct push "$CT_ID" "$LOCAL_PKG" /tmp/CubeX.zip
    pct exec "$CT_ID" -- sh -s << 'EOF'
unzip -q -o /tmp/CubeX.zip -d /opt/CubeX/
chmod +x /opt/CubeX/bin/* 2>/dev/null || true
ln -sf /opt/CubeX/bin/cubex /usr/local/bin/cubex 2>/dev/null || true
rm -f /tmp/CubeX.zip
EOF
elif [ -n "$CACHE_PKG" ] && [ -s "$CACHE_PKG" ]; then
    echo -e "${BLUE}检测到宿主机缓存发布包 (${CACHE_PKG})，直接推送至容器: /opt/CubeX...${NC}"
    pct push "$CT_ID" "$CACHE_PKG" /tmp/CubeX.zip
    pct exec "$CT_ID" -- sh -s << 'EOF'
unzip -q -o /tmp/CubeX.zip -d /opt/CubeX/
chmod +x /opt/CubeX/bin/* 2>/dev/null || true
ln -sf /opt/CubeX/bin/cubex /usr/local/bin/cubex 2>/dev/null || true
rm -f /tmp/CubeX.zip
EOF
elif [ -f "dist/cubexd-linux-amd64" ]; then
    echo -e "${BLUE}检测到本地构建产物，直接推送至容器: /opt/CubeX/bin/cubexd...${NC}"
    pct push "$CT_ID" dist/cubexd-linux-amd64 /opt/CubeX/bin/cubexd -perms 755
elif [ -f "/opt/cubex/bin/cubexd" ]; then
    pct push "$CT_ID" /opt/cubex/bin/cubexd /opt/CubeX/bin/cubexd -perms 755
else
    pct exec "$CT_ID" -- sh -s << 'EOF'
    set -e
    echo "🌐 正在从 GitHub Releases 拉取最新完整发布包 (双内核 + 101套规则集 + 控制面)..."
    REL_TAG=""
    for MIRROR in "https://gh-proxy.com" "https://gh.ddlc.top" "https://mirror.ghproxy.com"; do
        LATEST_JSON=$(curl -4 --http1.1 -fsSL --connect-timeout 3 -m 5 "${MIRROR}/https://api.github.com/repos/zyeyefyass/CubeX/releases/latest" 2>/dev/null || true)
        if [ -n "$LATEST_JSON" ]; then
            REL_TAG=$(echo "$LATEST_JSON" | grep -o '"tag_name": *"[^"]*"' | head -n1 | cut -d'"' -f4)
            [ -n "$REL_TAG" ] && break
        fi
    done
    [ -z "$REL_TAG" ] && REL_TAG="Test-v4.2"

    ZIP_NAME="CubeX-${REL_TAG}-linux-amd64.zip"
    TMP_ZIP="/tmp/${ZIP_NAME}"
    rm -f "$TMP_ZIP"

    DOWNLOAD_SUCCESS=0
    DOWNLOAD_URLS="https://gh-proxy.com/https://github.com/zyeyefyass/CubeX/releases/download/${REL_TAG}/${ZIP_NAME} https://gh.ddlc.top/https://github.com/zyeyefyass/CubeX/releases/download/${REL_TAG}/${ZIP_NAME} https://github.com/zyeyefyass/CubeX/releases/download/${REL_TAG}/${ZIP_NAME}"

    for DL_URL in $DOWNLOAD_URLS; do
        echo "   -> 正在尝试拉取: ${DL_URL}..."
        if curl -4 --http1.1 -sSL --connect-timeout 8 -m 180 -o "$TMP_ZIP" "$DL_URL" 2>/dev/null && [ -s "$TMP_ZIP" ]; then
            if unzip -t "$TMP_ZIP" >/dev/null 2>&1; then
                DOWNLOAD_SUCCESS=1
                break
            else
                rm -f "$TMP_ZIP"
            fi
        fi
    done

    if [ "$DOWNLOAD_SUCCESS" = "1" ] && [ -s "$TMP_ZIP" ]; then
        unzip -q -o "$TMP_ZIP" -d /opt/CubeX/ >/dev/null 2>&1
        chmod +x /opt/CubeX/bin/* 2>/dev/null || true
        ln -sf /opt/CubeX/bin/cubex /usr/local/bin/cubex 2>/dev/null || true
        rm -f "$TMP_ZIP"
        echo "✅ 完整发布包解压完成！"
    else
        echo "⚠️ 归档下载超时，降级拉取单二进制..."
        BIN_URL="https://github.com/zyeyefyass/CubeX/releases/download/${REL_TAG}/cubexd-linux-amd64"
        BIN_MIRROR="https://gh-proxy.com/$BIN_URL"
        curl -4 --http1.1 -sSL --connect-timeout 8 -m 120 -o /opt/CubeX/bin/cubexd "$BIN_URL" 2>/dev/null || curl -4 --http1.1 -sSL --connect-timeout 10 -m 120 -o /opt/CubeX/bin/cubexd "$BIN_MIRROR" 2>/dev/null || true
        chmod 755 /opt/CubeX/bin/cubexd 2>/dev/null || true
    fi
EOF
fi

# 11. 启动服务
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
else
    echo -e "${YELLOW}⚠️ 容器已就绪，但尚未下载到 CubeX 二进制文件。可通过命令手动推送:${NC}"
    echo -e "  pct push ${CT_ID} dist/cubexd-linux-amd64 /opt/CubeX/bin/cubexd -perms 755"
    echo -e "  pct exec ${CT_ID} -- rc-service cubexd start"
fi
