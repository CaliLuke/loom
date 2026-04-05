package testkit

import (
	"context"
	"time"
)

const shutdownTimeout = 5 * time.Second

func newShutdownContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), shutdownTimeout)
}
