package protocol

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/example/aigateway/pkg/logger"
	"github.com/example/aigateway/pkg/utils"
	"go.uber.org/zap"
)

type AnthropicRequest struct {
	Model       string              `json:"model"`
	Messages    []AnthropicMessage `json:"messages"`
	System      string              `json:"system,omitempty"`
	MaxTokens   int                 `json:"max_tokens"`
	Stream      bool                `json:"stream,omitempty"`
	Temperature float64             `json:"temperature,omitempty"`
	TopP        float64             `json:"top_p,omitempty"`
}

type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type AnthropicResponse struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Role         string             `json:"role"`
	Content      []AnthropicContent `json:"content"`
	StopReason   string             `json:"stop_reason"`
	StopSequence string             `json:"stop_sequence"`
	Usage        AnthropicUsage     `json:"usage"`
}

type AnthropicUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
}

type AnthropicConverter struct{}

func NewAnthropicConverter() *AnthropicConverter {
	return &AnthropicConverter{}
}

func (c *AnthropicConverter) DetectFormat(r *http.Request) ProtocolFormat {
	return FormatAnthropic
}

func (c *AnthropicConverter) ConvertRequest(ctx context.Context, r *http.Request, targetFormat ProtocolFormat) (*ProviderRequest, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	r.Body.Close()

	var anthropicReq AnthropicRequest
	if err := json.Unmarshal(body, &anthropicReq); err != nil {
		return nil, fmt.Errorf("failed to parse Anthropic request: %w", err)
	}

	switch targetFormat {
	case FormatAnthropic:
		return &ProviderRequest{
			Method: r.Method,
			URL:    r.URL.String(),
			Headers: map[string]string{
				"Content-Type":      "application/json",
				"anthropic-version": "2023-06-01",
			},
			Body: anthropicReq,
		}, nil

	case FormatOpenAI:
		messages := make([]OpenAIMessage, 0, len(anthropicReq.Messages))
		for _, m := range anthropicReq.Messages {
			messages = append(messages, OpenAIMessage{Role: m.Role, Content: m.Content})
		}
		if anthropicReq.System != "" {
			messages = append([]OpenAIMessage{{Role: "system", Content: anthropicReq.System}}, messages...)
		}
		openAIReq := OpenAIChatRequest{
			Model:       anthropicReq.Model,
			Messages:    messages,
			MaxTokens:   anthropicReq.MaxTokens,
			Stream:      anthropicReq.Stream,
			Temperature: anthropicReq.Temperature,
			TopP:        anthropicReq.TopP,
		}
		return &ProviderRequest{
			Method: r.Method,
			URL:    "/v1/chat/completions",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: openAIReq,
		}, nil

	case FormatGemini:
		contents := make([]GeminiContent, 0, len(anthropicReq.Messages))
		for _, m := range anthropicReq.Messages {
			role := m.Role
			if role == "assistant" {
				role = "model"
			}
			contents = append(contents, GeminiContent{
				Parts: []GeminiPart{{Text: m.Content}},
				Role:  role,
			})
		}
		geminiReq := GeminiRequest{
			Contents: contents,
			GenerationConfig: GeminiGenConfig{
				MaxOutputTokens: anthropicReq.MaxTokens,
				Temperature:     anthropicReq.Temperature,
				TopP:            anthropicReq.TopP,
			},
		}
		if anthropicReq.System != "" {
			geminiReq.SystemInstruction = &GeminiContent{
				Parts: []GeminiPart{{Text: anthropicReq.System}},
			}
			logger.L.Debug("converting Anthropic system prompt to Gemini systemInstruction",
				zap.Int("content_len", len(anthropicReq.System)))
		}
		return &ProviderRequest{
			Method: r.Method,
			URL:    "/v1/models/" + anthropicReq.Model + ":generateContent",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: geminiReq,
		}, nil

	default:
		return nil, fmt.Errorf("conversion from %s to %s not implemented", FormatAnthropic, targetFormat)
	}
}

func (c *AnthropicConverter) ConvertResponse(ctx context.Context, resp *ProviderResponse, sourceFormat ProtocolFormat) (*http.Response, error) {
	switch sourceFormat {
	case FormatAnthropic:
		body, err := json.Marshal(resp.Body)
		if err != nil {
			return nil, err
		}
		httpResp := &http.Response{
			StatusCode: resp.StatusCode,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(body)),
		}
		httpResp.Header.Set("Content-Type", "application/json")
		for k, v := range resp.Headers {
			httpResp.Header.Set(k, v)
		}
		return httpResp, nil

	case FormatOpenAI:
		var openAIResp OpenAIResponse
		data, err := json.Marshal(resp.Body)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &openAIResp); err != nil {
			return nil, err
		}
		anthropicResp := AnthropicResponse{
			ID:   "msg_" + utils.GenerateID(),
			Type: "message",
			Role: "assistant",
			Content: []AnthropicContent{{
				Type: "text",
				Text: openAIResp.Choices[0].Message.Content,
			}},
			Usage: AnthropicUsage{
				InputTokens:  openAIResp.Usage.PromptTokens,
				OutputTokens: openAIResp.Usage.CompletionTokens,
			},
		}
		out, _ := json.Marshal(anthropicResp)
		httpResp := &http.Response{
			StatusCode: resp.StatusCode,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(out)),
		}
		httpResp.Header.Set("Content-Type", "application/json")
		return httpResp, nil

	case FormatGemini:
		var geminiResp GeminiResponse
		data, err := json.Marshal(resp.Body)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &geminiResp); err != nil {
			return nil, err
		}
		anthropicResp := AnthropicResponse{
			ID:   "msg_" + utils.GenerateID(),
			Type: "message",
			Role: "assistant",
		}
		for _, cand := range geminiResp.Candidates {
			for _, part := range cand.Content.Parts {
				anthropicResp.Content = append(anthropicResp.Content, AnthropicContent{
					Type: "text",
					Text: part.Text,
				})
			}
		}
		anthropicResp.Usage = AnthropicUsage{
			InputTokens:  geminiResp.UsageMetadata.PromptTokenCount,
			OutputTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
		}
		out, _ := json.Marshal(anthropicResp)
		httpResp := &http.Response{
			StatusCode: resp.StatusCode,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(out)),
		}
		httpResp.Header.Set("Content-Type", "application/json")
		return httpResp, nil

	default:
		return nil, fmt.Errorf("conversion from %s to %s not implemented", sourceFormat, FormatAnthropic)
	}
}
