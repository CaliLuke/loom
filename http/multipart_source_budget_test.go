package http

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
)

type multipartSourceReader struct {
	*bytes.Reader
	terminal error
	reads    int
	closes   int
}

func TestMultipartSourceByteBudget(t *testing.T) {
	body, contentType := multipartSourceBody(t, [][2]string{{"value", "four"}, {"empty", ""}})
	emptyBody, emptyType := multipartSourceBody(t, [][2]string{{strings.Repeat("a", 180), ""}, {strings.Repeat("b", 180), ""}})
	for _, known := range []bool{false, true} {
		for _, tc := range []struct {
			name        string
			body        []byte
			contentType string
			limit       int64
			tooLarge    bool
		}{
			{"exact complete wire", body, contentType, int64(len(body)), false},
			{"one byte over", body, contentType, int64(len(body) - 1), true},
			{"empty field overhead", emptyBody, emptyType, 256, true},
			{"epilogue", append(bytes.Clone(body), []byte(strings.Repeat("e", 8192))...), contentType, int64(len(body)), true},
		} {
			lengthName := "unknown"
			if known {
				lengthName = "known"
			}
			t.Run(tc.name+"/"+lengthName, func(t *testing.T) {
				source := &multipartSourceReader{Reader: bytes.NewReader(tc.body)}
				req := httptest.NewRequest(http.MethodPost, "/", source)
				req.Header.Set("Content-Type", tc.contentType)
				req.ContentLength = -1
				if known {
					req.ContentLength = int64(len(tc.body))
				}
				form, err := ReadMultipartFormWithLimit(req, tc.limit)
				if tc.tooLarge {
					var serviceErr *loom.ServiceError
					require.ErrorAs(t, err, &serviceErr)
					require.Equal(t, loom.RequestBodyTooLarge, serviceErr.Name)
					require.Nil(t, form)
					require.LessOrEqual(t, source.reads, int(tc.limit+1))
					if known {
						require.Zero(t, source.reads)
					}
				} else {
					require.NoError(t, err)
					require.Equal(t, []string{"four"}, form.Values["value"])
					require.Equal(t, []string{""}, form.Values["empty"])
				}
				require.Zero(t, source.closes, "server retains request-body Close ownership")
				require.NoError(t, req.Body.Close())
				require.Equal(t, 1, source.closes)
			})
		}
	}
}

func TestMultipartSourceErrorSurvivesBufferedBoundary(t *testing.T) {
	body, contentType := multipartSourceBody(t, [][2]string{{"value", "four"}})
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"physical failure", errors.New("source read failed after data")},
		{"wrapped EOF", fmt.Errorf("source failure: %w", io.EOF)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := &multipartSourceReader{Reader: bytes.NewReader(body), terminal: tc.err}
			req := httptest.NewRequest(http.MethodPost, "/", source)
			req.Header.Set("Content-Type", contentType)
			form, err := ReadMultipartFormWithLimit(req, int64(len(body)))
			require.ErrorIs(t, err, tc.err)
			require.Nil(t, form)
			require.Zero(t, source.closes)
			require.NoError(t, req.Body.Close())
			require.Equal(t, 1, source.closes)
		})
	}
}

func TestMultipartBudgetSourceObservationPreservesRetries(t *testing.T) {
	terminal := errors.New("source temporarily unavailable")
	source := &multipartSourceReader{Reader: bytes.NewReader([]byte("a")), terminal: terminal}
	body := &requestBodyLimitReadCloser{ReadCloser: source, remaining: 2}
	buffer := make([]byte, 1)
	n, err := body.Read(buffer)
	require.Equal(t, 1, n)
	require.ErrorIs(t, err, terminal)
	require.ErrorIs(t, body.readErr, terminal)
	source.Reader.Reset([]byte("b"))
	source.terminal = nil
	n, err = body.Read(buffer)
	require.Equal(t, 1, n)
	require.NoError(t, err)
	require.Equal(t, []byte("b"), buffer)
	require.ErrorIs(t, body.readErr, terminal, "multipart completion retains the first observed error")
	require.Zero(t, source.closes)
	require.NoError(t, body.Close())
	require.Equal(t, 1, source.closes)
}

func TestMultipartDefaultEntryUsesPolicy(t *testing.T) {
	body, contentType := multipartSourceBody(t, [][2]string{{"value", "four"}})
	policy, err := NewRequestBodyPolicy(int64(len(body) - 1))
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	handler := policy.Handler(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		form, err := ReadMultipartForm(request)
		var serviceErr *loom.ServiceError
		require.ErrorAs(t, err, &serviceErr)
		require.Equal(t, loom.RequestBodyTooLarge, serviceErr.Name)
		require.Nil(t, form)
	}))
	handler.ServeHTTP(httptest.NewRecorder(), req)
}

func TestMultipartCallbackFailureDoesNotReadOrClose(t *testing.T) {
	body, contentType := multipartSourceBody(t, [][2]string{{"value", "four"}})
	source := &multipartSourceReader{Reader: bytes.NewReader(body)}
	req := httptest.NewRequest(http.MethodPost, "/", source)
	req.Header.Set("Content-Type", contentType)
	callbackErr := errors.New("callback rejected payload")
	err := DecodeMultipartRequestWithLimit(req, int64(len(body)), func(*multipart.Reader) error {
		return callbackErr
	})
	require.ErrorIs(t, err, callbackErr)
	require.Zero(t, source.reads)
	require.Zero(t, source.closes)
	require.NoError(t, req.Body.Close())
	require.Equal(t, 1, source.closes)
}

func TestMultipartCallbackCannotHideSizeError(t *testing.T) {
	body, contentType := multipartSourceBody(t, [][2]string{{"value", strings.Repeat("x", 512)}})
	for _, policyLimit := range []int64{0, 128} {
		t.Run(fmt.Sprint(policyLimit), func(t *testing.T) {
			source := &multipartSourceReader{Reader: bytes.NewReader(body)}
			req := httptest.NewRequest(http.MethodPost, "/", source)
			req.Header.Set("Content-Type", contentType)
			req.ContentLength = -1
			if policyLimit > 0 {
				req.Body = &requestBodyLimitReadCloser{ReadCloser: source, limit: policyLimit, remaining: policyLimit}
			}
			err := DecodeMultipartRequestWithLimit(req, 256, func(reader *multipart.Reader) error {
				part, err := reader.NextPart()
				require.NoError(t, err)
				_, err = io.ReadAll(part)
				require.Error(t, err)
				return nil
			})
			var serviceErr *loom.ServiceError
			require.ErrorAs(t, err, &serviceErr)
			require.Equal(t, loom.RequestBodyTooLarge, serviceErr.Name)
			limit := 256
			if policyLimit > 0 {
				limit = int(policyLimit)
			}
			require.LessOrEqual(t, source.reads, limit+1)
			require.Zero(t, source.closes)
			require.NoError(t, req.Body.Close())
			require.Equal(t, 1, source.closes)
		})
	}
}

func multipartSourceBody(t *testing.T, fields [][2]string) ([]byte, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.SetBoundary("loom-source-budget"))
	for _, field := range fields {
		require.NoError(t, writer.WriteField(field[0], field[1]))
	}
	require.NoError(t, writer.Close())
	return body.Bytes(), writer.FormDataContentType()
}

func (r *multipartSourceReader) Read(data []byte) (int, error) {
	n, err := r.Reader.Read(data)
	r.reads += n
	if n > 0 && r.Len() == 0 && r.terminal != nil {
		return n, r.terminal
	}
	return n, err
}

func (r *multipartSourceReader) Close() error {
	r.closes++
	return nil
}
