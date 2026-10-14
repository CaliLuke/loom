package streaming

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CaliLuke/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming/options"
)

type (
	subscriberOwner struct {
		stream      *Stream
		subscribe   func() <-chan *Event
		unsubscribe func(<-chan *Event)
		close       func()
		closing     func() bool
		count       func() int
	}
	subscriberReadGate struct {
		entered chan struct{}
		release chan struct{}
		once    sync.Once
		calls   atomic.Int32
	}
)

func TestSubscriberAdmissionAfterClose(t *testing.T) {
	for _, kind := range []string{"reader-unused", "reader-started", "sink-unused", "sink-started"} {
		t.Run(kind, func(t *testing.T) {
			rdb := startTestRedis(t)
			gate := &subscriberReadGate{}
			rdb.AddHook(gate)
			owner := newSubscriberOwner(t, rdb, kind)
			if kind == "reader-started" || kind == "sink-started" {
				owner.subscribe()
			}
			owner.close()
			before := gate.calls.Load()
			late := owner.subscribe()
			assertSubscriberClosed(t, late)
			require.Zero(t, owner.count(), "late registration must not retain a channel")
			owner.unsubscribe(late)
			require.Equal(t, before, gate.calls.Load(), "late registration must not start a reader")
		})
	}
}

func TestSubscriberAdmissionDoesNotStartClosedReader(t *testing.T) {
	stream := newTestStream(t, startTestRedis(t), "unused-reader")
	reader, err := stream.NewReader(t.Context())
	require.NoError(t, err)
	reader.Close()
	assertSubscriberClosed(t, reader.Subscribe())
	reader.wait.Wait() // Join an incorrectly admitted parent worker in the negative control.
	// Claiming an unused startup token after Close is safe and proves Subscribe
	// never admitted a worker, even if an incorrectly started worker exits before
	// this assertion and never reaches its Redis command.
	unused := false
	reader.startOnce.Do(func() {
		unused = true
	})
	require.True(t, unused, "Subscribe consumed the worker startup token after Close")
}

func TestSubscriberAdmissionWhileClosing(t *testing.T) {
	for _, kind := range []string{"reader", "sink"} {
		t.Run(kind, func(t *testing.T) {
			rdb := startTestRedis(t)
			gate := &subscriberReadGate{entered: make(chan struct{}), release: make(chan struct{})}
			rdb.AddHook(gate)
			owner := newSubscriberOwner(t, rdb, kind)
			first := owner.subscribe()
			select {
			case <-gate.entered:
			case <-time.After(time.Second):
				t.Fatal("worker did not reach read gate")
			}
			done := make(chan struct{})
			go func() {
				owner.close()
				close(done)
			}()
			var releaseOnce sync.Once
			release := func() {
				releaseOnce.Do(func() {
					close(gate.release)
				})
			}
			t.Cleanup(func() {
				release()
				select {
				case <-done:
				case <-time.After(time.Second):
					t.Error("Close did not join released worker")
				}
			})
			require.Eventually(t, owner.closing, time.Second, time.Millisecond)
			before := owner.count()
			late := owner.subscribe()
			assertSubscriberClosed(t, late)
			require.Equal(t, before, owner.count())
			release()
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("Close did not join released worker")
			}
			assertSubscriberClosed(t, first)
		})
	}
}

func TestSubscriberAdmissionConcurrentClose(t *testing.T) {
	for _, kind := range []string{"reader", "sink"} {
		t.Run(kind, func(t *testing.T) {
			owner := newSubscriberOwner(t, startTestRedis(t), kind)
			start := make(chan struct{})
			var wait sync.WaitGroup
			channels := make(chan (<-chan *Event), 24)
			for range 24 {
				wait.Go(func() {
					<-start
					channels <- owner.subscribe()
				})
			}
			wait.Go(func() {
				<-start
				owner.close()
			})
			close(start)
			wait.Wait()
			close(channels)
			for channel := range channels {
				assertSubscriberClosed(t, channel)
			}
			require.Zero(t, owner.count())
		})
	}
}

func TestSubscriberAdmissionDelivery(t *testing.T) {
	for _, kind := range []string{"reader", "sink"} {
		t.Run(kind, func(t *testing.T) {
			owner := newSubscriberOwner(t, startTestRedis(t), kind)
			for _, name := range []string{"first", "second"} {
				ch := owner.subscribe()
				id, err := owner.stream.Add(t.Context(), name, []byte("payload"))
				require.NoError(t, err)
				select {
				case event, ok := <-ch:
					require.True(t, ok)
					require.Equal(t, id, event.ID)
					require.Equal(t, name, event.EventName)
					require.Equal(t, []byte("payload"), event.Payload)
				case <-time.After(time.Second):
					t.Fatal("admitted subscriber received no event")
				}
				owner.unsubscribe(ch)
				assertSubscriberClosed(t, ch)
			}
		})
	}
}

func newSubscriberOwner(t *testing.T, rdb *redis.Client, kind string) subscriberOwner {
	t.Helper()
	stream := newTestStream(t, rdb, "subscriber-admission")
	if kind == "sink" || kind == "sink-unused" || kind == "sink-started" {
		sink := newTestSink(t, stream, "subscriber-admission")
		return subscriberOwner{
			stream:      stream,
			subscribe:   sink.Subscribe,
			unsubscribe: sink.Unsubscribe,
			close: func() {
				sink.Close(context.Background())
			},
			closing: func() bool {
				sink.lock.Lock()
				defer sink.lock.Unlock()
				return sink.lifetime.Stopping()
			},
			count: func() int {
				sink.lock.Lock()
				defer sink.lock.Unlock()
				return len(sink.subscribers)
			},
		}
	}
	reader, err := stream.NewReader(t.Context(), options.WithReaderBlockDuration(50*time.Millisecond))
	require.NoError(t, err)
	t.Cleanup(reader.Close)
	return subscriberOwner{
		stream:      stream,
		subscribe:   reader.Subscribe,
		unsubscribe: reader.Unsubscribe,
		close:       reader.Close,
		closing:     reader.isClosing,
		count: func() int {
			reader.lock.Lock()
			defer reader.lock.Unlock()
			return len(reader.subscribers)
		},
	}
}

func assertSubscriberClosed(t *testing.T, ch <-chan *Event) {
	t.Helper()
	select {
	case event, ok := <-ch:
		require.False(t, ok, "late channel must already be closed; got %v", event)
	default:
		t.Error("subscriber channel is still open")
	}
}

func (h *subscriberReadGate) DialHook(next redis.DialHook) redis.DialHook {
	return next
}
func (h *subscriberReadGate) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if cmd.Name() == "xread" || cmd.Name() == "xreadgroup" {
			h.calls.Add(1)
			if h.entered != nil {
				h.once.Do(func() {
					close(h.entered)
				})
				<-h.release
			}
		}
		return next(ctx, cmd)
	}
}
func (h *subscriberReadGate) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
