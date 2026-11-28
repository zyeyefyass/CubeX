# CubeX Linux / 通用宿主 (Docker 与 Native Systemd/OpenRC) 部署模块 (POSIX sh)
run_platform_linux() {
    clear 2>/dev/null || true
    cat << "EOF"
  ██████╗██╗   ██╗██████╗ ███████╗██╗  ██╗
 ██╔════╝██║   ██║██╔══██╗██╔════╝╚██╗██╔╝
 ██║     ██║   ██║██████╔╝█████╗   ╚███╔╝ 
 ██║     ██║   ██║██╔══██╗██╔══╝   ██╔██╗ 
 ╚██████╗╚██████╔╝██████╔╝███████╗██╔╝ ██╗
  ╚═════╝ ╚═════╝ ╚═════╝ ╚══════╝╚═╝  ╚═╝
 🌌 次世代双擎高可用旁路网关系统 · 软路由的天花板
 [Engine: Mihomo × sing-box v1.14+]
EOF
    echo ""

    if [ "$(id -u)" -ne 0 ]; then
        msg "${RED}❌ 错误: 此安装脚本需要 root 超级用户权限执行。${NC}"
        msg "${YELLOW}👉 请使用: sudo sh $0${NC}"
        exit 1
    fi

    detect_target_arch
    msg "${GREEN}🔍 检测到处理器架构: ${BOLD}${TARGET_ARCH}${NC}"
    ensure_linux_base_tools
    detect_host_topology_and_platform

    msg "${CYAN}🖥️  检测到宿主平台: ${BOLD}${DETECTED_HOST_NAME}${NC}"
    msg "${BLUE}📁 自动匹配推荐部署路径: ${BOLD}${DEFAULT_INSTALL_DIR}${NC}"
    if [ "$IS_IKUAI_GATEWAY" -eq 1 ]; then
        msg "${PURPLE}🚀 嗅探到上级主路由为: ${BOLD}爱快 iKuai 企业级路由 (${DEFAULT_GW})${NC}"
        msg "${CYAN}💡 【爱快宿主共生建议】: 部署完成后，在 CubeX 驾驶舱【宿主共生凭据】中填入爱快 IP (${DEFAULT_GW}) 与管理密码，系统将全自动下发 198.18.0.0/15 静态回指与 DHCP 联动加固！${NC}"
    fi

    [ -n "$CUSTOM_DIR" ] && DEFAULT_INSTALL_DIR="$CUSTOM_DIR"
    INSTALL_DIR="${DEFAULT_INSTALL_DIR}"

    if [ "$AUTO_CONFIRM" -eq 0 ]; then
        if [ ! -t 0 ] || [ ! -e /dev/tty ]; then
            AUTO_CONFIRM=1
        fi
    fi

    if [ "$AUTO_CONFIRM" -eq 0 ]; then
        echo ""
        msg "${BOLD}请选择或确认安装目录:${NC} (直接回车保持默认)"
        prompt_read "安装路径 [${DEFAULT_INSTALL_DIR}]: "
        INPUT_DIR="$REPLY"
        if [ -n "$INPUT_DIR" ]; then
            CLEAN_DIR=$(printf '%s\n' "$INPUT_DIR" | tr -d '\r\n' | grep -E '^/[a-zA-Z0-9_\.\/-]+$' || true)
            if [ -n "$CLEAN_DIR" ]; then
                INSTALL_DIR="$CLEAN_DIR"
            else
                INSTALL_DIR="$DEFAULT_INSTALL_DIR"
            fi
        fi

        if [ "$DETECTED_HOST" = "fnos" ] && [ "$INSTALL_DIR" = "/vol1/1000/cubex" ]; then
            msg "${GREEN}✨ 已应用飞牛官方推荐持久化目录: /vol1/1000/cubex${NC}"
        fi

        if [ -z "$DEPLOY_MODE" ]; then
            echo ""
            msg "${BOLD}请选择运行部署模式:${NC}"
            msg "  ${CYAN}1) Docker 容器化主旁一体部署 (推荐 · 隔离性好 · 飞牛/PVE/RouterOS/爱快首选)${NC}"
            msg "  ${PURPLE}2) 原生单二进制 Systemd/OpenRC 守护部署 (纯粹裸跑 · 极致轻量低开销 · 极客首选)${NC}"
            prompt_read "请输入选项 [1/2, 默认 1]: "
            case "$REPLY" in
                2) DEPLOY_MODE="native" ;;
                *) DEPLOY_MODE="docker" ;;
            esac
        fi
    else
        if [ -z "$DEPLOY_MODE" ]; then
            if command -v docker >/dev/null 2>&1; then
                DEPLOY_MODE="docker"
            else
                DEPLOY_MODE="native"
            fi
        fi
    fi

    msg "\n${YELLOW}⚙️  [1/4] 执行宿主底层网络协议栈加固 (开启内核转发与网络缓冲区优化)...${NC}"
    apply_linux_netstack_tuning

    msg "${YELLOW}📁 [2/4] 初始化目录与数据包释放 (${INSTALL_DIR})...${NC}"
    mkdir -p "$INSTALL_DIR/bin" "$INSTALL_DIR/etc/rules" "$INSTALL_DIR/data"

    CURRENT_SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
    SRC_ETC=""
    if [ -d "$CURRENT_SCRIPT_DIR/../etc" ]; then
        SRC_ETC="$CURRENT_SCRIPT_DIR/../etc"
    elif [ -d "$CURRENT_SCRIPT_DIR/etc" ]; then
        SRC_ETC="$CURRENT_SCRIPT_DIR/etc"
    elif [ -d "./etc" ]; then
        SRC_ETC="./etc"
    fi
    if [ -n "$SRC_ETC" ]; then
        for _pl_item in "$SRC_ETC"/*; do
            [ -e "$_pl_item" ] || continue
            _pl_base=$(basename "$_pl_item")
            case "$_pl_base" in
                config.yaml|config.json)
                    [ -e "$INSTALL_DIR/etc/$_pl_base" ] || cp -r "$_pl_item" "$INSTALL_DIR/etc/"
                    ;;
                *)
                    cp -r "$_pl_item" "$INSTALL_DIR/etc/"
                    ;;
            esac
        done
    fi

    if [ "$DETECTED_HOST" = "fnos" ]; then
        chown -R 1000:1000 "$INSTALL_DIR" 2>/dev/null || true
    fi

    if [ "$DEPLOY_MODE" = "docker" ]; then
        msg "${YELLOW}🐳 [3/4] 正在配置 Docker 容器化主旁一体网关...${NC}"
        if ! command -v docker >/dev/null 2>&1; then
            msg "${RED}⚠️  检测到当前系统未安装 Docker，正在自动降级为原生单二进制守护部署...${NC}"
            DEPLOY_MODE="native"
        fi
    fi

    APP_PORT="${PORT:-${CUBEX_PORT:-2026}}"
    if command -v ss >/dev/null 2>&1; then
        if ss -tuln | grep -q ":${APP_PORT} "; then
            if docker ps 2>/dev/null | grep -q "cubex" || (command -v systemctl >/dev/null 2>&1 && (systemctl is-active --quiet cubex 2>/dev/null || systemctl is-active --quiet cubexd 2>/dev/null)) || pgrep -x cubexd >/dev/null 2>&1; then
                msg "${CYAN}💡 检测到已有 CubeX 运行于端口 ${APP_PORT}，将执行平滑替换升级...${NC}"
            else
                msg "${YELLOW}⚠️ 宿主端口 ${APP_PORT} 已被其他应用占用，启动智能空闲端口探查...${NC}"
                while ss -tuln | grep -q ":${APP_PORT} "; do
                    APP_PORT=$((APP_PORT + 1))
                done
                msg "${GREEN}✅ 自动切换至空闲可用端口: ${APP_PORT}${NC}"
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
        msg "${GREEN}✅ 已生成 docker-compose.yml 配置文件 (内存配额: ${HOST_GOMEMLIMIT})${NC}"
        cd "$INSTALL_DIR"
        if command -v docker-compose >/dev/null 2>&1; then
            docker-compose down 2>/dev/null || true
            docker-compose up -d
        elif docker compose version >/dev/null 2>&1; then
            docker compose down 2>/dev/null || true
            docker compose up -d
        else
            DOCKER_IMG="docker.io/fyasshole/cubex:latest"
            msg "${CYAN}💡 正在拉取最新版 Docker 镜像 (${DOCKER_IMG})...${NC}"
            docker pull "$DOCKER_IMG" 2>/dev/null || true
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
        msg "${PURPLE}⚡ [3/4] 正在配置原生单二进制 Systemd/OpenRC 守护进程...${NC}"
        NEED_DOWNLOAD=1
        LOCAL_ZIP=$(ls "$CURRENT_SCRIPT_DIR/../dist/release/CubeX-"*"-linux-$TARGET_ARCH.zip" "$CURRENT_SCRIPT_DIR/../dist/CubeX-"*"-linux-$TARGET_ARCH.zip" "./dist/release/CubeX-"*"-linux-$TARGET_ARCH.zip" "./dist/CubeX-"*"-linux-$TARGET_ARCH.zip" "/root/CubeX-"*"-linux-$TARGET_ARCH.zip" "/tmp/CubeX-"*"-linux-$TARGET_ARCH.zip" 2>/dev/null | sort -V | tail -n1 || true)
        if [ -n "$LOCAL_ZIP" ] && [ -f "$LOCAL_ZIP" ] && zip_is_valid "$LOCAL_ZIP"; then
            msg "${GREEN}📦 发现本地发行归档包 (${LOCAL_ZIP})，直接 0 秒解压部署...${NC}"
            if command -v systemctl >/dev/null 2>&1; then systemctl stop cubex 2>/dev/null || true; fi
            if command -v rc-service >/dev/null 2>&1; then rc-service cubex stop 2>/dev/null || true; fi
            if [ -s "$INSTALL_DIR/data/config.ir.json" ]; then
                zip_extract_pkg "$LOCAL_ZIP" "$INSTALL_DIR" "1"
            else
                zip_extract_pkg "$LOCAL_ZIP" "$INSTALL_DIR" "0"
            fi
            NEED_DOWNLOAD=0
        elif [ -f "$CURRENT_SCRIPT_DIR/../dist/cubexd-linux-$TARGET_ARCH" ]; then
            if command -v systemctl >/dev/null 2>&1; then systemctl stop cubex 2>/dev/null || true; fi
            if command -v rc-service >/dev/null 2>&1; then rc-service cubex stop 2>/dev/null || true; fi
            cp "$CURRENT_SCRIPT_DIR/../dist/cubexd-linux-$TARGET_ARCH" "$INSTALL_DIR/bin/cubexd"
            NEED_DOWNLOAD=0
        fi

        if [ "$NEED_DOWNLOAD" -eq 1 ] && [ ! -f "$INSTALL_DIR/bin/cubexd" ]; then
            msg "${CYAN}🌐 正在从 GitHub Releases 加速源拉取最新完整发布包 (双内核 + 105套规则集 + 控制面)...${NC}"
            cubex_resolve_latest_tag "${CUSTOM_VER:-}"
            ZIP_NAME="CubeX-${REL_TAG}-linux-${TARGET_ARCH}.zip"
            ZIP_URL="https://github.com/zyeyefyass/CubeX/releases/download/${REL_TAG}/${ZIP_NAME}"
            TMP_ZIP="/tmp/${ZIP_NAME}"

            if [ -s "$TMP_ZIP" ] && zip_is_valid "$TMP_ZIP"; then
                msg "${GREEN}📦 发现本地已缓存的完整归档: ${TMP_ZIP}，秒级解压部署...${NC}"
            else
                msg "${CYAN}   -> 正在通过国内多级高速镜像源智能轮询下载 ${ZIP_NAME} (无代理极速拉取)...${NC}"
                cubex_download_with_mirrors "$ZIP_URL" "$TMP_ZIP" "zip_is_valid" "300" "0" || true
            fi

            if [ -s "$TMP_ZIP" ] && zip_is_valid "$TMP_ZIP"; then
                if [ -s "$INSTALL_DIR/data/config.ir.json" ]; then
                    zip_extract_pkg "$TMP_ZIP" "$INSTALL_DIR" "1"
                else
                    zip_extract_pkg "$TMP_ZIP" "$INSTALL_DIR" "0"
                fi
                msg "${GREEN}✅ 完整归档已成功释放至 ${INSTALL_DIR} (包含 Mihomo + sing-box 双核及全量规则)${NC}"
            else
                msg "${YELLOW}⚠️ 完整归档拉取失败，尝试独立拉取控制面与双引擎独立内核...${NC}"
                cubex_download_with_mirrors "https://github.com/zyeyefyass/CubeX/releases/download/${REL_TAG}/cubexd-linux-${TARGET_ARCH}" "$INSTALL_DIR/bin/cubexd" "elf_is_valid" "120" "0" || true
                if [ ! -f "$INSTALL_DIR/bin/sing-box" ]; then
                    msg "${CYAN}   -> 正在独立拉取 sing-box 核心...${NC}"
                    cubex_download_with_mirrors "https://github.com/zyeyefyass/CubeX/releases/download/${REL_TAG}/sing-box-linux-${TARGET_ARCH}" "$INSTALL_DIR/bin/sing-box" "elf_is_valid" "120" "0" || true
                fi
                if [ ! -f "$INSTALL_DIR/bin/mihomo" ]; then
                    msg "${CYAN}   -> 正在独立拉取 mihomo 核心...${NC}"
                    cubex_download_with_mirrors "https://github.com/zyeyefyass/CubeX/releases/download/${REL_TAG}/mihomo-linux-${TARGET_ARCH}" "$INSTALL_DIR/bin/mihomo" "elf_is_valid" "120" "0" || true
                fi
            fi
        fi

        chmod +x "$INSTALL_DIR/bin/"* 2>/dev/null || true
        if [ ! -f "$INSTALL_DIR/bin/cubexd" ] || [ ! -s "$INSTALL_DIR/bin/cubexd" ]; then
            msg "${RED}❌ 致命错误: 未能获取到有效的 cubexd 控制面二进制文件，请检查网络连接或权限后重试。${NC}"
            exit 1
        fi
        if [ ! -f "$INSTALL_DIR/bin/sing-box" ] && [ ! -f "$INSTALL_DIR/bin/mihomo" ]; then
            msg "${RED}❌ 致命错误: 双引擎核心文件 (sing-box / mihomo) 均缺失！${NC}"
            msg "${YELLOW}👉 解决建议: 请检查宿主网络/镜像源连接，或将完整的 CubeX-*-linux-${TARGET_ARCH}.zip 放置于 /tmp/ 目录后重新运行。${NC}"
            exit 1
        fi

        mkdir -p /usr/local/bin 2>/dev/null || true
        if [ -f "$INSTALL_DIR/bin/cubex" ]; then
            ln -sf "$INSTALL_DIR/bin/cubex" /usr/local/bin/cubex 2>/dev/null || cp "$INSTALL_DIR/bin/cubex" /usr/local/bin/cubex 2>/dev/null || true
            chmod +x /usr/local/bin/cubex 2>/dev/null || true
        fi

        if command -v systemctl >/dev/null 2>&1 && [ -d /etc/systemd/system ]; then
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
            msg "${GREEN}✅ systemd 服务已注册并启动 (/etc/systemd/system/cubex.service, 软上限: ${HOST_GOMEMLIMIT})${NC}"
        elif command -v rc-service >/dev/null 2>&1 || [ -d /etc/init.d ]; then
            cat <<EOF > /etc/init.d/cubex
#!/sbin/openrc-run
name="cubex"
description="CubeX System Control Plane Daemon"
command="$INSTALL_DIR/bin/cubexd"
command_args="-dir $INSTALL_DIR -listen :${APP_PORT}"
directory="$INSTALL_DIR"
supervisor="supervise-daemon"
respawn_max=0
output_log="/var/log/cubex.log"
error_log="/var/log/cubex.err"

export GOMEMLIMIT="${HOST_GOMEMLIMIT}"
export GOGC="${HOST_GOGC}"
export GODEBUG="madvdontneed=1"
export TZ="Asia/Shanghai"

depend() {
    need net
    after firewall
}
EOF
            chmod 755 /etc/init.d/cubex
            command -v rc-update >/dev/null 2>&1 && rc-update add cubex default >/dev/null 2>&1 || true
            command -v rc-service >/dev/null 2>&1 && { rc-service cubex restart >/dev/null 2>&1 || rc-service cubex start >/dev/null 2>&1 || true; }
            msg "${GREEN}✅ OpenRC 服务已注册并启动 (/etc/init.d/cubex, 软上限: ${HOST_GOMEMLIMIT})${NC}"
        else
            nohup "$INSTALL_DIR/bin/cubexd" -dir "$INSTALL_DIR" -listen ":${APP_PORT}" </dev/null >/var/log/cubexd.log 2>&1 &
            msg "${GREEN}✅ 后台守护进程已拉起 (PID: $!)${NC}"
        fi
    fi

    msg "${YELLOW}🔗 [4/4] 注册全局管理快捷指令 (cubex)...${NC}"
    mkdir -p /usr/local/bin 2>/dev/null || true
    CLI_INSTALLED=0
    for _pl_cli in "$INSTALL_DIR/bin/cubex" "$CURRENT_SCRIPT_DIR/../dist/cubex-linux-$TARGET_ARCH" "$CURRENT_SCRIPT_DIR/dist/cubex-linux-$TARGET_ARCH" "./dist/cubex-linux-$TARGET_ARCH"; do
        if [ -f "$_pl_cli" ]; then
            ln -sf "$_pl_cli" /usr/local/bin/cubex 2>/dev/null || cp "$_pl_cli" /usr/local/bin/cubex 2>/dev/null || true
            chmod +x /usr/local/bin/cubex 2>/dev/null || true
            CLI_INSTALLED=1
            break
        fi
    done

    if [ "$CLI_INSTALLED" -eq 0 ]; then
        msg "${CYAN}🌐 正在通过加速镜像源尝试拉取最新版原生运维 CLI (cubex-linux-${TARGET_ARCH})...${NC}"
        if cubex_download_with_mirrors "https://github.com/zyeyefyass/CubeX/releases/latest/download/cubex-linux-$TARGET_ARCH" "/usr/local/bin/cubex" "elf_is_valid" "30" "0"; then
            chmod +x /usr/local/bin/cubex 2>/dev/null || true
            CLI_INSTALLED=1
        fi
    fi

    if [ "$CLI_INSTALLED" -eq 0 ]; then
        cat <<EOF > /usr/local/bin/cubex
#!/bin/sh
APP_PORT="${APP_PORT}"
EOF
        cat << 'EOF' >> /usr/local/bin/cubex
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

    LOCAL_IP=$(ip route get 1.1.1.1 2>/dev/null | awk '{for(i=1;i<=NF;i++)if($i=="src")print $(i+1)}' || hostname -i 2>/dev/null | awk '{print $1}' || echo "127.0.0.1")
    check_resolv_self_loop

    echo ""
    msg "${GREEN}═════════════════════════════════════════════════════════════════════════${NC}"
    msg "${BOLD}${GREEN}🎉 恭喜！CubeX · 次世代双擎高可用旁路网关部署成功！${NC}"
    msg "${GREEN}═════════════════════════════════════════════════════════════════════════${NC}"
    msg "  🌐 ${BOLD}Web 控制驾驶舱:${NC}   ${CYAN}http://${LOCAL_IP}:${APP_PORT}${NC}"
    msg "  🖥️  ${BOLD}已识别宿主底座:${NC}   ${PURPLE}${DETECTED_HOST_NAME}${NC}"
    msg "  📁 ${BOLD}持久化工作目录:${NC}   ${BLUE}${INSTALL_DIR}${NC}"
    msg "  ⚡ ${BOLD}双核心代理架构:${NC}   ${GREEN}Mihomo Meta ⇄ sing-box v1.14+ (毫秒级零断网热切)${NC}"
    msg "  🛡️  ${BOLD}权威规则数据库:${NC}   ${CYAN}217heidai MRS 广告拦截 + 隐私反追踪 + 0 明文 DNS 阻断${NC}"
    msg "  💾 ${BOLD}宿主自适应内存:${NC}   ${YELLOW}GOMEMLIMIT=${HOST_GOMEMLIMIT} (物理内存分级防护)${NC}"
    echo ""
    msg "${BOLD}常用运维指令:${NC}"
    msg "  • 查看运行状态:   ${YELLOW}cubex status${NC}"
    msg "  • 追踪实时日志:   ${YELLOW}cubex log${NC}"
    msg "  • 重启网关服务:   ${YELLOW}cubex restart${NC}"
    msg "  • 热重载分流配置: ${YELLOW}cubex reload${NC}"
    if [ "$IS_IKUAI_GATEWAY" -eq 1 ]; then
        echo ""
        msg "${PURPLE}═════════════════════════════════════════════════════════════════════════${NC}"
        msg "${BOLD}${PURPLE}🚀 【爱快 iKuai 宿主联动中枢就绪】(已嗅探到上级爱快主路由: ${DEFAULT_GW})${NC}"
        msg "${PURPLE}═════════════════════════════════════════════════════════════════════════${NC}"
        msg "  💡 爱快旁路由拓扑核心：需在爱快中将 ${BOLD}198.18.0.0/15${NC} 静态回指路由指向 CubeX (${LOCAL_IP})"
        msg "  🔧 ${BOLD}全自动一键配置步骤:${NC}"
        msg "     1. 浏览器打开: ${CYAN}http://${LOCAL_IP}:${APP_PORT}${NC}"
        msg "     2. 进入左侧导航【宿主共生凭据】，选择【爱快 iKuai】"
        msg "     3. 填入爱快地址 (${DEFAULT_GW}) 与 Web 管理密码"
        msg "     4. 点击【一键自愈与规则加固】，系统自动写入 198.18.0.0/15 静态回指与 DHCP 首选 DNS！"
        msg "  💻 也可在当前终端执行: ${CYAN}cubex ikuai${NC} 触发自动同步"
    fi
    if [ -n "$RESOLV_LOOP_MATCHES" ]; then
        echo ""
        msg "${RED}═════════════════════════════════════════════════════════════════════════${NC}"
        msg "${BOLD}${RED}⚠️  [DNS 防自环告警] 检测到 /etc/resolv.conf 指向本机或回环地址:${RESOLV_LOOP_MATCHES}${NC}"
        msg "${YELLOW}   • 风险说明: 旁路由模式下若宿主机自身 DNS 指向自己，DHCP 续租后极易引发 DNS 自环死锁！${NC}"
        msg "${YELLOW}   • 引擎兜底: CubeX 将自动弃用 dns-local 并安全回退至 ${RESOLV_SAFE_GW}，防止断网。${NC}"
        msg "${CYAN}   • 温和修复指引 (切勿使用 chattr +i 锁死 /etc/resolv.conf，以免破坏 systemd-resolved / NetworkManager):${NC}"
        msg "     1. 为本机配置静态 IP，并将本机自身 DNS 固定为上级网关或公共 DNS (${BOLD}${RESOLV_SAFE_GW}${NC})"
        msg "     2. NetworkManager 示例: ${YELLOW}nmcli con mod <连接名> ipv4.dns \"${RESOLV_SAFE_GW}\" ipv4.ignore-auto-dns yes && nmcli con up <连接名>${NC}"
        msg "     3. systemd-resolved 示例: 在 ${YELLOW}/etc/systemd/resolved.conf${NC} 设置 ${YELLOW}DNS=${RESOLV_SAFE_GW}${NC} 后重启服务"
        msg "${RED}═════════════════════════════════════════════════════════════════════════${NC}"
    else
        echo ""
        msg "${CYAN}💡 [防 DNS 自环提示] 若后续将主路由 DHCP DNS 设为本机 (${LOCAL_IP})，请为本机固定静态 IP 与独立 DNS (${RESOLV_SAFE_GW})，避免 DHCP 续租覆盖 /etc/resolv.conf。${NC}"
    fi
    msg "${GREEN}═════════════════════════════════════════════════════════════════════════${NC}"
}
