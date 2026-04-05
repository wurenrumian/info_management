package config

import "testing"

func TestJWTSecretUsesEnvOrDefault(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	if got := JWTSecret(); got != "test-secret" {
		t.Fatalf("JWTSecret() = %q, want %q", got, "test-secret")
	}

	t.Setenv("JWT_SECRET", "")
	if got := JWTSecret(); got != "dev-secret-change-in-production" {
		t.Fatalf("JWTSecret() = %q, want default", got)
	}
}

func TestPrimaryUploadDirPrefersDocumentThenDefault(t *testing.T) {
	t.Setenv("DOCUMENT_UPLOAD_DIR", "/tmp/documents")
	if got := PrimaryUploadDir(); got != "/tmp/documents" {
		t.Fatalf("PrimaryUploadDir() = %q, want %q", got, "/tmp/documents")
	}

	t.Setenv("DOCUMENT_UPLOAD_DIR", "")
	if got := PrimaryUploadDir(); got != "./data/uploads" {
		t.Fatalf("PrimaryUploadDir() = %q, want default", got)
	}
}
