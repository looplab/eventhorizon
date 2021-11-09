package mongodb

import (
	"context"
	"time"
)

// GracefulContext is a context that can be gracefully cancelled after the original
// context has been canceled, but with a grace period to let operations finish.
// The KeepAlive method should be used to signal that  operations are still ongoing.
type GracefulContext struct {
	context.Context

	doneCh chan struct{}
}

// NewGracefulContext will create a GracefulContext with an idle time which should be
// longer than a typical operation cadence, and a "force after" duration which
// should be long enough to let some amount of operations to finish.
func NewGracefulContext(ctx context.Context, idle, forceAfter time.Duration) (
	c context.Context,
	keepAlive func(),
	cancel func(),
) {
	keepAliveCh := make(chan struct{})
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})

	go func() {
		defer close(doneCh)

		select {
		case <-ctx.Done():
			// Will trigger the graceful shutdown.
		case <-stopCh:
			// Will skip graceful shutdown.
			return
		}

		idleT := time.NewTimer(idle)
		forceShutdown := time.After(forceAfter)

		for {
			select {
			case <-keepAliveCh: // Reset the idle timer on activity.
				if !idleT.Stop() {
					<-idleT.C
				}

				idleT.Reset(idle)

				continue
			case <-idleT.C: // If there is no more activity, cancel.
				return
			case <-forceShutdown: // If there is too much activity, cancel anyway.
				return
			}
		}
	}()

	c = &GracefulContext{
		Context: ctx,
		doneCh:  doneCh,
	}

	keepAlive = func() {
		select {
		case keepAliveCh <- struct{}{}:
		default:
		}
	}

	cancel = func() {
		close(stopCh)
	}

	return c, keepAlive, cancel
}

// Done implements the Done method of the context.Context interface.
func (g *GracefulContext) Done() <-chan struct{} {
	return g.doneCh
}
