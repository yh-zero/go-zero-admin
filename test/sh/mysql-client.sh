#!/bin/sh
# Runs inside the MySQL container. Credentials stay in its environment.
set -eu
export MYSQL_PWD="${MYSQL_ROOT_PASSWORD:?MYSQL_ROOT_PASSWORD is required}"
database="${MYSQL_DATABASE:?MYSQL_DATABASE is required}"
action="$1"
shift
case "$action" in
  query) exec mysql -uroot --default-character-set=utf8mb4 --batch --skip-column-names "$database" --execute="$1" ;;
  run) exec mysql -uroot --default-character-set=utf8mb4 --batch "$database" < "$1" ;;
  backup) exec mysqldump -uroot --default-character-set=utf8mb4 --single-transaction --quick --routines --events --triggers --hex-blob --set-gtid-purged=OFF --no-tablespaces --databases "$database" --result-file="$1" ;;
  *) echo "Unknown database command" >&2; exit 2 ;;
esac
