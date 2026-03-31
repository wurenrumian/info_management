#!/usr/bin/env bash
set -euo pipefail

CONTAINER_NAME="${KINGBASE_CONTAINER_NAME:-kingbase-test}"

if ! command -v docker >/dev/null 2>&1; then
  echo "docker command not found."
  exit 1
fi

if docker ps -a --format '{{.Names}}' | grep -qx "$CONTAINER_NAME"; then
  docker rm -f "$CONTAINER_NAME" >/dev/null
  echo "Removed container: $CONTAINER_NAME"
else
  echo "Container not found: $CONTAINER_NAME"
fi
