package testdata

const UnaryRPCsClientEndpointInitCode = `// MethodUnaryRPCA calls the "MethodUnaryRPCA" function in
// service_unary_rp_cspb.ServiceUnaryRPCsClient interface.
func (c *Client) MethodUnaryRPCA() loom.Endpoint {
	return func(ctx context.Context, v any) (any, error) {
		inv := loomgrpc.NewInvoker(
			BuildMethodUnaryRPCAFunc(c.grpccli, c.opts...),
			EncodeMethodUnaryRPCARequest,
			DecodeMethodUnaryRPCAResponse)
		res, err := inv.Invoke(ctx, v)
		if err != nil {
			return nil, loom.Fault("%s", err.Error())
		}
		return res, nil
	}
}

// MethodUnaryRPCB calls the "MethodUnaryRPCB" function in
// service_unary_rp_cspb.ServiceUnaryRPCsClient interface.
func (c *Client) MethodUnaryRPCB() loom.Endpoint {
	return func(ctx context.Context, v any) (any, error) {
		inv := loomgrpc.NewInvoker(
			BuildMethodUnaryRPCBFunc(c.grpccli, c.opts...),
			EncodeMethodUnaryRPCBRequest,
			DecodeMethodUnaryRPCBResponse)
		res, err := inv.Invoke(ctx, v)
		if err != nil {
			return nil, loom.Fault("%s", err.Error())
		}
		return res, nil
	}
}
`

const UnaryRPCNoPayloadClientEndpointInitCode = `// MethodUnaryRPCNoPayload calls the "MethodUnaryRPCNoPayload" function in
// service_unary_rpc_no_payloadpb.ServiceUnaryRPCNoPayloadClient interface.
func (c *Client) MethodUnaryRPCNoPayload() loom.Endpoint {
	return func(ctx context.Context, v any) (any, error) {
		inv := loomgrpc.NewInvoker(
			BuildMethodUnaryRPCNoPayloadFunc(c.grpccli, c.opts...),
			nil,
			DecodeMethodUnaryRPCNoPayloadResponse)
		res, err := inv.Invoke(ctx, v)
		if err != nil {
			return nil, loom.Fault("%s", err.Error())
		}
		return res, nil
	}
}
`

const UnaryRPCNoResultClientEndpointInitCode = `// MethodUnaryRPCNoResult calls the "MethodUnaryRPCNoResult" function in
// service_unary_rpc_no_resultpb.ServiceUnaryRPCNoResultClient interface.
func (c *Client) MethodUnaryRPCNoResult() loom.Endpoint {
	return func(ctx context.Context, v any) (any, error) {
		inv := loomgrpc.NewInvoker(
			BuildMethodUnaryRPCNoResultFunc(c.grpccli, c.opts...),
			EncodeMethodUnaryRPCNoResultRequest,
			nil)
		res, err := inv.Invoke(ctx, v)
		if err != nil {
			return nil, loom.Fault("%s", err.Error())
		}
		return res, nil
	}
}
`

const UnaryRPCWithErrorsClientEndpointInitCode = `// MethodUnaryRPCWithErrors calls the "MethodUnaryRPCWithErrors" function in
// service_unary_rpc_with_errorspb.ServiceUnaryRPCWithErrorsClient interface.
func (c *Client) MethodUnaryRPCWithErrors() loom.Endpoint {
	return func(ctx context.Context, v any) (any, error) {
		inv := loomgrpc.NewInvoker(
			BuildMethodUnaryRPCWithErrorsFunc(c.grpccli, c.opts...),
			EncodeMethodUnaryRPCWithErrorsRequest,
			DecodeMethodUnaryRPCWithErrorsResponse)
		res, err := inv.Invoke(ctx, v)
		if err != nil {
			resp := loomgrpc.DecodeError(err)
			switch message := resp.(type) {
			case *service_unary_rpc_with_errorspb.MethodUnaryRPCWithErrorsInternalError:
				if err := ValidateMethodUnaryRPCWithErrorsInternalError(message); err != nil {
					return nil, err
				}
				return nil, NewMethodUnaryRPCWithErrorsInternalError(message)
			case *service_unary_rpc_with_errorspb.MethodUnaryRPCWithErrorsBadRequestError:
				if err := ValidateMethodUnaryRPCWithErrorsBadRequestError(message); err != nil {
					return nil, err
				}
				return nil, NewMethodUnaryRPCWithErrorsBadRequestError(message)
			case *service_unary_rpc_with_errorspb.MethodUnaryRPCWithErrorsCustomErrorError:
				return nil, NewMethodUnaryRPCWithErrorsCustomErrorError(message)
			case *loompb.ErrorResponse:
				return nil, loomgrpc.NewServiceError(message)
			default:
				return nil, loom.Fault("%s", err.Error())
			}
		}
		return res, nil
	}
}
`

const UnaryRPCAcronymClientEndpointInitCode = `// MethodUnaryRPCAcronymJWT calls the "MethodUnaryRPCAcronymJWT" function in
// service_unary_rpc_acronympb.ServiceUnaryRPCAcronymClient interface.
func (c *Client) MethodUnaryRPCAcronymJWT() loom.Endpoint {
	return func(ctx context.Context, v any) (any, error) {
		inv := loomgrpc.NewInvoker(
			BuildMethodUnaryRPCAcronymJWTFunc(c.grpccli, c.opts...),
			nil,
			nil)
		res, err := inv.Invoke(ctx, v)
		if err != nil {
			return nil, loom.Fault("%s", err.Error())
		}
		return res, nil
	}
}
`

const ServerStreamingRPCClientEndpointInitCode = `// MethodServerStreamingRPC calls the "MethodServerStreamingRPC" function in
// service_server_streaming_rpcpb.ServiceServerStreamingRPCClient interface.
func (c *Client) MethodServerStreamingRPC() loom.Endpoint {
	return func(ctx context.Context, v any) (any, error) {
		inv := loomgrpc.NewInvoker(
			BuildMethodServerStreamingRPCFunc(c.grpccli, c.opts...),
			EncodeMethodServerStreamingRPCRequest,
			DecodeMethodServerStreamingRPCResponse)
		res, err := inv.Invoke(ctx, v)
		if err != nil {
			return nil, loom.Fault("%s", err.Error())
		}
		return res, nil
	}
}
`

const ClientStreamingRPCClientEndpointInitCode = `// MethodClientStreamingRPC calls the "MethodClientStreamingRPC" function in
// service_client_streaming_rpcpb.ServiceClientStreamingRPCClient interface.
func (c *Client) MethodClientStreamingRPC() loom.Endpoint {
	return func(ctx context.Context, v any) (any, error) {
		inv := loomgrpc.NewInvoker(
			BuildMethodClientStreamingRPCFunc(c.grpccli, c.opts...),
			nil,
			DecodeMethodClientStreamingRPCResponse)
		res, err := inv.Invoke(ctx, v)
		if err != nil {
			return nil, loom.Fault("%s", err.Error())
		}
		return res, nil
	}
}
`

const ClientStreamingNoResultClientEndpointInitCode = `// MethodClientStreamingNoResult calls the "MethodClientStreamingNoResult"
// function in
// service_client_streaming_no_resultpb.ServiceClientStreamingNoResultClient
// interface.
func (c *Client) MethodClientStreamingNoResult() loom.Endpoint {
	return func(ctx context.Context, v any) (any, error) {
		inv := loomgrpc.NewInvoker(
			BuildMethodClientStreamingNoResultFunc(c.grpccli, c.opts...),
			nil,
			DecodeMethodClientStreamingNoResultResponse)
		res, err := inv.Invoke(ctx, v)
		if err != nil {
			return nil, loom.Fault("%s", err.Error())
		}
		return res, nil
	}
}
`

const ClientStreamingRPCWithPayloadClientEndpointInitCode = `// MethodClientStreamingRPCWithPayload calls the
// "MethodClientStreamingRPCWithPayload" function in
// service_client_streaming_rpc_with_payloadpb.ServiceClientStreamingRPCWithPayloadClient
// interface.
func (c *Client) MethodClientStreamingRPCWithPayload() loom.Endpoint {
	return func(ctx context.Context, v any) (any, error) {
		inv := loomgrpc.NewInvoker(
			BuildMethodClientStreamingRPCWithPayloadFunc(c.grpccli, c.opts...),
			EncodeMethodClientStreamingRPCWithPayloadRequest,
			DecodeMethodClientStreamingRPCWithPayloadResponse)
		res, err := inv.Invoke(ctx, v)
		if err != nil {
			return nil, loom.Fault("%s", err.Error())
		}
		return res, nil
	}
}
`

const BidirectionalStreamingRPCClientEndpointInitCode = `// MethodBidirectionalStreamingRPC calls the "MethodBidirectionalStreamingRPC"
// function in
// service_bidirectional_streaming_rpcpb.ServiceBidirectionalStreamingRPCClient
// interface.
func (c *Client) MethodBidirectionalStreamingRPC() loom.Endpoint {
	return func(ctx context.Context, v any) (any, error) {
		inv := loomgrpc.NewInvoker(
			BuildMethodBidirectionalStreamingRPCFunc(c.grpccli, c.opts...),
			nil,
			DecodeMethodBidirectionalStreamingRPCResponse)
		res, err := inv.Invoke(ctx, v)
		if err != nil {
			return nil, loom.Fault("%s", err.Error())
		}
		return res, nil
	}
}
`

const BidirectionalStreamingRPCWithPayloadClientEndpointInitCode = `// MethodBidirectionalStreamingRPCWithPayload calls the
// "MethodBidirectionalStreamingRPCWithPayload" function in
// service_bidirectional_streaming_rpc_with_payloadpb.ServiceBidirectionalStreamingRPCWithPayloadClient
// interface.
func (c *Client) MethodBidirectionalStreamingRPCWithPayload() loom.Endpoint {
	return func(ctx context.Context, v any) (any, error) {
		inv := loomgrpc.NewInvoker(
			BuildMethodBidirectionalStreamingRPCWithPayloadFunc(c.grpccli, c.opts...),
			EncodeMethodBidirectionalStreamingRPCWithPayloadRequest,
			DecodeMethodBidirectionalStreamingRPCWithPayloadResponse)
		res, err := inv.Invoke(ctx, v)
		if err != nil {
			return nil, loom.Fault("%s", err.Error())
		}
		return res, nil
	}
}
`

const BidirectionalStreamingRPCWithErrorsClientEndpointInitCode = `// MethodBidirectionalStreamingRPCWithErrors calls the
// "MethodBidirectionalStreamingRPCWithErrors" function in
// service_bidirectional_streaming_rpc_with_errorspb.ServiceBidirectionalStreamingRPCWithErrorsClient
// interface.
func (c *Client) MethodBidirectionalStreamingRPCWithErrors() loom.Endpoint {
	return func(ctx context.Context, v any) (any, error) {
		inv := loomgrpc.NewInvoker(
			BuildMethodBidirectionalStreamingRPCWithErrorsFunc(c.grpccli, c.opts...),
			nil,
			DecodeMethodBidirectionalStreamingRPCWithErrorsResponse)
		res, err := inv.Invoke(ctx, v)
		if err != nil {
			resp := loomgrpc.DecodeError(err)
			switch message := resp.(type) {
			case *loompb.ErrorResponse:
				return nil, loomgrpc.NewServiceError(message)
			default:
				return nil, loom.Fault("%s", err.Error())
			}
		}
		return res, nil
	}
}
`
