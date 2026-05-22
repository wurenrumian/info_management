#!/usr/bin/env bash

set -euo pipefail

MODE="${1:-all}"

case "$MODE" in
  all|notification|certificate)
    ;;
  -h|--help|help)
    echo "Usage: $0 [all|notification|certificate]"
    exit 0
    ;;
  *)
    echo "Invalid mode: $MODE"
    echo "Usage: $0 [all|notification|certificate]"
    exit 1
    ;;
esac

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SQL_FILE="$ROOT_DIR/scripts/dev/init_db.sql"

if [[ ! -f "$SQL_FILE" ]]; then
  echo "sql file not found: $SQL_FILE"
  exit 1
fi

compose_cmd=()
if docker compose version >/dev/null 2>&1; then
  compose_cmd=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
  compose_cmd=(docker-compose)
fi

run_via_compose() {
  local seed_mode="$1"
  local -a cmd=("${compose_cmd[@]}" exec -T kingbase ksql -U system -d kingbase -p 54321 -v ON_ERROR_STOP=1)
  if [[ -n "$seed_mode" ]]; then
    cmd+=(-v "seed_mode=$seed_mode")
  fi
  "${cmd[@]}" < "$SQL_FILE"
}

run_via_psql() {
  local seed_mode="$1"
  if [[ -z "${DATABASE_DSN:-}" ]]; then
    echo "DATABASE_DSN is required when running outside docker compose"
    exit 1
  fi
  if ! command -v psql >/dev/null 2>&1; then
    echo "psql command not found"
    exit 1
  fi

  if [[ -n "$seed_mode" ]]; then
    psql "$DATABASE_DSN" -v ON_ERROR_STOP=1 -v "seed_mode=$seed_mode" -f "$SQL_FILE"
  else
    psql "$DATABASE_DSN" -v ON_ERROR_STOP=1 -f "$SQL_FILE"
  fi
}

if [[ ${#compose_cmd[@]} -gt 0 ]]; then
  case "$MODE" in
    all) run_via_compose "" ;;
    notification) run_via_compose "notification" ;;
    certificate) run_via_compose "certificate" ;;
  esac
else
  case "$MODE" in
    all) run_via_psql "" ;;
    notification) run_via_psql "notification" ;;
    certificate) run_via_psql "certificate" ;;
  esac
fi
