package http

import (
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/CaliLuke/loom/observability/diagnostics"
	"github.com/CaliLuke/loom/observability/transport"
)

type (
	debugExchange struct {
		request     *http.Request
		requestBody []byte
		response    *http.Response
		capture     *transport.BodyReadCapture
		output      io.Writer
		printed     sync.Once
		policy      *diagnostics.Policy
		contentType string
	}
	debugCompletionBody struct {
		io.ReadCloser
		exchange *debugExchange
	}
)

// Fprint prints the latest exchange snapshot without consuming response bytes.
// An output error ends this best-effort diagnostic; it never alters body I/O.
func (dd *debugDoer) Fprint(w io.Writer) {
	if dd.exchange != nil {
		dd.exchange.print(w)
	}
}

// Read forwards the source result and prints the exchange after observed EOF.
func (b *debugCompletionBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if errors.Is(err, io.EOF) {
		b.exchange.finish()
	}
	return n, err
}

// Close always reaches the source, preserving its error. It prints once without
// waiting for an in-flight Read; that snapshot may therefore be incomplete.
func (b *debugCompletionBody) Close() error {
	err := b.ReadCloser.Close()
	b.exchange.finish()
	return err
}

func captureDebugExchange(
	req *http.Request, requestBody []byte, resp *http.Response, output io.Writer, policy *diagnostics.Policy,
) *debugExchange {
	metadata := new(http.Response)
	*metadata = *resp
	metadata.Header = policy.Headers(resp.Header)
	metadata.Body = nil
	body, capture := transport.CaptureBodyReads(resp.Body, int(debugBodyCaptureLimit))
	exchange := &debugExchange{
		request:     req,
		requestBody: requestBody,
		response:    metadata,
		capture:     capture,
		output:      output,
		policy:      policy,
		contentType: resp.Header.Get("Content-Type"),
	}
	if body == nil || body == http.NoBody {
		exchange.finish()
	} else {
		resp.Body = &debugCompletionBody{ReadCloser: body, exchange: exchange}
	}
	return exchange
}

func writeDebugHeaders(buf *strings.Builder, prefix string, headers http.Header) {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		buf.WriteString("\n" + prefix + " " + key + ": " + strings.Join(headers[key], ", "))
	}
}

func (e *debugExchange) finish() {
	e.printed.Do(func() {
		e.print(e.output)
	})
}

func (e *debugExchange) print(w io.Writer) {
	var buf strings.Builder
	buf.WriteString("> " + e.request.Method + " " + e.request.URL.String())
	writeDebugHeaders(&buf, ">", e.request.Header)
	if len(e.requestBody) > 0 {
		buf.WriteString("\n" + string(e.requestBody))
	}
	buf.WriteString("\n< " + e.response.Status)
	writeDebugHeaders(&buf, "<", e.response.Header)
	data, omitted, complete := e.capture.Body()
	data = e.policy.Body(e.contentType, data, omitted, complete)
	if len(data) > 0 {
		buf.WriteString("\n" + string(data))
	}
	if !complete {
		buf.WriteString("\n[response body capture stopped before EOF]")
	}
	buf.WriteByte('\n')
	if _, err := io.WriteString(w, buf.String()); err != nil {
		return
	}
}
