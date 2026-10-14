package streaming

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CaliLuke/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming/options"
)

type cursorReadinessHook struct {
	reads  atomic.Int32
	second chan struct{}
}

func TestReaderCursorReadinessSnapshot(t *testing.T) {
	for _, populated := range []bool{false, true} {
		t.Run(map[bool]string{false: "empty", true: "populated"}[populated], func(t *testing.T) {
			rdb := startTestRedis(t)
			stream := newTestStream(t, rdb, "cursor-snapshot")
			want := "0-0"
			if populated {
				var err error
				want, err = stream.Add(t.Context(), "before", []byte("before"))
				require.NoError(t, err)
			}
			reader, err := stream.NewReader(t.Context())
			require.NoError(t, err)
			t.Cleanup(reader.Close)
			require.Equal(t, want, reader.registrations[0].cursor, "newest must become one concrete position before publication")
		})
	}
}

func TestReaderCursorReadinessBetweenPolls(t *testing.T) {
	for _, tc := range []struct {
		name   string
		added  bool
		oldest bool
	}{
		{"initial newest", false, false},
		{"added newest", true, false},
		{"explicit oldest control", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rdb := startTestRedis(t)
			stream := newTestStream(t, rdb, "cursor-poll-main")
			opts := []options.Reader{options.WithReaderBlockDuration(5 * time.Millisecond)}
			if tc.oldest {
				opts = append(opts, options.WithReaderStartAtOldest())
			}
			reader, err := stream.NewReader(t.Context(), opts...)
			require.NoError(t, err)
			t.Cleanup(reader.Close)
			if tc.added {
				stream = newTestStream(t, rdb, "cursor-poll-added")
				require.NoError(t, reader.AddStream(t.Context(), stream, options.WithAddStreamStartAtNewest()))
			}
			// Directly execute complete polls: the insertion is strictly after
			// the first reply and before the second command, without sleeps.
			_, err = reader.xread(t.Context())
			require.ErrorIs(t, err, redis.Nil)
			id, err := stream.Add(t.Context(), "between", []byte("between"))
			require.NoError(t, err)
			messages, err := reader.xread(t.Context())
			require.NoError(t, err, "event inserted between polls must survive")
			require.Len(t, messages, 1)
			require.Len(t, messages[0].messages, 1)
			require.Equal(t, id, messages[0].messages[0].ID)
		})
	}
}

func TestReaderCursorReadinessMultistreamFanout(t *testing.T) {
	for _, addWhileBlocked := range []bool{false, true} {
		t.Run(map[bool]string{false: "registered before fanout", true: "added during fanout"}[addWhileBlocked], func(t *testing.T) {
			rdb := startTestRedis(t)
			hook := &cursorReadinessHook{second: make(chan struct{})}
			rdb.AddHook(hook)
			a := newTestStream(t, rdb, "cursor-fanout-a")
			b := newTestStream(t, rdb, "cursor-fanout-b")
			for range 3 {
				_, err := a.Add(t.Context(), "a", []byte("a"))
				require.NoError(t, err)
			}
			reader, err := a.NewReader(t.Context(), options.WithReaderStartAtOldest(),
				options.WithReaderMaxPolled(1), options.WithReaderBufferSize(1),
				options.WithReaderBlockDuration(5*time.Millisecond))
			require.NoError(t, err)
			t.Cleanup(reader.Close)
			if !addWhileBlocked {
				require.NoError(t, reader.AddStream(t.Context(), b, options.WithAddStreamStartAtNewest()))
			}
			sub := reader.Subscribe()
			select {
			case <-hook.second:
			case <-time.After(time.Second):
				t.Fatal("second A read did not complete")
			}
			// Buffer contains A1; A2 cannot finish fanout until we receive.
			// Therefore a third Redis read cannot precede the B insertion.
			if addWhileBlocked {
				require.NoError(t, reader.AddStream(t.Context(), b, options.WithAddStreamStartAtNewest()))
			}
			_, err = b.Add(t.Context(), "b", []byte("b"))
			require.NoError(t, err)
			got := map[string]int{}
			deadline := time.NewTimer(300 * time.Millisecond)
			defer deadline.Stop()
			for len(got) < 2 || got["a"] < 3 {
				select {
				case event := <-sub:
					require.NotNil(t, event)
					got[event.EventName]++
				case <-deadline.C:
					require.Equal(t, map[string]int{"a": 3, "b": 1}, got)
					return
				}
			}
			require.Equal(t, map[string]int{"a": 3, "b": 1}, got)
		})
	}
}

func TestReaderCursorReadinessFailedLookup(t *testing.T) {
	for _, added := range []bool{false, true} {
		t.Run(map[bool]string{false: "constructor", true: "added stream"}[added], func(t *testing.T) {
			rdb := startTestRedis(t)
			bad := newTestStream(t, rdb, "cursor-wrong-type")
			require.NoError(t, rdb.Set(t.Context(), bad.key, "not a stream", 0).Err())
			if !added {
				reader, err := bad.NewReader(t.Context())
				if reader != nil {
					t.Cleanup(reader.Close)
				}
				require.Error(t, err, "a failed newest lookup cannot publish a usable Reader")
				return
			}
			main := newTestStream(t, rdb, "cursor-healthy")
			reader, err := main.NewReader(t.Context(), options.WithReaderStartAtOldest())
			require.NoError(t, err)
			t.Cleanup(reader.Close)
			err = reader.AddStream(t.Context(), bad, options.WithAddStreamStartAtNewest())
			require.Error(t, err)
			require.Len(t, reader.registrations, 1)
			require.Same(t, main, reader.registrations[0].stream)
		})
	}
}

func TestReaderCursorReadinessExplicitAndDuplicateControls(t *testing.T) {
	for _, id := range []string{"0", "123-4"} {
		t.Run(id, func(t *testing.T) {
			rdb := startTestRedis(t)
			stream := newTestStream(t, rdb, "cursor-explicit")
			reader, err := stream.NewReader(t.Context(), options.WithReaderStartAfter(id))
			require.NoError(t, err)
			t.Cleanup(reader.Close)
			require.Equal(t, []*readerRegistration{{stream: stream, cursor: id}}, reader.registrations)
			// Duplicate registration remains a no-op even when a new newest
			// lookup would fail; preserve the already selected cursor.
			require.NoError(t, rdb.Set(t.Context(), stream.key, "not a stream", 0).Err())
			require.NoError(t, reader.AddStream(t.Context(), stream, options.WithAddStreamStartAtNewest()))
			require.Equal(t, []*readerRegistration{{stream: stream, cursor: id}}, reader.registrations)
			reader.Close()
			require.ErrorContains(t, reader.AddStream(t.Context(), stream), "reader is closing")
		})
	}
}

func (h *cursorReadinessHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *cursorReadinessHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		err := next(ctx, cmd)
		if cmd.Name() == "xread" && h.reads.Add(1) == 2 {
			close(h.second)
		}
		return err
	}
}

func (h *cursorReadinessHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
