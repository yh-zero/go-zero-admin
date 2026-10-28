#!/bin/sh
# go-zero-admin 生产部署入口。只依赖 docker compose 与 POSIX shell。
set -eu

cd "$(dirname "$0")/.."
ENV_FILE=docker/.env.deploy
BACKUP_DIR=data/db/backups
MIGRATION_DIR=data/db/migrations

dc() { docker compose --env-file "$ENV_FILE" -f docker/deploy-compose.yml "$@"; }
say() { printf '%s\n' "$*"; }
die() { printf '错误: %s\n' "$*" >&2; exit 1; }

need_env() { [ -f "$ENV_FILE" ] || die "缺少 $ENV_FILE，先运行 docker/deploy.sh init"; }
env_value() { sed -n "s/^$1=//p" "$ENV_FILE" | tail -n 1; }

sha256_of() {
    if command -v sha256sum >/dev/null 2>&1; then
        tr -d '\r' < "$1" | sha256sum | cut -d' ' -f1
    else
        tr -d '\r' < "$1" | shasum -a 256 | cut -d' ' -f1
    fi
}

random_secret() { od -An -tx1 -N32 /dev/urandom | tr -d ' \n'; }

# 在容器内执行一条 SQL；SQL 由本脚本展开后作为单个参数传入。
mysql_run() {
    dc exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -N -B -uroot "$MYSQL_DATABASE" -e "$1"' sh "$1"
}

applied_dump() {
    if [ "$(mysql_run "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='schema_migrations'")" = "0" ]; then
        return 0
    fi
    mysql_run "SELECT filename, checksum FROM schema_migrations ORDER BY filename"
}

cmd_init() {
    [ -e "$ENV_FILE" ] && die "$ENV_FILE 已存在；如需重建，先备份再手动删除"
    mysql_pw=$(random_secret)
    jwt=$(random_secret)
    app_jwt=$(random_secret)
    rpc=$(random_secret)
    user_pw=$(random_secret | cut -c1-24)
    storage_root_secret=$(random_secret)
    storage_app_secret=$(random_secret)
    cat > "$ENV_FILE" <<EOF
# 由 docker/deploy.sh init 生成；不要提交，不要复制进镜像。

# --- 必填（已自动生成强随机值）---
MYSQL_ROOT_PASSWORD=$mysql_pw
MYSQL_DATABASE=gozero-admin
JWT_ACCESS_SECRET=$jwt
APP_JWT_ACCESS_SECRET=$app_jwt
RPC_AUTH_TOKEN=$rpc
DEFAULT_USER_PASSWORD=$user_pw
RUSTFS_ACCESS_KEY=GOZEROSTORAGEROOT
RUSTFS_SECRET_KEY=$storage_root_secret
STORAGE_ACCESS_KEY_ID=GOZEROSTORAGEAPP
STORAGE_SECRET_ACCESS_KEY=$storage_app_secret

# --- 端口与路径 ---
JWT_ACCESS_EXPIRE=7200
APP_JWT_ACCESS_EXPIRE=900
APP_JWT_REFRESH_EXPIRE=2592000
RPC_AUTH_APP=applet-api
API_PORT=127.0.0.1:7001
WEB_PORT=127.0.0.1:8080
SWAGGER_PORT=8081
SWAGGER_API_URL=http://api:7001
FRONTEND_CONTEXT=../../go-zero-admin-vben
STORAGE_PORT=127.0.0.1:9000
STORAGE_CONSOLE_PORT=127.0.0.1:9001
# 对外部署时改为真实 HTTPS 文件域名及前端来源（协议+域名+端口，无路径）。
STORAGE_EXTERNAL_ENDPOINT=http://127.0.0.1:9000
STORAGE_CORS_ORIGINS=http://127.0.0.1:8080

# --- 可选：留空即关闭 ---
REDIS_PASSWORD=

# 只填实际反向代理容器的 IP/32 或 /128，多个用逗号分隔；留空表示不信任转发头。
APP_TRUSTED_PROXY_CIDRS=

SMTP_HOST=
SMTP_PORT=465
SMTP_USER=
SMTP_PASSWORD=
SMTP_FROM=
APP_EMAIL_VERIFICATION_REQUIRED=false

API_KEY_DEEPSEEK=
API_KEY_QWEN=
EOF
    chmod 600 "$ENV_FILE"
say "已生成 $ENV_FILE（权限 600）"
say "初始管理员密码重置值 DEFAULT_USER_PASSWORD=$user_pw"
say "下一步: docker/deploy.sh check"
}

cmd_check() {
    need_env
    if grep -q 'replace-with' "$ENV_FILE"; then
        die "$ENV_FILE 仍有占位值 replace-with-，请替换"
    fi
    for key in MYSQL_ROOT_PASSWORD JWT_ACCESS_SECRET APP_JWT_ACCESS_SECRET RPC_AUTH_TOKEN RUSTFS_ACCESS_KEY RUSTFS_SECRET_KEY STORAGE_ACCESS_KEY_ID STORAGE_SECRET_ACCESS_KEY STORAGE_EXTERNAL_ENDPOINT STORAGE_CORS_ORIGINS; do
        [ -n "$(env_value "$key")" ] || die "$key 为空"
    done
    [ "$(env_value JWT_ACCESS_SECRET)" = "$(env_value APP_JWT_ACCESS_SECRET)" ] && die "JWT_ACCESS_SECRET 与 APP_JWT_ACCESS_SECRET 必须不同"
    [ "$(env_value RUSTFS_ACCESS_KEY)" = "$(env_value STORAGE_ACCESS_KEY_ID)" ] && die "RustFS 管理身份和应用身份必须不同"
    for key in JWT_ACCESS_SECRET APP_JWT_ACCESS_SECRET RPC_AUTH_TOKEN; do
        value=$(env_value "$key")
        if [ "${#value}" -lt 32 ]; then say "警告: $key 短于 32 个字符，建议改用更长的随机值"; fi
    done
    dc config --quiet
say "配置校验通过"
}

cmd_up() {
    cmd_check
    dc up -d --wait mysql redis etcd
    dc up -d --build --wait rpc ai-rpc api web
    dc ps
say "页面: http://127.0.0.1:$(env_value WEB_PORT | sed 's/.*://')"
}

cmd_import() {
    need_env
    [ -f data/db/gozero-admin.sql ] || die '缺少 data/db/gozero-admin.sql'
    dc up -d --wait mysql
    tables=$(mysql_run "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE()" | tr -d "\r ")
    [ "$tables" = "0" ] || die "数据库已有 $tables 张表；全量 SQL 只能导入空库"
    dc cp data/db/gozero-admin.sql mysql:/tmp/gozero-admin.sql
    dc exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql --user=root --database="$MYSQL_DATABASE" < /tmp/gozero-admin.sql'
say "已导入 data/db/gozero-admin.sql"
}

# 只读比对：列出待应用 / 已应用 / 校验和不符。
cmd_migrate_check() {
    need_env
    dc up -d --wait mysql >/dev/null
    tmp=$(mktemp)
    applied_dump > "$tmp"
    [ -s "$tmp" ] || say 'schema_migrations 为空（未导入任何迁移记录）'
    for file in "$MIGRATION_DIR"/*.sql; do
        [ -e "$file" ] || continue
        name=$(basename "$file")
        want=$(sha256_of "$file")
        have=$(awk -v n="$name" '$1 == n { print $2 }' "$tmp")
        if [ -z "$have" ]; then
say "待应用      $name"
        elif [ "$have" = "$want" ]; then
say "已应用      $name"
        else
say "校验和不符  $name  已登记=$have 当前=$want"
        fi
    done
    rm -f "$tmp"
}

cmd_backup() {
    mkdir -p "$BACKUP_DIR"
    stamp=$(date +%Y%m%d-%H%M%S)
    file="$BACKUP_DIR/gozero-admin-full-$stamp.sql"
    dc exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysqldump --single-transaction --routines --triggers --databases "$MYSQL_DATABASE"' > "$file"
    printf '%s  %s\n' "$(sha256_of "$file")" "$(basename "$file")" > "$file.sha256"
say "备份: $file"
}

cmd_upgrade() {
    need_env
say "1/4 停止写库服务（api / ai-rpc / rpc）"
    dc stop api ai-rpc rpc
say "2/4 备份数据库"
    cmd_backup
say "3/4 比对迁移记录"
    tmp=$(mktemp)
    applied_dump > "$tmp"
    pending=0
    for file in "$MIGRATION_DIR"/*.sql; do
        [ -e "$file" ] || continue
        name=$(basename "$file")
        want=$(sha256_of "$file")
        have=$(awk -v n="$name" '$1 == n { print $2 }' "$tmp")
        if [ -n "$have" ] && [ "$have" != "$want" ]; then
            rm -f "$tmp"
die "$name 校验和不符（已登记 $have / 当前 $want）；请先核实数据库实际结构，不要直接重跑"
        fi
        [ -z "$have" ] || continue
        pending=$((pending + 1))
say "4/4 待应用: $name"
        printf "    执行并登记该迁移？[y/N] "
        answer=''
        read -r answer || answer=''
        case "$answer" in
            y|Y)
                if [ "$name" = '20261020_user_avatar_references.sql' ]; then
                    avatar_bucket=$(env_value STORAGE_PUBLIC_BUCKET)
                    avatar_bucket=${avatar_bucket:-gozero-public}
                    case "$avatar_bucket" in *[!a-z0-9.-]*) die 'STORAGE_PUBLIC_BUCKET 格式无效';; esac
                    { printf "SET @avatar_public_bucket='%s';\n" "$avatar_bucket"; cat "$file"; } | dc exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql --user=root --database="$MYSQL_DATABASE"' || die "$name 执行失败，已停止"
                else
                    dc exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql --user=root --database="$MYSQL_DATABASE"' < "$file" || die "$name 执行失败，已停止；核实部分执行状态后再处理，不要直接重跑"
                fi
                mysql_run "INSERT INTO schema_migrations(filename,checksum,applied_at) VALUES('$name','$want',NOW())"
say "    已应用并登记: $name"
                ;;
            *)
say "    已跳过: $name"
                ;;
        esac
    done
    rm -f "$tmp"
    [ "$pending" -gt 0 ] || say "没有待应用的迁移"
    dc up -d --build --wait rpc ai-rpc api web
    dc ps
}

cmd_rollback() {
    need_env
    [ $# -ge 1 ] || die "用法: docker/deploy.sh rollback <备份文件>"
    [ -f "$1" ] || die "找不到 $1"
say "即将用 $1 覆盖当前业务库；输入 yes 继续"
    answer=''
    read -r answer || answer=''
    [ "$answer" = "yes" ] || { say "已取消"; exit 1; }
    dc stop api ai-rpc rpc
    dc exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql --user=root' < "$1"
say "已恢复；检查后运行 docker/deploy.sh up"
}

cmd_swagger() { dc --profile tools up -d --wait swagger-ui; }

usage() {
    cat <<'EOF'
用法: docker/deploy.sh <命令>

  init            生成 docker/.env.deploy（随机密钥，权限 600）
  check           校验环境文件与 compose 配置
  up              校验后构建并启动全部服务
  import          向空库导入 data/db/gozero-admin.sql（非空库拒绝）
  migrate-check   只读比对 schema_migrations 与迁移文件
  upgrade         停写服务 → 备份 → 逐条确认执行迁移 → 启动
  rollback <文件> 用备份覆盖业务库
  swagger         启动 Swagger UI（profile tools）
  status          查看服务状态
  logs [服务]     查看日志
  stop            停止应用服务（保留数据卷）
EOF
}

command=${1:-help}
shift 2>/dev/null || true
case "$command" in
    init) cmd_init ;;
    check) cmd_check ;;
    up) cmd_up ;;
    import) cmd_import ;;
    migrate-check) cmd_migrate_check ;;
    upgrade) cmd_upgrade ;;
    rollback) cmd_rollback "$@" ;;
    swagger) cmd_swagger ;;
    status) dc ps ;;
    logs) dc logs --tail=100 "${1:-api}" ${2:-} ;;
    stop) dc stop ;;
    help|-h|--help) usage ;;
    *) usage; exit 1 ;;
esac
