package jsonrpc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	loomhttp "github.com/CaliLuke/loom/http"
	loomtransport "github.com/CaliLuke/loom/observability/transport"
	loom "github.com/CaliLuke/loom/pkg"
)

const maxJSONPrefixWhitespace = 4096

type (
	// HTTPDispatch calls the generated typed adapter for request. matched is
	// false when the service does not define the requested method.
	HTTPDispatch func(
		context.Context,
		*http.Request,
		*RawRequest,
		http.ResponseWriter,
	) (matched bool, err error)

	// HTTPHandlerSpec defines the service adapters used by the JSON-RPC HTTP
	// protocol runtime.
	HTTPHandlerSpec struct {
		// Service is the designed service name.
		Service string
		// Decoder returns the decoder for one HTTP request.
		Decoder func(*http.Request) loomhttp.Decoder
		// Encoder returns the encoder for one JSON-RPC response.
		Encoder func(context.Context, http.ResponseWriter) loomhttp.Encoder
		// Dispatch calls the generated typed method adapter.
		Dispatch HTTPDispatch
		// HandleFailure receives transport failures that cannot become JSON-RPC
		// responses.
		HandleFailure func(context.Context, http.ResponseWriter, error)
	}

	requestBody struct {
		io.Reader
		io.Closer
	}

	batchWriter struct {
		writer  io.Writer
		header  http.Header
		written bool
	}

	notificationWriter struct {
		header http.Header
	}
)

// NewHTTPHandler creates a JSON-RPC HTTP handler. The handler owns envelope
// validation, batch framing, notification suppression, and request observation.
func NewHTTPHandler(spec HTTPHandlerSpec) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeHTTP(w, r, spec)
	})
}

// ServeHTTP executes one JSON-RPC HTTP request with the supplied service
// adapters.
func ServeHTTP(w http.ResponseWriter, r *http.Request, spec HTTPHandlerSpec) {
	observer, observedWriter := loomtransport.BeginJSONRPCRequest(
		r.Context(),
		w,
		spec.Service,
		r,
	)
	defer observer.End()
	ctx := loomtransport.WithRequestObserver(r.Context(), observer)
	r = r.WithContext(ctx)

	buffered := bufio.NewReader(r.Body)
	first, err := firstJSONToken(buffered)
	if err != nil {
		observer.Fail(loomtransport.ReasonRequestDecodeFailed)
		if closeErr := r.Body.Close(); closeErr != nil {
			handleHTTPFailure(ctx, observedWriter, spec.HandleFailure, fmt.Errorf("failed to close request body: %w", closeErr))
		}
		handleHTTPFailure(ctx, observedWriter, spec.HandleFailure, fmt.Errorf("failed to read request body: %w", err))
		return
	}
	r.Body = &requestBody{Reader: buffered, Closer: r.Body}
	defer func() {
		if closeErr := r.Body.Close(); closeErr != nil {
			handleHTTPFailure(ctx, observedWriter, spec.HandleFailure, fmt.Errorf("failed to close request body: %w", closeErr))
		}
	}()

	if first == '[' {
		handleBatch(ctx, observedWriter, r, observer, spec)
		return
	}
	handleSingle(ctx, observedWriter, r, observer, spec)
}

// CodeForServiceError maps framework validation failures to JSON-RPC codes.
func CodeForServiceError(err *loom.ServiceError) Code {
	if err == nil {
		return InternalError
	}
	switch err.Name {
	case loom.RequestBodyTooLarge:
		return InvalidRequest
	case loom.InvalidFieldType,
		loom.MissingField,
		loom.InvalidEnumValue,
		loom.InvalidFormat,
		loom.InvalidPattern,
		loom.InvalidRange,
		loom.InvalidLength,
		loom.DecodePayload,
		loom.MissingPayload:
		return InvalidParams
	default:
		return InternalError
	}
}

func firstJSONToken(reader *bufio.Reader) (byte, error) {
	for offset := range maxJSONPrefixWhitespace {
		prefix, err := reader.Peek(offset + 1)
		if len(prefix) <= offset {
			if err == io.EOF {
				return 0, nil
			}
			return 0, fmt.Errorf("failed to read JSON prefix: %w", err)
		}
		value := prefix[offset]
		if value == ' ' || value == '\t' || value == '\n' || value == '\r' {
			continue
		}
		return value, nil
	}
	return 0, nil
}

func handleSingle(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	observer *loomtransport.RequestObserver,
	spec HTTPHandlerSpec,
) {
	var request RawRequest
	if err := spec.Decoder(r).Decode(&request); err != nil {
		observer.Fail(loomtransport.ReasonInvalidJSONRPCEnvelope)
		code, message, data := envelopeDecodeError(err)
		encodeJSONRPCResponse(ctx, w, MakeErrorResponse(nil, code, message, data), observer, spec)
		return
	}
	processRequest(ctx, w, r, &request, 0, observer, spec)
}

func handleBatch(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	observer *loomtransport.RequestObserver,
	spec HTTPHandlerSpec,
) {
	var requests []json.RawMessage
	if err := spec.Decoder(r).Decode(&requests); err != nil {
		observer.Fail(loomtransport.ReasonInvalidJSONRPCBatch)
		code, message, data := envelopeDecodeError(err)
		encodeJSONRPCResponse(ctx, w, MakeErrorResponse(nil, code, message, data), observer, spec)
		return
	}
	if len(requests) == 0 {
		observer.Fail(loomtransport.ReasonInvalidJSONRPCBatch)
		encodeJSONRPCResponse(
			ctx,
			w,
			MakeErrorResponse(nil, InvalidRequest, "Invalid request", nil),
			observer,
			spec,
		)
		return
	}

	observer.SetJSONRPC("", "", len(requests), false)
	w.Header().Set("Content-Type", "application/json")
	writer := &batchWriter{writer: w}
	for _, rawRequest := range requests {
		var request RawRequest
		if err := json.Unmarshal(rawRequest, &request); err != nil {
			observer.Fail(loomtransport.ReasonInvalidJSONRPCEnvelope)
			writeJSONRPCError(
				ctx,
				writer,
				&RawRequest{},
				InvalidRequest,
				"Invalid request",
				observer,
				spec,
			)
			continue
		}
		processRequest(ctx, writer, r, &request, len(requests), observer, spec)
	}
	if err := writer.Close(); err != nil {
		observer.Fail(loomtransport.ReasonResponseWriteFailed)
		handleHTTPFailure(ctx, w, spec.HandleFailure, err)
	}
}

func processRequest(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	request *RawRequest,
	batchCount int,
	observer *loomtransport.RequestObserver,
	spec HTTPHandlerSpec,
) {
	observer.SetJSONRPC(request.Method, IDToString(request.ID), batchCount, !request.HasID)
	if reason, message := invalidRequest(request); reason != loomtransport.ReasonOK {
		observer.Fail(reason)
		writeJSONRPCError(ctx, w, request, InvalidRequest, message, observer, spec)
		return
	}

	dispatchWriter := w
	if !request.HasID {
		dispatchWriter = &notificationWriter{}
	}
	matched, err := spec.Dispatch(ctx, r, request, dispatchWriter)
	if err != nil {
		observer.Fail(loomtransport.ReasonHandlerError)
		handleHTTPFailure(ctx, dispatchWriter, spec.HandleFailure, fmt.Errorf("handler error for %s: %w", request.Method, err))
		return
	}
	if !matched {
		observer.Fail(loomtransport.ReasonUnsupportedMethod)
		writeJSONRPCError(ctx, w, request, MethodNotFound, "Method not found", observer, spec)
	}
}

func writeJSONRPCError(
	ctx context.Context,
	w http.ResponseWriter,
	request *RawRequest,
	code Code,
	message string,
	observer *loomtransport.RequestObserver,
	spec HTTPHandlerSpec,
) {
	if !request.HasID && code != InvalidRequest {
		return
	}
	id := request.ID
	if !request.HasID {
		id = nil
	}
	encodeJSONRPCResponse(ctx, w, MakeErrorResponse(id, code, message, nil), observer, spec)
}

func encodeJSONRPCResponse(
	ctx context.Context,
	w http.ResponseWriter,
	response *Response,
	observer *loomtransport.RequestObserver,
	spec HTTPHandlerSpec,
) {
	if err := spec.Encoder(ctx, w).Encode(response); err != nil {
		observer.Fail(loomtransport.ReasonResponseWriteFailed)
		handleHTTPFailure(ctx, w, spec.HandleFailure, fmt.Errorf("failed to encode JSON-RPC response: %w", err))
	}
}

func envelopeDecodeError(err error) (Code, string, any) {
	var serviceError *loom.ServiceError
	if errors.As(err, &serviceError) && serviceError.Name == loom.RequestBodyTooLarge {
		return CodeForServiceError(serviceError), loom.ErrorSafeMessage(err), NewErrorData(err)
	}
	return ParseError, "Parse error", nil
}

func invalidRequest(request *RawRequest) (loomtransport.Reason, string) {
	if request.Invalid || request.JSONRPC != "2.0" {
		return loomtransport.ReasonInvalidJSONRPCEnvelope, "Invalid request"
	}
	if request.Method == "" {
		return loomtransport.ReasonInvalidJSONRPCMethod, "Missing method field"
	}
	return loomtransport.ReasonOK, ""
}

func handleHTTPFailure(
	ctx context.Context,
	w http.ResponseWriter,
	handleFailure func(context.Context, http.ResponseWriter, error),
	err error,
) {
	if handleFailure != nil {
		handleFailure(ctx, w, err)
	}
}

func (w *batchWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *batchWriter) WriteHeader(int) {
}

func (w *batchWriter) Write(data []byte) (int, error) {
	delimiter := byte(',')
	if !w.written {
		delimiter = '['
	}
	if _, err := w.writer.Write([]byte{delimiter}); err != nil {
		return 0, fmt.Errorf("write JSON-RPC batch delimiter: %w", err)
	}
	w.written = true
	return w.writer.Write(data)
}

func (w *batchWriter) Close() error {
	if !w.written {
		return nil
	}
	if _, err := w.writer.Write([]byte{']'}); err != nil {
		return fmt.Errorf("failed to close JSON-RPC batch response: %w", err)
	}
	return nil
}

func (w *notificationWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *notificationWriter) WriteHeader(int) {
}

func (w *notificationWriter) Write(data []byte) (int, error) {
	return len(data), nil
}
