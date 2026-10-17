#!/bin/sh
set -e

echo "🚀 [CubeX] Starting CubeX Container Initialization..."

# 1. Initialize /app/etc if empty or missing critical geodata / rules
if [ ! -f "/app/etc/config.yaml" ] && [ ! -f "/app/etc/config.json" ]; then
    echo "📦 [CubeX] Initializing /app/etc from defaults..."
    cp -rn /app/defaults/etc/* /app/etc/ 2>/dev/null || cp -r /app/defaults/etc/* /app/etc/
fi

# Always ensure rules and geodata exist in /app/etc
if [ ! -d "/app/etc/rules" ]; then
    echo "📦 [CubeX] Syncing dual-engine rulesets (MRS / SRS) to /app/etc/rules..."
    mkdir -p /app/etc/rules
    cp -r /app/defaults/etc/rules/* /app/etc/rules/ 2>/dev/null || true
fi

if [ ! -f "/app/etc/geoip.metadb" ] && [ -f "/app/defaults/etc/geoip.metadb" ]; then
    cp /app/defaults/etc/geoip.metadb /app/etc/geoip.metadb
fi
if [ ! -f "/app/etc/geosite.dat" ] && [ -f "/app/defaults/etc/geosite.dat" ]; then
    cp /app/defaults/etc/geosite.dat /app/etc/geosite.dat
fi

# 2. Initialize /app/data if empty
if [ ! -f "/app/data/config.ir.json" ]; then
    echo "📦 [CubeX] Initializing /app/data from defaults..."
    cp -rn /app/defaults/data/* /app/data/ 2>/dev/null || cp -r /app/defaults/data/* /app/data/
fi

# 3. Enable IP Forwarding and NAT Masquerade for Gateway / Secondary Router Mode
if [ -w /proc/sys/net/ipv4/ip_forward ]; then
    echo 1 > /proc/sys/net/ipv4/ip_forward 2>/dev/null || true
fi
# Enable SNAT Masquerade for any forwarded traffic exiting container (veth-cubex / eth0 / etc.)
iptables -t nat -A POSTROUTING -j MASQUERADE 2>/dev/null || true

# 4. Set working directory and listen port
LISTEN_PORT="${CUBEX_PORT:-2026}"
echo "🌟 [CubeX] Listening on 0.0.0.0:${LISTEN_PORT} (Dual-Engine: Mihomo + sing-box ready)"

cd /app
while true; do
    /app/bin/cubexd -dir /app -listen ":${LISTEN_PORT}" "$@" || true
    echo "🔄 [CubeX] Daemon exited. Respawning in 1 second..."
    sleep 1
done
