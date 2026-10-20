#!/bin/sh
set -eu

# Templates quote strings as YAML scalars; preserve special password characters.
newline='
'
for name in REDIS_ADDR REDIS_PASSWORD ETCD_ADDR JWT_ACCESS_SECRET APP_JWT_ACCESS_SECRET APP_TRUSTED_PROXY_CIDRS RPC_AUTH_APP RPC_AUTH_TOKEN STORAGE_ENDPOINT STORAGE_EXTERNAL_ENDPOINT STORAGE_REGION STORAGE_ACCESS_KEY_ID STORAGE_SECRET_ACCESS_KEY STORAGE_PUBLIC_BUCKET STORAGE_PRIVATE_BUCKET MYSQL_USER MYSQL_PASSWORD MYSQL_ADDR MYSQL_DATABASE DEFAULT_USER_PASSWORD SMTP_HOST SMTP_USER SMTP_PASSWORD SMTP_FROM; do
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

# CONFIG_SECTIONS lists the config.yaml.template sections to render, in order.
: "${CONFIG_SECTIONS:?CONFIG_SECTIONS must list the config sections}"
section() {
    awk -v name="$1" '
        $0 == "# >>> " name { inside = 1; next }
        $0 == "# <<< " name { inside = 0 }
        inside' /tmp/config.source
}
# The template is checked in with CRLF endings; normalize before matching markers.
tr -d '\r' < /app/config.yaml.template > /tmp/config.source
: > /tmp/config.template
for name in $CONFIG_SECTIONS; do
    section "$name" >> /tmp/config.template
    printf '\n' >> /tmp/config.template
done
envsubst < /tmp/config.template > /tmp/config.yaml
exec "$@"
