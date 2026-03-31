#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

export GOCACHE="${GOCACHE:-/tmp/go-build}"
export GOMODCACHE="${GOMODCACHE:-/tmp/go-mod}"

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
