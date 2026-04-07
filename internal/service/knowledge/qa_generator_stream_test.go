package knowledge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAICompatGeneratorGenerateUsesSSEStream(t *testing.T) {
	var gotStream bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		if v, ok := payload["stream"].(bool); ok {
			gotStream = v
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"{\\\"items\\\":[\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"{\\\"question\\\":\\\"q\\\",\\\"answer\\\":\\\"a\\\",\\\"attachment_file_ids\\\":[1]}\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"]}\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	g := NewOpenAICompatGenerator("", server.URL, "test-key", "openai/gpt-5.2")
	g.client = server.Client()

	items, err := g.Generate(context.Background(), []QADocumentInput{{
		FileID:      1,
		Title:       "doc1",
		ContentText: "abc",
	}}, QACountRange{Min: 1, Max: 2})

	require.NoError(t, err)
	require.True(t, gotStream)
	require.Len(t, items, 1)
	require.Equal(t, "q", items[0].Question)
	require.Equal(t, "a", items[0].Answer)
	require.Equal(t, []uint{1}, items[0].AttachmentFileIDs)
}
