package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	loomtransport "github.com/CaliLuke/loom/observability/transport"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

// AdmitStream decodes metadata credentials and authenticates before an initial
// stream receive. Decode must return only the metadata-derived service payload;
// check must be the shared method authentication entrypoint. Complete payload
// validation and endpoint security still run after the initial frame arrives.
// Incoming metadata survives callback context replacement, while RunCheck retains
// cancellation of the original request. Both dependencies are mandatory.
func AdmitStream(ctx context.Context, decode RequestDecoder, check security.EndpointCheck) (context.Context, error) {
	if decode == nil || check == nil {
		panic("stream admission requires a decoder and authentication check")
	}
	md, _ := metadata.FromIncomingContext(ctx)
	payload, err := decode(ctx, nil, md)
	if err != nil {
		return nil, requestDecodeError(ctx, err)
	}
	admitted, err := security.RunCheck(ctx, payload, check)
	if err != nil {
		return nil, err
	}
	// The decoder consumes transport-owned input, independent of authentication's
	// returned context. FromIncomingContext already returns a detached metadata map.
	return metadata.NewIncomingContext(admitted, md), nil
}

func requestDecodeError(ctx context.Context, err error) error {
	loomtransport.RequestObserverFromContext(ctx).Fail(loomtransport.ReasonRequestDecodeFailed)
	var serviceErr *loom.ServiceError
	if errors.As(err, &serviceErr) {
		return err
	}
	decoded := loom.DecodePayloadError(err.Error())
	return NewStatusError(codes.InvalidArgument, decoded, NewErrorResponse(decoded))
}
