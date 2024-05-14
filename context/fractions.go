package context

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

const errCauseFormat = "fraction %d has expired"

type withFractionCtx struct {
	context.Context
	sentinel       context.Context
	cancelSentinel context.CancelFunc

	cancelc   chan context.CancelFunc
	deadlines []time.Time
	current   atomic.Int64
}

func (c *withFractionCtx) Done() <-chan struct{} { return c.sentinel.Done() }

func (c *withFractionCtx) Next() context.Context {
	select {
	case <-c.sentinel.Done():
		// already cancelled
		return c.sentinel
	default:
	}
	select {
	case cancel := <-c.cancelc:
		// cancel the previous context if any
		cancel()
	default:
	}

	v := c.current.Load()
	if v >= int64(len(c.deadlines)) {
		// already comes to the end, so just return the sentinel
		return c.sentinel
	}
	ctx, cancel := context.WithDeadlineCause(c.Context, c.deadlines[v], fmt.Errorf(errCauseFormat, v))
	c.cancelc <- cancel
	c.current.Add(1)

	go func() {
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			c.cancelSentinel()
		}
	}()
	return ctx
}

func WithFractions(parent context.Context, fractions ...float64) (ChainingContext, context.CancelFunc) {
	deadline, ok := parent.Deadline()
	if !ok || len(fractions) == 0 || fractions[0] >= 1.0 {
		return &wrapper{parent}, func() {}
	}
	sentinel, cancel := context.WithCancel(parent)

	c := &withFractionCtx{
		Context:        parent,
		sentinel:       sentinel,
		cancelSentinel: cancel,
		cancelc:        make(chan context.CancelFunc, 1),
		deadlines:      fractionsToDeadlines(time.Now(), deadline, fractions),
	}

	return c, cancel
}

func fractionsToDeadlines(now, deadline time.Time, fractions []float64) []time.Time {
	left := deadline.Sub(now)
	if left < 0 {
		return nil
	}
	deadlines := make([]time.Time, 0, len(fractions))
	total := 0.0
	for _, f := range fractions {
		total += f
		deadlines = append(deadlines, now.Add(time.Duration(float64(left)*total)))
		if total >= 1.0 {
			break
		}
	}
	return deadlines
}
