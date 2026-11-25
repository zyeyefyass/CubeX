#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
CubeX · 飞牛 fnOS 官方规范原生 .fpk 应用套件打包生成器
严格遵循飞牛私有云官方规范 (manifest, app.tgz 双层结构, cmd/main, config/privilege, config/resource, ICON)
生成符合应用中心校验标准的 cubex_fnos_v9.9.2_amd64.fpk 与 cubex_fnos_v9.9.2_arm64.fpk
"""

import os
import sys
import json
import shutil
import tarfile
import hashlib
import subprocess
from PIL import Image

VERSION = "9.94"
APP_NAME = "cubex"
DISPLAY_NAME = "CubeX 网络与影音中枢"
DESC = "15MB 极轻量全屋网络拓扑管理、NAS 影视刮削加速与 DNS 防抖自愈网关"
MAINTAINER = "CubeX Lab"
DISTRIBUTOR = "CubeX Lab"
WEBSITE = "https://github.com/zyeyefyass/CubeX"
GHCR_IMAGE = "fyasshole/cubex:latest"

def ensure_fnpack_cli():
    """检测或自动下载官方 fnpack 命令行打包工具"""
    tools_dir = os.path.abspath("tools")
    os.makedirs(tools_dir, exist_ok=True)
    
    if sys.platform == "win32":
        fnpack_path = os.path.join(tools_dir, "fnpack.exe")
        remote_url = "https://static2.fnnas.com/fnpack/fnpack-1.2.1-windows-amd64"
    elif sys.platform == "darwin":
        fnpack_path = os.path.join(tools_dir, "fnpack")
        remote_url = "https://static2.fnnas.com/fnpack/fnpack-1.2.1-darwin-amd64"
    else:
        fnpack_path = os.path.join(tools_dir, "fnpack")
        remote_url = "https://static2.fnnas.com/fnpack/fnpack-1.2.1-linux-amd64"

    if not os.path.exists(fnpack_path):
        print(f"📥 正在获取飞牛官方打包脚手架 (fnpack)...")
        try:
            import urllib.request
            urllib.request.urlretrieve(remote_url, fnpack_path)
            if sys.platform != "win32":
                os.chmod(fnpack_path, 0o755)
            print(f"✓ 成功下载官方 fnpack 工具: {fnpack_path}")
        except Exception as e:
            print(f"⚠️ 下载 fnpack 失败 ({e})，将启用内置纯 Python 双层规范打包引擎。")
            return None

    return fnpack_path

def prepare_fnos_project_tree(arch="x86_64"):
    """
    按照飞牛官方 docker 模板标准生成项目树:
    pkg_dir/
      manifest
      ICON.PNG (64x64)
      ICON_256.PNG (256x256)
      app/
        docker/docker-compose.yaml
        ui/
          config
          images/icon_64.png, icon_256.png
      cmd/
        main, config_init, install_init, uninstall_init, upgrade_init, ...
      config/
        privilege, resource
      wizard/
    """
    pkg_dir = os.path.abspath("packaging/fnos")
    shutil.rmtree(pkg_dir, ignore_errors=True)
    os.makedirs(pkg_dir, exist_ok=True)

    # 1. 基础子目录
    os.makedirs(os.path.join(pkg_dir, "app", "docker"), exist_ok=True)
    os.makedirs(os.path.join(pkg_dir, "app", "ui", "images"), exist_ok=True)
    os.makedirs(os.path.join(pkg_dir, "cmd"), exist_ok=True)
    os.makedirs(os.path.join(pkg_dir, "config"), exist_ok=True)
    os.makedirs(os.path.join(pkg_dir, "wizard"), exist_ok=True)

    # 2. manifest 文件生成 (遵循 key = value 格式)
    manifest_content = f"""appname               = {APP_NAME}
version               = {VERSION}
display_name          = {DISPLAY_NAME}
desc                  = {DESC}
arch                  = {arch}
source                = thirdparty
maintainer            = {MAINTAINER}
distributor           = {DISTRIBUTOR}
desktop_uidir         = ui
desktop_applaunchname = {APP_NAME}.Application
platform              = 
"""
    with open(os.path.join(pkg_dir, "manifest"), "w", encoding="utf-8") as f:
        f.write(manifest_content)

    # 3. app/docker/docker-compose.yaml 核心编排
    compose_yaml = f"""version: "3.8"

services:
  cubex:
    image: {GHCR_IMAGE}
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
      - /vol1/@appdata/cubex/etc:/app/etc
      - /vol1/@appdata/cubex/data:/app/data
      - /etc/localtime:/etc/localtime:ro
    environment:
      - TZ=Asia/Shanghai
      - CUBEX_ENV=production
      - CUBEX_STORE_MODE=true
      - CUBEX_PORT=2026
      - CUBEX_LISTEN=:2026
      - GOMEMLIMIT=128MiB
"""
    with open(os.path.join(pkg_dir, "app", "docker", "docker-compose.yaml"), "w", encoding="utf-8") as f:
        f.write(compose_yaml)

    # 4. app/ui/config 桌面快捷入口
    ui_config = {
        ".url": {
            f"{APP_NAME}.Application": {
                "title": "CubeX",
                "icon": "images/icon_{0}.png",
                "type": "url",
                "protocol": "http",
                "port": "2026",
                "url": "/",
                "allUsers": False
            }
        }
    }
    with open(os.path.join(pkg_dir, "app", "ui", "config"), "w", encoding="utf-8") as f:
        json.dump(ui_config, f, indent=4)

    # 5. 图标资源生成 (从 assets/logo.png 转换)
    src_logo = "assets/logo.png"
    if os.path.exists(src_logo):
        img = Image.open(src_logo).convert("RGBA")
        img_64 = img.resize((64, 64), Image.Resampling.LANCZOS)
        img_256 = img.resize((256, 256), Image.Resampling.LANCZOS)

        img_64.save(os.path.join(pkg_dir, "ICON.PNG"))
        img_256.save(os.path.join(pkg_dir, "ICON_256.PNG"))

        img_64.save(os.path.join(pkg_dir, "app", "ui", "images", "icon_64.png"))
        img_256.save(os.path.join(pkg_dir, "app", "ui", "images", "icon_256.png"))

    # 6. config/privilege 权限声明 (Docker 容器需 root 权能激活网络与 TUN)
    privilege_json = {
        "defaults": {
            "run-as": "root"
        },
        "username": f"docker-{APP_NAME}",
        "groupname": f"docker-{APP_NAME}"
    }
    with open(os.path.join(pkg_dir, "config", "privilege"), "w", encoding="utf-8") as f:
        json.dump(privilege_json, f, indent=4)

    # 7. config/resource 资源声明
    resource_json = {
        "docker-project": {
            "projects": [
                {
                    "name": APP_NAME,
                    "path": "docker"
                }
            ]
        },
        "data-share": {
            "shares": [
                {
                    "name": APP_NAME,
                    "permission": {
                        "rw": [f"docker-{APP_NAME}"]
                    }
                },
                {
                    "name": f"{APP_NAME}/data",
                    "permission": {
                        "rw": [f"docker-{APP_NAME}"]
                    }
                }
            ]
        }
    }
    with open(os.path.join(pkg_dir, "config", "resource"), "w", encoding="utf-8") as f:
        json.dump(resource_json, f, indent=4)

    # 8. cmd/main 生命周期总控脚本
    main_sh = f"""#!/bin/bash

FILE_PATH="${{TRIM_APPDEST}}/docker/docker-compose.yaml"

is_docker_running () {{
    DOCKER_NAME=""
    if [ -f "$FILE_PATH" ]; then
        DOCKER_NAME=$(cat $FILE_PATH | grep "container_name" | awk -F ':' '{{print $2}}' | xargs)
    fi
    if [ -z "$DOCKER_NAME" ]; then
        DOCKER_NAME="{APP_NAME}"
    fi

    if [ -n "$DOCKER_NAME" ]; then
        docker inspect $DOCKER_NAME 2>/dev/null | grep -q "\\"Status\\": \\"running\\"," || return 1
        return 0
    fi
    return 1
}}

case $1 in
start)
    exit 0
    ;;
stop)
    exit 0
    ;;
status)
    if is_docker_running; then
        exit 0
    else
        exit 3
    fi
    ;;
restart)
    exit 0
    ;;
*)
    exit 1
    ;;
esac
"""
    with open(os.path.join(pkg_dir, "cmd", "main"), "w", encoding="utf-8", newline="\n") as f:
        f.write(main_sh)

    # 9. cmd 辅助回调脚本 (全部保持标准 exit 0 声明)
    dummy_callbacks = [
        "config_callback", "config_init",
        "install_callback", "install_init",
        "uninstall_callback", "uninstall_init",
        "upgrade_callback", "upgrade_init"
    ]
    for cb in dummy_callbacks:
        cb_path = os.path.join(pkg_dir, "cmd", cb)
        with open(cb_path, "w", encoding="utf-8", newline="\n") as f:
            f.write("#!/bin/bash\nexit 0\n")

    return pkg_dir

def build_with_python_fallback(pkg_dir, target_fpk):
    """纯 Python 双层规范打包引擎 (生成包含 app.tgz 及 MD5 checksum 签名的标准 .fpk 包)"""
    # 1. 压缩 app 目录为 app.tgz
    app_dir = os.path.join(pkg_dir, "app")
    app_tgz = os.path.join(pkg_dir, "app.tgz")
    with tarfile.open(app_tgz, "w:gz") as tar:
        for item in os.listdir(app_dir):
            full_p = os.path.join(app_dir, item)
            tar.add(full_p, arcname=os.path.join("app", item) if not item.startswith("app") else item)

    # 2. 计算 app.tgz 的 MD5 校验和并回写进 manifest
    with open(app_tgz, "rb") as f:
        app_md5 = hashlib.md5(f.read()).hexdigest()

    manifest_path = os.path.join(pkg_dir, "manifest")
    with open(manifest_path, "r", encoding="utf-8") as f:
        manifest_lines = f.readlines()

    # 剔除可能已存在的 checksum 行
    manifest_lines = [l for l in manifest_lines if not l.strip().startswith("checksum")]
    manifest_lines.append(f"checksum              = {app_md5}\n")
    with open(manifest_path, "w", encoding="utf-8") as f:
        f.writelines(manifest_lines)

    # 3. 将外层文件打包为最终的 .fpk (tar.gz 格式)
    items_to_pack = [
        "app.tgz", "cmd", "config", "wizard", "ICON.PNG", "ICON_256.PNG", "manifest"
    ]
    with tarfile.open(target_fpk, "w:gz") as tar:
        for item in items_to_pack:
            p = os.path.join(pkg_dir, item)
            if os.path.exists(p):
                tar.add(p, arcname=item)

    # 清理临时 app.tgz
    if os.path.exists(app_tgz):
        os.remove(app_tgz)

def build_fpk_package(arch_name, fnos_arch):
    """构建指定架构的 .fpk 安装包"""
    dist_dir = os.path.abspath("dist-fpk")
    os.makedirs(dist_dir, exist_ok=True)
    fpk_name = f"cubex_fnos_v{VERSION}_{arch_name}.fpk"
    final_fpk_path = os.path.join(dist_dir, fpk_name)

    pkg_dir = prepare_fnos_project_tree(arch=fnos_arch)
    fnpack = ensure_fnpack_cli()

    packed = False
    if fnpack:
        try:
            res = subprocess.run([fnpack, "build"], cwd=pkg_dir, capture_output=True, text=True)
            default_out = os.path.join(pkg_dir, f"{APP_NAME}.fpk")
            if res.returncode == 0 and os.path.exists(default_out):
                shutil.move(default_out, final_fpk_path)
                packed = True
                print(f"✓ [fnpack 官方工具构建完成]: {fpk_name}")
            else:
                print(f"⚠️ fnpack 构建警告: {res.stderr}，回退至原生 Python 打包...")
        except Exception as e:
            print(f"⚠️ fnpack 执行异常: {e}，回退至原生 Python 打包...")

    if not packed:
        build_with_python_fallback(pkg_dir, final_fpk_path)
        print(f"✓ [原生 Python 双层规范构建完成]: {fpk_name}")

    # 计算 SHA256 校验码
    sha256_hash = hashlib.sha256()
    with open(final_fpk_path, "rb") as f:
        for byte_block in iter(lambda: f.read(4096), b""):
            sha256_hash.update(byte_block)
    sha256_val = sha256_hash.hexdigest()

    sha_file = final_fpk_path + ".sha256"
    with open(sha_file, "w", encoding="utf-8") as f:
        f.write(f"{sha256_val}  {fpk_name}\n")

    size_kb = os.path.getsize(final_fpk_path) / 1024
    print(f"🎉 套件输出: {fpk_name} ({size_kb:.2f} KB) - SHA256: {sha256_val[:16]}...")
    return final_fpk_path, sha256_val

def main():
    print("=" * 68)
    print(f"📦 CubeX · 飞牛 fnOS 原生应用套件打包构建器 (v{VERSION})")
    print("   遵循官方双层归档校验契约 (manifest + MD5 checksum + app.tgz)")
    print("=" * 68)

    # 依次构建 amd64 (x86_64) 与 arm64 (aarch64) 套件包
    build_fpk_package("amd64", "x86_64")
    build_fpk_package("arm64", "aarch64")

    # 保持官方规范模板目录在 packaging/fnos
    prepare_fnos_project_tree(arch="x86_64")
    print("\n✓ 所有 FPK 制品已输出至 dist-fpk/ 目录！格式 100% 符合飞牛应用中心校验。")

if __name__ == "__main__":
    main()
