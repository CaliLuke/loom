package streaming

import (
	"testing"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/pulse"
)

func TestStreamEventsSkipsMalformedMessages(t *testing.T) {
	valid := redis.XMessage{ID: "5-0", Values: map[string]any{nameKey: "created", payloadKey: "payload"}}
	tests := []struct {
		name   string
		values map[string]any
	}{
		{name: "missing name", values: map[string]any{payloadKey: "payload"}},
		{name: "missing payload", values: map[string]any{nameKey: "created"}},
		{name: "nil values", values: nil},
		{name: "non-string name", values: map[string]any{nameKey: int64(1), payloadKey: "payload"}},
		{name: "non-string payload", values: map[string]any{nameKey: "created", payloadKey: []byte("payload")}},
		{name: "non-string topic", values: map[string]any{nameKey: "created", payloadKey: "payload", topicKey: 3}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sub := newEventSubscriber(2)
			require.NotPanics(t, func() {
				streamEvents(
					t.Context(),
					"events",
					streamKeyPrefix+"events",
					"",
					[]redis.XMessage{{ID: "4-0", Values: test.values}, valid},
					nil,
					[]*eventSubscriber{sub},
					nil,
					pulse.NoopLogger(),
					make(chan struct{}),
				)
			})
			require.Len(t, sub.ch, 1)
			event := <-sub.ch
			require.Equal(t, "5-0", event.ID)
			require.Equal(t, "created", event.EventName)
			require.Equal(t, []byte("payload"), event.Payload)
		})
	}
}

func TestEventCreatedAtToleratesMalformedIDs(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want time.Time
	}{
		{name: "redis id", id: "1700000000123-7", want: time.UnixMilli(1700000000123).UTC()},
		{name: "missing sequence", id: "1700000000123", want: time.UnixMilli(1700000000123).UTC()},
		{name: "empty", id: "", want: time.Unix(0, 0).UTC()},
		{name: "not a number", id: "abc-1", want: time.Unix(0, 0).UTC()},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got time.Time
			require.NotPanics(t, func() {
				got = (&Event{ID: test.id}).CreatedAt()
			})
			require.Equal(t, test.want, got)
		})
	}
}
