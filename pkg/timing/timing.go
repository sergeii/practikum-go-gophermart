package timing

import (
	"context"
	"time"
)

// Wait returns a context-interruptable timer in the form of a receive-only channel
func Wait(ctx context.Context, dur time.Duration) <-chan struct{} {
	waitCh := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			close(waitCh)
		case <-time.After(dur):
			waitCh <- struct{}{}
		}
	}()
	return waitCh
}
