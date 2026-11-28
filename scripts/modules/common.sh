# CubeX POSIX sh Common Module (Sourced/Inlined by build_installers.py)
# Zero bashisms: no [[, no arrays, no pipefail, no <<<, no read -p, no local keyword.

CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
PURPLE='\033[0;35m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'
BOLD='\033[1m'

CUBEX_DEFAULT_TAG="v9.991"
CUBEX_GH_MIRRORS="https://ghfast.top https://ghproxy.net https://gh-proxy.com https://gh.ddlc.top DIRECT"
CUBEX_FASTEST_MIRROR=""

msg() { printf '%b\n' "$*"; }

prompt_read() {
    _pr_prompt="$1"
    printf '%s' "$_pr_prompt"
    if [ -r /dev/tty ]; then
        read -r REPLY </dev/tty || REPLY=""
    elif [ -t 0 ]; then
        read -r REPLY || REPLY=""
    else
        REPLY=""
    fi
}

pkg_run_timeout() {
    _prt_secs="$1"
    shift
    if command -v timeout >/dev/null 2>&1; then
        if timeout 1 true >/dev/null 2>&1; then
            timeout "$_prt_secs" "$@" >/dev/null 2>&1 || true
        else
            timeout -t "$_prt_secs" "$@" >/dev/null 2>&1 || true
        fi
    else
        "$@" >/dev/null 2>&1 || true
    fi
}

detect_target_arch() {
    _dta_arch=$(uname -m 2>/dev/null || echo "unknown")
    case "$_dta_arch" in
        x86_64|amd64)
            TARGET_ARCH="amd64"
            ;;
        aarch64|arm64)
            TARGET_ARCH="arm64"
            ;;
        *)
            msg "${RED}❌ 不支持的 CPU 架构: $_dta_arch (仅支持 x86_64 / arm64)${NC}"
            exit 1
            ;;
    esac
}

ensure_linux_base_tools() {
    if ! command -v curl >/dev/null 2>&1 || ! command -v unzip >/dev/null 2>&1 || ! command -v iptables >/dev/null 2>&1; then
        msg "${YELLOW}📦 检测到缺少基础工具 (curl / unzip / iptables)，正在自动安装依赖...${NC}"
        if command -v apt-get >/dev/null 2>&1; then
            pkg_run_timeout 10 apt-get -o Acquire::http::Timeout=5 -o Acquire::https::Timeout=5 update -y
            pkg_run_timeout 20 apt-get -o Acquire::http::Timeout=5 -o Acquire::https::Timeout=5 install -y curl ca-certificates unzip iptables iproute2 procps
        elif command -v apk >/dev/null 2>&1; then
            pkg_run_timeout 10 apk update
            pkg_run_timeout 20 apk add --no-cache curl ca-certificates unzip iptables iproute2 procps
        elif command -v dnf >/dev/null 2>&1; then
            pkg_run_timeout 20 dnf install -y curl ca-certificates unzip iptables iproute procps-ng
        elif command -v yum >/dev/null 2>&1; then
            pkg_run_timeout 20 yum install -y curl ca-certificates unzip iptables iproute procps-ng
        fi
    fi
}

zip_is_valid() {
    _ziv_zf="$1"
    [ -s "$_ziv_zf" ] || return 1
    if command -v unzip >/dev/null 2>&1; then
        unzip -t "$_ziv_zf" >/dev/null 2>&1 || unzip -l "$_ziv_zf" >/dev/null 2>&1
        return $?
    elif command -v python3 >/dev/null 2>&1; then
        python3 -c 'import sys,zipfile; z=zipfile.ZipFile(sys.argv[1]); sys.exit(1 if z.testzip() else 0)' "$_ziv_zf" >/dev/null 2>&1
        return $?
    fi
    return 1
}

gzip_is_valid() {
    _giv_gf="$1"
    [ -s "$_giv_gf" ] || return 1
    gzip -t "$_giv_gf" >/dev/null 2>&1
    return $?
}

elf_is_valid() {
    _eiv_bf="$1"
    [ -s "$_eiv_bf" ] || return 1
    head -c 4 "$_eiv_bf" 2>/dev/null | grep -q "ELF"
    return $?
}

zip_extract_pkg() {
    _zep_zf="$1"
    _zep_dst="$2"
    _zep_preserve="$3"
    if command -v unzip >/dev/null 2>&1; then
        if [ "$_zep_preserve" = "1" ]; then
            unzip -q -o "$_zep_zf" -x "data/*" "etc/config.*" -d "$_zep_dst" 2>/dev/null || true
        else
            unzip -q -o "$_zep_zf" -d "$_zep_dst" 2>/dev/null || true
        fi
    elif command -v python3 >/dev/null 2>&1; then
        python3 -c '
import sys, os, zipfile, fnmatch
zf, dst, preserve = sys.argv[1], sys.argv[2], sys.argv[3] == "1"
with zipfile.ZipFile(zf, "r") as z:
    for m in z.infolist():
        name = m.filename.replace("\\", "/").strip("/")
        if not name: continue
        if preserve and (fnmatch.fnmatch(name, "data/*") or fnmatch.fnmatch(name, "etc/config.*")): continue
        t = os.path.join(dst, name)
        if m.is_dir() or name.endswith("/"): os.makedirs(t, exist_ok=True)
        else:
            os.makedirs(os.path.dirname(t), exist_ok=True)
            with z.open(m) as s, open(t, "wb") as d: d.write(s.read())
            if m.external_attr > 0: os.chmod(t, (m.external_attr >> 16) & 0o777)
' "$_zep_zf" "$_zep_dst" "$_zep_preserve" 2>/dev/null || true
    fi
}

cubex_probe_fastest_mirror() {
    if [ -n "$CUBEX_FASTEST_MIRROR" ]; then
        return 0
    fi
    for _pfm_m in $CUBEX_GH_MIRRORS; do
        if [ "$_pfm_m" = "DIRECT" ]; then
            _pfm_probe="https://github.com/zyeyefyass/CubeX/releases/latest"
        else
            _pfm_probe="${_pfm_m}/https://github.com/zyeyefyass/CubeX/releases/latest"
        fi
        if command -v curl >/dev/null 2>&1; then
            if curl -4 --http1.1 -fsSLI --connect-timeout 3 -m 4 "$_pfm_probe" >/dev/null 2>&1; then
                CUBEX_FASTEST_MIRROR="$_pfm_m"
                return 0
            fi
        elif command -v wget >/dev/null 2>&1; then
            if wget -4 -q --spider -T 4 "$_pfm_probe" >/dev/null 2>&1; then
                CUBEX_FASTEST_MIRROR="$_pfm_m"
                return 0
            fi
        fi
    done
    return 0
}

cubex_ordered_mirrors() {
    if [ -n "$CUBEX_FASTEST_MIRROR" ]; then
        _com_list="$CUBEX_FASTEST_MIRROR"
        for _com_m in $CUBEX_GH_MIRRORS; do
            if [ "$_com_m" != "$CUBEX_FASTEST_MIRROR" ]; then
                _com_list="${_com_list} ${_com_m}"
            fi
        done
        printf '%s\n' "$_com_list"
    else
        printf '%s\n' "$CUBEX_GH_MIRRORS"
    fi
}

cubex_resolve_latest_tag() {
    _rlt_custom="$1"
    if [ -n "$_rlt_custom" ]; then
        REL_TAG="$_rlt_custom"
        return 0
    fi
    REL_TAG=""
    cubex_probe_fastest_mirror
    for _rlt_m in $(cubex_ordered_mirrors); do
        _rlt_prefix=""
        if [ "$_rlt_m" != "DIRECT" ]; then
            _rlt_prefix="${_rlt_m}/"
        fi
        if command -v curl >/dev/null 2>&1; then
            _rlt_json=$(curl -4 --http1.1 -sSL --connect-timeout 4 -m 8 "${_rlt_prefix}https://api.github.com/repos/zyeyefyass/CubeX/releases/latest" 2>/dev/null || true)
            _rlt_cand=$(echo "$_rlt_json" | grep -o '"tag_name": *"[^"]*"' | head -n1 | cut -d'"' -f4 | grep -E '^v[0-9]+' || true)
            if [ -n "$_rlt_cand" ]; then
                REL_TAG="$_rlt_cand"
                return 0
            fi
            _rlt_cand=$(curl -4 --http1.1 -sSL --connect-timeout 3 -m 5 -I "${_rlt_prefix}https://github.com/zyeyefyass/CubeX/releases/latest" 2>/dev/null | grep -i "^location:" | grep -oE 'tag/v[0-9][^/ \r\n]*' | cut -d/ -f2 | tr -d '\r\n' | grep -E '^v[0-9]+' || true)
            if [ -n "$_rlt_cand" ]; then
                REL_TAG="$_rlt_cand"
                return 0
            fi
        fi
    done
    [ -z "$REL_TAG" ] && REL_TAG="$CUBEX_DEFAULT_TAG"
}

cubex_download_with_mirrors() {
    _dwm_raw_url="$1"
    _dwm_dest="$2"
    _dwm_validator="$3"
    _dwm_max_time="${4:-180}"
    _dwm_show_prog="${5:-0}"
    _dwm_tmp="${_dwm_dest}.dl.$$"

    cubex_probe_fastest_mirror
    for _dwm_m in $(cubex_ordered_mirrors); do
        if [ "$_dwm_m" != "DIRECT" ]; then
            _dwm_link="${_dwm_m}/${_dwm_raw_url}"
            msg "${CYAN}      [镜像尝试] 正在连接: ${_dwm_m}...${NC}"
        else
            _dwm_link="${_dwm_raw_url}"
            msg "${CYAN}      [官方直连] 正在连接: GitHub...${NC}"
        fi
        rm -f "$_dwm_tmp" 2>/dev/null || true
        if command -v curl >/dev/null 2>&1; then
            if [ "$_dwm_show_prog" = "1" ]; then
                curl -4 --http1.1 -# -L --connect-timeout 8 -m "$_dwm_max_time" -o "$_dwm_tmp" "$_dwm_link" 2>&1 || true
            else
                curl -4 --http1.1 -fsSL --connect-timeout 6 -m "$_dwm_max_time" -o "$_dwm_tmp" "$_dwm_link" 2>/dev/null || true
            fi
        elif command -v wget >/dev/null 2>&1; then
            wget -4 -q -T 20 -O "$_dwm_tmp" "$_dwm_link" 2>/dev/null || true
        fi

        _dwm_valid=0
        if [ -n "$_dwm_validator" ]; then
            if "$_dwm_validator" "$_dwm_tmp"; then
                _dwm_valid=1
            fi
        elif [ -s "$_dwm_tmp" ]; then
            _dwm_valid=1
        fi

        if [ "$_dwm_valid" -eq 1 ]; then
            mv -f "$_dwm_tmp" "$_dwm_dest"
            msg "${GREEN}      ✓ 下载成功并通过完整性校验！${NC}"
            return 0
        fi
        rm -f "$_dwm_tmp" 2>/dev/null || true
    done
    return 1
}

detect_host_topology_and_platform() {
    DETECTED_HOST="linux"
    DETECTED_HOST_NAME="Linux 通用宿主环境"
    DEFAULT_INSTALL_DIR="/opt/cubex"

    if [ -f "/etc/fnos-release" ] || [ -d "/vol1" ] || grep -qi "fnos" /etc/os-release 2>/dev/null || [ -d "/usr/trim" ]; then
        DETECTED_HOST="fnos"
        DETECTED_HOST_NAME="飞牛 fnOS 专属 NAS 环境"
        DEFAULT_INSTALL_DIR="/vol1/1000/cubex"
    elif [ -n "${ROUTEROS:-}" ] || [ -d "/nova" ] || grep -qi "routeros" /etc/os-release 2>/dev/null; then
        DETECTED_HOST="routeros"
        DETECTED_HOST_NAME="MikroTik RouterOS v7.x 容器沙盒"
        DEFAULT_INSTALL_DIR="/app"
    elif [ -d "/etc/pve" ] || command -v pveversion >/dev/null 2>&1; then
        DETECTED_HOST="pve"
        DETECTED_HOST_NAME="Proxmox VE (PVE) 宿主/LXC 环境"
        DEFAULT_INSTALL_DIR="/opt/cubex"
    elif [ -f "/etc/openwrt_release" ]; then
        DETECTED_HOST="openwrt"
        DETECTED_HOST_NAME="OpenWrt 软路由环境"
        DEFAULT_INSTALL_DIR="/opt/cubex"
    fi

    DEFAULT_GW=$(ip route show default 2>/dev/null | awk '/default/ {print $3}' | head -n 1)
    IS_IKUAI_GATEWAY=0
    IS_IKUAI_HOST=0

    if [ -n "$DEFAULT_GW" ] && command -v curl >/dev/null 2>&1; then
        GW_PROBE=$(curl -s -m 2 -I "http://$DEFAULT_GW/" 2>/dev/null || true)
        GW_BODY=$(curl -s -m 2 "http://$DEFAULT_GW/" 2>/dev/null || true)
        if echo "$GW_PROBE $GW_BODY" | grep -qi "ikuai"; then
            IS_IKUAI_GATEWAY=1
        fi
    fi

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
}

apply_linux_netstack_tuning() {
    sysctl -w net.ipv4.ip_forward=1 >/dev/null 2>&1 || true
    sysctl -w net.ipv6.conf.all.forwarding=1 >/dev/null 2>&1 || true
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

    if modprobe tcp_bbr >/dev/null 2>&1; then
        sysctl -w net.core.default_qdisc=fq >/dev/null 2>&1 || true
        sysctl -w net.ipv4.tcp_congestion_control=bbr >/dev/null 2>&1 || true
    fi

    if [ "$IS_IKUAI_GATEWAY" -eq 1 ] || [ "$IS_IKUAI_HOST" -eq 1 ]; then
        msg "${PURPLE}🚀 [爱快专属优化] 检测到爱快环境，正在注入爱快旁路四项底层加固...${NC}"
        sysctl -w net.ipv4.conf.all.rp_filter=0 >/dev/null 2>&1 || true
        sysctl -w net.ipv4.conf.default.rp_filter=0 >/dev/null 2>&1 || true
        for _alnt_dev in /proc/sys/net/ipv4/conf/*; do
            [ -d "$_alnt_dev" ] && echo 0 > "$_alnt_dev/rp_filter" 2>/dev/null || true
        done
        sysctl -w net.ipv4.conf.all.arp_ignore=1 >/dev/null 2>&1 || true
        sysctl -w net.ipv4.conf.all.arp_announce=2 >/dev/null 2>&1 || true
        sysctl -w net.ipv4.conf.default.arp_ignore=1 >/dev/null 2>&1 || true
        sysctl -w net.ipv4.conf.default.arp_announce=2 >/dev/null 2>&1 || true
        sysctl -w net.ipv4.conf.all.send_redirects=0 >/dev/null 2>&1 || true
        sysctl -w net.ipv4.conf.default.send_redirects=0 >/dev/null 2>&1 || true
        sysctl -w net.ipv4.conf.all.accept_redirects=0 >/dev/null 2>&1 || true
        sysctl -w net.ipv4.conf.default.accept_redirects=0 >/dev/null 2>&1 || true
        if command -v iptables >/dev/null 2>&1; then
            iptables -t mangle -C POSTROUTING -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu 2>/dev/null || \
            iptables -t mangle -A POSTROUTING -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu 2>/dev/null || true
        fi
        msg "${GREEN}  ✓ 爱快非对称路由直通 (rp_filter=0) 已生效${NC}"
        msg "${GREEN}  ✓ 爱快 ARP 抑风暴与防漂移 (arp_ignore=1/arp_announce=2) 已就绪${NC}"
        msg "${GREEN}  ✓ 爱快 PPPoE 自动 MSS 钳制已加载 (解决网页与测速截断卡顿)${NC}"
        msg "${GREEN}  ✓ 爱快 ICMP 环路抑制已就绪${NC}"
    fi

    if [ ! -c /dev/net/tun ]; then
        mkdir -p /dev/net 2>/dev/null || true
        mknod /dev/net/tun c 10 200 2>/dev/null || true
        chmod 666 /dev/net/tun 2>/dev/null || true
    fi
    modprobe tun 2>/dev/null || true

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
}

check_resolv_self_loop() {
    RESOLV_LOOP_MATCHES=""
    RESOLV_SAFE_GW="${DEFAULT_GW:-223.5.5.5}"
    if [ "$RESOLV_SAFE_GW" = "$LOCAL_IP" ] || [ -z "$RESOLV_SAFE_GW" ]; then
        RESOLV_SAFE_GW="223.5.5.5"
    fi
    if [ -r /etc/resolv.conf ]; then
        HOST_ALL_IPS=$(ip -o addr show 2>/dev/null | awk '{print $4}' | cut -d/ -f1 | tr '\n' ' ')
        RESOLV_NAMESERVERS=$(awk '/^[[:space:]]*nameserver[[:space:]]+/ {print $2}' /etc/resolv.conf 2>/dev/null || true)
        for _crsl_ns in $RESOLV_NAMESERVERS; do
            _crsl_clean="${_crsl_ns%%#*}"
            _crsl_nozone="${_crsl_clean%%%*}"
            case "$_crsl_nozone" in
                127.*|::1|0.0.0.0|::|"$LOCAL_IP")
                    RESOLV_LOOP_MATCHES="${RESOLV_LOOP_MATCHES} ${_crsl_clean}"
                    ;;
                *)
                    for _crsl_hip in $HOST_ALL_IPS; do
                        if [ "$_crsl_nozone" = "$_crsl_hip" ]; then
                            RESOLV_LOOP_MATCHES="${RESOLV_LOOP_MATCHES} ${_crsl_clean}"
                            break
                        fi
                    done
                    ;;
            esac
        done
    fi
}
