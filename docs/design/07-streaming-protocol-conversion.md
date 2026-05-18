# 流式协议转换设计

## 1. 概述

流式协议转换是AI聚合网关最复杂的部分，需要将不同服务商的流式响应格式实时转换为客户端期望的格式。

### 1.1 支持的流式格式

| 格式 | Content-Type | 分隔符 | 数据格式 |
|------|--------------|--------|----------|
| OpenAI SSE | text/event-stream | `data: ` | JSON |
| Anthropic SSE | text/event-stream | `data: ` | JSON |
| Gemini SSE | text/event-stream | `data: ` | JSON |

### 1.2 转换场景

```
场景1: OpenAI客户端 → OpenAI服务商（无需转换）
场景2: OpenAI客户端 → Anthropic服务商（需要转换）
场景3: Anthropic客户端 → OpenAI服务商（需要转换）
场景4: Gemini客户端 → OpenAI服务商（需要转换）
```

## 2. 流式响应格式对比

### 2.1 OpenAI流式格式

```http
HTTP/1.1 200 OK
Content-Type: text/event-stream

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-4","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]
```

### 2.2 Anthropic流式格式

```http
HTTP/1.1 200 OK
Content-Type: text/event-stream

event: message_start
data: {"type":"message_start","message":{"id":"msg_123","type":"message","role":"assistant","content":[]}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"!"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":5}}

event: message_stop
data: {"type":"message_stop"}
```

### 2.3 Gemini流式格式

```http
HTTP/1.1 200 OK
Content-Type: text/event-stream

data: {"candidates":[{"content":{"parts":[{"text":"Hello"}],"role":"model"},"finishReason":null,"index":0}]}

data: {"candidates":[{"content":{"parts":[{"text":"!"}],"role":"model"},"finishReason":"STOP","index":0}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5,"totalTokenCount":15}}
```

## 3. 流式转换器接口

```go
type StreamConverter interface {
    // 转换流式响应
    ConvertStream(
        ctx context.Context,
        sourceFormat ProtocolFormat,
        targetFormat ProtocolFormat,
        sourceStream <-chan StreamChunk,
        targetWriter *SSEHandler,
    ) error
}

type StreamChunk struct {
    Data      []byte
    Event     string // Anthropic使用event类型
    Error     error
    Done      bool
}
```

## 4. OpenAI → Anthropic 转换

### 4.1 转换逻辑

```go
type OpenAIToAnthropicStreamConverter struct{}

func (c *OpenAIToAnthropicStreamConverter) ConvertStream(
    ctx context.Context,
    sourceStream <-chan StreamChunk,
    targetWriter *SSEHandler,
) error {
    // 发送message_start
    messageStart := map[string]interface{}{
        "type": "message_start",
        "message": map[string]interface{}{
            "id":   "msg_" + generateID(),
            "type": "message",
            "role": "assistant",
            "content": []interface{}{},
        },
    }
    targetWriter.WriteEvent("message_start", messageStart)

    // 发送content_block_start
    contentBlockStart := map[string]interface{}{
        "type": "content_block_start",
        "index": 0,
        "content_block": map[string]interface{}{
            "type": "text",
            "text": "",
        },
    }
    targetWriter.WriteEvent("content_block_start", contentBlockStart)

    // 转换每个chunk
    for chunk := range sourceStream {
        if chunk.Error != nil {
            return chunk.Error
        }

        if chunk.Done {
            break
        }

        // 解析OpenAI格式
        var openAIChunk OpenAIStreamChunk
        if err := json.Unmarshal(chunk.Data, &openAIChunk); err != nil {
            continue
        }

        // 提取content
        if len(openAIChunk.Choices) > 0 {
            choice := openAIChunk.Choices[0]
            if choice.Delta.Content != "" {
                // 发送content_block_delta
                delta := map[string]interface{}{
                    "type": "content_block_delta",
                    "index": 0,
                    "delta": map[string]interface{}{
                        "type": "text_delta",
                        "text": choice.Delta.Content,
                    },
                }
                targetWriter.WriteEvent("content_block_delta", delta)
            }

            // 检查是否结束
            if choice.FinishReason != "" {
                // 发送content_block_stop
                targetWriter.WriteEvent("content_block_stop", map[string]interface{}{
                    "type": "content_block_stop",
                    "index": 0,
                })

                // 发送message_delta
                targetWriter.WriteEvent("message_delta", map[string]interface{}{
                    "type": "message_delta",
                    "delta": map[string]interface{}{
                        "stop_reason": mapFinishReason(choice.FinishReason),
                    },
                    "usage": map[string]interface{}{
                        "output_tokens": openAIChunk.Usage.CompletionTokens,
                    },
                })

                // 发送message_stop
                targetWriter.WriteEvent("message_stop", map[string]interface{}{
                    "type": "message_stop",
                })
            }
        }
    }

    return nil
}

func mapFinishReason(reason string) string {
    switch reason {
    case "stop":
        return "end_turn"
    case "length":
        return "max_tokens"
    default:
        return "end_turn"
    }
}
```

## 5. Anthropic → OpenAI 转换

### 5.1 转换逻辑

```go
type AnthropicToOpenAIStreamConverter struct{}

func (c *AnthropicToOpenAIStreamConverter) ConvertStream(
    ctx context.Context,
    sourceStream <-chan StreamChunk,
    targetWriter *SSEHandler,
) error {
    var contentStarted bool
    var chunkID = "chatcmpl-" + generateID()
    var created = time.Now().Unix()

    for chunk := range sourceStream {
        if chunk.Error != nil {
            return chunk.Error
        }

        if chunk.Done {
            break
        }

        // 解析Anthropic格式
        var event map[string]interface{}
        if err := json.Unmarshal(chunk.Data, &event); err != nil {
            continue
        }

        eventType, _ := event["type"].(string)

        switch eventType {
        case "message_start":
            // 发送初始chunk（包含role）
            openAIChunk := map[string]interface{}{
                "id":      chunkID,
                "object":  "chat.completion.chunk",
                "created": created,
                "model":   "gpt-4", // 从原始请求获取
                "choices": []map[string]interface{}{
                    {
                        "index": 0,
                        "delta": map[string]interface{}{
                            "role": "assistant",
                        },
                        "finish_reason": nil,
                    },
                },
            }
            targetWriter.WriteData(openAIChunk)
            contentStarted = true

        case "content_block_delta":
            if !contentStarted {
                continue
            }

            delta, _ := event["delta"].(map[string]interface{})
            text, _ := delta["text"].(string)

            if text != "" {
                openAIChunk := map[string]interface{}{
                    "id":      chunkID,
                    "object":  "chat.completion.chunk",
                    "created": created,
                    "model":   "gpt-4",
                    "choices": []map[string]interface{}{
                        {
                            "index": 0,
                            "delta": map[string]interface{}{
                                "content": text,
                            },
                            "finish_reason": nil,
                        },
                    },
                }
                targetWriter.WriteData(openAIChunk)
            }

        case "message_delta":
            delta, _ := event["delta"].(map[string]interface{})
            stopReason, _ := delta["stop_reason"].(string)

            openAIChunk := map[string]interface{}{
                "id":      chunkID,
                "object":  "chat.completion.chunk",
                "created": created,
                "model":   "gpt-4",
                "choices": []map[string]interface{}{
                    {
                        "index": 0,
                        "delta": map[string]interface{}{},
                        "finish_reason": mapStopReason(stopReason),
                    },
                },
            }
            targetWriter.WriteData(openAIChunk)

        case "message_stop":
            targetWriter.WriteData("[DONE]")
        }
    }

    return nil
}

func mapStopReason(reason string) string {
    switch reason {
    case "end_turn":
        return "stop"
    case "max_tokens":
        return "length"
    default:
        return "stop"
    }
}
```

## 6. 通用流式处理器

### 6.1 流式处理器接口

```go
type StreamHandler struct {
    writer      http.ResponseWriter
    flusher     http.Flusher
    done        chan struct{}
    mu          sync.Mutex
    closed      bool
}

func NewStreamHandler(w http.ResponseWriter) (*StreamHandler, error) {
    flusher, ok := w.(http.Flusher)
    if !ok {
        return nil, errors.New("streaming not supported")
    }

    // 设置SSE Headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no")

    return &StreamHandler{
        writer:  w,
        flusher: flusher,
        done:    make(chan struct{}),
    }, nil
}

// WriteData 写入SSE数据
func (h *StreamHandler) WriteData(data interface{}) error {
    h.mu.Lock()
    defer h.mu.Unlock()

    if h.closed {
        return errors.New("handler closed")
    }

    jsonBytes, err := json.Marshal(data)
    if err != nil {
        return err
    }

    _, err = fmt.Fprintf(h.writer, "data: %s\n\n", jsonBytes)
    if err != nil {
        return err
    }

    h.flusher.Flush()
    return nil
}

// WriteEvent 写入SSE事件（Anthropic格式）
func (h *StreamHandler) WriteEvent(event string, data interface{}) error {
    h.mu.Lock()
    defer h.mu.Unlock()

    if h.closed {
        return errors.New("handler closed")
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

// Close 关闭流
func (h *StreamHandler) Close() {
    h.mu.Lock()
    defer h.mu.Unlock()

    if !h.closed {
        h.closed = true
        close(h.done)
    }
}
```

## 7. 错误处理

### 7.1 流式错误处理

```go
func (h *StreamHandler) WriteError(err error) {
    h.mu.Lock()
    defer h.mu.Unlock()

    if h.closed {
        return
    }

    // 发送错误事件
    errorData := map[string]interface{}{
        "error": map[string]interface{}{
            "message": err.Error(),
            "type":    "server_error",
        },
    }

    jsonBytes, _ := json.Marshal(errorData)
    fmt.Fprintf(h.writer, "data: %s\n\n", jsonBytes)
    h.flusher.Flush()
}
```

### 7.2 超时处理

```go
func (g *Gateway) handleStreamWithTimeout(
    ctx context.Context,
    req *ChatRequest,
    w http.ResponseWriter,
    timeout time.Duration,
) {
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    // 监听超时
    go func() {
        <-ctx.Done()
        if ctx.Err() == context.DeadlineExceeded {
            // 发送超时错误
            streamHandler.WriteError(errors.New("request timeout"))
            streamHandler.Close()
        }
    }()

    // 正常处理流式响应
    g.handleStream(ctx, req, streamHandler)
}
```

## 8. 性能优化

### 8.1 缓冲区优化

```go
type BufferedStreamHandler struct {
    handler    *StreamHandler
    buffer     []byte
    bufferSize int
    mu         sync.Mutex
}

func NewBufferedStreamHandler(handler *StreamHandler, bufferSize int) *BufferedStreamHandler {
    return &BufferedStreamHandler{
        handler:    handler,
        buffer:     make([]byte, 0, bufferSize),
        bufferSize: bufferSize,
    }
}

func (h *BufferedStreamHandler) Write(data []byte) (int, error) {
    h.mu.Lock()
    defer h.mu.Unlock()

    h.buffer = append(h.buffer, data...)

    // 缓冲区满时刷新
    if len(h.buffer) >= h.bufferSize {
        return h.flush()
    }

    return len(data), nil
}

func (h *BufferedStreamHandler) flush() (int, error) {
    n, err := h.handler.writer.Write(h.buffer)
    h.buffer = h.buffer[:0]
    h.handler.flusher.Flush()
    return n, err
}
```

### 8.2 并发安全

```go
// 使用channel实现生产者-消费者模式
type StreamPipeline struct {
    input   chan StreamChunk
    output  chan StreamChunk
    done    chan struct{}
}

func NewStreamPipeline(bufferSize int) *StreamPipeline {
    return &StreamPipeline{
        input:  make(chan StreamChunk, bufferSize),
        output: make(chan StreamChunk, bufferSize),
        done:   make(chan struct{}),
    }
}

func (p *StreamPipeline) Start(converter StreamConverter) {
    go func() {
        defer close(p.output)
        converter.ConvertStream(context.Background(), p.input, nil)
    }()
}

func (p *StreamPipeline) Write(chunk StreamChunk) {
    p.input <- chunk
}

func (p *StreamPipeline) Read() <-chan StreamChunk {
    return p.output
}
```
