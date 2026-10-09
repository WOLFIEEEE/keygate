#!/bin/sh
set -eu
set -o pipefail
umask 077
backup_once() {
  test -n "${BACKUP_RECIPIENT:-}" || { echo 'Backup recipient is missing.' >&2; return 1; }
  backup_timestamp=$(date -u +%Y%m%dT%H%M%SZ)
  backup_stage=$(mktemp -d)
  pg_dump --format=custom --file="$backup_stage/database.dump" || { rm -rf "$backup_stage"; return 1; }
  cp /configuration/.env "$backup_stage/store.env" || { rm -rf "$backup_stage"; return 1; }
  # Published objects are immutable; a concurrent draft upload can be safely retried after restore.
  tar -czf "$backup_stage/artifacts.tar.gz" --exclude='.upload-*' -C /artifacts . || { rm -rf "$backup_stage"; return 1; }
  if ! tar -cf - -C "$backup_stage" . | age -r "$BACKUP_RECIPIENT" -o "/backups/$backup_timestamp.tar.age.partial"; then
    rm -rf "$backup_stage"
    rm -f "/backups/$backup_timestamp.tar.age.partial"
    return 1
  fi
  mv "/backups/$backup_timestamp.tar.age.partial" "/backups/$backup_timestamp.tar.age"
  rm -rf "$backup_stage"
  if [ -n "${BACKUP_REMOTE:-}" ]; then
    rclone --config /secrets/rclone.conf copy "/backups/$backup_timestamp.tar.age" "$BACKUP_REMOTE" || return 1
  fi
  find /backups -type f -name '*.tar.age' -mtime +14 -delete
  echo "Encrypted backup completed: $backup_timestamp"
}
if [ "${1:-}" = '--once' ]; then backup_once; exit; fi
while :; do
  if backup_once; then sleep 86400; else echo 'BACKUP FAILED; retrying in 5 minutes.' >&2; sleep 300; fi
done
