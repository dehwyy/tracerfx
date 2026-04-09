package nats

import (
	"github.com/nats-io/nats.go"
)

// NATSHeaderCarrier adapts nats.Header to propagation.TextMapCarrier.
type NATSHeaderCarrier struct {
	Header nats.Header
}

func (c *NATSHeaderCarrier) Get(key string) string {
	return c.Header.Get(key)
}

func (c *NATSHeaderCarrier) Set(key string, value string) {
	c.Header.Set(key, value)
}

func (c *NATSHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c.Header))
	for k := range c.Header {
		keys = append(keys, k)
	}

	return keys
}
