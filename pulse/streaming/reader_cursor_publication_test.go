package streaming

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming/options"
)

type (
	cursorLookupContextKey struct{}
	cursorLookupGate       struct {
		entered chan struct{}
		release chan struct{}
		once    sync.Once
	}
	cursorLookupHook struct {
		lookups atomic.Int32
	}
)

func TestReaderNewestSnapshotPrecedesSubscribe(t *testing.T) {
	for _, state := range []string{"missing", "populated", "trimmed"} {
		t.Run(state, func(t *testing.T) {
			rdb := startTestRedis(t)
			stream := newTestStream(t, rdb, "snapshot-subscribe")
			if state != "missing" {
				id, err := stream.Add(t.Context(), "before", []byte("old"))
				require.NoError(t, err)
				if state == "trimmed" {
					require.NoError(t, rdb.XDel(t.Context(), stream.key, id).Err())
				}
			}
			reader, err := stream.NewReader(t.Context(), options.WithReaderBlockDuration(5*time.Millisecond))
			require.NoError(t, err)
			t.Cleanup(reader.Close)
			id, err := stream.Add(t.Context(), "after", []byte("new"))
			require.NoError(t, err)
			sub := reader.Subscribe()
			select {
			case event := <-sub:
				require.NotNil(t, event)
				require.Equal(t, id, event.ID)
				require.Equal(t, "after", event.EventName)
			case <-time.After(time.Second):
				t.Fatal("event after snapshot was lost before Subscribe")
			}
			select {
			case event := <-sub:
				t.Errorf("unexpected historical or duplicate event: %#v", event)
			case <-time.After(30 * time.Millisecond):
			}
		})
	}
}

func TestReaderCursorPublicationCloseDuringLookup(t *testing.T) {
	rdb := startTestRedis(t)
	rdb.AddHook(new(cursorLookupHook))
	main := newTestStream(t, rdb, "cursor-close-main")
	added := newTestStream(t, rdb, "cursor-close-added")
	reader, err := main.NewReader(t.Context(), options.WithReaderStartAtOldest())
	require.NoError(t, err)
	t.Cleanup(reader.Close)
	gate := newCursorLookupGate(t)
	result := make(chan error, 1)
	go func() {
		result <- reader.AddStream(context.WithValue(t.Context(), cursorLookupContextKey{}, gate), added, options.WithAddStreamStartAtNewest())
	}()
	awaitCursorLookup(t, gate)
	closed := make(chan struct{})
	go func() {
		reader.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(time.Second):
		gate.unblock()
		<-closed
		t.Fatal("Close waited for an unpublished AddStream lookup")
	}
	gate.unblock()
	require.ErrorContains(t, <-result, "reader is closing")
	require.Len(t, reader.registrations, 1)
	require.Same(t, main, reader.registrations[0].stream)
}

func TestReaderCursorPublicationDuplicateWinner(t *testing.T) {
	rdb := startTestRedis(t)
	rdb.AddHook(new(cursorLookupHook))
	main := newTestStream(t, rdb, "cursor-duplicate-main")
	added := newTestStream(t, rdb, "cursor-duplicate-added")
	reader, err := main.NewReader(t.Context(), options.WithReaderStartAtOldest())
	require.NoError(t, err)
	t.Cleanup(reader.Close)
	gate := newCursorLookupGate(t)
	result := make(chan error, 1)
	go func() {
		result <- reader.AddStream(context.WithValue(t.Context(), cursorLookupContextKey{}, gate), added, options.WithAddStreamStartAtNewest())
	}()
	awaitCursorLookup(t, gate)
	winnerID, err := added.Add(t.Context(), "before-winner", []byte("winner"))
	require.NoError(t, err)
	require.NoError(t, reader.AddStream(t.Context(), added, options.WithAddStreamStartAtNewest()))
	gate.unblock()
	require.NoError(t, <-result)
	require.Equal(t, []*readerRegistration{{stream: main, cursor: "0"}, {stream: added, cursor: winnerID}}, reader.registrations, "slow snapshot must not overwrite the installed winner")
}

func TestReaderCursorPublicationCanceledLookup(t *testing.T) {
	rdb := startTestRedis(t)
	rdb.AddHook(new(cursorLookupHook))
	main := newTestStream(t, rdb, "cursor-cancel-main")
	added := newTestStream(t, rdb, "cursor-cancel-added")
	reader, err := main.NewReader(t.Context(), options.WithReaderStartAtOldest())
	require.NoError(t, err)
	t.Cleanup(reader.Close)
	gate := newCursorLookupGate(t)
	ctx, cancel := context.WithCancel(context.WithValue(t.Context(), cursorLookupContextKey{}, gate))
	defer cancel()
	result := make(chan error, 1)
	go func() {
		result <- reader.AddStream(ctx, added, options.WithAddStreamStartAtNewest())
	}()
	awaitCursorLookup(t, gate)
	cancel()
	require.ErrorIs(t, <-result, context.Canceled)
	require.Len(t, reader.registrations, 1)
	require.Same(t, main, reader.registrations[0].stream)
	failed, err := added.NewReader(ctx)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, failed)
}

func TestReaderCursorFailedAdditionPreservesActiveRead(t *testing.T) {
	rdb := startTestRedis(t)
	main := newTestStream(t, rdb, "cursor-failed-main")
	bad := newTestStream(t, rdb, "cursor-failed-added")
	require.NoError(t, rdb.Set(t.Context(), bad.key, "wrong type", 0).Err())
	reader, err := main.NewReader(t.Context(), options.WithReaderStartAtOldest())
	require.NoError(t, err)
	t.Cleanup(reader.Close)
	readCtx, _, armed := reader.armRead(t.Context())
	require.True(t, armed)
	defer reader.clearActiveRead()
	err = reader.AddStream(t.Context(), bad, options.WithAddStreamStartAtNewest())
	require.ErrorContains(t, err, bad.key)
	require.ErrorContains(t, err, "WRONGTYPE")
	require.NoError(t, readCtx.Err(), "failed addition must not cancel the existing read")
	require.Len(t, reader.registrations, 1)
	require.Same(t, main, reader.registrations[0].stream)
}

func TestReaderCursorLookupOnlyForNewNewestRegistration(t *testing.T) {
	rdb := startTestRedis(t)
	hook := new(cursorLookupHook)
	rdb.AddHook(hook)
	main := newTestStream(t, rdb, "cursor-count-main")
	added := newTestStream(t, rdb, "cursor-count-added")
	reader, err := main.NewReader(t.Context(), options.WithReaderBlockDuration(5*time.Millisecond))
	require.NoError(t, err)
	t.Cleanup(reader.Close)
	require.EqualValues(t, 1, hook.lookups.Load())
	require.NoError(t, reader.AddStream(t.Context(), added))
	require.EqualValues(t, 2, hook.lookups.Load(), "default addition retains authored newest mode")
	require.NoError(t, reader.AddStream(t.Context(), added))
	for range 2 {
		_, err := reader.xread(t.Context())
		require.ErrorIs(t, err, redis.Nil)
	}
	require.EqualValues(t, 2, hook.lookups.Load(), "polls and duplicate registration cannot re-resolve newest")
	reader.Close()
	require.Error(t, reader.AddStream(t.Context(), main))
	require.EqualValues(t, 2, hook.lookups.Load())
	explicit, err := main.NewReader(t.Context(), options.WithReaderStartAfter("123-4"))
	require.NoError(t, err)
	t.Cleanup(explicit.Close)
	require.NoError(t, explicit.AddStream(t.Context(), added))
	require.Equal(t, []*readerRegistration{{stream: main, cursor: "123-4"}, {stream: added, cursor: "123-4"}}, explicit.registrations)
	require.EqualValues(t, 2, hook.lookups.Load(), "explicit positions require no lookup")
}

func TestReaderNewestRequiresRedisClient(t *testing.T) {
	stream, err := NewStream("cursor-nil-client", nil)
	require.NoError(t, err)
	reader, err := stream.NewReader(t.Context())
	require.Nil(t, reader)
	require.ErrorContains(t, err, "Redis client is nil")
	explicit, err := stream.NewReader(t.Context(), options.WithReaderStartAtOldest())
	require.NoError(t, err)
	explicit.Close()
}

func newCursorLookupGate(t *testing.T) *cursorLookupGate {
	t.Helper()
	gate := &cursorLookupGate{entered: make(chan struct{}), release: make(chan struct{})}
	t.Cleanup(gate.unblock)
	return gate
}

func awaitCursorLookup(t *testing.T, gate *cursorLookupGate) {
	t.Helper()
	select {
	case <-gate.entered:
	case <-time.After(time.Second):
		t.Fatal("newest lookup was not observed")
	}
}

func (g *cursorLookupGate) unblock() {
	g.once.Do(func() {
		close(g.release)
	})
}

func (h *cursorLookupHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *cursorLookupHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		err := next(ctx, cmd)
		if cmd.Name() != "xrevrange" {
			return err
		}
		h.lookups.Add(1)
		if gate, ok := ctx.Value(cursorLookupContextKey{}).(*cursorLookupGate); ok {
			close(gate.entered)
			select {
			case <-gate.release:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return err
	}
}

func (h *cursorLookupHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
