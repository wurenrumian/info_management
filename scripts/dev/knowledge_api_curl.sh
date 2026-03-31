#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
ADMIN_USER_ID="${ADMIN_USER_ID:-200}"
ADMIN_ROLE="${ADMIN_ROLE:-2}"
ADMIN_CLASS_ID="${ADMIN_CLASS_ID:-1}"
ADMIN_GRADE="${ADMIN_GRADE:-2023}"

STUDENT_USER_ID="${STUDENT_USER_ID:-100}"
STUDENT_ROLE="${STUDENT_ROLE:-1}"
STUDENT_CLASS_ID="${STUDENT_CLASS_ID:-1}"
STUDENT_GRADE="${STUDENT_GRADE:-2023}"

DOCX_FILE="${DOCX_FILE:-/tmp/knowledge_demo.docx}"
XLSX_FILE="${XLSX_FILE:-/tmp/knowledge_demo.xlsx}"

if [[ ! -f "$DOCX_FILE" ]]; then
  echo "DOCX_FILE not found: $DOCX_FILE"
  exit 1
fi
if [[ ! -f "$XLSX_FILE" ]]; then
  echo "XLSX_FILE not found: $XLSX_FILE"
  exit 1
fi

echo "== 1) Health Check =="
curl -s "$BASE_URL/healthz"; echo

echo "== 2) Admin Import Knowledge (multipart) =="
curl -s -X POST "$BASE_URL/api/v1/admin/knowledge/import" \
  -H "X-User-Id: $ADMIN_USER_ID" \
  -H "X-User-Role: $ADMIN_ROLE" \
  -H "X-User-Class-Id: $ADMIN_CLASS_ID" \
  -H "X-User-Grade: $ADMIN_GRADE" \
  -F "question=奖学金申请材料有哪些" \
  -F "answer=请按附件准备并提交" \
  -F "keywords=奖学金,申请,材料" \
  -F "files=@$DOCX_FILE" \
  -F "files=@$XLSX_FILE"
echo

echo "== 3) Student Search by Normal Keywords =="
curl -s "$BASE_URL/api/v1/knowledge/search?q=奖学金申请" \
  -H "X-User-Id: $STUDENT_USER_ID" \
  -H "X-User-Role: $STUDENT_ROLE" \
  -H "X-User-Class-Id: $STUDENT_CLASS_ID" \
  -H "X-User-Grade: $STUDENT_GRADE"
echo

echo "== 4) Student Search by Doc Content Keywords =="
curl -s "$BASE_URL/api/v1/knowledge/search?q=综测排名证明" \
  -H "X-User-Id: $STUDENT_USER_ID" \
  -H "X-User-Role: $STUDENT_ROLE" \
  -H "X-User-Class-Id: $STUDENT_CLASS_ID" \
  -H "X-User-Grade: $STUDENT_GRADE"
echo

echo "== 5) Admin List Knowledge =="
curl -s "$BASE_URL/api/v1/admin/knowledge?query=奖学金&limit=20&offset=0" \
  -H "X-User-Id: $ADMIN_USER_ID" \
  -H "X-User-Role: $ADMIN_ROLE" \
  -H "X-User-Class-Id: $ADMIN_CLASS_ID" \
  -H "X-User-Grade: $ADMIN_GRADE"
echo

echo "Done."
