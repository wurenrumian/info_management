#!/usr/bin/env bash

# This script is intended to be sourced:
#   source ./scripts/dev/export.sh

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  echo "Please source this script instead of executing it."
  echo "Run: source ./scripts/dev/export.sh"
  exit 1
fi

DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-54321}"
DB_USER="${DB_USER:-system}"
DB_PASSWORD="${DB_PASSWORD:-123456}"
DB_NAME="${DB_NAME:-test}"
DB_SSLMODE="${DB_SSLMODE:-disable}"

export DATABASE_DSN="${DATABASE_DSN:-host=${DB_HOST} port=${DB_PORT} user=${DB_USER} password=${DB_PASSWORD} dbname=${DB_NAME} sslmode=${DB_SSLMODE}}"
export KINGBASE_DSN="${KINGBASE_DSN:-$DATABASE_DSN}"

echo "Loaded development database variables:"
echo "  DATABASE_DSN = [$DATABASE_DSN]"
echo "  KINGBASE_DSN = [$KINGBASE_DSN]"
