package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"manage/internal/auth"
	"manage/internal/http/middleware"
)

func TestIdentityMiddlewareInjectsActor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.IdentityFromHeaders())
	r.GET("/probe", func(c *gin.Context) {
		a, ok := auth.GetActor(c)
		if !ok {
			t.Fatal("expected actor in context")
		}
		c.JSON(http.StatusOK, gin.H{"user_id": a.UserID, "role": a.Role})
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("X-User-Id", "12")
	req.Header.Set("X-User-Role", "3")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if got := w.Body.String(); got != "{\"role\":3,\"user_id\":12}" {
		t.Fatalf("unexpected body: %s", got)
	}
}
