package http

import (
	"errors"
	"io"
	stdhttp "net/http"
	"net/url"
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

type recordingDoer struct {
	called bool
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
