#!/bin/sh
set -e

# Ensure glibc dynamic linker compatibility for sing-box on musl/Alpine
mkdir -p /lib64 2>/dev/null || true
if [ ! -e /lib64/ld-linux-x86-64.so.2 ] && [ -f /lib/ld-musl-x86_64.so.1 ]; then
    ln -s /lib/ld-musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2 2>/dev/null || true
fi
if [ ! -e /lib64/ld-linux-aarch64.so.1 ] && [ -f /lib/ld-musl-aarch64.so.1 ]; then
    ln -s /lib/ld-musl-aarch64.so.1 /lib64/ld-linux-aarch64.so.1 2>/dev/null || true
fi

echo "🚀 [CubeX] Starting CubeX Container Initialization..."

# 1. Initialize /app/etc if empty or missing critical geodata / rules
if [ ! -f "/app/etc/config.yaml" ] && [ ! -f "/app/etc/config.json" ]; then
    echo "📦 [CubeX] Initializing /app/etc from defaults..."
    cp -rn /app/defaults/etc/* /app/etc/ 2>/dev/null || cp -r /app/defaults/etc/* /app/etc/
fi

# Always ensure rules and geodata exist in /app/etc
mkdir -p /app/etc/rules
if [ -d "/app/defaults/etc/rules" ]; then
    echo "📦 [CubeX] Syncing dual-engine rulesets (MRS / SRS) to /app/etc/rules..."
    cp -rn /app/defaults/etc/rules/* /app/etc/rules/ 2>/dev/null || cp -r /app/defaults/etc/rules/* /app/etc/rules/ 2>/dev/null || true
fi

if [ ! -f "/app/etc/geoip.metadb" ] && [ -f "/app/defaults/etc/geoip.metadb" ]; then
    cp /app/defaults/etc/geoip.metadb /app/etc/geoip.metadb
fi
if [ ! -f "/app/etc/geosite.dat" ] && [ -f "/app/defaults/etc/geosite.dat" ]; then
    cp /app/defaults/etc/geosite.dat /app/etc/geosite.dat
fi
if [ ! -f "/app/etc/GeoIP.dat" ] && [ -f "/app/defaults/etc/GeoIP.dat" ]; then
    cp /app/defaults/etc/GeoIP.dat /app/etc/GeoIP.dat
fi

# 2. Initialize /app/data if empty
if [ ! -f "/app/data/config.ir.json" ]; then
    echo "📦 [CubeX] Initializing /app/data from defaults..."
    cp -rn /app/defaults/data/* /app/data/ 2>/dev/null || cp -r /app/defaults/data/* /app/data/
fi

# 3. Enable IP Forwarding and NAT Masquerade for Gateway / Secondary Router Mode
sysctl -w net.ipv4.ip_forward=1 >/dev/null 2>&1 || ( [ -w /proc/sys/net/ipv4/ip_forward ] && echo 1 > /proc/sys/net/ipv4/ip_forward 2>/dev/null ) || true
iptables -P FORWARD ACCEPT 2>/dev/null || true
# 3.1 Fast-Path PREROUTING Bypass for Local & Private Subnets (Minimizes CPU on fnOS / Docker / PVE)
iptables -t mangle -C PREROUTING -d 10.0.0.0/8 -j RETURN 2>/dev/null || iptables -t mangle -I PREROUTING 1 -d 10.0.0.0/8 -j RETURN 2>/dev/null || true
iptables -t mangle -C PREROUTING -d 172.16.0.0/12 -j RETURN 2>/dev/null || iptables -t mangle -I PREROUTING 2 -d 172.16.0.0/12 -j RETURN 2>/dev/null || true
iptables -t mangle -C PREROUTING -d 192.168.0.0/16 -j RETURN 2>/dev/null || iptables -t mangle -I PREROUTING 3 -d 192.168.0.0/16 -j RETURN 2>/dev/null || true
iptables -t mangle -C PREROUTING -d 127.0.0.0/8 -j RETURN 2>/dev/null || iptables -t mangle -I PREROUTING 4 -d 127.0.0.0/8 -j RETURN 2>/dev/null || true
iptables -t mangle -C PREROUTING -d 224.0.0.0/4 -j RETURN 2>/dev/null || iptables -t mangle -I PREROUTING 5 -d 224.0.0.0/4 -j RETURN 2>/dev/null || true

iptables -t nat -A POSTROUTING -j MASQUERADE 2>/dev/null || true

# Asymmetric bypass routing, ARP storm defense & ICMP redirect suppression (Essential for iKuai / ROS / PVE / fnOS)
sysctl -w net.ipv4.conf.all.rp_filter=0 2>/dev/null || true
sysctl -w net.ipv4.conf.default.rp_filter=0 2>/dev/null || true
sysctl -w net.ipv4.conf.all.arp_ignore=1 2>/dev/null || true
sysctl -w net.ipv4.conf.all.arp_announce=2 2>/dev/null || true
sysctl -w net.ipv4.conf.default.arp_ignore=1 2>/dev/null || true
sysctl -w net.ipv4.conf.default.arp_announce=2 2>/dev/null || true
sysctl -w net.ipv4.conf.all.send_redirects=0 2>/dev/null || true
sysctl -w net.ipv4.conf.default.send_redirects=0 2>/dev/null || true

# PPPoE WAN MTU 1492 TCP MSS Clamping (Prevents fragmentation packet drops on iKuai & general WANs)
iptables -t mangle -C POSTROUTING -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu 2>/dev/null || \
iptables -t mangle -A POSTROUTING -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu 2>/dev/null || true

# High-Performance TCP Socket & Queue Tuning (Universal Dual-Engine Line-Speed Base: Mihomo × sing-box)
if [ -d /sys/class/net/eth0 ]; then
    ip link set eth0 txqueuelen 10000 2>/dev/null || true
fi
sysctl -w net.core.somaxconn=16384 2>/dev/null || true
sysctl -w net.core.netdev_max_backlog=32768 2>/dev/null || true
sysctl -w net.ipv4.ip_local_port_range="10240 65535" 2>/dev/null || true
sysctl -w net.ipv4.tcp_rmem="16384 1048576 33554432" 2>/dev/null || true
sysctl -w net.ipv4.tcp_wmem="16384 1048576 33554432" 2>/dev/null || true
sysctl -w net.ipv4.tcp_window_scaling=1 2>/dev/null || true
sysctl -w net.ipv4.tcp_timestamps=1 2>/dev/null || true
sysctl -w net.ipv4.tcp_sack=1 2>/dev/null || true
sysctl -w net.ipv4.tcp_fastopen=3 2>/dev/null || true
sysctl -w net.ipv4.tcp_slow_start_after_idle=0 2>/dev/null || true
sysctl -w net.ipv4.tcp_max_syn_backlog=16384 2>/dev/null || true
sysctl -w net.ipv4.tcp_max_tw_buckets=1440000 2>/dev/null || true
sysctl -w net.ipv4.tcp_fin_timeout=10 2>/dev/null || true
sysctl -w net.ipv4.tcp_tw_reuse=1 2>/dev/null || true

# Probe BBR if supported
if modprobe tcp_bbr >/dev/null 2>&1; then
    sysctl -w net.core.default_qdisc=fq >/dev/null 2>&1 || true
    sysctl -w net.ipv4.tcp_congestion_control=bbr >/dev/null 2>&1 || true
fi

# 4. Set working directory and listen port
LISTEN_PORT="${CUBEX_PORT:-2026}"
echo "🌟 [CubeX] Listening on 0.0.0.0:${LISTEN_PORT} (Dual-Engine: Mihomo + sing-box ready)"

chmod +x /app/bin/* 2>/dev/null || true
cd /app
while true; do
    /app/bin/cubexd -dir /app -listen ":${LISTEN_PORT}" "$@" || true
    echo "🔄 [CubeX] Daemon exited. Respawning in 1 second..."
    sleep 1
done
