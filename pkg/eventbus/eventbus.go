package eventbus

import "sync"

type EventBus struct {
	subs map[string][]chan any
	mu   sync.RWMutex
}

func New() *EventBus {
	return &EventBus{subs: make(map[string][]chan any)}
}

func (eb *EventBus) Subscribe(topic string) chan any {
	ch := make(chan any, 100)
	eb.mu.Lock()
	eb.subs[topic] = append(eb.subs[topic], ch)
	eb.mu.Unlock()
	return ch
}

func (eb *EventBus) Unsubscribe(topic string, ch chan any) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	subs := eb.subs[topic]
	for i, s := range subs {
		if s == ch {
			eb.subs[topic] = append(subs[:i], subs[i+1:]...)
			close(ch)
			return
		}
	}
}

func (eb *EventBus) Publish(topic string, data any) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	for _, ch := range eb.subs[topic] {
		select {
		case ch <- data:
		default:
		}
	}
}
