package streaming

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	redis "github.com/redis/go-redis/v9"

	"github.com/CaliLuke/loom/clue/log"
	"github.com/CaliLuke/loom/pulse/pulse"
	"github.com/CaliLuke/loom/pulse/rmap"
	"github.com/CaliLuke/loom/pulse/streaming/options"
)

var (
	// ensureConsumerRetryDelay is used when blockDuration means "block indefinitely".
	ensureConsumerRetryDelay = 100 * time.Millisecond
	// maxJitterMs is the maximum retry backoff jitter in milliseconds.
	maxJitterMs = 5000
	// checkIdlePeriod is the period at which idle messages are checked.
	checkIdlePeriod = 500 * time.Millisecond
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

type (
	// Sink represents a stream sink.
	Sink struct {
		// Name is the sink name.
		Name string
		// closed is true if Close completed.
		closed bool
		// consumer is the sink consumer name.
		consumer string
		// leaseKeyName is the stale check lock key name.
		leaseKeyName []string
		// startID is the sink start event ID.
		startID string
		// noAck is true if there is no need to acknowledge events.
		noAck bool
		// lock is the sink mutex.
		lock sync.Mutex
		// streams are the streams the sink consumes events from.
		streams []*Stream
		// streamCursors is the stream cursors used to read events in
		// the form [stream1, ">", stream2, ">", ...]
		streamCursors []string
		// blockDuration is the XREADGROUP timeout.
		blockDuration time.Duration
		// maxPolled is the maximum number of events to read in one
		// XREADGROUP call.
		maxPolled int64
		// bufferSize is the sink channel buffer size.
		bufferSize int
		// subscribers receive sink event notifications.
		subscribers []*eventSubscriber
		// donechan is the sink done channel.
		donechan chan struct{}
		// wait is the sink cleanup wait group.
		wait sync.WaitGroup
		// readCancel cancels the active Redis read call.
		readCancel context.CancelFunc
		// closing is true if Close was called.
		closing bool
		// eventFilter is the event filter if any.
		eventFilter eventFilterFunc
		// consumersMap are the replicated maps used to track sink
		// consumers.  Each map key is the sink name and the value is a list
		// of consumer names.  consumersMap is indexed by stream name.
		// consumer names are unique for each in-process sink instance.
		consumersMap map[string]*rmap.Map
		// consumersKeepAliveMap records consumer keep-alives for this
		// sink (i.e. for all in-process instances of the sink).
		consumersKeepAliveMap *rmap.Map
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
	}

	// eventFilterFunc is the function used to filter events.
	eventFilterFunc func(*Event) bool
)

// newSink creates a new sink.
// Sinks use one Redis consumer per stream they are consuming from.
// Pulse maintains a pool of consumers per stream and reuses them when possible.
// This is because deleting a consumer causes Redis to drop its pending messages
// which is not the semantics Pulse wants to enforce.
//
//nolint:maintidx // Sink setup validates options and initializes Redis-backed state together.
func newSink(ctx context.Context, name string, stream *Stream, opts ...options.Sink) (*Sink, error) {
	o := options.ParseSinkOptions(opts...)
	var eventMatcher eventFilterFunc
	if o.Topic != "" {
		eventMatcher = func(e *Event) bool { return e.Topic == o.Topic }
	} else if o.TopicPattern != "" {
		topicPatternRegexp, err := regexp.Compile(o.TopicPattern)
		if err != nil {
			return nil, fmt.Errorf("topic pattern must be a valid regex: %w", err)
		}
		eventMatcher = func(e *Event) bool { return topicPatternRegexp.MatchString(e.Topic) }
	}

	if err := acquireLeaseScript.Load(ctx, stream.rdb).Err(); err != nil {
		return nil, fmt.Errorf("failed to load stale check lease script: %w", err)
	}

	logger := stream.rootLogger.WithPrefix("sink", name)
	cm, err := rmap.Join(ctx, consumersMapName(stream), stream.rdb, consumersMapOptions(stream, logger)...)
	if err != nil {
		return nil, fmt.Errorf("failed to join replicated map for sink %s: %w", name, err)
	}
	km, err := rmap.Join(ctx, sinkKeepAliveMapName(name), stream.rdb, rmap.WithLogger(logger))
	if err != nil {
		return nil, fmt.Errorf("failed to join replicated map for sink keep-alives %s: %w", name, err)
	}

	if err := stream.rdb.XGroupCreateMkStream(ctx, stream.key, name, o.LastEventID).Err(); err != nil && !isBusyGroupErr(err) {
		return nil, fmt.Errorf("failed to create Redis consumer group %s for stream %s: %w", name, stream.Name, err)
	}
	if err := stream.applyTTL(ctx); err != nil {
		return nil, fmt.Errorf("failed to apply stream TTL: %w", err)
	}

	sink := &Sink{
		Name:                  name,
		leaseKeyName:          []string{staleLockName(name)},
		startID:               o.LastEventID,
		noAck:                 o.NoAck,
		streams:               []*Stream{stream},
		streamCursors:         []string{stream.key, ">"},
		blockDuration:         o.BlockDuration,
		maxPolled:             o.MaxPolled,
		bufferSize:            o.BufferSize,
		donechan:              make(chan struct{}),
		eventFilter:           eventMatcher,
		consumersMap:          map[string]*rmap.Map{stream.Name: cm},
		consumersKeepAliveMap: km,
		ackGracePeriod:        o.AckGracePeriod,
		acquireLease:          acquireLeaseScript,
		logger:                logger,
		rdb:                   stream.rdb,
	}

	// Clean up any existing stale consumers before creating our own
	if err := sink.deleteStreamStaleConsumers(ctx, stream); err != nil {
		sink.logger.Error(fmt.Errorf("failed to cleanup stale consumers: %w", err))
	}

	consumer, err := sink.newConsumer(ctx, stream)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}
	sink.consumer = consumer
	sink.logger = sink.logger.WithPrefix("consumer", consumer)

	// create new logger context for goroutines.
	logCtx := context.Background()
	logCtx = log.WithContext(logCtx, ctx)

	sink.wait.Add(3)
	pulse.Go(logger, func() { sink.read(logCtx) })
	pulse.Go(logger, sink.periodicKeepAlive)
	pulse.Go(logger, sink.periodicIdleMessageCheck)

	sink.logger.Info("created", "start", sink.startID, "stream", stream.Name, "max_polled", sink.maxPolled, "block_duration", sink.blockDuration, "buffer_size", sink.bufferSize, "no_ack", sink.noAck, "ack_grace_period", sink.ackGracePeriod)

	return sink, nil
}

// Subscribe returns a channel that receives events from the sink.
func (s *Sink) Subscribe() <-chan *Event {
	sub := newEventSubscriber(s.bufferSize)
	s.lock.Lock()
	defer s.lock.Unlock()
	s.subscribers = append(s.subscribers, sub)
	return sub.ch
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
func (s *Sink) AddStream(ctx context.Context, stream *Stream, opts ...options.AddStream) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	for _, s := range s.streams {
		if s.Name == stream.Name {
			return nil
		}
	}
	startID := s.startID
	options := options.ParseAddStreamOptions(opts...)
	for _, option := range opts {
		option(&options)
	}
	if options.LastEventID != "" {
		startID = options.LastEventID
	}

	cm, err := rmap.Join(ctx, consumersMapName(stream), stream.rdb, consumersMapOptions(stream, stream.logger)...)
	if err != nil {
		return fmt.Errorf("failed to join consumer replicated map for stream %s: %w", stream.Name, err)
	}
	if _, err := cm.AppendValues(ctx, s.Name, s.consumer); err != nil {
		return fmt.Errorf("failed to append consumer %s to replicated map for stream %s: %w", s.consumer, stream.Name, err)
	}
	if err := stream.rdb.XGroupCreateMkStream(ctx, stream.key, s.Name, startID).Err(); err != nil && !isBusyGroupErr(err) {
		return fmt.Errorf("failed to create Redis consumer group %s for stream %s: %w", s.Name, stream.Name, err)
	}
	s.streams = append(s.streams, stream)
	s.streamCursors = make([]string, len(s.streams)*2)
	for i, stream := range s.streams {
		s.streamCursors[i] = stream.key
		s.streamCursors[len(s.streams)+i] = ">"
	}
	s.consumersMap[stream.Name] = cm
	s.cancelActiveRead()
	s.logger.Info("added", "stream", stream.Name)
	return nil
}

// RemoveStream removes the stream from the sink, it is idempotent.
func (s *Sink) RemoveStream(ctx context.Context, stream *Stream) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	found := false
	for i, st := range s.streams {
		if st == stream {
			s.streams = append(s.streams[:i], s.streams[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return nil
	}
	s.streamCursors = make([]string, len(s.streams)*2)
	for i, stream := range s.streams {
		s.streamCursors[i] = stream.key
		s.streamCursors[len(s.streams)+i] = ">"
	}
	s.cancelActiveRead()
	if err := s.removeStreamConsumer(ctx, stream); err != nil {
		return err
	}
	s.logger.Info("removed", "stream", stream.Name)
	return nil
}

// Close stops event polling, waits for all events to be processed, and closes the sink channel.
// It is safe to call Close multiple times.
func (s *Sink) Close(ctx context.Context) {
	s.lock.Lock()
	if s.closing {
		s.lock.Unlock()
		return
	}
	s.closing = true
	close(s.donechan)
	if s.readCancel != nil {
		s.readCancel()
	}
	s.lock.Unlock()
	s.wait.Wait()
	s.lock.Lock()
	defer s.lock.Unlock()
	for _, sub := range s.subscribers {
		sub.close()
	}
	s.subscribers = nil
	// Note: we do not delete the consumer from the keep-alive and consumer maps
	// so that another instance may claim any pending messages.
	s.consumersKeepAliveMap.Close()
	for _, cm := range s.consumersMap {
		cm.Close()
	}
	s.closed = true
	s.logger.Info("closed")
}

// IsClosed returns true if the sink was closed.
func (s *Sink) IsClosed() bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.closed
}
