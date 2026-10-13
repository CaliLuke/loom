package codegen

const metadataElementHarness = `package metadataelements_test

import (
	"context"
	"encoding/json/v2"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	svc "example.com/metadataelements/gen/elements"
	client "example.com/metadataelements/gen/grpc/elements/client"
	pb "example.com/metadataelements/gen/grpc/elements/pb"
	server "example.com/metadataelements/gen/grpc/elements/server"
)

type (
	implementation struct {
	}
	metadataPeer struct {
		pb.UnimplementedElementsServer
		values metadata.MD
	}
)

func TestMetadataElementValues(t *testing.T) {
	conn := metadataConnection(t, server.New(svc.NewEndpoints(implementation{}), nil))
	transport := client.NewClient(conn)
	remote := svc.NewClient(transport.Request(), transport.Header(), transport.Trailer())
	for name, call := range map[string]func(context.Context, *svc.Value) (*svc.Value, error){
		"request": remote.Request, "header": remote.Header, "trailer": remote.Trailer,
	} {
		t.Run(name, func(t *testing.T) {
			for _, supplied := range []bool{false, true} {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				result, err := call(ctx, metadataPayload(t, supplied))
				cancel()
				require.NoError(t, err)
				require.Equal(t, "body", result.Body)
				raw := reflect.ValueOf(result.Raw)
				require.Len(t, result.Raw, 2)
				require.Equal(t, []byte{0, 255, 1}, raw.Index(0).Bytes())
				require.Len(t, raw.Index(1).Bytes(), 0)
				require.Equal(t, "hello", reflect.ValueOf(result.Text).Index(0).String())
				require.True(t, reflect.ValueOf(result.Flags).Index(0).Bool())
				require.Equal(t, uint64(2), reflect.ValueOf(result.Counts).Index(0).Uint())
				require.Equal(t, 1.5, reflect.ValueOf(result.Ratios).Index(0).Float())
				want := "base"
				if supplied {
					want = "other"
				}
				require.Len(t, result.OptionalText, 1)
				require.Equal(t, want, reflect.ValueOf(result.OptionalText).Index(0).String())
			}
		})
	}
}

func TestMetadataElementAdmission(t *testing.T) {
	cases := []struct {
		name       string
		field      string
		value      []string
		diagnostic string
	}{
		{"malformed", "x-counts", []string{"oops"}, "counts"},
		{"inherited-min", "x-text", []string{"aa"}, "text"},
		{"inherited-max", "x-text", []string{"too-long-value"}, "text"},
		{"required-absent", "x-raw-bin", nil, "x-raw-bin"},
		{"present-empty", "x-optional_text", []string{""}, "optional_text"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, placement := range []string{"request", "header", "trailer"} {
				t.Run(placement, func(t *testing.T) {
					values := metadata.Pairs("x-raw-bin", string([]byte{0, 255, 1}), "x-text", "hello", "x-flags", "true", "x-counts", "2", "x-ratios", "1.5")
					if tc.value == nil {
						values.Delete(tc.field)
					} else {
						values.Set(tc.field, tc.value...)
					}
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					defer cancel()
					var err error
					if placement == "request" {
						conn := metadataConnection(t, server.New(svc.NewEndpoints(implementation{}), nil))
						_, err = pb.NewElementsClient(conn).Request(metadata.NewOutgoingContext(ctx, values), &pb.RequestRequest{Body: new("body")})
						_, decodeErr := server.DecodeRequestRequest(ctx, &pb.RequestRequest{Body: new("body")}, values)
						require.ErrorContains(t, decodeErr, tc.diagnostic)
					} else {
						conn := metadataConnection(t, metadataPeer{values: values})
						transport := client.NewClient(conn)
						remote := svc.NewClient(transport.Request(), transport.Header(), transport.Trailer())
						if placement == "header" {
							_, err = remote.Header(ctx, metadataPayload(t, false))
						} else {
							_, err = remote.Trailer(ctx, metadataPayload(t, false))
						}
					}
					if placement == "request" {
						require.Equal(t, codes.InvalidArgument, status.Code(err))
					} else {
						require.ErrorContains(t, err, tc.diagnostic)
					}
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

func (peer metadataPeer) Header(ctx context.Context, request *pb.HeaderRequest) (*pb.HeaderResponse, error) {
	if err := grpc.SendHeader(ctx, peer.values); err != nil {
		return nil, err
	}
	return &pb.HeaderResponse{Body: new("body")}, nil
}

func (peer metadataPeer) Trailer(ctx context.Context, request *pb.TrailerRequest) (*pb.TrailerResponse, error) {
	if err := grpc.SetTrailer(ctx, peer.values); err != nil {
		return nil, err
	}
	return &pb.TrailerResponse{Body: new("body")}, nil
}

func metadataPayload(t *testing.T, supplied bool) *svc.Value {
	t.Helper()
	var result svc.Value
	require.NoError(t, json.Unmarshal([]byte("{\"body\":\"body\",\"raw\":[\"AP8B\",\"\"],\"text\":[\"hello\"],\"flags\":[true],\"counts\":[2],\"ratios\":[1.5]}"), &result))
	if supplied {
		require.NoError(t, json.Unmarshal([]byte("{\"optional_text\":[\"other\"]}"), &result))
	}
	return &result
}

func metadataConnection(t *testing.T, implementation pb.ElementsServer) *grpc.ClientConn {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	host := grpc.NewServer()
	pb.RegisterElementsServer(host, implementation)
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
