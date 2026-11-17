# 【飞牛 fnOS 专属福利】PT 满速下载与 TMDB 海报刮削完美兼得！影音/NAS 专属旁路分流解法

> **标签**：`#飞牛fnOS` `#NAS` `#家庭影音` `#PT下载` `#TMDB刮削` `#Docker`  
> **项目地址**：https://github.com/zyeyefyass/CubeX  
> **适用底座**：飞牛 fnOS / 群晖 DSM / 极空间 / 威联通 / 通用 Linux Docker

---

### 一、 NAS 与影音玩家最纠结的“痛点”

各位玩飞牛 fnOS 或群晖的大佬，平时肯定遇到过这个尴尬难题：
1. **如果不给 NAS 挂旁路由**：Emby / Jellyfin / Plex 刮削海报墙极其缓慢甚至满屏裂图，SubHD 字幕下载失败，Docker Hub 拉取镜像三天两头超时报错；
2. **如果把 NAS 网关改成旁路由**：qBittorrent / Transmission 跑 BT/PT 下载时，几百 G 甚至几个 T 的大流量一不小心全走到外部加速通道上，不仅非必要外部带宽损耗严重，节点流量秒空，还可能因为滥用协议被封号！

难道就不能既让海报墙与 Docker 极速秒拉，又让 PT 下载 100% 走国内千兆物理宽带满血直连吗？

**CubeX v9.9.0 纯净稳定版正式推出了「🎬 影音/NAS 专属分流模式」！**

---

### 二、 影音/NAS 专属分流模式是怎么工作的？

在 CubeX 局域网设备控制台里，找到你的飞牛 NAS 或客厅蓝光播放机，点击切换为 **【🎬 影音专属模式】**，系统将自动加载双核复合智能编排：

- 🚀 **海报墙与元数据极速刮削**：`api.themoviedb.org`、`thetvdb.com`、`fanart.tv`、`subhd.tv` 以及 Docker 官方源，全自动走低延迟国际加速通道，海报秒出、简介秒加载、封面瀑布流丝滑刷出；
- 🏠 **PT/BT 大流量 100% 物理直连**：所有 Tracker 宣告、P2P 节点握手与大流量数据传输，100% 走局域网与运营商物理网关直接满血直通，跑满你的 2.5G/万兆外网带宽，彻底杜绝任何外部流量损耗！
- 📺 **客厅蓝光机/电视无缝体验**：电视机即便播放 4K 原盘，内网数据流原地直通 NAS，主路由 CPU 占用仅 1%，客厅看电视永远零卡顿！

---

### 三、 飞牛 fnOS 极速 2 步部署（Docker Compose）

CubeX 为飞牛 fnOS 用户量身定制了 Host 模式部署模板，零繁琐配置：

#### 步骤 1：在飞牛新建 `docker-compose.yml`
```yaml
version: "3.8"

services:
  cubex:
    image: fyasshole/cubex:latest
    container_name: cubex
    restart: unless-stopped
    network_mode: host
    privileged: true
    cap_add:
      - NET_ADMIN
      - NET_RAW
    devices:
      - /dev/net/tun:/dev/net/tun
    volumes:
      - /vol1/1000/docker/cubex/data:/app/data
      - /vol1/1000/docker/cubex/etc:/app/etc
      - /etc/localtime:/etc/localtime:ro
    environment:
      - TZ=Asia/Shanghai
      - CUBEX_EDITION=docker
```
*(注意：请将 `/vol1/1000/docker/cubex` 替换为您飞牛系统真实的存储路径)*

#### 步骤 2：启动并进入后台
启动后，在浏览器直接打开：`http://飞牛IP:2026` 即可看到高质感的 Twilight Crystal 暮光冰晶控制台！

---

### 四、 主路由加一条回指，全家立享

只需在您的主路由（RouterOS / 爱快 / OpenWrt）静态路由里加一条：
- **目标网段**：`198.18.0.0/15`
- **网关**：填写飞牛 NAS 的内网 IP

全家手机、电视、电脑**完全保持系统默认 DHCP（网关和 DNS 指向主路由）**，什么都不用改，新设备进门连上 Wi-Fi 自动享受分流与去广告，全家永不断网！

---

👉 **项目开源地址**：https://github.com/zyeyefyass/CubeX  
如果觉得这个功能拯救了你的 NAS，别忘了点个 Star 支持一下！
