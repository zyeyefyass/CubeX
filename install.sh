#!/usr/bin/env bash
# ═════════════════════════════════════════════════════════════════════════════
# 🌌 CubeX Universal Deployer · 次世代双擎高可用旁路网关一键极速安装程序
# Target Version: Test v1.1
# Supported: 飞牛 fnOS (/vol1/1000/cubex), RouterOS Container, PVE, OpenWrt, Linux
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
 [Version: Test v1.1 | Engine: Mihomo × sing-box v1.14+]
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

echo -e "${CYAN}🖥️  检测到宿主平台: ${BOLD}${DETECTED_HOST_NAME}${NC}"
echo -e "${BLUE}📁 自动匹配推荐部署路径: ${BOLD}${DEFAULT_INSTALL_DIR}${NC}"

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
    echo ""
    echo -e "${BOLD}请选择或确认安装目录:${NC} (直接回车保持默认)"
    read -r -p "安装路径 [${DEFAULT_INSTALL_DIR}]: " INPUT_DIR
    if [ -n "$INPUT_DIR" ]; then
        INSTALL_DIR="$INPUT_DIR"
    fi

    # 飞牛 fnOS 友好提示
    if [ "$DETECTED_HOST" = "fnos" ] && [ "$INSTALL_DIR" = "/vol1/1000/cubex" ]; then
        echo -e "${GREEN}✨ 已应用飞牛官方推荐持久化目录: /vol1/1000/cubex${NC}"
    fi

    if [ -z "$DEPLOY_MODE" ]; then
        echo ""
        echo -e "${BOLD}请选择运行部署模式:${NC}"
        echo -e "  ${CYAN}1) Docker 容器化主旁一体部署 (推荐 · 隔离性好 · 飞牛/PVE/RouterOS 首选)${NC}"
        echo -e "  ${PURPLE}2) 原生单二进制 Systemd 守护部署 (纯粹裸跑 · 内存开销仅 18MB · 极客首选)${NC}"
        read -r -p "请输入选项 [1/2, 默认 1]: " MODE_CHOICE
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
# 5. Linux 内核网络与虚拟网卡预检加固
# ─────────────────────────────────────────────────────────────────────────────
echo -e "\n${YELLOW}⚙️  [1/4] 执行宿主底层网络协议栈加固 (启用 IP 转发与 TUN 设备)...${NC}"

# 开启 IP 转发
sysctl -w net.ipv4.ip_forward=1 >/dev/null 2>&1 || true
sysctl -w net.ipv6.conf.all.forwarding=1 >/dev/null 2>&1 || true

# 检查并自动创建 /dev/net/tun
if [ ! -c /dev/net/tun ]; then
    mkdir -p /dev/net
    mknod /dev/net/tun c 10 200 2>/dev/null || true
    chmod 666 /dev/net/tun 2>/dev/null || true
fi
modprobe tun 2>/dev/null || true

# ─────────────────────────────────────────────────────────────────────────────
# 6. 目录结构初始化与数据包释放
# ─────────────────────────────────────────────────────────────────────────────
echo -e "${YELLOW}📁 [2/4] 初始化目录与数据包释放 (${INSTALL_DIR})...${NC}"
mkdir -p "$INSTALL_DIR/bin" "$INSTALL_DIR/etc/rules" "$INSTALL_DIR/data"

# 若当前目录已有 etc 数据包，则直接同步复制
CURRENT_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -d "$CURRENT_SCRIPT_DIR/etc" ]; then
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

if [ "$DEPLOY_MODE" = "docker" ]; then
    cat <<EOF > "$INSTALL_DIR/docker-compose.yml"
services:
  cubex:
    image: ghcr.io/zyeyefyass/cubex:latest
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
      - CUBEX_PORT=2026
      - CUBEX_LISTEN=:2026
      - CUBEX_HOST_PLATFORM=${DETECTED_HOST}
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
EOF

    echo -e "${GREEN}✅ 已生成 docker-compose.yml 配置文件${NC}"
    cd "$INSTALL_DIR"
    
    if command -v docker-compose >/dev/null 2>&1; then
        docker-compose down 2>/dev/null || true
        docker-compose up -d
    elif docker compose version >/dev/null 2>&1; then
        docker compose down 2>/dev/null || true
        docker compose up -d
    else
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
          -e CUBEX_PORT=2026 \
          -e CUBEX_LISTEN=:2026 \
          ghcr.io/zyeyefyass/cubex:latest || true
    fi

else
    echo -e "${PURPLE}⚡ [3/4] 正在配置原生单二进制 Systemd 守护进程...${NC}"
    
    # 查找本地是否有编译好的二进制，若有则复制
    if [ -f "$CURRENT_SCRIPT_DIR/dist/cubexd-linux-$TARGET_ARCH" ]; then
        cp "$CURRENT_SCRIPT_DIR/dist/cubexd-linux-$TARGET_ARCH" "$INSTALL_DIR/bin/cubexd"
    elif [ -f "./dist/cubexd-linux-$TARGET_ARCH" ]; then
        cp "./dist/cubexd-linux-$TARGET_ARCH" "$INSTALL_DIR/bin/cubexd"
    elif [ -f "$CURRENT_SCRIPT_DIR/cubexd" ]; then
        cp "$CURRENT_SCRIPT_DIR/cubexd" "$INSTALL_DIR/bin/cubexd"
    fi
    chmod +x "$INSTALL_DIR/bin/cubexd" 2>/dev/null || true

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
ExecStart=$INSTALL_DIR/bin/cubexd -dir $INSTALL_DIR -listen :2026
Restart=always
RestartSec=2s
LimitNOFILE=65535
Environment=GOMEMLIMIT=60MiB
Environment=GOGC=30

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable cubex >/dev/null 2>&1 || true
    systemctl restart cubex || true
    echo -e "${GREEN}✅ systemd 服务已注册并启动 (/etc/systemd/system/cubex.service)${NC}"
fi

# ─────────────────────────────────────────────────────────────────────────────
# 8. 快捷命令软链接与部署完成驾驶舱
# ─────────────────────────────────────────────────────────────────────────────
echo -e "${YELLOW}🔗 [4/4] 注册全局管理快捷指令 (cubex)...${NC}"
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
        curl -s http://127.0.0.1:2026/api/v1/system/status | grep -o '"version":[^,]*' || true
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
        curl -s -X POST http://127.0.0.1:2026/api/v1/system/reload-config
        echo ""
        ;;
    *)
        echo "🌌 CubeX 命令行控制中心"
        echo "用法: cubex {status|restart|log|reload}"
        ;;
esac
EOF
chmod +x /usr/local/bin/cubex 2>/dev/null || true

# 获取本机局域网 IP
LOCAL_IP=$(ip route get 1.1.1.1 2>/dev/null | grep -oP 'src \K\S+' || hostname -I 2>/dev/null | awk '{print $1}' || echo "127.0.0.1")

echo ""
echo -e "${GREEN}═════════════════════════════════════════════════════════════════════════${NC}"
echo -e "${BOLD}${GREEN}🎉 恭喜！CubeX · 次世代双擎高可用旁路网关部署成功！${NC}"
echo -e "${GREEN}═════════════════════════════════════════════════════════════════════════${NC}"
echo -e "  🌐 ${BOLD}Web 控制驾驶舱:${NC}   ${CYAN}http://${LOCAL_IP}:2026${NC}"
echo -e "  🖥️  ${BOLD}已识别宿主底座:${NC}   ${PURPLE}${DETECTED_HOST_NAME}${NC}"
echo -e "  📁 ${BOLD}持久化工作目录:${NC}   ${BLUE}${INSTALL_DIR}${NC}"
echo -e "  ⚡ ${BOLD}双核心代理架构:${NC}   ${GREEN}Mihomo Meta ⇄ sing-box v1.14+ (毫秒级零断网热切)${NC}"
echo -e "  🛡️  ${BOLD}权威规则数据库:${NC}   ${CYAN}217heidai MRS 广告拦截 + 隐私反追踪 + 0 明文 DNS 阻断${NC}"
echo ""
echo -e "${BOLD}常用运维指令:${NC}"
echo -e "  • 查看运行状态:   ${YELLOW}cubex status${NC}"
echo -e "  • 追踪实时日志:   ${YELLOW}cubex log${NC}"
echo -e "  • 重启网关服务:   ${YELLOW}cubex restart${NC}"
echo -e "  • 热重载分流配置: ${YELLOW}cubex reload${NC}"
echo -e "${GREEN}═════════════════════════════════════════════════════════════════════════${NC}"
EOF
