#!/bin/sh
# Run after filling .env and pointing LICENSE_DOMAIN at this Docker host.
set -eu
cd "$(dirname "$0")"
if [ ! -f .env ]; then
  cp .env.example .env
  chmod 600 .env
  echo 'Fill deployment/.env, then run deployment/start.sh again.'
  exit 1
fi
mkdir -p artifacts backups secrets
chmod 700 backups secrets
docker run --rm --user "$(id -u):$(id -g)" -v "$PWD:/setup" -w /setup python:3.13-alpine python scripts/configure.py .env
docker run --rm --user "$(id -u):$(id -g)" -v "$PWD:/setup" -w /setup python:3.13-alpine python scripts/check-credentials.py
docker compose build keygate backup
if [ ! -s secrets/backup-age-key.txt ]; then
  docker compose run --rm --no-deps --entrypoint age-keygen backup > secrets/backup-age-key.txt
  chmod 600 secrets/backup-age-key.txt
fi
backup_recipient=$(docker compose run --rm --no-deps --entrypoint age-keygen backup -y /secrets/backup-age-key.txt)
docker run --rm --user "$(id -u):$(id -g)" -e BACKUP_RECIPIENT="$backup_recipient" -v "$PWD:/setup" -w /setup python:3.13-alpine python -c 'import os,re,pathlib; p=pathlib.Path(".env"); s=p.read_text(); line="BACKUP_RECIPIENT="+os.environ["BACKUP_RECIPIENT"]; p.write_text(re.sub(r"^BACKUP_RECIPIENT=.*$",line,s,flags=re.M) if re.search(r"^BACKUP_RECIPIENT=",s,re.M) else s+"\n"+line+"\n"); p.chmod(0o600)'
docker compose up -d --wait postgres keygate caddy backup
initial_zip=$(find artifacts -maxdepth 1 -type f -name 'accessible-forms-pro-*.zip' | sort | tail -1)
if [ -z "$initial_zip" ]; then
  echo 'Place the packaged Pro ZIP in deployment/artifacts. The handoff bundle includes it.' >&2
  exit 1
fi
docker compose --profile tools run --rm publisher "$initial_zip"
docker compose --profile tools run --rm --entrypoint python publisher -c 'import json,os,urllib.request; u="https://"+os.environ["LICENSE_DOMAIN"]+"/ready"; r=json.load(urllib.request.urlopen(u,timeout=20)); assert r["ready"], r; print("Store is ready: "+u.removesuffix("/ready"))'
echo 'Store, private signed updates, checkout and encrypted backup scheduling are ready.'
echo 'Keep deployment/.env and deployment/secrets/backup-age-key.txt in your password manager.'
