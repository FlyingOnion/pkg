package context

import (
	"context"
	"time"
)

type ResetContext interface {
	context.Context
	// Reset resets the deadline with the given duration.
	// If the deadline is already passed, the context will be canceled and Reset will do nothing.
	// If you use timer in your context, stop it and drain the channel before calling *timer.Reset(d).
	Reset(d time.Duration)
}

type resetCtx struct {
	t   *time.Timer
	ctx context.Context
}

// Deadline is unsure because the timer could be reset, so we assume it has no deadline.
func (r *resetCtx) Deadline() (deadline time.Time, ok bool) { return }

func (r *resetCtx) Done() <-chan struct{} { return r.ctx.Done() }

func (r *resetCtx) Err() error { return r.ctx.Err() }

func (r *resetCtx) Value(key any) any { return nil }

func (r *resetCtx) Reset(d time.Duration) {
	if r.t == nil {
		return
	}
	r.t.Stop()
	select {
	case <-r.t.C:
		// deadline has already passed
		return
	default:
	}
	r.t.Reset(d)
}

// WithReset creates a resettable context.
func WithReset(d time.Duration) (ResetContext, context.CancelFunc) {
	if d <= 0 {
		return &wrapper{context.Background()}, func() {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	t := time.NewTimer(d)
	go func() {
		select {
		case <-t.C:
			cancel()
		case <-ctx.Done():
		}
	}()
	return &resetCtx{t, ctx}, func() {
		t.Stop()
		cancel()
	}
}
