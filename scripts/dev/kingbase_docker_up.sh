#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

CONTAINER_NAME="${KINGBASE_CONTAINER_NAME:-kingbase-test}"
KINGBASE_IMAGE="${KINGBASE_IMAGE:-kingbase_v009r001c010b0004_single_x86:v1}"
HOST_PORT="${KINGBASE_HOST_PORT:-54321}"
CONTAINER_PORT="${KINGBASE_CONTAINER_PORT:-54321}"
LICENSE_FILE="${KINGBASE_LICENSE_FILE:-$ROOT_DIR/license_71193_0.dat}"
EXTRA_RUN_ARGS="${KINGBASE_DOCKER_RUN_ARGS:-}"
DB_USER="${KINGBASE_DB_USER:-system}"
DB_PASSWORD="${KINGBASE_DB_PASSWORD:-123456}"
DB_NAME="${KINGBASE_DB_NAME:-test}"

if ! command -v docker >/dev/null 2>&1; then
  echo "docker command not found."
  exit 1
fi

if [[ -z "$KINGBASE_IMAGE" ]]; then
  echo "KINGBASE_IMAGE is empty."
  echo "Please set your local Kingbase image first, for example:"
  echo "  export KINGBASE_IMAGE='your-kingbase-image:tag'"
  echo "Then rerun: ./scripts/dev/kingbase_docker_up.sh"
  exit 1
fi

license_mount_args=()
if [[ -f "$LICENSE_FILE" ]]; then
  license_mount_args=(-v "$LICENSE_FILE:/home/system/license_71193_0.dat:ro")
  echo "Using license file: $LICENSE_FILE"
else
  echo "License file not found, continue without mounting: $LICENSE_FILE"
fi

if docker ps -a --format '{{.Names}}' | grep -qx "$CONTAINER_NAME"; then
  echo "Container already exists, removing: $CONTAINER_NAME"
  docker rm -f "$CONTAINER_NAME" >/dev/null
fi

echo "Starting Kingbase container..."
docker run -d \
  --name "$CONTAINER_NAME" \
  -p "$HOST_PORT:$CONTAINER_PORT" \
  "${license_mount_args[@]}" \
  $EXTRA_RUN_ARGS \
  "$KINGBASE_IMAGE" >/dev/null

echo "Container started: $CONTAINER_NAME"
echo "Port mapping: 127.0.0.1:$HOST_PORT -> $CONTAINER_PORT"

echo "Bootstrapping database account to match development standard..."
for i in $(seq 1 30); do
  if docker exec "$CONTAINER_NAME" bash -lc "ksql -U $DB_USER -d $DB_NAME -c \"select 1;\"" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

docker exec "$CONTAINER_NAME" bash -lc "ksql -U $DB_USER -d $DB_NAME -c \"ALTER USER $DB_USER WITH PASSWORD '$DB_PASSWORD';\"" >/dev/null
echo "Applied password for user '$DB_USER'."
echo
echo "Suggested DSN:"
echo "  export DATABASE_DSN='host=127.0.0.1 port=$HOST_PORT user=$DB_USER password=$DB_PASSWORD dbname=$DB_NAME sslmode=disable'"
echo "  export KINGBASE_DSN=\"\$DATABASE_DSN\""
