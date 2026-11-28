#!/bin/sh
# CubeX iStoreOS / OpenWrt Clean Uninstaller (Strict POSIX sh / ash)
# Zero bashisms: no [[, no declare, no source, no local, no arrays.
set -e

PURGE_DATA=0
for _u_arg in "$@"; do
    case "$_u_arg" in
        --purge)
            PURGE_DATA=1
            ;;
        -h|--help)
            printf '%s\n' "Usage: sh uninstall.sh [--purge]"
            printf '%s\n' "  --purge   Also remove persistent config and data directories"
            exit 0
            ;;
    esac
done

if [ "$(id -u)" -ne 0 ]; then
    printf '%s\n' "Error: uninstall.sh must be run as root." >&2
    exit 1
fi

_SELF_DIR=$(cd "$(dirname "$0")" && pwd)
INSTALL_DIR="$_SELF_DIR"
MANIFEST_FILE="${INSTALL_DIR}/install.manifest"

if [ ! -f "$MANIFEST_FILE" ] && [ -f "/opt/cubex/install.manifest" ]; then
    INSTALL_DIR="/opt/cubex"
    MANIFEST_FILE="/opt/cubex/install.manifest"
fi

if [ -f "$MANIFEST_FILE" ]; then
    _M_DIR=$(awk -F= '/^INSTALL_DIR=/{print $2; exit}' "$MANIFEST_FILE" 2>/dev/null || true)
    [ -n "$_M_DIR" ] && INSTALL_DIR="$_M_DIR"
fi

if [ -z "$INSTALL_DIR" ] || [ "$INSTALL_DIR" = "/" ]; then
    printf '%s\n' "Error: refusing invalid INSTALL_DIR='$INSTALL_DIR'." >&2
    exit 1
fi

printf '%s\n' "[CubeX] Stopping and disabling procd service..."
if [ -x /etc/init.d/cubex ]; then
    /etc/init.d/cubex stop >/dev/null 2>&1 || true
    /etc/init.d/cubex disable >/dev/null 2>&1 || true
fi
rm -f /etc/init.d/cubex /etc/rc.d/S*cubex /etc/rc.d/K*cubex 2>/dev/null || true

pkill -x cubexd >/dev/null 2>&1 || killall cubexd >/dev/null 2>&1 || true
pkill -x mihomo >/dev/null 2>&1 || killall mihomo >/dev/null 2>&1 || true
pkill -x sing-box >/dev/null 2>&1 || killall sing-box >/dev/null 2>&1 || true

printf '%s\n' "[CubeX] Removing LuCI menus, ACLs, and installed artifacts..."
if [ -f "$MANIFEST_FILE" ]; then
    while IFS= read -r _m_line || [ -n "$_m_line" ]; do
        case "$_m_line" in
            FILE=*)
                _m_path="${_m_line#FILE=}"
                [ -n "$_m_path" ] && rm -f "$_m_path" 2>/dev/null || true
                ;;
        esac
    done < "$MANIFEST_FILE"

    while IFS= read -r _m_line || [ -n "$_m_line" ]; do
        case "$_m_line" in
            DIR=*)
                _m_dir="${_m_line#DIR=}"
                [ -n "$_m_dir" ] && rmdir "$_m_dir" 2>/dev/null || true
                ;;
        esac
    done < "$MANIFEST_FILE"
fi

rm -f /usr/share/luci/menu.d/luci-app-cubex.json \
      /usr/share/rpcd/acl.d/luci-app-cubex.json \
      /www/luci-static/resources/view/cubex/dashboard.js \
      /usr/bin/cubex \
      /usr/local/bin/cubex \
      /etc/sysctl.d/99-cubex.conf \
      /etc/nftables.d/10-cubex-offload-exempt.nft 2>/dev/null || true
rmdir /www/luci-static/resources/view/cubex 2>/dev/null || true

printf '%s\n' "[CubeX] Restoring network and firewall rules..."
if command -v uci >/dev/null 2>&1; then
    _UCI_FW_CHANGED=0
    for _fw_key in cubex_bypass cubex_dnat cubex_hairpin cubex_dns_redir_udp cubex_dns_redir_tcp cubex_tproxy; do
        if uci -q get "firewall.${_fw_key}" >/dev/null 2>&1; then
            uci -q delete "firewall.${_fw_key}" || true
            _UCI_FW_CHANGED=1
        fi
    done
    if [ "$_UCI_FW_CHANGED" -eq 1 ]; then
        uci commit firewall >/dev/null 2>&1 || true
        /etc/init.d/firewall reload >/dev/null 2>&1 || true
    fi

    _UCI_NET_CHANGED=0
    for _net_key in cubex_fakeip cubex_shim cubex_shim_route; do
        if uci -q get "network.${_net_key}" >/dev/null 2>&1; then
            uci -q delete "network.${_net_key}" || true
            _UCI_NET_CHANGED=1
        fi
    done
    if [ "$_UCI_NET_CHANGED" -eq 1 ]; then
        uci commit network >/dev/null 2>&1 || true
        /etc/init.d/network reload >/dev/null 2>&1 || true
    fi
fi

if command -v ip >/dev/null 2>&1; then
    ip route del 198.18.0.0/15 2>/dev/null || true
fi

if command -v nft >/dev/null 2>&1; then
    nft delete chain inet fw4 cubex_flow_exempt 2>/dev/null || true
fi

if command -v iptables >/dev/null 2>&1; then
    iptables -D FORWARD -d 198.18.0.0/15 -j ACCEPT 2>/dev/null || true
    iptables -D FORWARD -s 198.18.0.0/15 -j ACCEPT 2>/dev/null || true
fi

if command -v sysctl >/dev/null 2>&1; then
    sysctl -p >/dev/null 2>&1 || true
fi

printf '%s\n' "[CubeX] Cleaning /etc/sysupgrade.conf..."
if [ -f /etc/sysupgrade.conf ]; then
    _SYSUP_TMP="/tmp/sysupgrade.conf.cubex.$$"
    awk '
        /^# BEGIN CUBEX$/ { skip=1; next }
        /^# END CUBEX$/ { skip=0; next }
        skip { next }
        { print }
    ' /etc/sysupgrade.conf | grep -v -E "(${INSTALL_DIR}|luci-app-cubex|/etc/init\.d/cubex|/etc/sysctl\.d/99-cubex\.conf|/etc/nftables\.d/10-cubex)" > "$_SYSUP_TMP" 2>/dev/null || true
    mv -f "$_SYSUP_TMP" /etc/sysupgrade.conf
fi

rm -rf /tmp/luci-indexcache* /tmp/luci-modulecache/* 2>/dev/null || true
if [ -x /etc/init.d/rpcd ]; then
    /etc/init.d/rpcd reload >/dev/null 2>&1 || true
fi
if [ -x /etc/init.d/uhttpd ]; then
    /etc/init.d/uhttpd reload >/dev/null 2>&1 || true
fi

rm -rf "${INSTALL_DIR}/bin" 2>/dev/null || true
rm -f "${INSTALL_DIR}/meta.json" "${INSTALL_DIR}/install.manifest" "${INSTALL_DIR}/uninstall.sh" 2>/dev/null || true

if [ "$PURGE_DATA" -eq 1 ]; then
    rm -rf "${INSTALL_DIR}/data" "${INSTALL_DIR}/etc" 2>/dev/null || true
    rmdir "${INSTALL_DIR}" 2>/dev/null || true
    printf '%s\n' "[CubeX] Uninstall complete (including persistent data purge)."
else
    rmdir "${INSTALL_DIR}" 2>/dev/null || true
    if [ -d "${INSTALL_DIR}" ]; then
        printf '%s\n' "[CubeX] Uninstall complete. Persistent config/data preserved in ${INSTALL_DIR} (pass --purge to delete)."
    else
        printf '%s\n' "[CubeX] Uninstall complete."
    fi
fi
