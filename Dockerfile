# syntax=docker/dockerfile:1
# ═════════════════════════════════════════════════════════════════════════════
# CubeX High-Performance Dual-Engine Gateway Dockerfile
# Multi-Arch Support: linux/amd64, linux/arm64
# ═════════════════════════════════════════════════════════════════════════════

# Stage 1: Build Web Frontend
FROM --platform=$BUILDPLATFORM node:22-alpine AS web-builder
WORKDIR /build/web
COPY web/package*.json ./
RUN npm install --no-audit --no-fund
COPY web/ ./
RUN npm run build

# Stage 2: Build Multi-Arch Gateway Daemon (cubexd)
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS server-builder
ENV GOTOOLCHAIN=auto
ARG TARGETOS
ARG TARGETARCH
WORKDIR /build/gateway
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /build/cmd/cubexd/dist ./cmd/cubexd/dist
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/cubexd ./cmd/cubexd

# Stage 3: Minimal Runtime Container
FROM --platform=$TARGETPLATFORM alpine:3.21 AS runtime
ARG TARGETARCH
ARG CUBEX_VERSION=v9.99

# Install essential network tools for transparent proxy, TUN, and routing (both cores are pure static ELFs)
RUN apk add --no-cache \
    iptables \
    ip6tables \
    nftables \
    iproute2 \
    curl \
    ca-certificates \
    tzdata \
    procps \
    && rm -rf /var/cache/apk/*

# OCI Image Brand Annotations & Standard Metadata
LABEL org.opencontainers.image.title="CubeX" \
      org.opencontainers.image.description="CubeX ${CUBEX_VERSION} · 双引擎宿主共生零感网关 · 多架构通用兼容" \
      org.opencontainers.image.vendor="CubeX Lab" \
      org.opencontainers.image.version="${CUBEX_VERSION}" \
      org.opencontainers.image.url="https://hub.docker.com/r/fyasshole/cubex" \
      org.opencontainers.image.source="https://github.com/zyeyefyass/CubeX" \
      org.opencontainers.image.icon="https://raw.githubusercontent.com/zyeyefyass/CubeX/main/assets/logo.png"

WORKDIR /app

# Create necessary directories
RUN mkdir -p /app/bin /app/etc /app/data /app/defaults/etc /app/defaults/data

# Copy built Gateway Daemon
COPY --from=server-builder /out/cubexd /app/bin/cubexd
COPY meta.json /app/meta.json

# Copy pre-compiled Multi-Arch Cores (mihomo & sing-box) based on TARGETARCH
COPY bin/${TARGETARCH}/mihomo /app/bin/mihomo
COPY bin/${TARGETARCH}/sing-box /app/bin/sing-box

# Copy default configurations, rulesets (MRS / SRS) and geodata
COPY etc/GeoIP.dat etc/geosite.dat /app/defaults/etc/
COPY etc/rules/ /app/defaults/etc/rules/
COPY packaging/defaults/data/ /app/defaults/data/

# Copy Entrypoint
COPY scripts/docker/entrypoint.sh /app/entrypoint.sh

RUN chmod +x /app/bin/cubexd /app/bin/mihomo /app/bin/sing-box /app/entrypoint.sh

# Web UI & TProxy / Mixed / REST API ports
EXPOSE 2026 7890 7891 7892 7893 7894 7895 9090 1053/udp 1053/tcp

VOLUME ["/app/data", "/app/etc"]

ENTRYPOINT ["/app/entrypoint.sh"]
