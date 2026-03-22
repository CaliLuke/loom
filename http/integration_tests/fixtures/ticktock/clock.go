package ticktock

import (
	"context"
	"time"

	clock "example.com/http-ticktock/gen/clock"
	"goa.design/clue/log"
	goa "github.com/CaliLuke/loom/v3/pkg"
)

type (
	// clocksrvc implements the clock service.
	clocksrvc struct{}
)

const (
	initialDelay = 800 * time.Millisecond
	streamPause  = 125 * time.Millisecond
)

// NewClock returns the clock service implementation.
func NewClock() clock.Service {
	return &clocksrvc{}
}

// Tick implements Tick.
func (s *clocksrvc) Tick(ctx context.Context, stream clock.TickServerStream) (err error) {
	log.Printf(ctx, "clock.Tick")
	return emit(ctx, stream.SendWithContext, "tick", []string{"tick-1", "tick-2", "tick-done"})
}

// Tock implements Tock.
func (s *clocksrvc) Tock(ctx context.Context, stream clock.TockServerStream) (err error) {
	log.Printf(ctx, "clock.Tock")
	return emit(ctx, stream.SendWithContext, "tock", []string{"tock-a", "tock-b", "tock-done"})
}

// Guarded implements Guarded.
func (s *clocksrvc) Guarded(ctx context.Context, p *clock.GuardedPayload, stream clock.GuardedServerStream) error {
	log.Printf(ctx, "clock.Guarded")
	if p.Token == nil || *p.Token != "open-sesame" {
		return goa.PermanentError("unauthorized", "missing or invalid token")
	}
	return emit(ctx, stream.SendWithContext, "guarded", []string{"guarded-1", "guarded-done"})
}

func emit(ctx context.Context, send func(context.Context, *clock.TickTockEvent) error, eventType string, values []string) error {
	timer := time.NewTimer(initialDelay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}

	for i, value := range values {
		if err := send(ctx, &clock.TickTockEvent{
			Event: eventType,
			Data:  value,
		}); err != nil {
			return err
		}
		if i == len(values)-1 {
			continue
		}

		timer := time.NewTimer(streamPause)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return nil
}
