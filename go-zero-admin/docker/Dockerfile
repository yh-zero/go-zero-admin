# 打包镜像：api / admin-rpc / app-rpc / file-rpc / ai-rpc / tenant-rpc（迁移在宿主跑 goose，见 dev-env.md §2）
# 离线约束：Docker Hub 不可达，二进制由 data/scripts/build-images.ps1 在宿主交叉编译到 docker/.build/
# 基础镜像只用本地已有的 alpine:3.24.2；CA 证书从本地 golang:1.27.1-alpine 复制（ai-rpc 出站 HTTPS 需要）

FROM alpine:3.24.2 AS base
COPY --from=golang:1.27.1-alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# DSN loc=Asia/Shanghai 需要时区库；alpine 无 tzdata，复用 Go 发行版自带 zoneinfo.zip
COPY --from=golang:1.27.1-alpine /usr/local/go/lib/time/zoneinfo.zip /usr/share/zoneinfo.zip
ENV ZONEINFO=/usr/share/zoneinfo.zip
WORKDIR /app

FROM base AS api
COPY docker/.build/api /app/api
ENTRYPOINT ["/app/api", "-f", "etc/admin.yaml"]

FROM base AS admin-rpc
COPY docker/.build/admin-rpc /app/admin-rpc
ENTRYPOINT ["/app/admin-rpc", "-f", "etc/admin.yaml"]

FROM base AS app-rpc
COPY docker/.build/app-rpc /app/app-rpc
ENTRYPOINT ["/app/app-rpc", "-f", "etc/app.yaml"]

FROM base AS file-rpc
COPY docker/.build/file-rpc /app/file-rpc
ENTRYPOINT ["/app/file-rpc", "-f", "etc/file.yaml"]

FROM base AS ai-rpc
COPY docker/.build/ai-rpc /app/ai-rpc
ENTRYPOINT ["/app/ai-rpc", "-f", "etc/ai.yaml"]

FROM base AS tenant-rpc
COPY docker/.build/tenant-rpc /app/tenant-rpc
ENTRYPOINT ["/app/tenant-rpc", "-f", "etc/tenant.yaml"]
