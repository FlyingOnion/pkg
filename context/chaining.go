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

// type ChainingContext struct {
// 	parent    context.Context
// 	sentinel  context.Context
// 	deadlines []time.Time
// 	doneC     chan struct{}
// 	current   atomic.Int64
// }

// func (c *ChainingContext) Deadline() (deadline time.Time, ok bool) { return c.Current().Deadline() }
// func (c *ChainingContext) Done() <-chan struct{}                   { return c.Current().Done() }
// func (c *ChainingContext) Err() error                              { return c.Current().Err() }

// func (c *ChainingContext) Next() context.Context {
// 	c.current.Add(1)
// 	return c.Current()
// }

// // newChainingContext creates a new chaining context.
// // parent should be cancellable, and all children should be under its control.
// func NewChainingContext(parent context.Context) (*ChainingContext, context.CancelFunc) {
// 	return &ChainingContext{parent: parent}
// }
