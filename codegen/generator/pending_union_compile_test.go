package generator

import (
	"testing"

	"github.com/CaliLuke/loom/codegen/testdata"
)

func TestPendingUnionTransports(t *testing.T) {
	runDesignHarness(t, "example.com/pending", testdata.PendingUnionDSL, pendingUnionHarness)
}

const pendingUnionHarness = `package pendingtest

import (
	"context"
	"encoding/json/v2"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	grpcclient "example.com/pending/gen/grpc/pending/client"
	pb "example.com/pending/gen/grpc/pending/pb"
	grpcserver "example.com/pending/gen/grpc/pending/server"
	httpclient "example.com/pending/gen/http/pending/client"
	httpserver "example.com/pending/gen/http/pending/server"
	rpcclient "example.com/pending/gen/jsonrpc/pendingrpc/client"
	rpcserver "example.com/pending/gen/jsonrpc/pendingrpc/server"
	svc "example.com/pending/gen/pending"
	rpc "example.com/pending/gen/pendingrpc"
)

type echo struct{}
type rpcEcho struct{}

var forwardValues = []string{
	"{\"type\":\"String\",\"value\":\"text\"}",
	"{\"type\":\"Inner\",\"value\":{\"type\":\"Int\",\"value\":42}}",
	"{\"type\":\"Inner\",\"value\":{\"type\":\"Later\",\"value\":{\"value\":\"leaf\"}}}",
}
var recursiveValues = []string{
	"{}",
	"{\"next\":{\"type\":\"Later\",\"value\":{\"value\":\"leaf\"}}}",
	"{\"next\":{\"type\":\"Node\",\"value\":{\"next\":{\"type\":\"Later\",\"value\":{\"value\":\"deep\"}}}}}",
}

func (echo) Forward(_ context.Context, value *svc.Outer) (*svc.Outer, error) {
	return value, nil
}
func (echo) Recursive(_ context.Context, value *svc.Node) (*svc.Node, error) {
	return value, nil
}
func (rpcEcho) Forward(_ context.Context, value *rpc.Outer) (*rpc.Outer, error) {
	return value, nil
}
func (rpcEcho) Recursive(_ context.Context, value *rpc.Node) (*rpc.Node, error) {
	return value, nil
}

func TestHTTP(t *testing.T) {
	mux := loomhttp.NewMuxer()
	httpserver.Mount(mux, httpserver.New(svc.NewEndpoints(echo{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	server := httptest.NewServer(mux)
	defer server.Close()
	transport := httpclient.NewClient("http", strings.TrimPrefix(server.URL, "http://"), server.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	client := svc.NewClient(transport.Forward(), transport.Recursive())
	checkExchange(t, client.Forward, forwardValues)
	checkExchange(t, client.Recursive, recursiveValues)
}

func TestJSONRPC(t *testing.T) {
	mux := loomhttp.NewMuxer()
	rpcserver.Mount(mux, rpcserver.New(rpc.NewEndpoints(rpcEcho{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
	server := httptest.NewServer(mux)
	defer server.Close()
	transport := rpcclient.NewClient("http", strings.TrimPrefix(server.URL, "http://"), server.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	client := rpc.NewClient(transport.Forward(), transport.Recursive())
	checkExchange(t, client.Forward, forwardValues)
	checkExchange(t, client.Recursive, recursiveValues)
}

func checkExchange[T any](t *testing.T, call func(context.Context, *T) (*T, error), cases []string) {
	t.Helper()
	for _, data := range cases {
		t.Run(data, func(t *testing.T) {
			var want T
			require.NoError(t, json.Unmarshal([]byte(data), &want))
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			got, err := call(ctx, &want)
			require.NoError(t, err)
			require.Equal(t, &want, got)
		})
	}
}

func TestGRPC(t *testing.T) {
	checkProto(t, forwardValues, grpcclient.NewProtoOuter, grpcserver.NewForwardPayload, func() *pb.Outer {
		return &pb.Outer{}
	})
	checkProto(t, forwardValues, grpcserver.NewProtoOuter, grpcclient.NewForwardResult, func() *pb.Outer {
		return &pb.Outer{}
	})
	checkProto(t, recursiveValues, grpcclient.NewProtoRecursiveRequest, grpcserver.NewRecursivePayload, func() *pb.RecursiveRequest {
		return &pb.RecursiveRequest{}
	})
	checkProto(t, recursiveValues, grpcserver.NewProtoRecursiveResponse, grpcclient.NewRecursiveResult, func() *pb.RecursiveResponse {
		return &pb.RecursiveResponse{}
	})
}

func checkProto[T any, P proto.Message](t *testing.T, cases []string, encode func(*T) P, decode func(P) *T, empty func() P) {
	t.Helper()
	for _, data := range cases {
		t.Run(data, func(t *testing.T) {
			var want T
			require.NoError(t, json.Unmarshal([]byte(data), &want))
			wire, err := proto.Marshal(encode(&want))
			require.NoError(t, err)
			message := empty()
			require.NoError(t, proto.Unmarshal(wire, message))
			require.Equal(t, &want, decode(message))
		})
	}
}
`
