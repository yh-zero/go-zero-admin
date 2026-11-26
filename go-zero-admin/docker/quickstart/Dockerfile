# 从仓库根目录构建后端和前端产物。
FROM golang:1.27.3-alpine AS go-build
WORKDIR /src
ENV CGO_ENABLED=0 GOMAXPROCS=2
# 单一代理被阻断时仍允许备用镜像和直连。
ENV GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct
COPY go-zero-admin/go.mod go-zero-admin/go.sum ./
RUN go mod download
COPY go-zero-admin/application/ ./application/
COPY go-zero-admin/pkg/ ./pkg/
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -mod=readonly -p 1 -trimpath -ldflags="-s -w" -o /out/api ./application/admin/api && \
    go build -mod=readonly -p 1 -trimpath -ldflags="-s -w" -o /out/admin-rpc ./application/admin/rpc && \
    go build -mod=readonly -p 1 -trimpath -ldflags="-s -w" -o /out/file-rpc ./application/file/rpc

FROM alpine:3.24.2 AS core
WORKDIR /app
COPY --from=go-build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=go-build /usr/local/go/lib/time/zoneinfo.zip /usr/share/zoneinfo.zip
ENV ZONEINFO=/usr/share/zoneinfo.zip
COPY --from=go-build /out/ ./

FROM node:24-bookworm-slim AS web-build
WORKDIR /src/go-zero-admin-react
COPY go-zero-admin-react/package.json go-zero-admin-react/pnpm-lock.yaml ./
RUN npm install --global "$(node -p 'require("./package.json").packageManager')" && \
    pnpm install --frozen-lockfile
COPY go-zero-admin-react/ ./
RUN GOMAXPROCS=2 RAYON_NUM_THREADS=2 pnpm run build

FROM nginx:1.27-alpine AS web
COPY go-zero-admin/docker/quickstart/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=web-build /src/go-zero-admin-react/dist/ /usr/share/nginx/html/
