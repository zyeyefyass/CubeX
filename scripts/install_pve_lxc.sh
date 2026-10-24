#!/usr/bin/env bash
# ==============================================================================
# CubeX - PVE LXC (CT) 容器极致轻量一键全自动部署脚本
# 适用环境: Proxmox VE 7.x / 8.x / 9.x (x86_64)
# 特性: 
#   1. 纯原生 Alpine Linux 极致精简底座 (仅占 30~70MB 内存，启动耗时 < 1秒)
#   2. 自动配置 /dev/net/tun 设备穿透与 nesting 特性 (原生支持 TUN / TProxy)
#   3. 全自动更新 Alpine 至最新版本，升级基础组件
#   4. 全自动部署并注册 CubeX OpenRC 开机守护进程
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
   PVE LXC 极速原生容器一键部署总线
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

# 2. 默认参数探查
# 自动寻找下一个未占用的 VMID (从 105 开始)
NEXT_ID=105
while pct status "$NEXT_ID" >/dev/null 2>&1 || [ -f "/etc/pve/lxc/${NEXT_ID}.conf" ] || [ -f "/etc/pve/qemu-server/${NEXT_ID}.conf" ]; do
    NEXT_ID=$((NEXT_ID + 1))
done

# 探查可用存储池 (优先 local-lvm, local-zfs, local)
DEFAULT_STORAGE="local-lvm"
if ! pvesm status -storage local-lvm >/dev/null 2>&1; then
    if pvesm status -storage local-zfs >/dev/null 2>&1; then
        DEFAULT_STORAGE="local-zfs"
    else
        DEFAULT_STORAGE="local"
    fi
fi

# 探查宿主网关与局域网段 (严禁写死，全自动自适应)
GW_IP=$(ip route | grep default | awk '{print $3}' | head -n1)
if [ -z "$GW_IP" ]; then
    GW_IP=$(ip route get 1.1.1.1 2>/dev/null | awk '{print $3}' | head -n1)
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
    SUBNET_PREFIX="192.168.1"
    GW_IP="192.168.1.1"
fi

# 动态选择一个未被 PVE 宿主自身或主路由占用的 IP 作为默认推荐值
CANDIDATE_LAST=7
if [ "$GW_IP" = "${SUBNET_PREFIX}.${CANDIDATE_LAST}" ] || [ "$HOST_IP" = "${SUBNET_PREFIX}.${CANDIDATE_LAST}" ]; then
    CANDIDATE_LAST=77
fi
DEFAULT_CT_IP="${SUBNET_PREFIX}.${CANDIDATE_LAST}/24"

echo -e "\n${YELLOW}${BOLD}⚙️ 请配置 CubeX LXC 容器基础参数 (直接回车保持默认):${NC}"
read -rp "容器 VMID [默认: $NEXT_ID]: " CT_ID
CT_ID=${CT_ID:-$NEXT_ID}

read -rp "容器静态 IP (CIDR 格式) [默认: $DEFAULT_CT_IP]: " CT_IP
CT_IP=${CT_IP:-$DEFAULT_CT_IP}

read -rp "上级主路由网关 IP [默认: $GW_IP]: " CT_GW
CT_GW=${CT_GW:-$GW_IP}

read -rp "LXC 存储池 [默认: $DEFAULT_STORAGE]: " CT_STORAGE
CT_STORAGE=${CT_STORAGE:-$DEFAULT_STORAGE}

read -rp "分配内存 (MB) [推荐 256~512MB, 默认: 256]: " CT_MEM
CT_MEM=${CT_MEM:-256}

read -rp "分配磁盘大小 (GB) [推荐 2~4GB, 默认: 2]: " CT_DISK
CT_DISK=${CT_DISK:-2}

read -rp "容器主机名 [默认: cubex-gateway]: " CT_NAME
CT_NAME=${CT_NAME:-cubex-gateway}

CT_CLEAN_IP=$(echo "$CT_IP" | cut -d'/' -f1)

# 3. 模板探查与准备
echo -e "\n${CYAN}📦 正在获取最新官方 Alpine Linux LXC 模板...${NC}"
pveam update >/dev/null 2>&1 || true

TEMPLATE=$(pveam available -section system | grep -i 'alpine' | tail -n1 | awk '{print $2}')
if [ -z "$TEMPLATE" ]; then
    TEMPLATE="alpine-3.20-default_20240606_amd64.tar.xz"
fi

if ! pveam list local | grep -q "$TEMPLATE"; then
    echo -e "${BLUE}正在下载 Alpine 模板: ${TEMPLATE}...${NC}"
    pveam download local "$TEMPLATE"
fi

# 4. 创建极速优化 LXC 容器
echo -e "\n${CYAN}🚀 正在创建 CubeX LXC 容器 (CT $CT_ID)...${NC}"
pct create "$CT_ID" "local:vztmpl/$TEMPLATE" \
    --ostype alpine \
    --hostname "$CT_NAME" \
    --cores 1 \
    --memory "$CT_MEM" \
    --swap 256 \
    --rootfs "${CT_STORAGE}:${CT_DISK}" \
    --net0 name=eth0,bridge=vmbr0,ip="$CT_IP",gw="$CT_GW",type=veth \
    --nameserver "$CT_GW" \
    --features nesting=1 \
    --unprivileged 0 \
    --onboot 1 \
    --start 0

# 5. 注入 TUN 设备穿透 (确保容器原生支持 /dev/net/tun 虚拟网卡)
echo -e "${CYAN}🔧 注入宿主内核 TUN 设备直通与 cgroup 白名单...${NC}"
CONF_FILE="/etc/pve/lxc/${CT_ID}.conf"
cat << 'EOF' >> "$CONF_FILE"

# === CubeX Native TUN & Acceleration Direct Pass ===
lxc.cgroup2.devices.allow: c 10:200 rwm
lxc.mount.entry: /dev/net/tun dev/net/tun none bind,create=file
EOF

# 6. 启动容器
echo -e "${CYAN}⚡ 启动 CubeX LXC 容器...${NC}"
pct start "$CT_ID"
sleep 2

# 7. 全自动容器内部配置、升级 Alpine 至最新版本并部署守护进程
echo -e "${CYAN}🔄 容器内部全自动初始化 (更新 Alpine 最新稳定源与核心网络依赖)...${NC}"
pct exec "$CT_ID" -- sh -c '
set -e
# 配置高速国内与官方镜像源并升级至最新版本
cat << "REPO" > /etc/apk/repositories
https://mirrors.aliyun.com/alpine/latest-stable/main
https://mirrors.aliyun.com/alpine/latest-stable/community
https://dl-cdn.alpinelinux.org/alpine/latest-stable/main
https://dl-cdn.alpinelinux.org/alpine/latest-stable/community
REPO

apk update
apk upgrade
apk add --no-cache curl ca-certificates iptables ip6tables iproute2 tzdata unzip

mkdir -p /opt/CubeX/bin /opt/CubeX/data /opt/CubeX/etc /opt/CubeX/rules

# 注册 OpenRC 系统级守护进程
cat << "INIT" > /etc/init.d/cubexd
#!/sbin/openrc-run
name="cubexd"
description="CubeX System Control Plane Daemon"
command="/opt/CubeX/bin/cubexd"
command_args=""
directory="/opt/CubeX"
command_background="yes"
pidfile="/run/cubexd.pid"
output_log="/var/log/cubexd.log"
error_log="/var/log/cubexd.err"

depend() {
    need net
    after firewall
}
INIT

chmod +x /etc/init.d/cubexd
rc-update add cubexd default
'

echo -e "${GREEN}✓ Alpine 容器底座已升级至最新版本并就绪！${NC}"

# 8. 下载或导入最新版 CubeX
echo -e "\n${CYAN}📥 正在获取最新版 CubeX 单二进制控制面...${NC}"
# 若当前目录下存在预构建产物，优先直接推入
if [ -f "dist/cubexd-linux-amd64" ]; then
    echo -e "${BLUE}检测到本地已构建的 cubexd，直接同步推送至容器...${NC}"
    pct push "$CT_ID" dist/cubexd-linux-amd64 /opt/CubeX/bin/cubexd -perms 755
else
    # 从 GitHub Release 拉取最新发行包
    pct exec "$CT_ID" -- sh -c '
    LATEST_URL=$(curl -sSL "https://api.github.com/repos/zyeyefyass/CubeX/releases" | grep "browser_download_url.*linux-amd64.zip" | head -n1 | cut -d "\"" -f4 || true)
    if [ -n "$LATEST_URL" ]; then
        curl -sSL -o /tmp/cubex.zip "$LATEST_URL"
        unzip -o /tmp/cubex.zip -d /opt/CubeX/
        chmod +x /opt/CubeX/bin/* || true
        rm -f /tmp/cubex.zip
    fi
    '
fi

# 9. 启动服务
echo -e "${CYAN}🚀 正在启动 CubeX 守护进程...${NC}"
pct exec "$CT_ID" -- rc-service cubexd restart || true
sleep 2

# 10. 检查运行状态
if pct exec "$CT_ID" -- pgrep -f cubexd >/dev/null 2>&1; then
    echo -e "\n${GREEN}${BOLD}═════════════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}${BOLD}🎉 恭喜！CubeX 宿主共生透明代理网关已在 PVE LXC 容器成功部署运行！${NC}"
    echo -e "${GREEN}${BOLD}═════════════════════════════════════════════════════════════════${NC}"
    echo -e "  📌 容器 ID     : ${BOLD}${CT_ID} (${CT_NAME})${NC}"
    echo -e "  🌐 旁路由 IP   : ${BOLD}${CT_CLEAN_IP}${NC}"
    echo -e "  🖥️ Web 控制台  : ${CYAN}${BOLD}http://${CT_CLEAN_IP}:2026${NC}"
    echo -e "  ⚡ 底座优势    : 原生共享宿主物理内核，内存仅需 70MB，启动 < 1 秒"
    echo -e "  🔑 初始账号    : admin  |  初始密码: admin123"
    echo -e "\n${YELLOW}💡 下一步联动指引:${NC}"
    echo -e "  1. 打开浏览器登录 Web 控制台: ${CYAN}http://${CT_CLEAN_IP}:2026${NC}"
    echo -e "  2. 进入【⚙️ 宿主共生与系统运维】，输入主路由（如 RouterOS / 爱快）的 IP 与凭据"
    echo -e "  3. 点击【⚡ 一键全栈自愈】，CubeX 将自动完成策略路由表创建、Fake-IP 回指与 0 秒逃生编排！"
    echo -e "${GREEN}${BOLD}═════════════════════════════════════════════════════════════════${NC}\n"
else
    echo -e "${YELLOW}⚠️ 容器已就绪，但尚未下载到 CubeX 二进制文件。可通过命令手动推送:${NC}"
    echo -e "  pct push ${CT_ID} <cubexd-linux-amd64> /opt/CubeX/bin/cubexd -perms 755"
    echo -e "  pct exec ${CT_ID} -- rc-service cubexd start"
fi
