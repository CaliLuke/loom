package transport_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/observability/transport"
)

// TestCaptureResponseHeaders freezes metadata only at an observable final commit.
func TestCaptureResponseHeaders(t *testing.T) {
	for _, mode := range []string{"header", "write", "string", "flush", "informational", "upgrade", "empty reader", "reader", "committed reader"} {
		t.Run(mode, func(t *testing.T) {
			original := &captureFastWriter{ResponseRecorder: httptest.NewRecorder()}
			writer, capture := transport.CaptureResponse(original, transport.WithResponseHeaderCapture())
			before, known := capture.Headers()
			require.False(t, known)
			require.Nil(t, before)
			writer.Header().Set("Content-Type", "application/json")
			switch mode {
			case "header":
				writer.WriteHeader(http.StatusCreated)
			case "write":
				_, err := writer.Write([]byte(`{}`))
				require.NoError(t, err)
			case "string":
				_, err := io.WriteString(writer, `{}`)
				require.NoError(t, err)
			case "flush":
				writer.(http.Flusher).Flush()
			case "informational":
				writer.Header().Set("Content-Type", "text/plain")
				writer.WriteHeader(http.StatusEarlyHints)
				_, known = capture.Headers()
				require.False(t, known)
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(http.StatusOK)
			case "upgrade":
				writer.WriteHeader(http.StatusSwitchingProtocols)
			case "committed reader":
				writer.WriteHeader(http.StatusOK)
				fallthrough
			case "reader", "empty reader":
				body := "{}"
				if mode == "empty reader" {
					body = ""
				}
				_, err := writer.(io.ReaderFrom).ReadFrom(strings.NewReader(body))
				require.NoError(t, err)
			}
			writer.Header().Set("Content-Type", "text/plain")
			result, known := capture.Headers()
			if mode == "reader" || mode == "empty reader" {
				require.False(t, known)
				require.Nil(t, result)
				return
			}
			require.True(t, known)
			require.Equal(t, "application/json", result.Get("Content-Type"))
			result.Set("Content-Type", "mutated snapshot")
			again, known := capture.Headers()
			require.True(t, known)
			require.Equal(t, "application/json", again.Get("Content-Type"))
		})
	}
}

// TestCaptureResponseHeadersDisabled preserves the default metrics-only footprint.
func TestCaptureResponseHeadersDisabled(t *testing.T) {
	writer, capture := transport.CaptureResponse(httptest.NewRecorder())
	writer.WriteHeader(http.StatusOK)
	header, known := capture.Headers()
	require.Nil(t, header)
	require.False(t, known)
}

// TestCaptureResponseHeadersConcurrentObservation never reads mutable writer headers from an accessor.
func TestCaptureResponseHeadersConcurrentObservation(t *testing.T) {
	writer, capture := transport.CaptureResponse(httptest.NewRecorder(), transport.WithResponseHeaderCapture())
	done := make(chan struct{})
	go func() {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		for range 100 {
			writer.Header().Set("Content-Type", "text/plain")
		}
		close(done)
	}()
	for {
		header, known := capture.Headers()
		if known {
			require.Equal(t, "application/json", header.Get("Content-Type"))
		}
		select {
		case <-done:
			return
		default:
		}
	}
}

// TestCaptureResponseHeadersImplicitSniff omits metadata whose commit timing is hidden by forwarding.
func TestCaptureResponseHeadersImplicitSniff(t *testing.T) {
	original := httptest.NewRecorder()
	writer, capture := transport.CaptureResponse(original, transport.WithResponseHeaderCapture())
	_, err := writer.Write([]byte("ordinary"))
	require.NoError(t, err)
	require.Equal(t, "ordinary", original.Body.String())
	header, known := capture.Headers()
	require.Nil(t, header)
	require.False(t, known)
}
