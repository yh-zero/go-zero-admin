# syntax=docker/dockerfile:1

FROM node:22-slim AS builder

ENV PNPM_HOME="/pnpm" \
    PATH="/pnpm:$PATH" \
    NODE_OPTIONS="--max-old-space-size=8192" \
    TZ="Asia/Shanghai" \
    CI="true"

RUN npm install --global corepack && corepack enable

WORKDIR /app
COPY . .

# Corepack reads the pinned pnpm version from package.json.
RUN --mount=type=cache,id=pnpm,target=/pnpm/store \
    corepack pnpm install --frozen-lockfile --store-dir /pnpm/store

ARG VITE_GLOB_API_URL=/api
ARG VITE_BASE=/

ENV VITE_GLOB_API_URL="${VITE_GLOB_API_URL}" \
    VITE_BASE="${VITE_BASE}"

# Loose mode passes the Docker build arguments through Turbo to Vite.
# Force prevents a cached build from reusing different API or base paths.
RUN corepack pnpm run build:antdv-next --force --env-mode=loose

FROM nginx:stable-alpine AS production

RUN rm /etc/nginx/conf.d/default.conf

COPY --from=builder /app/apps/web-antdv-next/dist /usr/share/nginx/html
COPY scripts/deploy/nginx.conf /etc/nginx/nginx.conf

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/healthz || exit 1

CMD ["nginx", "-g", "daemon off;"]
