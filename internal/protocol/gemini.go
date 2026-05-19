package protocol

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type GeminiRequest struct {
	Contents          []GeminiContent        `json:"contents"`
	SystemInstruction *GeminiContent         `json:"systemInstruction,omitempty"`
	GenerationConfig  GeminiGenConfig        `json:"generationConfig,omitempty"`
	SafetySettings    []GeminiSafetySetting  `json:"safetySettings,omitempty"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type GeminiPart struct {
	Text string `json:"text,omitempty"`
}

type GeminiGenConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	Temperature     float64 `json:"temperature,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
	TopK            int     `json:"topK,omitempty"`
}

type GeminiSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

type GeminiCandidate struct {
	Content       GeminiContent  `json:"content"`
	FinishReason  string         `json:"finishReason"`
	Index         int            `json:"index"`
}

type GeminiResponse struct {
	Candidates    []GeminiCandidate `json:"candidates"`
	UsageMetadata GeminiUsage       `json:"usageMetadata,omitempty"`
}

type GeminiUsage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

type GeminiConverter struct{}

func NewGeminiConverter() *GeminiConverter {
	return &GeminiConverter{}
}

func (c *GeminiConverter) DetectFormat(r *http.Request) ProtocolFormat {
	return FormatGemini
}

func (c *GeminiConverter) ConvertRequest(ctx context.Context, r *http.Request, targetFormat ProtocolFormat) (*ProviderRequest, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	r.Body.Close()

	var geminiReq GeminiRequest
	if err := json.Unmarshal(body, &geminiReq); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini request: %w", err)
	}

	switch targetFormat {
	case FormatGemini:
		return &ProviderRequest{
			Method: r.Method,
			URL:    r.URL.String(),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: geminiReq,
		}, nil

	case FormatOpenAI:
		messages := make([]OpenAIMessage, 0, len(geminiReq.Contents))
		for _, c := range geminiReq.Contents {
			content := ""
			for _, p := range c.Parts {
				content += p.Text
			}
			role := c.Role
			if role == "model" {
				role = "assistant"
			}
			if role == "" {
				role = "user"
			}
			messages = append(messages, OpenAIMessage{Role: role, Content: content})
		}
		openAIReq := OpenAIChatRequest{
			Messages: messages,
			MaxTokens: geminiReq.GenerationConfig.MaxOutputTokens,
			Temperature: geminiReq.GenerationConfig.Temperature,
			TopP:    geminiReq.GenerationConfig.TopP,
		}
		return &ProviderRequest{
			Method: r.Method,
			URL:    "/v1/chat/completions",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: openAIReq,
		}, nil

	case FormatAnthropic:
		messages := make([]AnthropicMessage, 0, len(geminiReq.Contents))
		for _, c := range geminiReq.Contents {
			content := ""
			for _, p := range c.Parts {
				content += p.Text
			}
			role := c.Role
			if role == "model" {
				role = "assistant"
			}
			if role == "" {
				role = "user"
			}
			contentBytes, _ := json.Marshal(content)
			messages = append(messages, AnthropicMessage{Role: role, Content: contentBytes})
		}
		anthropicReq := AnthropicRequest{
			Messages: messages,
			MaxTokens: geminiReq.GenerationConfig.MaxOutputTokens,
			Temperature: geminiReq.GenerationConfig.Temperature,
			TopP:    geminiReq.GenerationConfig.TopP,
		}
		return &ProviderRequest{
			Method: r.Method,
			URL:    "/v1/messages",
			Headers: map[string]string{
				"Content-Type":      "application/json",
				"anthropic-version": "2023-06-01",
			},
			Body: anthropicReq,
		}, nil

	default:
		return nil, fmt.Errorf("conversion from %s to %s not implemented", FormatGemini, targetFormat)
	}
}

func (c *GeminiConverter) ConvertResponse(ctx context.Context, resp *ProviderResponse, sourceFormat ProtocolFormat) (*http.Response, error) {
	switch sourceFormat {
	case FormatGemini:
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
		geminiResp := GeminiResponse{
			Candidates: make([]GeminiCandidate, 0, len(openAIResp.Choices)),
			UsageMetadata: GeminiUsage{
				PromptTokenCount:     openAIResp.Usage.PromptTokens,
				CandidatesTokenCount: openAIResp.Usage.CompletionTokens,
				TotalTokenCount:      openAIResp.Usage.TotalTokens,
			},
		}
		for _, choice := range openAIResp.Choices {
			geminiResp.Candidates = append(geminiResp.Candidates, GeminiCandidate{
				Content: GeminiContent{
					Parts: []GeminiPart{{Text: choice.Message.Content}},
					Role:  "model",
				},
				FinishReason: *choice.FinishReason,
				Index:        choice.Index,
			})
		}
		out, _ := json.Marshal(geminiResp)
		httpResp := &http.Response{
			StatusCode: resp.StatusCode,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(out)),
		}
		httpResp.Header.Set("Content-Type", "application/json")
		return httpResp, nil

	case FormatAnthropic:
		var anthropicResp AnthropicResponse
		data, err := json.Marshal(resp.Body)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &anthropicResp); err != nil {
			return nil, err
		}
		geminiResp := GeminiResponse{
			Candidates: make([]GeminiCandidate, 0, len(anthropicResp.Content)),
			UsageMetadata: GeminiUsage{
				PromptTokenCount:     anthropicResp.Usage.InputTokens,
				CandidatesTokenCount: anthropicResp.Usage.OutputTokens,
				TotalTokenCount:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
			},
		}
		for i, c := range anthropicResp.Content {
			geminiResp.Candidates = append(geminiResp.Candidates, GeminiCandidate{
				Content: GeminiContent{
					Parts: []GeminiPart{{Text: c.Text}},
					Role:  "model",
				},
				Index: i,
			})
		}
		out, _ := json.Marshal(geminiResp)
		httpResp := &http.Response{
			StatusCode: resp.StatusCode,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(out)),
		}
		httpResp.Header.Set("Content-Type", "application/json")
		return httpResp, nil

	default:
		return nil, fmt.Errorf("conversion from %s to %s not implemented", sourceFormat, FormatGemini)
	}
}
