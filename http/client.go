package http

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
)

type (
	// Doer is the HTTP client interface.
	Doer interface {
		Do(*http.Request) (*http.Response, error)
	}

	// DebugDoer is a Doer that can print the low level HTTP details.
	DebugDoer interface {
		Doer
		// Fprint prints the HTTP request and response details.
		Fprint(io.Writer)
	}

	// debugDoer wraps a doer and implements DebugDoer.
	debugDoer struct {
		Doer
		// Request is the captured request.
		Request *http.Request
		// Response is the captured response.
		Response *http.Response
	}

	debugReplayBody struct {
		io.Reader
		io.Closer
	}

	// ClientError is an error returned by a HTTP service client.
	ClientError struct {
		// Name is a name for this class of errors.
		Name string
		// Message contains the specific error details.
		Message string
		// Service is the name of the service.
		Service string
		// Method is the name of the service method.
		Method string
		// Is the error temporary?
		Temporary bool
		// Is the error a timeout?
		Timeout bool
		// Is the error a server-side fault?
		Fault bool
		// The original error if any
		Err error
	}
)

const (
	debugBodyCaptureLimit = int64(64 << 10)
	debugRedactedValue    = "[REDACTED]"
)

func (c ClientError) Unwrap() error {
	return c.Err
}

// NewDebugDoer wraps the given doer and captures the request and response so
// they can be printed.
func NewDebugDoer(d Doer) DebugDoer {
	return &debugDoer{Doer: d}
}

// Do captures the request and response.
func (dd *debugDoer) Do(req *http.Request) (*http.Response, error) {
	capturedRequest, err := captureDebugRequest(req)
	if err != nil {
		return nil, err
	}

	resp, err := dd.Doer.Do(req)
	if err != nil {
		return nil, err
	}

	dd.Request = capturedRequest
	dd.Response = captureDebugResponse(resp)

	dd.Fprint(os.Stderr)

	return resp, nil
}

func captureDebugRequest(req *http.Request) (*http.Request, error) {
	captured := req.Clone(req.Context())
	captured.Header = redactDebugHeaders(req.Header)
	captured.URL = redactDebugURL(req.URL)
	if req.Body == nil {
		return captured, nil
	}

	var (
		body      io.ReadCloser
		data      []byte
		truncated bool
		err       error
	)
	if req.GetBody != nil {
		body, err = req.GetBody()
		if err != nil {
			return nil, fmt.Errorf("capture request body: %w", err)
		}
		data, truncated, err = readDebugBody(body)
		if closeErr := body.Close(); closeErr != nil {
			return nil, fmt.Errorf("close captured request body: %w", closeErr)
		}
	} else {
		body = req.Body
		data, truncated, err = readDebugBody(body)
		req.Body = &debugReplayBody{
			Reader: io.MultiReader(bytes.NewReader(data), body),
			Closer: body,
		}
	}
	if err != nil {
		return nil, fmt.Errorf("capture request body: %w", err)
	}
	captured.Body = io.NopCloser(bytes.NewReader(redactDebugBody(req.Header, data, truncated)))
	return captured, nil
}

func redactDebugURL(original *url.URL) *url.URL {
	if original == nil {
		return nil
	}
	redacted := original.Clone()
	if redacted.User != nil {
		username := redacted.User.Username()
		if _, hasPassword := redacted.User.Password(); hasPassword {
			redacted.User = url.UserPassword(username, debugRedactedValue)
		}
	}
	query := redacted.Query()
	for name := range query {
		if isSensitiveDebugName(name) {
			query[name] = []string{debugRedactedValue}
		}
	}
	redacted.RawQuery = query.Encode()
	return redacted
}

func captureDebugResponse(resp *http.Response) *http.Response {
	captured := new(http.Response)
	*captured = *resp
	captured.Header = redactDebugHeaders(resp.Header)
	if resp.Body == nil {
		return captured
	}

	body := resp.Body
	data, truncated, err := readDebugBody(body)
	if err != nil {
		message := fmt.Appendf(nil, "!!failed to read response: %s", err)
		captured.Body = io.NopCloser(bytes.NewReader(message))
		resp.Body = io.NopCloser(bytes.NewReader(message))
		return captured
	}
	resp.Body = &debugReplayBody{
		Reader: io.MultiReader(bytes.NewReader(data), body),
		Closer: body,
	}
	captured.Body = io.NopCloser(bytes.NewReader(redactDebugBody(resp.Header, data, truncated)))
	return captured
}

func readDebugBody(body io.Reader) ([]byte, bool, error) {
	data, err := io.ReadAll(io.LimitReader(body, debugBodyCaptureLimit+1))
	if err != nil {
		return data, false, err
	}
	return data, int64(len(data)) > debugBodyCaptureLimit, nil
}

func redactDebugHeaders(headers http.Header) http.Header {
	redacted := headers.Clone()
	for name := range redacted {
		if isSensitiveDebugName(name) {
			redacted[name] = []string{debugRedactedValue}
		}
	}
	return redacted
}

func redactDebugBody(headers http.Header, data []byte, truncated bool) []byte {
	if truncated {
		return fmt.Appendf(nil, "[body omitted after %d bytes]", debugBodyCaptureLimit)
	}
	if len(data) == 0 {
		return nil
	}

	mediaType, _, err := mime.ParseMediaType(headers.Get("Content-Type"))
	if err != nil {
		return data
	}
	switch {
	case mediaType == "application/json" || strings.HasSuffix(mediaType, "+json"):
		return redactDebugJSON(data)
	case mediaType == "application/x-www-form-urlencoded":
		return redactDebugForm(data)
	default:
		return data
	}
}

func redactDebugJSON(data []byte) []byte {
	value := jsontext.Value(data)
	if !value.IsValid() {
		return []byte("[invalid JSON body omitted]")
	}
	redacted, err := redactDebugJSONValue(value)
	if err != nil {
		return []byte("[JSON body redaction failed]")
	}
	return redacted
}

func redactDebugJSONValue(value jsontext.Value) (jsontext.Value, error) {
	switch value.Kind() {
	case '{':
		var object map[string]jsontext.Value
		if err := json.Unmarshal(value, &object); err != nil {
			return nil, err
		}
		for name, field := range object {
			if isSensitiveDebugName(name) {
				replacement, err := json.Marshal(debugRedactedValue)
				if err != nil {
					return nil, err
				}
				object[name] = replacement
				continue
			}
			redacted, err := redactDebugJSONValue(field)
			if err != nil {
				return nil, err
			}
			object[name] = redacted
		}
		return json.Marshal(object, json.Deterministic(true))
	case '[':
		var items []jsontext.Value
		if err := json.Unmarshal(value, &items); err != nil {
			return nil, err
		}
		for index, item := range items {
			redacted, err := redactDebugJSONValue(item)
			if err != nil {
				return nil, err
			}
			items[index] = redacted
		}
		return json.Marshal(items)
	default:
		return value.Clone(), nil
	}
}

func redactDebugForm(data []byte) []byte {
	values, err := url.ParseQuery(string(data))
	if err != nil {
		return []byte("[invalid form body omitted]")
	}
	for name := range values {
		if isSensitiveDebugName(name) {
			values[name] = []string{debugRedactedValue}
		}
	}
	return []byte(values.Encode())
}

func isSensitiveDebugName(name string) bool {
	normalized := strings.NewReplacer("-", "_", " ", "_").Replace(strings.ToLower(name))
	switch normalized {
	case "authorization", "cookie", "set_cookie", "proxy_authorization",
		"password", "passwd", "secret", "token", "api_key", "x_api_key",
		"access_token", "refresh_token", "client_secret", "credential",
		"credentials", "session", "session_id":
		return true
	}
	return strings.HasSuffix(normalized, "_token") ||
		strings.HasSuffix(normalized, "_secret") ||
		strings.HasSuffix(normalized, "_api_key")
}

// Printf dumps the captured request and response details to w.
func (dd *debugDoer) Fprint(w io.Writer) {
	if dd.Request == nil {
		return
	}
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "> %s %s", dd.Request.Method, dd.Request.URL.String())

	keys := make([]string, len(dd.Request.Header))
	i := 0
	for k := range dd.Request.Header {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(buf, "\n> %s: %s", k, strings.Join(dd.Request.Header[k], ", ")) // nolint: errcheck
	}

	b, _ := io.ReadAll(dd.Request.Body)
	if len(b) > 0 {
		dd.Request.Body = io.NopCloser(bytes.NewBuffer(b)) // reset the request body
		buf.WriteByte('\n')                                // nolint: errcheck
		buf.Write(b)                                       // nolint: errcheck
	}

	if dd.Response == nil {
		w.Write(buf.Bytes()) // nolint: errcheck
		return
	}
	fmt.Fprintf(buf, "\n< %s", dd.Response.Status)

	keys = make([]string, len(dd.Response.Header))
	i = 0
	for k := range dd.Response.Header {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(buf, "\n< %s: %s", k, strings.Join(dd.Response.Header[k], ", ")) // nolint: errcheck
	}

	rb, _ := io.ReadAll(dd.Response.Body) // this is reading from a memory buffer so safe to ignore errors
	if len(rb) > 0 {
		dd.Response.Body = io.NopCloser(bytes.NewBuffer(rb)) // reset the response body
		buf.WriteByte('\n')                                  // nolint: errcheck
		buf.Write(rb)                                        // nolint: errcheck
	}
	w.Write(buf.Bytes())  // nolint: errcheck
	w.Write([]byte{'\n'}) // nolint: errcheck
}

// Error builds an error message.
func (c ClientError) Error() string {
	return fmt.Sprintf("[%s %s]: %s", c.Service, c.Method, c.Message)
}

// ErrInvalidType is the error returned when the wrong type is given to a
// method function.
func ErrInvalidType(svc, m, expected string, actual any) error {
	msg := fmt.Sprintf("invalid value expected %s, got %v", expected, actual)
	return &ClientError{Name: "invalid_type", Message: msg, Service: svc, Method: m}
}

// ErrEncodingError is the error returned when the encoder fails to encode the
// request body.
func ErrEncodingError(svc, m string, err error) error {
	msg := fmt.Sprintf("failed to encode request body: %s", err)
	return &ClientError{Name: "encoding_error", Message: msg, Service: svc, Method: m, Err: err}
}

// ErrInvalidURL is the error returned when the URL computed for an method is
// invalid.
func ErrInvalidURL(svc, m, u string, err error) error {
	msg := fmt.Sprintf("invalid URL %s: %s", u, err)
	return &ClientError{Name: "invalid_url", Message: msg, Service: svc, Method: m, Err: err}
}

// ErrDecodingError is the error returned when the decoder fails to decode the
// response body.
func ErrDecodingError(svc, m string, err error) error {
	msg := fmt.Sprintf("failed to decode response body: %s", err)
	return &ClientError{Name: "decoding_error", Message: msg, Service: svc, Method: m, Err: err}
}

// ErrValidationError is the error returned when the response body is properly
// received and decoded but fails validation.
func ErrValidationError(svc, m string, err error) error {
	msg := fmt.Sprintf("invalid response: %s", err)
	return &ClientError{Name: "validation_error", Message: msg, Service: svc, Method: m, Err: err}
}

// ErrInvalidResponse is the error returned when the service responded with an
// unexpected response status code.
func ErrInvalidResponse(svc, m string, code int, body string) error {
	var b string
	if body != "" {
		b = ", body: "
	}
	msg := fmt.Sprintf("invalid response code %#v"+b+"%s", code, body)

	temporary := code == http.StatusServiceUnavailable ||
		code == http.StatusConflict ||
		code == http.StatusTooManyRequests ||
		code == http.StatusGatewayTimeout

	timeout := code == http.StatusRequestTimeout ||
		code == http.StatusGatewayTimeout

	fault := code == http.StatusInternalServerError ||
		code == http.StatusNotImplemented ||
		code == http.StatusBadGateway

	return &ClientError{Name: "invalid_response", Message: msg, Service: svc, Method: m,
		Temporary: temporary, Timeout: timeout, Fault: fault}
}

// ErrRequestError is the error returned when the request fails to be sent.
func ErrRequestError(svc, m string, err error) error {
	temporary := false
	timeout := false
	var nerr net.Error
	if errors.As(err, &nerr) {
		timeout = nerr.Timeout()
	}
	return &ClientError{Name: "request_error", Message: err.Error(), Service: svc, Method: m,
		Temporary: temporary, Timeout: timeout, Err: err}
}
