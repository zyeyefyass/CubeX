FROM alpine:3.20 AS runner

RUN apk add --no-cache ca-certificates tzdata nftables iproute2 iptables bind-tools curl && \
    mkdir -p /opt/CubeX/bin /opt/CubeX/etc /opt/CubeX/data

WORKDIR /opt/CubeX

COPY cubexd /opt/CubeX/bin/cubexd
COPY cubex /opt/CubeX/bin/cubex
RUN chmod +x /opt/CubeX/bin/* && ln -s /opt/CubeX/bin/cubex /usr/local/bin/cubex

EXPOSE 2026 7890 7892 7893 1053 9090

ENTRYPOINT ["/opt/CubeX/bin/cubexd", "-dir", "/opt/CubeX"]
