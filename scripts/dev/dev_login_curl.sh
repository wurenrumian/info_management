#!/usr/bin/env bash

set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
STUDENT_ID="${STUDENT_ID:-2020001}"
ROLE="${ROLE:-1}"

resp="$(
  curl -sS -X POST "$BASE_URL/api/v1/dev/register-or-login" \
    -H "Content-Type: application/json" \
    -d "{\"student_id\":\"$STUDENT_ID\",\"role\":$ROLE}"
)"

echo "$resp"

token="$(sed -n 's/.*"token":"\([^"]*\)".*/\1/p' <<<"$resp" | head -n1)"
if [[ -n "$token" ]]; then
  echo
  echo "TOKEN=$token"
fi
