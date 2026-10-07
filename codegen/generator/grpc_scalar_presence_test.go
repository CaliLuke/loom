package generator

import (
	"testing"

	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestGRPCScalarPresence(t *testing.T) {
	runDesignHarness(t, "example.com/presence", testdata.DefaultFieldsDSL, grpcScalarPresenceHarness)
}

const grpcScalarPresenceHarness = `package presence

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	svc "example.com/presence/gen/default_fields"
	"example.com/presence/gen/grpc/default_fields/client"
	pb "example.com/presence/gen/grpc/default_fields/pb"
	"example.com/presence/gen/grpc/default_fields/server"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
	std "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type omittedStreamResult struct {
	pb.UnimplementedDefaultFieldsServer
}

// Each required field has a present zero value; presence must survive bytes
// on the wire, independently of the service's value representation.
func zeroRequest(t *testing.T) *pb.ScalarsRequest {
	t.Helper()
	msg := &pb.ScalarsRequest{}
	reflect := msg.ProtoReflect()
	for n := protoreflect.FieldNumber(1); n <= 12; n++ {
		field := reflect.Descriptor().Fields().ByNumber(n)
		require.True(t, field.HasPresence(), field.Name())
		reflect.Set(field, reflect.NewField(field))
	}
	return msg
}

func wire(t *testing.T, msg proto.Message) proto.Message {
	t.Helper()
	data, err := proto.Marshal(msg)
	require.NoError(t, err)
	decoded := msg.ProtoReflect().New().Interface()
	require.NoError(t, proto.Unmarshal(data, decoded))
	return decoded
}

func TestRequiredScalarWirePresence(t *testing.T) {
	ctx := context.Background()
	for n := protoreflect.FieldNumber(1); n <= 12; n++ {
		request := zeroRequest(t)
		field := request.ProtoReflect().Descriptor().Fields().ByNumber(n)
		t.Run(string(field.Name()), func(t *testing.T) {
			present := wire(t, request).(*pb.ScalarsRequest)
			require.True(t, present.ProtoReflect().Has(field))
			payload, err := server.DecodeScalarsRequest(ctx, present, nil)
			require.NoError(t, err)
			value := payload.(*svc.PresenceScalars)
			require.Equal(t, "", value.RequiredDefault)
			require.Equal(t, "fallback", value.OptionalDefault)
			require.Nil(t, value.Optional)
			require.Equal(t, svc.PresenceText(""), value.Text)
			require.False(t, value.Flag)
			require.Zero(t, value.Integer)
			require.Zero(t, value.Float64)
			require.Empty(t, value.Bytes)

			request.ProtoReflect().Clear(field)
			absent := wire(t, request).(*pb.ScalarsRequest)
			absentBytes, err := proto.Marshal(absent)
			require.NoError(t, err)
			presentBytes, err := proto.Marshal(present)
			require.NoError(t, err)
			require.NotEqual(t, absentBytes, presentBytes)
			payload, err = server.DecodeScalarsRequest(ctx, absent, nil)
			require.Nil(t, payload)
			var missing *loom.ServiceError
			require.ErrorAs(t, err, &missing)
			require.Equal(t, loom.MissingField, missing.Name)
			require.Equal(t, strings.TrimSuffix(string(field.Name()), "_"), *missing.Field)

			// Client decoding must enforce the same contract on results.
			encoded, err := server.EncodeScalarsResponse(ctx, value, nil, nil)
			require.NoError(t, err)
			response := wire(t, encoded.(proto.Message)).(*pb.ScalarsResponse)
			result, err := client.DecodeScalarsResponse(ctx, response, nil, nil)
			require.NoError(t, err)
			require.Equal(t, value, result)
			response.ProtoReflect().Clear(response.ProtoReflect().Descriptor().Fields().ByNumber(n))
			result, err = client.DecodeScalarsResponse(ctx, wire(t, response), nil, nil)
			require.Nil(t, result)
			require.ErrorAs(t, err, &missing)
			require.Equal(t, loom.MissingField, missing.Name)
		})
	}
}

func TestDefaultsAndConstraints(t *testing.T) {
	request := zeroRequest(t)
	request.OptionalDefault = proto.String("")
	payload, err := server.DecodeScalarsRequest(context.Background(), wire(t, request), nil)
	require.NoError(t, err)
	require.Equal(t, "", payload.(*svc.PresenceScalars).OptionalDefault)
	request.Text = proto.String("long")
	_, err = server.DecodeScalarsRequest(context.Background(), wire(t, request), nil)
	var invalid *loom.ServiceError
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, loom.InvalidLength, invalid.Name)
	request.Text = proto.String("")
	request.ProtoReflect().Set(request.ProtoReflect().Descriptor().Fields().ByNumber(11), protoreflect.ValueOfBytes([]byte("long")))
	_, err = server.DecodeScalarsRequest(context.Background(), wire(t, request), nil)
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, loom.InvalidLength, invalid.Name)
}

func TestGeneratedHandlers(t *testing.T) {
	calls := 0
	endpoints := &svc.Endpoints{
		Scalars: func(_ context.Context, v any) (any, error) {
			calls++
			return v, nil
		},
		Stream: func(_ context.Context, v any) (any, error) {
			stream := v.(*svc.StreamEndpointInput).Stream
			value, err := stream.Recv()
			if err != nil {
				return nil, err
			}
			return nil, stream.Send(value)
		},
	}
	generated := server.New(endpoints, nil, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, present := range []bool{false, true} {
		req := zeroRequest(t)
		if !present {
			req.Flag = nil
		}
		before := calls
		result, err := generated.Scalars(ctx, req)
		if present {
			require.NoError(t, err)
			require.NotNil(t, result.Flag)
			require.False(t, *result.Flag)
			require.Equal(t, before+1, calls)
		} else {
			require.Equal(t, codes.InvalidArgument, status.Code(err))
			require.Nil(t, result)
			require.Equal(t, before, calls)
		}
	}
	conn := serve(t, generated)
	raw := pb.NewDefaultFieldsClient(conn)
	for _, present := range []bool{false, true} {
		stream, err := raw.Stream(ctx)
		require.NoError(t, err)
		req := &pb.StreamStreamingRequest{}
		data, err := proto.Marshal(zeroRequest(t))
		require.NoError(t, err)
		require.NoError(t, proto.Unmarshal(data, req))
		if !present {
			req.Flag = nil
		}
		require.NoError(t, stream.Send(req))
		response, err := stream.Recv()
		if present {
			require.NoError(t, err)
			require.NotNil(t, response.Flag)
			require.False(t, *response.Flag)
		} else {
			require.Equal(t, codes.InvalidArgument, status.Code(err))
			require.Nil(t, response)
		}
		require.NoError(t, stream.CloseSend())
	}
}

func (omittedStreamResult) Stream(stream pb.DefaultFields_StreamServer) error {
	return stream.Send(&pb.StreamResponse{})
}

func TestGeneratedClientRejectsOmittedStreamResult(t *testing.T) {
	conn := serve(t, omittedStreamResult{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := client.NewClient(conn).Stream()(ctx, nil)
	require.NoError(t, err)
	stream := result.(svc.StreamClientStream)
	value, err := stream.Recv()
	require.Nil(t, value)
	var missing *loom.ServiceError
	require.ErrorAs(t, err, &missing)
	require.Equal(t, loom.MissingField, missing.Name)
	require.Len(t, missing.History(), 12)
	require.NoError(t, stream.Close())
}

func serve(t *testing.T, service pb.DefaultFieldsServer) *std.ClientConn {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	srv := std.NewServer()
	pb.RegisterDefaultFieldsServer(srv, service)
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
	return conn
}

func TestCLIValidatesBeforeConversion(t *testing.T) {
	for _, input := range []string{"", "{}"} {
		value, err := client.BuildScalarsPayload(input)
		require.Nil(t, value)
		var missing *loom.ServiceError
		require.ErrorAs(t, err, &missing)
		require.Equal(t, loom.MissingField, missing.Name)
	}
	value, err := client.BuildScalarsPayload(` + "`" + `{"text":"","flag":false,"integer":"0","integer32":0,"integer64":"0","unsigned":"0","unsigned32":0,"unsigned64":"0","float32":0,"float64":0,"bytes_":"","requiredDefault":"","optionalDefault":""}` + "`" + `)
	require.NoError(t, err)
	require.Equal(t, "", value.RequiredDefault)
	require.Equal(t, "", value.OptionalDefault)
	for _, input := range []string{"null", "{"} {
		value, err := client.BuildScalarsPayload(input)
		require.Nil(t, value)
		require.ErrorContains(t, err, "invalid JSON")
	}
}

func TestRootByteAliasPresence(t *testing.T) {
	ctx := context.Background()
	value, err := server.DecodeBlobRequest(ctx, wire(t, &pb.PresenceBlob{}), nil)
	require.Nil(t, value)
	var missing *loom.ServiceError
	require.ErrorAs(t, err, &missing)
	require.Equal(t, loom.MissingField, missing.Name)
	value, err = server.DecodeBlobRequest(ctx, wire(t, &pb.PresenceBlob{Field: []byte{}}), nil)
	require.NoError(t, err)
	require.Equal(t, svc.PresenceBlob{}, value)
	encoded, err := server.EncodeBlobResponse(ctx, value, nil, nil)
	require.NoError(t, err)
	decoded, err := client.DecodeBlobResponse(ctx, wire(t, encoded.(proto.Message)), nil, nil)
	require.NoError(t, err)
	require.Equal(t, value, decoded)
	fromCLI, err := client.BuildBlobPayload("{\"field\":\"\"}")
	require.NoError(t, err)
	require.Equal(t, value, fromCLI)
	_, err = server.DecodeBlobRequest(ctx, wire(t, &pb.PresenceBlob{Field: []byte("long")}), nil)
	var invalid *loom.ServiceError
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, loom.InvalidLength, invalid.Name)
}
`
