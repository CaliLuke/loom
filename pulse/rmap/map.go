// Package rmap maintains a Redis-backed replicated map whose local state is
// ordered by incarnation and revision, with Pub/Sub hints and snapshot reconciliation.
package rmap

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/CaliLuke/go-redis/v9"

	"github.com/CaliLuke/loom/pulse/internal/lifecycle"
	"github.com/CaliLuke/loom/pulse/internal/rmapstate"
	"github.com/CaliLuke/loom/pulse/pulse"
)

type (
	// Map is a replicated map that emits events when elements
	// change. Multiple processes can join the same replicated map and
	// update it.
	Map struct {
		Name                 string
		chankey              string                // Redis pubsub channel name
		hashkey              string                // Redis hash key
		ttl                  time.Duration         // TTL applied to hashkey (optional)
		ttlSliding           bool                  // refresh TTL on every write when true
		msgch                <-chan *redis.Message // channel to receive map updates
		chans                []chan EventKind      // channels to send notifications
		done                 chan struct{}         // channel to signal shutdown
		closectx             context.Context       // context canceled by Close
		closer               context.CancelFunc    // cancels closectx
		wait                 sync.WaitGroup        // wait for read goroutine to exit
		logger               pulse.Logger          // logger
		sub                  *redis.PubSub         // subscription to map updates
		rdb                  *redis.Client
		setScript            *rmapstate.Script
		testAndSetScript     *rmapstate.Script
		setIfNotExistsScript *rmapstate.Script
		incrScript           *rmapstate.Script
		appendScript         *rmapstate.Script
		appendUniqueScript   *rmapstate.Script
		removeScript         *rmapstate.Script
		delScript            *rmapstate.Script
		testAndDelScript     *rmapstate.Script
		testAndResetScript   *rmapstate.Script
		resetScript          *rmapstate.Script
		destroyScript        *rmapstate.Script

		lock       sync.RWMutex
		content    map[string]string
		rev        uint64
		generation string
		lifetime   lifecycle.Lifetime // owns stop admission and cleanup joining
		closed     bool               // true if Close returned - used by tests

		observations chan observationRequest // consumed only by the replica owner
	}

	// EventKind is the type of map event.
	EventKind int

	// mapChange captures the event kind and revision applied to the local replica.
	mapChange struct {
		kind    EventKind
		rev     uint64
		refresh bool
	}

	// setResult is the previous value and committed revision returned by Set.
	setResult struct {
		prev       string
		existed    bool
		generation string
		rev        uint64
	}

	// observationRequest establishes a causal barrier after a committed write.
	observationRequest struct {
		ctx        context.Context
		generation string
		rev        uint64
		result     chan error
	}

	// replicaState is one complete authoritative snapshot and recovery history.
	replicaState struct {
		content    map[string]string
		generation string
		rev        uint64
		reset      uint64
		deleted    uint64
	}
)

const (
	revField          = "=rev"
	kindField         = "=kind" // accepted only during coordinated metadata adoption
	protocolField     = "=protocol"
	generationField   = "=generation"
	resetField        = "=reset"
	deleteField       = "=delete"
	mapProtocol       = "1"
	reconcileInterval = time.Second
)

const (
	// EventChange is the event emitted when a key is added or changed.
	EventChange EventKind = iota + 1
	// EventDelete is the event emitted when a key is deleted.
	EventDelete
	// EventReset is the event emitted when the map is reset.
	EventReset
)

// Join retrieves the content of the replicated map with the given name and
// subscribes to updates. The local content is eventually consistent across all
// nodes that join the replicated map with the same name.
//
// Clients can call the Map method on the returned Map to retrieve a copy of
// its content and subscribe to its C channel to receive updates when the
// content changes (note that multiple remote changes may result in a single
// notification). The returned Map is safe for concurrent use.
//
// Clients should call Close before exiting to stop updates and release
// resources resulting in a read-only point-in-time copy.
func Join(ctx context.Context, name string, rdb *redis.Client, opts ...MapOption) (*Map, error) {
	if !rmapstate.ValidName(name) {
		return nil, fmt.Errorf("pulse map: not a valid map name %q", name)
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if rdb == nil {
		return nil, fmt.Errorf("pulse map: %s Redis client cannot be nil", name)
	}
	o := parseOptions(opts...)
	if o.Logger == nil {
		o.Logger = pulse.NoopLogger()
	}
	if o.TTL < 0 {
		return nil, fmt.Errorf("pulse map: %s ttl must be >= 0", name)
	}
	closectx, closer := context.WithCancel(context.WithoutCancel(ctx))
	sm := &Map{
		Name:                 name,
		chankey:              fmt.Sprintf("map:%s:updates", name),
		hashkey:              fmt.Sprintf("map:%s:content", name),
		ttl:                  o.TTL,
		ttlSliding:           o.TTLSliding,
		done:                 make(chan struct{}),
		closectx:             closectx,
		closer:               closer,
		logger:               o.Logger.WithPrefix("map", name),
		rdb:                  rdb,
		content:              make(map[string]string),
		observations:         make(chan observationRequest),
		setScript:            luaSet,
		testAndSetScript:     luaTestAndSet,
		setIfNotExistsScript: luaSetIfNotExists,
		incrScript:           luaIncr,
		appendScript:         luaAppend,
		appendUniqueScript:   luaAppendUnique,
		removeScript:         luaRemove,
		delScript:            luaDelete,
		testAndDelScript:     luaTestAndDel,
		testAndResetScript:   luaTestAndReset,
		resetScript:          luaReset,
		destroyScript:        luaDestroy,
	}
	if err := sm.init(ctx); err != nil {
		closer()
		return nil, err
	}

	// read updates
	sm.wait.Add(1)
	pulse.Go(sm.logger, sm.run)

	sm.logger.Info("joined")
	return sm, nil
}

// Map returns a copy of the replicated map content.
func (sm *Map) Map() map[string]string {
	sm.lock.RLock()
	defer sm.lock.RUnlock()

	hash := make(map[string]string, len(sm.content))
	for k, v := range sm.content {
		hash[k] = v
	}
	return hash
}

// Keys returns a copy of the replicated map keys.
func (sm *Map) Keys() []string {
	sm.lock.RLock()
	defer sm.lock.RUnlock()

	keys := make([]string, 0, len(sm.content))
	for k := range sm.content {
		keys = append(keys, k)
	}
	return keys
}

// Subscribe returns a channel that receives notifications when the map
// changes. The channel is closed when the map is stopped. This channel simply
// notifies that the map has changed, it does not provide the actual changes,
// instead the Map method should be used to read the current content.  This
// allows the notification to be sent without blocking. Multiple remote updates
// may result in a single notification. Each subscriber retains its strongest
// unread kind: EventReset takes precedence over EventDelete, then EventChange.
// Subscribe returns nil if the map is stopped.
func (sm *Map) Subscribe() <-chan EventKind {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	if sm.lifetime.Stopping() {
		return nil
	}
	c := make(chan EventKind, 1) // Buffer 1 notification so we don't have to block.
	sm.chans = append(sm.chans, c)
	return c
}

// Unsubscribe removes the given channel from the list of subscribers and closes it.
func (sm *Map) Unsubscribe(c <-chan EventKind) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	if sm.lifetime.Stopping() {
		return
	}
	for i, ch := range sm.chans {
		if ch == c {
			close(sm.chans[i])
			sm.chans = append(sm.chans[:i], sm.chans[i+1:]...)
			return
		}
	}
}

// Len returns the number of items in the replicated map.
func (sm *Map) Len() int {
	sm.lock.RLock()
	defer sm.lock.RUnlock()
	return len(sm.content)
}

// Get returns the value for the given key.
func (sm *Map) Get(key string) (string, bool) {
	sm.lock.RLock()
	defer sm.lock.RUnlock()
	res, ok := sm.content[key]
	return res, ok
}

// GetValues returns the list values for the given key.
// This is a convenience method intended to be used in conjunction with
// AppendValues and RemoveValues. List values are stored as JSON arrays.
func (sm *Map) GetValues(key string) ([]string, bool) {
	val, ok := sm.Get(key)
	if !ok {
		return nil, false
	}
	return parseValues(val), true
}
