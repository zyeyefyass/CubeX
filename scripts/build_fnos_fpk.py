#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
CubeX · 飞牛 fnOS 原生 .fpk 应用套件打包生成器
遵循飞牛 fnOS 应用中心官方套件打包规范 (manifest, compose.yaml, ICON, privilege)
生成 cubex_fnos_v9.9.0_amd64.fpk 与 cubex_fnos_v9.9.0_arm64.fpk
"""

import os
import sys
import json
import shutil
import tarfile
import hashlib
from PIL import Image

VERSION = "9.9.1"
APP_NAME = "cubex"
DISPLAY_NAME = "CubeX 网络与影音中枢"
DESC = "15MB 极轻量全屋网络拓扑管理、NAS 影视刮削加速与 DNS 防抖自愈网关"
MAINTAINER = "CubeX Lab"
WEBSITE = "https://github.com/zyeyefyass/CubeX"
GHCR_IMAGE = f"ghcr.io/zyeyefyass/cubex:{VERSION}"

def prepare_fnos_packaging_dir():
    pkg_dir = os.path.abspath("packaging/fnos")
    os.makedirs(pkg_dir, exist_ok=True)
    os.makedirs(os.path.join(pkg_dir, "config"), exist_ok=True)
    os.makedirs(os.path.join(pkg_dir, "ui"), exist_ok=True)

    # 1. 生成 manifest (键值对格式)
    manifest_kv = f"""appname = {APP_NAME}
display_name = {DISPLAY_NAME}
version = {VERSION}
desc = {DESC}
maintainer = {MAINTAINER}
author = {MAINTAINER}
website = {WEBSITE}
platform = all
desktop_uidir = ui
icon = ICON.PNG
icon_large = ICON_256.PNG
port = 2026
protocol = http
"""
    with open(os.path.join(pkg_dir, "manifest"), "w", encoding="utf-8") as f:
        f.write(manifest_kv)

    # 2. 生成 manifest.json (JSON 格式)
    manifest_json = {
        "manifest_version": 1,
        "appname": APP_NAME,
        "name": APP_NAME,
        "display_name": DISPLAY_NAME,
        "version": VERSION,
        "platform": ["x86_64", "aarch64"],
        "desc": DESC,
        "description": DESC,
        "author": MAINTAINER,
        "maintainer": MAINTAINER,
        "website": WEBSITE,
        "icon": "ICON.PNG",
        "icon_large": "ICON_256.PNG",
        "desktop": {
            "display_name": "CubeX",
            "icon": "ICON.PNG",
            "url": "http://{HOST}:{PORT}/",
            "port": 2026,
            "protocol": "http"
        }
    }
    with open(os.path.join(pkg_dir, "manifest.json"), "w", encoding="utf-8") as f:
        json.dump(manifest_json, f, ensure_ascii=False, indent=2)

    # 3. 生成 compose.yaml (Docker Compose 编排)
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
      - /vol1/@appdata/cubex/etc:/opt/CubeX/etc
      - /vol1/@appdata/cubex/data:/opt/CubeX/data
      - /vol1/@appdata/cubex/rules:/opt/CubeX/rules
      - /etc/localtime:/etc/localtime:ro
    environment:
      - TZ=Asia/Shanghai
      - CUBEX_ENV=production
      - CUBEX_STORE_MODE=true
      - CUBEX_PORT=2026
      - CUBEX_LISTEN=:2026
      - GOMEMLIMIT=128MiB
"""
    with open(os.path.join(pkg_dir, "compose.yaml"), "w", encoding="utf-8") as f:
        f.write(compose_yaml)
    # 同时保留 docker-compose.yml 兼容不同版本检测
    with open(os.path.join(pkg_dir, "docker-compose.yml"), "w", encoding="utf-8") as f:
        f.write(compose_yaml)

    # 4. 生成 config/privilege 权限声明
    privilege_json = {
        "defaults": {
            "run-as": "root"
        }
    }
    with open(os.path.join(pkg_dir, "config", "privilege"), "w", encoding="utf-8") as f:
        json.dump(privilege_json, f, indent=2)

    # 5. 生成图标 ICON.PNG (128x128) 和 ICON_256.PNG (256x256)
    src_logo = "assets/logo.png"
    if os.path.exists(src_logo):
        img = Image.open(src_logo).convert("RGBA")
        img_128 = img.resize((128, 128), Image.Resampling.LANCZOS)
        img_128.save(os.path.join(pkg_dir, "ICON.PNG"))
        
        img_256 = img.resize((256, 256), Image.Resampling.LANCZOS)
        img_256.save(os.path.join(pkg_dir, "ICON_256.PNG"))
        print(f"✓ Generated ICON.PNG and ICON_256.PNG from {src_logo}")

    print(f"✓ Prepared packaging directory at {pkg_dir}")
    return pkg_dir

def build_fpk_package(arch):
    pkg_dir = os.path.abspath("packaging/fnos")
    dist_dir = os.path.abspath("dist-fpk")
    os.makedirs(dist_dir, exist_ok=True)

    fpk_name = f"cubex_fnos_v{VERSION}_{arch}.fpk"
    fpk_path = os.path.join(dist_dir, fpk_name)

    # 飞牛 fpk 本质是规范的 tar.gz 归档包
    with tarfile.open(fpk_path, "w:gz") as tar:
        for root, dirs, files in os.walk(pkg_dir):
            for file in files:
                full_path = os.path.join(root, file)
                rel_path = os.path.relpath(full_path, pkg_dir)
                tar.add(full_path, arcname=rel_path)

    # 计算 SHA256 校验码
    sha256_hash = hashlib.sha256()
    with open(fpk_path, "rb") as f:
        for byte_block in iter(lambda: f.read(4096), b""):
            sha256_hash.update(byte_block)
    sha256_val = sha256_hash.hexdigest()

    sha_file = fpk_path + ".sha256"
    with open(sha_file, "w", encoding="utf-8") as f:
        f.write(f"{sha256_val}  {fpk_name}\n")

    size_kb = os.path.getsize(fpk_path) / 1024
    print(f"🎉 Successfully built: {fpk_name} ({size_kb:.2f} KB) - SHA256: {sha256_val[:16]}...")
    return fpk_path, sha256_val

def main():
    print("=" * 65)
    print(f"📦 CubeX · 飞牛 fnOS 原生应用套件打包构建器 (v{VERSION})")
    print("=" * 65)

    prepare_fnos_packaging_dir()
    for arch in ["amd64", "arm64"]:
        build_fpk_package(arch)

    print("\n✓ 所有 FPK 制品已输出至 dist-fpk/ 目录！")

if __name__ == "__main__":
    main()
