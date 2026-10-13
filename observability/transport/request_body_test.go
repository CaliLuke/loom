package transport_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/observability/transport"
)

type (
	requestCaptureSource struct {
		data     string
		err      error
		closeErr error
		reads    []int
		closes   int
	}
	blockingCaptureSource struct {
		entered chan struct{}
		closed  chan struct{}
		once    sync.Once
	}
)

func TestCaptureRequestBodyReadResults(t *testing.T) {
	boom := errors.New("original failure")
	for _, tc := range []struct {
		name, data string
		err        error
		complete   bool
	}{
		{"zero success", "", nil, false},
		{"partial success", "abc", nil, false},
		{"zero error", "", boom, false},
		{"partial error", "abc", boom, false},
		{"zero EOF", "", io.EOF, true},
		{"partial EOF", "abc", io.EOF, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := &requestCaptureSource{data: tc.data, err: tc.err, closeErr: boom}
			body, capture := transport.CaptureRequestBody(source, 4)
			require.Empty(t, source.reads)
			buffer := make([]byte, 6)
			n, err := body.Read(buffer)
			if tc.err == nil {
				require.NoError(t, err)
			} else {
				require.Same(t, tc.err, err)
			}
			require.Equal(t, tc.data, string(buffer[:n]))
			require.Equal(t, []int{6}, source.reads)
			preview, omitted, complete := capture.Body()
			require.Equal(t, tc.data, string(preview))
			require.False(t, omitted)
			require.Equal(t, tc.complete, complete)
			if len(preview) > 0 {
				preview[0] = 'z'
				again, _, _ := capture.Body()
				require.Equal(t, tc.data, string(again))
			}
			for range 2 {
				require.Same(t, boom, body.Close())
			}
			require.Equal(t, 2, source.closes)
			_, _, afterClose := capture.Body()
			require.Equal(t, tc.complete, afterClose)
		})
	}
}

func TestCaptureRequestBodyEmptyAndLimit(t *testing.T) {
	for _, original := range []io.ReadCloser{nil, http.NoBody} {
		body, capture := transport.CaptureRequestBody(original, 4)
		require.Equal(t, original, body)
		preview, omitted, complete := capture.Body()
		require.Nil(t, preview)
		require.False(t, omitted)
		require.True(t, complete)
	}
	for _, limit := range []int{-1, 0} {
		require.Panics(t, func() {
			transport.CaptureRequestBody(http.NoBody, limit)
		})
	}
	for _, size := range []int{0, 3, 4, 5, 100000} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			text := strings.Repeat("x", size)
			body, capture := transport.CaptureRequestBody(io.NopCloser(strings.NewReader(text)), 4)
			got, err := io.ReadAll(body)
			require.NoError(t, err)
			require.Equal(t, text, string(got))
			preview, omitted, complete := capture.Body()
			require.True(t, complete)
			require.Equal(t, size > 4, omitted)
			if omitted {
				require.Nil(t, preview)
			} else {
				require.Equal(t, text, string(preview))
			}
			require.NoError(t, body.Close())
		})
	}
}

func TestCaptureRequestBodyNestedAndAcrossReads(t *testing.T) {
	source := &requestCaptureSource{data: "ab"}
	inner, first := transport.CaptureRequestBody(source, 4)
	body, second := transport.CaptureRequestBody(inner, 4)
	for i := range 3 {
		buffer := make([]byte, 2)
		n, err := body.Read(buffer)
		require.NoError(t, err)
		require.Equal(t, 2, n)
		for _, capture := range []*transport.RequestBodyCapture{first, second} {
			preview, omitted, complete := capture.Body()
			require.Equal(t, i == 2, omitted)
			require.False(t, complete)
			if omitted {
				require.Nil(t, preview)
			} else {
				require.Equal(t, strings.Repeat("ab", i+1), string(preview))
			}
		}
	}
	require.Equal(t, []int{2, 2, 2}, source.reads)
	require.NoError(t, body.Close())
	require.Equal(t, 1, source.closes)
}

func TestCaptureRequestBodyCloseUnblocksReadAndObservation(t *testing.T) {
	source := &blockingCaptureSource{entered: make(chan struct{}), closed: make(chan struct{})}
	body, capture := transport.CaptureRequestBody(source, 4)
	defer func() {
		require.NoError(t, source.Close())
	}()
	readDone := make(chan error, 1)
	go func() {
		var buffer [4]byte
		_, err := body.Read(buffer[:])
		readDone <- err
	}()
	<-source.entered
	observed := make(chan struct{})
	go func() {
		capture.Body()
		close(observed)
	}()
	select {
	case <-observed:
	case <-time.After(time.Second):
		t.Fatal("snapshot blocked behind underlying Read")
	}
	closeDone := make(chan error, 1)
	go func() {
		closeDone <- body.Close()
	}()
	select {
	case err := <-closeDone:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Close blocked behind underlying Read")
	}
	require.ErrorIs(t, <-readDone, io.ErrClosedPipe)
	preview, omitted, complete := capture.Body()
	require.Empty(t, preview)
	require.False(t, omitted)
	require.False(t, complete)
}

func TestCaptureRequestBodyConcurrentObservation(t *testing.T) {
	body, capture := transport.CaptureRequestBody(io.NopCloser(strings.NewReader(strings.Repeat("x", 32))), 16)
	done := make(chan error, 1)
	go func() {
		var buffer [1]byte
		for {
			if _, err := body.Read(buffer[:]); err != nil {
				done <- err
				return
			}
		}
	}()
	for {
		preview, omitted, _ := capture.Body()
		require.LessOrEqual(t, len(preview), 16)
		if omitted {
			require.Nil(t, preview)
		}
		select {
		case err := <-done:
			require.ErrorIs(t, err, io.EOF)
			require.NoError(t, body.Close())
			return
		default:
		}
	}
}

func (r *requestCaptureSource) Read(p []byte) (int, error) {
	r.reads = append(r.reads, len(p))
	return copy(p, r.data), r.err
}

func (r *requestCaptureSource) Close() error {
	r.closes++
	return r.closeErr
}

func (r *blockingCaptureSource) Read([]byte) (int, error) {
	close(r.entered)
	<-r.closed
	return 0, io.ErrClosedPipe
}

func (r *blockingCaptureSource) Close() error {
	r.once.Do(func() {
		close(r.closed)
	})
	return nil
}
