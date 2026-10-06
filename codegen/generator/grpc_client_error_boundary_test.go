package generator

import (
	"testing"

	d "github.com/CaliLuke/loom/dsl"
)

func TestGRPCClientErrorBoundary(t *testing.T) {
	runDesignHarness(t, "example.com/clientboundary", grpcClientErrorBoundaryDSL, grpcClientErrorBoundaryHarness)
}

func grpcClientErrorBoundaryDSL() {
	d.API("clientboundary", func() {
	})
	request := d.Type("Request", func() {
		d.Field(1, "name", d.String)
		d.Required("name")
	})
	reply := d.Type("Reply", func() {
		d.Field(1, "state", d.String, func() {
			d.Enum("ok")
		})
		d.Required("state")
		d.Meta("struct:name:proto", "Reply")
	})
	problem := d.Type("Problem", func() {
		d.Field(1, "reason", d.String)
		d.Required("reason")
		d.Meta("struct:name:proto", "Problem")
	})
	d.Service("boundary", func() {
		for _, name := range []string{"plain", "declared", "watch"} {
			d.Method(name, func() {
				d.Payload(request)
				if name == "watch" {
					d.StreamingResult(reply)
				} else {
					d.Result(reply)
				}
				if name != "plain" {
					d.Error("problem", problem)
				}
				d.GRPC(func() {
					if name != "plain" {
						d.Response("problem", d.CodeInvalidArgument)
					}
				})
			})
		}
	})
}

const grpcClientErrorBoundaryHarness = `package clientboundary

import (
	"context"
	"testing"

	svc "example.com/clientboundary/gen/boundary"
	"example.com/clientboundary/gen/grpc/boundary/client"
	pb "example.com/clientboundary/gen/grpc/boundary/pb"
	lg "github.com/CaliLuke/loom/grpc"
	loompb "github.com/CaliLuke/loom/grpc/pb"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
	std "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func TestLocalAndRemoteErrors(t *testing.T) {
	calls := 0
	state := "ok"
	var remoteErr error
	conn, err := std.NewClient("passthrough:///unused", std.WithTransportCredentials(insecure.NewCredentials()),
		std.WithUnaryInterceptor(func(_ context.Context, _ string, _, reply any, _ *std.ClientConn, _ std.UnaryInvoker, opts ...std.CallOption) error {
			calls++
			require.NotEmpty(t, opts)
			if remoteErr != nil {
				return remoteErr
			}
			reply.(*pb.Reply).State = state
			return nil
		}),
		std.WithStreamInterceptor(func(_ context.Context, _ *std.StreamDesc, _ *std.ClientConn, _ string, _ std.Streamer, _ ...std.CallOption) (std.ClientStream, error) {
			calls++
			return nil, remoteErr
		}))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})
	c := client.NewClient(conn, std.WaitForReady(true))
	t.Run("encoding", func(t *testing.T) {
		for _, endpoint := range []loom.Endpoint{c.Plain(), c.Declared(), c.Watch()} {
			before := calls
			_, err := endpoint(context.Background(), 123)
			var typed *lg.ClientError
			require.ErrorAs(t, err, &typed)
			require.Equal(t, "invalid_type", typed.Name)
			require.Equal(t, before, calls)
		}
	})
	t.Run("decoding", func(t *testing.T) {
		for _, endpoint := range []loom.Endpoint{c.Plain(), c.Declared()} {
			state = "invalid"
			_, err := endpoint(context.Background(), &svc.Request{Name: "valid"})
			var validation *loom.ServiceError
			require.ErrorAs(t, err, &validation)
			require.Equal(t, loom.InvalidEnumValue, validation.Name)
			require.NotNil(t, validation.Field)
			require.Equal(t, "message.state", *validation.Field)
			state = "ok"
			result, err := endpoint(context.Background(), &svc.Request{Name: "valid"})
			require.NoError(t, err)
			require.Equal(t, "ok", result.(*svc.Reply).State)
		}
	})
	detail := &loompb.ErrorResponse{Name: "remote", Id: "remote-id", Msg: "remote failure", Temporary: true}
	st, err := status.New(codes.Unavailable, "remote failure").WithDetails(detail)
	require.NoError(t, err)
	remoteErr = st.Err()
	for _, endpoint := range []loom.Endpoint{c.Plain(), c.Declared(), c.Watch()} {
		_, err := endpoint(context.Background(), &svc.Request{Name: "valid"})
		var serviceErr *loom.ServiceError
		require.ErrorAs(t, err, &serviceErr)
		require.Equal(t, "remote", serviceErr.Name)
		require.Equal(t, "remote-id", serviceErr.ID)
		require.True(t, serviceErr.Temporary)
	}
	st, err = status.New(codes.InvalidArgument, "declared failure").WithDetails(&pb.Problem{Reason: "declared"})
	require.NoError(t, err)
	remoteErr = st.Err()
	for _, endpoint := range []loom.Endpoint{c.Declared(), c.Watch()} {
		_, err := endpoint(context.Background(), &svc.Request{Name: "valid"})
		var problem *svc.Problem
		require.ErrorAs(t, err, &problem)
		require.Equal(t, "declared", problem.Reason)
	}
	remoteErr = status.Error(codes.Internal, "no detail")
	for _, endpoint := range []loom.Endpoint{c.Plain(), c.Declared(), c.Watch()} {
		_, err := endpoint(context.Background(), &svc.Request{Name: "valid"})
		var fault *loom.ServiceError
		require.ErrorAs(t, err, &fault)
		require.True(t, fault.Fault)
	}
}
`
