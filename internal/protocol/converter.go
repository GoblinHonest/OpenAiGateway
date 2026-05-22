package protocol

import (
	"context"
	"net/http"
	"strings"
)

type ProtocolFormat string

const (
	FormatOpenAI    ProtocolFormat = "openai"
	FormatAnthropic ProtocolFormat = "anthropic"
	FormatGemini    ProtocolFormat = "gemini"
)

type ProviderRequest struct {
	Method       string
	URL          string
	Headers      map[string]string
	Body         any
	TargetFormat ProtocolFormat
}

type ProviderResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       any
}

type StreamChunk struct {
	Data      []byte
	Event     string
	Error     error
	Done      bool
}

type Converter interface {
	DetectFormat(r *http.Request) ProtocolFormat
	ConvertRequest(ctx context.Context, req *http.Request, targetFormat ProtocolFormat) (*ProviderRequest, error)
	ConvertResponse(ctx context.Context, resp *ProviderResponse, sourceFormat ProtocolFormat) (*http.Response, error)
}

type StreamConverter interface {
	ConvertStream(ctx context.Context, sourceStream <-chan StreamChunk, targetWriter StreamWriter) error
}

type StreamWriter interface {
	WriteData(data []byte) error
	WriteJSON(data any) error
	WriteEvent(event string, data any) error
	Close()
	WriteError(err error)
}

func DetectFormat(r *http.Request) ProtocolFormat {
	path := r.URL.Path
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

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
