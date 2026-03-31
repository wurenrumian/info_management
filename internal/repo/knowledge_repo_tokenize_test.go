package repo

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTokenizeSearchQuery(t *testing.T) {
	tokens := tokenizeSearchQuery("奖学金申请 materials 2026")
	require.Contains(t, tokens, "奖学金申请")
	require.Contains(t, tokens, "奖学")
	require.Contains(t, tokens, "学金")
	require.Contains(t, tokens, "金申")
	require.Contains(t, tokens, "申请")
	require.Contains(t, tokens, "materials")
	require.Contains(t, tokens, "2026")
}
