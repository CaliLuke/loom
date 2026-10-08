package http

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json/v2"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/CaliLuke/loom/internal/httpcodec"
	loom "github.com/CaliLuke/loom/pkg"
)

const (
	// AcceptTypeKey is the context key used to store the value of the HTTP
	// request Accept-Type header. The value may be used by encoders and
	// decoders to implement a content type negotiation algorithm.
	AcceptTypeKey contextKey = iota + 1

	// ContentTypeKey is the context key used to store the value of the HTTP
	// response Content-Type header when explicitly set in the DSL. The value
	// may be used by encoders to set the header appropriately.
	ContentTypeKey

	// LastEventIDKey is the context key used to store the Last-Event-ID
	// request header for Server-Sent Events endpoints.
	LastEventIDKey

	// DefaultMaxRequestBodyBytes is the default maximum number of request-body
	// bytes decoded by Loom's built-in HTTP decoders.
	DefaultMaxRequestBodyBytes = 32 << 20

	// DefaultMaxErrorBodyBytes is the default maximum number of unexpected
	// response-body bytes included in client-side response errors.
	DefaultMaxErrorBodyBytes = 64 << 10
)

type (
	// Decoder provides the actual decoding algorithm used to load HTTP
	// request and response bodies.
	Decoder interface {
		// Decode decodes into v.
		Decode(v any) error
	}

	// Encoder provides the actual encoding algorithm used to write HTTP
	// request and response bodies.
	Encoder interface {
		// Encode encodes v.
		Encode(v any) error
	}

	// EncodingFunc allows a function with appropriate signature to act as a
	// Decoder/Encoder.
	EncodingFunc func(v any) error

	// private type used to define context keys.
	contextKey int

	decodeMode uint8

	jsonResponseEncoder struct {
		writer io.Writer
	}
)

const (
	decodeResponse decodeMode = iota
	decodeRequest
)

// RequestDecoder returns a HTTP request body decoder suitable for the given
// request. The decoder handles the following mime types:
//
//   - application/json using package encoding/json/v2
//   - application/xml using package encoding/xml
//   - application/gob using package encoding/gob
//   - text/html and text/plain for strings
//
// RequestDecoder defaults to JSON when Content-Type is missing and rejects
// unsupported media types.
func RequestDecoder(r *http.Request) Decoder {
	limit := RequestBodyLimit(r.Context())
	selected := httpcodec.RequestContentType(r.Header.Get("Content-Type"))
	switch selected.Kind {
	case httpcodec.JSON:
		decode := decodeJSON
		if rejectNull, _ := r.Context().Value(nonNullableBodyKey{}).(bool); rejectNull {
			decode = decodeNonNullableJSON
		}
		return newLimitedDecoder(r.Body, decode, decodeRequest, limit)
	case httpcodec.GOB:
		return newLimitedDecoder(r.Body, decodeGOB, decodeRequest, limit)
	case httpcodec.XML:
		return newLimitedDecoder(r.Body, decodeXML, decodeRequest, limit)
	case httpcodec.Text:
		return newTextDecoder(r.Body, selected.Media, decodeRequest, limit)
	default:
		return newUnsupportedDecoder(selected.Media)
	}
}

// ResponseEncoder returns a HTTP response encoder leveraging the mime type
// set in the context under the AcceptTypeKey or the ContentTypeKey if any.
// The encoder supports the following mime types:
//
//   - application/json using package encoding/json/v2
//   - application/xml using package encoding/xml
//   - application/gob using package encoding/gob
//   - text/html and text/plain for strings
//
// ResponseEncoder defaults to the JSON encoder if the context AcceptTypeKey or
// ContentTypeKey value does not match any of the supported mime types or is
// missing altogether.
func ResponseEncoder(ctx context.Context, w http.ResponseWriter) Encoder {
	accept := stringContextValue(ctx, AcceptTypeKey)
	ct := stringContextValue(ctx, ContentTypeKey)
	if ct != "" {
		enc := selectedResponseEncoder(w, httpcodec.ResponseContentType(ct))
		SetContentType(w, ct)
		return enc
	}
	enc, mt := negotiatedResponseEncoder(w, accept)
	SetContentType(w, mt)
	return enc
}

func stringContextValue(ctx context.Context, key any) string {
	if value := ctx.Value(key); value != nil {
		return value.(string)
	}
	return ""
}

func requestAcceptHeader(req *http.Request) string {
	if req == nil {
		return ""
	}
	return strings.Join(req.Header.Values("Accept"), ",")
}

func negotiatedResponseEncoder(w http.ResponseWriter, accept string) (Encoder, string) {
	selected := httpcodec.ResponseAccept(accept)
	return selectedResponseEncoder(w, selected), selected.Media
}

func selectedResponseEncoder(w http.ResponseWriter, selected httpcodec.Selection) Encoder {
	switch selected.Kind {
	case httpcodec.JSON:
		return newJSONResponseEncoder(w)
	case httpcodec.XML:
		return xml.NewEncoder(w)
	case httpcodec.GOB:
		return gob.NewEncoder(w)
	case httpcodec.Text:
		return newTextEncoder(w, selected.Media)
	default:
		return nil
	}
}

// RequestEncoder returns a HTTP request encoder.
// The encoder uses package encoding/json/v2.
func RequestEncoder(r *http.Request) Encoder {
	const k = "Content-Type"
	if h := r.Header.Get(k); h == "" {
		r.Header.Set(k, "application/json")
	}
	enc := new(jsonEncoder)
	r.Body = enc
	// GetBody enables request retry on HTTP/2 connections when the server
	// sends GOAWAY during graceful shutdown. Without GetBody, the HTTP transport
	// cannot retry because the request body has already been consumed.
	r.GetBody = enc.GetBody
	return enc
}

// jsonEncoder implements io.ReadCloser and provides GetBody functionality
// to support HTTP/2 request retries during server graceful shutdown (GOAWAY).
type jsonEncoder struct {
	b []byte
	r bytes.Reader
}

var errEncodeNotCalled = errors.New("RequestEncoder: Encode must be called prior to reading")

var errRequestBodyTooLarge = errors.New("request body too large")

func (je *jsonEncoder) Read(b []byte) (n int, err error) {
	if len(je.b) == 0 {
		return 0, errEncodeNotCalled
	}
	return je.r.Read(b)
}

func (*jsonEncoder) Close() (err error) { return nil }

func (je *jsonEncoder) Encode(v any) error {
	b, err := json.Marshal(v, loom.JSONOptions())
	if err != nil {
		return err
	}
	je.b = b
	je.r = *bytes.NewReader(b)
	return nil
}

func newJSONResponseEncoder(w io.Writer) *jsonResponseEncoder {
	return &jsonResponseEncoder{writer: w}
}

func (e *jsonResponseEncoder) Encode(v any) error {
	data, err := json.Marshal(v, loom.JSONOptions())
	if err != nil {
		return err
	}
	data = append(data, '\n')
	written, err := e.writer.Write(data)
	if err != nil {
		return err
	}
	if written != len(data) {
		return io.ErrShortWrite
	}
	return nil
}

// GetBody returns a new reader of the encoded bytes, enabling request retries.
// This is required for HTTP/2 connections to handle server GOAWAY during graceful shutdown.
func (je *jsonEncoder) GetBody() (io.ReadCloser, error) {
	if len(je.b) == 0 {
		return nil, errEncodeNotCalled
	}
	return io.NopCloser(bytes.NewReader(je.b)), nil
}

// ResponseDecoder returns a HTTP response decoder.
// The decoder handles the following content types:
//
//   - application/json using package encoding/json/v2 (default)
//   - application/xml using package encoding/xml
//   - application/gob using package encoding/gob
//   - text/html and text/plain for strings
func ResponseDecoder(resp *http.Response) Decoder {
	selected := httpcodec.ResponseDecode(resp.Header.Get("Content-Type"))
	switch selected.Kind {
	case httpcodec.XML:
		return newLimitedDecoder(resp.Body, decodeXML, decodeResponse, DefaultMaxRequestBodyBytes)
	case httpcodec.GOB:
		return newLimitedDecoder(resp.Body, decodeGOB, decodeResponse, DefaultMaxRequestBodyBytes)
	case httpcodec.Text:
		return newTextDecoder(resp.Body, selected.Media, decodeResponse, DefaultMaxRequestBodyBytes)
	default:
		return newLimitedDecoder(resp.Body, decodeJSON, decodeResponse, DefaultMaxRequestBodyBytes)
	}
}

// ErrorEncoder returns an encoder that encodes errors returned by service
// methods. The default encoder checks whether the error is a Loom ServiceError
// struct and if so uses the error temporary and timeout fields to infer a
// proper HTTP status code and marshals the error struct to the body using the
// provided encoder. If the error is not a Loom ServiceError struct then it is
// encoded as a permanent internal server error. This behavior as well as the
// shape of the response can be overridden by providing a non-nil formatter.
func ErrorEncoder(encoder func(context.Context, http.ResponseWriter) Encoder, formatter func(ctx context.Context, err error) StatusCoder) func(context.Context, http.ResponseWriter, error) error {
	defaultFormatter := formatter == nil
	if formatter == nil {
		formatter = NewErrorResponse
	}
	return func(ctx context.Context, w http.ResponseWriter, err error) error {
		if defaultFormatter {
			ctx = context.WithValue(ctx, ContentTypeKey, ProblemJSONContentType)
		}
		enc := encoder(ctx, w)
		resp := formatter(ctx, err)
		w.WriteHeader(resp.StatusCode())
		return enc.Encode(resp)
	}
}

// ReadUnexpectedResponseBody reads the bounded body text used in generated
// client-side invalid-response errors.
func ReadUnexpectedResponseBody(resp *http.Response) (string, error) {
	if resp == nil || resp.Body == nil {
		return "", nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, DefaultMaxErrorBodyBytes))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// ReadResponseBody reads the bounded response body used when generated clients
// restore the response body after decoding. Read failures return any captured
// prefix along with the error; oversized bodies return no bytes.
func ReadResponseBody(resp *http.Response) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, nil
	}
	return readAllLimited(resp.Body, DefaultMaxRequestBodyBytes)
}

// SafeDecodePayloadMessage returns a stable client-facing description for a
// request body decode failure without exposing Go decoder internals.
func SafeDecodePayloadMessage(error) string {
	return "invalid request body"
}

// Decode implements the Decoder interface. It simply calls f(v).
func (f EncodingFunc) Decode(v any) error { return f(v) }

// Encode implements the Encoder interface. It simply calls f(v).
func (f EncodingFunc) Encode(v any) error { return f(v) }

// SetContentType initializes the response Content-Type header given a MIME
// type. If the Content-Type header is already set and the MIME type is
// "application/json" or "application/xml" then SetContentType appends a suffix
// to the header ("+json" or "+xml" respectively).
func SetContentType(w http.ResponseWriter, ct string) {
	h := w.Header().Get("Content-Type")
	if h == "" {
		w.Header().Set("Content-Type", ct)
		return
	}
	// RFC6839 only defines suffixes for a few mime types, we only concern
	// ourselves with JSON and XML.
	if ct != "application/json" && ct != "application/xml" {
		w.Header().Set("Content-Type", ct)
		return
	}
	if strings.Contains(h, "+") {
		return
	}
	suffix := "+json"
	if ct == "application/xml" {
		suffix = "+xml"
	}
	w.Header().Set("Content-Type", h+suffix)
}

func newTextEncoder(w io.Writer, ct string) Encoder {
	return &textEncoder{w, ct}
}

type textEncoder struct {
	w  io.Writer
	ct string
}

func (e *textEncoder) Encode(v any) (err error) {
	switch c := v.(type) {
	case string:
		_, err = e.w.Write([]byte(c))
	case *string: // v may be a string pointer when the Response Body is set to the field of a custom response type.
		_, err = e.w.Write([]byte(*c))
	case []byte:
		_, err = e.w.Write(c)
	default:
		err = fmt.Errorf("can't encode %T as %s", c, e.ct)
	}
	return
}

func newTextDecoder(r io.Reader, ct string, mode decodeMode, limit int64) Decoder {
	return &textDecoder{r: r, ct: ct, mode: mode, limit: limit}
}

func newLimitedDecoder(r io.Reader, decode func([]byte, any) error, mode decodeMode, limit int64) Decoder {
	return &limitedDecoder{r: r, decode: decode, mode: mode, limit: limit}
}

type textDecoder struct {
	r     io.Reader
	ct    string
	mode  decodeMode
	limit int64
}

type limitedDecoder struct {
	r      io.Reader
	decode func([]byte, any) error
	mode   decodeMode
	limit  int64
}

func (d *limitedDecoder) Decode(v any) error {
	b, err := readAllLimited(d.r, d.limit)
	if err != nil {
		return requestBodyDecodeError(err, d.mode)
	}
	return d.decode(b, v)
}

func (e *textDecoder) Decode(v any) error {
	b, err := readAllLimited(e.r, e.limit)
	if err != nil {
		return requestBodyDecodeError(err, e.mode)
	}
	switch c := v.(type) {
	case *string:
		*c = string(b)
	case *[]byte:
		*c = b
	default:
		return fmt.Errorf("can't decode %s to %T", e.ct, c)
	}
	return nil
}

func decodeJSON(data []byte, v any) error {
	if jsonWhitespaceOnly(data) {
		return io.EOF
	}
	return json.Unmarshal(data, v, loom.JSONOptions())
}

func decodeXML(data []byte, v any) error {
	return xml.NewDecoder(bytes.NewReader(data)).Decode(v)
}

func decodeGOB(data []byte, v any) error {
	return gob.NewDecoder(bytes.NewReader(data)).Decode(v)
}

func jsonWhitespaceOnly(data []byte) bool {
	for _, b := range data {
		switch b {
		case ' ', '\t', '\r', '\n':
		default:
			return false
		}
	}
	return true
}

func readAllLimited(r io.Reader, limit int64) ([]byte, error) {
	limited := io.LimitReader(r, limit+1)
	b, err := io.ReadAll(limited)
	if err != nil {
		return b, err
	}
	if int64(len(b)) > limit {
		return nil, errRequestBodyTooLarge
	}
	return b, nil
}

func requestBodyDecodeError(err error, mode decodeMode) error {
	if mode == decodeRequest {
		return NormalizeRequestBodyDecodeError(err)
	}
	return err
}

// newUnsupportedDecoder returns a decoder that returns an error indicating that
// the content type is not supported.
func newUnsupportedDecoder(ct string) Decoder {
	return &unsupportedDecoder{ct}
}

type unsupportedDecoder struct {
	ct string
}

func (e *unsupportedDecoder) Decode(_ any) error {
	return loom.UnsupportedMediaTypeError(e.ct)
}
