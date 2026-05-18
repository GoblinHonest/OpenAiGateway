package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type StreamChunk struct {
	Data  []byte
	Event string
	Error error
	Done  bool
}

type StreamReader struct {
	reader  *bufio.Reader
	done    chan struct{}
	mu      sync.Mutex
	closed  bool
}

func NewStreamReader(reader io.Reader) *StreamReader {
	return &StreamReader{
		reader: bufio.NewReader(reader),
		done:   make(chan struct{}),
	}
}

func (sr *StreamReader) Stream(ctx context.Context) <-chan StreamChunk {
	ch := make(chan StreamChunk, 100)

	go func() {
		defer close(ch)
		defer close(sr.done)

		for {
			select {
			case <-ctx.Done():
				ch <- StreamChunk{Error: ctx.Err(), Done: true}
				return
			default:
			}

			line, err := sr.reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					ch <- StreamChunk{Done: true}
				} else {
					ch <- StreamChunk{Error: err, Done: true}
				}
				return
			}

			line = strings.TrimRight(line, "\r\n")

			if line == "" {
				continue
			}

			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					ch <- StreamChunk{Done: true}
					return
				}
				ch <- StreamChunk{Data: []byte(data)}
			} else if strings.HasPrefix(line, "event: ") {
				event := strings.TrimPrefix(line, "event: ")
				ch <- StreamChunk{Event: event}
			}
		}
	}()

	return ch
}

func (sr *StreamReader) Close() {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.closed = true
}

type SSEHandler struct {
	writer  io.Writer
	flusher http.Flusher
	done    chan struct{}
	mu      sync.Mutex
	closed  bool
}

func NewSSEHandler(w http.ResponseWriter) (*SSEHandler, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	return &SSEHandler{
		writer:  w,
		flusher: flusher,
		done:    make(chan struct{}),
	}, nil
}

func (h *SSEHandler) WriteData(data []byte) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return fmt.Errorf("handler closed")
	}

	_, err := fmt.Fprintf(h.writer, "data: %s\n\n", data)
	if err != nil {
		return err
	}

	h.flusher.Flush()
	return nil
}

func (h *SSEHandler) WriteJSON(data any) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return h.WriteData(jsonBytes)
}

func (h *SSEHandler) WriteEvent(event string, data any) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return fmt.Errorf("handler closed")
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(h.writer, "event: %s\ndata: %s\n\n", event, jsonBytes)
	if err != nil {
		return err
	}

	h.flusher.Flush()
	return nil
}

func (h *SSEHandler) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.closed {
		fmt.Fprintf(h.writer, "data: [DONE]\n\n")
		h.flusher.Flush()
		h.closed = true
		close(h.done)
	}
}

func (h *SSEHandler) WriteError(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return
	}

	errorData := map[string]any{
		"error": map[string]any{
			"message": err.Error(),
			"type":    "server_error",
		},
	}

	jsonBytes, _ := json.Marshal(errorData)
	fmt.Fprintf(h.writer, "data: %s\n\n", jsonBytes)
	h.flusher.Flush()
}

func (h *SSEHandler) MonitorClient(ctx context.Context) {
	go func() {
		select {
		case <-ctx.Done():
			h.Close()
		case <-h.done:
			return
		}
	}()

	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				h.mu.Lock()
				if !h.closed {
					_, err := fmt.Fprintf(h.writer, ": keepalive\n\n")
					if err != nil {
						h.mu.Unlock()
						h.Close()
						return
					}
					h.flusher.Flush()
				}
				h.mu.Unlock()
			case <-h.done:
				return
			}
		}
	}()
}
