package grpc

import (
	"context"

	"google.golang.org/grpc/metadata"
)

type (
	// RequestDecoder is used by the server to decode gRPC request message type
	// and any incoming metadata to a Loom value.
	RequestDecoder func(ctx context.Context, pb any, md metadata.MD) (v any, err error)

	// RequestEncoder is used by the client to encode a Loom value to a gRPC message
	// type and sets the outgoing metadata.
	RequestEncoder func(ctx context.Context, v any, md *metadata.MD) (pb any, err error)

	// ResponseDecoder is used by the client to decode gRPC response message
	// type and any incoming metadata (headers and trailers) to a Loom value.
	ResponseDecoder func(ctx context.Context, pb any, hdr, trlr metadata.MD) (v any, err error)

	// ResponseEncoder is used by the server to encode a Loom value to a gRPC response
	// message type and sets the response headers and trailers.
	ResponseEncoder func(ctx context.Context, v any, hdr, trlr *metadata.MD) (pb any, err error)
)
