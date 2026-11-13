# 🌌 CubeX v9.0 LTS 保姆级快速上手指引 (Quickstart Guide)

感谢关注 **CubeX · 宿主共生零感网络系统**！为了让你在 **5 分钟内** 完成从部署到全屋无感分流（国内流量主路由满血直出 + 国际业务协同流量双擎接管 + 容器断电秒级逃生），请按以下三步操作：

---

## 第一步：理解 CubeX 的「宿主共生」网络哲学（为什么不改网关？）

传统旁路由要求把局域网所有设备的默认网关改成旁路由 IP，一旦旁路由重启或死机，全屋立刻断网，且国内大流量也要绕道旁路由白白消耗 CPU。

**CubeX 采用「主路由共生 + Fake-IP 回程劫持」架构：**
1. **局域网设备（手机/电脑/电视）的默认网关与 DNS 保持指向你的主路由（如 `10.0.0.1` 或 `192.168.1.1`）完全不变！**
2. 主路由将上游 DNS 指向 CubeX（例如 `10.0.0.7`）：
   - 当访问**国内常用网络**时，CubeX 返回真实国内 IP，流量直接由主路由硬件交换芯片转发直出（**0 绕路、0 延迟损耗**）；
   - 当访问**国际跨境业务协同网络**时，CubeX 瞬间返回 `198.18.0.0/15` 网段的 Fake-IP；
3. 主路由只需配置**一条静态路由**：把目标为 `198.18.0.0/15` 的流量下一跳指给 CubeX（`10.0.0.7`），即可实现丝滑接管！

---

## 第二步：按你的硬件底座一键部署 CubeX

假设你的主路由网段为 `10.0.0.1/24`，分配给 CubeX 的静态 IP 为 `10.0.0.7`（请按你实际内网网段替换）：

### 方案 A：PVE LXC 容器部署（极力推荐 · 稳态内存仅 ~35MB）
1. 在 PVE 创建 Debian 12 / Alpine LXC 容器（1核 CPU / 128M~256M 内存即可），IP 设为 `10.0.0.7/24`，网关 `10.0.0.1`。
2. **关键一步（开启 TUN 设备透传）**：在 PVE 宿主机终端执行（将 `105` 替换为你的容器 CTID）：
   ```bash
   cat >> /etc/pve/lxc/105.conf << 'EOF'
   lxc.cgroup2.devices.allow: c 10:200 rwm
   lxc.mount.entry: /dev/net/tun dev/net/tun none bind,create=file
   EOF
   pct reboot 105
   ```
3. 进入容器拉取并启动 `cubexd` 单二进制守护进程：
   ```bash
   mkdir -p /var/lib/cubex && cd /var/lib/cubex
   curl -L -o cubexd https://github.com/zyeyefyass/CubeX/releases/latest/download/cubexd-linux-amd64
   chmod +x cubexd
   CUBEX_HOST_IP="10.0.0.7" ./cubexd
   ```

### 方案 B：OpenWrt / iStoreOS / 飞牛 fnOS / 群晖 (Docker 一键部署)
在宿主机终端执行以下命令（使用 `host` 网络模式或 `macvlan`）：
```bash
docker run -d --name cubex \
  --restart always \
  --network host \
  --cap-add=NET_ADMIN \
  --cap-add=NET_RAW \
  --device /dev/net/tun:/dev/net/tun \
  -e CUBEX_HOST_IP="10.0.0.7" \
  -e CUBEX_CONTROL_ADDR=":2026" \
  -e CUBEX_MEMORY_PROFILE="ultra-light" \
  -v /etc/cubex:/var/lib/cubex \
  ghcr.io/zyeyefyass/cubex:latest
```

### 方案 C：RouterOS v7 原生容器 (RB5009 / CCR2004 / x86)
1. 在 RouterOS 创建 `veth-cubex` 接口（IP `10.0.0.7/24`，网关 `10.0.0.1`）并加入 `bridge`。
2. 拉取 `arm64` 或 `amd64` 镜像启动容器，挂载 `/var/lib/cubex` 持久化目录。

---

## 第三步：30 秒完成面板点火与主路由共生绑定

1. **打开 WebUI 控制台**：
   浏览器访问 `http://10.0.0.7:2026`（进入疯狂动物城 120FPS 流光控制台，支持按 `Ctrl + K` 呼出极客指令舱）。
2. **导入订阅并选择引擎**：
   在「规则与订阅」填入你的合规国际业务协同网络节点配置链接，点击更新；顶部可随时一键无感热切 `Mihomo` 或 `sing-box v1.14+`（切换耗时小于 65ms）。
3. **主路由共生绑定与毫秒级逃生（以 RouterOS v7 为例）**：
   直接在面板点击「生成 ROS 共生脚本」，或在 RouterOS 终端粘贴以下 3 行核心配置（将 `10.0.0.7` 替换为你的 CubeX IP）：
   ```routeros
   # 1. 将 Fake-IP 流量导向 CubeX 隧道（国内真实 IP 自动走主路由满血直出）
   /ip route add dst-address=198.18.0.0/15 gateway=10.0.0.7 distance=1 comment="CubeX-FakeIP-Backtrack"

   # 2. 主路由 DNS 指向 CubeX 零泄露 DNS 端口
   /ip dns set servers=10.0.0.7 allow-remote-requests=yes
   /ip dns cache flush

   # 3. 启用 Netwatch 看门狗（CubeX 面板内支持一键下发带 2.2s 防抖的完整逃生脚本）
   ```
   *(注：若主路由为 **OpenWrt / 爱快 iKuai**，只需在主路由添加一条静态路由：目标网络 `198.18.0.0/255.254.0.0` 网关填 `10.0.0.7`，并将主路由 DHCP/上游 DNS 设为 `10.0.0.7` 即可！)*

---

## 🛠️ 常见排障与极客快捷键

- **`Ctrl + K` / `Cmd + K`**：在 WebUI 任意界面呼出「极客瞬态指令舱」，一键执行双核热切、开启/退出「旁路维护模式（不断网停机）」、导出脱敏诊断包。
- **环境自检接口**：访问 `http://10.0.0.7:2026/api/v1/onboarding/doctor` 可查看当前容器的 `/dev/net/tun`、`CAP_NET_ADMIN` 与 `nftables` 内核快转状态。
