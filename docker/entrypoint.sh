#!/bin/sh
set -eu

envsubst < /app/config.yaml.template > /tmp/config.yaml
exec "$@"
