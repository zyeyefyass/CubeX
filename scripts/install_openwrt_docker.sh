#!/bin/sh
# ═════════════════════════════════════════════════════════════════════════════
# 🌌 CubeX 官方 OpenWrt / iStoreOS Docker 单机寄生自闭环一键安装脚本
# 文件名: install_openwrt_docker.sh
# 适用平台: OpenWrt 21.02+ / 22.03+ / 23.05+ / 24.x / iStoreOS (x86_64, aarch64, arm)
# ═════════════════════════════════════════════════════════════════════════════

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
if [ -f "${SCRIPT_DIR}/openwrt/setup-openwrt.sh" ]; then
    exec sh "${SCRIPT_DIR}/openwrt/setup-openwrt.sh" "$@"
fi
if [ -f "/tmp/setup-openwrt.sh" ]; then
    exec sh "/tmp/setup-openwrt.sh" "$@"
fi

# 在线拉取官方最新 setup-openwrt.sh 并执行 (加速镜像优先 + 头部校验防 502 HTML 污染)
TMP_SETUP="/tmp/setup-openwrt.sh"
DL_SETUP="/tmp/setup-openwrt.sh.dl.$$"
for MIRROR in "https://ghfast.top" "https://ghproxy.net" "https://gh-proxy.com" "https://gh.ddlc.top" ""; do
    URL="${MIRROR}/https://raw.githubusercontent.com/zyeyefyass/CubeX/main/scripts/openwrt/setup-openwrt.sh"
    if [ -z "$MIRROR" ]; then
        URL="https://raw.githubusercontent.com/zyeyefyass/CubeX/main/scripts/openwrt/setup-openwrt.sh"
    fi
    rm -f "$DL_SETUP"
    if command -v curl >/dev/null 2>&1; then
        curl -4 -fsSL --connect-timeout 5 -m 15 -o "$DL_SETUP" "$URL" 2>/dev/null || true
    elif command -v wget >/dev/null 2>&1; then
        wget -q -T 15 -O "$DL_SETUP" "$URL" 2>/dev/null || true
    fi
    if [ -s "$DL_SETUP" ] && head -n 1 "$DL_SETUP" 2>/dev/null | grep -q "^#!/bin/sh"; then
        mv -f "$DL_SETUP" "$TMP_SETUP"
        break
    fi
    rm -f "$DL_SETUP"
done

if [ -s "$TMP_SETUP" ] && head -n 1 "$TMP_SETUP" 2>/dev/null | grep -q "^#!/bin/sh"; then
    chmod +x "$TMP_SETUP"
    exec sh "$TMP_SETUP" "$@"
else
    echo "❌ 无法获取有效的 setup-openwrt.sh，请检查网络连接！"
    exit 1
fi
