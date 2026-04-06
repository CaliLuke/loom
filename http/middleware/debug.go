package middleware

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"

	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/middleware"
)

// responseDupper tees the response to a buffer and a response writer.
type responseDupper struct {
	http.ResponseWriter
	Buffer *bytes.Buffer
	Status int
}

// Debug returns a debug middleware which prints detailed information about
// incoming requests and outgoing responses including all headers, parameters
// and bodies.
func Debug(mux loomhttp.Muxer, w io.Writer) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			buf := &bytes.Buffer{}
			reqID := requestID(r)
			writeDebugRequest(buf, mux, reqID, r)

			dupper := &responseDupper{ResponseWriter: rw, Buffer: &bytes.Buffer{}}
			h.ServeHTTP(dupper, r)
			writeDebugResponse(buf, reqID, dupper)
			buf.WriteByte('\n')
			w.Write(buf.Bytes()) // nolint: errcheck
		})
	}
}

func requestID(r *http.Request) any {
	reqID := r.Context().Value(middleware.RequestIDKey)
	if reqID == nil {
		return shortID()
	}
	return reqID
}

func writeDebugRequest(buf *bytes.Buffer, mux loomhttp.Muxer, reqID any, r *http.Request) {
	fmt.Fprintf(buf, "> [%s] %s %s", reqID, r.Method, r.URL.String())
	writeDebugMap(buf, reqID, ">", stringMap(r.Header))
	writeDebugMap(buf, reqID, ">", mux.Vars(r))
	body := readDebugBody(r)
	writeDebugBody(buf, reqID, body)
	r.Body = io.NopCloser(bytes.NewBuffer(body))
}

func writeDebugResponse(buf *bytes.Buffer, reqID any, dupper *responseDupper) {
	fmt.Fprintf(buf, "\n< [%s] %s", reqID, http.StatusText(dupper.Status))
	writeDebugMap(buf, reqID, "<", stringMap(dupper.Header()))
	if dupper.Buffer.Len() > 0 {
		writeDebugBody(buf, reqID, []byte(dupper.Buffer.String()))
	}
}

func stringMap(values map[string][]string) map[string]string {
	flat := make(map[string]string, len(values))
	for key, value := range values {
		flat[key] = strings.Join(value, ", ")
	}
	return flat
}

func writeDebugMap(buf *bytes.Buffer, reqID any, prefix string, values map[string]string) {
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

func writeDebugBody(buf *bytes.Buffer, reqID any, body []byte) {
	if len(body) == 0 {
		return
	}
	buf.WriteByte('\n')
	for _, line := range strings.Split(string(body), "\n") {
		fmt.Fprintf(buf, "[%s] %s\n", reqID, line)
	}
}

// Write writes the data to the buffer and connection as part of an HTTP reply.
func (r *responseDupper) Write(b []byte) (int, error) {
	r.Buffer.Write(b)
	return r.ResponseWriter.Write(b)
}

// WriteHeader records the status and sends an HTTP response header with status code.
func (r *responseDupper) WriteHeader(s int) {
	r.Status = s
	r.ResponseWriter.WriteHeader(s)
}

// Hijack supports the http.Hijacker interface.
func (r *responseDupper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := r.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("debug middleware: inner ResponseWriter cannot be hijacked: %T", r.ResponseWriter)
}

// shortID produces a " unique" 6 bytes long string.
// Do not use as a reliable way to get unique IDs, instead use for things like logging.
func shortID() string {
	b := make([]byte, 6)
	io.ReadFull(rand.Reader, b) // nolint: errcheck
	return base64.RawURLEncoding.EncodeToString(b)
}
