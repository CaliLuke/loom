package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/internal/identifier"
	"github.com/CaliLuke/loom/observability/transport"
)

const debugResponseCaptureLimit = 64 << 10

// Debug returns a debug middleware which prints detailed information about
// incoming requests and outgoing responses including all headers, parameters
// and bodies. Response body capture is limited to 64 KiB; oversized response
// bodies are omitted with an explicit marker, without limiting wire bytes.
// The wrapper preserves the underlying response writer capabilities. Request
// bodies are currently read in full before the handler runs.
func Debug(mux loomhttp.Muxer, w io.Writer) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			buf := &bytes.Buffer{}
			reqID := shortID()
			writeDebugRequest(buf, mux, reqID, r)

			writer, capture := transport.CaptureResponse(rw, transport.WithResponseBodyCapture(debugResponseCaptureLimit))
			h.ServeHTTP(writer, r)
			writeDebugResponse(buf, reqID, rw.Header(), capture)
			buf.WriteByte('\n')
			w.Write(buf.Bytes()) // nolint: errcheck
		})
	}
}

func writeDebugRequest(buf *bytes.Buffer, mux loomhttp.Muxer, reqID string, r *http.Request) {
	fmt.Fprintf(buf, "> [%s] %s %s", reqID, r.Method, r.URL.String())
	writeDebugMap(buf, reqID, ">", stringMap(r.Header))
	writeDebugMap(buf, reqID, ">", mux.Vars(r))
	body := readDebugBody(r)
	writeDebugBody(buf, reqID, body)
	r.Body = io.NopCloser(bytes.NewBuffer(body))
}

func writeDebugResponse(buf *bytes.Buffer, reqID string, header http.Header, capture *transport.ResponseCapture) {
	fmt.Fprintf(buf, "\n< [%s] %s", reqID, http.StatusText(capture.StatusCode()))
	writeDebugMap(buf, reqID, "<", stringMap(header))
	body, omitted := capture.Body()
	if omitted {
		body = fmt.Appendf(nil, "[body omitted after %d bytes]", debugResponseCaptureLimit)
	}
	writeDebugBody(buf, reqID, body)
}

func stringMap(values map[string][]string) map[string]string {
	flat := make(map[string]string, len(values))
	for key, value := range values {
		flat[key] = strings.Join(value, ", ")
	}
	return flat
}

func writeDebugMap(buf *bytes.Buffer, reqID, prefix string, values map[string]string) {
	keys := sortedDebugKeys(values)
	for _, key := range keys {
		fmt.Fprintf(buf, "\n%s [%s] %s: %s", prefix, reqID, key, values[key])
	}
}

func sortedDebugKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func readDebugBody(r *http.Request) []byte {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return []byte("failed to read body: " + err.Error())
	}
	return body
}

func writeDebugBody(buf *bytes.Buffer, reqID string, body []byte) {
	if len(body) == 0 {
		return
	}
	buf.WriteByte('\n')
	for _, line := range strings.Split(string(body), "\n") {
		fmt.Fprintf(buf, "[%s] %s\n", reqID, line)
	}
}

// shortID produces a " unique" 6 bytes long string.
// Do not use as a reliable way to get unique IDs, instead use for things like logging.
func shortID() string {
	return identifier.MustBase64(6)
}
