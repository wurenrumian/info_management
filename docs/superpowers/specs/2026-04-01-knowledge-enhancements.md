# Knowledge 模块功能增强设计（PDF提取 + 测试补齐 + 批量导入）

**日期**：2026-04-01  
**模块**：Phase 2-B Knowledge Base  
**变更类型**：功能增强（P0/P1 优先级补齐）

---

## 1. 目标

扩展 `knowledge` 模块的文档导入能力，支持 **PDF 文件的正文提取与索引**，使 PDF 附件中的内容可通过 `content_text` 字段参与全文检索。

---

## 2. 现状与问题

### 现状
当前 `extractor.go` 仅支持：
- ✅ `.docx`：通过 `word/document.xml` 提取
- ✅ `.xlsx`：通过 `xl/worksheets/*.xml` + `sharedStrings.xml` 提取  
- ❌ `.pdf`：返回空字符串，仅保存附件链接

### 问题
- 用户上传的PDF政策文件、通知等无法被搜索
- 知识库检索覆盖率不足，影响用户体验

---

## 3. 技术选型

| 选项 | 协议 | 理由 |
|------|------|------|
| **pdftotext** (poppler-utils) | GPL | ✅ 提取效果最好，中文支持优秀<br>✅ 公文/政策文件提取准确率高<br>✅ 通过 Go exec.Command 包装，无 C 依赖<br>⚠️ 需系统安装 poppler-utils |
| pdfcpu | MIT | ⚠️ 纯Go但对复杂公文PDF提取效果一般 |
| unidoc/unipdf | 商业 | ❌ 需要许可证 |
| rsc.io/pdf | BSD | ⚠️ 功能极简，无中文优化 |

**决策**：采用 `pdftotext` CLI（poppler-utils），通过 Go `exec.Command` 调用

---

## 4. 实现方案

### 4.1 依赖安装

系统依赖（部署时确保安装）：
```bash
# Debian/Ubuntu
apt-get install -y poppler-utils
# RHEL/CentOS
yum install -y poppler-utils
# Alpine
apk add poppler-utils
```

Go 代码无需额外依赖，使用标准库 `os/exec`。

### 4.2 代码变更

**文件**：`internal/service/knowledge/extractor.go`

```go
import (
    // ... 现有导入
    "os/exec"
)

func ExtractTextFromFile(path string) string {
    ext := strings.ToLower(filepath.Ext(path))
    switch ext {
    case ".docx":
        // 现有逻辑
    case ".xlsx":
        // 现有逻辑
    case ".pdf":
        text, err := extractPDF(path)
        if err != nil {
            // 记录日志但不阻塞导入
            return ""
        }
        return text
    default:
        return ""
    }
}

func extractPDF(path string) (string, error) {
    // 使用 pdftotext CLI 提取
    cmd := exec.Command("pdftotext", "-layout", "-q", path, "-")
    out, err := cmd.Output()
    if err != nil {
        return "", err
    }
    
    result := normalizeText(string(out))
    if result == "" {
        return "", nil  // 无文本内容（可能是扫描件）
    }
    return result, nil
}
```

### 4.3 注意事项

- **性能**：`pdftotext` 是 C++ 实现，提取速度快，适合公文类 PDF
- **扫描件**：无文本层的扫描PDF将返回空，不影响附件保存
- **密码保护**：忽略加密PDF（返回空），不中断流程
- **部署**：需确保运行环境安装 `poppler-utils`（Dockerfile 中已包含）

---

## 5. 测试方案

### 5.1 单元测试

新增 `extractor_pdf_test.go`：

```go
func TestExtractPDF_TextExtraction(t *testing.T) {
    // 使用测试PDF（含中英文混合）
    path := "testdata/sample.pdf"
    text := extractPDF(path)
    require.Contains(t, text, "期望关键词")
}

func TestExtractPDF_EncryptedReturnsEmpty(t *testing.T) {
    path := "testdata/encrypted.pdf"
    text := extractPDF(path)
    require.Empty(t, text)
}
```

### 5.2 集成测试

扩展现有 `knowledge_handler_test.go`：

```go
func TestKnowledgeSearchHitsImportedPDFContent(t *testing.T) {
    // 准备一个含文本的PDF（如政策文件）
    pdfData := generateTestPDF("这是一份测试PDF政策文件")
    // 上传并导入
    // 搜索"政策文件"应命中
}
```

### 5.3 API 脚本测试

更新 `scripts/dev/knowledge_api_curl.sh`：
- 增加 PDF 导入用例
- 验证 PDF 抽取的 `content_text` 能通过全文检索命中
- 脚本支持 PDF 文件缺失时跳过 PDF 测试（不中断整体流程）

---

## 6. API 文档更新

**文件**：`docs/api/phase2-knowledge-api.md` 第184行后追加：

```markdown
当前支持文件类型：
- `pdf`（已支持正文抽取）
- `docx`
- `xlsx`
- `doc` / `xls`（仅附件，不保证正文抽取）
```

并在第192行后补充：
```markdown
PDF 正文抽取说明：
- 需要 PDF 包含可复制文本层（非扫描件）
- 逐页提取，合并空格；无文本则不影响附件保存
```

---

## 7. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| pdftotext 未安装 | 运行时错误 | 启动时检测，Dockerfile 预装 |
| 大PDF内存峰值 | OOM | 限制上传大小（已有） |
| 扫描PDF无文本 | 功能未生效 | 文档明确说明，不阻塞导入 |
| 中文编码问题 | 乱码 | pdftotext 原生支持 UTF-8，中文公文提取效果好 |

---

## 8. 验收标准

1. 📄 文档：`docs/api/phase2-knowledge-api.md` 已更新类型列表
2. ✅ 测试：`extractor_pdf_test.go` 覆盖正常/异常场景
3. ✅ 集成：知识库导入PDF后，`content_text` 非空，搜索命中
4. ✅ 向后兼容：原有docx/xlsx功能不受影响
5. 📜 脚本：`knowledge_api_curl.sh` 包含 PDF 导入和搜索用例

---

## 9. 相关文件

- `internal/service/knowledge/extractor.go`（主修改）
- `internal/service/knowledge/extractor_pdf_test.go`（新建）
- `docs/api/phase2-knowledge-api.md`（更新）
- `scripts/dev/make_demo_knowledge_files.sh`（更新：生成PDF）
- `scripts/dev/knowledge_api_curl.sh`（更新：PDF导入/搜索用例）

---

## 10. P1 补充：Service 与 Extractor 独立单元测试

### 10.1 现状问题

当前 `service/knowledge/service.go` 的校验逻辑（Create/Patch 参数检查）和 `extractor.go` 的 DOCX/XLSX 提取逻辑仅在 handler 集成测试中间接覆盖，缺少独立单元测试。

### 10.2 新增测试文件

**`internal/service/knowledge/service_test.go`**：

```go
func TestServiceCreateRejectsEmptyQuestion(t *testing.T) {
    db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    svc := knowledge.NewService(db)
    item := &model.KnowledgeItem{Question: "", Answer: "answer"}
    err := svc.Create(item)
    require.Error(t, err)
    require.Equal(t, "missing fields", err.Error())
}

func TestServiceCreateRejectsEmptyAnswer(t *testing.T) {
    db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    svc := knowledge.NewService(db)
    item := &model.KnowledgeItem{Question: "q", Answer: ""}
    err := svc.Create(item)
    require.Error(t, err)
    require.Equal(t, "missing fields", err.Error())
}

func TestServicePatchRejectsEmptyUpdates(t *testing.T) {
    db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    svc := knowledge.NewService(db)
    err := svc.Patch(1, map[string]any{})
    require.Error(t, err)
    require.Equal(t, "empty patch", err.Error())
}

func TestServicePatchRejectsZeroID(t *testing.T) {
    db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    svc := knowledge.NewService(db)
    err := svc.Patch(0, map[string]any{"question": "x"})
    require.Error(t, err)
    require.Equal(t, "invalid id", err.Error())
}

func TestServiceNormalizePage(t *testing.T) {
    // limit=0 → defaultLimit
    l, o := normalizePage(0, 0)
    require.Equal(t, 20, l)
    require.Equal(t, 0, o)

    // limit>maxLimit → maxLimit
    l, o = normalizePage(200, 10)
    require.Equal(t, 100, l)
    require.Equal(t, 10, o)

    // offset<0 → 0
    l, o = normalizePage(10, -5)
    require.Equal(t, 10, l)
    require.Equal(t, 0, o)
}
```

**`internal/service/knowledge/extractor_test.go`**：

```go
func TestExtractDocx_BasicTextExtraction(t *testing.T) {
    // 构建最小 DOCX 并验证文本提取
    docxData := buildTestDocx("测试文档内容")
    path := filepath.Join(t.TempDir(), "test.docx")
    require.NoError(t, os.WriteFile(path, docxData, 0644))
    text := extractDocx(path)
    require.Contains(t, text, "测试文档内容")
}

func TestExtractXlsx_BasicTextExtraction(t *testing.T) {
    // 构建最小 XLSX 并验证文本提取
    xlsxData := buildTestXlsx("单元格内容")
    path := filepath.Join(t.TempDir(), "test.xlsx")
    require.NoError(t, os.WriteFile(path, xlsxData, 0644))
    text := extractXlsx(path)
    require.Contains(t, text, "单元格内容")
}

func TestExtractTextFromFile_UnsupportedType(t *testing.T) {
    result := ExtractTextFromFile("test.txt")
    require.Empty(t, result)
}

func TestExtractTextFromFile_NonExistentFile(t *testing.T) {
    result := ExtractTextFromFile("nonexistent.docx")
    require.Empty(t, result)
}

func TestNormalizeText_MultipleSpaces(t *testing.T) {
    require.Equal(t, "a b c", normalizeText("a   b  c"))
}

func TestNormalizeText_EmptyString(t *testing.T) {
    require.Empty(t, normalizeText(""))
    require.Empty(t, normalizeText("   "))
}
```

### 10.3 验收标准

- `go test ./internal/service/knowledge/... -count=1` 全通过
- 新增测试覆盖 service 校验逻辑 + extractor 核心提取逻辑

---

## 11. P1 补充：批量导入支持（Excel 多行 → 多条知识）

### 11.1 目标

支持上传一个 `.xlsx` 文件，每一行作为一条知识条目批量导入，降低管理员维护成本。

### 11.2 Excel 格式约定

| 列名 | 必填 | 说明 |
|------|------|------|
| `question` | 是 | 问题 |
| `answer` | 是 | 答案 |
| `keywords` | 否 | 逗号分隔关键词 |
| `attachment_url` | 否 | 已有附件 URL（不上传文件） |

### 11.3 新增 API

**`POST /api/v1/admin/knowledge/import-batch`**

- Content-Type: `multipart/form-data`
- 表单字段：`file`（单个 `.xlsx` 文件）

成功响应：

```json
{
  "data": {
    "imported": 42,
    "skipped": 3,
    "errors": [
      {"row": 5, "reason": "missing question"},
      {"row": 12, "reason": "missing answer"}
    ]
  }
}
```

### 11.4 实现方案

**新增文件**：`internal/service/knowledge/batch_import.go`

```go
type BatchImportResult struct {
    Imported int
    Skipped  int
    Errors   []BatchImportError
}

type BatchImportError struct {
    Row    int
    Reason string
}

func (s *Service) BatchImportFromXlsx(path string, actorID uint) (*BatchImportResult, error) {
    // 1. 解析 Excel
    // 2. 逐行校验 question/answer 非空
    // 3. 逐行 Create
    // 4. 记录汇总 admin_log
}
```

**handler 新增**：`internal/http/handler/admin_knowledge_handler.go` 追加 `ImportBatchKnowledge` 方法

**路由注册**：`router.go` 追加 `admin.POST("/knowledge/import-batch", ...)`

### 11.5 测试方案

**`internal/http/handler/knowledge_handler_test.go` 追加**：

```go
func TestAdminKnowledgeImportBatch(t *testing.T) {
    // 构建测试 Excel（含 5 行有效 + 2 行缺失字段）
    xlsxData := buildTestBatchXlsx()
    // 上传并验证 imported/skipped/errors 计数
}
```

**`internal/service/knowledge/batch_import_test.go`**：

```go
func TestBatchImport_AllValidRows(t *testing.T) {
    // 全部有效行 → imported=N, skipped=0, errors=[]
}

func TestBatchImport_SomeInvalidRows(t *testing.T) {
    // 部分缺失 question/answer → 校验 skipped 和 errors
}

func TestBatchImport_EmptyFile(t *testing.T) {
    // 空 Excel → imported=0, skipped=0
}
```

### 11.6 注意事项

- **事务**：批量导入应使用事务，单行失败不影响其他行（不整体回滚）
- **性能**：单次导入限制最大行数（建议 500 行），超限返回 400
- **审计**：汇总写入一条 `admin_logs`，action=`knowledge.import-batch`

### 11.7 验收标准

1. ✅ API：`POST /api/v1/admin/knowledge/import-batch` 可用
2. ✅ 测试：覆盖全有效/部分无效/空文件场景
3. ✅ 文档：`docs/api/phase2-knowledge-api.md` 更新批量导入说明
4. ✅ 限制：行数上限校验生效
5. ✅ 审计：批量导入操作写入 admin_logs
