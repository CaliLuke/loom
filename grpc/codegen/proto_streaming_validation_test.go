package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestProtoStreamingRootValidation(t *testing.T) {
	root := RunGRPCDSL(t, testdata.StreamingValidationDSL)
	var rendered string
	for _, file := range ServerTypeFiles("", CreateGRPCServices(root)) {
		rendered += sectionCode(t, file.AllSections()[1:]...)
	}
	for _, name := range []string{"UploadStreamingRequest", "EnvelopeStreamItem", "DirectStreamingRequest", "MappingStreamingRequest", "TextStreamingRequest", "Tags"} {
		require.Contains(t, rendered, "func Validate"+name+"(")
	}
	require.Contains(t, rendered, "len(stream.Field) < 2")
	require.Contains(t, rendered, "len(stream.Field) > 3")
	require.Contains(t, rendered, "utf8.RuneCountInString(e) < 1")
	require.Contains(t, rendered, "utf8.RuneCountInString(*stream.Field) < 2")
}

func TestGeneratedStreamingRootValidation(t *testing.T) {
	runGeneratedRoundTrip(t, "example.com/streamvalidation", testdata.StreamingValidationDSL, streamingValidationHarness)
}

const streamingValidationHarness = `package roundtrip

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
 "google.golang.org/protobuf/proto"

	pb "%[1]s/gen/grpc/validated/pb"
	"%[1]s/gen/grpc/validated/server"
	validated "%[1]s/gen/validated"
)

type service struct {
	validated.Service
}

func (service) Upload(_ context.Context, stream validated.UploadServerStream) error {
	value, err := stream.Recv()
	if err != nil {
		return err
	}
	return stream.SendAndClose(len(value))
}

func (service) Envelope(_ context.Context, _ *validated.EnvelopePayload, stream validated.EnvelopeServerStream) error {
	value, err := stream.Recv()
	if err != nil {
		return err
	}
	return stream.SendAndClose(len(value))
}

func TestCollectionValidation(t *testing.T) {
	for _, test := range []struct {
		name  string
		value []string
		valid bool
	}{
		{"nil", nil, false},
		{"empty", []string{}, false},
		{"short", []string{"one"}, false},
		{"minimum", []string{"one", "two"}, true},
		{"maximum", []string{"one", "two", "three"}, true},
		{"long", []string{"one", "two", "three", "four"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, err := range []error{
				server.ValidateUploadStreamingRequest(&pb.UploadStreamingRequest{Field: test.value}),
				server.ValidateEnvelopeStreamItem(&pb.EnvelopeStreamItem{Field: test.value}),
				server.ValidateDirectStreamingRequest(&pb.DirectStreamingRequest{Field: test.value}),
				server.ValidateTags(&pb.Tags{Field: test.value}),
			} {
				require.Equal(t, test.valid, err == nil)
			}
		})
	}
	for _, value := range [][]string{{"", "two"}, {"one", ""}} {
		require.Error(t, server.ValidateUploadStreamingRequest(&pb.UploadStreamingRequest{Field: value}))
		require.Error(t, server.ValidateEnvelopeStreamItem(&pb.EnvelopeStreamItem{Field: value}))
		require.Error(t, server.ValidateTags(&pb.Tags{Field: value}))
	}
	for _, size := range []int{0, 1, 2, 3, 4} {
		value := make(map[string]string)
		for i := range size {
			value[string(rune('a'+i))] = "value"
		}
		err := server.ValidateMappingStreamingRequest(&pb.MappingStreamingRequest{Field: value})
		require.Equal(t, size >= 2 && size <= 3, err == nil)
	}
	for _, value := range []string{"", "a", "ab", "abc", "abcd", "é界"} {
		err := server.ValidateTextStreamingRequest(&pb.TextStreamingRequest{Field: proto.String(value)})
		size := len([]rune(value))
		require.Equal(t, size >= 2 && size <= 3, err == nil)
	}
}

func TestStreamReceiveValidation(t *testing.T) {
	listener := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	pb.RegisterValidatedServer(srv, server.New(validated.NewEndpoints(service{}), nil))
	go func() {
		if err := srv.Serve(listener); err != nil {
			t.Error(err)
		}
	}()
	t.Cleanup(srv.Stop)
	conn, err := grpc.NewClient("passthrough:///bufconn", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})
	client := pb.NewValidatedClient(conn)
	for _, value := range [][]string{nil, {}, {"one"}, {"one", "two"}, {"one", "two", "three"}, {"one", "two", "three", "four"}, {"", "two"}} {
		valid := len(value) >= 2 && len(value) <= 3
		for _, item := range value {
			valid = valid && item != ""
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		stream, err := client.Upload(ctx)
		require.NoError(t, err)
		require.NoError(t, stream.Send(&pb.UploadStreamingRequest{Field: value}))
		response, err := stream.CloseAndRecv()
		if valid {
			require.NoError(t, err)
			require.EqualValues(t, len(value), response.GetField())
		} else {
			require.Equal(t, codes.InvalidArgument, status.Code(err))
		}
		cancel()
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		envelope, err := client.Envelope(ctx)
		require.NoError(t, err)
		require.NoError(t, envelope.Send(&pb.EnvelopeStreamingRequest{Body: &pb.EnvelopeStreamingRequest_InitialPayload{InitialPayload: &pb.EnvelopeRequest{Label: proto.String("batch")}}}))
		require.NoError(t, envelope.Send(&pb.EnvelopeStreamingRequest{Body: &pb.EnvelopeStreamingRequest_StreamItem{StreamItem: &pb.EnvelopeStreamItem{Field: value}}}))
		result, err := envelope.CloseAndRecv()
		if valid {
			require.NoError(t, err)
			require.EqualValues(t, len(value), result.GetField())
		} else {
			require.Equal(t, codes.InvalidArgument, status.Code(err))
		}
		cancel()
	}
}
`
