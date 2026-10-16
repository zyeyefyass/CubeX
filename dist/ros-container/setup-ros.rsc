# ════════════════════════════════════════════════════════════════════
# 🌌 CubeX RouterOS v7 Container (VETH) 自动装配脚本
# ════════════════════════════════════════════════════════════════════

# 1. 创建容器网桥与 VETH 接口
/interface bridge add name=dockers-br comment="CubeX Container Bridge"
/ip address add address=172.17.0.1/24 interface=dockers-br comment="CubeX Gateway"
/interface veth add name=veth-cubex address=172.17.0.2/24 gateway=172.17.0.1 comment="CubeX VETH"
/interface bridge port add bridge=dockers-br interface=veth-cubex

# 2. 挂载持久化存储目录
/container mounts add name=cubex-data src=/usb1/cubex-data dst=/opt/CubeX/data
/container mounts add name=cubex-etc src=/usb1/cubex-etc dst=/opt/CubeX/etc

# 3. 创建并启动 CubeX 容器
/container config set ram-high=90M registry-url=https://registry-1.docker.io tmpdir=/usb1/pull
/container add file=cubex-v7.tar interface=veth-cubex mounts=cubex-data,cubex-etc hostname=CubeX start-on-boot=yes comment="CubeX Dual-Engine Gateway"
/container start [find comment="CubeX Dual-Engine Gateway"]
