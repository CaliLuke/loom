package rmap

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/CaliLuke/go-redis/v9"

	"github.com/CaliLuke/loom/pulse/internal/keyttl"
	"github.com/CaliLuke/loom/pulse/internal/redisop"
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
		sm.wlock.Lock()
		for rev, waiters := range sm.waiters {
			for _, w := range waiters {
				w.cancel()
			}
			delete(sm.waiters, rev)
		}
		sm.wlock.Unlock()
		sm.lock.Lock()
		sm.closed = true
		sm.lock.Unlock()
	})
}

// init initializes the map.
func (sm *Map) init(ctx context.Context) error {
	// Make sure scripts are cached.
	for _, script := range []*redis.Script{
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

	// read initial content
	// Note: there's a (very) small window where we might be receiving
	// updates for changes that are already applied by the time we call
	// HGetAll. This is not a problem because we'll just overwrite the
	// local copy with the same data.
	cmd := redisop.Client(sm.rdb).HGetAll(ctx, sm.hashkey)
	if err := cmd.Err(); err != nil {
		if closeErr := sm.sub.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close subscription: %w", closeErr))
		}
		return fmt.Errorf("pulse map: %s failed to read initial content: %w", sm.Name, err)
	}
	sm.content = cmd.Val()
	sm.rev, _ = extractState(sm.content)

	return nil
}

// run updates the local copy of the replicated map whenever a remote update is
// received and sends notifications when needed.
func (sm *Map) run() {
	defer sm.wait.Done()
	for {
		select {
		case msg, ok := <-sm.msgch:
			if !ok {
				// disconnected from Redis server, attempt to reconnect forever
				sm.logger.Error(fmt.Errorf("disconnected"))
				sm.reconnect()
				continue
			}
			parts := strings.SplitN(msg.Payload, ":", 2)
			if len(parts) != 2 {
				sm.logger.Error(fmt.Errorf("invalid payload"), "payload", msg.Payload)
				continue
			}
			op, data := parts[0], []byte(parts[1])
			sm.lock.Lock()
			change, applied, err := sm.applyMessageLocked(op, data)
			if err != nil {
				sm.logger.Error(err, "payload", msg.Payload)
				sm.lock.Unlock()
				continue
			}
			if applied {
				sm.notifySubscribers(change.kind)
			}
			sm.lock.Unlock()
			if applied {
				sm.notifyWaitersUpTo(change.rev)
			}

		case <-sm.done:
			sm.logger.Info("closed")
			// Join subscriber admission that began before Stop sealed the owner.
			sm.lock.Lock()
			for _, c := range sm.chans {
				close(c)
			}
			sm.chans = nil
			sm.lock.Unlock()
			sm.closeSubscription(sm.sub)
			return
		}
	}
}

// runLuaScript validates the user key and executes the given Lua script. Reset
// and Destroy remain available after Close because they mutate Redis state even
// when the local replica is already frozen.
func (sm *Map) runLuaScript(ctx context.Context, name string, script *redis.Script, args ...any) (any, error) {
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

// reconnect attempts to reconnect to the Redis server forever.
//
//nolint:maintidx // Reconnect keeps resubscribe and snapshot replay in one loop.
func (sm *Map) reconnect() {
	var count int
	for {
		count++
		sm.logger.Info("reconnect", "attempt", count)
		if sm.closectx.Err() != nil {
			return
		}
		sm.lock.RLock()
		closing := sm.lifetime.Stopping()
		sm.lock.RUnlock()
		if closing {
			return
		}

		sub, err := sm.subscribe(sm.closectx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			sm.logger.Error(fmt.Errorf("failed to reconnect: %w", err), "attempt", count)
			sleep := time.Duration(rand.Float64()*5+1) * time.Second
			timer := time.NewTimer(sleep)
			select {
			case <-timer.C:
			case <-sm.closectx.Done():
				timer.Stop()
				return
			}
			continue
		}

		msgch := sub.Channel()

		cmd := redisop.Client(sm.rdb).HGetAll(sm.closectx, sm.hashkey)
		if err := cmd.Err(); err != nil {
			sm.closeSubscription(sub)
			if errors.Is(err, context.Canceled) {
				return
			}
			sm.logger.Error(fmt.Errorf("failed to resync: %w", err), "attempt", count)
			sleep := time.Duration(rand.Float64()*5+1) * time.Second
			timer := time.NewTimer(sleep)
			select {
			case <-timer.C:
			case <-sm.closectx.Done():
				timer.Stop()
				return
			}
			continue
		}

		sm.lock.Lock()
		if sm.lifetime.Stopping() {
			sm.lock.Unlock()
			sm.closeSubscription(sub)
			return
		}

		// Resync local state from Redis, then apply any messages that were queued
		// while reading the snapshot under the same lock so readers never observe
		// intermediate state regressions. Revisioned messages at or below the
		// snapshot revision are ignored.
		oldRev := sm.rev
		sm.content = cmd.Val()
		var recoveredKind EventKind
		sm.rev, recoveredKind = extractState(sm.content)
		if sm.rev <= oldRev {
			recoveredKind = EventChange
		}
		drainErr := false
		for {
			select {
			case msg, ok := <-msgch:
				if !ok {
					drainErr = true
					goto drained
				}
				parts := strings.SplitN(msg.Payload, ":", 2)
				if len(parts) != 2 {
					sm.logger.Error(fmt.Errorf("invalid payload"), "payload", msg.Payload)
					continue
				}
				op, data := parts[0], []byte(parts[1])
				change, applied, err := sm.applyMessageLocked(op, data)
				if err != nil {
					sm.logger.Error(err, "payload", msg.Payload)
					continue
				}
				if applied {
					recoveredKind = strongerEventKind(recoveredKind, change.kind)
				}
			default:
				goto drained
			}
		}

	drained:
		if drainErr {
			sm.lock.Unlock()
			sm.closeSubscription(sub)
			sleep := time.Duration(rand.Float64()*5+1) * time.Second
			timer := time.NewTimer(sleep)
			select {
			case <-timer.C:
			case <-sm.closectx.Done():
				timer.Stop()
				return
			}
			continue
		}

		oldSub := sm.sub
		sm.sub = sub
		sm.msgch = msgch
		sm.notifySubscribers(recoveredKind)
		sm.lock.Unlock()
		if oldSub != nil {
			sm.closeSubscription(oldSub)
		}

		sm.notifyWaitersUpTo(sm.rev)

		sm.logger.Info("reconnected")
		return
	}
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
