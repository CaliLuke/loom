package testdata

var ServerStreamingServerStructCode = `// MethodServerStreamingUserTypeRPCServerStream implements the
// serviceserverstreamingusertyperpc.MethodServerStreamingUserTypeRPCServerStream
// interface.
type MethodServerStreamingUserTypeRPCServerStream struct {
	stream service_server_streaming_user_type_rpcpb.ServiceServerStreamingUserTypeRPC_MethodServerStreamingUserTypeRPCServer
	ctx    context.Context
}
`

var ServerStreamingServerSendCode = `// SendWithContext streams instances of
// "service_server_streaming_user_type_rpcpb.MethodServerStreamingUserTypeRPCResponse"
// to the "MethodServerStreamingUserTypeRPC" endpoint gRPC stream with context.
func (s *MethodServerStreamingUserTypeRPCServerStream) SendWithContext(ctx context.Context, res *serviceserverstreamingusertyperpc.UserType) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	v := NewProtoUserTypeMethodServerStreamingUserTypeRPCResponse(res)
	return loomgrpc.ObserveStreamWriteError(s.ctx, loomgrpc.SendStreamMessage(ctx, s.ctx, v, s.stream.Send))
}

// Send streams instances of
// "service_server_streaming_user_type_rpcpb.MethodServerStreamingUserTypeRPCResponse"
// to the "MethodServerStreamingUserTypeRPC" endpoint gRPC stream.
func (s *MethodServerStreamingUserTypeRPCServerStream) Send(res *serviceserverstreamingusertyperpc.UserType) error {
	return s.SendWithContext(s.ctx, res)
}
`
var ServerStreamingServerCloseCode = `func (s *MethodServerStreamingUserTypeRPCServerStream) Close() error {
	// nothing to do here
	return nil
}
`

var ServerStreamingClientStructCode = `// MethodServerStreamingUserTypeRPCClientStream implements the
// serviceserverstreamingusertyperpc.MethodServerStreamingUserTypeRPCClientStream
// interface.
type MethodServerStreamingUserTypeRPCClientStream struct {
	stream service_server_streaming_user_type_rpcpb.ServiceServerStreamingUserTypeRPC_MethodServerStreamingUserTypeRPCClient
	ctx    context.Context
}
`
var ServerStreamingClientRecvCode = `// RecvWithContext reads instances of
// "service_server_streaming_user_type_rpcpb.MethodServerStreamingUserTypeRPCResponse"
// from the "MethodServerStreamingUserTypeRPC" endpoint gRPC stream with
// context.
func (s *MethodServerStreamingUserTypeRPCClientStream) RecvWithContext(ctx context.Context) (*serviceserverstreamingusertyperpc.UserType, error) {
	var res *serviceserverstreamingusertyperpc.UserType
	if err := ctx.Err(); err != nil {
		return res, err
	}
	v, err := loomgrpc.ReceiveStreamMessage(ctx, s.ctx, false, s.stream.Recv)
	if err != nil {
		return res, err
	}
	return NewMethodServerStreamingUserTypeRPCResponseUserType(v), nil
}

// Recv reads instances of
// "service_server_streaming_user_type_rpcpb.MethodServerStreamingUserTypeRPCResponse"
// from the "MethodServerStreamingUserTypeRPC" endpoint gRPC stream.
func (s *MethodServerStreamingUserTypeRPCClientStream) Recv() (*serviceserverstreamingusertyperpc.UserType, error) {
	return s.RecvWithContext(s.ctx)
}
`
var ServerStreamingResultWithViewsServerStructCode = `// MethodServerStreamingUserTypeRPCServerStream implements the
// serviceserverstreamingusertyperpc.MethodServerStreamingUserTypeRPCServerStream
// interface.
type MethodServerStreamingUserTypeRPCServerStream struct {
	stream service_server_streaming_user_type_rpcpb.ServiceServerStreamingUserTypeRPC_MethodServerStreamingUserTypeRPCServer
	ctx    context.Context
	view   string
}
`

var ServerStreamingResultWithViewsServerSendCode = `// SendWithContext streams instances of
// "service_server_streaming_user_type_rpcpb.MethodServerStreamingUserTypeRPCResponse"
// to the "MethodServerStreamingUserTypeRPC" endpoint gRPC stream with context.
func (s *MethodServerStreamingUserTypeRPCServerStream) SendWithContext(ctx context.Context, res *serviceserverstreamingusertyperpc.ResultType) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	vres, err := serviceserverstreamingusertyperpc.NewViewedResultType(res, s.view)
	if err != nil {
		return loomgrpc.ObserveStreamEncodeError(s.ctx, err)
	}
	v := NewProtoResultTypeViewMethodServerStreamingUserTypeRPCResponse(vres.Projected)
	return loomgrpc.ObserveStreamWriteError(s.ctx, loomgrpc.SendStreamMessage(ctx, s.ctx, v, s.stream.Send))
}

// Send streams instances of
// "service_server_streaming_user_type_rpcpb.MethodServerStreamingUserTypeRPCResponse"
// to the "MethodServerStreamingUserTypeRPC" endpoint gRPC stream.
func (s *MethodServerStreamingUserTypeRPCServerStream) Send(res *serviceserverstreamingusertyperpc.ResultType) error {
	return s.SendWithContext(s.ctx, res)
}
`
var ServerStreamingResultWithViewsServerSetViewCode = `// SetView sets the view.
func (s *MethodServerStreamingUserTypeRPCServerStream) SetView(view string) {
	s.view = view
}
`

var ServerStreamingResultWithViewsClientStructCode = `// MethodServerStreamingUserTypeRPCClientStream implements the
// serviceserverstreamingusertyperpc.MethodServerStreamingUserTypeRPCClientStream
// interface.
type MethodServerStreamingUserTypeRPCClientStream struct {
	stream service_server_streaming_user_type_rpcpb.ServiceServerStreamingUserTypeRPC_MethodServerStreamingUserTypeRPCClient
	ctx    context.Context
	view   string
}
`
var ServerStreamingResultWithViewsClientRecvCode = `// RecvWithContext reads instances of
// "service_server_streaming_user_type_rpcpb.MethodServerStreamingUserTypeRPCResponse"
// from the "MethodServerStreamingUserTypeRPC" endpoint gRPC stream with
// context.
func (s *MethodServerStreamingUserTypeRPCClientStream) RecvWithContext(ctx context.Context) (*serviceserverstreamingusertyperpc.ResultType, error) {
	var res *serviceserverstreamingusertyperpc.ResultType
	if err := ctx.Err(); err != nil {
		return res, err
	}
	v, err := loomgrpc.ReceiveStreamMessage(ctx, s.ctx, false, s.stream.Recv)
	if err != nil {
		return res, err
	}
	proj := NewMethodServerStreamingUserTypeRPCResponseResultTypeView(v)
	vres := &serviceserverstreamingusertyperpcviews.ResultType{Projected: proj, View: s.view}
	if err := serviceserverstreamingusertyperpcviews.ValidateResultType(vres); err != nil {
		return nil, err
	}
	out, err := serviceserverstreamingusertyperpc.NewResultType(vres)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Recv reads instances of
// "service_server_streaming_user_type_rpcpb.MethodServerStreamingUserTypeRPCResponse"
// from the "MethodServerStreamingUserTypeRPC" endpoint gRPC stream.
func (s *MethodServerStreamingUserTypeRPCClientStream) Recv() (*serviceserverstreamingusertyperpc.ResultType, error) {
	return s.RecvWithContext(s.ctx)
}
`
var ServerStreamingResultWithViewsClientSetViewCode = `// SetView sets the view.
func (s *MethodServerStreamingUserTypeRPCClientStream) SetView(view string) {
	s.view = view
}
`

var ServerStreamingResultCollectionWithExplicitViewServerSendCode = `// SendWithContext streams instances of
// "service_server_streaming_result_type_collection_with_explicit_viewpb.ResultTypeCollection"
// to the "MethodServerStreamingResultTypeCollectionWithExplicitView" endpoint
// gRPC stream with context.
func (s *MethodServerStreamingResultTypeCollectionWithExplicitViewServerStream) SendWithContext(ctx context.Context, res serviceserverstreamingresulttypecollectionwithexplicitview.ResultTypeCollection) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	vres, err := serviceserverstreamingresulttypecollectionwithexplicitview.NewViewedResultTypeCollection(res, "tiny")
	if err != nil {
		return loomgrpc.ObserveStreamEncodeError(s.ctx, err)
	}
	v, err := NewProtoResultTypeCollectionViewResultTypeCollection(vres.Projected)
	if err != nil {
		return loomgrpc.ObserveStreamEncodeError(s.ctx, err)
	}
	return loomgrpc.ObserveStreamWriteError(s.ctx, loomgrpc.SendStreamMessage(ctx, s.ctx, v, s.stream.Send))
}

// Send streams instances of
// "service_server_streaming_result_type_collection_with_explicit_viewpb.ResultTypeCollection"
// to the "MethodServerStreamingResultTypeCollectionWithExplicitView" endpoint
// gRPC stream.
func (s *MethodServerStreamingResultTypeCollectionWithExplicitViewServerStream) Send(res serviceserverstreamingresulttypecollectionwithexplicitview.ResultTypeCollection) error {
	return s.SendWithContext(s.ctx, res)
}
`
var ServerStreamingResultCollectionWithExplicitViewClientRecvCode = `// RecvWithContext reads instances of
// "service_server_streaming_result_type_collection_with_explicit_viewpb.ResultTypeCollection"
// from the "MethodServerStreamingResultTypeCollectionWithExplicitView"
// endpoint gRPC stream with context.
func (s *MethodServerStreamingResultTypeCollectionWithExplicitViewClientStream) RecvWithContext(ctx context.Context) (serviceserverstreamingresulttypecollectionwithexplicitview.ResultTypeCollection, error) {
	var res serviceserverstreamingresulttypecollectionwithexplicitview.ResultTypeCollection
	if err := ctx.Err(); err != nil {
		return res, err
	}
	v, err := loomgrpc.ReceiveStreamMessage(ctx, s.ctx, false, s.stream.Recv)
	if err != nil {
		return res, err
	}
	proj, err := NewResultTypeCollectionResultTypeCollection(v)
	if err != nil {
		return nil, err
	}
	vres := serviceserverstreamingresulttypecollectionwithexplicitviewviews.ResultTypeCollection{Projected: proj, View: "tiny"}
	if err := serviceserverstreamingresulttypecollectionwithexplicitviewviews.ValidateResultTypeCollection(vres); err != nil {
		return nil, err
	}
	out, err := serviceserverstreamingresulttypecollectionwithexplicitview.NewResultTypeCollection(vres)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Recv reads instances of
// "service_server_streaming_result_type_collection_with_explicit_viewpb.ResultTypeCollection"
// from the "MethodServerStreamingResultTypeCollectionWithExplicitView"
// endpoint gRPC stream.
func (s *MethodServerStreamingResultTypeCollectionWithExplicitViewClientStream) Recv() (serviceserverstreamingresulttypecollectionwithexplicitview.ResultTypeCollection, error) {
	return s.RecvWithContext(s.ctx)
}
`
var ServerStreamingPrimitiveServerSendCode = `// SendWithContext streams instances of
// "service_server_streaming_rpcpb.MethodServerStreamingRPCResponse" to the
// "MethodServerStreamingRPC" endpoint gRPC stream with context.
func (s *MethodServerStreamingRPCServerStream) SendWithContext(ctx context.Context, res string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	v := NewProtoMethodServerStreamingRPCResponse(res)
	return loomgrpc.ObserveStreamWriteError(s.ctx, loomgrpc.SendStreamMessage(ctx, s.ctx, v, s.stream.Send))
}

// Send streams instances of
// "service_server_streaming_rpcpb.MethodServerStreamingRPCResponse" to the
// "MethodServerStreamingRPC" endpoint gRPC stream.
func (s *MethodServerStreamingRPCServerStream) Send(res string) error {
	return s.SendWithContext(s.ctx, res)
}
`
var ServerStreamingPrimitiveClientRecvCode = `// RecvWithContext reads instances of
// "service_server_streaming_rpcpb.MethodServerStreamingRPCResponse" from the
// "MethodServerStreamingRPC" endpoint gRPC stream with context.
func (s *MethodServerStreamingRPCClientStream) RecvWithContext(ctx context.Context) (string, error) {
	var res string
	if err := ctx.Err(); err != nil {
		return res, err
	}
	v, err := loomgrpc.ReceiveStreamMessage(ctx, s.ctx, false, s.stream.Recv)
	if err != nil {
		return res, err
	}
	if err = ValidateMethodServerStreamingRPCResponse(v); err != nil {
		return res, err
	}
	return NewMethodServerStreamingRPCResponseMethodServerStreamingRPCResponse(v), nil
}

// Recv reads instances of
// "service_server_streaming_rpcpb.MethodServerStreamingRPCResponse" from the
// "MethodServerStreamingRPC" endpoint gRPC stream.
func (s *MethodServerStreamingRPCClientStream) Recv() (string, error) {
	return s.RecvWithContext(s.ctx)
}
`
var ServerStreamingArrayServerSendCode = `// SendWithContext streams instances of
// "service_server_streaming_arraypb.MethodServerStreamingArrayResponse" to the
// "MethodServerStreamingArray" endpoint gRPC stream with context.
func (s *MethodServerStreamingArrayServerStream) SendWithContext(ctx context.Context, res []int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	v := NewProtoMethodServerStreamingArrayResponse(res)
	return loomgrpc.ObserveStreamWriteError(s.ctx, loomgrpc.SendStreamMessage(ctx, s.ctx, v, s.stream.Send))
}

// Send streams instances of
// "service_server_streaming_arraypb.MethodServerStreamingArrayResponse" to the
// "MethodServerStreamingArray" endpoint gRPC stream.
func (s *MethodServerStreamingArrayServerStream) Send(res []int) error {
	return s.SendWithContext(s.ctx, res)
}
`
var ServerStreamingArrayClientRecvCode = `// RecvWithContext reads instances of
// "service_server_streaming_arraypb.MethodServerStreamingArrayResponse" from
// the "MethodServerStreamingArray" endpoint gRPC stream with context.
func (s *MethodServerStreamingArrayClientStream) RecvWithContext(ctx context.Context) ([]int, error) {
	var res []int
	if err := ctx.Err(); err != nil {
		return res, err
	}
	v, err := loomgrpc.ReceiveStreamMessage(ctx, s.ctx, false, s.stream.Recv)
	if err != nil {
		return res, err
	}
	return NewMethodServerStreamingArrayResponseMethodServerStreamingArrayResponse(v), nil
}

// Recv reads instances of
// "service_server_streaming_arraypb.MethodServerStreamingArrayResponse" from
// the "MethodServerStreamingArray" endpoint gRPC stream.
func (s *MethodServerStreamingArrayClientStream) Recv() ([]int, error) {
	return s.RecvWithContext(s.ctx)
}
`
var ServerStreamingMapServerSendCode = `// SendWithContext streams instances of
// "service_server_streaming_mappb.MethodServerStreamingMapResponse" to the
// "MethodServerStreamingMap" endpoint gRPC stream with context.
func (s *MethodServerStreamingMapServerStream) SendWithContext(ctx context.Context, res map[string]*serviceserverstreamingmap.UserType) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	v, err := NewProtoMethodServerStreamingMapResponse(res)
	if err != nil {
		return loomgrpc.ObserveStreamEncodeError(s.ctx, err)
	}
	return loomgrpc.ObserveStreamWriteError(s.ctx, loomgrpc.SendStreamMessage(ctx, s.ctx, v, s.stream.Send))
}

// Send streams instances of
// "service_server_streaming_mappb.MethodServerStreamingMapResponse" to the
// "MethodServerStreamingMap" endpoint gRPC stream.
func (s *MethodServerStreamingMapServerStream) Send(res map[string]*serviceserverstreamingmap.UserType) error {
	return s.SendWithContext(s.ctx, res)
}
`
var ServerStreamingMapClientRecvCode = `// RecvWithContext reads instances of
// "service_server_streaming_mappb.MethodServerStreamingMapResponse" from the
// "MethodServerStreamingMap" endpoint gRPC stream with context.
func (s *MethodServerStreamingMapClientStream) RecvWithContext(ctx context.Context) (map[string]*serviceserverstreamingmap.UserType, error) {
	var res map[string]*serviceserverstreamingmap.UserType
	if err := ctx.Err(); err != nil {
		return res, err
	}
	v, err := loomgrpc.ReceiveStreamMessage(ctx, s.ctx, false, s.stream.Recv)
	if err != nil {
		return res, err
	}
	converted, err := NewMethodServerStreamingMapResponseMethodServerStreamingMapResponse(v)
	if err != nil {
		return res, err
	}
	return converted, nil
}

// Recv reads instances of
// "service_server_streaming_mappb.MethodServerStreamingMapResponse" from the
// "MethodServerStreamingMap" endpoint gRPC stream.
func (s *MethodServerStreamingMapClientStream) Recv() (map[string]*serviceserverstreamingmap.UserType, error) {
	return s.RecvWithContext(s.ctx)
}
`
var ServerStreamingServerRPCSharedResultRecvCode = `// RecvWithContext reads instances of
// "service_server_streaming_rpcpb.MethodServerStreamingRPCResponse" from the
// "MethodServerStreamingRPC" endpoint gRPC stream with context.
func (s *MethodServerStreamingRPCClientStream) RecvWithContext(ctx context.Context) (*serviceserverstreamingrpc.UserType, error) {
	var res *serviceserverstreamingrpc.UserType
	if err := ctx.Err(); err != nil {
		return res, err
	}
	v, err := loomgrpc.ReceiveStreamMessage(ctx, s.ctx, false, s.stream.Recv)
	if err != nil {
		return res, err
	}
	return NewMethodServerStreamingRPCResponseUserType(v), nil
}

// Recv reads instances of
// "service_server_streaming_rpcpb.MethodServerStreamingRPCResponse" from the
// "MethodServerStreamingRPC" endpoint gRPC stream.
func (s *MethodServerStreamingRPCClientStream) Recv() (*serviceserverstreamingrpc.UserType, error) {
	return s.RecvWithContext(s.ctx)
}

// RecvWithContext reads instances of
// "service_server_streaming_rpcpb.OtherMethodServerStreamingRPCResponse" from
// the "OtherMethodServerStreamingRPC" endpoint gRPC stream with context.
func (s *OtherMethodServerStreamingRPCClientStream) RecvWithContext(ctx context.Context) (*serviceserverstreamingrpc.UserType, error) {
	var res *serviceserverstreamingrpc.UserType
	if err := ctx.Err(); err != nil {
		return res, err
	}
	v, err := loomgrpc.ReceiveStreamMessage(ctx, s.ctx, false, s.stream.Recv)
	if err != nil {
		return res, err
	}
	return NewOtherMethodServerStreamingRPCResponseUserType(v), nil
}

// Recv reads instances of
// "service_server_streaming_rpcpb.OtherMethodServerStreamingRPCResponse" from
// the "OtherMethodServerStreamingRPC" endpoint gRPC stream.
func (s *OtherMethodServerStreamingRPCClientStream) Recv() (*serviceserverstreamingrpc.UserType, error) {
	return s.RecvWithContext(s.ctx)
}
`
var ClientStreamingServerStructCode = `// MethodClientStreamingRPCServerStream implements the
// serviceclientstreamingrpc.MethodClientStreamingRPCServerStream interface.
type MethodClientStreamingRPCServerStream struct {
	stream service_client_streaming_rpcpb.ServiceClientStreamingRPC_MethodClientStreamingRPCServer
	ctx    context.Context
}
`

var ClientStreamingServerSendCode = `// SendAndCloseWithContext streams instances of
// "service_client_streaming_rpcpb.MethodClientStreamingRPCResponse" to the
// "MethodClientStreamingRPC" endpoint gRPC stream with context.
func (s *MethodClientStreamingRPCServerStream) SendAndCloseWithContext(ctx context.Context, res string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	v := NewProtoMethodClientStreamingRPCResponse(res)
	return loomgrpc.ObserveStreamWriteError(s.ctx, loomgrpc.SendStreamMessage(ctx, s.ctx, v, s.stream.SendAndClose))
}

// SendAndClose streams instances of
// "service_client_streaming_rpcpb.MethodClientStreamingRPCResponse" to the
// "MethodClientStreamingRPC" endpoint gRPC stream.
func (s *MethodClientStreamingRPCServerStream) SendAndClose(res string) error {
	return s.SendAndCloseWithContext(s.ctx, res)
}
`
var ClientStreamingServerRecvCode = `// RecvWithContext reads instances of
// "service_client_streaming_rpcpb.MethodClientStreamingRPCStreamingRequest"
// from the "MethodClientStreamingRPC" endpoint gRPC stream with context.
func (s *MethodClientStreamingRPCServerStream) RecvWithContext(ctx context.Context) (int, error) {
	var res int
	if err := ctx.Err(); err != nil {
		return res, err
	}
	v, err := loomgrpc.ReceiveStreamMessage(ctx, s.ctx, false, s.stream.Recv)
	if err != nil {
		return res, loomgrpc.ObserveStreamDecodeError(s.ctx, err)
	}
	if err = ValidateMethodClientStreamingRPCStreamingRequest(v); err != nil {
		return res, loomgrpc.ObserveStreamDecodeError(s.ctx, err)
	}
	return NewMethodClientStreamingRPCStreamingRequestMethodClientStreamingRPCStreamingRequest(v), nil
}

// Recv reads instances of
// "service_client_streaming_rpcpb.MethodClientStreamingRPCStreamingRequest"
// from the "MethodClientStreamingRPC" endpoint gRPC stream.
func (s *MethodClientStreamingRPCServerStream) Recv() (int, error) {
	return s.RecvWithContext(s.ctx)
}
`
var ClientStreamingClientStructCode = `// MethodClientStreamingRPCClientStream implements the
// serviceclientstreamingrpc.MethodClientStreamingRPCClientStream interface.
type MethodClientStreamingRPCClientStream struct {
	stream service_client_streaming_rpcpb.ServiceClientStreamingRPC_MethodClientStreamingRPCClient
	ctx    context.Context
}
`
var ClientStreamingClientSendCode = `// SendWithContext streams instances of
// "service_client_streaming_rpcpb.MethodClientStreamingRPCStreamingRequest" to
// the "MethodClientStreamingRPC" endpoint gRPC stream with context.
func (s *MethodClientStreamingRPCClientStream) SendWithContext(ctx context.Context, res int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	v := NewProtoMethodClientStreamingRPCStreamingRequest(res)
	return loomgrpc.SendStreamMessage(ctx, s.ctx, v, s.stream.Send)
}

// Send streams instances of
// "service_client_streaming_rpcpb.MethodClientStreamingRPCStreamingRequest" to
// the "MethodClientStreamingRPC" endpoint gRPC stream.
func (s *MethodClientStreamingRPCClientStream) Send(res int) error {
	return s.SendWithContext(s.ctx, res)
}
`
var ClientStreamingClientRecvCode = `// CloseAndRecvWithContext reads instances of
// "service_client_streaming_rpcpb.MethodClientStreamingRPCResponse" from the
// "MethodClientStreamingRPC" endpoint gRPC stream with context.
func (s *MethodClientStreamingRPCClientStream) CloseAndRecvWithContext(ctx context.Context) (string, error) {
	var res string
	if err := ctx.Err(); err != nil {
		return res, err
	}
	v, err := loomgrpc.ReceiveStreamMessage(ctx, s.ctx, true, s.stream.CloseAndRecv)
	if err != nil {
		return res, err
	}
	if err = ValidateMethodClientStreamingRPCResponse(v); err != nil {
		return res, err
	}
	return NewMethodClientStreamingRPCResponseMethodClientStreamingRPCResponse(v), nil
}

// CloseAndRecv reads instances of
// "service_client_streaming_rpcpb.MethodClientStreamingRPCResponse" from the
// "MethodClientStreamingRPC" endpoint gRPC stream.
func (s *MethodClientStreamingRPCClientStream) CloseAndRecv() (string, error) {
	return s.CloseAndRecvWithContext(s.ctx)
}
`
var ClientStreamingServerNoResultCloseCode = `func (s *MethodClientStreamingNoResultServerStream) Close() error {
	// synchronize stream
	return loomgrpc.ObserveStreamWriteError(s.ctx, loomgrpc.SendStreamMessage(s.ctx, s.ctx, &service_client_streaming_no_resultpb.MethodClientStreamingNoResultResponse{}, s.stream.SendAndClose))
}
`

var ClientStreamingClientNoResultCloseCode = `func (s *MethodClientStreamingNoResultClientStream) Close() error {
	// synchronize and report any server error
	_, err := loomgrpc.ReceiveStreamMessage(s.ctx, s.ctx, true, s.stream.CloseAndRecv)
	return err
}
`

var BidirectionalStreamingServerStructCode = `// MethodBidirectionalStreamingRPCServerStream implements the
// servicebidirectionalstreamingrpc.MethodBidirectionalStreamingRPCServerStream
// interface.
type MethodBidirectionalStreamingRPCServerStream struct {
	stream service_bidirectional_streaming_rpcpb.ServiceBidirectionalStreamingRPC_MethodBidirectionalStreamingRPCServer
	ctx    context.Context
	view   string
}
`

var BidirectionalStreamingServerSendCode = `// SendWithContext streams instances of
// "service_bidirectional_streaming_rpcpb.MethodBidirectionalStreamingRPCResponse"
// to the "MethodBidirectionalStreamingRPC" endpoint gRPC stream with context.
func (s *MethodBidirectionalStreamingRPCServerStream) SendWithContext(ctx context.Context, res *servicebidirectionalstreamingrpc.ID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	vres, err := servicebidirectionalstreamingrpc.NewViewedID(res, "default")
	if err != nil {
		return loomgrpc.ObserveStreamEncodeError(s.ctx, err)
	}
	v := NewProtoIDViewMethodBidirectionalStreamingRPCResponse(vres.Projected)
	return loomgrpc.ObserveStreamWriteError(s.ctx, loomgrpc.SendStreamMessage(ctx, s.ctx, v, s.stream.Send))
}

// Send streams instances of
// "service_bidirectional_streaming_rpcpb.MethodBidirectionalStreamingRPCResponse"
// to the "MethodBidirectionalStreamingRPC" endpoint gRPC stream.
func (s *MethodBidirectionalStreamingRPCServerStream) Send(res *servicebidirectionalstreamingrpc.ID) error {
	return s.SendWithContext(s.ctx, res)
}
`
var BidirectionalStreamingServerRecvCode = `// RecvWithContext reads instances of
// "service_bidirectional_streaming_rpcpb.MethodBidirectionalStreamingRPCStreamingRequest"
// from the "MethodBidirectionalStreamingRPC" endpoint gRPC stream with context.
func (s *MethodBidirectionalStreamingRPCServerStream) RecvWithContext(ctx context.Context) (int, error) {
	var res int
	if err := ctx.Err(); err != nil {
		return res, err
	}
	v, err := loomgrpc.ReceiveStreamMessage(ctx, s.ctx, false, s.stream.Recv)
	if err != nil {
		return res, loomgrpc.ObserveStreamDecodeError(s.ctx, err)
	}
	if err = ValidateMethodBidirectionalStreamingRPCStreamingRequest(v); err != nil {
		return res, loomgrpc.ObserveStreamDecodeError(s.ctx, err)
	}
	return NewMethodBidirectionalStreamingRPCStreamingRequestMethodBidirectionalStreamingRPCStreamingRequest(v), nil
}

// Recv reads instances of
// "service_bidirectional_streaming_rpcpb.MethodBidirectionalStreamingRPCStreamingRequest"
// from the "MethodBidirectionalStreamingRPC" endpoint gRPC stream.
func (s *MethodBidirectionalStreamingRPCServerStream) Recv() (int, error) {
	return s.RecvWithContext(s.ctx)
}
`
var BidirectionalStreamingServerCloseCode = `func (s *MethodBidirectionalStreamingRPCServerStream) Close() error {
	// nothing to do here
	return nil
}
`

var BidirectionalStreamingClientStructCode = `// MethodBidirectionalStreamingRPCClientStream implements the
// servicebidirectionalstreamingrpc.MethodBidirectionalStreamingRPCClientStream
// interface.
type MethodBidirectionalStreamingRPCClientStream struct {
	stream service_bidirectional_streaming_rpcpb.ServiceBidirectionalStreamingRPC_MethodBidirectionalStreamingRPCClient
	ctx    context.Context
	view   string
}
`
var BidirectionalStreamingClientSendCode = `// SendWithContext streams instances of
// "service_bidirectional_streaming_rpcpb.MethodBidirectionalStreamingRPCStreamingRequest"
// to the "MethodBidirectionalStreamingRPC" endpoint gRPC stream with context.
func (s *MethodBidirectionalStreamingRPCClientStream) SendWithContext(ctx context.Context, res int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	v := NewProtoMethodBidirectionalStreamingRPCStreamingRequest(res)
	return loomgrpc.SendStreamMessage(ctx, s.ctx, v, s.stream.Send)
}

// Send streams instances of
// "service_bidirectional_streaming_rpcpb.MethodBidirectionalStreamingRPCStreamingRequest"
// to the "MethodBidirectionalStreamingRPC" endpoint gRPC stream.
func (s *MethodBidirectionalStreamingRPCClientStream) Send(res int) error {
	return s.SendWithContext(s.ctx, res)
}
`
var BidirectionalStreamingClientRecvCode = `// RecvWithContext reads instances of
// "service_bidirectional_streaming_rpcpb.MethodBidirectionalStreamingRPCResponse"
// from the "MethodBidirectionalStreamingRPC" endpoint gRPC stream with context.
func (s *MethodBidirectionalStreamingRPCClientStream) RecvWithContext(ctx context.Context) (*servicebidirectionalstreamingrpc.ID, error) {
	var res *servicebidirectionalstreamingrpc.ID
	if err := ctx.Err(); err != nil {
		return res, err
	}
	v, err := loomgrpc.ReceiveStreamMessage(ctx, s.ctx, false, s.stream.Recv)
	if err != nil {
		return res, err
	}
	proj := NewMethodBidirectionalStreamingRPCResponseIDView(v)
	vres := &servicebidirectionalstreamingrpcviews.ID{Projected: proj, View: "default"}
	if err := servicebidirectionalstreamingrpcviews.ValidateID(vres); err != nil {
		return nil, err
	}
	out, err := servicebidirectionalstreamingrpc.NewID(vres)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Recv reads instances of
// "service_bidirectional_streaming_rpcpb.MethodBidirectionalStreamingRPCResponse"
// from the "MethodBidirectionalStreamingRPC" endpoint gRPC stream.
func (s *MethodBidirectionalStreamingRPCClientStream) Recv() (*servicebidirectionalstreamingrpc.ID, error) {
	return s.RecvWithContext(s.ctx)
}
`
var BidirectionalStreamingClientCloseCode = `func (s *MethodBidirectionalStreamingRPCClientStream) Close() error {
	// Close the send direction of the stream
	return loomgrpc.CloseStreamSend(s.ctx, s.ctx, s.stream.CloseSend)
}
`
