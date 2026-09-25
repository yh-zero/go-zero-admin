#!/bin/sh
# Usage: sh test/sh/db.sh init|migrate|backup|status [development|deploy]
set -eu
action="${1:-status}"
environment="${2:-development}"
case "$action" in init|migrate|backup|status) ;; *) echo 'Unknown action' >&2; exit 2 ;; esac
case "$environment" in development|deploy) ;; *) echo 'Unknown environment' >&2; exit 2 ;; esac
project_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$project_root"
compose() {
    if [ "$environment" = deploy ]; then
        docker compose --env-file docker/.env.deploy -f docker/deploy-compose.yml "$@"
    else
        docker compose -f docker-compose.yml "$@"
    fi
}
container=''
locked=false
helper="/tmp/gozero-db-client-$$.sh"
cleanup() {
    if [ "$locked" = true ]; then docker exec "$container" rmdir /tmp/gozero-db-migrate.lock >/dev/null || true; fi
    if [ -n "$container" ]; then docker exec "$container" rm -f -- "$helper" >/dev/null || true; fi
}
trap cleanup EXIT
client() {
    if [ -n "${GOZERO_DB_NAME:-}" ]; then
        docker exec -e "MYSQL_DATABASE=$GOZERO_DB_NAME" "$container" sh "$helper" "$@"
    else
        docker exec "$container" sh "$helper" "$@"
    fi
}
query() { client query "$1"; }
migration_hash() { bom=$(printf '\357\273\277'); sed "1s/^$bom//; s/\r$//" "$1" | sha256sum | cut -d ' ' -f 1; }
backup() {
    mkdir -p bin/db-backups
    name="gozero-$environment-$(date +%Y%m%d-%H%M%S)-$$.sql"
    target="$project_root/bin/db-backups/$name"
    [ ! -e "$target" ] || { echo 'Backup already exists' >&2; exit 1; }
    client backup "/tmp/$name"
    docker cp "$container:/tmp/$name" "$target.partial" >/dev/null
    grep -q '^-- Dump completed on ' "$target.partial"
    grep -q '^CREATE TABLE ' "$target.partial"
    remote_hash=$(docker exec "$container" sha256sum "/tmp/$name" | cut -d ' ' -f 1)
    local_hash=$(sha256sum "$target.partial" | cut -d ' ' -f 1)
    [ "$remote_hash" = "$local_hash" ] || { echo 'Backup checksum mismatch' >&2; exit 1; }
    mv -- "$target.partial" "$target"
    printf '%s  %s\n' "$local_hash" "$name" > "$target.sha256"
    docker exec "$container" rm -f -- "/tmp/$name"
    printf 'Backup saved: %s\n' "$target"
}
if [ "$action" = init ]; then
    if [ "$environment" = development ]; then
        compose up -d --wait --wait-timeout 180 mysql redis etcd swagger-ui
    else
        compose up -d --wait --wait-timeout 180 mysql redis etcd
    fi
fi
container=$(compose ps -q mysql)
[ -n "$container" ] || { echo 'Start MySQL first, or use init.' >&2; exit 1; }
docker cp test/sh/mysql-client.sh "$container:$helper" >/dev/null
query 'SELECT 1;' >/dev/null
printf 'Database: %s\n' "$(query 'SELECT DATABASE();')"
if [ "$action" = backup ]; then backup; exit 0; fi
table_exists=$(query "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='schema_migrations';")
pending=false
for file in data/db/migrations/*.sql; do
    name=$(basename "$file")
    case "$name" in *[!a-zA-Z0-9_.-]*) echo 'Unsupported migration filename' >&2; exit 1 ;; esac
    hash=$(migration_hash "$file")
    existing=''
    if [ "$table_exists" = 1 ]; then existing=$(query "SELECT checksum FROM schema_migrations WHERE filename='$name';"); fi
    if [ -n "$existing" ]; then
        [ "$existing" = "$hash" ] || { echo "Applied migration modified: $name. Add a new migration." >&2; exit 1; }
        printf 'APPLIED %s\n' "$name"
    else
        pending=true
        printf 'PENDING %s\n' "$name"
    fi
done
if [ "$action" = status ] || [ "$pending" = false ]; then exit 0; fi
docker exec "$container" mkdir /tmp/gozero-db-migrate.lock
locked=true
backup
query 'CREATE TABLE IF NOT EXISTS schema_migrations (filename VARCHAR(191) NOT NULL PRIMARY KEY, checksum CHAR(64) NOT NULL, applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP) ENGINE=InnoDB;' >/dev/null
for file in data/db/migrations/*.sql; do
    name=$(basename "$file")
    hash=$(migration_hash "$file")
    existing=$(query "SELECT checksum FROM schema_migrations WHERE filename='$name';")
    if [ -n "$existing" ]; then
        [ "$existing" = "$hash" ] || { echo "Applied migration checksum mismatch: $name" >&2; exit 1; }
        continue
    fi
    remote="/tmp/gozero-migration-$name"
    docker cp "$file" "$container:$remote" >/dev/null
    client run "$remote"
    query "INSERT INTO schema_migrations(filename,checksum) VALUES('$name','$hash');" >/dev/null
    docker exec "$container" rm -f -- "$remote"
    printf 'APPLIED %s\n' "$name"
done
echo 'Migration complete. Restart RPC/API and reload frontend permissions.'
