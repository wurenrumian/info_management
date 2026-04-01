package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateAndParseToken(t *testing.T) {
	secret := "test-secret-key-32bytes!!"
	token, err := GenerateToken(1, 1, 10, "2023", secret)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := ParseToken(token, secret)
	require.NoError(t, err)
	require.Equal(t, uint(1), claims.UserID)
	require.Equal(t, 1, claims.Role)
	require.Equal(t, uint(10), claims.ClassID)
	require.Equal(t, "2023", claims.Grade)
}

func TestParseTokenRejectsInvalid(t *testing.T) {
	_, err := ParseToken("invalid.token.here", "wrong-secret")
	require.Error(t, err)
}
