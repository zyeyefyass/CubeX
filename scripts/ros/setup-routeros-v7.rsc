# ═════════════════════════════════════════════════════════════════════════════
# 🌌 CubeX 官方 MikroTik RouterOS v7 容器一键原生协同与对齐脚本
# 适用平台: RouterOS v7.12+ (x86 / CHR / RB5009 / CCR2004 / CCR2116 / RB4011)
# ═════════════════════════════════════════════════════════════════════════════

:log info "🌌 开始自动配置 CubeX 次世代网关 RouterOS 原生联动..."

# 1. 创建专用 VETH 虚拟网卡 (若不存在)
/interface veth
:if ([:len [find name="veth-cubex"]] = 0) do={
    add name=veth-cubex address=192.168.88.240/24 gateway=192.168.88.1
    :log info "✅ 已创建 VETH 网卡: veth-cubex (IP: 192.168.88.240/24)"
}

# 2. 将 VETH 桥接到默认内网 Bridge
/interface bridge port
:if ([:len [find interface="veth-cubex"]] = 0) do={
    add bridge=bridge interface=veth-cubex
    :log info "✅ 已将 veth-cubex 挂载至主网桥 bridge"
}

# 3. 创建独立策略路由表 (table 8228 · 防死锁)
/routing table
:if ([:len [find name="table-cubex-8228"]] = 0) do={
    add fib name=table-cubex-8228
    :log info "✅ 已创建隔离策略路由表: table-cubex-8228"
}

# 4. 默认策略路由指向 CubeX 旁路网关
/ip route
:if ([:len [find routing-table=table-cubex-8228]] = 0) do={
    add dst-address=0.0.0.0/0 gateway=192.168.88.240 routing-table=table-cubex-8228 check-gateway=ping comment="CubeX-Default-Gateway"
    :log info "✅ 已添加默认策略路由条目 ➔ 192.168.88.240 (带 ping 自愈检测)"
}

# 5. 防死锁与局域网直连豁免 (Mangle 规则)
/ip firewall mangle
:if ([:len [find comment="CubeX-Bypass-Local"]] = 0) do={
    add chain=prerouting src-address=192.168.88.240 action=accept comment="CubeX-Bypass-Local"
    add chain=prerouting dst-address=192.168.88.0/24 action=accept comment="CubeX-Bypass-LAN"
    :log info "✅ 已添加 Mangle 防回环与本地局域网豁免规则"
}

# 6. 容器存储与拉取目录配置 (/container/config)
/container config
set ram-high=256M registry-url=https://registry-1.docker.io tmpdir=disk1/pull

:log info "🎉 RouterOS v7 与 CubeX 硬件级联动基础配置已全部完成！"
:log info "💡 离线安装: 将 cubex-ros-amd64.tar (或 arm64) 上传至 Files，随后执行:"
:log info "   /container/add file=cubex-ros-amd64.tar interface=veth-cubex root-dir=disk1/cubex start-on-boot=yes"
