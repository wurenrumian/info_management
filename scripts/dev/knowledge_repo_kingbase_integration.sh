#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

# ==================== Go 缓存设置（关键修改部分）====================
# 使用持久化缓存路径，避免每次都重新下载模块
export GOCACHE="${GOCACHE:-${HOME}/.cache/go-build}"
export GOMODCACHE="${GOMODCACHE:-${HOME}/go/pkg/mod}"

# 可选：如果你想让 Go 使用系统默认路径（最推荐），可以直接注释掉下面两行
# export GOCACHE="${GOCACHE:-$(go env GOCACHE)}"
# export GOMODCACHE="${GOMODCACHE:-$(go env GOMODCACHE)}"

# ==================== Kingbase DSN 处理 ====================
if [[ -z "${KINGBASE_DSN:-}" && -n "${DATABASE_DSN:-}" ]]; then
  export KINGBASE_DSN="$DATABASE_DSN"
fi

if [[ -z "${KINGBASE_DSN:-}" ]]; then
  echo "KINGBASE_DSN is empty (DATABASE_DSN is also empty)."
  echo "Example:"
  echo "  export DATABASE_DSN='host=127.0.0.1 port=54321 user=system password=123456 dbname=test sslmode=disable'"
  echo "  # or export KINGBASE_DSN directly"
  echo "Then rerun: ./scripts/dev/knowledge_repo_kingbase_integration.sh"
  exit 1
fi

echo "==> Running Kingbase integration test for knowledge repo"
go test ./internal/repo -tags=integration -run TestKnowledgeRepoSearchWithKingbase -count=1
echo "==> Kingbase integration test passed"