package streaming

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming/options"
)

type optionAdmissionHook struct {
	commands atomic.Int32
}

var errOptionAdmissionRedis = errors.New("option admission reached Redis")

func TestConsumerNumericOptionsRejectBeforeRedis(t *testing.T) {
	for _, tc := range []struct {
		name   string
		reader bool
		buffer int
		grace  time.Duration
		want   string
	}{
		{name: "sink zero grace", grace: 0, want: "ack grace period"},
		{name: "sink negative grace", grace: -time.Second, want: "ack grace period"},
		{name: "sink negative buffer", buffer: -1, grace: time.Second, want: "buffer size"},
		{name: "reader negative buffer", reader: true, buffer: -1, want: "buffer size"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stream, hook := optionAdmissionStream(t)
			if tc.reader {
				reader, err := stream.NewReader(t.Context(), options.WithReaderBufferSize(tc.buffer))
				require.Nil(t, reader)
				require.ErrorContains(t, err, tc.want)
			} else {
				sink, err := stream.NewSink(t.Context(), "options", options.WithSinkBufferSize(tc.buffer), options.WithSinkAckGracePeriod(tc.grace))
				require.Nil(t, sink)
				require.ErrorContains(t, err, tc.want)
			}
			require.Zero(t, hook.commands.Load(), "invalid options must not reach Redis or publish membership")
		})
	}
}

func TestConsumerNumericOptionsPreserveAdmission(t *testing.T) {
	for _, reader := range []bool{false, true} {
		for _, zeroBuffer := range []bool{false, true} {
			t.Run(map[bool]string{false: "sink", true: "reader"}[reader]+"/"+map[bool]string{false: "default buffer", true: "zero buffer"}[zeroBuffer], func(t *testing.T) {
				stream, hook := optionAdmissionStream(t)
				var err error
				if reader {
					opts := []options.Reader{options.WithReaderBlockDuration(0)}
					if zeroBuffer {
						opts = append(opts, options.WithReaderBufferSize(0))
					}
					var result *Reader
					result, err = stream.NewReader(t.Context(), opts...)
					require.Nil(t, result)
				} else {
					opts := []options.Sink{options.WithSinkBlockDuration(0), options.WithSinkAckGracePeriod(time.Nanosecond)}
					if zeroBuffer {
						opts = append(opts, options.WithSinkBufferSize(0))
					}
					var result *Sink
					result, err = stream.NewSink(t.Context(), "options", opts...)
					require.Nil(t, result)
				}
				require.ErrorIs(t, err, errOptionAdmissionRedis)
				require.EqualValues(t, 1, hook.commands.Load(), "accepted options reach existing native setup")
			})
		}
	}
}

func TestConsumerNumericOptionsUseFinalValue(t *testing.T) {
	for _, domain := range []string{"reader buffer", "sink buffer", "sink grace"} {
		for _, invalidLast := range []bool{false, true} {
			t.Run(domain+"/"+map[bool]string{false: "valid last", true: "invalid last"}[invalidLast], func(t *testing.T) {
				stream, hook := optionAdmissionStream(t)
				var err error
				if domain == "reader buffer" {
					values := []int{-1, 0}
					if invalidLast {
						values[0], values[1] = values[1], values[0]
					}
					var result *Reader
					result, err = stream.NewReader(t.Context(), options.WithReaderBufferSize(values[0]), options.WithReaderBufferSize(values[1]))
					require.Nil(t, result)
				} else {
					var opts []options.Sink
					if domain == "sink buffer" {
						opts = []options.Sink{options.WithSinkBufferSize(-1), options.WithSinkBufferSize(0)}
					} else {
						opts = []options.Sink{options.WithSinkAckGracePeriod(0), options.WithSinkAckGracePeriod(time.Second)}
					}
					if invalidLast {
						opts[0], opts[1] = opts[1], opts[0]
					}
					var result *Sink
					result, err = stream.NewSink(t.Context(), "options", opts...)
					require.Nil(t, result)
				}
				if invalidLast {
					require.Error(t, err)
					require.NotErrorIs(t, err, errOptionAdmissionRedis)
					require.Zero(t, hook.commands.Load())
				} else {
					require.ErrorIs(t, err, errOptionAdmissionRedis)
					require.EqualValues(t, 1, hook.commands.Load())
				}
			})
		}
	}
}

func TestConsumerOptionsRetainPatternAdmission(t *testing.T) {
	for _, reader := range []bool{false, true} {
		t.Run(map[bool]string{false: "sink", true: "reader"}[reader], func(t *testing.T) {
			stream, hook := optionAdmissionStream(t)
			var err error
			if reader {
				var result *Reader
				result, err = stream.NewReader(t.Context(), options.WithReaderTopicPattern("["))
				require.Nil(t, result)
			} else {
				var result *Sink
				result, err = stream.NewSink(t.Context(), "options", options.WithSinkTopicPattern("["))
				require.Nil(t, result)
			}
			require.ErrorContains(t, err, "topic pattern must be a valid regex")
			require.Zero(t, hook.commands.Load())
		})
	}
}

func optionAdmissionStream(t *testing.T) (*Stream, *optionAdmissionHook) {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: "unused.invalid:1"})
	t.Cleanup(func() {
		require.NoError(t, client.Close())
	})
	hook := &optionAdmissionHook{}
	client.AddHook(hook)
	stream, err := NewStream("option-admission", client)
	require.NoError(t, err)
	return stream, hook
}

func (h *optionAdmissionHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *optionAdmissionHook) ProcessHook(redis.ProcessHook) redis.ProcessHook {
	return func(context.Context, redis.Cmder) error {
		h.commands.Add(1)
		return errOptionAdmissionRedis
	}
}

func (h *optionAdmissionHook) ProcessPipelineHook(redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(context.Context, []redis.Cmder) error {
		h.commands.Add(1)
		return errOptionAdmissionRedis
	}
}
