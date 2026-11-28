# CubeX 飞牛 fnOS 专属 Docker 部署模块 (POSIX sh)
run_platform_fnos() {
    msg "${CYAN}=================================================================${NC}"
    msg "${CYAN}  🌌 CubeX ${CUBEX_DEFAULT_TAG} · 飞牛 fnOS 专属宿主共生透明网关部署${NC}"
    msg "${CYAN}  🛡️ 网关与DNS统一协同 · 全家终端零改动 · 全链路自愈${NC}"
    msg "${CYAN}=================================================================${NC}"

    if [ "$(id -u)" -ne 0 ]; then
        msg "${RED}❌ 请使用 root 权限或 sudo 运行此脚本！${NC}"
        exit 1
    fi

    if ! command -v docker >/dev/null 2>&1; then
        msg "${RED}❌ 未检测到 Docker 环境，请先在飞牛应用中心安装 Docker！${NC}"
        exit 1
    fi

    _fnos_def_route=$(ip route show default 2>/dev/null | head -n 1 || true)
    _fnos_gw=$(printf '%s\n' "$_fnos_def_route" | awk '{print $3}' || true)
    _fnos_iface=$(printf '%s\n' "$_fnos_def_route" | awk '{print $5}' || true)
    _fnos_host_ip=""
    if [ -n "$_fnos_iface" ]; then
        _fnos_host_ip=$(ip -4 addr show "$_fnos_iface" 2>/dev/null | awk '/inet /{print $2}' | cut -d/ -f1 | head -n 1 || true)
    fi
    if [ -z "$_fnos_host_ip" ]; then
        _fnos_host_ip=$(ip route get 1.1.1.1 2>/dev/null | awk '{for(i=1;i<=NF;i++)if($i=="src")print $(i+1)}' | head -n 1 || true)
    fi

    msg "${GREEN}✓ 动态网络探查成功:${NC}"
    msg "  * 飞牛 NAS IP   : ${YELLOW}${_fnos_host_ip:-自动获取}${NC}"
    msg "  * 主路由网关 IP : ${YELLOW}${_fnos_gw:-自动获取}${NC}"
    msg "  * 物理网络接口  : ${YELLOW}${_fnos_iface:-默认}${NC}"

    _fnos_storage="/var/lib/cubex"
    if [ -d "/vol1/1000" ]; then
        _fnos_storage="/vol1/1000/docker/cubex"
    elif [ -d "/vol1" ]; then
        _fnos_storage="/vol1/docker/cubex"
    elif [ -d "/vol2" ]; then
        _fnos_storage="/vol2/docker/cubex"
    fi

    msg "  * 数据持久化路径: ${YELLOW}${_fnos_storage}${NC}"
    mkdir -p "${_fnos_storage}/etc" "${_fnos_storage}/data"

    if [ ! -c /dev/net/tun ]; then
        msg "${YELLOW}🔧 正在初始化 /dev/net/tun 虚拟网卡通道...${NC}"
        mkdir -p /dev/net
        mknod /dev/net/tun c 10 200 2>/dev/null || true
        chmod 666 /dev/net/tun 2>/dev/null || true
    fi
    modprobe tun 2>/dev/null || true

    if docker ps -a --format '{{.Names}}' | grep -Eq '^cubex$'; then
        msg "${YELLOW}🔄 正在更新现有 CubeX 容器...${NC}"
        docker stop cubex >/dev/null 2>&1 || true
        docker rm cubex >/dev/null 2>&1 || true
    fi

    msg "${GREEN}🚀 正在拉取官方最新镜像 (fyasshole/cubex:latest)...${NC}"
    docker pull fyasshole/cubex:latest

    msg "${GREEN}🚀 正在启动 CubeX 宿主共生容器...${NC}"
    docker run -d \
      --name cubex \
      --restart always \
      --net=host \
      --privileged \
      --cap-add=NET_ADMIN \
      --cap-add=NET_RAW \
      --device /dev/net/tun:/dev/net/tun \
      -v "${_fnos_storage}/etc:/app/etc" \
      -v "${_fnos_storage}/data:/app/data" \
      -v /etc/localtime:/etc/localtime:ro \
      -e TZ=Asia/Shanghai \
      -e CUBEX_ENV=production \
      -e GOMEMLIMIT=128MiB \
      -e CUBEX_PORT=2026 \
      -e CUBEX_LISTEN=:2026 \
      fyasshole/cubex:latest

    msg "\n${GREEN}=================================================================${NC}"
    msg "${GREEN}  🎉 恭喜！CubeX ${CUBEX_DEFAULT_TAG} 在飞牛 fnOS 上启动成功！${NC}"
    msg "${GREEN}=================================================================${NC}"
    msg "👉 Web 控制台访问入口: ${CYAN}http://${_fnos_host_ip:-127.0.0.1}:2026${NC}"
    msg "👉 主路由联动两步法:"
    msg "   1. 登录控制台【宿主共生】输入 RouterOS / 爱快 凭据，点击一键自愈；"
    msg "   2. 全屋终端保持默认 DHCP，网关+DNS全指主路由，永不断线！"
}
