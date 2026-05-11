package rmap

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// redisKeyRegex is a regular expression that matches valid Redis keys.
var redisKeyRegex = regexp.MustCompile(`^[^ \0\*\?\[\]]{1,512}$`)

func isValidRedisKeyName(key string) bool {
	return redisKeyRegex.MatchString(key)
}

// unpackString reads a length-prefixed string from a buffer using struct.pack
// format "ic0"
func unpackString(data []byte) (string, []byte, error) {
	if len(data) < 4 {
		return "", nil, fmt.Errorf("buffer too short for length")
	}
	length := int(binary.LittleEndian.Uint32(data))
	data = data[4:]
	if len(data) < length {
		return "", nil, fmt.Errorf("buffer too short for string")
	}
	return string(data[:length]), data[length:], nil
}

func parseValues(value string) []string {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "[") {
		var decoded []string
		if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
			return decoded
		}
	}
	return strings.Split(value, ",")
}

func parseRevisionString(s string) (uint64, bool) {
	rev, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return rev, true
}

func extractState(content map[string]string) (uint64, EventKind) {
	if content == nil {
		return 0, EventChange
	}
	kind := EventChange
	if s, ok := content[kindField]; ok {
		kind = parseStoredKind(s)
		delete(content, kindField)
	}
	s, ok := content[revField]
	if !ok {
		return 0, kind
	}
	delete(content, revField)
	if s == "" {
		return 0, kind
	}
	rev, ok := parseRevisionString(s)
	if !ok {
		return 0, kind
	}
	return rev, kind
}

func unpackRevision(data []byte) (uint64, bool) {
	if len(data) < 4 {
		return 0, false
	}
	s, _, err := unpackString(data)
	if err != nil {
		return 0, false
	}
	return parseRevisionString(s)
}

// runSetScript executes the Set Lua script and returns the previous value plus
// the committed revision observed by Redis.
func (sm *Map) runSetScript(ctx context.Context, key, value string) (setResult, error) {
	raw, err := sm.runLuaScript(ctx, "set", sm.setScript, key, value)
	if err != nil {
		return setResult{}, err
	}
	values, ok := raw.([]any)
	if !ok || len(values) != 3 {
		return setResult{}, fmt.Errorf("pulse map: %s returned invalid set result %T", sm.Name, raw)
	}
	existed, ok := values[0].(int64)
	if !ok {
		return setResult{}, fmt.Errorf("pulse map: %s returned invalid set existence flag %T", sm.Name, values[0])
	}
	prev, ok := values[1].(string)
	if !ok {
		return setResult{}, fmt.Errorf("pulse map: %s returned invalid set previous value %T", sm.Name, values[1])
	}
	revString, ok := values[2].(string)
	if !ok {
		return setResult{}, fmt.Errorf("pulse map: %s returned invalid set revision %T", sm.Name, values[2])
	}
	rev, ok := parseRevisionString(revString)
	if !ok {
		return setResult{}, fmt.Errorf("pulse map: %s returned invalid set revision %q", sm.Name, revString)
	}
	return setResult{prev: prev, existed: existed == 1, rev: rev}, nil
}

// registerWaiter installs a waiter unless the local replica has already caught
// up to the committed revision it targets.
func (sm *Map) registerWaiter(waiter *setWaiter) bool {
	sm.wlock.Lock()
	defer sm.wlock.Unlock()
	sm.lock.RLock()
	defer sm.lock.RUnlock()
	if sm.rev >= waiter.rev {
		return true
	}
	sm.waiters[waiter.rev] = append(sm.waiters[waiter.rev], waiter)
	return false
}

// removeWaiter unregisters a waiter after SetAndWait returns or is canceled.
func (sm *Map) removeWaiter(waiter *setWaiter) {
	sm.wlock.Lock()
	defer sm.wlock.Unlock()
	waiters, ok := sm.waiters[waiter.rev]
	if !ok {
		return
	}
	filtered := waiters[:0]
	for _, current := range waiters {
		if current != waiter {
			filtered = append(filtered, current)
		}
	}
	if len(filtered) == 0 {
		delete(sm.waiters, waiter.rev)
		return
	}
	sm.waiters[waiter.rev] = filtered
}

// notifyWaitersUpTo releases every waiter whose committed revision is now
// visible in the local replica.
func (sm *Map) notifyWaitersUpTo(rev uint64) {
	sm.wlock.Lock()
	var ready []*setWaiter
	for waiterRev, waiters := range sm.waiters {
		if waiterRev > rev {
			continue
		}
		ready = append(ready, waiters...)
		delete(sm.waiters, waiterRev)
	}
	sm.wlock.Unlock()
	for _, waiter := range ready {
		select {
		case waiter.ch <- struct{}{}:
		case <-waiter.ctx.Done():
		default:
		}
	}
}

// notifySubscribers sends a best-effort notification to every subscriber while
// the map lock is held, so the subscriber list stays stable during iteration.
func (sm *Map) notifySubscribers(kind EventKind) {
	for _, c := range sm.chans {
		select {
		case c <- kind:
		default:
		}
	}
}

// applyMessageLocked applies one pubsub message to the local replica. The map
// lock must be held and the returned revision is the highest revision observed
// after the update is applied.
//
//nolint:maintidx // Pubsub payload decoding is intentionally centralized.
func (sm *Map) applyMessageLocked(op string, data []byte) (mapChange, bool, error) {
	switch op {
	case "destroy":
		rev, ok := parseRevisionString(string(data))
		if !ok {
			return mapChange{}, false, fmt.Errorf("invalid destroy payload")
		}
		if rev <= sm.rev {
			return mapChange{}, false, nil
		}
		sm.rev = rev
		sm.content = make(map[string]string)
		sm.logger.Debug("destroy")
		return mapChange{kind: EventReset, rev: rev}, true, nil
	case "reset":
		rev, ok := parseRevisionString(string(data))
		if !ok {
			return mapChange{}, false, fmt.Errorf("invalid reset payload")
		}
		if rev <= sm.rev {
			return mapChange{}, false, nil
		}
		sm.rev = rev
		sm.content = make(map[string]string)
		sm.logger.Debug("reset")
		return mapChange{kind: EventReset, rev: rev}, true, nil
	case "del":
		key, rest, err := unpackString(data)
		if err != nil {
			return mapChange{}, false, fmt.Errorf("invalid del payload: %w", err)
		}
		if key == revField || key == kindField {
			return mapChange{}, false, nil
		}
		rev, ok := unpackRevision(rest)
		if !ok {
			return mapChange{}, false, fmt.Errorf("invalid del revision")
		}
		if rev <= sm.rev {
			return mapChange{}, false, nil
		}
		sm.rev = rev
		delete(sm.content, key)
		sm.logger.Debug("deleted", "key", key)
		return mapChange{kind: EventDelete, rev: rev}, true, nil
	case "set":
		key, rest, err := unpackString(data)
		if err != nil {
			return mapChange{}, false, fmt.Errorf("invalid set key: %w", err)
		}
		if key == revField || key == kindField {
			return mapChange{}, false, nil
		}
		val, rest, err := unpackString(rest)
		if err != nil {
			return mapChange{}, false, fmt.Errorf("invalid set value: %w", err)
		}
		rev, ok := unpackRevision(rest)
		if !ok {
			return mapChange{}, false, fmt.Errorf("invalid set revision")
		}
		if rev <= sm.rev {
			return mapChange{}, false, nil
		}
		sm.rev = rev
		sm.content[key] = val
		sm.logger.Debug("set", "key", key, "val", val)
		return mapChange{kind: EventChange, rev: rev}, true, nil
	default:
		return mapChange{}, false, fmt.Errorf("invalid payload")
	}
}

func parseStoredKind(kind string) EventKind {
	switch kind {
	case "del":
		return EventDelete
	case "destroy", "reset":
		return EventReset
	default:
		return EventChange
	}
}

func strongerEventKind(current, next EventKind) EventKind {
	if eventPriority(next) > eventPriority(current) {
		return next
	}
	return current
}

func eventPriority(kind EventKind) int {
	switch kind {
	case EventReset:
		return 3
	case EventDelete:
		return 2
	case EventChange:
		return 1
	default:
		return 0
	}
}
