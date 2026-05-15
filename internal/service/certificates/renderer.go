package certificates

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// Renderer renders a certificate payload with a server-side template.
type Renderer interface {
	Render(ctx context.Context, templatePath string, payload map[string]any) ([]byte, error)
}

// NoopRenderer is the default lightweight renderer used in tests/dev.
type NoopRenderer struct{}

// NewNoopRenderer creates the default renderer implementation.
func NewNoopRenderer() Renderer {
	return NoopRenderer{}
}

// Render serializes the template path and payload as a placeholder artifact.
func (NoopRenderer) Render(_ context.Context, templatePath string, payload map[string]any) ([]byte, error) {
	if strings.TrimSpace(templatePath) == "" {
		return nil, errors.New("empty template path")
	}
	return json.Marshal(map[string]any{
		"template_path": templatePath,
		"payload":       payload,
	})
}

// FailingRenderer is used by tests to simulate render failures.
type FailingRenderer struct {
	Err error
}

// Render always fails with the configured error.
func (r FailingRenderer) Render(_ context.Context, _ string, _ map[string]any) ([]byte, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	return nil, errors.New("render failed")
}
