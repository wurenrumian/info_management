#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# 挂载到 backend 容器内 /data/uploads 的宿主机目录
UPLOAD_DIR="${UPLOAD_DIR:-$ROOT_DIR/data/uploads}"

# backend 容器当前运行用户为 appuser(10001:10001)
UPLOAD_UID="${UPLOAD_UID:-10001}"
UPLOAD_GID="${UPLOAD_GID:-10001}"
UPLOAD_MODE="${UPLOAD_MODE:-0775}"
UPLOAD_SUBDIRS="${UPLOAD_SUBDIRS:-images avatars knowledge }"

run_with_optional_sudo() {
  if "$@" >/dev/null 2>&1; then
    return 0
  fi
  if command -v sudo >/dev/null 2>&1; then
    sudo "$@"
    return 0
  fi
  return 1
}

echo "Preparing upload directory..."
echo "  UPLOAD_DIR=$UPLOAD_DIR"
echo "  UPLOAD_UID=$UPLOAD_UID"
echo "  UPLOAD_GID=$UPLOAD_GID"
echo "  UPLOAD_MODE=$UPLOAD_MODE"
echo "  UPLOAD_SUBDIRS=$UPLOAD_SUBDIRS"

if ! run_with_optional_sudo mkdir -p "$UPLOAD_DIR"; then
  echo "[ERROR] failed to create upload dir: $UPLOAD_DIR"
  exit 1
fi

for subdir in $UPLOAD_SUBDIRS; do
  if ! run_with_optional_sudo mkdir -p "$UPLOAD_DIR/$subdir"; then
    echo "[ERROR] failed to create upload subdir: $UPLOAD_DIR/$subdir"
    exit 1
  fi
done

# 目录属主对齐容器运行用户，避免 'upload failed'（Permission denied）
if ! run_with_optional_sudo chown -R "${UPLOAD_UID}:${UPLOAD_GID}" "$UPLOAD_DIR"; then
  echo "[WARN] failed to chown $UPLOAD_DIR, fallback to permissive mode 0777 for dev."
  UPLOAD_MODE="0777"
fi

if ! run_with_optional_sudo chmod -R "$UPLOAD_MODE" "$UPLOAD_DIR"; then
  echo "[WARN] failed to chmod $UPLOAD_DIR, you may still hit permission issues."
fi

echo
echo "Prepared successfully:"
if ! ls -ld "$UPLOAD_DIR"; then
  true
fi
