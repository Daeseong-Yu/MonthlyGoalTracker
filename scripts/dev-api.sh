#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
ENV_FILE="$ROOT_DIR/.env"

if [ -f "$ENV_FILE" ]; then
  set -a
  . "$ENV_FILE"
  set +a
fi

: "${APP_HOST:=127.0.0.1}"
: "${APP_PORT:=8080}"

if [ -z "${DATABASE_URL:-}" ]; then
  echo "DATABASE_URL is required. Copy .env.example to .env or export DATABASE_URL." >&2
  exit 1
fi

export APP_HOST APP_PORT DATABASE_URL

cd "$ROOT_DIR/backend"
go run ./cmd/api
