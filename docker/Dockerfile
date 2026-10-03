# 在一个镜像构建阶段编译 API、业务 RPC 和 AI RPC 三个服务。
FROM golang:1.26-alpine AS builder

WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/applet-api ./application/applet/api \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/applet-rpc ./application/applet/rpc \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/applet-ai-rpc ./application/ai/rpc

FROM alpine:3.22 AS runtime
RUN apk add --no-cache ca-certificates tzdata gettext
WORKDIR /app
COPY --from=builder /out/applet-api /usr/local/bin/applet-api
COPY --from=builder /out/applet-rpc /usr/local/bin/applet-rpc
COPY --from=builder /out/applet-ai-rpc /usr/local/bin/applet-ai-rpc
COPY docker/entrypoint.sh /usr/local/bin/app-entrypoint
RUN chmod +x /usr/local/bin/app-entrypoint
ENV TZ=Asia/Shanghai
ENTRYPOINT ["/usr/local/bin/app-entrypoint"]

FROM runtime AS api
COPY docker/api.yaml.template /app/config.yaml.template
CMD ["applet-api", "-f", "/tmp/config.yaml"]

FROM runtime AS rpc
COPY docker/rpc.yaml.template /app/config.yaml.template
CMD ["applet-rpc", "-f", "/tmp/config.yaml"]

FROM runtime AS ai-rpc
COPY docker/ai-rpc.yaml.template /app/config.yaml.template
CMD ["applet-ai-rpc", "-f", "/tmp/config.yaml"]
