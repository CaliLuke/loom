package codegen

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

const wrapperContractMatrixHarness = `package probe

import (
    "context"
    "net"
    "sync/atomic"
    "testing"
    "time"

    "github.com/stretchr/testify/require"
    "google.golang.org/protobuf/proto"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/status"
    "google.golang.org/grpc/test/bufconn"
    loomgrpc "github.com/CaliLuke/loom/grpc"
    loompb "github.com/CaliLuke/loom/grpc/pb"

    "example.com/wrapperprobe/gen/dual"
    "example.com/wrapperprobe/gen/grpc/dual/client"
    pb "example.com/wrapperprobe/gen/grpc/dual/pb"
    "example.com/wrapperprobe/gen/grpc/dual/server"
)

func TestOccurrenceContracts(t *testing.T) {
    t.Run("permissive and strict valid", func(t *testing.T) {
        value := &dual.Data{Free: %[1]s, Strict: %[2]s}
        request := client.NewProtoEchoRequest(value)
        data, err := proto.Marshal(request)
        require.NoError(t, err)
        decoded := &pb.EchoRequest{}
        require.NoError(t, proto.Unmarshal(data, decoded))
        require.NoError(t, server.ValidateEchoRequest(decoded))
        payload, err := server.NewEchoPayload(decoded)
        require.NoError(t, err)
        require.Equal(t, value, payload)
        response := server.NewProtoEchoResponse(value)
        responseData, err := proto.MarshalOptions{Deterministic: true}.Marshal(response)
        require.NoError(t, err)
        requestData, err := proto.MarshalOptions{Deterministic: true}.Marshal(request)
        require.NoError(t, err)
        require.Equal(t, requestData, responseData, "request/result contents retain the same protobuf field contract")
        decodedResponse := &pb.EchoResponse{}
        require.NoError(t, proto.Unmarshal(responseData, decodedResponse))
        require.NoError(t, client.ValidateEchoResponse(decodedResponse))
        result, err := client.NewEchoResult(decodedResponse)
        require.NoError(t, err)
        require.Equal(t, value, result)
    })
    t.Run("strict violation", func(t *testing.T) {
        value := &dual.Data{Strict: %[3]s}
        request := client.NewProtoEchoRequest(value)
        data, err := proto.Marshal(request)
        require.NoError(t, err)
        decoded := &pb.EchoRequest{}
        require.NoError(t, proto.Unmarshal(data, decoded))
        require.ErrorContains(t, server.ValidateEchoRequest(decoded), "length")
        response := server.NewProtoEchoResponse(value)
        require.ErrorContains(t, client.ValidateEchoResponse(response), "length")
    })
}

func TestMountedContracts(t *testing.T) {
    var calls atomic.Int32
    received := make(chan *dual.Data, 1)
    endpoints := &dual.Endpoints{Echo: func(_ context.Context, value any) (any, error) {
        calls.Add(1)
        received <- value.(*dual.Data)
        return value, nil
    }}
    listener := bufconn.Listen(1 << 20)
    host := grpc.NewServer()
    pb.RegisterDualServer(host, server.New(endpoints, nil))
    done := make(chan error, 1)
    go func() {
        done <- host.Serve(listener)
    }()
    t.Cleanup(func() {
        host.Stop()
        require.NoError(t, <-done)
    })
    conn, err := grpc.NewClient("passthrough:///wrapper", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
        return listener.Dial()
    }), grpc.WithTransportCredentials(insecure.NewCredentials()))
    require.NoError(t, err)
    t.Cleanup(func() {
        require.NoError(t, conn.Close())
    })
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    remote := pb.NewDualClient(conn)
    response, err := remote.Echo(ctx, client.NewProtoEchoRequest(&dual.Data{Strict: %[3]s}))
    require.Equal(t, codes.InvalidArgument, status.Code(err))
    require.Nil(t, response)
    problem, ok := loomgrpc.DecodeError(err).(*loompb.ErrorResponse)
    require.True(t, ok)
    require.Equal(t, "invalid_length", problem.Name)
    require.Equal(t, "validation error", status.Convert(err).Message())
    require.Zero(t, calls.Load())
    value := &dual.Data{Free: %[1]s, Strict: %[2]s}
    response, err = remote.Echo(ctx, client.NewProtoEchoRequest(value))
    require.NoError(t, err)
    require.Equal(t, int32(1), calls.Load())
    select {
    case actual := <-received:
        require.Equal(t, value, actual)
    case <-ctx.Done():
        require.NoError(t, ctx.Err())
    }
    require.NoError(t, client.ValidateEchoResponse(response))
    result, err := client.NewEchoResult(response)
    require.NoError(t, err)
    require.Equal(t, value, result)
}
`

func TestWrapperContractTraversalOrder(t *testing.T) {
	for _, shape := range []string{"array-array", "array-map", "map-array", "map-map"} {
		for _, reverse := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/strict-first-%t", shape, reverse), func(t *testing.T) {
				root := RunGRPCDSL(t, wrapperContractMatrixDSL(shape, reverse, false))
				free, strict := wrapperContractValues(shape, false)
				runWrapperContractModule(t, root, fmt.Sprintf(wrapperContractMatrixHarness, free, strict, strings.ReplaceAll(strings.ReplaceAll(strict, "1, 2", "1"), `, "b": 2`, "")))
			})
		}
	}
}

func TestWrapperContractNamedBoundaryControl(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(fmt.Sprintf("strict-first-%t", reverse), func(t *testing.T) {
			root := RunGRPCDSL(t, wrapperContractMatrixDSL("array-array", reverse, true))
			free, strict := wrapperContractValues("array-array", true)
			runWrapperContractModule(t, root, fmt.Sprintf(wrapperContractMatrixHarness, free, strict, strings.ReplaceAll(strings.ReplaceAll(strict, "1, 2", "1"), `, "b": 2`, "")))
		})
	}
}

func wrapperContractMatrixDSL(shape string, reverse, named bool) func() {
	return func() {
		free, strict := wrapperContractFields(shape, named)
		data := Type("Data", func() {
			if reverse {
				Field(2, "strict", strict)
				Field(1, "free", free)
			} else {
				Field(1, "free", free)
				Field(2, "strict", strict)
			}
		})
		Service("dual", func() {
			Method("echo", func() {
				Payload(data)
				Result(data)
				GRPC(func() {
				})
			})
		})
	}
}

func wrapperContractFields(shape string, named bool) (expr.DataType, expr.DataType) {
	var inner expr.DataType = ArrayOf(Int)
	if shape == "array-map" || shape == "map-map" {
		inner = MapOf(String, Int)
	}
	minimum := func() {
		MinLength(2)
	}
	if named {
		free := Type("FreeItems", inner)
		strict := Type("StrictItems", inner, minimum)
		return ArrayOf(free), ArrayOf(strict)
	}
	if shape == "map-array" || shape == "map-map" {
		return MapOf(String, inner), MapOf(String, inner, func() {
			Elem(minimum)
		})
	}
	return ArrayOf(inner), ArrayOf(inner, minimum)
}

func wrapperContractValues(shape string, named bool) (string, string) {
	if named {
		return "[]dual.FreeItems{{1}}", "[]dual.StrictItems{{1, 2}}"
	}
	switch shape {
	case "array-array":
		return "[][]int{{1}}", "[][]int{{1, 2}}"
	case "array-map":
		return `[]map[string]int{{"a": 1}}`, `[]map[string]int{{"a": 1, "b": 2}}`
	case "map-array":
		return `map[string][]int{"x": {1}}`, `map[string][]int{"x": {1, 2}}`
	default:
		return `map[string]map[string]int{"x": {"a": 1}}`, `map[string]map[string]int{"x": {"a": 1, "b": 2}}`
	}
}
