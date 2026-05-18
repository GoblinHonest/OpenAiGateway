package event

import (
	"sync"
	"time"
)

type EventType string

const (
	EventCircuitBreakerChange EventType = "circuit_breaker_change"
	EventProviderHealthChange EventType = "provider_health_change"
	EventTokenExhausted       EventType = "token_exhausted"
	EventQuotaExceeded        EventType = "quota_exceeded"
)

type Event struct {
	Type      EventType
	Payload   any
	Timestamp time.Time
}

type EventBus struct {
	subscribers map[EventType][]chan Event
	mu          sync.RWMutex
	bufferSize  int
}

func NewEventBus(bufferSize int) *EventBus {
	return &EventBus{
		subscribers: make(map[EventType][]chan Event),
		bufferSize:  bufferSize,
	}
}

func (b *EventBus) Subscribe(eventType EventType, bufferSize ...int) chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	bs := b.bufferSize
	if len(bufferSize) > 0 {
		bs = bufferSize[0]
	}

	ch := make(chan Event, bs)
	b.subscribers[eventType] = append(b.subscribers[eventType], ch)
	return ch
}

func (b *EventBus) Unsubscribe(eventType EventType, ch chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subs := b.subscribers[eventType]
	for i, sub := range subs {
		if sub == ch {
			b.subscribers[eventType] = append(subs[:i], subs[i+1:]...)
			close(ch)
			return
		}
	}
}

func (b *EventBus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.subscribers[event.Type] {
		select {
		case ch <- event:
		default:
		}
	}
}
