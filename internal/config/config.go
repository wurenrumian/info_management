package config

import (
	"os"
	"strings"
)

const (
	defaultJWTSecret  = "dev-secret-change-in-production"
	defaultUploadDir  = "./data/uploads"
	defaultServerPort = "8080"
)

const (
	DefaultDevClassID      uint   = 10
	DefaultDevGrade        string = "2020"
	DefaultDevMajor        string = "信息管理"
	DefaultPublicClassID   uint   = 9999
	DefaultPublicClassName        = "未绑定班级"
)

func DatabaseDSN() string {
	return strings.TrimSpace(os.Getenv("DATABASE_DSN"))
}

func Port() string {
	return getEnv("PORT", defaultServerPort)
}

func JWTSecret() string {
	return getEnv("JWT_SECRET", defaultJWTSecret)
}

func WechatAppID() string {
	return strings.TrimSpace(os.Getenv("WECHAT_APP_ID"))
}

func WechatAppSecret() string {
	return strings.TrimSpace(os.Getenv("WECHAT_APP_SECRET"))
}

func AIBaseURL() string {
	return strings.TrimSpace(os.Getenv("AI_BASE_URL"))
}

func AIProvider() string {
	return getEnv("AI_PROVIDER", "openrouter")
}

func AIAPIKey() string {
	return strings.TrimSpace(os.Getenv("AI_API_KEY"))
}

func AIModel() string {
	return getEnv("AI_MODEL", "openai/gpt-5.2")
}

func DocumentUploadDir() string {
	return getEnv("DOCUMENT_UPLOAD_DIR", defaultUploadDir)
}

func PrimaryUploadDir() string {
	return getEnv("DOCUMENT_UPLOAD_DIR", defaultUploadDir)
}

func IsDevEnv() bool {
	return strings.TrimSpace(os.Getenv("APP_ENV")) == "dev"
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
