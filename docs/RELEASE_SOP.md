# CubeX 正式发版硬门禁与标准作业规范 (Release SOP)

> 生效版本：自 `v9.98` 起永久生效  
> 适用范围：CubeX 全平台双核版本构建、隔离靶场验收、签审放行与多平台发布

---

## 一、五道发版硬门禁（任一不通过一票否决）

### 门禁 1：全量安装脚本无代理直连实测门禁
- 实测全套安装与初始化脚本（`install.sh`、`install_pve_lxc.sh`、`scripts/docker/entrypoint.sh`、`setup-openwrt.sh`、`install_ros_container.rsc`、`setup-routeros-v7.rsc`、`scripts/fnos/docker-compose.yml`）时，靶机**严禁注入** `http_proxy`、`https_proxy`、`ALL_PROXY` 等代理环境变量。
- 所有离线包直装、镜像校验、兜底解压与二次重跑幂等保留必须在纯无代理隔离环境下真实跑通，并在验收报告中诚实披露实际测试拓扑与网络链路。

### 门禁 2：三类宿主与三大主路由默认强制真机/虚机联调门禁
- 每轮发版前必须在与生产完全隔离的靶场网桥（严禁触碰生产旁路由与主路由）完成全矩阵实测：
  1. **宿主机/容器底座**：Debian（原生 systemd）、Alpine（PVE LXC 容器）、Docker / 飞牛 `fnOS`（统一 `/app/etc` 与 `/app/data` 挂载路径）；
  2. **三大主路由底座**：
     - **MikroTik RouterOS v7**（`cubex` 路由表、`198.18.0.0/15` 回指与黑洞兜底、Netwatch 毫秒逃生、旧表迁移与悬空 `*hex` Mangle 清理、对称撤回与 `.rsc` 脚本 SSH 导入验证）；
     - **爱快 iKuai 4.x**（Web-RPC 静态回指、DHCP 主 DNS 注入、漂移自愈与零残留撤回）；
     - **OpenWrt / iStoreOS 25.x**（UCI 静态回指、`nftables` 加速豁免/DNS 拦截、看门狗逃生恢复与零残留撤回）。
- 实测结束后必须清理测试节点与凭据并停机全部靶机（确认 `stopped`）。

### 门禁 3：双核单一真源、静态链接与零 `x86-64-v3/v4` 审计门禁
- `meta.json`、`Dockerfile`、`.github/workflows/docker-publish.yml` 与本地 `bin/amd64/`、`bin/arm64/` 必须严格对齐同一双核版本（`sing-box` + `mihomo`），确保 `sing-box` 为纯静态 musl 二进制且 `mihomo` amd64 采用 `-compatible-` 通用指令集。
- `scripts/audit_release_artifacts.py` 强制逆向解包全量发布归档、OCI 镜像层与独立二进制，确保：
  - 零 `x86-64-v3` / `v4` 微架构受限指令集残留；
  - 全部独立与嵌套 `cubexd` 二进制内嵌目标版本号 100% 一致；
  - 零订阅、零私有节点、零凭证、零敏感词与敏感图、零未授权 Windows 产物。

### 门禁 4：两段式签审与发版前代码健康全绿门禁
- 严格执行**“先隔离验收并呈报实测证据 → 获总架构师终审签字 → 再统一对齐版本号、打包与发布”**两段式流程；未获签字前严禁执行 `bump_version.py`、打 Git Tag 或推送 Release。
- 发版前本地门禁必须全部全绿通过：`go vet ./...`、`go test -count=1 ./...`、`python scripts/audit_codebase_health.py`（0 警告 0 阻断），并对新构建的 `.fpk` 套件包抽检内部挂载路径（必须为 `/app/etc` 与 `/app/data`）。

### 门禁 5：全平台发布一致性与 10 版本存留清算门禁
- GitHub Release 必须校验全部 19 项资产（含 `SHA256SUMS`）均为 `state: uploaded` 且字节与本地一致，正文采用纯中文简明说明（无重复主标题）并附完整资产清单表。
- Tag 推送后确认 GitHub Actions 多架构 Docker 镜像（`vX.YZ` 与 `latest`）构建推送成功，同步更新 Docker Hub 仓库说明页并回查 `PATCH/GET HTTP 200`（UTF-8 ≤ 25,000 字节）。
- 每次发版收尾强制运行 `scripts/enforce_10_version_retention.py`，确保 GitHub Releases、Git Tags、Docker Hub 与本地 `dist/release` 仅保留最新 10 个版本，并保持工作区 `git status` 洁净。
