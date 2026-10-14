package streaming

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	redis "github.com/CaliLuke/go-redis/v9"

	"github.com/CaliLuke/loom/pulse/internal/lifecycle"
	"github.com/CaliLuke/loom/pulse/internal/redisop"
	"github.com/CaliLuke/loom/pulse/pulse"
	"github.com/CaliLuke/loom/pulse/streaming/options"
)

type (
	// Reader represents a stream reader.
	Reader struct {
		// closed is true if Close completed.
		closed bool
		// startID is the reader start event ID.
		startID string
		// lock is the reader mutex.
		lock sync.Mutex
		// registrations own each installed stream's identity and cursor.
		registrations []*readerRegistration
		// blockDuration is the XREADBLOCK timeout.
		blockDuration time.Duration
		// maxPolled is the maximum number of events to read in one
		// XREADBLOCK call.
		maxPolled int64
		// buffer size of the reader channel.
		bufferSize int
		// subscribers receive event notifications.
		subscribers []*eventSubscriber
		// startOnce is used to ensure the reader is started only once.
		startOnce sync.Once
		// donechan is the reader donechan channel.
		donechan chan struct{}
		// wait is the reader cleanup wait group.
		wait sync.WaitGroup
		// readCancel cancels the active Redis read call.
		readCancel context.CancelFunc
		// runtimeCtx owns the reader background loop.
		runtimeCtx context.Context
		// runtimeCancel stops the reader background loop.
		runtimeCancel context.CancelFunc
		// lifetime seals admission and joins cleanup across concurrent Close calls.
		lifetime lifecycle.Lifetime
		// eventFilter is the event filter if any.
		eventFilter eventFilterFunc
		// logger is the logger used by the reader.
		logger pulse.Logger
		// rdb is the redis connection.
		rdb *redis.Client
	}

	// Acker is the interface used by events to acknowledge themselves.
	Acker interface {
		XAck(ctx context.Context, streamKey, sinkName string, ids ...string) *redis.IntCmd
	}

	// readerRegistration identifies one installation, even if its Stream is
	// removed and later added again. cursor is protected by Reader.lock.
	readerRegistration struct {
		stream *Stream
		cursor string
	}

	// readerRead retains the registration identities and physical arguments
	// captured together at admission, independently of later membership changes.
	readerRead struct {
		registrations []*readerRegistration
		args          []string
	}

	readerBatch struct {
		registration *readerRegistration
		messages     []redis.XMessage
	}

	eventSubscriber struct {
		ch     chan *Event
		done   chan struct{}
		lock   sync.Mutex
		wait   sync.WaitGroup
		closed bool
	}

	// Event is a stream event.
	Event struct {
		// ID is the unique event ID.
		ID string
		// StreamName is the name of the stream the event belongs to.
		StreamName string
		// SinkName is the name of the sink the event belongs to.
		SinkName string
		// EventName is the producer-defined event name.
		EventName string
		// Topic is the producer-defined event topic if any, empty string if none.
		Topic string
		// Payload is the event payload.
		Payload []byte
		// Acker is the redis client used to acknowledge events.
		Acker Acker
		// streamKey is the Redis key of the stream.
		streamKey string
	}
)

// newReader creates a new reader.
func newReader(ctx context.Context, stream *Stream, opts ...options.Reader) (*Reader, error) {
	o := options.ParseReaderOptions(opts...)
	if err := validateConsumerBufferSize(o.BufferSize); err != nil {
		return nil, err
	}
	var eventFilter eventFilterFunc
	if o.Topic != "" {
		eventFilter = func(e *Event) bool {
			return e.Topic == o.Topic
		}
	} else if o.TopicPattern != "" {
		topicPatternRegexp, err := regexp.Compile(o.TopicPattern)
		if err != nil {
			return nil, fmt.Errorf("topic pattern must be a valid regex: %w", err)
		}
		eventFilter = func(e *Event) bool {
			return topicPatternRegexp.MatchString(e.Topic)
		}
	}

	cursor, err := prepareReaderCursor(ctx, stream.rdb, stream.key, o.LastEventID)
	if err != nil {
		return nil, err
	}
	runtimeCtx, runtimeCancel := context.WithCancel(context.WithoutCancel(ctx))
	reader := &Reader{
		startID:       o.LastEventID,
		registrations: []*readerRegistration{{stream: stream, cursor: cursor}},
		blockDuration: o.BlockDuration,
		maxPolled:     o.MaxPolled,
		bufferSize:    o.BufferSize,
		donechan:      make(chan struct{}),
		runtimeCtx:    runtimeCtx,
		runtimeCancel: runtimeCancel,
		eventFilter:   eventFilter,
		logger:        stream.rootLogger.WithPrefix("reader", stream.Name),
		rdb:           stream.rdb,
	}

	return reader, nil
}

// Subscribe returns a channel that receives events from the stream.
// The channel is closed when the reader is closed. If shutdown has started,
// Subscribe returns an already-closed channel without starting a reader.
func (r *Reader) Subscribe() <-chan *Event {
	r.lock.Lock()
	defer r.lock.Unlock()
	ch, admitted := registerSubscriber(&r.subscribers, r.bufferSize, r.lifetime.Stopping() || r.closed)
	if admitted {
		r.start()
	}
	return ch
}

// Unsubscribe removes the channel from the reader subscribers and closes it.
func (r *Reader) Unsubscribe(c <-chan *Event) {
	r.lock.Lock()
	var sub *eventSubscriber
	for i, candidate := range r.subscribers {
		if candidate.ch == c {
			sub = candidate
			r.subscribers = append(r.subscribers[:i], r.subscribers[i+1:]...)
			break
		}
	}
	r.lock.Unlock()
	if sub != nil {
		sub.close()
	}
}

// AddStream adds a stream using the reader's authored start position unless opts
// overrides it. Newest is resolved once when the stream is added. Explicit IDs
// are reused unchanged. Adding a handle with an already registered stream key
// is a no-op. A failed newest lookup leaves existing registrations and active
// reads unchanged.
func (r *Reader) AddStream(ctx context.Context, stream *Stream, opts ...options.AddStream) error {
	r.lock.Lock()
	allowed, err := r.canAddStream(stream.key)
	startID := r.startID
	r.lock.Unlock()
	if !allowed {
		return err
	}
	o := options.ParseAddStreamOptions(opts...)
	if o.LastEventID != "" {
		startID = o.LastEventID
	}
	cursor, err := prepareReaderCursor(ctx, r.rdb, stream.key, startID)
	if err != nil {
		return err
	}

	r.lock.Lock()
	defer r.lock.Unlock()
	allowed, err = r.canAddStream(stream.key)
	if !allowed {
		return err
	}
	r.registrations = append(r.registrations, &readerRegistration{stream: stream, cursor: cursor})
	r.cancelActiveRead()
	r.logger.Info("added", "stream", stream.Name)
	return nil
}

// RemoveStream removes the stream by its key; equivalent handles are interchangeable.
// Removal is idempotent. Events from an already-admitted read may still reach
// subscribers. If the stream is
// added again, those earlier reads cannot advance the new registration's cursor.
func (r *Reader) RemoveStream(ctx context.Context, stream *Stream) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.lifetime.Stopping() {
		return fmt.Errorf("reader is closing")
	}
	if i := r.registrationIndex(stream.key); i >= 0 {
		r.registrations = slices.Delete(r.registrations, i, i+1)
		r.cancelActiveRead()
	}
	r.logger.Info("removed", "stream", stream.Name)
	return nil
}

// Close stops event polling and closes the reader channel. It is safe to call
// Close multiple times.
func (r *Reader) Close() {
	r.lifetime.Close(r.stop, func() {
		// Synchronize with Subscribe admitting and starting its worker under lock
		// before Wait observes the count. Stop has already sealed new admission.
		r.lock.Lock()
		r.cancelActiveRead()
		r.lock.Unlock()
		r.wait.Wait()
		r.lock.Lock()
		defer r.lock.Unlock()
		r.closed = true
		r.logger.Info("stopped")
	})
}

func (r *Reader) stop() {
	close(r.donechan)
	if r.runtimeCancel != nil {
		r.runtimeCancel()
	}
}

func (r *Reader) closeFromReadLoop() {
	r.lifetime.Stop(r.stop)
}

// IsClosed returns true if the reader is stopped.
func (r *Reader) IsClosed() bool {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.closed
}

// start starts the reader's read goroutine if it is not already running.
func (r *Reader) start() {
	r.startOnce.Do(func() {
		r.wait.Add(1)
		pulse.Go(r.logger, func() {
			r.read(r.runtimeCtx)
		})
	})
}

// read reads events from the streams and sends them to the reader channel.
func (r *Reader) read(ctx context.Context) {
	defer r.cleanup()
	for {
		streamsEvents, err := r.xread(ctx)
		if r.isClosing() {
			return
		}
		if err != nil {
			if err := handleReadError(err, r.logger, r.donechan); err != nil {
				r.logger.Error(fmt.Errorf("fatal error while reading events: %w, stopping", err))
				r.closeFromReadLoop()
				return
			}
			continue
		}

		for _, batch := range streamsEvents {
			r.fanOut(ctx, batch.registration, batch.messages)
		}
	}
}

func (r *Reader) fanOut(ctx context.Context, registration *readerRegistration, messages []redis.XMessage) {
	if len(messages) == 0 {
		return
	}
	streamKey := registration.stream.key
	streamName := streamKey[len(streamKeyPrefix):]
	subscribers, filter := r.snapshotFanOut()
	streamEvents(ctx, streamName, streamKey, "", messages, filter, subscribers, r.rdb, r.logger, r.donechan)

	lastID := messages[len(messages)-1].ID
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.lifetime.Stopping() {
		return
	}
	if slices.Contains(r.registrations, registration) {
		registration.cursor = lastID
	}
}

func (r *Reader) snapshotFanOut() ([]*eventSubscriber, eventFilterFunc) {
	r.lock.Lock()
	defer r.lock.Unlock()
	subscribers := make([]*eventSubscriber, len(r.subscribers))
	copy(subscribers, r.subscribers)
	return subscribers, r.eventFilter
}

func (r *Reader) xread(ctx context.Context) ([]readerBatch, error) {
	readCtx, admitted, armed := r.armRead(ctx)
	if !armed {
		return nil, context.Canceled
	}
	defer r.clearActiveRead()

	r.logger.Debug("reading", "streams", admitted.args, "max", r.maxPolled, "block", r.blockDuration)
	client, block := redisop.Read(r.rdb, r.blockDuration)
	streams, err := client.XRead(readCtx, &redis.XReadArgs{
		Streams: admitted.args,
		Count:   r.maxPolled,
		Block:   block,
	}).Result()
	if err != nil {
		return nil, err
	}
	batches := make([]readerBatch, len(streams))
	for i, stream := range streams {
		index := slices.IndexFunc(admitted.registrations, func(registration *readerRegistration) bool {
			return registration.stream.key == stream.Stream
		})
		if index < 0 {
			return nil, fmt.Errorf("read returned unrequested stream %q", stream.Stream)
		}
		batches[i] = readerBatch{registration: admitted.registrations[index], messages: stream.Messages}
	}
	return batches, nil
}

func (r *Reader) armRead(ctx context.Context) (context.Context, readerRead, bool) {
	readCtx, cancel := context.WithCancel(ctx)
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.lifetime.Stopping() {
		cancel()
		return readCtx, readerRead{}, false
	}
	admitted := readerRead{
		registrations: slices.Clone(r.registrations),
		args:          make([]string, 2*len(r.registrations)),
	}
	for i, registration := range admitted.registrations {
		admitted.args[i] = registration.stream.key
		admitted.args[len(admitted.registrations)+i] = registration.cursor
	}
	r.readCancel = cancel
	return readCtx, admitted, true
}

func (r *Reader) cancelActiveRead() {
	if r.readCancel != nil {
		r.readCancel()
	}
}

func (r *Reader) clearActiveRead() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.readCancel = nil
}

// cleanup removes the consumer from the consumer groups and removes the reader
// from the readers map. This method is called automatically when the reader is
// stopped.
func (r *Reader) cleanup() {
	r.lock.Lock()
	subscribers := r.subscribers
	r.subscribers = nil
	r.closed = true
	r.lock.Unlock()
	for _, sub := range subscribers {
		sub.close()
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	r.wait.Done()
}

// isClosing returns true if the reader is stopping.
func (r *Reader) isClosing() bool {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.lifetime.Stopping()
}

// CreatedAt returns the event creation time (millisecond precision) encoded in
// the millisecond part of a Redis stream ID. It returns the Unix epoch when the
// ID does not start with a decimal millisecond timestamp.
func (e *Event) CreatedAt() time.Time {
	tss, _, _ := strings.Cut(e.ID, "-")
	ts, err := strconv.ParseInt(tss, 10, 64)
	if err != nil {
		ts = 0
	}
	seconds := ts / 1000
	nanos := (ts % 1000) * 1_000_000
	return time.Unix(seconds, nanos).UTC()
}

// streamEvents filters Redis messages and sends them to each subscriber.
// Malformed and filtered entries are skipped; in a sink consumer group they
// are acknowledged so they do not stay pending and get reclaimed forever.
func streamEvents(
	ctx context.Context,
	streamName string,
	streamKey string,
	sinkName string,
	msgs []redis.XMessage,
	eventFilter eventFilterFunc,
	subscribers []*eventSubscriber,
	rdb *redis.Client,
	logger pulse.Logger,
	done <-chan struct{},
) {
	if len(msgs) == 0 {
		return
	}
	for _, event := range msgs {
		ev, err := eventFromMessage(streamName, streamKey, sinkName, event, rdb)
		if err != nil {
			logger.Error(err, "id", event.ID, "stream", streamName)
			ackRejectedEvent(ctx, streamKey, sinkName, event.ID, rdb, logger)
			continue
		}
		if eventFilter != nil && !eventFilter(ev) {
			logger.Debug("event filtered", "event", ev.EventName, "id", ev.ID, "stream", streamName)
			ackRejectedEvent(ctx, streamKey, sinkName, event.ID, rdb, logger)
			continue
		}
		logger.Debug("event", "stream", streamName, "event", ev.EventName, "id", ev.ID, "channels", len(subscribers))
		for _, sub := range subscribers {
			if !sub.send(ev, done) {
				return
			}
		}
	}
}

// ackRejectedEvent completes a group-final rejection without deleting the stored
// entry. A failure before commit leaves the entry eligible for existing reclaim;
// a lost reply after commit has already cleared pending ownership.
func ackRejectedEvent(ctx context.Context, streamKey, sinkName, id string, rdb *redis.Client, logger pulse.Logger) {
	if sinkName == "" || rdb == nil {
		return
	}
	if err := redisop.Client(rdb).XAck(ctx, streamKey, sinkName, id).Err(); err != nil {
		logger.Error(fmt.Errorf("ack rejected stream %q entry %q: %w", streamKey, id, err))
	}
}

// eventFromMessage decodes one Redis stream entry written by Stream.Add. It
// returns an error instead of panicking when another writer added an entry
// without string name and payload fields or with a non-string topic.
func eventFromMessage(streamName, streamKey, sinkName string, msg redis.XMessage, acker Acker) (*Event, error) {
	name, ok := msg.Values[nameKey].(string)
	if !ok {
		return nil, fmt.Errorf("stream %q entry %q: event name field %q is %T, want string", streamName, msg.ID, nameKey, msg.Values[nameKey])
	}
	payload, ok := msg.Values[payloadKey].(string)
	if !ok {
		return nil, fmt.Errorf("stream %q entry %q: event payload field %q is %T, want string", streamName, msg.ID, payloadKey, msg.Values[payloadKey])
	}
	var topic string
	if value, present := msg.Values[topicKey]; present {
		if topic, ok = value.(string); !ok {
			return nil, fmt.Errorf("stream %q entry %q: event topic field %q is %T, want string", streamName, msg.ID, topicKey, value)
		}
	}
	return &Event{
		ID:         msg.ID,
		StreamName: streamName,
		SinkName:   sinkName,
		EventName:  name,
		Topic:      topic,
		Payload:    []byte(payload),
		streamKey:  streamKey,
		Acker:      acker,
	}, nil
}

// registerSubscriber admits a channel under the Reader or Sink lifecycle lock.
// Once shutdown begins, no channel is retained and no worker may be started.
func registerSubscriber(subscribers *[]*eventSubscriber, bufferSize int, stopped bool) (<-chan *Event, bool) {
	sub := newEventSubscriber(bufferSize)
	if stopped {
		sub.close()
		return sub.ch, false
	}
	*subscribers = append(*subscribers, sub)
	return sub.ch, true
}

func newEventSubscriber(bufferSize int) *eventSubscriber {
	return &eventSubscriber{
		ch:   make(chan *Event, bufferSize),
		done: make(chan struct{}),
	}
}

func (s *eventSubscriber) send(ev *Event, done <-chan struct{}) bool {
	if !s.beginSend() {
		return true
	}
	defer s.wait.Done()
	select {
	case s.ch <- ev:
		return true
	case <-s.done:
		return true
	case <-done:
		return false
	}
}

func (s *eventSubscriber) beginSend() bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.closed {
		return false
	}
	s.wait.Add(1)
	return true
}

func (s *eventSubscriber) close() {
	s.lock.Lock()
	if s.closed {
		s.lock.Unlock()
		return
	}
	s.closed = true
	close(s.done)
	s.lock.Unlock()
	s.wait.Wait()
	close(s.ch)
}

// handleReadError retries retryable read errors and ignores non-retryable.
func handleReadError(err error, logger pulse.Logger, done <-chan struct{}) error {
	if strings.Contains(err.Error(), "stream key no longer exists") {
		return err // Fatal error
	}
	if errors.Is(err, context.Canceled) {
		return nil // Read was interrupted by Close or a stream set change.
	}
	if errors.Is(err, redis.Nil) {
		return nil // No event at this time, just loop
	}
	if strings.Contains(err.Error(), "NOGROUP") {
		return nil // Consumer group was removed with RemoveStream, just loop (s.streamCursors will be updated)
	}
	// Retryable error, sleep and loop
	d := time.Duration(rand.Intn(maxJitterMs)) * time.Millisecond
	logger.Error(fmt.Errorf("failed to read events: %w, retrying in %v", err, d))
	select {
	case <-time.After(d):
	case <-done:
	}
	return nil
}
