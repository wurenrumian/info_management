#!/usr/bin/env bash

# 个人集成测试使用，如需本地运行可能要更改一些参数

set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
JWT_SECRET="${JWT_SECRET:-dev-secret-change-in-production}"

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
PDF_FILE="${PDF_FILE:-/tmp/knowledge_demo.pdf}"

for cmd in curl openssl sed; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "$cmd command not found"
    exit 1
  fi
done

generate_token() {
  local user_id="$1"
  local role="$2"
  local class_id="$3"
  local grade="$4"

  local header
  header=$(echo -n '{"alg":"HS256","typ":"JWT"}' | openssl base64 | tr -d '\n' | tr '+/' '-_' | tr -d '=')
  local payload
  payload=$(echo -n "{\"sub\":$user_id,\"role\":$role,\"class_id\":$class_id,\"grade\":\"$grade\"}" | openssl base64 | tr -d '\n' | tr '+/' '-_' | tr -d '=')
  local signature
  signature=$(echo -n "$header.$payload" | openssl dgst -sha256 -hmac "$JWT_SECRET" -binary | openssl base64 | tr -d '\n' | tr '+/' '-_' | tr -d '=')
  echo "$header.$payload.$signature"
}

ADMIN_TOKEN=$(generate_token "$ADMIN_USER_ID" "$ADMIN_ROLE" "$ADMIN_CLASS_ID" "$ADMIN_GRADE")
STUDENT_TOKEN=$(generate_token "$STUDENT_USER_ID" "$STUDENT_ROLE" "$STUDENT_CLASS_ID" "$STUDENT_GRADE")

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local label="$3"
  if [[ "$haystack" != *"$needle"* ]]; then
    echo "[FAIL] $label: expect response contains '$needle'"
    echo "response: $haystack"
    exit 1
  fi
  echo "[PASS] $label"
}

assert_contains_any() {
  local haystack="$1"
  local label="$2"
  shift 2
  local needle
  for needle in "$@"; do
    if [[ "$haystack" == *"$needle"* ]]; then
      echo "[PASS] $label"
      return
    fi
  done
  echo "[FAIL] $label: none of expected markers found"
  echo "response: $haystack"
  exit 1
}

if [[ ! -f "$DOCX_FILE" ]]; then
  echo "DOCX_FILE not found: $DOCX_FILE"
  exit 1
fi
if [[ ! -f "$XLSX_FILE" ]]; then
  echo "XLSX_FILE not found: $XLSX_FILE"
  exit 1
fi
if [[ ! -f "$PDF_FILE" ]]; then
  echo "[WARN] PDF_FILE not found: $PDF_FILE, skipping PDF tests"
  SKIP_PDF=1
else
  SKIP_PDF=0
fi

echo "== 1) Health Check =="
health_resp="$(curl -s "$BASE_URL/healthz")"
echo "$health_resp"
assert_contains "$health_resp" "ok" "health check"
echo

echo "== 2) Admin Import Knowledge (multipart) =="
import_resp="$(curl -s -X POST "$BASE_URL/api/v1/admin/knowledge/import" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -F "question=奖学金申请材料有哪些" \
  -F "answer=请按附件准备并提交" \
  -F "keywords=奖学金,申请,材料" \
  -F "files=@$DOCX_FILE" \
  -F "files=@$XLSX_FILE")"
echo "$import_resp"
assert_contains_any "$import_resp" "import knowledge" "\"question\":\"奖学金申请材料有哪些\"" "\"Question\":\"奖学金申请材料有哪些\""
import_id="$(sed -n 's/.*"id":[[:space:]]*\([0-9][0-9]*\).*/\1/p' <<<"$import_resp" | head -n1)"
if [[ -z "$import_id" ]]; then
  import_id="$(sed -n 's/.*"ID":[[:space:]]*\([0-9][0-9]*\).*/\1/p' <<<"$import_resp" | head -n1)"
fi
if [[ -z "$import_id" ]]; then
  echo "[FAIL] import knowledge: failed to parse id"
  exit 1
fi
echo "Imported knowledge id: $import_id"
echo

if [[ "$SKIP_PDF" -eq 0 ]]; then
  echo "== 2b) Admin Import PDF Knowledge =="
  pdf_import_resp="$(curl -s -X POST "$BASE_URL/api/v1/admin/knowledge/import" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -F "question=C++学习路线指南" \
    -F "answer=请参考附件PDF文档" \
    -F "keywords=C++,后端,学习路线" \
    -F "files=@$PDF_FILE")"
  echo "$pdf_import_resp"
  assert_contains "$pdf_import_resp" "C++学习路线指南" "pdf import knowledge"
  pdf_import_id="$(sed -n 's/.*"id":[[:space:]]*\([0-9][0-9]*\).*/\1/p' <<<"$pdf_import_resp" | head -n1)"
  if [[ -z "$pdf_import_id" ]]; then
    pdf_import_id="$(sed -n 's/.*"ID":[[:space:]]*\([0-9][0-9]*\).*/\1/p' <<<"$pdf_import_resp" | head -n1)"
  fi
  echo "Imported PDF knowledge id: $pdf_import_id"
  echo
fi

echo "== 3) Student Search by Normal Keywords =="
search_keyword_resp="$(curl -s "$BASE_URL/api/v1/knowledge/search?q=奖学金申请" \
  -H "Authorization: Bearer $STUDENT_TOKEN")"
echo "$search_keyword_resp"
assert_contains "$search_keyword_resp" "\"total\":" "student search has total"
assert_contains "$search_keyword_resp" "奖学金申请材料有哪些" "student search keyword hit"
echo

echo "== 4) Student Search by Doc Content Keywords =="
search_doc_resp="$(curl -s "$BASE_URL/api/v1/knowledge/search?q=综测排名证明" \
  -H "Authorization: Bearer $STUDENT_TOKEN")"
echo "$search_doc_resp"
assert_contains "$search_doc_resp" "\"total\":" "doc search has total"
assert_contains "$search_doc_resp" "奖学金申请材料有哪些" "student search doc content hit"
echo

if [[ "$SKIP_PDF" -eq 0 ]]; then
  echo "== 4b) Student Search by PDF Content Keywords =="
  search_pdf_resp="$(curl -s "$BASE_URL/api/v1/knowledge/search?q=C++技术栈" \
    -H "Authorization: Bearer $STUDENT_TOKEN")"
  echo "$search_pdf_resp"
  assert_contains "$search_pdf_resp" "\"total\":" "pdf search has total"
  assert_contains "$search_pdf_resp" "C++学习路线指南" "student search pdf content hit"
  echo
fi

echo "== 5) Admin List Knowledge =="
admin_list_resp="$(curl -s "$BASE_URL/api/v1/admin/knowledge?query=奖学金&limit=20&offset=0" \
  -H "Authorization: Bearer $ADMIN_TOKEN")"
echo "$admin_list_resp"
assert_contains "$admin_list_resp" "\"total\":" "admin list has total"
assert_contains "$admin_list_resp" "奖学金申请材料有哪些" "admin list query hit"
echo

echo "== 6) Admin Get Knowledge By ID =="
admin_get_resp="$(curl -s "$BASE_URL/api/v1/admin/knowledge/$import_id" \
  -H "Authorization: Bearer $ADMIN_TOKEN")"
echo "$admin_get_resp"
assert_contains_any "$admin_get_resp" "admin get by id" "\"id\":$import_id" "\"ID\":$import_id"
echo

echo "== 7) Admin Patch Knowledge =="
admin_patch_resp="$(curl -s -X PATCH "$BASE_URL/api/v1/admin/knowledge/$import_id" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"answer":"请按最新附件材料提交"}')"
echo "$admin_patch_resp"
assert_contains "$admin_patch_resp" "\"updated\":true" "admin patch success"
echo

echo "== 8) Admin Patch Non-Existing Knowledge =="
admin_patch_404_resp="$(curl -s -X PATCH "$BASE_URL/api/v1/admin/knowledge/99999999" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"answer":"not found"}')"
echo "$admin_patch_404_resp"
assert_contains "$admin_patch_404_resp" "knowledge not found" "admin patch 404"
echo

echo "== 9) Admin Delete Knowledge =="
admin_delete_resp="$(curl -s -X DELETE "$BASE_URL/api/v1/admin/knowledge/$import_id" \
  -H "Authorization: Bearer $ADMIN_TOKEN")"
echo "$admin_delete_resp"
assert_contains "$admin_delete_resp" "\"deleted\":true" "admin delete success"
echo

echo "== 10) Admin Get Deleted Knowledge =="
admin_get_deleted_resp="$(curl -s "$BASE_URL/api/v1/admin/knowledge/$import_id" \
  -H "Authorization: Bearer $ADMIN_TOKEN")"
echo "$admin_get_deleted_resp"
assert_contains "$admin_get_deleted_resp" "knowledge not found" "admin get deleted item 404"
echo

if [[ "$SKIP_PDF" -eq 0 && -n "${pdf_import_id:-}" ]]; then
  echo "== 10b) Admin Delete PDF Knowledge =="
  admin_delete_pdf_resp="$(curl -s -X DELETE "$BASE_URL/api/v1/admin/knowledge/$pdf_import_id" \
    -H "Authorization: Bearer $ADMIN_TOKEN")"
  echo "$admin_delete_pdf_resp"
  assert_contains "$admin_delete_pdf_resp" "\"deleted\":true" "admin delete pdf success"
  echo
fi

echo "Done."
