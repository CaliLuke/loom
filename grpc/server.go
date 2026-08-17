package grpc

import (
	"context"
	"errors"
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/runtime/protoiface"

	loomtransport "github.com/CaliLuke/loom/observability/transport"
	loom "github.com/CaliLuke/loom/pkg"
)

type (
	// ErrorMapping describes a designed gRPC status and detail message.
	ErrorMapping struct {
		// Code is the gRPC status code for the designed error.
		Code codes.Code
		// Detail is the designed protocol buffer error detail.
		Detail protoiface.MessageV1
	}

	// ErrorMapper maps a typed service error to its designed gRPC contract.
	// The boolean result reports whether the error has a designed mapping.
	ErrorMapper func(string, error) (ErrorMapping, bool, error)

	// UnaryServerSpec describes one generated unary gRPC method adapter.
	UnaryServerSpec struct {
		// Service is the Loom service name.
		Service string
		// Method is the Loom method name.
		Method string
		// Handler decodes, invokes, and encodes the typed endpoint.
		Handler UnaryHandler
		// MapError maps designed service errors to gRPC status contracts.
		MapError ErrorMapper
	}

	// StreamServerSpec describes one generated streaming gRPC method adapter.
	StreamServerSpec struct {
		// Service is the Loom service name.
		Service string
		// Method is the Loom method name.
		Method string
		// Decode creates the typed endpoint stream input.
		Decode func(context.Context) (any, error)
		// Handle invokes the typed streaming endpoint.
		Handle func(context.Context, any) error
		// MapError maps designed service errors to gRPC status contracts.
		MapError ErrorMapper
	}
)

// ServeUnary executes one unary gRPC request and owns its context, status,
// metadata, and observation lifecycle.
func ServeUnary(ctx context.Context, request any, spec UnaryServerSpec) (any, error) {
	ctx, observer := beginServerRequest(ctx, spec.Service, spec.Method)
	defer observer.End()

	response, err := spec.Handler.Handle(ctx, request)
	if err != nil {
		observer.Fail(loomtransport.ReasonHandlerError)
		return nil, EncodeServerError(err, spec.MapError)
	}
	return response, nil
}

// ServeStream executes one streaming gRPC request and owns its context,
// status, observation, and clean completion lifecycle.
func ServeStream(ctx context.Context, spec StreamServerSpec) error {
	ctx, observer := beginServerRequest(ctx, spec.Service, spec.Method)
	defer observer.End()
	observer.EmitStreamOpen()

	input, err := spec.Decode(ctx)
	if err != nil {
		observer.Fail(loomtransport.ReasonRequestDecodeFailed)
		return EncodeServerError(err, spec.MapError)
	}
	if err := spec.Handle(ctx, input); err != nil {
		observer.Fail(loomtransport.ReasonHandlerError)
		return EncodeServerError(err, spec.MapError)
	}

	observer.EmitStreamClose()
	return nil
}

// EncodeServerError converts an endpoint error to a gRPC status. Designed
// mappings take precedence over Loom's general status conversion.
func EncodeServerError(err error, mapper ErrorMapper) error {
	if err == nil {
		return nil
	}
	if mapper != nil {
		var named loom.LoomErrorNamer
		if errors.As(err, &named) {
			mapping, ok, mapErr := mapper(named.LoomErrorName(), err)
			if mapErr != nil {
				return EncodeError(mapErr)
			}
			if ok {
				if mapping.Detail == nil {
					return NewStatusError(mapping.Code, err)
				}
				return NewStatusError(mapping.Code, err, mapping.Detail)
			}
		}
	}
	return EncodeError(err)
}

// ObserveStreamEncodeError classifies an error that prevented a typed stream
// response from being converted to its protocol buffer representation.
func ObserveStreamEncodeError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	loomtransport.RequestObserverFromContext(ctx).Fail(loomtransport.ReasonResponseWriteFailed)
	return err
}

// ObserveStreamWriteError classifies a failed gRPC stream send and emits the
// corresponding stream failure event.
func ObserveStreamWriteError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	observer := loomtransport.RequestObserverFromContext(ctx)
	observer.Fail(loomtransport.ReasonStreamWriteFailed)
	observer.EmitStreamFailure(loomtransport.ReasonStreamWriteFailed)
	return err
}

// ObserveStreamDecodeError classifies a failed gRPC stream receive or request
// conversion. End-of-stream is a clean completion and is not classified as a
// failure.
func ObserveStreamDecodeError(ctx context.Context, err error) error {
	if err == nil || errors.Is(err, io.EOF) {
		return err
	}
	loomtransport.RequestObserverFromContext(ctx).Fail(loomtransport.ReasonRequestDecodeFailed)
	return err
}

func beginServerRequest(ctx context.Context, service, method string) (context.Context, *loomtransport.RequestObserver) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = context.WithValue(ctx, loom.MethodKey, method)
	ctx = context.WithValue(ctx, loom.ServiceKey, service)
	observer := loomtransport.BeginRequest(ctx, loomtransport.TransportGRPC, service, method)
	ctx = loomtransport.WithRequestObserver(ctx, observer)
	return ctx, observer
}
