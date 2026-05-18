package resilience

import (
	"bytes"
	"io"
	"net/http"
)

type BufferedRequest struct {
	Original *http.Request
	Body     []byte
}

func NewBufferedRequest(r *http.Request) (*BufferedRequest, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	r.Body.Close()

	return &BufferedRequest{
		Original: r,
		Body:     body,
	}, nil
}

func (br *BufferedRequest) Clone() *http.Request {
	clone := br.Original.Clone(br.Original.Context())
	clone.Body = io.NopCloser(bytes.NewReader(br.Body))
	return clone
}

func (br *BufferedRequest) CloneWithContext(ctx *http.Request) *http.Request {
	clone := br.Original.Clone(ctx.Context())
	clone.Body = io.NopCloser(bytes.NewReader(br.Body))
	return clone
}
