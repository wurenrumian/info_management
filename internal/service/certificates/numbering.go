package certificates

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

func buildCertificateNo(approvalType string, approvalID uint, now time.Time) string {
	prefix := strings.ToUpper(strings.TrimSpace(approvalType))
	if prefix == "" {
		prefix = "CERT"
	}
	return fmt.Sprintf("%s-%s-%06d", prefix, now.Format("20060102"), approvalID)
}

func buildVerificationCode(approvalType string, approvalID uint, now time.Time) string {
	prefix := strings.ToUpper(strings.TrimSpace(approvalType))
	if prefix == "" {
		prefix = "VC"
	}
	return fmt.Sprintf("%s-%d-%d", prefix, approvalID, now.UnixNano())
}

func buildVerificationHash(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}
