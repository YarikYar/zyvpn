#!/usr/bin/env bash
# Deploy script run on the docker host (192.168.1.223) by CI or manually.
#
# Idempotent: pulls latest main, rebuilds changed images, recreates only
# affected services. The .env file with secrets is provisioned out of band
# and lives at /opt/compose/zyvpn/.env (not in git).

set -euo pipefail

REPO_DIR="/opt/compose/zyvpn"
BRANCH="${DEPLOY_BRANCH:-main}"

cd "$REPO_DIR"

echo "==> fetching origin/$BRANCH"
git fetch --quiet origin "$BRANCH"
git checkout --quiet "$BRANCH"
git reset --hard "origin/$BRANCH"

if [ ! -f .env ]; then
    echo "ERROR: $REPO_DIR/.env is missing — provision it manually with secrets" >&2
    exit 1
fi

echo "==> building images"
docker compose build

echo "==> applying compose"
docker compose up -d --remove-orphans

echo "==> pruning dangling images"
docker image prune -f >/dev/null

echo "==> status"
docker compose ps

echo "==> done"
