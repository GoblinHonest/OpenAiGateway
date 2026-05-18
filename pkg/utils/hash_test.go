package utils

import (
	"testing"
)

func TestGenerateAPIKey_Uniqueness(t *testing.T) {
	keys := make(map[string]bool)

	for i := 0; i < 10; i++ {
		plainKey, _, _, err := GenerateAPIKey()
		if err != nil {
			t.Fatalf("GenerateAPIKey failed: %v", err)
		}

		if len(plainKey) == 0 {
			t.Fatal("expected non-empty key")
		}

		if keys[plainKey] {
			t.Errorf("duplicate key generated: %q", plainKey)
		}
		keys[plainKey] = true
	}
}

func TestGenerateAPIKey_Format(t *testing.T) {
	plainKey, keyHash, keyPrefix, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey failed: %v", err)
	}

	// Should start with "sk-"
	if len(plainKey) < 3 || plainKey[:3] != "sk-" {
		t.Errorf("key should start with sk-, got %q", plainKey)
	}

	// Prefix should be first 7 chars
	if keyPrefix != plainKey[:7] {
		t.Errorf("prefix mismatch: %q != %q", keyPrefix, plainKey[:7])
	}

	// Hash should be non-empty
	if keyHash == "" {
		t.Error("keyHash should not be empty")
	}
}

func TestGenerateRequestID_Uniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateRequestID()
		if ids[id] {
			t.Errorf("duplicate request ID: %q", id)
		}
		ids[id] = true
	}
}

func TestGenerateID_Format(t *testing.T) {
	id := GenerateID()
	if len(id) != 8 {
		t.Errorf("expected 8-char ID, got %q (len=%d)", id, len(id))
	}
}

func TestHashKey_Deterministic(t *testing.T) {
	h1 := HashKey("my-secret-key")
	h2 := HashKey("my-secret-key")
	if h1 != h2 {
		t.Errorf("HashKey should be deterministic: %q != %q", h1, h2)
	}
}

func TestHashKey_DifferentInputs(t *testing.T) {
	h1 := HashKey("key-a")
	h2 := HashKey("key-b")
	if h1 == h2 {
		t.Error("different inputs should produce different hashes")
	}
}

func TestConstantTimeCompare(t *testing.T) {
	if !ConstantTimeCompare("abc", "abc") {
		t.Error("same strings should match")
	}
	if ConstantTimeCompare("abc", "abd") {
		t.Error("different strings should not match")
	}
}
