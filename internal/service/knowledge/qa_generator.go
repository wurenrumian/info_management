package knowledge

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"manage/internal/config"
)

var (
	ErrInvalidQACountRange = errors.New("invalid qa_count_range")
	ErrInvalidDraftJSON    = errors.New("invalid draft json")
	ErrInvalidDraftItem    = errors.New("invalid item")
	ErrGeneratePreview     = errors.New("generate preview failed")
)

type QACountRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type QADraft struct {
	Question          string   `json:"question"`
	Answer            string   `json:"answer"`
	Keywords          []string `json:"keywords"`
	AttachmentFileIDs []uint   `json:"attachment_file_ids"`
}

type QAPreviewResult struct {
	Items []QADraft `json:"items"`
}

type QADocumentInput struct {
	FileID      uint
	Title       string
	FilePath    string
	URL         string
	ContentText string
}

type QAGenerator interface {
	Generate(ctx context.Context, docs []QADocumentInput, countRange QACountRange) ([]QADraft, error)
}

func ValidateCountRange(r QACountRange) error {
	if r.Min < 1 || r.Max < r.Min || r.Max > 30 {
		return ErrInvalidQACountRange
	}
	return nil
}

func ParseDraftsJSON(raw string, countRange QACountRange, allowedFileIDs map[uint]struct{}) ([]QADraft, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, ErrInvalidDraftJSON
	}
	raw = extractFirstJSONObject(raw)
	if raw == "" {
		return nil, ErrInvalidDraftJSON
	}
	var wrapped QAPreviewResult
	if err := json.Unmarshal([]byte(raw), &wrapped); err != nil {
		return nil, ErrInvalidDraftJSON
	}
	if err := ValidateDrafts(wrapped.Items, countRange, allowedFileIDs); err != nil {
		return nil, err
	}
	return wrapped.Items, nil
}

func ValidateDrafts(items []QADraft, countRange QACountRange, allowedFileIDs map[uint]struct{}) error {
	if err := ValidateCountRange(countRange); err != nil {
		return err
	}
	if len(items) < countRange.Min || len(items) > countRange.Max {
		return ErrInvalidDraftItem
	}
	for _, item := range items {
		if strings.TrimSpace(item.Question) == "" || strings.TrimSpace(item.Answer) == "" {
			return ErrInvalidDraftItem
		}
		if item.Keywords != nil && len(item.Keywords) == 0 {
			return ErrInvalidDraftItem
		}
		for _, fileID := range item.AttachmentFileIDs {
			if fileID == 0 {
				return ErrInvalidDraftItem
			}
			if allowedFileIDs != nil {
				if _, ok := allowedFileIDs[fileID]; !ok {
					return ErrInvalidDraftItem
				}
			}
		}
	}
	return nil
}

func NewQAGeneratorFromConfig() QAGenerator {
	primary := NewOpenAICompatGenerator(config.AIProvider(), config.AIBaseURL(), config.AIAPIKey(), config.AIModel())
	return &resilientGenerator{
		primary:  primary,
		fallback: NewHeuristicQAGenerator(),
	}
}

type resilientGenerator struct {
	primary  QAGenerator
	fallback QAGenerator
}

func (g *resilientGenerator) Generate(ctx context.Context, docs []QADocumentInput, countRange QACountRange) ([]QADraft, error) {
	log.Printf("[knowledge-ai] generate start docs=%d range=%d-%d", len(docs), countRange.Min, countRange.Max)
	items, err := g.primary.Generate(ctx, docs, countRange)
	if err == nil {
		log.Printf("[knowledge-ai] primary generate success items=%d", len(items))
		return items, nil
	}
	log.Printf("[knowledge-ai] primary generate failed err=%v; fallback=heuristic", err)
	start := time.Now()
	fallbackItems, fallbackErr := g.fallback.Generate(ctx, docs, countRange)
	if fallbackErr != nil {
		log.Printf("[knowledge-ai] fallback generate failed err=%v", fallbackErr)
		return nil, fallbackErr
	}
	log.Printf("[knowledge-ai] fallback generate success items=%d cost_ms=%d", len(fallbackItems), time.Since(start).Milliseconds())
	return fallbackItems, nil
}

type HeuristicQAGenerator struct{}

func NewHeuristicQAGenerator() *HeuristicQAGenerator {
	return &HeuristicQAGenerator{}
}

func (g *HeuristicQAGenerator) Generate(_ context.Context, docs []QADocumentInput, countRange QACountRange) ([]QADraft, error) {
	if err := ValidateCountRange(countRange); err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, ErrGeneratePreview
	}
	items := make([]QADraft, 0, countRange.Min)
	for i := 0; i < countRange.Min; i++ {
		doc := docs[i%len(docs)]
		title := strings.TrimSuffix(strings.TrimSpace(doc.Title), filepath.Ext(doc.Title))
		if title == "" {
			title = "该文档"
		}
		snippet := firstSnippet(doc.ContentText)
		if snippet == "" {
			snippet = "请查看附件原文。"
		}
		items = append(items, QADraft{
			Question:          fmt.Sprintf("%s 主要内容是什么？", title),
			Answer:            snippet,
			Keywords:          extractKeywords(doc.ContentText, title),
			AttachmentFileIDs: []uint{doc.FileID},
		})
	}
	allowed := make(map[uint]struct{}, len(docs))
	for _, doc := range docs {
		allowed[doc.FileID] = struct{}{}
	}
	if err := ValidateDrafts(items, countRange, allowed); err != nil {
		return nil, err
	}
	return items, nil
}

type OpenAICompatGenerator struct {
	provider string
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

func NewOpenAICompatGenerator(provider, baseURL, apiKey, model string) *OpenAICompatGenerator {
	if model == "" {
		model = "openai/gpt-5.2"
	}
	provider = "openrouter"
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api"
	}
	return &OpenAICompatGenerator{
		provider: provider,
		baseURL:  baseURL,
		apiKey:   strings.TrimSpace(apiKey),
		model:    strings.TrimSpace(model),
		client:  &http.Client{},
	}
}

func (g *OpenAICompatGenerator) Provider() string {
	return g.provider
}

func (g *OpenAICompatGenerator) BaseURL() string {
	return g.baseURL
}

func (g *OpenAICompatGenerator) Model() string {
	return g.model
}

func (g *OpenAICompatGenerator) Generate(ctx context.Context, docs []QADocumentInput, countRange QACountRange) ([]QADraft, error) {
	if err := ValidateCountRange(countRange); err != nil {
		return nil, err
	}
	if strings.TrimSpace(g.baseURL) == "" || strings.TrimSpace(g.apiKey) == "" {
		log.Printf("[knowledge-ai] provider=%s skipped: missing base_url or api_key", g.provider)
		return nil, ErrGeneratePreview
	}
	if len(docs) == 0 {
		log.Printf("[knowledge-ai] provider=%s skipped: empty docs input", g.provider)
		return nil, ErrGeneratePreview
	}
	allowed := make(map[uint]struct{}, len(docs))
	for _, doc := range docs {
		allowed[doc.FileID] = struct{}{}
	}
	log.Printf("[knowledge-ai] provider=%s model=%s request docs=%d range=%d-%d strategy=all-in-one", g.provider, g.model, len(docs), countRange.Min, countRange.Max)

	payload := map[string]any{
		"model": g.model,
		"messages": []map[string]string{
			{"role": "system", "content": "你是高校管理系统知识库问答生成助手，固定使用 friendly 风格。请先像真实学生提问、像有经验且耐心的老师或辅导员回答，再提取关键词。问题要口语化、具体、可检索；回答要亲和、直接、完整、可执行，不说空话。回答中如果有步骤/条件，必须使用自然换行（\\n）分段，不要挤成一行。关键词必须是可检索主题词，严禁模板词、占位符、字段名、文件名。你必须只输出 JSON，格式严格为 {\"items\":[...]}，不要输出任何解释文本。"},
			{"role": "user", "content": buildUserPrompt(docs, countRange)},
		},
		"stream":          true,
		"effort":          "none",
		"temperature":     0.2,
	}
	b, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+"/v1/chat/completions", bytes.NewReader(b))
	if err != nil {
		return nil, ErrGeneratePreview
	}
	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://localhost")
	req.Header.Set("X-Title", "info_management")

	resp, err := g.client.Do(req)
	if err != nil {
		log.Printf("[knowledge-ai] provider=%s http request failed err=%v", g.provider, err)
		return nil, ErrGeneratePreview
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		log.Printf("[knowledge-ai] provider=%s non-2xx status=%d body=%s", g.provider, resp.StatusCode, truncateForLog(string(raw), 600))
		return nil, ErrGeneratePreview
	}

	content, err := readModelContent(resp.Body)
	if err != nil {
		log.Printf("[knowledge-ai] provider=%s read model content failed err=%v", g.provider, err)
		return nil, ErrGeneratePreview
	}
	if strings.TrimSpace(content) == "" {
		log.Printf("[knowledge-ai] provider=%s empty choices/content", g.provider)
		return nil, ErrGeneratePreview
	}
	items, err := ParseDraftsJSON(content, countRange, allowed)
	if err != nil {
		log.Printf("[knowledge-ai] provider=%s parse drafts failed err=%v content=%s", g.provider, err, truncateForLog(content, 600))
		return nil, ErrGeneratePreview
	}
	log.Printf("[knowledge-ai] provider=%s success items=%d", g.provider, len(items))
	return items, nil
}

func readModelContent(body io.Reader) (string, error) {
	b, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	raw := strings.TrimSpace(string(b))
	if raw == "" {
		return "", nil
	}
	if strings.Contains(raw, "data:") {
		streamContent, err := readSSEContent(strings.NewReader(raw))
		if err == nil && strings.TrimSpace(streamContent) != "" {
			return streamContent, nil
		}
	}

	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", nil
	}
	return out.Choices[0].Message.Content, nil
}

func readSSEContent(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var payload strings.Builder
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		if data == "[DONE]" {
			break
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content json.RawMessage `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		for _, choice := range chunk.Choices {
			payload.WriteString(extractDeltaContent(choice.Delta.Content))
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return payload.String(), nil
}

func extractDeltaContent(raw json.RawMessage) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return ""
	}

	var plain string
	if err := json.Unmarshal(raw, &plain); err == nil {
		return plain
	}

	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return ""
	}

	var out strings.Builder
	for _, p := range parts {
		if p.Type == "text" {
			out.WriteString(p.Text)
		}
	}
	return out.String()
}

func buildUserPrompt(docs []QADocumentInput, countRange QACountRange) string {
	docPayload := make([]map[string]any, 0, len(docs))
	for _, doc := range docs {
		docPayload = append(docPayload, map[string]any{
			"file_id":      doc.FileID,
			"title":        doc.Title,
			"file_path":    doc.FilePath,
			"url":          doc.URL,
			"content_text": doc.ContentText,
		})
	}
	b, _ := json.Marshal(docPayload)
	example := `合格示例（必须严格模仿输出格式，不要加任何前后缀）：
{"items":[{"question":"申请奖学金需要准备哪些材料？","answer":"通常需要提交成绩单和综合测评证明。\n另外，请在学院规定的截止日期前完成线上填报，避免错过申请时间。","keywords":["奖学金申请","成绩单","综合测评证明","线上填报"],"attachment_file_ids":[12]}]}

不合格示例（禁止）：
1) 使用 markdown 代码块：
[代码块开始]
{...}
[代码块结束]
2) 在 JSON 前后加解释文字：
下面是结果：{...}
3) JSON 不完整或被截断：
{"items":[{"question":"..."`
	return fmt.Sprintf(
		"请基于输入文档生成 %d 到 %d 条问答。\n"+
			"风格要求：\n"+
			"1) question 用正常人会问的话，避免官方标题式句子。\n"+
			"2) answer 先直接回答，再补充关键条件/步骤，固定 friendly 语气，表达温和但准确；涉及步骤或条件时必须用换行分段。\n"+
			"3) keywords 输出 2~5 个，优先名词短语，覆盖主题、材料、流程、时间节点。\n"+
			"   关键词禁止包含：文件名（如 .pdf/.docx/.xlsx）、字段名（如 学生姓名/班级/学号）、模板词（如 填表项/签字栏）、占位符（如下划线____、XXX、N/A）。\n"+
			"4) attachment_file_ids 只能引用下方提供的 file_id。\n"+
			"5) 必须仅输出一行完整 JSON：{\"items\":[...]}。\n"+
			"   严禁输出 ```json 代码块、严禁输出“下面是结果”等解释文字、严禁输出不完整 JSON。\n\n"+
			"%s\n\n"+
			"文档数据：%s",
		countRange.Min,
		countRange.Max,
		example,
		string(b),
	)
}

var splitter = regexp.MustCompile(`[，。；：,.!?、\n\r\t ]+`)

func firstSnippet(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	parts := splitter.Split(content, -1)
	filtered := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			filtered = append(filtered, p)
		}
		if len(filtered) >= 20 {
			break
		}
	}
	if len(filtered) == 0 {
		return ""
	}
	return strings.Join(filtered, " ")
}

func extractKeywords(content, fallback string) []string {
	parts := splitter.Split(strings.TrimSpace(content), -1)
	out := make([]string, 0, 3)
	seen := map[string]struct{}{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if len([]rune(p)) < 2 {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
		if len(out) == 3 {
			break
		}
	}
	if len(out) > 0 {
		return out
	}
	fallback = strings.TrimSpace(fallback)
	if fallback == "" {
		return []string{"知识库"}
	}
	return []string{fallback}
}

func truncateForLog(s string, max int) string {
	if max <= 0 {
		return ""
	}
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}

// extractFirstJSONObject returns the first complete top-level JSON object from text.
// It is tolerant to wrappers like markdown fences or explanatory prefixes/suffixes.
func extractFirstJSONObject(s string) string {
	s = strings.TrimSpace(s)
	start := strings.Index(s, "{")
	if start < 0 {
		return ""
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(s); i++ {
		ch := s[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return strings.TrimSpace(s[start : i+1])
			}
		}
	}
	return ""
}
