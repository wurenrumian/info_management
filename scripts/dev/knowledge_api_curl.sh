#!/usr/bin/env bash

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

extract_id() {
  local body="$1"
  local id
  id="$(sed -n 's/.*"id":[[:space:]]*\([0-9][0-9]*\).*/\1/p' <<<"$body" | head -n1)"
  if [[ -z "$id" ]]; then
    id="$(sed -n 's/.*"ID":[[:space:]]*\([0-9][0-9]*\).*/\1/p' <<<"$body" | head -n1)"
  fi
  echo "$id"
}

upload_file() {
  local file_path="$1"
  local label="$2"

  local resp
  resp="$(curl -s -X POST "$BASE_URL/api/v1/files/upload" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -F "file=@$file_path")"
  echo "$resp"
  local file_id
  file_id="$(extract_id "$resp")"
  if [[ -z "$file_id" ]]; then
    echo "[FAIL] $label: failed to parse file id"
    exit 1
  fi
  echo "$file_id"
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

ADMIN_TOKEN=$(generate_token "$ADMIN_USER_ID" "$ADMIN_ROLE" "$ADMIN_CLASS_ID" "$ADMIN_GRADE")
STUDENT_TOKEN=$(generate_token "$STUDENT_USER_ID" "$STUDENT_ROLE" "$STUDENT_CLASS_ID" "$STUDENT_GRADE")

echo "== 1) Health Check =="
health_resp="$(curl -s "$BASE_URL/healthz")"
echo "$health_resp"
assert_contains "$health_resp" "ok" "health check"
echo

echo "== 2) Admin Create Knowledge =="
create_resp="$(curl -s -X POST "$BASE_URL/api/v1/admin/knowledge" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"question":"奖学金申请材料有哪些","answer":"请按附件准备并提交","keywords":["奖学金","申请","材料"]}')"
echo "$create_resp"
assert_contains "$create_resp" "奖学金申请材料有哪些" "create knowledge"
knowledge_id="$(extract_id "$create_resp")"
if [[ -z "$knowledge_id" ]]; then
  echo "[FAIL] create knowledge: failed to parse id"
  exit 1
fi
echo "Created knowledge id: $knowledge_id"
echo

echo "== 3) Upload DOCX/XLSX Files =="
docx_file_id="$(upload_file "$DOCX_FILE" "upload docx" | tail -n1)"
echo "DOCX file id: $docx_file_id"
xlsx_file_id="$(upload_file "$XLSX_FILE" "upload xlsx" | tail -n1)"
echo "XLSX file id: $xlsx_file_id"
echo

echo "== 4) Bind DOCX/XLSX To Knowledge =="
bind_resp="$(curl -s -X POST "$BASE_URL/api/v1/admin/knowledge/$knowledge_id/attachments" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"file_ids\":[${docx_file_id},${xlsx_file_id}]}")"
echo "$bind_resp"
assert_contains "$bind_resp" '"added_count":2' "bind attachments"
echo

echo "== 4a) Search Files By DOCX Content =="
files_search_docx_resp="$(curl -s "$BASE_URL/api/v1/files/search?q=综测排名证明" \
  -H "Authorization: Bearer $STUDENT_TOKEN")"
echo "$files_search_docx_resp"
assert_contains "$files_search_docx_resp" '"total":' "files search has total"
assert_contains "$files_search_docx_resp" "综测排名证明" "files search docx snippet hit"
assert_contains "$files_search_docx_resp" "\"id\":$docx_file_id" "files search includes docx file id"
assert_contains "$files_search_docx_resp" '"/uploads/documents/' "files search includes file url"
echo

if [[ "$SKIP_PDF" -eq 0 ]]; then
  echo "== 4b) Upload+Bind PDF To Knowledge =="
  pdf_file_id="$(upload_file "$PDF_FILE" "upload pdf" | tail -n1)"
  echo "PDF file id: $pdf_file_id"
  bind_pdf_resp="$(curl -s -X POST "$BASE_URL/api/v1/admin/knowledge/$knowledge_id/attachments" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"file_ids\":[${pdf_file_id}]}")"
  echo "$bind_pdf_resp"
  assert_contains "$bind_pdf_resp" '"added_count":1' "bind pdf attachment"
  echo

  echo "== 4c) Search Files By PDF Content =="
  files_search_pdf_resp="$(curl -s "$BASE_URL/api/v1/files/search?q=C++技术栈" \
    -H "Authorization: Bearer $STUDENT_TOKEN")"
  echo "$files_search_pdf_resp"
  assert_contains "$files_search_pdf_resp" '"total":' "files search pdf has total"
  assert_contains "$files_search_pdf_resp" "C++技术栈" "files search pdf snippet hit"
  assert_contains "$files_search_pdf_resp" "\"id\":$pdf_file_id" "files search includes pdf file id"
  echo
fi

echo "== 5) Student Search by Normal Keywords =="
search_keyword_resp="$(curl -s "$BASE_URL/api/v1/knowledge/search?q=奖学金申请" \
  -H "Authorization: Bearer $STUDENT_TOKEN")"
echo "$search_keyword_resp"
assert_contains "$search_keyword_resp" '"total":' "student search has total"
assert_contains "$search_keyword_resp" "奖学金申请材料有哪些" "student search keyword hit"
echo

echo "== 6) Student Search by Doc Content Keywords =="
search_doc_resp="$(curl -s "$BASE_URL/api/v1/knowledge/search?q=综测排名证明" \
  -H "Authorization: Bearer $STUDENT_TOKEN")"
echo "$search_doc_resp"
assert_contains "$search_doc_resp" '"total":' "doc search has total"
assert_contains "$search_doc_resp" "奖学金申请材料有哪些" "student search doc content hit"
echo

if [[ "$SKIP_PDF" -eq 0 ]]; then
  echo "== 6b) Student Search by PDF Content Keywords =="
  search_pdf_resp="$(curl -s "$BASE_URL/api/v1/knowledge/search?q=C++技术栈" \
    -H "Authorization: Bearer $STUDENT_TOKEN")"
  echo "$search_pdf_resp"
  assert_contains "$search_pdf_resp" '"total":' "pdf search has total"
  assert_contains "$search_pdf_resp" "奖学金申请材料有哪些" "student search pdf content hit"
  echo
fi

echo "== 7) Admin List/Detail =="
admin_list_resp="$(curl -s "$BASE_URL/api/v1/admin/knowledge?query=奖学金&limit=20&offset=0" \
  -H "Authorization: Bearer $ADMIN_TOKEN")"
echo "$admin_list_resp"
assert_contains "$admin_list_resp" '"total":' "admin list has total"

admin_get_resp="$(curl -s "$BASE_URL/api/v1/admin/knowledge/$knowledge_id" \
  -H "Authorization: Bearer $ADMIN_TOKEN")"
echo "$admin_get_resp"
assert_contains "$admin_get_resp" "\"id\":$knowledge_id" "admin get by id"
echo

echo "== 8) Admin List Attachments =="
list_attach_resp="$(curl -s "$BASE_URL/api/v1/admin/knowledge/$knowledge_id/attachments" \
  -H "Authorization: Bearer $ADMIN_TOKEN")"
echo "$list_attach_resp"
assert_contains "$list_attach_resp" "\"file_id\":$docx_file_id" "list has docx"
assert_contains "$list_attach_resp" "\"file_id\":$xlsx_file_id" "list has xlsx"
echo

echo "== 9) Admin Patch Knowledge =="
admin_patch_resp="$(curl -s -X PATCH "$BASE_URL/api/v1/admin/knowledge/$knowledge_id" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"answer":"请按最新附件材料提交"}')"
echo "$admin_patch_resp"
assert_contains "$admin_patch_resp" '"updated":true' "admin patch success"
echo

echo "== 10) Admin Detach One Attachment =="
detach_resp="$(curl -s -X DELETE "$BASE_URL/api/v1/admin/knowledge/$knowledge_id/attachments/$docx_file_id" \
  -H "Authorization: Bearer $ADMIN_TOKEN")"
echo "$detach_resp"
assert_contains "$detach_resp" '"deleted":true' "admin detach success"
echo

echo "== 11) Admin Delete Knowledge =="
admin_delete_resp="$(curl -s -X DELETE "$BASE_URL/api/v1/admin/knowledge/$knowledge_id" \
  -H "Authorization: Bearer $ADMIN_TOKEN")"
echo "$admin_delete_resp"
assert_contains "$admin_delete_resp" '"deleted":true' "admin delete success"
echo

echo "Done."
