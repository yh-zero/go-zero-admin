# A shared base stage downloads modules once; each target compiles and ships
# only its own binary.
FROM golang:1.27.1-alpine AS base
WORKDIR /src
RUN apk add --no-cache git
ENV GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=off
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .

FROM base AS build-api
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/applet-api ./application/applet/api

FROM base AS build-rpc
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/applet-rpc ./application/applet/rpc

FROM base AS build-ai-rpc
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/applet-ai-rpc ./application/ai/rpc

FROM alpine:3.22 AS runtime
RUN apk add --no-cache ca-certificates tzdata gettext
WORKDIR /app
COPY docker/entrypoint.sh /usr/local/bin/app-entrypoint
COPY docker/config.yaml.template /app/config.yaml.template
RUN chmod +x /usr/local/bin/app-entrypoint
ENV TZ=Asia/Shanghai
ENTRYPOINT ["/usr/local/bin/app-entrypoint"]

FROM runtime AS api
COPY --from=build-api /out/applet-api /usr/local/bin/applet-api
ENV CONFIG_SECTIONS=api
CMD ["applet-api", "-f", "/tmp/config.yaml"]

FROM runtime AS rpc
COPY --from=build-rpc /out/applet-rpc /usr/local/bin/applet-rpc
COPY data/ipregion/ /app/data/ipregion/
ENV CONFIG_SECTIONS="common rpc" \
    SERVICE_NAMESPACE=rpc \
    ETCD_NAMESPACE=applet.rpc
CMD ["applet-rpc", "-f", "/tmp/config.yaml"]

FROM runtime AS ai-rpc
COPY --from=build-ai-rpc /out/applet-ai-rpc /usr/local/bin/applet-ai-rpc
ENV CONFIG_SECTIONS="common ai-rpc" \
    SERVICE_NAMESPACE=ai-rpc \
    ETCD_NAMESPACE=ai.rpc
CMD ["applet-ai-rpc", "-f", "/tmp/config.yaml"]
