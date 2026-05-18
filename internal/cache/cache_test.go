package cache

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestBuildCacheKey_Deterministic(t *testing.T) {
	body := []byte(`{"model":"gpt-4","messages":[{"role":"user","content":"hello"}]}`)

	key1 := BuildCacheKey("openai", "gpt-4", body)
	key2 := BuildCacheKey("openai", "gpt-4", body)

	if key1 != key2 {
		t.Errorf("expected same key for same input, got %q != %q", key1, key2)
	}
}

func TestBuildCacheKey_DifferentInputs(t *testing.T) {
	key1 := BuildCacheKey("openai", "gpt-4", []byte(`{"model":"gpt-4"}`))
	key2 := BuildCacheKey("openai", "gpt-4", []byte(`{"model":"gpt-3.5"}`))

	if key1 == key2 {
		t.Errorf("expected different keys for different bodies")
	}
}

func TestBuildCacheKey_Prefix(t *testing.T) {
	key := BuildCacheKey("openai", "gpt-4", []byte(`{}`))

	if key[:9] != "llm_cache" {
		t.Errorf("expected prefix llm_cache, got %q", key[:9])
	}
}

func TestBuildCacheKey_DifferentProviders(t *testing.T) {
	body := []byte(`{}`)
	key1 := BuildCacheKey("openai", "gpt-4", body)
	key2 := BuildCacheKey("anthropic", "gpt-4", body)

	if key1 == key2 {
		t.Errorf("expected different keys for different providers")
	}
}

func TestBuildCacheStatsKey(t *testing.T) {
	key := BuildCacheStatsKey("openai")
	if key != "cache_stats:openai" {
		t.Errorf("expected cache_stats:openai, got %q", key)
	}
}

func TestCacheEntry_MarshalRoundtrip(t *testing.T) {
	entry := &CacheEntry{
		Body:       []byte(`{"id":"chatcmpl-123","choices":[]}`),
		StatusCode: 200,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}

	var decoded CacheEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.StatusCode != 200 {
		t.Errorf("expected 200, got %d", decoded.StatusCode)
	}
	if string(decoded.Body) != string(entry.Body) {
		t.Errorf("body mismatch")
	}
}

func TestParseCacheUsage_DeepSeek(t *testing.T) {
	usage := map[string]any{
		"prompt_cache_hit_tokens":  float64(300),
		"prompt_cache_miss_tokens": float64(700),
	}

	result := ParseCacheUsage("deepseek", usage)
	if !result.IsCacheHit {
		t.Error("expected cache hit")
	}
	if result.CacheReadTokens != 300 {
		t.Errorf("expected 300 cached tokens, got %d", result.CacheReadTokens)
	}
}

func TestParseCacheUsage_Nil(t *testing.T) {
	result := ParseCacheUsage("openai", nil)
	if result.IsCacheHit {
		t.Error("expected no cache hit for nil usage")
	}
}

func TestExtractUsageFromResponse_Default(t *testing.T) {
	resp := map[string]any{
		"usage": map[string]any{
			"prompt_tokens":     float64(100),
			"completion_tokens": float64(50),
		},
	}
	body, _ := json.Marshal(resp)

	input, output, cached := ExtractUsageFromResponse("unknown_provider", body)
	if input != 100 {
		t.Errorf("expected 100 input tokens, got %d", input)
	}
	if output != 50 {
		t.Errorf("expected 50 output tokens, got %d", output)
	}
	if cached != 0 {
		t.Errorf("expected 0 cached tokens, got %d", cached)
	}
}

func TestExtractUsageFromResponse_InvalidJSON(t *testing.T) {
	input, output, cached := ExtractUsageFromResponse("openai", []byte(`not-json`))
	if input != 0 || output != 0 || cached != 0 {
		t.Errorf("expected all zeros for invalid JSON, got %d/%d/%d", input, output, cached)
	}
}

// Benchmark for key generation
func BenchmarkBuildCacheKey(b *testing.B) {
	body := []byte(`{"model":"gpt-4","messages":[...]}`)
	for i := 0; i < b.N; i++ {
		_ = BuildCacheKey("openai", "gpt-4", body)
	}
}

func ExampleBuildCacheKey() {
	key := BuildCacheKey("openai", "gpt-4", []byte(`{}`))
	fmt.Println(key[:9]) // prefix is always llm_cache
	// Output: llm_cache
}
