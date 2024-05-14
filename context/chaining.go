package context

import (
	"context"
	"time"
)

type ChainingContext interface {
	context.Context
	Next() context.Context
}

type wrapper struct {
	context.Context
}

func (c *wrapper) Next() context.Context { return c.Context }

func (c *wrapper) Reset(time.Duration) {}
