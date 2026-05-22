package protocol

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/example/aigateway/pkg/logger"
	"github.com/example/aigateway/pkg/utils"
	"go.uber.org/zap"
)

type OpenAIChatRequest struct {
	Model       string            `json:"model"`
	Messages    []OpenAIMessage   `json:"messages"`
	Stream      bool              `json:"stream"`
	Temperature float64           `json:"temperature,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	TopP        float64           `json:"top_p,omitempty"`
	Tools       []OpenAITool      `json:"tools,omitempty"`
	ToolChoice  any               `json:"tool_choice,omitempty"`
	Metadata    map[string]any    `json:"metadata,omitempty"`
}

type OpenAIMessage struct {
	Role       string          `json:"role"`
	Content    string          `json:"content"`
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	Name       string          `json:"name,omitempty"`
}

type OpenAITool struct {
	Type     string         `json:"type"`
	Function OpenAIFunction `json:"function"`
}

type OpenAIFunction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

type OpenAIToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function OpenAIFunctionCall `json:"function"`
}

type OpenAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type OpenAIRequest struct {
	Request  OpenAIChatRequest
	Original *http.Request
}

type OpenAIResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   OpenAIUsage  `json:"usage,omitempty"`
}

type OpenAIChoice struct {
	Index        int            `json:"index"`
	Message      OpenAIMessage  `json:"message,omitempty"`
	Delta        OpenAIMessage  `json:"delta,omitempty"`
	FinishReason *string        `json:"finish_reason"`
}

type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type OpenAIConverter struct{}

func NewOpenAIConverter() *OpenAIConverter {
	return &OpenAIConverter{}
}

func (c *OpenAIConverter) DetectFormat(r *http.Request) ProtocolFormat {
	return FormatOpenAI
}

func (c *OpenAIConverter) ConvertRequest(ctx context.Context, r *http.Request, targetFormat ProtocolFormat) (*ProviderRequest, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	r.Body.Close()

	var openAIReq OpenAIChatRequest
	if err := json.Unmarshal(body, &openAIReq); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI request: %w", err)
	}

	// 透传客户端请求头
	clientHeaders := make(map[string]string)
	for k := range r.Header {
		clientHeaders[k] = r.Header.Get(k)
	}

	switch targetFormat {
	case FormatOpenAI:
		clientHeaders["Content-Type"] = "application/json"
		return &ProviderRequest{
			Method:       r.Method,
			URL:          r.URL.String(),
			Headers:      clientHeaders,
			Body:         openAIReq,
			TargetFormat: targetFormat,
		}, nil

	case FormatAnthropic:
		messages := make([]AnthropicMessage, 0, len(openAIReq.Messages))
		for _, m := range openAIReq.Messages {
			contentBytes, _ := json.Marshal(m.Content)
			messages = append(messages, AnthropicMessage{Role: m.Role, Content: contentBytes})
		}
		anthropicReq := AnthropicRequest{
			Model:       openAIReq.Model,
			Messages:    messages,
			MaxTokens:   openAIReq.MaxTokens,
			Stream:      openAIReq.Stream,
			Temperature: openAIReq.Temperature,
			TopP:        openAIReq.TopP,
		}
		clientHeaders["Content-Type"] = "application/json"
		clientHeaders["anthropic-version"] = "2023-06-01"
		return &ProviderRequest{
			Method:       r.Method,
			URL:          "/v1/messages",
			Headers:      clientHeaders,
			Body:         anthropicReq,
			TargetFormat: targetFormat,
		}, nil

	case FormatGemini:
		contents := make([]GeminiContent, 0, len(openAIReq.Messages))
		var systemInstruction *GeminiContent
		for _, m := range openAIReq.Messages {
			role := m.Role
			if role == "assistant" {
				role = "model"
			}
			if role == "system" {
				// Gemini supports systemInstruction as a top-level field
				systemInstruction = &GeminiContent{
					Parts: []GeminiPart{{Text: m.Content}},
				}
				logger.L.Debug("converting system message to Gemini systemInstruction",
					zap.Int("content_len", len(m.Content)))
				continue
			}
			contents = append(contents, GeminiContent{
				Parts: []GeminiPart{{Text: m.Content}},
				Role:  role,
			})
		}
		geminiReq := GeminiRequest{
			Contents:          contents,
			SystemInstruction: systemInstruction,
			GenerationConfig: GeminiGenConfig{
				MaxOutputTokens: openAIReq.MaxTokens,
				Temperature:     openAIReq.Temperature,
				TopP:            openAIReq.TopP,
			},
		}
		clientHeaders["Content-Type"] = "application/json"
		return &ProviderRequest{
			Method:       r.Method,
			URL:          "/v1/models/" + openAIReq.Model + ":generateContent",
			Headers:      clientHeaders,
			Body:         geminiReq,
			TargetFormat: targetFormat,
		}, nil

	default:
		return nil, fmt.Errorf("conversion from %s to %s not implemented", FormatOpenAI, targetFormat)
	}
}

func (c *OpenAIConverter) ConvertResponse(ctx context.Context, resp *ProviderResponse, sourceFormat ProtocolFormat) (*http.Response, error) {
	// 获取响应体的字节数据
	var bodyBytes []byte
	switch v := resp.Body.(type) {
	case []byte:
		bodyBytes = v
	case string:
		bodyBytes = []byte(v)
	default:
		var err error
		bodyBytes, err = json.Marshal(resp.Body)
		if err != nil {
			return nil, err
		}
	}

	switch sourceFormat {
	case FormatOpenAI:
		httpResp := &http.Response{
			StatusCode: resp.StatusCode,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
		}
		httpResp.Header.Set("Content-Type", "application/json")
		for k, v := range resp.Headers {
			httpResp.Header.Set(k, v)
		}
		return httpResp, nil

	case FormatAnthropic:
		var anthropicResp AnthropicResponse
		if err := json.Unmarshal(bodyBytes, &anthropicResp); err != nil {
			return nil, err
		}
		openAIResp := OpenAIResponse{
			ID:      anthropicResp.ID,
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4",
			Choices: make([]OpenAIChoice, len(anthropicResp.Content)),
			Usage: OpenAIUsage{
				PromptTokens:     anthropicResp.Usage.InputTokens,
				CompletionTokens: anthropicResp.Usage.OutputTokens,
				TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
			},
		}
		for i, c := range anthropicResp.Content {
			finishReason := "stop"
			if anthropicResp.StopReason != "" {
				finishReason = anthropicResp.StopReason
			}
			openAIResp.Choices[i] = OpenAIChoice{
				Index: i,
				Message: OpenAIMessage{
					Role:    "assistant",
					Content: c.Text,
				},
				FinishReason: &finishReason,
			}
		}
		out, _ := json.Marshal(openAIResp)
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
		openAIResp := OpenAIResponse{
			ID:      "chatcmpl-" + utils.GenerateID(),
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4",
			Choices: make([]OpenAIChoice, 0, len(geminiResp.Candidates)),
			Usage: OpenAIUsage{
				PromptTokens:     geminiResp.UsageMetadata.PromptTokenCount,
				CompletionTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
				TotalTokens:      geminiResp.UsageMetadata.TotalTokenCount,
			},
		}
		for _, cand := range geminiResp.Candidates {
			content := ""
			for _, part := range cand.Content.Parts {
				content += part.Text
			}
			finishReason := cand.FinishReason
			openAIResp.Choices = append(openAIResp.Choices, OpenAIChoice{
				Index: cand.Index,
				Message: OpenAIMessage{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: &finishReason,
			})
		}
		out, _ := json.Marshal(openAIResp)
		httpResp := &http.Response{
			StatusCode: resp.StatusCode,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(out)),
		}
		httpResp.Header.Set("Content-Type", "application/json")
		return httpResp, nil

	default:
		return nil, fmt.Errorf("conversion from %s to %s not implemented", sourceFormat, FormatOpenAI)
	}
}
