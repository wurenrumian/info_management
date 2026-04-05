package knowledge_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	ksvc "manage/internal/service/knowledge"
)

func TestParseDraftsRejectsInvalidJSON(t *testing.T) {
	allowed := map[uint]struct{}{1: {}}
	_, err := ksvc.ParseDraftsJSON(`{`, ksvc.QACountRange{Min: 1, Max: 2}, allowed)
	require.ErrorIs(t, err, ksvc.ErrInvalidDraftJSON)
}

func TestParseDraftsRejectsOutOfRangeCount(t *testing.T) {
	allowed := map[uint]struct{}{1: {}}
	raw := `{"items":[{"question":"q","answer":"a","keywords":["k"],"attachment_file_ids":[1]}]}`
	_, err := ksvc.ParseDraftsJSON(raw, ksvc.QACountRange{Min: 2, Max: 3}, allowed)
	require.ErrorIs(t, err, ksvc.ErrInvalidDraftItem)
}

func TestParseDraftsAcceptsValidPayload(t *testing.T) {
	allowed := map[uint]struct{}{1: {}}
	raw := `{"items":[{"question":"q","answer":"a","keywords":["k"],"attachment_file_ids":[1]}]}`
	items, err := ksvc.ParseDraftsJSON(raw, ksvc.QACountRange{Min: 1, Max: 2}, allowed)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "q", items[0].Question)
}

func TestParseDraftsAcceptsMissingKeywords(t *testing.T) {
	allowed := map[uint]struct{}{1: {}}
	raw := `{"items":[{"question":"q","answer":"a","attachment_file_ids":[1]}]}`
	items, err := ksvc.ParseDraftsJSON(raw, ksvc.QACountRange{Min: 1, Max: 2}, allowed)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Nil(t, items[0].Keywords)
}

func TestParseDraftsAcceptsWrappedJSON(t *testing.T) {
	allowed := map[uint]struct{}{1: {}}
	raw := "下面是结果：\n```json\n{\"items\":[{\"question\":\"q\",\"answer\":\"a\",\"keywords\":[\"k\"],\"attachment_file_ids\":[1]}]}\n```\n"
	items, err := ksvc.ParseDraftsJSON(raw, ksvc.QACountRange{Min: 1, Max: 2}, allowed)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "q", items[0].Question)
}

func TestParseDraftsRejectsNoCompleteJSONObject(t *testing.T) {
	allowed := map[uint]struct{}{1: {}}
	raw := "{\"items\":[{\"question\":\"q\""
	_, err := ksvc.ParseDraftsJSON(raw, ksvc.QACountRange{Min: 1, Max: 2}, allowed)
	require.ErrorIs(t, err, ksvc.ErrInvalidDraftJSON)
}

func TestGeneratePreviewFailsWhenAIConfigMissing(t *testing.T) {
	g := ksvc.NewOpenAICompatGenerator("", "", "", "openai/gpt-5.2")
	_, err := g.Generate(context.Background(), []ksvc.QADocumentInput{
		{FileID: 1, Title: "doc1", ContentText: "abc"},
	}, ksvc.QACountRange{Min: 1, Max: 2})
	require.ErrorIs(t, err, ksvc.ErrGeneratePreview)
}

func TestNewOpenAICompatGeneratorDefaultsToOpenRouter(t *testing.T) {
	g := ksvc.NewOpenAICompatGenerator("", "", "test-key", "")
	require.Equal(t, "openrouter", g.Provider())
	require.Equal(t, "https://openrouter.ai/api", g.BaseURL())
	require.Equal(t, "openai/gpt-5.2", g.Model())
}

func TestParseDraftsAcceptsRequestedCountRange(t *testing.T) {
	allowed := map[uint]struct{}{99901: {}}
	raw := `{"items":[{"question":"q1","answer":"a1","attachment_file_ids":[99901]},{"question":"q2","answer":"a2","attachment_file_ids":[99901]}]}`
	items, err := ksvc.ParseDraftsJSON(raw, ksvc.QACountRange{Min: 1, Max: 3}, allowed)
	require.NoError(t, err)
	require.Len(t, items, 2)
}

func TestOpenAICompatGeneratorIntegration(t *testing.T) {
	if os.Getenv("RUN_AI_INTEGRATION_TEST") != "1" {
		t.Skip("set RUN_AI_INTEGRATION_TEST=1 to enable real AI integration test")
	}

	baseURL := os.Getenv("AI_BASE_URL")
	apiKey := os.Getenv("AI_API_KEY")
	model := os.Getenv("AI_MODEL")

	if apiKey == "" || model == "" {
		t.Skip("AI_API_KEY / AI_MODEL are required")
	}

	g := ksvc.NewOpenAICompatGenerator("", baseURL, apiKey, model)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	items, err := g.Generate(ctx, []ksvc.QADocumentInput{
		{
			FileID:      99901,
			Title:       "奖学金申请须知.docx",
			FilePath:    "knowledge/2026/04/demo.docx",
			URL:         "/uploads/knowledge/2026/04/demo.docx",
			ContentText: "奖学金申请需提交成绩单、综测证明，并在学院截止日前完成线上填报。",
		},
	}, ksvc.QACountRange{Min: 1, Max: 3})

	require.NoError(t, err)
	require.NotEmpty(t, items)
	require.NotEmpty(t, items[0].Question)
	require.NotEmpty(t, items[0].Answer)
	require.NotEmpty(t, items[0].AttachmentFileIDs)
	require.Equal(t, uint(99901), items[0].AttachmentFileIDs[0])
}
