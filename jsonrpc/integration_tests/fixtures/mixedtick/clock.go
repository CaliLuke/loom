package mixedtick

import (
	"context"
	"time"

	clock "example.com/mixedtick/gen/clock"
)

type clocksrvc struct{}

func NewClock() clock.Service {
	return &clocksrvc{}
}

func (s *clocksrvc) Initialize(ctx context.Context, p *clock.InitializePayload) (*clock.InitializeResult, error) {
	return &clock.InitializeResult{
		ID:              p.ID,
		ProtocolVersion: stringPtr("2026-03-22"),
	}, nil
}

func (s *clocksrvc) Tick(ctx context.Context, p *clock.TickPayload, stream clock.TickServerStream) error {
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ticker.C:
	}

	if err := stream.Send(ctx, &clock.TickResult{Value: stringPtr("tick-1")}); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ticker.C:
	}

	return stream.SendAndClose(ctx, &clock.TickResult{Value: stringPtr("tick-done")})
}

func stringPtr(v string) *string {
	return &v
}
