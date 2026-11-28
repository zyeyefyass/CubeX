#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
CubeX Pure Multi-Arch Linux Cross-Compiler
Reads version from meta.json and compiles linux/{amd64,arm64} cubexd & cubex
with -trimpath -ldflags="-s -w -buildid= -X main.version=<ver>".
"""
import json
import os
import subprocess
import sys

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
os.chdir(ROOT)


def read_meta_version() -> str:
    with open(os.path.join(ROOT, "meta.json"), "r", encoding="utf-8") as f:
        ver = str(json.load(f).get("version", "")).strip()
    if not ver:
        raise RuntimeError("meta.json missing 'version'")
    return ver if ver.startswith("v") else f"v{ver}"


def main() -> int:
    ver = read_meta_version()
    os.makedirs(os.path.join(ROOT, "dist"), exist_ok=True)
    env_base = os.environ.copy()
    env_base["CGO_ENABLED"] = "0"

    ldflags = f"-s -w -buildid= -X main.version={ver}"
    targets = [
        ("amd64", "./cmd/cubexd", "dist/cubexd-linux-amd64"),
        ("amd64", "./cmd/cubex", "dist/cubex-linux-amd64"),
        ("arm64", "./cmd/cubexd", "dist/cubexd-linux-arm64"),
        ("arm64", "./cmd/cubex", "dist/cubex-linux-arm64"),
    ]

    print(f"🔨 Compiling CubeX {ver} Linux amd64/arm64 binaries (pure static, no AST bloat)...")
    for arch, pkg, out in targets:
        env = env_base.copy()
        env["GOOS"] = "linux"
        env["GOARCH"] = arch
        cmd = ["go", "build", "-trimpath", f"-ldflags={ldflags}", "-o", out, pkg]
        print(f"  -> {out} ({arch})")
        res = subprocess.run(cmd, env=env, capture_output=True, text=True)
        if res.returncode != 0:
            print(f"  ❌ Failed compiling {out}: {res.stderr}", file=sys.stderr)
            return 1
        size_mb = os.path.getsize(out) / (1024 * 1024)
        print(f"     ✅ {size_mb:.2f} MB")

    print("🎉 All Linux multi-arch binaries compiled cleanly!")
    return 0


if __name__ == "__main__":
    sys.exit(main())
