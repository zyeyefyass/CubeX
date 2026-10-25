#!/usr/bin/env bash
# ════════════════════════════════════════════════════════════════════
# 🌌 CubeX Monorepo 全架构自动构建脚本 (Linux / macOS)
# ════════════════════════════════════════════════════════════════════
set -e

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "🌌 [1/3] 构建 Twilight Crystal Web 前端..."
cd "$ROOT_DIR/web"
npm run build

echo "⚙️ [2/3] 编译 Linux amd64 生产级单二进制..."
cd "$ROOT_DIR"
mkdir -p dist
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/cubexd-linux-amd64 ./cmd/cubexd
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/cubex-linux-amd64 ./cmd/cubex

echo "🚀 [3/3] 编译 Linux arm64 (RouterOS RB5009 / 软路由) 二进制..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/cubexd-linux-arm64 ./cmd/cubexd
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/cubex-linux-arm64 ./cmd/cubex

echo "🎉 全部架构编译完毕！产物已就绪在 dist 目录中。"
ls -lh dist/cubex*
