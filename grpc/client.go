package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type (
	// Invoker invokes a gRPC method. The request and response types
	// are Loom types.
	Invoker interface {
		Invoke(ctx context.Context, req any) (res any, err error)
	}

	// RemoteFunc invokes a RPC method.
	RemoteFunc func(ctx context.Context, reqpb any, opts ...grpc.CallOption) (respb any, err error)

	streamInvoker struct {
		Invoker
	}

	cliInvoker struct {
		encoder RequestEncoder
		decoder ResponseDecoder
		fn      RemoteFunc
	}
)

// NewInvoker returns an invoker to invoke gRPC methods.
func NewInvoker(fn RemoteFunc, enc RequestEncoder, dec ResponseDecoder) Invoker {
	return &cliInvoker{
		fn:      fn,
		encoder: enc,
		decoder: dec,
	}
}

// NewStreamInvoker creates an invoker whose opening context belongs to the
// returned stream. Generated streaming response decoders retain this context;
// terminal receive or active operation cancellation releases it. CloseSend keeps
// it alive. Opening and decoding failures release it before Invoke returns.
func NewStreamInvoker(fn RemoteFunc, enc RequestEncoder, dec ResponseDecoder) Invoker {
	return &streamInvoker{Invoker: NewInvoker(fn, enc, dec)}
}

// Invoke transfers the opening context to a successfully decoded stream.
func (s *streamInvoker) Invoke(ctx context.Context, request any) (response any, err error) {
	ctx, owner := newStreamContext(ctx, true)
	transferred := false
	defer func() {
		if !transferred {
			owner.abort(context.Canceled)
		}
	}()
	response, err = s.Invoker.Invoke(ctx, request)
	transferred = err == nil && response != nil
	return response, err
}

// Invoke invokes the given remote gRPC client method.
func (d *cliInvoker) Invoke(ctx context.Context, req any) (any, error) {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}

	var (
		reqpb any
		err   error
	)
	{
		if d.encoder != nil {
			// Encode gRPC request and outgoing metadata
			if reqpb, err = d.encoder(ctx, req, &md); err != nil {
				return nil, err
			}
		}
	}

	ctx = metadata.NewOutgoingContext(ctx, md)

	var (
		respb any

		hdr  = metadata.MD{}
		trlr = metadata.MD{}
	)
	{
		// Invoke remote method
		if respb, err = d.fn(ctx, reqpb, grpc.Header(&hdr), grpc.Trailer(&trlr)); err != nil {
			return nil, err
		}
	}

	var (
		res any
	)
	{
		if d.decoder != nil {
			// Decode gRPC response and incoming header and trailer metadata
			if res, err = d.decoder(ctx, respb, hdr, trlr); err != nil {
				return nil, err
			}
		}
	}

	return res, nil
}
