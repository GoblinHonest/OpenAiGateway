package protocol

import (
	"testing"

	"github.com/example/aigateway/internal/cache"
)

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		path     string
		expected ProtocolFormat
	}{
		{"/v1/chat/completions", FormatOpenAI},
		{"/v1/completions", FormatOpenAI},
		{"/v1/embeddings", FormatOpenAI},
		{"/v1/messages", FormatAnthropic},
		{"/v1/models/gemini-pro:generateContent", FormatGemini},
		{"/unknown", FormatOpenAI},
	}

	for _, tt := range tests {
		result := detectFormatFromPath(tt.path)
		if result != tt.expected {
			t.Errorf("detectFormat(%q) = %s, want %s", tt.path, result, tt.expected)
		}
	}
}

func detectFormatFromPath(path string) ProtocolFormat {
	switch {
	case path == "/v1/chat/completions" || path == "/v1/completions" || path == "/v1/embeddings":
		return FormatOpenAI
	case path == "/v1/messages":
		return FormatAnthropic
	case contains(path, "generateContent") || contains(path, "gemini"):
		return FormatGemini
	default:
		return FormatOpenAI
	}
}

func TestParseCacheUsage_OpenAI(t *testing.T) {
	usage := map[string]any{
		"prompt_tokens_details": map[string]any{
			"cached_tokens": float64(800),
		},
	}

	result := cache.ParseCacheUsage("openai", usage)
	if !result.IsCacheHit {
		t.Error("expected cache hit")
	}
	if result.CacheReadTokens != 800 {
		t.Errorf("expected 800 cached tokens, got %d", result.CacheReadTokens)
	}
}

func TestParseCacheUsage_Anthropic(t *testing.T) {
	usage := map[string]any{
		"cache_read_input_tokens":     float64(500),
		"cache_creation_input_tokens": float64(200),
	}

	result := cache.ParseCacheUsage("anthropic", usage)
	if !result.IsCacheHit {
		t.Error("expected cache hit")
	}
	if result.CacheReadTokens != 500 {
		t.Errorf("expected 500 cached tokens, got %d", result.CacheReadTokens)
	}
}

func TestParseCacheUsage_NoCache(t *testing.T) {
	usage := map[string]any{}

	result := cache.ParseCacheUsage("openai", usage)
	if result.IsCacheHit {
		t.Error("expected no cache hit")
	}
}

func TestProtocolFormatValues(t *testing.T) {
	if FormatOpenAI != "openai" {
		t.Errorf("expected openai, got %s", FormatOpenAI)
	}
	if FormatAnthropic != "anthropic" {
		t.Errorf("expected anthropic, got %s", FormatAnthropic)
	}
	if FormatGemini != "gemini" {
		t.Errorf("expected gemini, got %s", FormatGemini)
	}
}
