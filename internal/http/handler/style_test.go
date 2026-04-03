package handler

import (
	"os"
	"strings"
	"testing"
)

func TestHandlersAvoidAnonymousRequestStructs(t *testing.T) {
	files := []string{
		"notification_handler.go",
		"me_handler.go",
	}

	for _, file := range files {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		if strings.Contains(string(b), "var req struct") {
			t.Fatalf("%s still uses anonymous request struct", file)
		}
	}
}
