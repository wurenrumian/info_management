package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"manage/internal/auth"
	"manage/internal/http/middleware"
	"manage/internal/testutil"
)

func TestJWTAuthInjectsActor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.JWTAuth("test-secret"))
	r.GET("/probe", func(c *gin.Context) {
		a, ok := auth.GetActor(c)
		require.True(t, ok, "expected actor in context")
		c.JSON(http.StatusOK, gin.H{"user_id": a.UserID, "role": a.Role})
	})

	t.Setenv("JWT_SECRET", "test-secret")
	token := testutil.GenerateTestToken(12, 3, 10, "2023")
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"user_id":12`)
	require.Contains(t, w.Body.String(), `"role":3`)
}

func TestJWTAuthRejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.JWTAuth("test-secret"))
	r.GET("/probe", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuthRejectsInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.JWTAuth("test-secret"))
	r.GET("/probe", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOptionalJWTAuthInjectsActorWhenTokenPresent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.OptionalJWTAuth("test-secret"))
	r.GET("/probe", func(c *gin.Context) {
		a, ok := auth.GetActor(c)
		require.True(t, ok, "expected actor in context")
		c.JSON(http.StatusOK, gin.H{"user_id": a.UserID, "role": a.Role})
	})

	t.Setenv("JWT_SECRET", "test-secret")
	token := testutil.GenerateTestToken(12, 3, 10, "2023")
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"user_id":12`)
	require.Contains(t, w.Body.String(), `"role":3`)
}

func TestOptionalJWTAuthPassesWhenNoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.OptionalJWTAuth("test-secret"))
	r.GET("/probe", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}
