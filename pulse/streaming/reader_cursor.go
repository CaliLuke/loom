package streaming

import (
	"context"
	"fmt"
	"slices"

	"github.com/CaliLuke/go-redis/v9"

	"github.com/CaliLuke/loom/pulse/internal/redisop"
)

// prepareReaderCursor resolves newest once, before a stream registration is
// published. Explicit positions retain their spelling and require no Redis I/O.
func prepareReaderCursor(ctx context.Context, rdb *redis.Client, key, startID string) (string, error) {
	if startID != "$" {
		return startID, nil
	}
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("prepare newest cursor for stream %q: %w", key, err)
	}
	if rdb == nil {
		return "", fmt.Errorf("prepare newest cursor for stream %q: Redis client is nil", key)
	}
	messages, err := redisop.Client(rdb).XRevRangeN(ctx, key, "+", "-", 1).Result()
	if err != nil {
		return "", fmt.Errorf("prepare newest cursor for stream %q: %w", key, err)
	}
	if len(messages) == 0 {
		return "0-0", nil
	}
	return messages[0].ID, nil
}

// canAddStream checks registration admission while r.lock is held. Both the
// lookup's start and its publication must check the same current state.
func (r *Reader) canAddStream(key string) (bool, error) {
	if r.lifetime.Stopping() {
		return false, fmt.Errorf("reader is closing")
	}
	return r.registrationIndex(key) < 0, nil
}

// registrationIndex resolves public stream membership while r.lock is held.
// Read completion instead compares registration instances to preserve lifetimes.
func (r *Reader) registrationIndex(key string) int {
	return slices.IndexFunc(r.registrations, func(registration *readerRegistration) bool {
		return registration.stream.key == key
	})
}
