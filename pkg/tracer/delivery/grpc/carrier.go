package grpc

import (
	"google.golang.org/grpc/metadata"
)

// MetadataCarrier adapts gRPC metadata.MD to propagation.TextMapCarrier.
type MetadataCarrier struct {
	MD *metadata.MD
}

func (c *MetadataCarrier) Get(key string) string {
	values := c.MD.Get(key)
	if len(values) == 0 {
		return ""
	}

	return values[0]
}

func (c *MetadataCarrier) Set(key string, value string) {
	c.MD.Set(key, value)
}

func (c *MetadataCarrier) Keys() []string {
	keys := make([]string, 0, len(*c.MD))
	for k := range *c.MD {
		keys = append(keys, k)
	}

	return keys
}
