package http

import (
	"bytes"
	"errors"
	"io"
	stdhttp "net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientError_Unwrap(t *testing.T) {
	sentinelError := errors.New("this is na error")
	alternateSentinelError := errors.New("another error")

	tests := []struct {
		name             string
		err              error
		checkedSentinel  error
		expectedCausedBy bool
	}{
		{
			name: "caused by sentinel",
			err: ErrRequestError(
				"AService",
				"Something went wrong",
				sentinelError,
			),
			checkedSentinel:  sentinelError,
			expectedCausedBy: true,
		},
		{
			name: "null cause hypothesis",
			err: ErrRequestError(
				"AService",
				"Something went wrong",
				sentinelError,
			),
			checkedSentinel:  alternateSentinelError,
			expectedCausedBy: false,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				isCausedBy := errors.Is(tt.err, tt.checkedSentinel)

				if isCausedBy != tt.expectedCausedBy {
					if tt.expectedCausedBy {
						t.Errorf("got error %#v, should be caused by %#v", tt.err, tt.checkedSentinel)
					} else {
						t.Errorf("got error %#v, must NOT be caused by %#v", tt.err, tt.checkedSentinel)
					}
				}
			},
		)
	}
}

func TestDebugDoerDoReturnsErrorWhenRequestBodyCaptureFails(t *testing.T) {
	sentinelError := errors.New("read failed")
	doer := &recordingDoer{}
	req := &stdhttp.Request{
		Method: "POST",
		URL:    &url.URL{Scheme: "https", Host: "example.com", Path: "/widgets"},
		Body:   &failingReadCloser{err: sentinelError},
	}

	resp, err := NewDebugDoer(doer).Do(req)
	if resp != nil {
		require.NoError(t, resp.Body.Close())
	}

	require.Nil(t, resp)
	require.Error(t, err)
	require.ErrorContains(t, err, "capture request body")
	require.ErrorIs(t, err, sentinelError)
	require.False(t, doer.called)
}

func TestDebugDoerRedactsSensitiveCapture(t *testing.T) {
	doer := &debugCaptureDoer{
		t:            t,
		responseBody: `{"result":"ok","access_token":"response-secret"}`,
		responseHeader: stdhttp.Header{
			"Content-Type": []string{"application/json"},
			"Set-Cookie":   []string{"session=response-cookie"},
			"X-Api-Key":    []string{"response-key"},
		},
	}
	req, err := stdhttp.NewRequest(
		stdhttp.MethodPost,
		"https://example.com/widgets?access_token=query-secret&view=full",
		strings.NewReader(`{"name":"widget","password":"request-secret"}`),
	)
	require.NoError(t, err)
	req.GetBody = nil
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer request-token")
	req.Header.Set("Cookie", "session=request-cookie")

	debug := NewDebugDoer(doer)
	resp, err := debug.Do(req)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, resp.Body.Close())
	}()

	var output bytes.Buffer
	debug.Fprint(&output)
	captured := output.String()
	for _, secret := range []string{
		"request-secret",
		"response-secret",
		"request-token",
		"request-cookie",
		"response-cookie",
		"response-key",
		"query-secret",
	} {
		require.NotContains(t, captured, secret)
	}
	require.Contains(t, captured, "[REDACTED]")
	require.Contains(t, captured, `"name":"widget"`)
	require.Contains(t, captured, `"result":"ok"`)
	require.Contains(t, captured, "view=full")
}

func TestDebugDoerBoundsCaptureWithoutTruncatingTransportBodies(t *testing.T) {
	requestBody := strings.Repeat("r", 70<<10) + "request-tail"
	responseBody := strings.Repeat("s", 70<<10) + "response-tail"
	requestReads := &countingReadCloser{Reader: strings.NewReader(requestBody)}
	responseReads := &countingReadCloser{Reader: strings.NewReader(responseBody)}
	doer := &debugCaptureDoer{
		t:             t,
		requestReads:  requestReads,
		responseReads: responseReads,
	}
	req, err := stdhttp.NewRequest(stdhttp.MethodPost, "https://example.com/widgets", requestReads)
	require.NoError(t, err)
	req.GetBody = nil

	debug := NewDebugDoer(doer)
	resp, err := debug.Do(req)
	require.NoError(t, err)
	require.LessOrEqual(t, responseReads.bytesRead, int(debugBodyCaptureLimit+1))
	response, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	require.Equal(t, requestBody, doer.requestBody)
	require.Equal(t, responseBody, string(response))

	var output bytes.Buffer
	debug.Fprint(&output)
	captured := output.String()
	require.NotContains(t, captured, "request-tail")
	require.NotContains(t, captured, "response-tail")
	require.Contains(t, captured, "body omitted after 65536 bytes")
	require.Less(t, output.Len(), 140<<10)
}

func TestRedactDebugBody(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		truncated   bool
		contains    []string
		excludes    []string
	}{
		{
			name:        "nested JSON",
			contentType: "application/problem+json",
			body:        `{"name":"widget","auth":{"refresh_token":"secret"}}`,
			contains:    []string{`"name":"widget"`, `"refresh_token":"[REDACTED]"`},
			excludes:    []string{"secret"},
		},
		{
			name:        "large JSON number",
			contentType: "application/json",
			body:        `{"id":900719925474099312345,"password":"secret"}`,
			contains:    []string{`"id":900719925474099312345`, `"password":"[REDACTED]"`},
			excludes:    []string{"secret"},
		},
		{
			name:        "form",
			contentType: "application/x-www-form-urlencoded",
			body:        "name=widget&password=secret&session_id=session-secret",
			contains:    []string{"name=widget", "password=%5BREDACTED%5D", "session_id=%5BREDACTED%5D"},
			excludes:    []string{"secret", "session-secret"},
		},
		{
			name:        "invalid JSON",
			contentType: "application/json",
			body:        `{"password":"secret"`,
			contains:    []string{"[invalid JSON body omitted]"},
			excludes:    []string{"secret"},
		},
		{
			name:        "truncated body",
			contentType: "text/plain",
			body:        "secret-prefix",
			truncated:   true,
			contains:    []string{"body omitted after 65536 bytes"},
			excludes:    []string{"secret-prefix"},
		},
		{
			name:        "ordinary text",
			contentType: "text/plain",
			body:        "diagnostic details",
			contains:    []string{"diagnostic details"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := stdhttp.Header{"Content-Type": []string{tt.contentType}}
			got := string(redactDebugBody(headers, []byte(tt.body), tt.truncated))
			for _, value := range tt.contains {
				require.Contains(t, got, value)
			}
			for _, value := range tt.excludes {
				require.NotContains(t, got, value)
			}
		})
	}
}

type recordingDoer struct {
	called bool
}

type debugCaptureDoer struct {
	t              *testing.T
	requestBody    string
	responseBody   string
	responseHeader stdhttp.Header
	requestReads   *countingReadCloser
	responseReads  *countingReadCloser
}

func (d *debugCaptureDoer) Do(req *stdhttp.Request) (*stdhttp.Response, error) {
	d.t.Helper()
	if d.requestReads != nil {
		require.LessOrEqual(d.t, d.requestReads.bytesRead, int(debugBodyCaptureLimit+1))
	}
	body, err := io.ReadAll(req.Body)
	require.NoError(d.t, err)
	d.requestBody = string(body)
	var responseBody io.ReadCloser = io.NopCloser(strings.NewReader(d.responseBody))
	if d.responseReads != nil {
		responseBody = d.responseReads
	}
	return &stdhttp.Response{
		Status:     "200 OK",
		StatusCode: stdhttp.StatusOK,
		Header:     d.responseHeader.Clone(),
		Body:       responseBody,
	}, nil
}

type countingReadCloser struct {
	io.Reader
	bytesRead int
}

func (r *countingReadCloser) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.bytesRead += n
	return n, err
}

func (r *countingReadCloser) Close() error {
	return nil
}

func (d *recordingDoer) Do(*stdhttp.Request) (*stdhttp.Response, error) {
	d.called = true
	return &stdhttp.Response{
		StatusCode: stdhttp.StatusOK,
		Body:       io.NopCloser(&failingReader{}),
	}, nil
}

type failingReadCloser struct {
	err error
}

func (r *failingReadCloser) Read([]byte) (int, error) {
	return 0, r.err
}

func (r *failingReadCloser) Close() error {
	return nil
}

type failingReader struct{}

func (r *failingReader) Read([]byte) (int, error) {
	return 0, io.EOF
}
