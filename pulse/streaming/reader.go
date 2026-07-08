package streaming

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	redis "github.com/redis/go-redis/v9"

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
		// streams are the streams the reader consumes events from.
		streams []*Stream
		// streamKeys is the stream names used to read events in
		// the same order as streamCursors
		streamKeys []string
		// streamCursors is the stream cursors used to read events in
		// the same order as streamNames
		streamCursors []string
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
		// closing is true if Close was called.
		closing bool
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
func newReader(stream *Stream, opts ...options.Reader) (*Reader, error) {
	o := options.ParseReaderOptions(opts...)
	var eventFilter eventFilterFunc
	if o.Topic != "" {
		eventFilter = func(e *Event) bool { return e.Topic == o.Topic }
	} else if o.TopicPattern != "" {
		topicPatternRegexp, err := regexp.Compile(o.TopicPattern)
		if err != nil {
			return nil, fmt.Errorf("topic pattern must be a valid regex: %w", err)
		}
		eventFilter = func(e *Event) bool { return topicPatternRegexp.MatchString(e.Topic) }
	}

	reader := &Reader{
		startID:       o.LastEventID,
		streams:       []*Stream{stream},
		streamKeys:    []string{stream.key},
		streamCursors: []string{o.LastEventID},
		blockDuration: o.BlockDuration,
		maxPolled:     o.MaxPolled,
		bufferSize:    o.BufferSize,
		donechan:      make(chan struct{}),
		eventFilter:   eventFilter,
		logger:        stream.rootLogger.WithPrefix("reader", stream.Name),
		rdb:           stream.rdb,
	}

	return reader, nil
}

// Subscribe returns a channel that receives events from the stream.
// The channel is closed when the reader is closed.
func (r *Reader) Subscribe() <-chan *Event {
	sub := newEventSubscriber(r.bufferSize)
	r.lock.Lock()
	defer r.lock.Unlock()
	r.subscribers = append(r.subscribers, sub)
	r.start()
	return sub.ch
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

// AddStream adds the stream to the sink. By default the stream cursor starts at
// the same timestamp as the sink main stream cursor.  This can be overridden
// with opts. AddStream does nothing if the stream is already part of the sink.
func (r *Reader) AddStream(ctx context.Context, stream *Stream, opts ...options.AddStream) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.closing {
		return fmt.Errorf("reader is closing")
	}
	for _, key := range r.streamKeys {
		if key == stream.key {
			return nil
		}
	}
	startID := r.startID
	o := options.ParseAddStreamOptions(opts...)
	if o.LastEventID != "" {
		startID = o.LastEventID
	}
	r.streams = append(r.streams, stream)
	r.streamKeys = append(r.streamKeys, stream.key)
	r.streamCursors = append(r.streamCursors, startID)
	r.cancelActiveRead()
	r.logger.Info("added", "stream", stream.Name)
	return nil
}

// RemoveStream removes the stream from the sink, it is idempotent.
func (r *Reader) RemoveStream(ctx context.Context, stream *Stream) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.closing {
		return fmt.Errorf("reader is closing")
	}
	for i, st := range r.streams {
		if st == stream {
			r.streams = append(r.streams[:i], r.streams[i+1:]...)
			r.streamKeys = append(r.streamKeys[:i], r.streamKeys[i+1:]...)
			r.streamCursors = append(r.streamCursors[:i], r.streamCursors[i+1:]...)
			r.cancelActiveRead()
			break
		}
	}
	r.logger.Info("removed", "stream", stream.Name)
	return nil
}

// Close stops event polling and closes the reader channel. It is safe to call
// Close multiple times.
func (r *Reader) Close() {
	if !r.beginClose() {
		return
	}
	r.wait.Wait()
	r.lock.Lock()
	defer r.lock.Unlock()
	r.closed = true
	r.logger.Info("stopped")
}

func (r *Reader) beginClose() bool {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.closing {
		return false
	}
	r.closing = true
	close(r.donechan)
	if r.readCancel != nil {
		r.readCancel()
	}
	return true
}

func (r *Reader) closeFromReadLoop() {
	r.beginClose()
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
		pulse.Go(r.logger, func() { r.read(context.Background()) })
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

		r.fanOutStreams(streamsEvents)
	}
}

func (r *Reader) fanOutStreams(streamsEvents []redis.XStream) {
	for _, events := range streamsEvents {
		r.fanOut(events.Stream, events.Messages)
	}
}

func (r *Reader) fanOut(streamKey string, messages []redis.XMessage) {
	if len(messages) == 0 {
		return
	}
	streamName := streamKey[len(streamKeyPrefix):]
	subscribers, filter := r.snapshotFanOut()
	streamEvents(streamName, streamKey, "", messages, filter, subscribers, r.rdb, r.logger, r.donechan)

	lastID := messages[len(messages)-1].ID
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.closing {
		return
	}
	for i := range r.streamKeys {
		if r.streamKeys[i] == streamKey {
			r.streamCursors[i] = lastID
			break
		}
	}
}

func (r *Reader) snapshotFanOut() ([]*eventSubscriber, eventFilterFunc) {
	r.lock.Lock()
	defer r.lock.Unlock()
	subscribers := make([]*eventSubscriber, len(r.subscribers))
	copy(subscribers, r.subscribers)
	return subscribers, r.eventFilter
}

func (r *Reader) xread(ctx context.Context) ([]redis.XStream, error) {
	// copy so no two goroutines can share the memory
	readCtx, cancel := context.WithCancel(ctx)
	r.lock.Lock()
	readStreams := make([]string, 0, len(r.streamKeys)+len(r.streamCursors))
	readStreams = append(readStreams, r.streamKeys...)
	readStreams = append(readStreams, r.streamCursors...)
	r.readCancel = cancel
	r.lock.Unlock()
	defer r.clearActiveRead()

	r.logger.Debug("reading", "streams", readStreams, "max", r.maxPolled, "block", r.blockDuration)
	return r.rdb.XRead(readCtx, &redis.XReadArgs{
		Streams: readStreams,
		Count:   r.maxPolled,
		Block:   r.blockDuration,
	}).Result()
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
	return r.closing
}

// CreatedAt returns the event creation time (millisecond precision).
func (e *Event) CreatedAt() time.Time {
	tss := e.ID[:strings.IndexByte(e.ID, '-')]
	ts, _ := strconv.ParseInt(tss, 10, 64)
	seconds := ts / 1000
	nanos := (ts % 1000) * 1_000_000
	return time.Unix(seconds, nanos).UTC()
}

// streamEvents filters Redis messages and sends them to each subscriber.
func streamEvents(
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
		var topic string
		if t, ok := event.Values[topicKey]; ok {
			topic = t.(string)
		}
		ev := &Event{
			ID:         event.ID,
			StreamName: streamName,
			SinkName:   sinkName,
			EventName:  event.Values[nameKey].(string),
			Topic:      topic,
			Payload:    []byte(event.Values[payloadKey].(string)),
			streamKey:  streamKey,
			Acker:      rdb,
		}
		if eventFilter != nil && !eventFilter(ev) {
			logger.Debug("event filtered", "event", ev.EventName, "id", ev.ID, "stream", streamName)
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
