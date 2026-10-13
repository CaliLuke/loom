package middleware_test

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	loomhttp "github.com/CaliLuke/loom/http"
	httpm "github.com/CaliLuke/loom/http/middleware"
	loom "github.com/CaliLuke/loom/pkg"
)

type debugRequestBody struct {
	data      string
	readErr   error
	closeErr  error
	reads     int
	closes    int
	readSizes []int
}

func TestDebugRequestReadErrorTransparency(t *testing.T) {
	readErr := errors.New("original read failure")
	for _, data := range []string{"", "partial"} {
		t.Run(data, func(t *testing.T) {
			source := &debugRequestBody{data: data, readErr: readErr}
			var output bytes.Buffer
			h := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				require.ErrorIs(t, err, readErr)
				require.Equal(t, data, string(body))
			})
			r := httptest.NewRequest(http.MethodPost, "/", nil)
			r.Body = source
			httpm.Debug(loomhttp.NewMuxer(), &output)(h).ServeHTTP(httptest.NewRecorder(), r)
		})
	}
}

func TestDebugRequestReadAndCloseOwnership(t *testing.T) {
	for _, read := range []bool{false, true} {
		t.Run(map[bool]string{false: "unread", true: "partial"}[read], func(t *testing.T) {
			closeErr := errors.New("original close failure")
			source := &debugRequestBody{data: "abcdef", closeErr: closeErr}
			var output bytes.Buffer
			h := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				require.Zero(t, source.reads, "Debug must not read before handler entry")
				if read {
					var body [2]byte
					n, err := r.Body.Read(body[:])
					require.NoError(t, err)
					require.Equal(t, "ab", string(body[:n]))
					require.Equal(t, []int{2}, source.readSizes)
				}
				for range 2 {
					require.ErrorIs(t, r.Body.Close(), closeErr)
				}
			})
			r := httptest.NewRequest(http.MethodPost, "/", nil)
			r.Body = source
			httpm.Debug(loomhttp.NewMuxer(), &output)(h).ServeHTTP(httptest.NewRecorder(), r)
			require.Equal(t, 2, source.closes)
			require.Equal(t, map[bool]int{false: 0, true: 1}[read], source.reads)
			require.Contains(t, output.String(), "[request body capture stopped before EOF]")
			require.NotContains(t, output.String(), "cdef")
		})
	}
}

func TestDebugRequestBodyPolicyTransparency(t *testing.T) {
	policy, err := loomhttp.NewRequestBodyPolicy(4)
	require.NoError(t, err)
	for _, outerDebug := range []bool{false, true} {
		for _, length := range []int64{-1, 6} {
			t.Run(strings.Join([]string{map[bool]string{false: "outer-policy", true: "outer-debug"}[outerDebug], map[int64]string{-1: "unknown", 6: "known"}[length]}, "/"), func(t *testing.T) {
				var output bytes.Buffer
				source := &debugRequestBody{data: "abcdef"}
				h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					body, readErr := io.ReadAll(r.Body)
					var serviceErr *loom.ServiceError
					require.ErrorAs(t, readErr, &serviceErr)
					require.Equal(t, "request_too_large", serviceErr.Name)
					if length == -1 {
						require.Equal(t, "abcd", string(body))
						require.Equal(t, "f", source.data, "only the size policy overflow probe may consume beyond the limit")
					} else {
						require.Empty(t, body)
						require.Zero(t, source.reads)
					}
					w.WriteHeader(http.StatusRequestEntityTooLarge)
				})
				var wrapped http.Handler = policy.Handler(httpm.Debug(loomhttp.NewMuxer(), &output)(h))
				if outerDebug {
					wrapped = httpm.Debug(loomhttp.NewMuxer(), &output)(policy.Handler(h))
				}
				r := httptest.NewRequest(http.MethodPost, "/", nil)
				r.Body, r.ContentLength = source, length
				response := httptest.NewRecorder()
				wrapped.ServeHTTP(response, r)
				require.Equal(t, http.StatusRequestEntityTooLarge, response.Code)
			})
		}
	}
}

func TestDebugRequestCaptureLimit(t *testing.T) {
	for _, size := range []int{0, 65535, 65536, 65537, 200000} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			body := strings.Repeat("x", size)
			var output bytes.Buffer
			h := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				require.Equal(t, body, string(got))
			})
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
			httpm.Debug(loomhttp.NewMuxer(), &output)(h).ServeHTTP(httptest.NewRecorder(), r)
			if size > 65536 {
				require.True(t, strings.Contains(output.String(), "[body omitted after 65536 bytes]"), "missing omission marker")
				require.NotContains(t, output.String(), strings.Repeat("x", 10))
			} else {
				require.Contains(t, output.String(), body)
				require.NotContains(t, output.String(), "omitted")
			}
			require.NotContains(t, output.String(), "stopped before EOF")
		})
	}
}

func TestDebugRequestCloseErrorTransparency(t *testing.T) {
	closeErr := errors.New("original close failure")
	source := &debugRequestBody{data: "a", closeErr: closeErr}
	var output bytes.Buffer
	h := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		for range 2 {
			if err := r.Body.Close(); !errors.Is(err, closeErr) {
				t.Errorf("Close returned %v, want original error %v", err, closeErr)
			}
		}
	})
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Body = source
	httpm.Debug(loomhttp.NewMuxer(), &output)(h).ServeHTTP(httptest.NewRecorder(), r)
	require.Equal(t, 2, source.closes)
}

func TestDebugRequestNilBody(t *testing.T) {
	var output bytes.Buffer
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Body = nil
	h := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		require.Nil(t, r.Body)
	})
	require.NotPanics(t, func() {
		httpm.Debug(loomhttp.NewMuxer(), &output)(h).ServeHTTP(httptest.NewRecorder(), r)
	})
}

func TestDebugRequestCompleteLogTiming(t *testing.T) {
	var output bytes.Buffer
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Zero(t, output.Len())
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, "request", string(body))
		require.Zero(t, output.Len())
		_, err = w.Write([]byte("response"))
		require.NoError(t, err)
		require.Zero(t, output.Len())
	})
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("request"))
	httpm.Debug(loomhttp.NewMuxer(), &output)(h).ServeHTTP(httptest.NewRecorder(), r)
	require.Less(t, strings.Index(output.String(), "request"), strings.Index(output.String(), "response"))
	require.NotContains(t, output.String(), "stopped before EOF")
}

func TestDebugRequestHandlerRunsBeforeUploadCompletes(t *testing.T) {
	reader, writer := io.Pipe()
	defer func() {
		require.NoError(t, reader.Close())
	}()
	entered := make(chan struct{})
	done := make(chan struct{})
	var output bytes.Buffer
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		close(entered)
	})
	r := httptest.NewRequest(http.MethodPost, "/", reader)
	go func() {
		defer close(done)
		httpm.Debug(loomhttp.NewMuxer(), &output)(h).ServeHTTP(httptest.NewRecorder(), r)
	}()
	started := false
	select {
	case <-entered:
		started = true
	case <-time.After(time.Second):
	}
	// Release the old eager reader before asserting so the negative test leaves
	// no background goroutine or open pipe behind.
	require.NoError(t, writer.Close())
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Debug did not finish after upload completion")
	}
	require.True(t, started, "handler was delayed until upload completion")
}

func TestDebugRequestPolicyExactLimit(t *testing.T) {
	policy, err := loomhttp.NewRequestBodyPolicy(4)
	require.NoError(t, err)
	for _, outerDebug := range []bool{false, true} {
		for _, length := range []int64{-1, 4} {
			t.Run(fmt.Sprintf("outer-debug=%t/length=%d", outerDebug, length), func(t *testing.T) {
				var output bytes.Buffer
				h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					body, readErr := io.ReadAll(r.Body)
					require.NoError(t, readErr)
					require.Equal(t, "abcd", string(body))
					w.WriteHeader(http.StatusNoContent)
				})
				var wrapped http.Handler = policy.Handler(httpm.Debug(loomhttp.NewMuxer(), &output)(h))
				if outerDebug {
					wrapped = httpm.Debug(loomhttp.NewMuxer(), &output)(policy.Handler(h))
				}
				r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("abcd"))
				r.ContentLength = length
				response := httptest.NewRecorder()
				wrapped.ServeHTTP(response, r)
				require.Equal(t, http.StatusNoContent, response.Code)
				require.Contains(t, output.String(), "abcd")
				require.NotContains(t, output.String(), "stopped before EOF")
			})
		}
	}
}

func TestDebugRequestStreamingResponseBeforeUploadCompletes(t *testing.T) {
	reader, writer := io.Pipe()
	release := make(chan struct{})
	finished := make(chan struct{})
	var once sync.Once
	var output bytes.Buffer
	policy, err := loomhttp.NewRequestBodyPolicy(4)
	require.NoError(t, err)
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		controller := http.NewResponseController(w)
		if controlErr := controller.EnableFullDuplex(); controlErr != nil {
			t.Error(controlErr)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		if _, writeErr := io.WriteString(w, "data: ready\n\n"); writeErr != nil {
			t.Error(writeErr)
			return
		}
		if flushErr := controller.Flush(); flushErr != nil {
			t.Error(flushErr)
			return
		}
		<-release
	})
	wrapped := policy.Handler(httpm.Debug(loomhttp.NewMuxer(), &output)(h))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(finished)
		wrapped.ServeHTTP(w, r)
	}))
	defer func() {
		once.Do(func() {
			close(release)
		})
		require.NoError(t, writer.Close())
		require.NoError(t, reader.Close())
		server.Close()
	}()
	request, err := http.NewRequest(http.MethodPost, server.URL, reader)
	require.NoError(t, err)
	client := server.Client()
	client.Timeout = 3 * time.Second
	response, err := client.Do(request)
	require.NoError(t, err, "response must arrive while upload remains open")
	defer func() {
		require.NoError(t, response.Body.Close())
	}()
	line, err := bufio.NewReader(response.Body).ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "data: ready\n", line)
	require.NoError(t, writer.Close())
	once.Do(func() {
		close(release)
	})
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("handler did not finish after stream release")
	}
	require.Contains(t, output.String(), "[request body capture stopped before EOF]")
	require.Contains(t, output.String(), "data: ready")
}

func (b *debugRequestBody) Read(p []byte) (int, error) {
	b.reads++
	b.readSizes = append(b.readSizes, len(p))
	n := copy(p, b.data)
	b.data = b.data[n:]
	if len(b.data) == 0 {
		if b.readErr != nil {
			return n, b.readErr
		}
		return n, io.EOF
	}
	return n, nil
}

func (b *debugRequestBody) Close() error {
	b.closes++
	return b.closeErr
}
