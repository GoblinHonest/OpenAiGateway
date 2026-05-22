package protocol

import (
	"context"
	"encoding/json"
	"time"
	"unicode"

	"github.com/example/aigateway/pkg/utils"
)

// estimateTokens 根据文本内容估算 token 数
func estimateTokens(text string) int {
	tokens := 0
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			tokens += 1 // 中文约 1-2 字符/token，取 1 偏多
		} else {
			tokens += 1 // 英文约 4 字符/token，按字符计偏多但安全
		}
	}
	// 粗略除以 4 作为英文基准，中文已经按 1 算了所以整体偏保守
	if tokens > 0 {
		tokens = tokens/3 + 1
	}
	return tokens
}

type OpenAIToAnthropicStreamConverter struct{}

func NewOpenAIToAnthropicStreamConverter() *OpenAIToAnthropicStreamConverter {
	return &OpenAIToAnthropicStreamConverter{}
}

func (c *OpenAIToAnthropicStreamConverter) ConvertStream(
	ctx context.Context,
	sourceStream <-chan StreamChunk,
	targetWriter StreamWriter,
) error {
	messageStart := map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":      "msg_" + utils.GenerateID(),
			"type":    "message",
			"role":    "assistant",
			"content": []any{},
			"usage":   map[string]any{"input_tokens": 0, "output_tokens": 0},
		},
	}
	if err := targetWriter.WriteEvent("message_start", messageStart); err != nil {
		return err
	}

	contentBlockStart := map[string]any{
		"type": "content_block_start",
		"index": 0,
		"content_block": map[string]any{
			"type": "text",
			"text": "",
		},
	}
	if err := targetWriter.WriteEvent("content_block_start", contentBlockStart); err != nil {
		return err
	}

	var outputText string
	var hasUsage bool

	for chunk := range sourceStream {
		if chunk.Error != nil {
			return chunk.Error
		}
		if chunk.Done {
			break
		}

		var openAIChunk struct {
			Choices []struct {
				Index        int    `json:"index"`
				Delta        struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage,omitempty"`
		}
		if err := json.Unmarshal(chunk.Data, &openAIChunk); err != nil {
			continue
		}

		if openAIChunk.Usage != nil && (openAIChunk.Usage.PromptTokens > 0 || openAIChunk.Usage.CompletionTokens > 0) {
			hasUsage = true
		}

		if len(openAIChunk.Choices) > 0 {
			choice := openAIChunk.Choices[0]

			if choice.Delta.Content != "" {
				outputText += choice.Delta.Content
				delta := map[string]any{
					"type": "content_block_delta",
					"index": 0,
					"delta": map[string]any{
						"type": "text_delta",
						"text": choice.Delta.Content,
					},
				}
				if err := targetWriter.WriteEvent("content_block_delta", delta); err != nil {
					return err
				}
			}

			if choice.FinishReason != nil && *choice.FinishReason != "" {
				if err := targetWriter.WriteEvent("content_block_stop", map[string]any{
					"type":  "content_block_stop",
					"index": 0,
				}); err != nil {
					return err
				}

				outputTokens := 0
				if hasUsage && openAIChunk.Usage != nil {
					outputTokens = openAIChunk.Usage.CompletionTokens
				} else {
					outputTokens = estimateTokens(outputText)
				}

				usage := map[string]any{"output_tokens": outputTokens}

				if err := targetWriter.WriteEvent("message_delta", map[string]any{
					"type": "message_delta",
					"delta": map[string]any{
						"stop_reason": mapOpenAIFinishReason(*choice.FinishReason),
					},
					"usage": usage,
				}); err != nil {
					return err
				}

				if err := targetWriter.WriteEvent("message_stop", map[string]any{
					"type": "message_stop",
				}); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func mapOpenAIFinishReason(reason string) string {
	switch reason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "content_filter":
		return "content_filter"
	default:
		return "end_turn"
	}
}

type AnthropicToOpenAIStreamConverter struct{}

func NewAnthropicToOpenAIStreamConverter() *AnthropicToOpenAIStreamConverter {
	return &AnthropicToOpenAIStreamConverter{}
}

func (c *AnthropicToOpenAIStreamConverter) ConvertStream(
	ctx context.Context,
	sourceStream <-chan StreamChunk,
	targetWriter StreamWriter,
) error {
	chunkID := "chatcmpl-" + utils.GenerateID()
	created := time.Now().Unix()
	contentStarted := false

	for chunk := range sourceStream {
		if chunk.Error != nil {
			return chunk.Error
		}
		if chunk.Done {
			break
		}

		// 使用 Event 字段或从 JSON 中解析 type
		eventType := chunk.Event
		if eventType == "" {
			var event map[string]any
			if err := json.Unmarshal(chunk.Data, &event); err != nil {
				continue
			}
			eventType, _ = event["type"].(string)
		}

		// 解析事件数据
		var event map[string]any
		if err := json.Unmarshal(chunk.Data, &event); err != nil {
			continue
		}

		switch eventType {
		case "message_start":
			openAIChunk := map[string]any{
				"id":      chunkID,
				"object":  "chat.completion.chunk",
				"created": created,
				"model":   "gpt-4",
				"choices": []map[string]any{
					{
						"index": 0,
						"delta": map[string]any{
							"role": "assistant",
						},
						"finish_reason": nil,
					},
				},
			}
			if err := targetWriter.WriteJSON(openAIChunk); err != nil {
				return err
			}
			contentStarted = true

		case "content_block_delta":
			if !contentStarted {
				continue
			}
			delta, _ := event["delta"].(map[string]any)
			text, _ := delta["text"].(string)
			if text == "" {
				continue
			}

			openAIChunk := map[string]any{
				"id":      chunkID,
				"object":  "chat.completion.chunk",
				"created": created,
				"model":   "gpt-4",
				"choices": []map[string]any{
					{
						"index": 0,
						"delta": map[string]any{
							"content": text,
						},
						"finish_reason": nil,
					},
				},
			}
			if err := targetWriter.WriteJSON(openAIChunk); err != nil {
				return err
			}

		case "message_delta":
			delta, _ := event["delta"].(map[string]any)
			stopReason, _ := delta["stop_reason"].(string)

			openAIChunk := map[string]any{
				"id":      chunkID,
				"object":  "chat.completion.chunk",
				"created": created,
				"model":   "gpt-4",
				"choices": []map[string]any{
					{
						"index": 0,
						"delta":        map[string]any{},
						"finish_reason": mapAnthropicStopReason(stopReason),
					},
				},
			}
			if err := targetWriter.WriteJSON(openAIChunk); err != nil {
				return err
			}

			targetWriter.Close()
			return nil
		}
	}

	return nil
}

func mapAnthropicStopReason(reason string) string {
	switch reason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	default:
		return "stop"
	}
}

// ============================================================
// Gemini stream converters
// ============================================================

// --- OpenAI → Gemini ---

type OpenAIToGeminiStreamConverter struct{}

func NewOpenAIToGeminiStreamConverter() *OpenAIToGeminiStreamConverter {
	return &OpenAIToGeminiStreamConverter{}
}

func (c *OpenAIToGeminiStreamConverter) ConvertStream(
	ctx context.Context,
	sourceStream <-chan StreamChunk,
	targetWriter StreamWriter,
) error {
	contentStarted := false

	for chunk := range sourceStream {
		if chunk.Error != nil {
			return chunk.Error
		}
		if chunk.Done {
			break
		}

		var openAIChunk struct {
			Choices []struct {
				Index int `json:"index"`
				Delta struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage,omitempty"`
		}
		if err := json.Unmarshal(chunk.Data, &openAIChunk); err != nil {
			continue
		}

		if len(openAIChunk.Choices) == 0 {
			continue
		}
		choice := openAIChunk.Choices[0]

		geminiChunk := map[string]any{
			"candidates": []map[string]any{
				{
					"content": map[string]any{
						"parts": []map[string]any{
							{"text": choice.Delta.Content},
						},
						"role": "model",
					},
					"index": choice.Index,
				},
			},
		}

		if choice.FinishReason != nil && *choice.FinishReason != "" {
			geminiChunk["candidates"].([]map[string]any)[0]["finishReason"] = mapOpenAIFinishReasonToGemini(*choice.FinishReason)
		}
		if openAIChunk.Usage != nil {
			geminiChunk["usageMetadata"] = map[string]any{
				"promptTokenCount":     openAIChunk.Usage.PromptTokens,
				"candidatesTokenCount": openAIChunk.Usage.CompletionTokens,
				"totalTokenCount":      openAIChunk.Usage.TotalTokens,
			}
		}

		if err := targetWriter.WriteJSON(geminiChunk); err != nil {
			return err
		}

		if choice.Delta.Content != "" {
			contentStarted = true
		}
	}

	_ = contentStarted
	return nil
}

// --- Gemini → OpenAI ---

type GeminiToOpenAIStreamConverter struct{}

func NewGeminiToOpenAIStreamConverter() *GeminiToOpenAIStreamConverter {
	return &GeminiToOpenAIStreamConverter{}
}

func (c *GeminiToOpenAIStreamConverter) ConvertStream(
	ctx context.Context,
	sourceStream <-chan StreamChunk,
	targetWriter StreamWriter,
) error {
	chunkID := "chatcmpl-" + utils.GenerateID()
	created := time.Now().Unix()

	for chunk := range sourceStream {
		if chunk.Error != nil {
			return chunk.Error
		}
		if chunk.Done {
			break
		}

		var geminiChunk struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
					Role string `json:"role"`
				} `json:"content"`
				FinishReason string `json:"finishReason"`
				Index        int    `json:"index"`
			} `json:"candidates"`
			UsageMetadata *struct {
				PromptTokenCount     int `json:"promptTokenCount"`
				CandidatesTokenCount int `json:"candidatesTokenCount"`
				TotalTokenCount      int `json:"totalTokenCount"`
			} `json:"usageMetadata,omitempty"`
		}
		if err := json.Unmarshal(chunk.Data, &geminiChunk); err != nil {
			continue
		}

		if len(geminiChunk.Candidates) == 0 {
			continue
		}
		cand := geminiChunk.Candidates[0]

		text := ""
		for _, part := range cand.Content.Parts {
			text += part.Text
		}

		openAIChunk := map[string]any{
			"id":      chunkID,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   "gemini-pro",
			"choices": []map[string]any{
				{
					"index": cand.Index,
					"delta": map[string]any{
						"content": text,
					},
				},
			},
		}

		if cand.Content.Role == "model" {
			openAIChunk["choices"].([]map[string]any)[0]["delta"].(map[string]any)["role"] = "assistant"
		}

		if cand.FinishReason != "" {
			openAIChunk["choices"].([]map[string]any)[0]["finish_reason"] = mapGeminiFinishReason(cand.FinishReason)
		}
		if geminiChunk.UsageMetadata != nil {
			openAIChunk["usage"] = map[string]any{
				"prompt_tokens":     geminiChunk.UsageMetadata.PromptTokenCount,
				"completion_tokens": geminiChunk.UsageMetadata.CandidatesTokenCount,
				"total_tokens":      geminiChunk.UsageMetadata.TotalTokenCount,
			}
		}

		if err := targetWriter.WriteJSON(openAIChunk); err != nil {
			return err
		}
	}

	return nil
}

// --- Anthropic → Gemini ---

type AnthropicToGeminiStreamConverter struct{}

func NewAnthropicToGeminiStreamConverter() *AnthropicToGeminiStreamConverter {
	return &AnthropicToGeminiStreamConverter{}
}

func (c *AnthropicToGeminiStreamConverter) ConvertStream(
	ctx context.Context,
	sourceStream <-chan StreamChunk,
	targetWriter StreamWriter,
) error {
	for chunk := range sourceStream {
		if chunk.Error != nil {
			return chunk.Error
		}
		if chunk.Done {
			break
		}

		var event map[string]any
		if err := json.Unmarshal(chunk.Data, &event); err != nil {
			continue
		}

		eventType, _ := event["type"].(string)

		switch eventType {
		case "content_block_delta":
			delta, _ := event["delta"].(map[string]any)
			text, _ := delta["text"].(string)

			geminiChunk := map[string]any{
				"candidates": []map[string]any{
					{
						"content": map[string]any{
							"parts": []map[string]any{
								{"text": text},
							},
							"role": "model",
						},
						"index": 0,
					},
				},
			}
			if err := targetWriter.WriteJSON(geminiChunk); err != nil {
				return err
			}

		case "message_delta":
			delta, _ := event["delta"].(map[string]any)
			stopReason, _ := delta["stop_reason"].(string)
			usage, _ := event["usage"].(map[string]any)

			geminiChunk := map[string]any{
				"candidates": []map[string]any{
					{
						"content": map[string]any{
							"parts": []map[string]any{},
							"role":  "model",
						},
						"finishReason": mapAnthropicStopReasonToGemini(stopReason),
						"index":        0,
					},
				},
			}
			if usage != nil {
				outputTokens, _ := usage["output_tokens"].(float64)
				geminiChunk["usageMetadata"] = map[string]any{
					"candidatesTokenCount": int(outputTokens),
				}
			}
			if err := targetWriter.WriteJSON(geminiChunk); err != nil {
				return err
			}
			return nil
		}
	}

	return nil
}

// --- Gemini → Anthropic ---

type GeminiToAnthropicStreamConverter struct{}

func NewGeminiToAnthropicStreamConverter() *GeminiToAnthropicStreamConverter {
	return &GeminiToAnthropicStreamConverter{}
}

func (c *GeminiToAnthropicStreamConverter) ConvertStream(
	ctx context.Context,
	sourceStream <-chan StreamChunk,
	targetWriter StreamWriter,
) error {
	// Send message_start
	messageStart := map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":      "msg_" + utils.GenerateID(),
			"type":    "message",
			"role":    "assistant",
			"content": []any{},
			"usage":   map[string]any{"input_tokens": 0, "output_tokens": 0},
		},
	}
	if err := targetWriter.WriteEvent("message_start", messageStart); err != nil {
		return err
	}

	// Send content_block_start
	contentBlockStart := map[string]any{
		"type": "content_block_start",
		"index": 0,
		"content_block": map[string]any{
			"type": "text",
			"text": "",
		},
	}
	if err := targetWriter.WriteEvent("content_block_start", contentBlockStart); err != nil {
		return err
	}

	for chunk := range sourceStream {
		if chunk.Error != nil {
			return chunk.Error
		}
		if chunk.Done {
			break
		}

		var geminiChunk struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
				FinishReason string `json:"finishReason"`
			} `json:"candidates"`
			UsageMetadata *struct {
				PromptTokenCount     int `json:"promptTokenCount"`
				CandidatesTokenCount int `json:"candidatesTokenCount"`
				TotalTokenCount      int `json:"totalTokenCount"`
			} `json:"usageMetadata,omitempty"`
		}
		if err := json.Unmarshal(chunk.Data, &geminiChunk); err != nil {
			continue
		}

		if len(geminiChunk.Candidates) == 0 {
			continue
		}
		cand := geminiChunk.Candidates[0]

		text := ""
		for _, part := range cand.Content.Parts {
			text += part.Text
		}

		if text != "" {
			delta := map[string]any{
				"type": "content_block_delta",
				"index": 0,
				"delta": map[string]any{
					"type": "text_delta",
					"text": text,
				},
			}
			if err := targetWriter.WriteEvent("content_block_delta", delta); err != nil {
				return err
			}
		}

		if cand.FinishReason != "" {
			// content_block_stop
			if err := targetWriter.WriteEvent("content_block_stop", map[string]any{
				"type":  "content_block_stop",
				"index": 0,
			}); err != nil {
				return err
			}

			usage := map[string]any{"output_tokens": 0}
			if geminiChunk.UsageMetadata != nil {
				usage["output_tokens"] = geminiChunk.UsageMetadata.CandidatesTokenCount
			}

			// message_delta
			if err := targetWriter.WriteEvent("message_delta", map[string]any{
				"type": "message_delta",
				"delta": map[string]any{
					"stop_reason": mapGeminiFinishReasonToAnthropic(cand.FinishReason),
				},
				"usage": usage,
			}); err != nil {
				return err
			}

			// message_stop
			if err := targetWriter.WriteEvent("message_stop", map[string]any{
				"type": "message_stop",
			}); err != nil {
				return err
			}
			return nil
		}
	}

	return nil
}

// --- Gemini finish reason mappings ---

func mapOpenAIFinishReasonToGemini(reason string) string {
	switch reason {
	case "stop":
		return "STOP"
	case "length":
		return "MAX_TOKENS"
	case "content_filter":
		return "SAFETY"
	default:
		return "STOP"
	}
}

func mapGeminiFinishReason(reason string) string {
	switch reason {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY":
		return "content_filter"
	default:
		return "stop"
	}
}

func mapAnthropicStopReasonToGemini(reason string) string {
	switch reason {
	case "end_turn":
		return "STOP"
	case "max_tokens":
		return "MAX_TOKENS"
	default:
		return "STOP"
	}
}

func mapGeminiFinishReasonToAnthropic(reason string) string {
	switch reason {
	case "STOP":
		return "end_turn"
	case "MAX_TOKENS":
		return "max_tokens"
	default:
		return "end_turn"
	}
}
