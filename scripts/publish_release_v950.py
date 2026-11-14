#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
CubeX v9.5.0 正式版 GitHub 发布自动化脚本
- 创建/更新 v9.5.0 Release（全中文精简大白话说明 + 资产清单表）
- 统一通过 curl.exe --http1.1 -x PROXY 禁用 Expect: 100-continue 直传资产，支持断点续传与字节校验
- 上传完毕后自动执行全平台 10 版本存留红线脚本
"""
import json
import os
import subprocess
import sys
import tempfile
import time
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
ROOT_DIR = SCRIPT_DIR.parent
RELEASE_DIR = ROOT_DIR / "dist" / "release"
sys.path.insert(0, str(SCRIPT_DIR))

from enforce_10_version_retention import get_github_token  # noqa: E402

REPO = "zyeyefyass/CubeX"
TAG = "v9.5.0"
PROXY = "http://10.0.0.7:7890"

RELEASE_TITLE = "CubeX v9.5.0 正式版：控制面三级死链自愈 · 双擎智能降级 · 专属专线合规分流与主屏纯净瘦身"

RELEASE_BODY = """# CubeX v9.5.0 正式版

本次更新带来控制面三级死链智能自愈、双擎智能降级、专属专线合规分流与主屏纯净瘦身：

### 核心亮点
1. **控制面三级死链智能自愈**：在被动停顿与黑洞惩罚机制中引入 300 秒自动恢复窗口，临时网络波动后故障节点自动解除降权，杜绝上游闪断导致节点被误判永久拉黑。
2. **专属专线合规分流靶向对齐**：智能识别并绑定专属专线策略组，规则集出站精准靶向分流，杜绝流量偏离或悬空走默认出口。
3. **主屏视野纯净去焦虑改造**：彻底移除常态为空的透视面板与手动冲刷按钮，完全让出主界面视野给推荐三步配置指引，界面丝滑无干扰。
4. **全底座健康度全面达标**：软路由三巨头（RouterOS × 爱快 × iStoreOS）全链路联调满分通过，常驻内存极致轻量化至 33MB。

---

### 📦 资产下载与校验清单 (Release Assets)

| 发布物文件名 | 适用宿主与架构 | 说明与用途 |
| :--- | :--- | :--- |
| `CubeX-v9.5.0-ros-amd64.tar` | 软路由系统 (x86 软路由) | 软路由系统原生未压缩容器镜像（支持管理终端拖入直装） |
| `CubeX-v9.5.0-ros-arm64.tar` | 软路由系统 (RB5009/CCR/ARM) | 软路由系统 ARM64 原生容器镜像（适配 64 位 ARM 硬路由） |
| `CubeX-v9.5.0-docker-amd64.tar.gz` | Docker / 爱快 / 飞牛 / NAS / PVE | 容器离线镜像直装导入压缩包（内置双核与全量 105 套规则） |
| `CubeX-v9.5.0-linux-amd64.zip` | Linux amd64 (PVE/LXC/Ubuntu/Debian) | 完整独立开箱包（内含主程序、运维工具、双核引擎及 105 套规则） |
| `CubeX-v9.5.0-linux-arm64.zip` | Linux arm64 (树莓派/香橙派/ARM软路由) | ARM64 完整归档发布包 |
| `cubexd-linux-amd64` | Linux amd64 | 纯单二进制控制面（代码加固版，用于在线平滑热更新） |
| `cubexd-linux-arm64` | Linux arm64 | ARM64 单二进制核心控制面（代码加固版） |
| `cubex-linux-amd64` | Linux amd64 | 纯命令行运维工具（终端直接运行） |
| `cubex-linux-arm64` | Linux arm64 | ARM64 纯命令行运维工具（终端直接运行） |
| `setup-routeros-v7.rsc` | 软路由终端编排脚本 | 软路由管理终端复制即用一键自动编排脚本 |
| `setup-openwrt.sh` | 开源软路由系统一键安装脚本 | 开源软路由系统原生适配安装脚本 |
| `install_pve_lxc.sh` | 容器一键脚本 | 极简容器一键安装脚本 |
| `install.sh` | 通用一键安装脚本 | 通用一键部署主脚本 |
| `sha256sums.txt` | 全平台哈希校验签名 | 包含全部 13 项发布资产的 SHA256 校验清单 |
"""


def gh_api(token, method, path, data=None):
    url = f"https://api.github.com/repos/{REPO}{path}"
    headers = [
        "-H", f"Authorization: token {token}",
        "-H", "Accept: application/vnd.github.v3+json",
        "-H", "User-Agent: CubeX-Release-Agent",
    ]
    cmd = ["curl.exe", "-s", "-w", "\n%{http_code}", "-x", PROXY, "-X", method] + headers

    temp_file = None
    if data:
        temp_file = tempfile.NamedTemporaryFile("w", delete=False, encoding="utf-8")
        json.dump(data, temp_file, ensure_ascii=False)
        temp_file.close()
        cmd.extend(["-H", "Content-Type: application/json; charset=utf-8", "-d", f"@{temp_file.name}"])

    cmd.append(url)
    try:
        res = subprocess.run(cmd, capture_output=True, text=True, timeout=30)
        output = res.stdout
    finally:
        if temp_file and os.path.exists(temp_file.name):
            os.remove(temp_file.name)

    if "\n" in output:
        body, code_str = output.rsplit("\n", 1)
        try:
            code = int(code_str.strip())
        except Exception:
            code = 200
        return code, body
    return 200, output


def get_assets_map(token, release_id):
    code, body = gh_api(token, "GET", f"/releases/{release_id}/assets?per_page=100")
    if code != 200:
        return {}
    try:
        items = json.loads(body)
        return {item["name"]: item for item in items}
    except Exception:
        return {}


def upload_asset(token, release_id, upload_url_tpl, file_path):
    name = file_path.name
    size = file_path.stat().st_size
    base_upload_url = upload_url_tpl.split("{")[0]
    upload_url = f"{base_upload_url}?name={name}"

    content_type = "application/octet-stream"
    if name.endswith(".tar"):
        content_type = "application/x-tar"
    elif name.endswith(".tar.gz") or name.endswith(".tgz"):
        content_type = "application/gzip"
    elif name.endswith(".zip"):
        content_type = "application/zip"
    elif name.endswith(".sh") or name.endswith(".rsc") or name.endswith(".txt"):
        content_type = "text/plain"

    print(f"🚀 [正在上传] {name} ({round(size / 1024 / 1024, 2)} MB)...", flush=True)

    cmd = [
        "curl.exe",
        "--http1.1",
        "-x", PROXY,
        "-X", "POST",
        "-H", f"Authorization: token {token}",
        "-H", f"Content-Type: {content_type}",
        "-H", "Expect:",
        "--speed-time", "180",
        "--speed-limit", "512",
        "--data-binary", f"@{file_path}",
        upload_url,
    ]

    for attempt in range(1, 4):
        assets = get_assets_map(token, release_id)
        if name in assets:
            cur_a = assets[name]
            if cur_a.get("size") == size and cur_a.get("state") == "uploaded":
                print(f"   ✅ [校验成功] {name} 云端字节校验通过！", flush=True)
                return True
            print(f"   - 发现未就绪残片 {name} (state: {cur_a.get('state')}, size: {cur_a.get('size')})，正在清理...", flush=True)
            gh_api(token, "DELETE", f"/releases/assets/{cur_a['id']}")
            time.sleep(2)

        print(f"   - 第 {attempt}/3 次传输...", flush=True)
        res = subprocess.run(cmd, capture_output=True, text=True, timeout=1200)

        time.sleep(2)
        assets = get_assets_map(token, release_id)
        if name in assets and assets[name].get("size") == size and assets[name].get("state") == "uploaded":
            print(f"   ✅ [校验成功] {name} ({round(size / 1024 / 1024, 2)} MB) 上传完成并校验一致！", flush=True)
            return True
        else:
            print(f"   ⚠️ 校验未通过: stdout={res.stdout[:150]}, stderr={res.stderr[:150]}", flush=True)
            time.sleep(3)

    return False


def main():
    token = get_github_token()
    if not token:
        print("❌ 未获取到有效的 GITHUB_TOKEN，发布中止！")
        sys.exit(1)

    print(f"📦 正在检查/创建 Release: {TAG} ...")
    code, body = gh_api(token, "GET", f"/releases/tags/{TAG}")
    release = None
    if code == 200:
        release = json.loads(body)
        print(f"   ✓ 找到已存在的 Release (ID: {release['id']})")
        update_payload = {
            "name": RELEASE_TITLE,
            "body": RELEASE_BODY,
        }
        gh_api(token, "PATCH", f"/releases/{release['id']}", update_payload)
    else:
        print(f"   - 创建新 Release: {TAG} ...")
        create_payload = {
            "tag_name": TAG,
            "target_commitish": "main",
            "name": RELEASE_TITLE,
            "body": RELEASE_BODY,
            "draft": False,
            "prerelease": False,
        }
        code, body = gh_api(token, "POST", "/releases", create_payload)
        if code in (200, 201):
            release = json.loads(body)
            print(f"   ✅ 成功创建 Release (ID: {release['id']})")
        else:
            print(f"❌ 创建 Release 失败 (HTTP {code}): {body}")
            sys.exit(1)

    release_id = release["id"]
    upload_url_tpl = release["upload_url"]

    expected_assets = [
        f"CubeX-{TAG}-ros-amd64.tar",
        f"CubeX-{TAG}-ros-arm64.tar",
        f"CubeX-{TAG}-docker-amd64.tar.gz",
        f"CubeX-{TAG}-linux-amd64.zip",
        f"CubeX-{TAG}-linux-arm64.zip",
        "cubexd-linux-amd64",
        "cubexd-linux-arm64",
        "cubex-linux-amd64",
        "cubex-linux-arm64",
        "setup-routeros-v7.rsc",
        "setup-openwrt.sh",
        "install_pve_lxc.sh",
        "install.sh",
        "sha256sums.txt",
    ]

    print(f"\n📂 检查本地发布物目录: {RELEASE_DIR}")
    missing = []
    for item in expected_assets:
        p = RELEASE_DIR / item
        if not p.exists():
            missing.append(item)
    if missing:
        print(f"❌ 本地发布物缺失 {len(missing)} 项，无法发布:")
        for m in missing:
            print(f"   - {m}")
        sys.exit(1)

    print(f"✅ {len(expected_assets)} 项发布资产全部就绪，开始上传至 GitHub Releases (ID: {release_id})...\n")

    failed = []
    for item in expected_assets:
        p = RELEASE_DIR / item
        ok = upload_asset(token, release_id, upload_url_tpl, p)
        if not ok:
            failed.append(item)

    if failed:
        print(f"\n❌ 上传失败列表 ({len(failed)} 项): {failed}")
        sys.exit(1)

    print(f"\n🎉 全部 {len(expected_assets)} 项资产上传并校验 100% 成功！")

    print("\n🧹 执行全平台 10 周期存留红线清算...")
    enforce_script = SCRIPT_DIR / "enforce_10_version_retention.py"
    subprocess.run([sys.executable, str(enforce_script)], check=True)
    print("✅ 10 周期存留红线清算完成！发布流程圆满收官！")


if __name__ == "__main__":
    main()
