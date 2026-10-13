package http

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type (
	debugReadStep struct {
		data string
		err  error
	}
	debugStepBody struct {
		steps    []debugReadStep
		closes   int
		closeErr error
	}
	debugResponseDoer struct {
		body        io.ReadCloser
		contentType string
	}
	debugLockedOutput struct {
		mu   sync.Mutex
		data bytes.Buffer
	}
	debugBlockedBody struct {
		started chan struct{}
		closed  chan struct{}
		release chan struct{}
		once    sync.Once
		err     error
	}
)

func TestDebugResponseCompletionTiming(t *testing.T) {
	sentinel := errors.New("read failure")
	for _, tc := range []struct {
		name  string
		steps []debugReadStep
	}{
		{"EOF", []debugReadStep{{"", io.EOF}}},
		{"bytes with EOF", []debugReadStep{{"last", io.EOF}}},
		{"error then resumed EOF", []debugReadStep{{"first", sentinel}, {"last", io.EOF}}},
		{"early Close", []debugReadStep{{"partial", nil}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := &debugStepBody{steps: append([]debugReadStep(nil), tc.steps...)}
			var output bytes.Buffer
			debug := NewDebugDoer(debugResponseDoer{body: source}).(*debugDoer)
			debug.output = &output
			response, err := debug.Do(httptest.NewRequest("GET", "http://example.test/timing", nil))
			require.NoError(t, err)
			require.Zero(t, output.Len())
			var consumed strings.Builder
			for _, step := range tc.steps {
				buf := make([]byte, 64)
				n, err := response.Body.Read(buf)
				require.Equal(t, step.data, string(buf[:n]))
				require.Equal(t, step.err, err)
				consumed.WriteString(step.data)
				if errors.Is(step.err, io.EOF) {
					require.Equal(t, 1, strings.Count(output.String(), "> GET "))
				} else {
					require.Zero(t, output.Len(), "non-EOF reads must not finish observation")
				}
			}
			require.NoError(t, response.Body.Close())
			require.Equal(t, 1, strings.Count(output.String(), "> GET "))
			require.Contains(t, output.String(), consumed.String())
			saved := output.String()
			require.NoError(t, response.Body.Close())
			require.Equal(t, 2, source.closes)
			require.Equal(t, saved, output.String())
			for range 2 {
				var snapshot bytes.Buffer
				debug.Fprint(&snapshot)
				require.Equal(t, saved, snapshot.String())
			}
			require.Equal(t, saved, output.String(), "manual snapshots must not duplicate automatic output")
		})
	}
}

func TestDebugResponseCompletionEmptyAndCloseFailure(t *testing.T) {
	closeErr := errors.New("close failure")
	for _, tc := range []struct {
		name      string
		body      io.ReadCloser
		immediate bool
		closeErr  error
	}{
		{"nil", nil, true, nil},
		{"NoBody", http.NoBody, true, nil},
		{"empty reader", io.NopCloser(strings.NewReader("")), false, nil},
		{"Close error", &debugStepBody{closeErr: closeErr}, false, closeErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			debug := NewDebugDoer(debugResponseDoer{body: tc.body}).(*debugDoer)
			debug.output = &output
			response, err := debug.Do(httptest.NewRequest("GET", "http://example.test/empty", nil))
			require.NoError(t, err)
			require.Equal(t, tc.immediate, output.Len() > 0)
			if response.Body != nil {
				require.Equal(t, tc.closeErr, response.Body.Close())
			}
			require.Equal(t, 1, strings.Count(output.String(), "> GET "))
			require.Equal(t, !tc.immediate, strings.Contains(output.String(), "stopped before EOF"))
		})
	}
}

func TestDebugResponseCompletionKeepsExchangeIdentity(t *testing.T) {
	var output bytes.Buffer
	debug := NewDebugDoer(debugResponseDoer{body: io.NopCloser(strings.NewReader("first-body"))}).(*debugDoer)
	debug.output = &output
	first, err := debug.Do(httptest.NewRequest("GET", "http://example.test/first", nil))
	require.NoError(t, err)
	debug.Doer = debugResponseDoer{body: io.NopCloser(strings.NewReader("second-body"))}
	second, err := debug.Do(httptest.NewRequest("GET", "http://example.test/second", nil))
	require.NoError(t, err)
	for _, response := range []*http.Response{second, first} {
		_, err := io.ReadAll(response.Body)
		require.NoError(t, err)
	}
	require.NoError(t, first.Body.Close())
	require.NoError(t, second.Body.Close())
	text := output.String()
	require.Equal(t, 1, strings.Count(text, "/first"))
	require.Equal(t, 1, strings.Count(text, "/second"))
	require.Less(t, strings.Index(text, "/second"), strings.Index(text, "second-body"))
	require.Less(t, strings.Index(text, "second-body"), strings.Index(text, "/first"))
	require.Less(t, strings.Index(text, "/first"), strings.Index(text, "first-body"))
	var snapshot bytes.Buffer
	debug.Fprint(&snapshot)
	require.Contains(t, snapshot.String(), "/second")
	require.NotContains(t, snapshot.String(), "/first")
}

func TestDebugResponseCompletionRedactionAndRetention(t *testing.T) {
	for _, tc := range []struct {
		name, contentType, data, want string
		complete                      bool
		excluded                      string
	}{
		{"JSON", "application/json", `{"nested":{"password":"secret"},"ok":true}`, `"ok":true`, true, "secret"},
		{"form", "application/x-www-form-urlencoded", "password=secret&ok=yes", "ok=yes", true, "secret"},
		{"malformed JSON", "application/json", `{"password":"secret"`, "invalid JSON body omitted", true, "secret"},
		{"malformed form", "application/x-www-form-urlencoded", "password=%XXsecret", "invalid form body omitted", true, "secret"},
		{"incomplete JSON", "application/problem+json", `{"password":"secret"}`, "incomplete structured body omitted", false, "secret"},
		{"incomplete form", "application/x-www-form-urlencoded", "password=secret", "incomplete structured body omitted", false, "secret"},
		{"incomplete text", "text/plain", "visible", "visible", false, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			readErr := error(nil)
			if tc.complete {
				readErr = io.EOF
			}
			source := &debugStepBody{steps: []debugReadStep{{tc.data, readErr}}}
			var output bytes.Buffer
			debug := NewDebugDoer(debugResponseDoer{body: source, contentType: tc.contentType}).(*debugDoer)
			debug.output = &output
			response, err := debug.Do(httptest.NewRequest("GET", "http://example.test/redaction", nil))
			require.NoError(t, err)
			buf := make([]byte, 128)
			n, err := response.Body.Read(buf)
			require.Equal(t, tc.data, string(buf[:n]))
			require.Equal(t, readErr, err)
			var snapshot bytes.Buffer
			debug.Fprint(&snapshot)
			require.Contains(t, snapshot.String(), tc.want)
			if tc.excluded != "" {
				require.NotContains(t, snapshot.String(), tc.excluded)
			}
			require.NoError(t, response.Body.Close())
			require.Contains(t, output.String(), tc.want)
			if tc.excluded != "" {
				require.NotContains(t, output.String(), tc.excluded)
			}
			require.Equal(t, !tc.complete, strings.Contains(output.String(), "stopped before EOF"))
		})
	}
	for _, size := range []int{0, int(debugBodyCaptureLimit) - 1, int(debugBodyCaptureLimit), int(debugBodyCaptureLimit) + 1, 4 * int(debugBodyCaptureLimit)} {
		t.Run(fmt.Sprintf("bytes-%d", size), func(t *testing.T) {
			data := strings.Repeat("x", size)
			var output bytes.Buffer
			debug := NewDebugDoer(debugResponseDoer{body: io.NopCloser(strings.NewReader(data))}).(*debugDoer)
			debug.output = &output
			response, err := debug.Do(httptest.NewRequest("GET", "http://example.test/bounded", nil))
			require.NoError(t, err)
			got, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			require.Equal(t, data, string(got))
			if size > int(debugBodyCaptureLimit) {
				require.Contains(t, output.String(), "body omitted after 65536 bytes")
				require.NotContains(t, output.String(), "xxxxx")
			} else {
				require.Contains(t, output.String(), data)
				require.NotContains(t, output.String(), "body omitted")
			}
		})
	}
}

func TestDebugResponseConcurrentSnapshots(t *testing.T) {
	var output debugLockedOutput
	debug := NewDebugDoer(debugResponseDoer{body: io.NopCloser(strings.NewReader(strings.Repeat("x", 4096)))}).(*debugDoer)
	debug.output = &output
	response, err := debug.Do(httptest.NewRequest("GET", "http://example.test/concurrent", nil))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, response.Body.Close())
	}()
	done := make(chan error, 1)
	go func() {
		buf := make([]byte, 16)
		for {
			_, readErr := response.Body.Read(buf)
			if readErr != nil {
				done <- readErr
				return
			}
		}
	}()
	for range 100 {
		var snapshot bytes.Buffer
		debug.Fprint(&snapshot)
	}
	require.ErrorIs(t, <-done, io.EOF)
	require.NoError(t, response.Body.Close())
	require.Equal(t, 1, strings.Count(output.String(), "> GET "))
}

func TestDebugRequestGetBodyRemainsIndependent(t *testing.T) {
	doer := &debugCaptureDoer{t: t, responseBody: "ok"}
	request, err := http.NewRequest("POST", "http://example.test/request", strings.NewReader("original request"))
	require.NoError(t, err)
	originalBody := request.Body
	debug := NewDebugDoer(doer).(*debugDoer)
	debug.output = io.Discard
	response, err := debug.Do(request)
	require.NoError(t, err)
	require.Equal(t, originalBody, request.Body, "GetBody capture must not replace the transport body")
	require.Equal(t, "original request", doer.requestBody)
	require.NoError(t, response.Body.Close())
	require.NoError(t, request.Body.Close())
}

func TestDebugResponseDoDoesNotWaitForStream(t *testing.T) {
	reader, writer := io.Pipe()
	t.Cleanup(func() {
		require.NoError(t, reader.Close())
		require.NoError(t, writer.Close())
	})
	debug := NewDebugDoer(debugResponseDoer{body: reader}).(*debugDoer)
	debug.output = io.Discard
	done := make(chan *http.Response, 1)
	failed := make(chan error, 1)
	go func() {
		response, err := debug.Do(httptest.NewRequest("GET", "http://example.test/stream", nil)) //nolint:bodyclose // Ownership transfers through done; receiver closes it, and cleanup closes the pipe on timeout.
		if err != nil {
			failed <- err
			return
		}
		done <- response
	}()
	select {
	case response := <-done:
		require.NoError(t, response.Body.Close())
	case err := <-failed:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Error("Do waited for unread stream bytes")
	}
}

func TestDebugResponseConcurrentCloseAndSnapshot(t *testing.T) {
	sentinel := errors.New("source closed")
	source := &debugBlockedBody{started: make(chan struct{}), closed: make(chan struct{}), release: make(chan struct{}), err: sentinel}
	var output debugLockedOutput
	debug := NewDebugDoer(debugResponseDoer{body: source}).(*debugDoer)
	debug.output = &output
	response, err := debug.Do(httptest.NewRequest("GET", "http://example.test/blocked", nil))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, response.Body.Close())
	}()
	readDone := make(chan debugReadStep, 1)
	go func() {
		buf := make([]byte, 8)
		n, err := response.Body.Read(buf)
		readDone <- debugReadStep{string(buf[:n]), err}
	}()
	<-source.started
	var snapshot bytes.Buffer
	debug.Fprint(&snapshot)
	require.Contains(t, snapshot.String(), "stopped before EOF")
	closeDone := make(chan error, 1)
	go func() {
		closeDone <- response.Body.Close()
	}()
	select {
	case err := <-closeDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		close(source.release)
		t.Fatal("Close blocked on capture while source Read remained in flight")
	}
	require.Contains(t, output.String(), "stopped before EOF")
	require.NotContains(t, output.String(), "late")
	close(source.release)
	result := <-readDone
	require.Equal(t, "late", result.data)
	require.Same(t, sentinel, result.err)
	snapshot.Reset()
	debug.Fprint(&snapshot)
	require.Contains(t, snapshot.String(), "late")
	require.NotContains(t, output.String(), "late", "completed automatic snapshot stays stable")
	require.Equal(t, 1, strings.Count(output.String(), "> GET "))
}

// Do returns metadata specific to this exchange without reading its body.
func (d debugResponseDoer) Do(*http.Request) (*http.Response, error) {
	return &http.Response{Status: "200 OK", Header: http.Header{"Content-Type": []string{d.contentType}}, Body: d.body}, nil
}

// Read emits the next configured source result.
func (b *debugStepBody) Read(p []byte) (int, error) {
	if len(b.steps) == 0 {
		return 0, io.EOF
	}
	step := b.steps[0]
	b.steps = b.steps[1:]
	return copy(p, step.data), step.err
}

// Close counts every original operation and returns the source error.
func (b *debugStepBody) Close() error {
	b.closes++
	return b.closeErr
}

// Write serializes diagnostic writes for concurrent tests.
func (o *debugLockedOutput) Write(p []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.data.Write(p)
}

// String returns a synchronized diagnostic snapshot.
func (o *debugLockedOutput) String() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.data.String()
}

// Read waits for Close and a test release before returning partial source bytes.
func (b *debugBlockedBody) Read(p []byte) (int, error) {
	close(b.started)
	<-b.closed
	<-b.release
	return copy(p, "late"), b.err
}

// Close releases the underlying read without waiting for it to finish.
func (b *debugBlockedBody) Close() error {
	b.once.Do(func() {
		close(b.closed)
	})
	return nil
}
