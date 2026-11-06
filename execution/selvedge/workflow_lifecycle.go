package selvedge

import (
	"context"
	"errors"
	"fmt"

	sv "github.com/CaliLuke/selvedge"
)

func (c *workflowCore) start(ctx context.Context) error {
	if c == nil {
		return ErrNotRunning
	}
	c.mu.Lock()
	if c.state != workflowNew {
		c.mu.Unlock()
		return fmt.Errorf("%w: cannot start this workflow role again", ErrNotRunning)
	}
	c.state = workflowStarting
	life, cancel := context.WithCancel(context.Background())
	c.cancelLife = cancel
	c.mu.Unlock()

	watched := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		cancel()
		close(watched)
	})
	if ctx.Err() != nil {
		cancel()
	}
	err := c.native.start(life)
	if !stop() {
		<-watched
	}
	cancelled := life.Err() != nil

	c.mu.Lock()
	defer c.mu.Unlock()
	switch {
	case c.shutdownRequested:
		c.state = workflowStopping
		go c.cleanup(err == nil)
		return errors.Join(ErrNotRunning, err)
	case err == nil && cancelled:
		c.state = workflowStopping
		go c.cleanup(true)
		return errors.Join(ErrNotRunning, ctx.Err())
	case err != nil:
		cancel()
		c.state = workflowStopped
		close(c.cleanupDone)
		return err
	default:
		c.state = workflowRunning
		return nil
	}
}

func (c *workflowCore) shutdown(ctx context.Context) error {
	if c == nil {
		return ErrNotRunning
	}
	c.mu.Lock()
	switch c.state {
	case workflowNew:
		c.state = workflowStopped
		close(c.cleanupDone)
	case workflowStarting:
		c.shutdownRequested = true
	case workflowRunning:
		c.state = workflowStopping
		go c.cleanup(true)
	}
	done := c.cleanupDone
	c.mu.Unlock()
	select {
	case <-done:
		return c.cleanupResult
	default:
	}
	select {
	case <-done:
		return c.cleanupResult
	case <-ctx.Done():
		select {
		case <-done:
			return c.cleanupResult
		default:
			return fmt.Errorf("%w: %w", ErrShutdownIncomplete, ctx.Err())
		}
	}
}

func (c *workflowCore) ready() error {
	if c == nil {
		return ErrNotRunning
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state != workflowRunning {
		return ErrNotRunning
	}
	return nil
}

func (c *workflowCore) cleanup(started bool) {
	c.operations.Wait()
	if c.cancelLife != nil {
		c.cancelLife()
	}
	var result error
	if started {
		for {
			result = c.native.shutdown(context.Background())
			if !errors.Is(result, sv.ErrShutdownInterrupted) {
				break
			}
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cleanupResult = result
	c.state = workflowStopped
	close(c.cleanupDone)
}
