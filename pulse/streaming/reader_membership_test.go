package streaming

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming/options"
)

func TestReaderStreamMembership(t *testing.T) {
	for _, handle := range []string{"same", "equivalent", "absent"} {
		t.Run(handle, func(t *testing.T) {
			rdb := startTestRedis(t)
			stream := newTestStream(t, rdb, "membership")
			equivalent := newTestStream(t, rdb, "membership")
			other := newTestStream(t, rdb, "other")
			reader, err := stream.NewReader(t.Context(), options.WithReaderStartAtOldest())
			require.NoError(t, err)
			t.Cleanup(reader.Close)
			original := reader.registrations[0]
			require.NoError(t, reader.AddStream(t.Context(), equivalent))
			require.Equal(t, []*readerRegistration{original}, reader.registrations)
			target := stream
			switch handle {
			case "equivalent":
				target = equivalent
			case "absent":
				target = other
			}
			require.NoError(t, reader.RemoveStream(t.Context(), target))
			require.NoError(t, reader.RemoveStream(t.Context(), target))
			if handle == "absent" {
				require.Equal(t, []*readerRegistration{original}, reader.registrations)
				return
			}
			require.Empty(t, reader.registrations)
			require.NoError(t, reader.AddStream(t.Context(), equivalent, options.WithAddStreamStartAfter("123-4")))
			require.Len(t, reader.registrations, 1)
			require.NotSame(t, original, reader.registrations[0])
			require.Equal(t, "123-4", reader.registrations[0].cursor)
			require.NoError(t, reader.RemoveStream(t.Context(), stream))
			require.Empty(t, reader.registrations)
		})
	}
}
