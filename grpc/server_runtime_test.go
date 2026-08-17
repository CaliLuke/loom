package grpc

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	stdgrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	loompb "github.com/CaliLuke/loom/grpc/pb"
	loomtransport "github.com/CaliLuke/loom/observability/transport"
	loom "github.com/CaliLuke/loom/pkg"
)

type (
	serverEventRecorder struct {
		mu     sync.Mutex
		events []loomtransport.Event
	}

	runtimeBufconnServer struct {
		observer loomtransport.Observer
	}

	runtimeBufconnService interface {
		runtimeBufconnService()
	}
)

func (r *serverEventRecorder) ObserveEvent(_ context.Context, event loomtransport.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}

func (r *serverEventRecorder) snapshot() []loomtransport.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]loomtransport.Event(nil), r.events...)
}

func (*runtimeBufconnServer) runtimeBufconnService() {}

func (s *runtimeBufconnServer) unary(ctx context.Context, request *loompb.ErrorResponse) (*loompb.ErrorResponse, error) {
	ctx = loomtransport.WithObserver(ctx, s.observer)
	handler := NewUnaryHandler(
		func(endpointCtx context.Context, input any) (any, error) {
			value := input.(*loompb.ErrorResponse)
			if value.Name == "fail" {
				return nil, loom.PermanentError("denied", "request denied")
			}
			value.Msg, _ = endpointCtx.Value(loom.MethodKey).(string)
			return value, nil
		},
		func(_ context.Context, value any, md metadata.MD) (any, error) {
			request := value.(*loompb.ErrorResponse)
			if requestIDs := md.Get("request-id"); len(requestIDs) > 0 {
				request.Id = requestIDs[0]
			}
			return request, nil
		},
		func(_ context.Context, value any, header, trailer *metadata.MD) (any, error) {
			header.Set("response-header", "present")
			trailer.Set("response-trailer", "present")
			return value, nil
		},
	)
	response, err := ServeUnary(ctx, request, UnaryServerSpec{
		Service:  "Runtime",
		Method:   "Unary",
		Handler:  handler,
		MapError: runtimeTestErrorMapper,
	})
	if err != nil {
		return nil, err
	}
	return response.(*loompb.ErrorResponse), nil
}

func (s *runtimeBufconnServer) stream(request *loompb.ErrorResponse, stream stdgrpc.ServerStream) error {
	ctx := loomtransport.WithObserver(stream.Context(), s.observer)
	handler := NewStreamHandler(
		func(_ context.Context, input any) (any, error) {
			value := input.(*loompb.ErrorResponse)
			for _, message := range []string{value.Name + "-one", value.Name + "-two"} {
				if err := stream.SendMsg(&loompb.ErrorResponse{Name: message}); err != nil {
					return nil, err
				}
			}
			return nil, nil
		},
		func(_ context.Context, value any, _ metadata.MD) (any, error) {
			return value, nil
		},
	)
	return ServeStream(ctx, StreamServerSpec{
		Service: "Runtime",
		Method:  "Stream",
		Decode: func(ctx context.Context) (any, error) {
			return handler.Decode(ctx, request)
		},
		Handle:   handler.Handle,
		MapError: runtimeTestErrorMapper,
	})
}

func runtimeTestErrorMapper(name string, err error) (ErrorMapping, bool, error) {
	if name != "denied" {
		return ErrorMapping{}, false, nil
	}
	return ErrorMapping{Code: codes.PermissionDenied, Detail: NewErrorResponse(err)}, true, nil
}

func TestServerRuntimeUnaryBufconnLifecycle(t *testing.T) {
	recorder := &serverEventRecorder{}
	connection := startRuntimeBufconnServer(t, recorder)
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("request-id", "request-42"))
	var header metadata.MD
	var trailer metadata.MD
	response := &loompb.ErrorResponse{}

	err := connection.Invoke(
		ctx,
		"/loom.test.Runtime/Unary",
		&loompb.ErrorResponse{Name: "ok"},
		response,
		stdgrpc.Header(&header),
		stdgrpc.Trailer(&trailer),
	)

	require.NoError(t, err)
	require.Equal(t, "ok", response.Name)
	require.Equal(t, "request-42", response.Id)
	require.Equal(t, "Unary", response.Msg)
	require.Equal(t, []string{"present"}, header.Get("response-header"))
	require.Equal(t, []string{"present"}, trailer.Get("response-trailer"))
	events := recorder.snapshot()
	require.Len(t, events, 2)
	require.Equal(t, loomtransport.EventKindRequestStart, events[0].Kind)
	require.Equal(t, loomtransport.EventKindRequestFinish, events[1].Kind)
	require.Equal(t, loomtransport.TransportGRPC, events[1].Transport)
	require.Equal(t, "Runtime", events[1].Service)
	require.Equal(t, "Unary", events[1].Method)
}

func TestServerRuntimeMapsDesignedStatusOverBufconn(t *testing.T) {
	recorder := &serverEventRecorder{}
	connection := startRuntimeBufconnServer(t, recorder)

	err := connection.Invoke(
		context.Background(),
		"/loom.test.Runtime/Unary",
		&loompb.ErrorResponse{Name: "fail"},
		&loompb.ErrorResponse{},
	)

	require.Equal(t, codes.PermissionDenied, status.Code(err))
	events := recorder.snapshot()
	require.Len(t, events, 2)
	require.Equal(t, loomtransport.EventKindRequestFailure, events[1].Kind)
	require.Equal(t, loomtransport.ReasonHandlerError, events[1].Reason)
}

func TestServerRuntimeStreamingBufconnCleanCompletion(t *testing.T) {
	recorder := &serverEventRecorder{}
	connection := startRuntimeBufconnServer(t, recorder)
	desc := &runtimeBufconnServiceDesc.Streams[0]
	stream, err := connection.NewStream(context.Background(), desc, "/loom.test.Runtime/Stream")
	require.NoError(t, err)
	require.NoError(t, stream.SendMsg(&loompb.ErrorResponse{Name: "tick"}))
	require.NoError(t, stream.CloseSend())

	for _, want := range []string{"tick-one", "tick-two"} {
		response := &loompb.ErrorResponse{}
		require.NoError(t, stream.RecvMsg(response))
		require.Equal(t, want, response.Name)
	}
	require.ErrorIs(t, stream.RecvMsg(&loompb.ErrorResponse{}), io.EOF)
	events := recorder.snapshot()
	require.Len(t, events, 4)
	require.Equal(t, loomtransport.EventKindRequestStart, events[0].Kind)
	require.Equal(t, loomtransport.EventKindStreamOpen, events[1].Kind)
	require.Equal(t, loomtransport.EventKindStreamClose, events[2].Kind)
	require.Equal(t, loomtransport.EventKindRequestFinish, events[3].Kind)
}

func TestObserveStreamErrorClassifiesOrigin(t *testing.T) {
	tests := []struct {
		name       string
		observe    func(context.Context, error) error
		wantEvents []loomtransport.EventKind
		wantReason loomtransport.Reason
	}{
		{
			name:       "response conversion",
			observe:    ObserveStreamEncodeError,
			wantEvents: []loomtransport.EventKind{loomtransport.EventKindRequestStart, loomtransport.EventKindRequestFailure},
			wantReason: loomtransport.ReasonResponseWriteFailed,
		},
		{
			name:       "stream write",
			observe:    ObserveStreamWriteError,
			wantEvents: []loomtransport.EventKind{loomtransport.EventKindRequestStart, loomtransport.EventKindStreamFailure, loomtransport.EventKindRequestFailure},
			wantReason: loomtransport.ReasonStreamWriteFailed,
		},
		{
			name:       "request decode",
			observe:    ObserveStreamDecodeError,
			wantEvents: []loomtransport.EventKind{loomtransport.EventKindRequestStart, loomtransport.EventKindRequestFailure},
			wantReason: loomtransport.ReasonRequestDecodeFailed,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := &serverEventRecorder{}
			ctx := loomtransport.WithObserver(context.Background(), recorder)
			ctx, observer := beginServerRequest(ctx, "Runtime", "Stream")
			wantErr := errors.New("stream operation failed")

			require.ErrorIs(t, test.observe(ctx, wantErr), wantErr)
			observer.End()

			events := recorder.snapshot()
			require.Len(t, events, len(test.wantEvents))
			for index, kind := range test.wantEvents {
				require.Equal(t, kind, events[index].Kind)
			}
			require.Equal(t, test.wantReason, events[len(events)-1].Reason)
		})
	}
}

func TestObserveStreamDecodeErrorTreatsEOFAsClean(t *testing.T) {
	recorder := &serverEventRecorder{}
	ctx := loomtransport.WithObserver(context.Background(), recorder)
	ctx, observer := beginServerRequest(ctx, "Runtime", "Stream")

	require.ErrorIs(t, ObserveStreamDecodeError(ctx, io.EOF), io.EOF)
	observer.End()

	events := recorder.snapshot()
	require.Len(t, events, 2)
	require.Equal(t, loomtransport.EventKindRequestFinish, events[1].Kind)
	require.Equal(t, loomtransport.ReasonOK, events[1].Reason)
}

func TestObserveStreamErrorsAreConcurrentSafe(t *testing.T) {
	for range 100 {
		recorder := &serverEventRecorder{}
		ctx := loomtransport.WithObserver(context.Background(), recorder)
		ctx, observer := beginServerRequest(ctx, "Runtime", "Stream")
		start := make(chan struct{})
		observedErrors := make(chan error, 2)
		var wait sync.WaitGroup
		wait.Add(2)
		go func() {
			defer wait.Done()
			<-start
			observedErrors <- ObserveStreamWriteError(ctx, errors.New("send failed"))
		}()
		go func() {
			defer wait.Done()
			<-start
			observedErrors <- ObserveStreamDecodeError(ctx, errors.New("receive failed"))
		}()

		close(start)
		wait.Wait()
		close(observedErrors)
		for observedErr := range observedErrors {
			require.Error(t, observedErr)
		}
		observer.End()

		events := recorder.snapshot()
		require.Len(t, events, 3)
		require.Equal(t, loomtransport.EventKindRequestStart, events[0].Kind)
		require.Equal(t, loomtransport.EventKindStreamFailure, events[1].Kind)
		require.Equal(t, loomtransport.EventKindRequestFailure, events[2].Kind)
		require.Contains(t, []loomtransport.Reason{
			loomtransport.ReasonRequestDecodeFailed,
			loomtransport.ReasonStreamWriteFailed,
		}, events[2].Reason)
	}
}

func startRuntimeBufconnServer(t *testing.T, observer loomtransport.Observer) *stdgrpc.ClientConn {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	server := stdgrpc.NewServer()
	server.RegisterService(&runtimeBufconnServiceDesc, &runtimeBufconnServer{observer: observer})
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- server.Serve(listener)
	}()
	connection, err := stdgrpc.NewClient(
		"passthrough:///bufconn",
		stdgrpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		stdgrpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, connection.Close())
		server.Stop()
		require.NoError(t, listener.Close())
		require.NoError(t, <-serveErrors)
	})
	return connection
}

var runtimeBufconnServiceDesc = stdgrpc.ServiceDesc{
	ServiceName: "loom.test.Runtime",
	HandlerType: (*runtimeBufconnService)(nil),
	Methods: []stdgrpc.MethodDesc{
		{
			MethodName: "Unary",
			Handler: func(service any, ctx context.Context, decode func(any) error, _ stdgrpc.UnaryServerInterceptor) (any, error) {
				request := &loompb.ErrorResponse{}
				if err := decode(request); err != nil {
					return nil, err
				}
				return service.(*runtimeBufconnServer).unary(ctx, request)
			},
		},
	},
	Streams: []stdgrpc.StreamDesc{
		{
			StreamName:    "Stream",
			ServerStreams: true,
			Handler: func(service any, stream stdgrpc.ServerStream) error {
				request := &loompb.ErrorResponse{}
				if err := stream.RecvMsg(request); err != nil {
					return err
				}
				return service.(*runtimeBufconnServer).stream(request, stream)
			},
		},
	},
}
