# Knowledge PDF 正文提取增强设计

**日期**：2026-04-01  
**模块**：Phase 2-B Knowledge Base  
**变更类型**：功能增强（扩展文档类型支持）

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
| **pdfcpu** | MIT | ✅ 纯Go实现，无C依赖<br>✅ 支持文本提取（`ExtractText`）<br>✅ 轻量（~2MB）<br>✅ 社区活跃 |
| unidoc/unipdf | 商业 | ❌ 需要许可证 |
| rsc.io/pdf | BSD | ⚠️ 功能极简，无中文优化 |
| pdftotext | GPL | ❌ 依赖系统poppler，部署复杂 |

**决策**：采用 `github.com/pdfcpu/pdfcpu/pkg/pdfcpu`

---

## 4. 实现方案

### 4.1 依赖添加

`go.mod` 新增：
```go
require github.com/pdfcpu/pdfcpu v0.9.1
```

### 4.2 代码变更

**文件**：`internal/service/knowledge/extractor.go`

```go
import (
    // ... 现有导入
    "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
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
    // pdfcpu 文本提取
    ctx, err := pdfcpu.ReadContextFile(path)
    if err != nil {
        return "", err
    }
    
    var buf strings.Builder
    pageCount := ctx.PageCount
    
    // 逐页提取
    for i := 1; i <= pageCount; i++ {
        txt, err := ctx.ExtractText(i)
        if err != nil {
            continue  // 单页失败不影响其他页
        }
        if strings.TrimSpace(txt) != "" {
            buf.WriteString(txt)
            buf.WriteByte(' ')
        }
    }
    
    result := normalizeText(buf.String())
    if result == "" {
        return "", nil  // 无文本内容（可能是扫描件）
    }
    return result, nil
}
```

### 4.3 注意事项

- **性能**：`ReadContextFile` 会解析整个PDF，但导入是低频操作，可接受
- **扫描件**：无文本层的扫描PDF将返回空，不影响附件保存
- **密码保护**：忽略加密PDF（返回空），不中断流程
- **内存**：逐页处理，避免大文件OOM

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

### 5.3 金仓集成测试

更新 `scripts/dev/knowledge_repo_kingbase_integration.sh`：
- 增加 PDF 导入用例
- 验证 PDF 抽取的 `content_text` 能通过 FTS 检索

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
| pdfcpu API 变更 | 编译失败 | 锁定版本 v0.9.1 |
| 大PDF内存峰值 | OOM | 逐页处理，限制上传大小（已有） |
| 扫描PDF无文本 | 功能未生效 | 文档明确说明，不阻塞导入 |
| 中文编码问题 | 乱码 | pdfcpu 支持 UTF-8，需测试验证 |

---

## 8. 验收标准

1. 📄 文档：`docs/api/phase2-knowledge-api.md` 已更新类型列表
2. ✅ 测试：`extractor_pdf_test.go` 覆盖正常/异常场景
3. ✅ 集成：知识库导入PDF后，`content_text` 非空，搜索命中
4. ✅ 向后兼容：原有docx/xlsx功能不受影响
5. 📜 脚本：金仓集成测试包含PDF用例

---

## 9. 相关文件

- `internal/service/knowledge/extractor.go`（主修改）
- `internal/service/knowledge/extractor_pdf_test.go`（新建）
- `docs/api/phase2-knowledge-api.md`（更新）
- `scripts/dev/knowledge_repo_kingbase_integration.sh`（更新）
