#!/bin/sh
# Disaster recovery only. Restore a trusted backup created by this deployment.
set -eu
umask 077
cd "$(dirname "$0")"
if [ "$#" -ne 2 ] || [ "$2" != '--confirm-replace' ]; then
  echo 'Usage: deployment/restore.sh backups/TIMESTAMP.tar.age --confirm-replace' >&2
  echo 'This replaces the store database, private artifacts and deployment configuration.' >&2
  exit 1
fi
backup_name=$(basename "$1")
case "$backup_name" in *[!A-Za-z0-9._-]*|'') echo 'Invalid backup filename.' >&2; exit 1;; esac
test -s "backups/$backup_name"
test -s secrets/backup-age-key.txt
docker compose build backup
restore_stage=$(mktemp -d "$PWD/.restore-XXXXXXXX")
trap 'rm -rf "$restore_stage"' EXIT HUP INT TERM
docker compose run --rm --no-deps --entrypoint sh -e RESTORE_UID="$(id -u)" -e RESTORE_GID="$(id -g)" -v "$restore_stage:/restore" backup -ec '
  set -o pipefail
  umask 077
  age -d -i /secrets/backup-age-key.txt "/backups/$1" |
    tar -xf - -C /restore ./database.dump ./store.env ./artifacts.tar.gz
  test -s /restore/database.dump
  test -s /restore/store.env
  test -s /restore/artifacts.tar.gz
  chown "$RESTORE_UID:$RESTORE_GID" /restore/database.dump /restore/store.env /restore/artifacts.tar.gz
' sh "$backup_name"
docker compose stop caddy keygate backup
cp "$restore_stage/store.env" .env
chmod 600 .env
docker compose up -d --wait postgres
docker compose run --rm --no-deps --entrypoint sh -v "$restore_stage:/restore:ro" backup -ec '
  pg_restore --clean --if-exists --single-transaction --exit-on-error --no-owner --no-privileges -d "$PGDATABASE" /restore/database.dump
'
docker compose -f compose.yaml -f compose.restore.yaml run --rm --no-deps --entrypoint sh -v "$restore_stage:/restore:ro" backup -ec '
  if tar -tzf /restore/artifacts.tar.gz | awk '\''/(^|\/)\.\.($|\/)|^\// {bad=1} END {exit !bad}'\''; then
    echo "Unsafe artifact archive." >&2; exit 1
  fi
  find /artifacts -mindepth 1 -maxdepth 1 -exec rm -rf {} +
  tar -xzf /restore/artifacts.tar.gz -C /artifacts
'
docker compose up -d --wait keygate caddy backup
echo 'Restored the database, configuration and private downloads. Check /ready and sign in as the original owner.'
