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

const debugBodyCaptureLimit = 64 << 10

// Debug returns a debug middleware which prints detailed information about
// incoming requests and outgoing responses including all headers, parameters
// and bodies. Each body preview is limited to 64 KiB; oversized previews are
// omitted with an explicit marker, without limiting application bytes. Request
// capture observes downstream reads without draining or replacing errors. A
// preview without observed EOF is marked incomplete. The summary is emitted
// when the handler returns. Response writer capabilities are preserved.
func Debug(mux loomhttp.Muxer, w io.Writer) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			buf := &bytes.Buffer{}
			reqID := shortID()
			writeDebugRequest(buf, mux, reqID, r)
			body, requestCapture := transport.CaptureBodyReads(r.Body, debugBodyCaptureLimit)
			r.Body = body

			writer, capture := transport.CaptureResponse(rw, transport.WithResponseBodyCapture(debugBodyCaptureLimit))
			h.ServeHTTP(writer, r)
			writeDebugRequestBody(buf, reqID, requestCapture)
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
}

func writeDebugResponse(buf *bytes.Buffer, reqID string, header http.Header, capture *transport.ResponseCapture) {
	fmt.Fprintf(buf, "\n< [%s] %s", reqID, http.StatusText(capture.StatusCode()))
	writeDebugMap(buf, reqID, "<", stringMap(header))
	body, omitted := capture.Body()
	if omitted {
		body = fmt.Appendf(nil, "[body omitted after %d bytes]", debugBodyCaptureLimit)
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

func writeDebugRequestBody(buf *bytes.Buffer, reqID string, capture *transport.BodyReadCapture) {
	body, omitted, complete := capture.Body()
	if omitted {
		body = fmt.Appendf(nil, "[body omitted after %d bytes]", debugBodyCaptureLimit)
	}
	writeDebugBody(buf, reqID, body)
	if !complete {
		writeDebugBody(buf, reqID, []byte("[request body capture stopped before EOF]"))
	}
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
