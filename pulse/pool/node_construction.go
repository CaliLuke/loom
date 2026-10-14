package pool

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/CaliLuke/go-redis/v9"

	"github.com/CaliLuke/loom/pulse/rmap"
)

type (
	// nodeConstruction owns acquired handles and attempted private writes until
	// the complete node takes ownership. It is confined to the constructor.
	nodeConstruction struct {
		shutdownTTL time.Duration
		releases    []nodeConstructionRelease
	}

	nodeConstructionRelease struct {
		name    string
		release func(context.Context) error
	}
)

// own records a successful local acquisition or, before issuing a remote write,
// its withdrawal obligation. A failed reply does not imply that no write occurred.
func (c *nodeConstruction) own(name string, release func(context.Context) error) {
	c.releases = append(c.releases, nodeConstructionRelease{name: name, release: release})
}

// joinMap keeps acquisition and local handle ownership inseparable. Shared map
// data belongs to the pool and must never be destroyed by constructor rollback.
func (c *nodeConstruction) joinMap(
	ctx context.Context, name string, rdb *redis.Client, opts ...rmap.MapOption,
) (*rmap.Map, error) {
	m, err := rmap.Join(ctx, name, rdb, opts...)
	if err != nil {
		return nil, err
	}
	c.own("close map "+name, func(context.Context) error {
		m.Close()
		return nil
	})
	return m, nil
}

// transfer relinquishes all constructor obligations before the completed node
// subscribes or starts its runtime. The constructor calls it exactly once.
func (c *nodeConstruction) transfer() {
	c.releases = nil
}

// abort continues after every error and consumes its stack exactly once. Local
// Close remains synchronous; finite subordinate operations are owned by redisop
// and the streaming/rmap lifecycle, not by a detached cleanup goroutine.
func (c *nodeConstruction) abort(ctx context.Context) error {
	bound := c.shutdownTTL
	if bound <= 0 {
		bound = defaultPoolOptions().workerShutdownTTL
	}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), bound)
	defer cancel()
	var errs []error
	for len(c.releases) > 0 {
		last := len(c.releases) - 1
		step := c.releases[last]
		c.releases = c.releases[:last]
		if err := step.release(cleanupCtx); err != nil {
			errs = append(errs, fmt.Errorf("AddNode rollback: %s: %w", step.name, err))
		}
	}
	return errors.Join(errs...)
}
