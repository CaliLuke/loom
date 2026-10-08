package generator

import "testing"

func TestGRPCClientErrorCause(t *testing.T) {
	runDesignHarness(t, "example.com/clientboundary", grpcClientErrorBoundaryDSL, grpcClientErrorCauseHarness+grpcNativeContextHarness)
}

const grpcClientErrorCauseHarness = `package clientboundary

import (
	"context"
	"errors"
	"fmt"
	"testing"

	svc "example.com/clientboundary/gen/boundary"
	"example.com/clientboundary/gen/grpc/boundary/client"
	pb "example.com/clientboundary/gen/grpc/boundary/pb"
	loompb "github.com/CaliLuke/loom/grpc/pb"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
	std "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type failedStream struct {
	std.ClientStream
	err error
}

func (s failedStream) SendMsg(any) error {
	return nil
}

func (s failedStream) CloseSend() error {
	return nil
}

func (s failedStream) RecvMsg(any) error {
	return s.err
}

func TestGenericErrorsRetainTransportCause(t *testing.T) {
	detail := &loompb.ErrorResponse{
		Name: "remote", Id: "wire-id", Msg: "wire message", Timeout: true, Temporary: true, Fault: true,
		History: []*loompb.ErrorField{{Name: "invalid", Field: "name", Msg: "invalid name"}, nil, {Name: "missing", Msg: "missing value"}},
	}
	for _, code := range []codes.Code{codes.Canceled, codes.DeadlineExceeded, codes.Unavailable} {
		for _, wrapped := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/wrapped=%t", code, wrapped), func(t *testing.T) {
				st, err := status.New(code, "transport message").WithDetails(detail, &pb.Problem{Reason: proto.String("extra detail")})
				require.NoError(t, err)
				original := st.Err()
				cause := original
				if wrapped {
					cause = fmt.Errorf("outer: %w", cause)
				}
				conn, err := std.NewClient("passthrough:///unused", std.WithTransportCredentials(insecure.NewCredentials()),
					std.WithUnaryInterceptor(func(context.Context, string, any, any, *std.ClientConn, std.UnaryInvoker, ...std.CallOption) error {
						return cause
					}),
					std.WithStreamInterceptor(func(context.Context, *std.StreamDesc, *std.ClientConn, string, std.Streamer, ...std.CallOption) (std.ClientStream, error) {
						return failedStream{err: cause}, nil
					}))
				require.NoError(t, err)
				t.Cleanup(func() {
					require.NoError(t, conn.Close())
				})
				c := client.NewClient(conn)
				receive := func(ctx context.Context, payload any) (any, error) {
					result, err := c.Watch()(ctx, payload)
					if err != nil {
						return nil, err
					}
					return result.(svc.WatchClientStream).Recv()
				}
				for _, endpoint := range []loom.Endpoint{c.Plain(), c.Declared(), receive} {
					_, err := endpoint(context.Background(), &svc.Request{Name: "valid"})
					require.ErrorIs(t, err, cause)
					require.ErrorIs(t, err, original)
					require.Equal(t, code, status.Code(err))
					require.False(t, errors.Is(err, context.Canceled))
					require.False(t, errors.Is(err, context.DeadlineExceeded))
					var transport interface {
						GRPCStatus() *status.Status
					}
					require.ErrorAs(t, err, &transport)
					require.True(t, proto.Equal(st.Proto(), transport.GRPCStatus().Proto()))
					var decoded *loom.ServiceError
					require.ErrorAs(t, err, &decoded)
					require.Equal(t, "remote", decoded.Name)
					require.Equal(t, "wire-id", decoded.ID)
					require.Equal(t, "wire message", decoded.Message)
					require.True(t, decoded.Timeout)
					require.True(t, decoded.Temporary)
					require.True(t, decoded.Fault)
					history := decoded.History()
					require.Len(t, history, 3)
					require.Equal(t, "invalid", history[0].Name)
					require.NotNil(t, history[0].Field)
					require.Equal(t, "name", *history[0].Field)
					require.Empty(t, history[1].Name) // A nil protobuf entry is received as an empty message.
					require.Equal(t, "missing", history[2].Name)
				}
			})
		}
	}
}
`

const grpcNativeContextHarness = `
func TestNativeContextStatus(t *testing.T) {
	for _, code := range []codes.Code{codes.Canceled, codes.DeadlineExceeded} {
		for _, ended := range []bool{false, true} {
			for _, details := range []bool{false, true} {
				for _, opening := range []bool{false, true} {
					st := status.New(code, "remote stop")
					if details {
						var err error
						st, err = st.WithDetails(&pb.Reply{State: proto.String("ok")})
						require.NoError(t, err)
					}
					original := st.Err()
					for _, cause := range []error{original, fmt.Errorf("outer: %w", original)} {
						conn, err := std.NewClient("passthrough:///unused", std.WithTransportCredentials(insecure.NewCredentials()),
							std.WithUnaryInterceptor(func(context.Context, string, any, any, *std.ClientConn, std.UnaryInvoker, ...std.CallOption) error {
								return cause
							}),
							std.WithStreamInterceptor(func(context.Context, *std.StreamDesc, *std.ClientConn, string, std.Streamer, ...std.CallOption) (std.ClientStream, error) {
								if opening {
									return nil, cause
								}
								return failedStream{err: cause}, nil
							}))
						require.NoError(t, err)
						c := client.NewClient(conn)
						receive := func(ctx context.Context, payload any) (any, error) {
							result, err := c.Watch()(ctx, payload)
							if err != nil {
								return nil, err
							}
							return result.(svc.WatchClientStream).Recv()
						}
						ctx, cancel := context.WithCancel(context.Background())
						if ended {
							cancel()
						}
						for _, endpoint := range []loom.Endpoint{c.Plain(), c.Declared(), receive} {
							_, err := endpoint(ctx, &svc.Request{Name: "valid"})
							require.ErrorIs(t, err, original)
							require.Equal(t, code, status.Code(err))
							require.False(t, errors.Is(err, context.Canceled))
							require.False(t, errors.Is(err, context.DeadlineExceeded))
							require.Equal(t, st.Details(), status.Convert(err).Details())
						}
						cancel()
						require.NoError(t, conn.Close())
					}
				}
			}
		}
	}
}
`
