package context

import (
	"context"
	"time"
)

type withParents struct {
	d *time.Time
	context.Context
}

func (w *withParents) Deadline() (deadline time.Time, ok bool) {
	if w.d == nil {
		return
	}
	return *w.d, true
}

func earliestDeadline(ctxes ...context.Context) (earliest *time.Time) {
	for _, c := range ctxes {
		d, ok := c.Deadline()
		if !ok {
			continue
		}
		if earliest == nil || d.Before(*earliest) {
			earliest = &d
		}
	}
	return
}

func WithParents(parents ...context.Context) (context.Context, context.CancelFunc) {
	if len(parents) == 0 {
		return context.WithCancel(context.Background())
	}
	if len(parents) == 1 {
		return context.WithCancel(parents[0])
	}
	child, cancel := context.WithCancel(context.Background())
	ctx := &withParents{
		d:       earliestDeadline(parents...),
		Context: child,
	}

	for _, p := range parents {
		go func(c context.Context) {
			select {
			case <-c.Done():
				cancel()
			case <-ctx.Done():
				// cancelled by another parent
			}
		}(p)
	}
	return ctx, cancel
}
