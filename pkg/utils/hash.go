package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
)

func GenerateAPIKey() (plainKey string, keyHash string, keyPrefix string, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", "", fmt.Errorf("failed to generate random key: %w", err)
	}
	plainKey = "sk-" + base64.URLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(plainKey))
	keyHash = fmt.Sprintf("%x", hash)
	keyPrefix = plainKey[:7]
	return
}

func GenerateAdminToken() string {
	return "admin-" + uuid.New().String()
}

func HashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", hash)
}

func ConstantTimeCompare(a, b string) bool {
	aHash := sha256.Sum256([]byte(a))
	bHash := sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(aHash[:], bHash[:]) == 1
}

func GenerateRequestID() string {
	return uuid.New().String()
}

func GenerateID() string {
	return uuid.New().String()[:8]
}
