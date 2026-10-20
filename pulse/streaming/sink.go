package streaming

import (
	"context"
	"fmt"
	"sync"
	"time"

	redis "github.com/CaliLuke/go-redis/v9"
	"github.com/oklog/ulid/v2"

	"github.com/CaliLuke/loom/clue/log"
	"github.com/CaliLuke/loom/pulse/internal/lifecycle"
	"github.com/CaliLuke/loom/pulse/internal/redisop"
	"github.com/CaliLuke/loom/pulse/pulse"
	"github.com/CaliLuke/loom/pulse/rmap"
	"github.com/CaliLuke/loom/pulse/streaming/options"
)

const defaultIdleCheckPeriod = 500 * time.Millisecond

var (
	// ensureConsumerRetryDelay is the minimum delay before retrying sink recovery.
	ensureConsumerRetryDelay = 100 * time.Millisecond
	// maxJitterMs is the maximum retry backoff jitter in milliseconds.
	maxJitterMs = 5000
)

// acquireLeaseScript is the script used to acquire the idle message check lease.
var acquireLeaseScript = redis.NewScript(`
    local key = KEYS[1]
    local new_value = ARGV[1]
    local current_time = ARGV[2]

    local current_value = redis.call("GET", key)
    if current_value == false or tonumber(current_value) < tonumber(current_time) then
        redis.call("SET", key, new_value, "PX", ARGV[3])
        return 1
    end
    return 0
`)

// deleteIdleConsumerScript deletes a consumer from a consumer group only if it
// owns no pending entries. KEYS[1] is the stream key, ARGV[1] the group and
// ARGV[2] the consumer. It returns 1 if the consumer was deleted, 0 otherwise.
var deleteIdleConsumerScript = redis.NewScript(`
    local pending = redis.call("XPENDING", KEYS[1], ARGV[1], "-", "+", 1, ARGV[2])
    if #pending > 0 then
        return 0
    end
    redis.call("XGROUP", "DELCONSUMER", KEYS[1], ARGV[1], ARGV[2])
    return 1
`)

// destroyGroupScript destroys the consumer group of a sink only if the
// replicated consumers map of the stream holds no consumer and the group has
// no pending entries. The checks and destroy run atomically, so a group that
// another instance joined or that acquired pending work during cleanup is kept.
// KEYS[1] is the hash key of the consumers map and KEYS[2] the stream key.
// ARGV[1] is the sink name, which is both the map key and the group name.
// ARGV[2] is the validated empty membership record. The script compares it
// again before destruction, retaining identity. It returns 1 if destroyed.
var destroyGroupScript = redis.NewScript(`
    if redis.call("HGET", KEYS[1], ARGV[1]) ~= ARGV[2] then
        return 0
    end
    local pending = redis.pcall("XPENDING", KEYS[2], ARGV[1])
    if pending.err then
        if string.sub(pending.err, 1, 7) == "NOGROUP" then
            return 0
        end
        return redis.error_reply(pending.err)
    end
    if pending[1] > 0 then
        return 0
    end
    return redis.call("XGROUP", "DESTROY", KEYS[2], ARGV[1])
`)

type (
	sinkRuntime struct {
		idleCheckPeriod time.Duration
		idleChecks      <-chan time.Time
	}

	// Sink represents a stream sink.
	Sink struct {
		// Name is the sink name.
		Name string
		// closed is true if Close completed.
		closed bool
		// consumer is the sink consumer name.
		consumer string
		// retiredConsumers are owned names awaiting removal from every stream map.
		retiredConsumers []string
		// leaseKeyName is the stale check lock key name.
		leaseKeyName []string
		// startID is the sink start event ID.
		startID string
		// noAck is true if there is no need to acknowledge events.
		noAck bool
		// lock is the sink mutex.
		lock sync.Mutex
		// streams own active stream identities, group starts and unfinished recovery.
		streams []sinkRegistration
		// blockDuration is the XREADGROUP timeout.
		blockDuration time.Duration
		// maxPolled is the maximum number of events to read in one
		// XREADGROUP call.
		maxPolled int64
		// bufferSize is the sink channel buffer size.
		bufferSize int
		// subscribers receive sink event notifications.
		subscribers []*eventSubscriber
		// deliveryStarted records first-subscriber activation under lock.
		deliveryStarted bool
		// donechan is the sink done channel.
		donechan chan struct{}
		// wait is the sink cleanup wait group.
		wait sync.WaitGroup
		// readCancel cancels the active Redis read call.
		readCancel context.CancelFunc
		// runtimeCtx owns all background Redis operations.
		runtimeCtx context.Context
		// runtimeCancel stops all background Redis operations.
		runtimeCancel context.CancelFunc
		// lifetime seals admission and joins cleanup across concurrent Close calls.
		lifetime lifecycle.Lifetime
		// eventFilter is the event filter if any.
		eventFilter eventFilterFunc
		// filterIdentity is the admitted effective group filter.
		filterIdentity sinkFilter
		// consumersMap are the replicated maps used to track sink
		// consumers. Each map key is the sink name and the value is a versioned
		// filter identity and consumer set.  consumersMap is indexed by stream name.
		// consumer names are unique for each in-process sink instance. Maps
		// also retain ownership of group cleanup after failed RemoveStream calls.
		consumersMap map[string]*sinkConsumerMap
		// ackGracePeriod is the grace period after which an event is
		// considered unacknowledged.
		ackGracePeriod time.Duration
		// lastKeepAlive is the last keep-alive timestamp for this consumer.
		lastKeepAlive int64
		// logger is the logger used by the sink.
		logger pulse.Logger
		// acquireLease is the acquire lease script.
		acquireLease *redis.Script
		// rdb is the redis connection.
		rdb *redis.Client
		// idleCheckPeriod controls the stale-consumer lease interval.
		idleCheckPeriod time.Duration
		// idleChecks triggers stale-consumer checks when provided by tests.
		idleChecks <-chan time.Time
	}

	// eventFilterFunc is the function used to filter events.
	eventFilterFunc func(*Event) bool
)

// newSink creates a new sink.
// Sinks use one Redis consumer per stream they are consuming from.
// Pulse maintains a pool of consumers per stream and reuses them when possible.
// This is because deleting a consumer causes Redis to drop its pending messages
// which is not the semantics Pulse wants to enforce.
// The consumer goes to the replicated consumers map of the stream before the
// consumer group and the consumer are created, so a concurrent RemoveStream
// of another instance of the sink sees it and keeps the group
// (tla/SinkAddStream.tla). If newSink fails, the map does not hold the
// consumer.
//
//nolint:maintidx // Sink setup validates options and initializes Redis-backed state together.
func newSink(ctx context.Context, name string, stream *Stream, runtime sinkRuntime, opts ...options.Sink) (*Sink, error) {
	o := options.ParseSinkOptions(opts...)
	if err := validateConsumerBufferSize(o.BufferSize); err != nil {
		return nil, err
	}
	if o.AckGracePeriod <= 0 {
		return nil, fmt.Errorf("sink ack grace period must be > 0, got %s", o.AckGracePeriod)
	}
	filter, eventMatcher, err := newSinkFilter(o.Topic, o.TopicPattern)
	if err != nil {
		return nil, err
	}

	if err := acquireLeaseScript.Load(ctx, redisop.Client(stream.rdb)).Err(); err != nil {
		return nil, fmt.Errorf("failed to load stale check lease script: %w", err)
	}

	logger := stream.rootLogger.WithPrefix("sink", name)
	replica, err := rmap.Join(ctx, consumersMapName(stream), stream.rdb, rmap.WithLogger(logger))
	if err != nil {
		return nil, fmt.Errorf("failed to join replicated map for sink %s: %w", name, err)
	}
	cm := &sinkConsumerMap{stream: stream, replica: replica}
	consumer := ulid.Make().String()
	if _, err := changeSinkMembership(ctx, cm, name, filter, sinkMemberChange{add: consumer}); err != nil {
		err = rollbackConsumer(ctx, cm, name, consumer, filter, err)
		cm.replica.Close()
		return nil, fmt.Errorf("failed to register consumer %s in replicated map for stream %s: %w", consumer, stream.Name, err)
	}
	if err := stream.createGroup(ctx, name, o.LastEventID); err != nil {
		err = fmt.Errorf("failed to create Redis consumer group %s for stream %s: %w", name, stream.Name, err)
		err = rollbackConsumer(ctx, cm, name, consumer, filter, err)
		cm.replica.Close()
		return nil, err
	}

	runtimeBase := log.WithContext(context.WithoutCancel(ctx), ctx)
	runtimeCtx, runtimeCancel := context.WithCancel(runtimeBase)
	sink := &Sink{
		Name:            name,
		consumer:        consumer,
		leaseKeyName:    []string{staleLockName(name)},
		startID:         o.LastEventID,
		noAck:           o.NoAck,
		streams:         []sinkRegistration{{stream: stream, startID: o.LastEventID}},
		blockDuration:   o.BlockDuration,
		maxPolled:       o.MaxPolled,
		bufferSize:      o.BufferSize,
		donechan:        make(chan struct{}),
		eventFilter:     eventMatcher,
		filterIdentity:  filter,
		consumersMap:    map[string]*sinkConsumerMap{stream.Name: cm},
		ackGracePeriod:  o.AckGracePeriod,
		acquireLease:    acquireLeaseScript,
		logger:          logger,
		rdb:             stream.rdb,
		runtimeCtx:      runtimeCtx,
		runtimeCancel:   runtimeCancel,
		idleCheckPeriod: runtime.idleCheckPeriod,
		idleChecks:      runtime.idleChecks,
	}
	if sink.idleCheckPeriod <= 0 {
		sink.idleCheckPeriod = defaultIdleCheckPeriod
	}

	// Clean up any existing stale consumers before creating our own
	if err := sink.deleteStreamStaleConsumers(ctx, stream); err != nil {
		sink.logger.Error(fmt.Errorf("failed to cleanup stale consumers: %w", err))
	}

	if err := sink.createConsumer(ctx, stream, consumer); err != nil {
		err = rollbackConsumer(ctx, cm, name, consumer, filter, fmt.Errorf("failed to create consumer: %w", err))
		runtimeCancel()
		cm.replica.Close()
		return nil, err
	}
	sink.logger = sink.logger.WithPrefix("consumer", consumer)

	sink.wait.Add(1)
	pulse.Go(logger, func() {
		sink.periodicKeepAlive(runtimeCtx)
	})

	sink.logger.Info("created", "start", sink.startID, "stream", stream.Name, "max_polled", sink.maxPolled, "block_duration", sink.blockDuration, "buffer_size", sink.bufferSize, "no_ack", sink.noAck, "ack_grace_period", sink.ackGracePeriod)

	return sink, nil
}

// Subscribe returns a channel that receives events from the sink. The first
// successful subscription starts reading and claiming pending events. Delivery
// continues if all subscribers later unsubscribe. If shutdown has started,
// Subscribe returns an already-closed channel.
func (s *Sink) Subscribe() <-chan *Event {
	s.lock.Lock()
	defer s.lock.Unlock()
	ch, admitted := registerSubscriber(&s.subscribers, s.bufferSize, s.lifetime.Stopping() || s.closed)
	if admitted && !s.deliveryStarted {
		s.deliveryStarted = true
		s.wait.Add(2)
		pulse.Go(s.logger, func() {
			s.read(s.runtimeCtx)
		})
		pulse.Go(s.logger, func() {
			s.periodicIdleMessageCheck(s.runtimeCtx)
		})
	}
	return ch
}

// Unsubscribe removes the channel from the sink and closes it.
func (s *Sink) Unsubscribe(c <-chan *Event) {
	s.lock.Lock()
	var sub *eventSubscriber
	for i, candidate := range s.subscribers {
		if candidate.ch == c {
			sub = candidate
			s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
			break
		}
	}
	s.lock.Unlock()
	if sub != nil {
		sub.close()
	}
}

// Ack acknowledges the event.
func (s *Sink) Ack(ctx context.Context, e *Event) error {
	err := e.Acker.XAck(ctx, e.streamKey, e.SinkName, e.ID).Err()
	if err != nil {
		s.logger.Error(err, "ack", e.ID, "stream", e.StreamName)
		return err
	}
	s.logger.Debug("acked", "event", e.ID, "stream", e.StreamName, "from-sink", e.SinkName)
	return nil
}

// AddStream adds the stream to the sink. By default the stream cursor starts at
// the same timestamp as the sink main stream cursor.  This can be overridden
// with opts. AddStream does nothing if the stream is already part of the sink.
// It creates the stream if missing and applies the stream TTL in the same
// script as the consumer group, as NewSink does. If AddStream fails, the
// replicated consumers map of the stream does not hold the sink consumer.
func (s *Sink) AddStream(ctx context.Context, stream *Stream, opts ...options.AddStream) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.lifetime.Stopping() {
		return fmt.Errorf("sink is closing")
	}
	for _, registration := range s.streams {
		if registration.stream.Name == stream.Name {
			return nil
		}
	}
	startID := s.startID
	addStreamOptions := options.ParseAddStreamOptions(opts...)
	if addStreamOptions.LastEventID != "" {
		startID = addStreamOptions.LastEventID
	}

	cm := s.consumersMap[stream.Name]
	joined := cm == nil
	if joined {
		var err error
		var replica *rmap.Map
		replica, err = rmap.Join(ctx, consumersMapName(stream), stream.rdb, rmap.WithLogger(stream.logger))
		if err != nil {
			return fmt.Errorf("failed to join consumer replicated map for stream %s: %w", stream.Name, err)
		}
		cm = &sinkConsumerMap{stream: stream, replica: replica}
	}
	// The consumer goes to the map before the group is created, so a
	// concurrent RemoveStream of another instance of the sink sees it and
	// keeps the group (tla/SinkAddStream.tla).
	if _, err := changeSinkMembership(ctx, cm, s.Name, s.filterIdentity, sinkMemberChange{add: s.consumer}); err != nil {
		err = rollbackConsumer(ctx, cm, s.Name, s.consumer, s.filterIdentity, err)
		if joined {
			cm.replica.Close()
		}
		return fmt.Errorf("failed to register consumer %s in replicated map for stream %s: %w", s.consumer, stream.Name, err)
	}
	if err := stream.createGroup(ctx, s.Name, startID); err != nil {
		err = fmt.Errorf("failed to create Redis consumer group %s for stream %s: %w", s.Name, stream.Name, err)
		err = rollbackConsumer(ctx, cm, s.Name, s.consumer, s.filterIdentity, err)
		if joined {
			cm.replica.Close()
		}
		return err
	}
	s.streams = append(s.streams, sinkRegistration{stream: stream, startID: startID})
	s.consumersMap[stream.Name] = cm
	s.cancelActiveRead()
	s.logger.Info("added", "stream", stream.Name)
	return nil
}

// RemoveStream removes the stream from the sink, it is idempotent. It removes
// the sink consumer from the replicated consumers map of the stream and, when
// no consumer remains and no pending entries remain, destroys the consumer
// group. A group with pending entries is kept so a later sink can claim them.
// The group is also kept if another instance added the stream in the meantime.
// If map removal fails, the local stream is retained. If group cleanup fails
// after deregistration, polling stops but cleanup remains owned: retry
// RemoveStream to finish, or AddStream to resume using the stream.
func (s *Sink) RemoveStream(ctx context.Context, stream *Stream) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.lifetime.Stopping() {
		return fmt.Errorf("sink is closing")
	}
	cm := s.consumersMap[stream.Name]
	if cm == nil {
		return nil
	}
	removed, err := s.removeStreamConsumer(ctx, stream)
	if !removed {
		return err
	}
	for i, st := range s.streams {
		if st.stream.Name == stream.Name {
			s.streams = append(s.streams[:i], s.streams[i+1:]...)
			break
		}
	}
	s.cancelActiveRead()
	if err != nil {
		// Keep the map owned so RemoveStream can retry group cleanup,
		// AddStream can reuse it, and Close can always release it.
		return err
	}
	cm.replica.Close()
	delete(s.consumersMap, stream.Name)
	s.logger.Info("removed", "stream", stream.Name)
	return nil
}

// Close stops event polling, waits for all events to be processed, and closes the sink channel.
// It is safe to call Close multiple times.
func (s *Sink) Close(ctx context.Context) {
	s.lifetime.Close(func() {
		close(s.donechan)
		if s.runtimeCancel != nil {
			s.runtimeCancel()
		}
	}, func() {
		s.lock.Lock()
		s.cancelActiveRead()
		s.lock.Unlock()
		s.wait.Wait()
		s.lock.Lock()
		defer s.lock.Unlock()
		for _, sub := range s.subscribers {
			sub.close()
		}
		s.subscribers = nil
		for _, cm := range s.consumersMap {
			cm.replica.Close()
		}
		s.closed = true
		s.logger.Info("closed")
	})
}

// IsClosed returns true if the sink was closed.
func (s *Sink) IsClosed() bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.closed
}
