package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"manage/internal/http/router"
)

func TestHealthz(t *testing.T) {
	r := router.New(nil)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if got := w.Body.String(); got != "{\"status\":\"ok\"}" {
		t.Fatalf("expected body %q, got %q", "{\"status\":\"ok\"}", got)
	}
}
