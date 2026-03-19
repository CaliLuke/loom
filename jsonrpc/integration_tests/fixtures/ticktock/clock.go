package ticktock

import (
	"context"
	"time"

	clock "example.com/ticktock/gen/clock"
)

type clocksrvc struct{}

func NewClock() clock.Service {
	return &clocksrvc{}
}

func (s *clocksrvc) Tick(ctx context.Context, p *clock.TickPayload, stream clock.TickServerStream) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for _, value := range []string{"tick-1", "tick-2"} {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		if err := stream.Send(ctx, &clock.TickResult{Value: stringPtr(value)}); err != nil {
			return err
		}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ticker.C:
	}

	return stream.SendAndClose(ctx, &clock.TickResult{Value: stringPtr("tick-done")})
}

func (s *clocksrvc) Tock(ctx context.Context, p *clock.TockPayload, stream clock.TockServerStream) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for _, value := range []string{"tock-a", "tock-b"} {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		if err := stream.Send(ctx, &clock.TockResult{Value: stringPtr(value)}); err != nil {
			return err
		}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ticker.C:
	}

	return stream.SendAndClose(ctx, &clock.TockResult{Value: stringPtr("tock-finished")})
}

func stringPtr(v string) *string {
	return &v
}
