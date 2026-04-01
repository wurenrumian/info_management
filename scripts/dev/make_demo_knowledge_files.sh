#!/usr/bin/env bash

# 个人集成测试使用，如需本地运行可能要更改一些参数
set -euo pipefail

OUT_DIR="${1:-/tmp}"
mkdir -p "$OUT_DIR"

DOCX_PATH="$OUT_DIR/knowledge_demo.docx"
XLSX_PATH="$OUT_DIR/knowledge_demo.xlsx"
PDF_PATH="$OUT_DIR/knowledge_demo.pdf"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SAMPLE_PDF="$ROOT_DIR/C++体系学习建议.pdf"

workdir="$(mktemp -d)"
trap 'rm -rf "$workdir"' EXIT

# ---------- DOCX ----------
mkdir -p "$workdir/docx/_rels" "$workdir/docx/word"

cat > "$workdir/docx/[Content_Types].xml" <<'XML'
<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>
XML

cat > "$workdir/docx/_rels/.rels" <<'XML'
<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>
XML

cat > "$workdir/docx/word/document.xml" <<'XML'
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>奖学金申请材料说明</w:t></w:r></w:p>
    <w:p><w:r><w:t>申请人需提交综测排名证明、成绩单和个人陈述。</w:t></w:r></w:p>
    <w:p><w:r><w:t>提交截止时间：2026年4月15日。</w:t></w:r></w:p>
  </w:body>
</w:document>
XML

(
  cd "$workdir/docx"
  zip -q -r "$DOCX_PATH" .
)

# ---------- XLSX ----------
mkdir -p "$workdir/xlsx/_rels" "$workdir/xlsx/xl/_rels" "$workdir/xlsx/xl/worksheets"

cat > "$workdir/xlsx/[Content_Types].xml" <<'XML'
<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
  <Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
  <Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/>
</Types>
XML

cat > "$workdir/xlsx/_rels/.rels" <<'XML'
<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>
XML

cat > "$workdir/xlsx/xl/workbook.xml" <<'XML'
<?xml version="1.0" encoding="UTF-8"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets>
    <sheet name="Sheet1" sheetId="1" r:id="rId1"/>
  </sheets>
</workbook>
XML

cat > "$workdir/xlsx/xl/_rels/workbook.xml.rels" <<'XML'
<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/>
</Relationships>
XML

cat > "$workdir/xlsx/xl/sharedStrings.xml" <<'XML'
<?xml version="1.0" encoding="UTF-8"?>
<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="6" uniqueCount="6">
  <si><t>字段</t></si>
  <si><t>内容</t></si>
  <si><t>申请材料</t></si>
  <si><t>综测排名证明</t></si>
  <si><t>成绩单</t></si>
  <si><t>个人陈述</t></si>
</sst>
XML

cat > "$workdir/xlsx/xl/worksheets/sheet1.xml" <<'XML'
<?xml version="1.0" encoding="UTF-8"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
    <row r="1">
      <c r="A1" t="s"><v>0</v></c>
      <c r="B1" t="s"><v>1</v></c>
    </row>
    <row r="2">
      <c r="A2" t="s"><v>2</v></c>
      <c r="B2" t="s"><v>3</v></c>
    </row>
    <row r="3">
      <c r="A3" t="s"><v>2</v></c>
      <c r="B3" t="s"><v>4</v></c>
    </row>
    <row r="4">
      <c r="A4" t="s"><v>2</v></c>
      <c r="B4" t="s"><v>5</v></c>
    </row>
  </sheetData>
</worksheet>
XML

(
  cd "$workdir/xlsx"
  zip -q -r "$XLSX_PATH" .
)

# ---------- PDF ----------
if [[ -f "$SAMPLE_PDF" ]]; then
  cp "$SAMPLE_PDF" "$PDF_PATH"
else
  echo "[WARN] sample PDF not found at $SAMPLE_PDF, skipping PDF generation"
fi

echo "Generated:"
echo "  $DOCX_PATH"
echo "  $XLSX_PATH"
echo "  $PDF_PATH"
