#!/bin/sh
set -eu

# Templates quote strings as YAML scalars; preserve special password characters.
newline='
'
for name in REDIS_ADDR REDIS_PASSWORD ETCD_ADDR JWT_ACCESS_SECRET OSS_ENDPOINT OSS_ACCESS_KEY_ID OSS_ACCESS_KEY_SECRET OSS_BUCKET_NAME MYSQL_USER MYSQL_PASSWORD MYSQL_ADDR MYSQL_DATABASE DEFAULT_USER_PASSWORD; do
    # The sentinel prevents command substitution from dropping trailing newlines.
    # Remove only the single newline printenv itself appends.
    value=$(printenv "$name" || true; printf '.')
    value=${value%.}
    value=${value%"$newline"}
    case "$value" in
        *"$(printf '\r')"*|*"$newline"*) echo "Configuration variable $name must be a single line" >&2; exit 1 ;;
    esac
    escaped=$(printf '%s' "$value" | sed "s/'/''/g")
    export "$name=$escaped"
done
envsubst < /app/config.yaml.template > /tmp/config.yaml
exec "$@"
