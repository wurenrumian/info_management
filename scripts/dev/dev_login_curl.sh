#!/usr/bin/env bash

set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
STUDENT_ID="${STUDENT_ID:-2020001}"
ROLE="${ROLE:-1}"

resp_and_code="$(
  curl -sS -w $'\n%{http_code}' -X POST "$BASE_URL/api/v1/dev/register-or-login" \
    -H "Content-Type: application/json" \
    -d "{\"student_id\":\"$STUDENT_ID\",\"role\":$ROLE}"
)"

resp="${resp_and_code%$'\n'*}"
status="${resp_and_code##*$'\n'}"

echo "$resp"

if [[ "$status" != "200" ]]; then
  echo
  echo "HTTP $status"
  exit 1
fi

token="$(sed -n 's/.*"token":[[:space:]]*"\([^"]*\)".*/\1/p' <<<"$resp" | head -n1)"
if [[ -n "$token" ]]; then
  echo
  echo "TOKEN=$token"
fi
