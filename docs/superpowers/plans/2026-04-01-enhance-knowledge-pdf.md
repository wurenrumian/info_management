# Knowledge PDF 提取增强实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 knowledge 模块扩展 PDF 正文提取能力，支持 PDF 附件内容参与全文检索。

**Architecture:** 在 `extractor.go` 新增 `extractPDF` 函数，使用 pdfcpu 逐页提取文本；不修改现有 handler/API 契约，仅增强 `content_text` 生成逻辑。

**Tech Stack:** Go, pdfcpu v0.9.1, GORM, Gin, SQLite (tests), Kingbase (integration)

---

## 文件结构说明

```
internal/service/knowledge/
├── extractor.go          # 修改：新增PDF提取分支
├── extractor_pdf_test.go # 新建：PDF单元测试
docs/api/
├── phase2-knowledge-api.md # 更新：文件类型列表
scripts/dev/
└── knowledge_repo_kingbase_integration.sh # 更新：增PDF用例
```

---

### Task 1: 依赖与测试数据准备

**Files:**
- Modify: `go.mod`
- Create: `internal/service/knowledge/testdata/sample.pdf`
- Create: `internal/service/knowledge/testdata/encrypted.pdf`

- [ ] **Step 1: 添加 pdfcpu 依赖**

```bash
go get github.com/pdfcpu/pdfcpu@v0.9.1
```

验证：`go mod tidy` 应无错误

- [ ] **Step 2: 生成测试用 PDF 文件**

```bash
mkdir -p internal/service/knowledge/testdata

# 生成含中英文的 sample.pdf（内容包含"这是一份测试PDF政策文件"）
# 可使用 LibreOffice 或在线工具转换为 PDF
# 文件名: internal/service/knowledge/testdata/sample.pdf

# 生成加密 PDF（密码: test）
# 用作异常场景测试
# 文件名: internal/service/knowledge/testdata/encrypted.pdf
```

**Verification:** 两个文件存在且可读

---

### Task 2: 实现 PDF 提取核心逻辑

**Files:**
- Modify: `internal/service/knowledge/extractor.go`

- [ ] **Step 1: 导入包**

在 extractor.go 顶部新增：
```go
import "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
```

- [ ] **Step 2: 添加 PDF case**

在 `ExtractTextFromFile` 的 switch 中增加：
```go
case ".pdf":
    text, err := extractPDF(path)
    if err != nil {
        return ""
    }
    return text
```

- [ ] **Step 3: 实现 extractPDF 函数**

在文件末尾添加：
```go
func extractPDF(path string) (string, error) {
    ctx, err := pdfcpu.ReadContextFile(path)
    if err != nil {
        return "", err
    }
    
    var buf strings.Builder
    pageCount := ctx.PageCount
    
    for i := 1; i <= pageCount; i++ {
        txt, err := ctx.ExtractText(i)
        if err != nil {
            continue  // 跳过失败页面
        }
        if strings.TrimSpace(txt) != "" {
            buf.WriteString(txt)
            buf.WriteByte(' ')
        }
    }
    
    result := normalizeText(buf.String())
    if result == "" {
        return "", nil
    }
    return result, nil
}
```

---

### Task 3: 单元测试

**Files:**
- Create: `internal/service/knowledge/extractor_pdf_test.go`

- [ ] **Step 1: 编写测试框架**

```go
package knowledge

import (
    "testing"
    "manage/internal/service/knowledge"
    "github.com/stretchr/testify/require"
)

func TestExtractPDF_TextExtraction(t *testing.T) {
    path := "internal/service/knowledge/testdata/sample.pdf"
    text := extractPDF(path)
    require.Contains(t, text, "这是一份测试PDF政策文件")
}

func TestExtractPDF_EncryptedReturnsEmpty(t *testing.T) {
    path := "internal/service/knowledge/testdata/encrypted.pdf"
    text := extractPDF(path)
    require.Empty(t, text)
}

func TestExtractPDF_NonExistentFile(t *testing.T) {
    text := extractPDF("nonexistent.pdf")
    require.Empty(t, text)
}
```

- [ ] **Step 2: 运行测试验证失败（RED）**

```bash
go test ./internal/service/knowledge -run TestExtractPDF -v
```
Expected: 编译失败（extractPDF 未定义）或测试失败

- [ ] **Step 3: 实现后运行到 GREEN**

```bash
go test ./internal/service/knowledge -run TestExtractPDF -v
```
Expected: 所有测试 PASS

---

### Task 4: 集成测试更新

**Files:**
- Modify: `internal/http/handler/knowledge_handler_test.go`

- [ ] **Step 1: 新增 PDF 导入与搜索测试**

在文件末尾追加（参考第280行 doctest 模式）：

```go
func TestKnowledgeSearchHitsImportedPDFContent(t *testing.T) {
    uploadDir := t.TempDir()
    t.Setenv("KNOWLEDGE_UPLOAD_DIR", uploadDir)

    db, r := setupKnowledgeTestRouter(t)

    // 生成测试PDF（使用已有的sample.pdf构造multipart）
    pdfPath := "internal/service/knowledge/testdata/sample.pdf"
    pdfBytes, err := os.ReadFile(pdfPath)
    require.NoError(t, err)

    var body bytes.Buffer
    writer := multipart.NewWriter(&body)
    require.NoError(t, writer.WriteField("question", "PDF政策"))
    require.NoError(t, writer.WriteField("answer", "请查看PDF附件"))
    part, err := writer.CreateFormFile("files", "policy.pdf")
    require.NoError(t, err)
    _, err = part.Write(pdfBytes)
    require.NoError(t, writer.Close())

    importReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/knowledge/import", &body)
    importReq.Header.Set("Content-Type", writer.FormDataContentType())
    importReq.Header.Set("X-User-Id", "200")
    importReq.Header.Set("X-User-Role", "2")
    importW := httptest.NewRecorder()
    r.ServeHTTP(importW, importReq)
    require.Equal(t, http.StatusOK, importW.Code)

    // 搜索 PDF 中的一个关键词
    searchReq := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge/search?q=测试PDF", nil)
    searchReq.Header.Set("X-User-Id", "100")
    searchReq.Header.Set("X-User-Role", "1")
    searchW := httptest.NewRecorder()
    r.ServeHTTP(searchW, searchReq)

    require.Equal(t, http.StatusOK, searchW.Code)
    require.Contains(t, searchW.Body.String(), "PDF政策")
}
```

- [ ] **Step 2: 运行完整 handler 测试**

```bash
go test ./internal/http/handler -v
```
Expected: 新增测试通过

---

### Task 5: API 文档与脚本更新

**Files:**
- Modify: `docs/api/phase2-knowledge-api.md`
- Modify: `scripts/dev/knowledge_repo_kingbase_integration.sh`

- [ ] **Step 1: 更新 API 文档支持类型列表**

查找第184行附近的"当前支持文件类型"，修改为：
```markdown
当前支持文件类型：
- `pdf`（已支持正文抽取）
- `docx`
- `xlsx`
- `doc` / `xls`（仅附件，不保证正文抽取）
```

在第192行后新增：
```markdown
PDF 正文抽取说明：
- 需要 PDF 包含可复制文本层（非扫描件）
- 逐页提取，合并空格；无文本则不影响附件保存
```

- [ ] **Step 2: 金仓集成测试脚本增加 PDF 用例**

编辑 `scripts/dev/knowledge_repo_kingbase_integration.sh`，在已有导入流程后追加：

```bash
# PDF import test
upload_dir=$(mktemp -d)
trap 'rm -rf "$upload_dir"' EXIT
cp testdata/sample.pdf "$upload_dir/"

# 调用 /api/v1/admin/knowledge/import 上传 sample.pdf
# 然后调用 /api/v1/knowledge/search?q=测试PDF 验证命中
# 失败则 exit 1
```

（具体脚本实现可复用现有doctest逻辑）

- [ ] **Step 3: 脚本可执行性验证**

```bash
chmod +x scripts/dev/knowledge_repo_kingbase_integration.sh
```

---

### Task 6: 全量测试与提交

- [ ] **Step 1: 运行所有测试并修复至全绿**

```bash
go test ./... -count=1
```
回填任何失败的测试，直到全部通过

- [ ] **Step 2: 格式化代码**

```bash
go fmt ./...
```

- [ ] **Step 3: 提交变更**

```bash
git add go.mod go.sum
git add internal/service/knowledge/extractor.go
git add internal/service/knowledge/extractor_pdf_test.go
git add internal/service/knowledge/testdata/
git add internal/http/handler/knowledge_handler_test.go
git add docs/api/phase2-knowledge-api.md
git add scripts/dev/knowledge_repo_kingbase_integration.sh
git commit -m "feat(knowledge): support PDF text extraction for search indexing"
```

---

## 完成核查

- [ ] PDF 导入后 `content_text` 非空
- [ ] 能用关键词搜索到 PDF 内容
- [ ] 单元测试覆盖正常/异常流程
- [ ] 文档与脚本同步更新
- [ ] 不破坏现有知识库功能（docx/xlsx）

---

