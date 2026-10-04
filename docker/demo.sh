#!/bin/sh
# Linux public demo entrypoint; never clears volumes or modifies host services.
set -eu
action=${1:-status}
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"
# Use this deployment's persisted selectors, never a caller's other environment.
# prepare still accepts DEMO_PROJECT_NAME before persisting the new settings.
if [ "$action" != prepare ]; then
    unset COMPOSE_PROJECT_NAME DEMO_PROJECT_NAME DEPLOY_ROOT
fi
export GOZERO_DEPLOY_COMPOSE_FILE=docker/demo-compose.yml
compose() {
    (unset COMPOSE_PROJECT_NAME DEMO_PROJECT_NAME DEPLOY_ROOT
     docker compose --env-file docker/.env.deploy -f "$GOZERO_DEPLOY_COMPOSE_FILE" "$@")
}
case "$action" in
  prepare)
    # Only prepare in an explicitly selected, empty/previously bootstrapped demo.
    python3 docker/demo-bootstrap.py prepare "${2:?Provide absolute project root}" "${3:?Provide domain}" "${4:?Provide IP}"
    compose config --quiet
    ;;
  init)
    compose config --quiet
    # Runtime images contain no Go/Node toolchains and no deployment env file.
    compose build api rpc ai-rpc
    sh test/sh/db.sh init deploy
    python3 docker/demo-bootstrap.py seed
    ;;
  start)
    python3 docker/demo-bootstrap.py check
    compose up -d api rpc ai-rpc web
    compose ps
    ;;
  status)
    compose ps
    sh test/sh/db.sh status deploy
    ;;
  stop) compose stop web api ai-rpc rpc ;;
  backup) sh test/sh/db.sh backup deploy ;;
  *) echo 'Usage: sh docker/demo.sh prepare ROOT DOMAIN IP | init | start | status | stop | backup' >&2; exit 2 ;;
esac
