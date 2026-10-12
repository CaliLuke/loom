package generator

import "testing"

func TestGRPCJoinedServerErrors(t *testing.T) {
	runDesignHarness(t, "example.com/clientboundary", grpcClientErrorBoundaryDSL, grpcJoinedServerErrorHarness)
}

const grpcJoinedServerErrorHarness = `package clientboundary

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	svc "example.com/clientboundary/gen/boundary"
	pb "example.com/clientboundary/gen/grpc/boundary/pb"
	"example.com/clientboundary/gen/grpc/boundary/server"
	lg "github.com/CaliLuke/loom/grpc"
	loompb "github.com/CaliLuke/loom/grpc/pb"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
	std "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

func TestGeneratedServerAggregation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		code   codes.Code
		custom bool
	}{
		{"plain secret", errors.New("private database password"), codes.Unknown, false},
		{"wrapped secret", fmt.Errorf("private query: %w", errors.New("private database password")), codes.Unknown, false},
		{"safe public", loom.WithErrorRemedy(loom.Fault("private database password"), &loom.ErrorRemedy{SafeMessage: "safe public detail"}), codes.Internal, false},
		{"canceled", context.Canceled, codes.Canceled, false},
		{"deadline", fmt.Errorf("operation: %w", context.DeadlineExceeded), codes.DeadlineExceeded, false},
		{"context plus cleanup", errors.Join(context.Canceled, errors.New("cleanup")), codes.Unknown, false},
		{"single custom", &svc.Problem{Reason: "bad input"}, codes.InvalidArgument, true},
		{"wrapped custom", fmt.Errorf("context: %w", &svc.Problem{Reason: "bad input"}), codes.InvalidArgument, true},
		{"same custom", errors.Join(&svc.Problem{Reason: "one"}, &svc.Problem{Reason: "two"}), codes.InvalidArgument, false},
		{"mixed", errors.Join(&svc.Problem{Reason: "bad input"}, errors.New("cleanup failed")), codes.Unknown, false},
		{"mixed reversed", errors.Join(errors.New("cleanup failed"), &svc.Problem{Reason: "bad input"}), codes.Unknown, false},
		{"same status", errors.Join(status.Error(codes.Unavailable, "one"), status.Error(codes.Unavailable, "two")), codes.Unavailable, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fail := func(context.Context, any) (any, error) {
				return nil, tc.err
			}
			endpoints := &svc.Endpoints{Plain: loom.Endpoint(fail), Declared: loom.Endpoint(fail), Watch: loom.Endpoint(fail)}
			listener := bufconn.Listen(1024 * 1024)
			srv := std.NewServer()
			pb.RegisterBoundaryServer(srv, server.New(endpoints, nil, nil))
			done := make(chan error, 1)
			go func() {
				done <- srv.Serve(listener)
			}()
			t.Cleanup(func() {
				srv.Stop()
				require.NoError(t, <-done)
			})
			conn, err := std.NewClient("passthrough:///bufconn", std.WithTransportCredentials(insecure.NewCredentials()),
				std.WithContextDialer(func(context.Context, string) (net.Conn, error) {
					return listener.Dial()
				}))
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, conn.Close())
			})
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			client := pb.NewBoundaryClient(conn)
			_, unaryErr := client.Declared(ctx, &pb.DeclaredRequest{Name: proto.String("ok")})
			stream, err := client.Watch(ctx, &pb.WatchRequest{Name: proto.String("ok")})
			require.NoError(t, err)
			_, streamErr := stream.Recv()
			for _, got := range []error{unaryErr, streamErr} {
				require.Equal(t, tc.code, status.Code(got))
				wantMessage := loom.ErrorSafeMessage(tc.err)
				if tc.name == "canceled" {
					wantMessage = context.Canceled.Error()
				} else if tc.name == "deadline" {
					wantMessage = context.DeadlineExceeded.Error()
				}
				require.Equal(t, wantMessage, status.Convert(got).Message())
				require.NotContains(t, status.Convert(got).String(), "private")
				detail := lg.DecodeError(got)
				if tc.name == "canceled" || tc.name == "deadline" {
					require.Nil(t, detail)
				} else if tc.custom {
					require.IsType(t, &pb.Problem{}, detail)
				} else {
					require.IsType(t, &loompb.ErrorResponse{}, detail)
					response := detail.(*loompb.ErrorResponse)
					require.Equal(t, loom.ErrorSafeMessage(tc.err), response.Msg)
					require.True(t, response.Fault)
					require.False(t, response.Temporary)
				}
			}
		})
	}
}
`
