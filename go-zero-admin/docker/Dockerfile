# syntax=docker/dockerfile:1
# 多阶段构建：api / admin-rpc / app-rpc / file-rpc / ai-rpc / tenant-rpc / migrate
# TODO: 骨架阶段，待代码生成后验证构建

FROM golang:1.27.3-bookworm AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN set -e \
 && CGO_ENABLED=0 go build -ldflags='-s -w' -o /out/api        ./application/admin/api \
 && CGO_ENABLED=0 go build -ldflags='-s -w' -o /out/admin-rpc  ./application/admin/rpc \
 && CGO_ENABLED=0 go build -ldflags='-s -w' -o /out/app-rpc    ./application/app/rpc \
 && CGO_ENABLED=0 go build -ldflags='-s -w' -o /out/file-rpc   ./application/file/rpc \
 && CGO_ENABLED=0 go build -ldflags='-s -w' -o /out/ai-rpc     ./application/ai/rpc \
 && CGO_ENABLED=0 go build -ldflags='-s -w' -o /out/tenant-rpc ./application/tenant/rpc

FROM golang:1.27.3-bookworm AS tools
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.24.3

FROM debian:bookworm-slim AS base
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl netcat-openbsd \
 && rm -rf /var/lib/apt/lists/*
WORKDIR /app

FROM base AS api
COPY --from=builder /out/api /app/api
ENTRYPOINT ["/app/api", "-f", "etc/admin.yaml"]

FROM base AS admin-rpc
COPY --from=builder /out/admin-rpc /app/admin-rpc
ENTRYPOINT ["/app/admin-rpc", "-f", "etc/admin.yaml"]

FROM base AS app-rpc
COPY --from=builder /out/app-rpc /app/app-rpc
ENTRYPOINT ["/app/app-rpc", "-f", "etc/app.yaml"]

FROM base AS file-rpc
COPY --from=builder /out/file-rpc /app/file-rpc
ENTRYPOINT ["/app/file-rpc", "-f", "etc/file.yaml"]

FROM base AS ai-rpc
COPY --from=builder /out/ai-rpc /app/ai-rpc
ENTRYPOINT ["/app/ai-rpc", "-f", "etc/ai.yaml"]

FROM base AS tenant-rpc
COPY --from=builder /out/tenant-rpc /app/tenant-rpc
ENTRYPOINT ["/app/tenant-rpc", "-f", "etc/tenant.yaml"]

FROM base AS migrate
COPY --from=tools /go/bin/goose /usr/local/bin/goose
COPY data/db/migrations /migrations
COPY data/db/seeds /seeds
ENTRYPOINT ["goose", "-dir", "/migrations/business", "mysql", "root:${MYSQL_ROOT_PASSWORD}@tcp(mysql:3306)/${MYSQL_DATABASE}", "up"]
