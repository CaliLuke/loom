package rmap

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Close closes the connection to the map, freeing resources. It is safe to
// call Close multiple times.
func (sm *Map) Close() {
	sm.lock.Lock()
	if sm.closing {
		sm.lock.Unlock()
		return
	}
	sm.closing = true
	sm.lock.Unlock()

	if sm.closer != nil {
		sm.closer()
	}

	// Signal run() to stop and wait for it to complete
	close(sm.done)
	sm.wait.Wait()

	// Clean up all waiters
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
		if err := script.Load(ctx, sm.rdb).Err(); err != nil {
			return fmt.Errorf("pulse map: %s failed to load Lua scripts %v: %w", sm.Name, script, err)
		}
	}

	// Subscribe to updates.
	sm.sub = sm.rdb.Subscribe(ctx, sm.chankey)
	_, err := sm.sub.Receive(ctx) // Fail fast if we can't subscribe.
	if err != nil {
		_ = sm.sub.Close()
		return fmt.Errorf("pulse map: %s failed to join: %w", sm.Name, err)
	}
	sm.msgch = sm.sub.Channel()

	// read initial content
	// Note: there's a (very) small window where we might be receiving
	// updates for changes that are already applied by the time we call
	// HGetAll. This is not a problem because we'll just overwrite the
	// local copy with the same data.
	cmd := sm.rdb.HGetAll(ctx, sm.hashkey)
	if err := cmd.Err(); err != nil {
		_ = sm.sub.Unsubscribe(ctx, sm.chankey)
		_ = sm.sub.Close()
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
			// no need to lock, stopping is true
			for _, c := range sm.chans {
				close(c)
			}
			if sm.sub != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := sm.sub.Unsubscribe(ctx, sm.chankey); err != nil {
					sm.logger.Error(fmt.Errorf("failed to unsubscribe: %w", err))
				}
				cancel()
				if err := sm.sub.Close(); err != nil {
					sm.logger.Error(fmt.Errorf("failed to close subscription: %w", err))
				}
			}
			return
		}
	}
}

// runLuaScript validates the user key and executes the given Lua script. Reset
// and Destroy remain available after Close because they mutate Redis state even
// when the local replica is already frozen.
func (sm *Map) runLuaScript(ctx context.Context, name string, script *redis.Script, args ...any) (any, error) {
	sm.lock.RLock()
	if sm.closing && name != "reset" && name != "destroy" {
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
	res, err := script.Run(ctx, sm.rdb, []string{sm.hashkey, sm.chankey}, args...).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("pulse map: %s failed to run %q for key %s: %w", sm.Name, name, key, err)
	}
	if err := sm.applyTTL(ctx); err != nil {
		return nil, fmt.Errorf("pulse map: %s failed to apply TTL: %w", sm.Name, err)
	}

	return res, nil
}

func (sm *Map) applyTTL(ctx context.Context) error {
	if sm.ttl <= 0 {
		return nil
	}
	if sm.ttlSliding {
		return sm.rdb.Expire(ctx, sm.hashkey, sm.ttl).Err()
	}
	_, err := sm.rdb.ExpireNX(ctx, sm.hashkey, sm.ttl).Result()
	return err
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
		closing := sm.closing
		sm.lock.RUnlock()
		if closing {
			return
		}

		sub := sm.rdb.Subscribe(sm.closectx, sm.chankey)
		_, err := sub.Receive(sm.closectx)
		if err != nil {
			_ = sub.Close()
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

		cmd := sm.rdb.HGetAll(sm.closectx, sm.hashkey)
		if err := cmd.Err(); err != nil {
			_ = sub.Close()
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
		if sm.closing {
			sm.lock.Unlock()
			_ = sub.Close()
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
			_ = sub.Close()
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
			_ = oldSub.Close()
		}

		sm.notifyWaitersUpTo(sm.rev)

		sm.logger.Info("reconnected")
		return
	}
}
