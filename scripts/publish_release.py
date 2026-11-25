#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
CubeX 原生双栈与极速热切版 GitHub 发布自动化脚本
- 创建/更新 Release（全中文精简大白话说明 + 资产清单表）
- 统一通过 curl.exe --http1.1 -x PROXY 禁用 Expect: 100-continue 直传资产，支持断点续传与字节校验
- 上传完毕后自动执行全平台 10 版本存留红线脚本
"""
import json
import os
import socket
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

def detect_proxy():
    env = os.environ.get("CUBEX_RELEASE_PROXY") or os.environ.get("HTTPS_PROXY") or os.environ.get("HTTP_PROXY")
    if env:
        if env.strip().upper() == "DIRECT":
            return ""
        return env
    for host in ["127.0.0.1"]:
        try:
            s = socket.socket()
            s.settimeout(0.3)
            s.connect((host, 7890))
            s.close()
            return f"http://{host}:7890"
        except Exception:
            pass
    return ""

PROXY = detect_proxy()

def get_tag():
    try:
        with open(ROOT_DIR / "meta.json", "r", encoding="utf-8") as f:
            v = json.load(f).get("version", "v9.93")
            if not v.startswith("v"):
                v = f"v{v}"
            return v
    except Exception:
        return "v9.93"

TAG = get_tag()

RELEASE_TITLE = f"CubeX {TAG} · 根治宿主机 DNS 自环死锁 · 防死锁 dns-hosts 规则 · 启动期自环巡检 · RouterOS 对称撤回"

RELEASE_BODY = f"""1. **根治宿主机 DHCP 续租引发的 DNS 自环死锁**：`sing-box` 编译前自动巡检宿主机 `/etc/resolv.conf`，一旦发现指向 `127.0.0.1`、`::1` 或本机局域网 IP，自动弃用 `type: "local"` 并安全回退至上级网关或 `223.5.5.5`，彻底消除自己问自己的死循环风暴。
2. **接入防死锁 `dns-hosts` 规则**：将内置公共 DNS 静态解析表（`dns.alidns.com`、`doh.pub`、`dns.google`、`cloudflare-dns.com`）接入 `dns.rules` 优先命中，消除引导域名解析死锁。
3. **新增启动期 DNS 自环巡检指引**：在 `install.sh` 与 `cubexd` 启动预检中温和检测本机 DNS 自环风险并输出固定 DNS 修复指引，不锁死系统网络管理文件。
4. **补齐 RouterOS 一键对称撤回**：完善 `POST /api/v1/host/bypass/remove` 对 RouterOS 的原生撤回能力，完整清理旁路路由、规则与看门狗并恢复上游 DNS，与爱快、开源软路由完全对称。

---

### 📦 资产下载与校验清单 (Release Assets)

> [!TIP]
> **各平台底座发布物对照表**：请根据您的宿主底座与硬件架构选择对应的下载资产（完整包内置双核与全量 105 套规则）：

| 发布物文件名 | 适用宿主与架构 | 说明与用途 |
| :--- | :--- | :--- |
| `cubex_fnos_{TAG}_amd64.fpk` | 飞牛 fnOS (x86_64) | 飞牛私有云官方应用中心原生套件安装包 |
| `cubex_fnos_{TAG}_arm64.fpk` | 飞牛 fnOS (aarch64) | 飞牛私有云 ARM64 官方应用中心套件包 (适配 OECT / 树莓派 / RK3566) |
| `CubeX-{TAG}-ros-amd64.tar` | 软路由系统 (x86 软路由) | 软路由系统原生未压缩容器镜像（支持管理终端拖入直装） |
| `CubeX-{TAG}-ros-arm64.tar` | 软路由系统 (RB5009/CCR/ARM) | 软路由系统 ARM64 原生容器镜像（适配 64 位 ARM 硬路由） |
| `CubeX-{TAG}-docker-amd64.tar.gz` | Docker / 爱快 / 飞牛 / NAS / PVE | x86 容器离线镜像直装导入压缩包（内置双核与全量 105 套规则） |
| `CubeX-{TAG}-docker-arm64.tar.gz` | Docker / 树莓派 / ARM64 NAS | ARM64 容器离线镜像直装导入压缩包（内置双核与全量 105 套规则） |
| `CubeX-{TAG}-linux-amd64.zip` | 通用 Linux (x86_64 / Debian / PVE) | 完整离线发布归档包（含纯净版双核心、规则集与安装脚本） |
| `CubeX-{TAG}-linux-arm64.zip` | 通用 Linux (ARM64 / aarch64) | ARM64 完整离线归档包（含纯净版双核心、规则集与安装脚本） |
| `cubexd-linux-amd64` | Linux x86_64 (单二进制) | 后端控制面主守护进程单二进制（支持原地替换平滑热升级） |
| `cubexd-linux-arm64` | Linux ARM64 (单二进制) | ARM64 后端控制面主守护进程单二进制（支持原地替换平滑热升级） |
| `cubex-linux-amd64` | Linux x86_64 (CLI) | 命令行终端运维管理工具单二进制 |
| `cubex-linux-arm64` | Linux ARM64 (CLI) | ARM64 命令行终端运维管理工具单二进制 |
| `install_ros_container.rsc` | RouterOS 终端 | RouterOS 原生一键部署与逃生看门狗自动化配置脚本 |
| `install.sh` | 通用一键安装脚本 | 通用一键部署主脚本 |
| `SHA256SUMS` | 全平台校验文件 | 所有发布产物完整性 SHA256 校验清单 |
"""


def gh_api(token, method, path, data=None):
    url = f"https://api.github.com/repos/{REPO}{path}"
    headers = [
        "-H", f"Authorization: token {token}",
        "-H", "Accept: application/vnd.github.v3+json",
        "-H", "User-Agent: CubeX-Release-Agent",
    ]
    curl_bin = r"C:\Program Files\Git\mingw64\bin\curl.exe"
    if not os.path.exists(curl_bin):
        curl_bin = "curl.exe"
    cmd = [curl_bin, "-s", "-w", "\n%{http_code}", "-X", method] + headers
    if PROXY:
        cmd.extend(["-x", PROXY])

    temp_file = None
    if data:
        temp_file = tempfile.NamedTemporaryFile("w", delete=False, encoding="utf-8")
        json.dump(data, temp_file, ensure_ascii=False)
        temp_file.close()
        cmd.extend(["-H", "Content-Type: application/json; charset=utf-8", "-d", f"@{temp_file.name}"])

    cmd.append(url)
    try:
        res = subprocess.run(cmd, capture_output=True, text=True, timeout=90)
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
    elif name.endswith(".sh") or name.endswith(".rsc") or name.endswith(".txt") or name == "SHA256SUMS":
        content_type = "text/plain"
    elif name.endswith(".fpk"):
        content_type = "application/octet-stream"

    # Pre-check if already uploaded cleanly
    assets = get_assets_map(token, release_id)
    if name in assets:
        cur_a = assets[name]
        if cur_a.get("size") == size and cur_a.get("state") == "uploaded":
            print(f"✅ [已存在且完整] {name} ({round(size / 1024 / 1024, 2)} MB) 跳过重复上传", flush=True)
            return True
        print(f" - 发现未就绪残片 {name} (state: {cur_a.get('state')}, size: {cur_a.get('size')})，正在清理...", flush=True)
        gh_api(token, "DELETE", f"/releases/assets/{cur_a['id']}")
        time.sleep(3)

    print(f"🚀 [正在上传] {name} ({round(size / 1024 / 1024, 2)} MB)...", flush=True)

    curl_bin = r"C:\Program Files\Git\mingw64\bin\curl.exe"
    if not os.path.exists(curl_bin):
        curl_bin = "curl.exe"

    cmd = [
        curl_bin,
        "--http1.1",
        "-s", "-S",
        "-w", "\n%{http_code}",
        "-L",
        "--connect-timeout", "25",
        "--speed-limit", "20480",
        "--speed-time", "30",
        "--max-time", "1200",
        "-X", "POST",
        "-H", f"Authorization: token {token}",
        "-H", f"Content-Type: {content_type}",
        "-H", "Expect:",
        "--data-binary", f"@{file_path}",
    ]
    if PROXY:
        cmd.extend(["-x", PROXY])
    cmd.append(upload_url)

    for attempt in range(1, 5):
        # Ensure no dirty starter exists before attempting upload
        assets = get_assets_map(token, release_id)
        if name in assets:
            cur_a = assets[name]
            if cur_a.get("size") == size and cur_a.get("state") == "uploaded":
                print(f"   ✅ [校验成功] {name} 云端校验通过！", flush=True)
                return True
            print(f"   - 清理未就绪残片 {name} (ID: {cur_a['id']})...", flush=True)
            gh_api(token, "DELETE", f"/releases/assets/{cur_a['id']}")
            time.sleep(3)

        print(f"   - 第 {attempt}/4 次传输...", flush=True)
        res = subprocess.run(cmd, capture_output=True, text=True, timeout=1220)

        lines = res.stdout.strip().splitlines()
        http_code = lines[-1].strip() if lines else ""

        # Wait for GitHub background storage finalization
        print("   - 等待 GitHub 云端入库终审校验...", flush=True)
        success = False
        max_waits = 12 if (res.returncode == 0 and http_code in ("200", "201")) else 3
        for wait_idx in range(max_waits):
            time.sleep(5)
            assets = get_assets_map(token, release_id)
            if name in assets:
                cur_a = assets[name]
                if cur_a.get("size") == size and cur_a.get("state") == "uploaded":
                    print(f"   ✅ [校验成功] {name} ({round(size / 1024 / 1024, 2)} MB) 上传完成并校验一致！", flush=True)
                    success = True
                    break
                else:
                    print(f"     ...等待中 (state: {cur_a.get('state')}, size: {cur_a.get('size')}/{size})", flush=True)
        if success:
            return True

        print(f"   ⚠️ 第 {attempt} 次校验未通过: code={res.returncode}, http={http_code}, stderr={res.stderr[:150]}", flush=True)
        time.sleep(5)

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
        print(f"   - 创建新 Release (Draft 状态): {TAG} ...")
        create_payload = {
            "tag_name": TAG,
            "target_commitish": "main",
            "name": RELEASE_TITLE,
            "body": RELEASE_BODY,
            "draft": True,
            "prerelease": False,
        }
        code, body = gh_api(token, "POST", "/releases", create_payload)
        if code in (200, 201):
            release = json.loads(body)
            print(f"   ✅ 成功创建 Draft Release (ID: {release['id']})")
        else:
            print(f"❌ 创建 Release 失败 (HTTP {code}): {body}")
            sys.exit(1)

    release_id = release["id"]
    upload_url_tpl = release["upload_url"]

    expected_assets = [
        f"cubex_fnos_{TAG}_amd64.fpk",
        f"cubex_fnos_{TAG}_arm64.fpk",
        f"CubeX-{TAG}-ros-amd64.tar",
        f"CubeX-{TAG}-ros-arm64.tar",
        f"CubeX-{TAG}-docker-amd64.tar.gz",
        f"CubeX-{TAG}-docker-arm64.tar.gz",
        f"CubeX-{TAG}-linux-amd64.zip",
        f"CubeX-{TAG}-linux-arm64.zip",
        "cubexd-linux-amd64",
        "cubexd-linux-arm64",
        "cubex-linux-amd64",
        "cubex-linux-arm64",
        "install_ros_container.rsc",
        "install_openwrt_docker.sh",
        "setup-routeros-v7.rsc",
        "setup-openwrt.sh",
        "install_pve_lxc.sh",
        "install.sh",
        "SHA256SUMS",
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

    print(f"✅ {len(expected_assets)} 项发布资产全部就绪，开始上传至 GitHub Draft Release (ID: {release_id})...\n")

    failed = []
    for item in expected_assets:
        p = RELEASE_DIR / item
        ok = upload_asset(token, release_id, upload_url_tpl, p)
        if not ok:
            failed.append(item)

    if failed:
        print(f"\n❌ 上传失败列表 ({len(failed)} 项): {failed}")
        sys.exit(1)

    print(f"\n🔍 [终审核验] 逐一核验 GitHub 云端资产状态 (state: uploaded & 字节数一致)...")
    final_assets = get_assets_map(token, release_id)
    verify_failed = False
    for item in expected_assets:
        p = RELEASE_DIR / item
        exp_sz = p.stat().st_size
        if item not in final_assets:
            print(f"   ❌ 缺失云端资产: {item}")
            verify_failed = True
            continue
        cur = final_assets[item]
        state = cur.get("state")
        actual_sz = cur.get("size")
        if state != "uploaded" or actual_sz != exp_sz:
            print(f"   ❌ 资产异常: {item} (state={state}, size={actual_sz}/{exp_sz})")
            verify_failed = True
        else:
            print(f"   ✓ {item}: state=uploaded, size={actual_sz:,} 字节 (PASS)")

    if verify_failed:
        print("\n🚨 资产终审核验未通过，Release 保持 Draft 状态，中止正式发布！")
        sys.exit(1)

    print(f"\n🎉 全部 {len(expected_assets)} 项资产在 Draft 状态下核验 100% 满分通过！")
    print("🚀 正在将 Release 由 Draft 切换为正式发布 (draft: false)...")
    patch_code, patch_res = gh_api(token, "PATCH", f"/releases/{release_id}", {"draft": False})
    if patch_code == 200:
        pub_rel = json.loads(patch_res)
        print(f"✅ 正式发布成功！Release URL: {pub_rel.get('html_url')}")
    else:
        print(f"❌ 切换正式发布失败 (HTTP {patch_code}): {patch_res}")
        sys.exit(1)

    print("\n🧹 执行全平台 10 周期存留红线清算...")
    enforce_script = SCRIPT_DIR / "enforce_10_version_retention.py"
    subprocess.run([sys.executable, str(enforce_script)], check=True)
    print("✅ 10 周期存留红线清算完成！发布流程圆满收官！")


if __name__ == "__main__":
    main()
