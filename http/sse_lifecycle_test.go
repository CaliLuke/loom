package http

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

type countedSSEBody struct {
	io.Reader
	closes   atomic.Int32
	closeErr error
}

func TestSSEStreamReaderLibraryState(t *testing.T) {
	reader := NewSSEStreamReader(io.NopCloser(strings.NewReader("id: first\ndata: one\n\ndata: two\n\nid:\ndata: three\n\n")))
	defer func() { require.NoError(t, reader.Close()) }()
	for _, want := range []SSEEvent{{ID: "first", Data: "one"}, {ID: "first", Data: "two"}, {Data: "three"}} {
		got, err := reader.ReadEvent(context.Background())
		require.NoError(t, err)
		require.Equal(t, want, got)
	}
	_, err := reader.ReadEvent(context.Background())
	require.ErrorIs(t, err, io.EOF)
}

func TestSSEStreamReaderCloseReleasesIterator(t *testing.T) {
	for _, readFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "before_read", true: "between_events"}[readFirst], func(t *testing.T) {
			closeErr := errors.New("close failed")
			body := &countedSSEBody{Reader: strings.NewReader("data: one\n\ndata: two\n\n"), closeErr: closeErr}
			reader := NewSSEStreamReader(body)
			if readFirst {
				event, err := reader.ReadEvent(context.Background())
				require.NoError(t, err)
				require.Equal(t, "one", event.Data)
			}
			require.ErrorIs(t, reader.Close(), closeErr)
			require.ErrorIs(t, reader.Close(), closeErr)
			require.Equal(t, int32(1), body.closes.Load())
			_, err := reader.ReadEvent(context.Background())
			require.ErrorIs(t, err, io.EOF)
		})
	}
}

func (b *countedSSEBody) Close() error {
	b.closes.Add(1)
	return b.closeErr
}
