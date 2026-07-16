package streaming

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/pulse"
	"github.com/CaliLuke/loom/pulse/streaming/options"
)

func TestNewStreamBuildsLocalMetadataWithoutRedis(t *testing.T) {
	stream, err := NewStream("events", nil,
		options.WithStreamMaxLen(42),
		options.WithStreamSlidingTTL(30*time.Second),
	)

	require.NoError(t, err)
	require.Equal(t, "events", stream.Name)
	require.Equal(t, 42, stream.MaxLen)
	require.Equal(t, 30*time.Second, stream.ttl)
	require.True(t, stream.ttlSliding)
	require.Equal(t, streamKeyPrefix+"events", stream.key)
}

func TestNewStreamValidatesNameAndTTL(t *testing.T) {
	_, err := NewStream("bad name", nil)
	require.ErrorContains(t, err, "not a valid name")

	_, err = NewStream("events", nil, options.WithStreamTTL(-time.Second))
	require.ErrorContains(t, err, "ttl must be >= 0")
}

func TestStreamKeyValidation(t *testing.T) {
	require.True(t, isValidRedisKeyName("events:tenant-1"))
	require.False(t, isValidRedisKeyName(""))
	require.False(t, isValidRedisKeyName("events tenant"))
	require.False(t, isValidRedisKeyName(strings.Repeat("a", 513)))
}

func TestSinkNameHelpers(t *testing.T) {
	stream := &Stream{Name: "events"}

	require.Equal(t, "stream:events:sinks", consumersMapName(stream))
	require.Equal(t, "sink:orders:keepalive", sinkKeepAliveMapName("orders"))
	require.Equal(t, "sink:orders:stalelease", staleLockName("orders"))
}

func TestIsBusyGroupErr(t *testing.T) {
	require.False(t, isBusyGroupErr(nil))
	require.False(t, isBusyGroupErr(errors.New("some other redis error")))
	require.True(t, isBusyGroupErr(errors.New("BUSYGROUP Consumer Group name already exists")))
}

func TestReaderCloseIsIdempotentAndReleasesLock(t *testing.T) {
	reader := &Reader{
		donechan: make(chan struct{}),
		logger:   pulse.NoopLogger(),
	}

	reader.Close()
	reader.Close()

	require.Eventually(t, func() bool {
		return reader.IsClosed()
	}, 100*time.Millisecond, time.Millisecond)
}

func TestReaderFatalCloseSignalDoesNotWaitForReadGoroutine(t *testing.T) {
	reader := &Reader{
		donechan: make(chan struct{}),
		logger:   pulse.NoopLogger(),
	}
	reader.wait.Add(1)

	reader.closeFromReadLoop()
	reader.cleanup()

	require.True(t, reader.isClosing())
	require.True(t, reader.IsClosed())
}

func TestReaderDoesNotArmReadAfterCloseStarts(t *testing.T) {
	reader := &Reader{closing: true}

	readCtx, _, armed := reader.armRead(t.Context())

	require.False(t, armed)
	require.ErrorIs(t, readCtx.Err(), context.Canceled)
	require.Nil(t, reader.readCancel)
}

func TestSinkDoesNotArmReadAfterCloseStarts(t *testing.T) {
	sink := &Sink{closing: true}

	readCtx, _, _, armed := sink.armRead(t.Context())

	require.False(t, armed)
	require.ErrorIs(t, readCtx.Err(), context.Canceled)
	require.Nil(t, sink.readCancel)
}

func TestReaderAddStreamDeduplicatesByStreamKey(t *testing.T) {
	stream := &Stream{Name: "events", key: streamKeyPrefix + "events", rootLogger: pulse.NoopLogger()}
	reader, err := newReader(t.Context(), stream)
	require.NoError(t, err)

	require.NoError(t, reader.AddStream(t.Context(), stream))

	require.Len(t, reader.streams, 1)
	require.Equal(t, []string{stream.key}, reader.streamKeys)
}

func TestReaderAddStreamAfterCloseReturnsError(t *testing.T) {
	stream := &Stream{Name: "events", key: streamKeyPrefix + "events", rootLogger: pulse.NoopLogger()}
	reader, err := newReader(t.Context(), stream)
	require.NoError(t, err)
	reader.Close()

	require.ErrorContains(t, reader.AddStream(t.Context(), stream), "reader is closing")
}

func TestStreamEventsStopsWhenDoneClosesOnBackpressure(t *testing.T) {
	sub := newEventSubscriber(1)
	sub.ch <- &Event{}
	done := make(chan struct{})
	close(done)

	streamEvents(
		"events",
		streamKeyPrefix+"events",
		"",
		[]redis.XMessage{{
			ID: "1-0",
			Values: map[string]any{
				nameKey:    "created",
				payloadKey: "payload",
			},
		}},
		nil,
		[]*eventSubscriber{sub},
		nil,
		pulse.NoopLogger(),
		done,
	)
}

func TestStreamEventsDoesNotHoldReaderLockWhileSubscriberIsBackpressured(t *testing.T) {
	reader := &Reader{
		lock:        sync.Mutex{},
		subscribers: []*eventSubscriber{newEventSubscriber(1)},
		logger:      pulse.NoopLogger(),
		donechan:    make(chan struct{}),
	}
	reader.subscribers[0].ch <- &Event{}
	message := redis.XMessage{
		ID: "1-0",
		Values: map[string]any{
			nameKey:    "created",
			payloadKey: "payload",
		},
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		reader.fanOut(streamKeyPrefix+"events", []redis.XMessage{message})
	}()

	require.Eventually(t, func() bool {
		if reader.lock.TryLock() {
			reader.lock.Unlock()
			close(reader.donechan)
			return true
		}
		return false
	}, 100*time.Millisecond, time.Millisecond)
	require.Eventually(t, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, 100*time.Millisecond, time.Millisecond)
}

func TestReaderUnsubscribeDuringBackpressuredFanOutDoesNotPanic(t *testing.T) {
	reader := &Reader{
		subscribers: []*eventSubscriber{newEventSubscriber(1)},
		logger:      pulse.NoopLogger(),
		donechan:    make(chan struct{}),
	}
	reader.subscribers[0].ch <- &Event{}
	message := redis.XMessage{
		ID: "1-0",
		Values: map[string]any{
			nameKey:    "created",
			payloadKey: "payload",
		},
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		reader.fanOut(streamKeyPrefix+"events", []redis.XMessage{message})
	}()

	reader.Unsubscribe(reader.subscribers[0].ch)
	require.Eventually(t, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, 100*time.Millisecond, time.Millisecond)
}

func TestSinkUnsubscribeDuringBackpressuredFanOutDoesNotPanic(t *testing.T) {
	sink := &Sink{
		subscribers: []*eventSubscriber{newEventSubscriber(1)},
		logger:      pulse.NoopLogger(),
		donechan:    make(chan struct{}),
	}
	sink.subscribers[0].ch <- &Event{}
	message := redis.XMessage{
		ID: "1-0",
		Values: map[string]any{
			nameKey:    "created",
			payloadKey: "payload",
		},
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		subscribers, filter := sink.snapshotFanOut()
		streamEvents("events", streamKeyPrefix+"events", "sink", []redis.XMessage{message}, filter, subscribers, nil, pulse.NoopLogger(), sink.donechan)
	}()

	sink.Unsubscribe(sink.subscribers[0].ch)
	require.Eventually(t, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, 100*time.Millisecond, time.Millisecond)
}

func TestSinkRetryDelayHandlesZeroBlockDurationAndClosedSink(t *testing.T) {
	sink := &Sink{
		blockDuration: 0,
		donechan:      make(chan struct{}),
	}

	require.NotPanics(t, func() {
		require.True(t, sink.waitBeforeEnsureConsumerRetry())
	})

	close(sink.donechan)
	require.False(t, sink.waitBeforeEnsureConsumerRetry())
}

func TestHandleReadErrorStopsImmediatelyWhenDoneCloses(t *testing.T) {
	done := make(chan struct{})
	close(done)

	err := handleReadError(errors.New("temporary redis failure"), pulse.NoopLogger(), done)

	require.NoError(t, err)
}
