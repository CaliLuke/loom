package streaming

import (
	"context"
	"fmt"
	"slices"

	"github.com/redis/go-redis/v9"
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
	messages, err := rdb.XRevRangeN(ctx, key, "+", "-", 1).Result()
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
	if r.closing {
		return false, fmt.Errorf("reader is closing")
	}
	return !slices.ContainsFunc(r.registrations, func(registration *readerRegistration) bool {
		return registration.stream.key == key
	}), nil
}
