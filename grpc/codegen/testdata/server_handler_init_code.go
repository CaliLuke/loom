package testdata

const (
	UnaryRPCsServerHandlerInitCode = `// NewMethodUnaryRPCAHandler creates a gRPC handler which serves the
// "ServiceUnaryRPCs" service "MethodUnaryRPCA" endpoint.
func NewMethodUnaryRPCAHandler(endpoint loom.Endpoint, h loomgrpc.UnaryHandler) loomgrpc.UnaryHandler {
	if h == nil {
		h = loomgrpc.NewUnaryHandler(endpoint, DecodeMethodUnaryRPCARequest, EncodeMethodUnaryRPCAResponse)
	}
	return h
}

// NewMethodUnaryRPCBHandler creates a gRPC handler which serves the
// "ServiceUnaryRPCs" service "MethodUnaryRPCB" endpoint.
func NewMethodUnaryRPCBHandler(endpoint loom.Endpoint, h loomgrpc.UnaryHandler) loomgrpc.UnaryHandler {
	if h == nil {
		h = loomgrpc.NewUnaryHandler(endpoint, DecodeMethodUnaryRPCBRequest, EncodeMethodUnaryRPCBResponse)
	}
	return h
}
`

	UnaryRPCNoPayloadServerHandlerInitCode = `// NewMethodUnaryRPCNoPayloadHandler creates a gRPC handler which serves the
// "ServiceUnaryRPCNoPayload" service "MethodUnaryRPCNoPayload" endpoint.
func NewMethodUnaryRPCNoPayloadHandler(endpoint loom.Endpoint, h loomgrpc.UnaryHandler) loomgrpc.UnaryHandler {
	if h == nil {
		h = loomgrpc.NewUnaryHandler(endpoint, nil, EncodeMethodUnaryRPCNoPayloadResponse)
	}
	return h
}
`

	UnaryRPCNoResultServerHandlerInitCode = `// NewMethodUnaryRPCNoResultHandler creates a gRPC handler which serves the
// "ServiceUnaryRPCNoResult" service "MethodUnaryRPCNoResult" endpoint.
func NewMethodUnaryRPCNoResultHandler(endpoint loom.Endpoint, h loomgrpc.UnaryHandler) loomgrpc.UnaryHandler {
	if h == nil {
		h = loomgrpc.NewUnaryHandler(endpoint, DecodeMethodUnaryRPCNoResultRequest, EncodeMethodUnaryRPCNoResultResponse)
	}
	return h
}
`

	ServerStreamingRPCServerHandlerInitCode = `// NewMethodServerStreamingRPCHandler creates a gRPC handler which serves the
// "ServiceServerStreamingRPC" service "MethodServerStreamingRPC" endpoint.
func NewMethodServerStreamingRPCHandler(endpoint loom.Endpoint, h loomgrpc.StreamHandler) loomgrpc.StreamHandler {
	if h == nil {
		h = loomgrpc.NewStreamHandler(endpoint, DecodeMethodServerStreamingRPCRequest)
	}
	return h
}
`

	ClientStreamingRPCServerHandlerInitCode = `// NewMethodClientStreamingRPCHandler creates a gRPC handler which serves the
// "ServiceClientStreamingRPC" service "MethodClientStreamingRPC" endpoint.
func NewMethodClientStreamingRPCHandler(endpoint loom.Endpoint, h loomgrpc.StreamHandler) loomgrpc.StreamHandler {
	if h == nil {
		h = loomgrpc.NewStreamHandler(endpoint, nil)
	}
	return h
}
`

	ClientStreamingRPCWithPayloadServerHandlerInitCode = `// NewMethodClientStreamingRPCWithPayloadHandler creates a gRPC handler which
// serves the "ServiceClientStreamingRPCWithPayload" service
// "MethodClientStreamingRPCWithPayload" endpoint.
func NewMethodClientStreamingRPCWithPayloadHandler(endpoint loom.Endpoint, h loomgrpc.StreamHandler) loomgrpc.StreamHandler {
	if h == nil {
		h = loomgrpc.NewStreamHandler(endpoint, DecodeMethodClientStreamingRPCWithPayloadRequest)
	}
	return h
}
`

	BidirectionalStreamingRPCServerHandlerInitCode = `// NewMethodBidirectionalStreamingRPCHandler creates a gRPC handler which
// serves the "ServiceBidirectionalStreamingRPC" service
// "MethodBidirectionalStreamingRPC" endpoint.
func NewMethodBidirectionalStreamingRPCHandler(endpoint loom.Endpoint, h loomgrpc.StreamHandler) loomgrpc.StreamHandler {
	if h == nil {
		h = loomgrpc.NewStreamHandler(endpoint, nil)
	}
	return h
}
`

	BidirectionalStreamingRPCWithPayloadServerHandlerInitCode = `// NewMethodBidirectionalStreamingRPCWithPayloadHandler creates a gRPC handler
// which serves the "ServiceBidirectionalStreamingRPCWithPayload" service
// "MethodBidirectionalStreamingRPCWithPayload" endpoint.
func NewMethodBidirectionalStreamingRPCWithPayloadHandler(endpoint loom.Endpoint, h loomgrpc.StreamHandler) loomgrpc.StreamHandler {
	if h == nil {
		h = loomgrpc.NewStreamHandler(endpoint, DecodeMethodBidirectionalStreamingRPCWithPayloadRequest)
	}
	return h
}
`
)
