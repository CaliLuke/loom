package codegen

const binaryMetadataAggregateHarness = `package binaryaggregate_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	svc "example.com/binaryaggregate/gen/binary"
	client "example.com/binaryaggregate/gen/grpc/binary/client"
	pb "example.com/binaryaggregate/gen/grpc/binary/pb"
	server "example.com/binaryaggregate/gen/grpc/binary/server"
)

type (
	implementation struct {
	}
	peer struct {
		pb.UnimplementedBinaryServer
		values metadata.MD
	}
)

func TestBinaryAggregateValues(t *testing.T) {
	for _, supplied := range []string{"absent", "empty", "value"} {
		t.Run(supplied, func(t *testing.T) {
			values := metadata.Pairs("x-raw-bin", string([]byte{0, 255, 1}), "x-items-bin", "a\n", "x-items-bin", "", "x-text", "hello")
			want := "default"
			if supplied != "absent" {
				want = ""
				if supplied == "value" {
					want = string([]byte{255, 0})
				}
				values.Set("x-optional-bin", want)
			}
			for _, placement := range []string{"request", "header", "trailer"} {
				t.Run(placement, func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					defer cancel()
					if placement == "request" {
						conn := connection(t, server.New(svc.NewEndpoints(implementation{}), nil))
						result, err := pb.NewBinaryClient(conn).Request(metadata.NewOutgoingContext(ctx, values), &pb.RequestRequest{Body: new("body")})
						require.NoError(t, err)
						require.Equal(t, []byte{0, 255, 1}, result.GetRaw())
						require.Equal(t, want, string(result.GetOptional()))
						require.NotNil(t, result.GetItems())
						require.Len(t, result.GetItems().GetField(), 2)
						require.Equal(t, []byte("a\n"), result.GetItems().GetField()[0])
						require.Empty(t, result.GetItems().GetField()[1])
						require.Equal(t, "hello", result.GetText())
						return
					}
					conn := connection(t, peer{values: values})
					transport := client.NewClient(conn)
					remote := svc.NewClient(transport.Request(), transport.Header(), transport.Trailer())
					call := remote.Header
					if placement == "trailer" {
						call = remote.Trailer
					}
					result, err := call(ctx, &svc.Value{Body: "body", Raw: svc.Raw{1}})
					require.NoError(t, err)
					require.Equal(t, svc.Raw{0, 255, 1}, result.Raw)
					require.Equal(t, want, string(result.Optional))
					require.Equal(t, svc.RawList{svc.Raw("a\n"), svc.Raw{}}, result.Items)
					require.NotNil(t, result.Text)
					require.Equal(t, "hello", *result.Text)
				})
			}
		})
	}
}

func (implementation) Request(ctx context.Context, value *svc.Value) (*svc.Value, error) {
	return value, nil
}

func (implementation) Header(ctx context.Context, value *svc.Value) (*svc.Value, error) {
	return value, nil
}

func (implementation) Trailer(ctx context.Context, value *svc.Value) (*svc.Value, error) {
	return value, nil
}

func (p peer) Header(ctx context.Context, request *pb.HeaderRequest) (*pb.HeaderResponse, error) {
	if err := grpc.SendHeader(ctx, p.values); err != nil {
		return nil, err
	}
	return &pb.HeaderResponse{Body: new("body")}, nil
}

func (p peer) Trailer(ctx context.Context, request *pb.TrailerRequest) (*pb.TrailerResponse, error) {
	if err := grpc.SetTrailer(ctx, p.values); err != nil {
		return nil, err
	}
	return &pb.TrailerResponse{Body: new("body")}, nil
}

func connection(t *testing.T, implementation pb.BinaryServer) *grpc.ClientConn {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	host := grpc.NewServer()
	pb.RegisterBinaryServer(host, implementation)
	done := make(chan error, 1)
	go func() {
		done <- host.Serve(listener)
	}()
	t.Cleanup(func() {
		host.Stop()
		require.NoError(t, <-done)
	})
	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})
	return conn
}
`
