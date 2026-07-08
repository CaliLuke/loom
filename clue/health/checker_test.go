package health

import (
	"context"
	"testing"
	"time"
)

func TestCheckerPingContextInheritsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, healthy := NewChecker(contextPinger{
		name: "dep",
		ping: func(ctx context.Context) error {
			return ctx.Err()
		},
	}).Check(ctx)

	if healthy {
		t.Fatal("expected canceled dependency context to make health check unhealthy")
	}
}

func TestNewPingerDefaultTimeout(t *testing.T) {
	pinger := NewPinger("dep", "example.com")
	client, ok := pinger.(*client)
	if !ok {
		t.Fatalf("expected *client, got %T", pinger)
	}
	if got, want := client.httpClient.Timeout, 5*time.Second; got != want {
		t.Fatalf("expected default timeout %s, got %s", want, got)
	}
}

func TestCheckerPingPanicMakesDependencyUnhealthy(t *testing.T) {
	res, healthy := NewChecker(contextPinger{
		name: "dep",
		ping: func(context.Context) error {
			panic("boom")
		},
	}).Check(context.Background())

	if healthy {
		t.Fatal("expected panicking dependency to make health check unhealthy")
	}
	if got := res.Status["dep"]; got != "NOT OK" {
		t.Fatalf("expected panicking dependency status NOT OK, got %q", got)
	}
}

type contextPinger struct {
	name string
	ping func(context.Context) error
}

func (p contextPinger) Name() string {
	return p.name
}

func (p contextPinger) Ping(ctx context.Context) error {
	return p.ping(ctx)
}
