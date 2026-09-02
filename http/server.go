package http

import (
	"context"
	"net/http"

	loomtransport "github.com/CaliLuke/loom/observability/transport"
	loom "github.com/CaliLuke/loom/pkg"
)

type (
	// UnaryResult carries an endpoint result to its typed response adapter.
	// Result identifies the designed result type, and Value retains the raw
	// endpoint value for generated validation.
	UnaryResult[Result any] struct {
		// Value is the raw value returned by the service endpoint.
		Value any
	}

	// UnaryHandlerSpec defines the typed adapters and runtime policy for one
	// ordinary unary HTTP endpoint.
	UnaryHandlerSpec[Payload, Result any] struct {
		// Service is the designed service name.
		Service string
		// Method is the designed service method name.
		Method string
		// Decode converts an HTTP request into the service payload. A nil
		// function supplies the zero value of Payload.
		Decode func(*http.Request) (Payload, error)
		// Invoke calls the typed service endpoint.
		Invoke func(context.Context, Payload) (Result, error)
		// EncodeResponse writes a successful typed result.
		EncodeResponse func(context.Context, http.ResponseWriter, Result) error
		// EncodeError writes a request or endpoint error.
		EncodeError func(context.Context, http.ResponseWriter, error) error
		// HandleFailure receives errors that cannot be written as responses.
		HandleFailure func(context.Context, http.ResponseWriter, error)
	}

	// Server is the HTTP server interface used to wrap the server handlers
	// with the given middleware.
	Server interface {
		Use(func(http.Handler) http.Handler)
	}

	// Mounter is the interface for servers that allow mounting their endpoints
	// into a muxer.
	Mounter interface {
		Mount(Muxer)
	}

	// Servers is a list of servers.
	Servers []Server
)

// Use wraps the servers with the given middleware.
func (s Servers) Use(m func(http.Handler) http.Handler) {
	for _, v := range s {
		v.Use(m)
	}
}

// Mount will go through all the servers and mount them into the Muxer. It will
// panic unless all servers satisfy the Mounter interface.
func (s Servers) Mount(mux Muxer) {
	for _, v := range s {
		m := v.(Mounter)
		m.Mount(mux)
	}
}

// AsHandlerFunc adapts handler to the function type required by Muxer.
func AsHandlerFunc(handler http.Handler) http.HandlerFunc {
	if handlerFunc, ok := handler.(http.HandlerFunc); ok {
		return handlerFunc
	}
	return handler.ServeHTTP
}

// MountHandler registers handler for method and pattern on mux. Generated
// servers use this function so handler adaptation remains runtime behavior.
func MountHandler(mux Muxer, method, pattern string, handler http.Handler) {
	mux.Handle(method, pattern, AsHandlerFunc(handler))
}

// NewUnaryHandler creates an HTTP handler from typed endpoint adapters. It
// owns the request context, observation, and failure sequence.
func NewUnaryHandler[Payload, Result any](spec UnaryHandlerSpec[Payload, Result]) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, loom.MethodKey, spec.Method)
		ctx = context.WithValue(ctx, loom.ServiceKey, spec.Service)
		observer, observedWriter := loomtransport.BeginHTTPRequest(ctx, w, spec.Service, spec.Method, r)
		defer observer.End()

		var payload Payload
		if spec.Decode != nil {
			var err error
			payload, err = spec.Decode(r)
			if err != nil {
				observer.Fail(loomtransport.ReasonRequestDecodeFailed)
				encodeUnaryError(ctx, observedWriter, err, spec.EncodeError, spec.HandleFailure)
				return
			}
		}
		result, err := spec.Invoke(ctx, payload)
		if err != nil {
			observer.Fail(loomtransport.ReasonHandlerError)
			encodeUnaryError(ctx, observedWriter, err, spec.EncodeError, spec.HandleFailure)
			return
		}
		if err := spec.EncodeResponse(ctx, observedWriter, result); err != nil {
			observer.Fail(loomtransport.ReasonResponseWriteFailed)
			if responseWriterCommitted(observedWriter) {
				if spec.HandleFailure != nil {
					spec.HandleFailure(ctx, observedWriter, err)
				}
				return
			}
			encodeUnaryError(ctx, observedWriter, err, spec.EncodeError, spec.HandleFailure)
		}
	})
}

func responseWriterCommitted(w http.ResponseWriter) bool {
	capture, ok := w.(interface{ StatusCode() int })
	return ok && capture.StatusCode() != 0
}

func encodeUnaryError(
	ctx context.Context,
	w http.ResponseWriter,
	err error,
	encode func(context.Context, http.ResponseWriter, error) error,
	handleFailure func(context.Context, http.ResponseWriter, error),
) {
	if encodeErr := encode(ctx, w, err); encodeErr != nil && handleFailure != nil {
		handleFailure(ctx, w, encodeErr)
	}
}
