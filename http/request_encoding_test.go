package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

type framingBody struct {
	read   func([]byte) (int, error)
	close  func() error
	closes atomic.Int32
}

func TestFinalizeRequestEncoding(t *testing.T) {
	for _, tc := range []struct {
		name     string
		header   []string
		body     string
		known    int64
		transfer []string
		want     int64
		fail     bool
	}{
		{name: "absent", body: "abc", known: -1, want: -1},
		{name: "positive", header: []string{"3"}, body: "abc", want: 3},
		{name: "equal repeated", header: []string{"3", "3"}, body: "abc", want: 3},
		{name: "zero", header: []string{"0"}},
		{name: "zero with data", header: []string{"0"}, body: "a", fail: true},
		{name: "empty", header: []string{""}, fail: true},
		{name: "negative", header: []string{"-1"}, fail: true},
		{name: "plus", header: []string{"+1"}, fail: true},
		{name: "overflow", header: []string{"9223372036854775808"}, fail: true},
		{name: "comma", header: []string{"1, 1"}, fail: true},
		{name: "conflict", header: []string{"1", "2"}, fail: true},
		{name: "known mismatch", header: []string{"2"}, body: "abc", known: 3, fail: true},
		{name: "transfer conflict", header: []string{"3"}, body: "abc", transfer: []string{"chunked"}, fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := strings.NewReader(tc.body)
			b := &framingBody{read: reader.Read}
			req := &http.Request{Method: http.MethodPost, Header: make(http.Header), Body: b, ContentLength: tc.known, TransferEncoding: tc.transfer}
			replay := func() (io.ReadCloser, error) {
				return io.NopCloser(strings.NewReader("replay")), nil
			}
			req.GetBody = replay
			if tc.header != nil {
				req.Header["Content-Length"] = tc.header
			}
			err := FinalizeRequestEncoding(req, "svc", "method", nil)
			if tc.fail {
				require.Error(t, err)
				require.EqualValues(t, 1, b.closes.Load())
				require.Same(t, b, req.Body)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, req.ContentLength)
			if tc.header != nil && tc.want == 0 {
				require.Equal(t, http.NoBody, req.Body)
				require.EqualValues(t, 1, b.closes.Load())
				r, e := req.GetBody()
				require.NoError(t, e)
				require.Equal(t, http.NoBody, r)
			} else {
				require.Same(t, b, req.Body)
				require.Zero(t, b.closes.Load())
				r, e := req.GetBody()
				require.NoError(t, e)
				got, e := io.ReadAll(r)
				require.NoError(t, e)
				require.Equal(t, "replay", string(got))
				require.NoError(t, r.Close())
			}
		})
	}
}

func TestFinalizeRequestEncodingFailures(t *testing.T) {
	readErr := errors.New("read failed")
	closeErr := errors.New("close failed")
	priorErr := errors.New("encoder failed")
	for _, tc := range []struct {
		name     string
		prior    error
		read     error
		wantRead bool
	}{
		{name: "read and close", read: readErr, wantRead: true},
		{name: "close after EOF", read: io.EOF},
		{name: "prior encoding", prior: priorErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := &framingBody{read: func([]byte) (int, error) {
				return 0, tc.read
			}, close: func() error {
				return closeErr
			}}
			req := &http.Request{Method: http.MethodPost, Header: http.Header{"Content-Length": []string{"0"}}, Body: b}
			err := FinalizeRequestEncoding(req, "svc", "method", tc.prior)
			require.ErrorIs(t, err, closeErr)
			if tc.wantRead {
				require.ErrorIs(t, err, readErr)
			}
			if tc.prior != nil {
				require.ErrorIs(t, err, priorErr)
			}
			require.EqualValues(t, 1, b.closes.Load())
			require.Same(t, b, req.Body)
		})
	}
}

func TestFinalizeRequestEncodingEmptyReads(t *testing.T) {
	for _, eof := range []bool{false, true} {
		t.Run(map[bool]string{false: "no progress", true: "EOF after empty reads"}[eof], func(t *testing.T) {
			reads := 0
			b := &framingBody{read: func([]byte) (int, error) {
				reads++
				if eof && reads == 3 {
					return 0, io.EOF
				}
				return 0, nil
			}}
			req := &http.Request{Method: http.MethodPost, Header: http.Header{"Content-Length": []string{"0"}}, Body: b}
			err := FinalizeRequestEncoding(req, "svc", "method", nil)
			if eof {
				require.NoError(t, err)
				require.Equal(t, http.NoBody, req.Body)
			} else {
				require.ErrorIs(t, err, io.ErrNoProgress)
				require.Same(t, b, req.Body)
			}
			require.EqualValues(t, 1, b.closes.Load())
		})
	}
}

func TestFinalizeRequestEncodingCancellation(t *testing.T) {
	for _, phase := range []string{"before", "read", "EOF"} {
		t.Run(phase, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			closed := make(chan struct {
			})
			closeErr := errors.New("close failed")
			b := &framingBody{close: func() error {
				close(closed)
				return closeErr
			}}
			b.read = func([]byte) (int, error) {
				cancel()
				if phase == "read" {
					<-closed
					return 0, io.ErrClosedPipe
				}
				return 0, io.EOF
			}
			if phase == "before" {
				cancel()
			}
			req := (&http.Request{Method: http.MethodPost, Header: http.Header{"Content-Length": []string{"0"}}, Body: b}).WithContext(ctx)
			err := FinalizeRequestEncoding(req, "svc", "method", nil)
			require.ErrorIs(t, err, context.Canceled)
			require.ErrorIs(t, err, closeErr)
			if phase == "read" {
				require.ErrorIs(t, err, io.ErrClosedPipe)
			}
			require.EqualValues(t, 1, b.closes.Load())
			require.Same(t, b, req.Body)
		})
	}
}

func TestFinalizeRequestEncodingWaitsForCallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	closeStarted := make(chan struct {
	})
	closeRelease := make(chan struct {
	})
	readReturned := make(chan struct {
	})
	cause := errors.New("close callback")
	b := &framingBody{
		read: func([]byte) (int, error) {
			cancel()
			<-closeStarted
			close(readReturned)
			return 0, io.EOF
		},
		close: func() error {
			close(closeStarted)
			<-closeRelease
			return cause
		},
	}
	req := (&http.Request{Method: http.MethodPost, Header: http.Header{"Content-Length": []string{"0"}}, Body: b}).WithContext(ctx)
	result := make(chan error, 1)
	go func() {
		result <- FinalizeRequestEncoding(req, "svc", "method", nil)
	}()
	<-readReturned
	select {
	case err := <-result:
		t.Errorf("returned before callback completed: %v", err)
	default:
	}
	close(closeRelease)
	err := <-result
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorIs(t, err, cause)
	require.EqualValues(t, 1, b.closes.Load())
	require.Same(t, b, req.Body)
}

func TestFinalizeRequestEncodingReadCauses(t *testing.T) {
	cause := errors.New("read cause")
	for _, n := range []int{0, 1} {
		b := &framingBody{read: func([]byte) (int, error) {
			return n, errors.Join(io.EOF, cause)
		}}
		req := &http.Request{Method: http.MethodPost, Header: http.Header{"Content-Length": []string{"0"}}, Body: b}
		err := FinalizeRequestEncoding(req, "svc", "method", nil)
		require.ErrorIs(t, err, cause)
		require.EqualValues(t, 1, b.closes.Load())
		require.Same(t, b, req.Body)
	}
}

func TestFinalizeRequestEncodingMissingBodies(t *testing.T) {
	for _, body := range []io.ReadCloser{nil, http.NoBody} {
		for _, length := range []string{"0", "1"} {
			req := &http.Request{Method: http.MethodPost, Header: http.Header{"Content-Length": []string{length}}, Body: body}
			err := FinalizeRequestEncoding(req, "svc", "method", nil)
			if length == "0" {
				require.NoError(t, err)
				require.Equal(t, http.NoBody, req.Body)
			} else {
				require.Error(t, err)
			}
		}
	}
}

func TestFinalizeRequestEncodingCancellationAfterEOF(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := &framingBody{
		read: func([]byte) (int, error) {
			return 0, io.EOF
		},
		close: func() error {
			cancel()
			return nil
		},
	}
	req := (&http.Request{Method: http.MethodPost, Header: http.Header{"Content-Length": []string{"0"}}, Body: b}).WithContext(ctx)
	err := FinalizeRequestEncoding(req, "svc", "method", nil)
	require.ErrorIs(t, err, context.Canceled)
	require.EqualValues(t, 1, b.closes.Load())
	require.Same(t, b, req.Body)
}

func TestFinalizeRequestEncodingMethods(t *testing.T) {
	for _, method := range []string{"", "GET", "HEAD", "DELETE", "OPTIONS", "TRACE", "CONNECT", "POST", "PUT", "PATCH", "post", "CUSTOM"} {
		for _, length := range []string{"", "0", "3"} {
			t.Run(method+"/"+length, func(t *testing.T) {
				text := "abc"
				if length == "0" {
					text = ""
				}
				reader := strings.NewReader(text)
				b := &framingBody{read: reader.Read}
				req := &http.Request{Method: method, Header: make(http.Header), Body: b}
				if length != "" {
					req.Header.Set("Content-Length", length)
				}
				err := FinalizeRequestEncoding(req, "svc", "method", nil)
				supported := length != "0" || method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch
				if supported {
					require.NoError(t, err)
				} else {
					require.ErrorContains(t, err, "explicit Content-Length 0 cannot be sent")
					require.EqualValues(t, 1, b.closes.Load())
					require.Same(t, b, req.Body)
				}
			})
		}
	}
}

func (b *framingBody) Read(p []byte) (int, error) {
	return b.read(p)
}
func (b *framingBody) Close() error {
	b.closes.Add(1)
	if b.close != nil {
		return b.close()
	}
	return nil
}
