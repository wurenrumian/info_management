package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModelTagsUseExplicitVarcharInsteadOfSize(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob model files: %v", err)
	}

	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		if strings.Contains(string(b), `size:`) {
			t.Fatalf("%s still contains gorm size tag", file)
		}
	}
}
