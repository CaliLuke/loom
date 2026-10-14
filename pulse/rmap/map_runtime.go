package rmap

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CaliLuke/go-redis/v9"

	"github.com/CaliLuke/loom/pulse/internal/keyttl"
	"github.com/CaliLuke/loom/pulse/internal/redisop"
	"github.com/CaliLuke/loom/pulse/internal/rmapstate"
)

// Close closes the connection to the map, freeing resources. It is safe to
// call Close multiple times.
func (sm *Map) Close() {
	sm.lifetime.Close(func() {
		if sm.closer != nil {
			sm.closer()
		}
		close(sm.done)
	}, func() {
		sm.wait.Wait()
		sm.lock.Lock()
		sm.closed = true
		sm.lock.Unlock()
	})
}

// init initializes the map.
func (sm *Map) init(ctx context.Context) error {
	// Make sure scripts are cached.
	for _, script := range []*rmapstate.Script{
		sm.appendScript,
		sm.appendUniqueScript,
		sm.delScript,
		sm.incrScript,
		sm.removeScript,
		sm.resetScript,
		sm.destroyScript,
		sm.setScript,
		sm.testAndDelScript,
		sm.testAndResetScript,
		sm.testAndSetScript,
		sm.setIfNotExistsScript,
		luaSnapshot,
	} {
		if err := script.Load(ctx, redisop.Client(sm.rdb)).Err(); err != nil {
			return fmt.Errorf("pulse map: %s failed to load Lua scripts %v: %w", sm.Name, script, err)
		}
	}

	// Subscribe to updates.
	sub, err := sm.subscribe(ctx)
	if err != nil {
		return fmt.Errorf("pulse map: %s failed to join: %w", sm.Name, err)
	}
	sm.sub = sub
	sm.msgch = sm.sub.Channel()

	state, err := sm.readSnapshot(ctx)
	if err != nil {
		if closeErr := sm.sub.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close subscription: %w", closeErr))
		}
		return fmt.Errorf("pulse map: %s failed to read initial content: %w", sm.Name, err)
	}
	sm.content, sm.generation, sm.rev = state.content, state.generation, state.rev

	return nil
}

// run is the sole publisher of complete replica state. Network operations run
// outside the local read lock, but snapshots and deltas never publish concurrently.
func (sm *Map) run() {
	defer sm.wait.Done()
	ticker := time.NewTicker(reconcileInterval)
	defer ticker.Stop()
	defer func() {
		sm.lock.Lock()
		for _, c := range sm.chans {
			close(c)
		}
		sm.chans = nil
		sm.lock.Unlock()
		sm.closeSubscription(sm.sub)
		sm.logger.Info("closed")
	}()
	for {
		if sm.lifetime.Stopping() {
			return
		}
		select {
		case msg, ok := <-sm.msgch:
			if !ok {
				sm.closeSubscription(sm.sub)
				sm.sub, sm.msgch = nil, nil
				if err := sm.reconnect(); err != nil {
					sm.logger.Error(err)
				} else if err := sm.reconcile(sm.closectx); err != nil {
					sm.logger.Error(err)
				}
				continue
			}
			if err := sm.receiveMessage(msg.Payload); err != nil {
				sm.logger.Error(err)
			}
		case request := <-sm.observations:
			request.result <- sm.observe(request)
		case <-ticker.C:
			if sm.sub == nil {
				if err := sm.reconnect(); err != nil {
					sm.logger.Error(err)
				}
			}
			if err := sm.reconcile(sm.closectx); err != nil {
				sm.logger.Error(err)
			}
		case <-sm.done:
			return
		}
	}
}

func (sm *Map) receiveMessage(payload string) error {
	parts := strings.SplitN(payload, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid map payload")
	}
	sm.lock.Lock()
	if sm.lifetime.Stopping() {
		sm.lock.Unlock()
		return nil
	}
	change, applied, err := sm.applyMessageLocked(parts[0], []byte(parts[1]))
	if applied {
		sm.notifySubscribers(change.kind)
	}
	sm.lock.Unlock()
	if err != nil {
		return err
	}
	if change.refresh {
		return sm.reconcile(sm.closectx)
	}
	return nil
}

// observe runs after the write's reply and serializes its read with publication.
func (sm *Map) observe(request observationRequest) error {
	ctx, cancel := context.WithCancel(request.ctx)
	stop := context.AfterFunc(sm.closectx, cancel)
	defer stop()
	defer cancel()
	if err := ctx.Err(); err != nil {
		return err
	}
	if sm.generation == request.generation && sm.rev >= request.rev {
		return nil
	}
	if err := sm.reconcile(ctx); err != nil {
		return err
	}
	if sm.generation != request.generation || sm.rev < request.rev {
		return fmt.Errorf("pulse map: %s committed incarnation retired before observation", sm.Name)
	}
	return nil
}

func (sm *Map) readSnapshot(ctx context.Context) (replicaState, error) {
	content, err := redisop.Client(sm.rdb).HGetAll(ctx, sm.hashkey).Result()
	if err != nil {
		return replicaState{}, fmt.Errorf("pulse map: %s read snapshot: %w", sm.Name, err)
	}
	if len(content) > 0 && content[protocolField] == "" {
		values, err := luaSnapshot.Run(ctx, redisop.Client(sm.rdb), []string{sm.hashkey}).StringSlice()
		if err != nil {
			return replicaState{}, fmt.Errorf("pulse map: %s adopt snapshot: %w", sm.Name, err)
		}
		if len(values)%2 != 0 {
			return replicaState{}, fmt.Errorf("pulse map: %s invalid snapshot", sm.Name)
		}
		content = make(map[string]string, len(values)/2)
		for i := 0; i < len(values); i += 2 {
			content[values[i]] = values[i+1]
		}
	}
	return extractState(content)
}

func (sm *Map) reconcile(ctx context.Context) error {
	state, err := sm.readSnapshot(ctx)
	if err != nil {
		return err
	}
	sm.lock.Lock()
	defer sm.lock.Unlock()
	if sm.lifetime.Stopping() {
		return fmt.Errorf("pulse map: %s is stopped", sm.Name)
	}
	if state.generation == sm.generation && state.rev == sm.rev {
		return nil
	}
	kind := EventChange
	switch {
	case sm.generation != "" && state.generation != sm.generation:
		kind = EventReset
	case state.reset > sm.rev:
		kind = EventReset
	case state.deleted > sm.rev:
		kind = EventDelete
	}
	sm.content, sm.generation, sm.rev = state.content, state.generation, state.rev
	sm.notifySubscribers(kind)
	return nil
}

// runLuaScript validates the user key and executes the given Lua script. Reset
// and Destroy remain available after Close because they mutate Redis state even
// when the local replica is already frozen.
func (sm *Map) runLuaScript(ctx context.Context, name string, script *rmapstate.Script, args ...any) (any, error) {
	sm.lock.RLock()
	if sm.lifetime.Stopping() && name != "reset" && name != "destroy" {
		sm.lock.RUnlock()
		return "", fmt.Errorf("pulse map: %s is stopped", sm.Name)
	}
	sm.lock.RUnlock()
	key := args[0].(string)
	if len(key) == 0 {
		return nil, fmt.Errorf("pulse map: %s key cannot be empty in %q", sm.Name, name)
	}
	if strings.Contains(key, "=") {
		return nil, fmt.Errorf("pulse map: %s key %q cannot contain '=' in %q", sm.Name, key, name)
	}
	ttlSeconds, ttlSliding := keyttl.Args(sm.ttl, sm.ttlSliding)
	args = append(args[:len(args):len(args)], ttlSeconds, ttlSliding)
	res, err := script.Run(ctx, redisop.Client(sm.rdb), []string{sm.hashkey, sm.chankey}, args...).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("pulse map: %s failed to run %q for key %s: %w", sm.Name, name, key, err)
	}

	return res, nil
}

// reconnect makes one bounded attempt. Periodic reconciliation owns retries,
// so a disconnected subscription cannot monopolize write-observation requests.
func (sm *Map) reconnect() error {
	sub, err := sm.subscribe(sm.closectx)
	if err != nil {
		return fmt.Errorf("pulse map: %s resubscribe: %w", sm.Name, err)
	}
	if sm.lifetime.Stopping() {
		sm.closeSubscription(sub)
		return nil
	}
	sm.sub, sm.msgch = sub, sub.Channel()
	return nil
}

// subscribe bounds both connection acquisition and the first subscription reply.
func (sm *Map) subscribe(ctx context.Context) (*redis.PubSub, error) {
	bounded, cancel := redisop.Context(ctx, sm.rdb)
	defer cancel()
	borrowed := redisop.Client(sm.rdb)
	sub := borrowed.Subscribe(bounded, sm.chankey)
	if _, err := sub.ReceiveTimeout(bounded, borrowed.Options().ReadTimeout); err != nil {
		if closeErr := sub.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close subscription: %w", closeErr))
		}
		return nil, err
	}
	return sub, nil
}

func (sm *Map) closeSubscription(sub *redis.PubSub) {
	if sub != nil {
		if err := sub.Close(); err != nil {
			sm.logger.Error(fmt.Errorf("failed to close subscription: %w", err))
		}
	}
}
