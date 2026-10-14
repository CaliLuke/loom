package streaming

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CaliLuke/go-redis/v9"
	"github.com/CaliLuke/loom/pulse/streaming/options"
	"github.com/stretchr/testify/require"
)

type registrationBoundaryHook struct {
	reads   atomic.Int32
	first   chan struct{}
	second  chan struct{}
	release chan struct{}
	once    sync.Once
}

func TestReaderRegistrationBoundaries(t *testing.T) {
	for _, stage := range []string{"reply", "fanout"} {
		for _, action := range []string{"explicit", "newest", "unsubscribe", "unchanged", "remove", "close"} {
			t.Run(stage+"/"+action, func(t *testing.T) {
				rdb := startTestRedis(t)
				hook := &registrationBoundaryHook{first: make(chan struct{}), second: make(chan struct{}), release: make(chan struct{})}
				if stage == "fanout" {
					hook.unblock()
				}
				rdb.AddHook(hook)
				stream := newTestStream(t, rdb, "registration-boundary")
				firstID, err := stream.Add(t.Context(), "first", []byte("one"))
				require.NoError(t, err)
				secondID, err := stream.Add(t.Context(), "second", []byte("two"))
				require.NoError(t, err)
				reader, err := stream.NewReader(t.Context(), options.WithReaderStartAtOldest(), options.WithReaderBufferSize(0), options.WithReaderBlockDuration(5*time.Millisecond))
				require.NoError(t, err)
				t.Cleanup(reader.Close)
				t.Cleanup(hook.unblock)
				sub := reader.Subscribe()
				awaitRegistrationBoundary(t, hook.first, "first reply")
				remaining := []string{firstID, secondID}
				if stage == "fanout" {
					// Receiving the first event proves fanout has started. The
					// second unbuffered event cannot complete before mutation.
					receiveRegistrationEvent(t, sub, firstID)
					remaining = remaining[1:]
				}
				want := secondID
				switch action {
				case "explicit", "newest", "unsubscribe":
					require.NoError(t, reader.RemoveStream(t.Context(), stream))
					want = fmt.Sprintf("%d-0", time.Now().Add(time.Hour).UnixMilli())
					opt := options.WithAddStreamStartAfter(want)
					if action == "newest" {
						want, err = stream.Add(t.Context(), "boundary", []byte("excluded"))
						require.NoError(t, err)
						opt = options.WithAddStreamStartAtNewest()
					}
					require.NoError(t, reader.AddStream(t.Context(), stream, opt))
				case "remove":
					require.NoError(t, reader.RemoveStream(t.Context(), stream))
				case "close":
					hook.unblock()
					reader.Close()
					require.Equal(t, []*readerRegistration{{stream: stream, cursor: "0"}}, reader.registrations, "Close suppresses advancement")
					return
				}
				if action == "unsubscribe" {
					reader.Unsubscribe(sub)
				}
				hook.unblock()
				if action != "unsubscribe" {
					for _, id := range remaining {
						receiveRegistrationEvent(t, sub, id)
					}
				}
				awaitRegistrationBoundary(t, hook.second, "next reply after completion")
				reader.lock.Lock()
				cursors := make([]string, len(reader.registrations))
				for i, registration := range reader.registrations {
					cursors[i] = registration.cursor
				}
				reader.lock.Unlock()
				if action == "remove" {
					require.Empty(t, cursors, "completion must not resurrect a removed registration")
				} else {
					require.Equal(t, []string{want}, cursors, "completion belongs to the admitted registration")
				}
			})
		}
	}
}

func TestReaderRegistrationMultistreamReplacements(t *testing.T) {
	for _, stage := range []string{"reply", "fanout"} {
		t.Run(stage, func(t *testing.T) {
			rdb := startTestRedis(t)
			hook := &registrationBoundaryHook{first: make(chan struct{}), second: make(chan struct{}), release: make(chan struct{})}
			if stage == "fanout" {
				hook.unblock()
			}
			rdb.AddHook(hook)
			a := newTestStream(t, rdb, "registration-a")
			b := newTestStream(t, rdb, "registration-b")
			a1, err := a.Add(t.Context(), "a1", []byte("one"))
			require.NoError(t, err)
			a2, err := a.Add(t.Context(), "a2", []byte("two"))
			require.NoError(t, err)
			b1, err := b.Add(t.Context(), "b1", []byte("three"))
			require.NoError(t, err)
			reader, err := a.NewReader(t.Context(), options.WithReaderStartAtOldest(), options.WithReaderBufferSize(0), options.WithReaderBlockDuration(5*time.Millisecond))
			require.NoError(t, err)
			t.Cleanup(reader.Close)
			t.Cleanup(hook.unblock)
			require.NoError(t, reader.AddStream(t.Context(), b))
			sub := reader.Subscribe()
			awaitRegistrationBoundary(t, hook.first, "first multistream reply")
			remaining := []struct {
				id, name string
			}{{a1, "a1"}, {a2, "a2"}, {b1, "b1"}}
			if stage == "fanout" {
				require.Equal(t, "a1", receiveRegistrationEvent(t, sub, a1).EventName)
				remaining = remaining[1:]
			}
			want := ""
			for generation := 1; generation <= 2; generation++ {
				handle := a
				if generation == 1 {
					handle = newTestStream(t, rdb, a.Name)
				}
				require.NoError(t, reader.RemoveStream(t.Context(), handle))
				want = fmt.Sprintf("%d-0", time.Now().Add(time.Duration(generation)*time.Hour).UnixMilli())
				require.NoError(t, reader.AddStream(t.Context(), a, options.WithAddStreamStartAfter(want)))
			}
			hook.unblock()
			for _, event := range remaining {
				require.Equal(t, event.name, receiveRegistrationEvent(t, sub, event.id).EventName)
			}
			awaitRegistrationBoundary(t, hook.second, "reply after multistream completion")
			reader.lock.Lock()
			keys := make([]string, len(reader.registrations))
			for i, registration := range reader.registrations {
				keys[i] = registration.stream.key
			}
			cursors := make([]string, len(reader.registrations))
			for i, registration := range reader.registrations {
				cursors[i] = registration.cursor
			}
			reader.lock.Unlock()
			require.Equal(t, []string{b.key, a.key}, keys)
			require.Equal(t, []string{b1, want}, cursors, "old A batch must not overwrite replacement; B still advances after slice index shifts")
		})
	}
}

func receiveRegistrationEvent(t *testing.T, sub <-chan *Event, id string) *Event {
	t.Helper()
	select {
	case event := <-sub:
		require.NotNil(t, event)
		require.Equal(t, id, event.ID, "admitted batch keeps its original delivery order")
		return event
	case <-time.After(time.Second):
		t.Fatal("admitted batch did not deliver the expected event")
	}
	return nil
}

func awaitRegistrationBoundary(t *testing.T, ch <-chan struct{}, stage string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatalf("timed out at %s", stage)
	}
}

func (h *registrationBoundaryHook) unblock() {
	h.once.Do(func() {
		close(h.release)
	})
}

func (h *registrationBoundaryHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *registrationBoundaryHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func (h *registrationBoundaryHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		err := next(ctx, cmd)
		if cmd.Name() == "xread" {
			switch h.reads.Add(1) {
			case 1:
				close(h.first)
				<-h.release
			case 2:
				close(h.second)
			}
		}
		return err
	}
}
