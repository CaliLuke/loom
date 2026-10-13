package codegen

const metadataDefaultsWireHarness = `package rt

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	loomgrpc "github.com/CaliLuke/loom/grpc"
	loompb "github.com/CaliLuke/loom/grpc/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	client "example.com/mdprobe/gen/grpc/mdsvc/client"
	pb "example.com/mdprobe/gen/grpc/mdsvc/pb"
	server "example.com/mdprobe/gen/grpc/mdsvc/server"
	mdsvc "example.com/mdprobe/gen/mdsvc"
)

func connectMetadataServer(t *testing.T, implementation pb.MdsvcServer) *grpc.ClientConn {
	t.Helper()
	listener := bufconn.Listen(1 << 20)
	host := grpc.NewServer()
	pb.RegisterMdsvcServer(host, implementation)
	done := make(chan error, 1)
	go func() {
		done <- host.Serve(listener)
	}()
	t.Cleanup(func() {
		host.Stop()
		require.NoError(t, <-done)
	})
	conn, err := grpc.NewClient("passthrough:///metadata", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})
	return conn
}

func TestMetadataMounted(t *testing.T) {
	for _, tc := range []struct {
		name string
		metadata metadata.MD
		limit int
		lang string
		invalid bool
	}{
		{"absence", metadata.Pairs("must", "present"), 3, "en", false},
		{"explicit zero empty", metadata.Pairs("must", "present", "limit", "0", "lang", "", "flag", "false", "data", "", "tags", "", "numbers", "0"), 0, "", false},
		{"ordinary", metadata.Pairs("must", "present", "limit", "9", "lang", "it", "data", "printable bytes"), 9, "it", false},
		{"malformed", metadata.Pairs("must", "present", "limit", "bad"), 0, "", true},
		{"required missing", metadata.MD{}, 0, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			received := make(chan *mdsvc.V, 1)
			endpoints := &mdsvc.Endpoints{Echo: func(_ context.Context, value any) (any, error) {
				calls.Add(1)
				received <- value.(*mdsvc.V)
				return value, nil
			}}
			conn := connectMetadataServer(t, server.New(endpoints, nil))
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			ctx = metadata.NewOutgoingContext(ctx, tc.metadata)
			var headers, trailers metadata.MD
			body := "body"
			response, err := pb.NewMdsvcClient(conn).Echo(ctx, &pb.EchoRequest{Body: &body}, grpc.Header(&headers), grpc.Trailer(&trailers))
			if tc.invalid {
				require.Equal(t, codes.InvalidArgument, status.Code(err))
				require.Nil(t, response)
				detail, ok := loomgrpc.DecodeError(err).(*loompb.ErrorResponse)
				require.True(t, ok)
				if tc.name == "malformed" {
					require.Equal(t, "invalid_field_type", detail.Name)
					require.Equal(t, "validation error", status.Convert(err).Message())
				} else {
					require.Equal(t, "missing_field", detail.Name)
					require.Contains(t, status.Convert(err).Message(), "must")
				}
				require.Zero(t, calls.Load())
				return
			}
			require.NoError(t, err)
			require.Equal(t, int32(1), calls.Load())
			var payload *mdsvc.V
			select {
			case payload = <-received:
			case <-ctx.Done():
				require.NoError(t, ctx.Err())
			}
			require.Equal(t, tc.limit, payload.Limit)
			require.Equal(t, tc.lang, payload.Lang)
			decoded, err := client.DecodeEchoResponse(ctx, response, headers, trailers)
			require.NoError(t, err)
			require.Equal(t, payload, decoded)
		})
	}
}

type metadataPeer struct {
	pb.UnimplementedMdsvcServer
	values metadata.MD
}

func (s *metadataPeer) Echo(ctx context.Context, request *pb.EchoRequest) (*pb.EchoResponse, error) {
	if err := grpc.SetHeader(ctx, s.values); err != nil {
		return nil, err
	}
	if err := grpc.SetTrailer(ctx, s.values); err != nil {
		return nil, err
	}
	return &pb.EchoResponse{Body: request.Body}, nil
}

func TestMetadataPeerDefaults(t *testing.T) {
	for _, tc := range []struct {
		name string
		values metadata.MD
		limit int
		lang string
	}{
		{"absent", metadata.Pairs("must", "present"), 3, "en"},
		{"supplied zero empty", metadata.Pairs("must", "present", "limit", "0", "lang", ""), 0, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conn := connectMetadataServer(t, &metadataPeer{values: tc.values})
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			body := "body"
			var headers, trailers metadata.MD
			response, err := pb.NewMdsvcClient(conn).Echo(ctx, &pb.EchoRequest{Body: &body}, grpc.Header(&headers), grpc.Trailer(&trailers))
			require.NoError(t, err)
			decoded, err := client.DecodeEchoResponse(ctx, response, headers, trailers)
			require.NoError(t, err)
			value := decoded.(*mdsvc.V)
			require.Equal(t, tc.limit, value.Limit)
			require.Equal(t, tc.lang, value.Lang)
			require.True(t, value.Flag)
			require.Equal(t, []byte("fallback"), value.Data)
			require.Equal(t, mdsvc.Label("fallback"), value.Label)
			require.Equal(t, []string{"fallback"}, value.Tags)
			require.Equal(t, []int{3}, value.Numbers)
		})
	}
}
`
