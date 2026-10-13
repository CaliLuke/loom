package codegen

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
)

func TestElementAdmissionTransport(t *testing.T) {
	runWrapperContractModule(t, RunGRPCDSL(t, elementAdmissionTransportDSL), elementAdmissionTransportHarness)
}

func elementAdmissionTransportDSL() {
	leaf := Type("Leaf", func() {
		Field(1, "value", String)
	})
	record := Type("Record", func() {
		Field(1, "items", ArrayOf(leaf))
	})
	failure := Type("Failure", func() {
		ErrorName(1, "name", String)
		Field(2, "items", ArrayOf(leaf))
		Required("name")
	})
	Service("admission", func() {
		Method("echo", func() {
			Payload(record)
			Result(record)
			GRPC(func() {})
		})
		Method("bad", func() {
			Result(record)
			GRPC(func() {})
		})
		Method("any", func() {
			Result(func() {
				Field(1, "value", Any)
			})
			GRPC(func() {})
		})
		Method("fail", func() {
			Error("failure", failure)
			GRPC(func() {
				Response("failure", CodeInternal)
			})
		})
		Method("stream", func() {
			StreamingPayload(record)
			StreamingResult(record)
			GRPC(func() {})
		})
		Method("badstream", func() {
			StreamingResult(record)
			GRPC(func() {})
		})
	})
}

const elementAdmissionTransportHarness = `package probe

import (
 "context"
 "errors"
 "io"
 "net"
 "sync/atomic"
 "testing"
 "time"

 "github.com/stretchr/testify/require"
 "google.golang.org/grpc"
 "google.golang.org/grpc/credentials/insecure"
 "google.golang.org/grpc/status"
 "google.golang.org/grpc/codes"
 "google.golang.org/grpc/test/bufconn"

 loom "github.com/CaliLuke/loom/pkg"
 svc "example.com/wrapperprobe/gen/admission"
 "example.com/wrapperprobe/gen/grpc/admission/client"
 pb "example.com/wrapperprobe/gen/grpc/admission/pb"
 "example.com/wrapperprobe/gen/grpc/admission/server"
)

type implementation struct {
 unary atomic.Int32
 stream atomic.Int32
 done chan struct{}
}

func invalid() *svc.Record {
 return &svc.Record{Items: []*svc.Leaf{nil}}
}

func valid() *svc.Record {
 return &svc.Record{Items: []*svc.Leaf{{Value: new("kept")}}}
}

func (s *implementation) Echo(_ context.Context, value *svc.Record) (*svc.Record, error) {
 s.unary.Add(1)
 return value, nil
}
func (s *implementation) Bad(context.Context) (*svc.Record, error) {
 return invalid(), nil
}
func (s *implementation) Any(context.Context) (*svc.AnyResult, error) {
 return &svc.AnyResult{Value: loom.JSONValue("{")}, nil
}
func (s *implementation) Fail(context.Context) error {
 return &svc.Failure{Name: "failure", Items: []*svc.Leaf{nil}}
}
func (s *implementation) Stream(_ context.Context, stream svc.StreamServerStream) error {
 defer close(s.done)
 for {
  value, err := stream.Recv()
  if errors.Is(err, io.EOF) {
   return nil
  }
  if err != nil {
   return err
  }
  s.stream.Add(1)
  if err := stream.Send(value); err != nil {
   return err
  }
 }
}
func (s *implementation) Badstream(_ context.Context, stream svc.BadstreamServerStream) error {
 if err := stream.Send(valid()); err != nil {
  return err
 }
 return stream.Send(invalid())
}

func TestAdmissionBeforeTransport(t *testing.T) {
 impl := &implementation{done: make(chan struct{})}
 listener := bufconn.Listen(1024*1024)
 transport := grpc.NewServer()
 pb.RegisterAdmissionServer(transport, server.New(svc.NewEndpoints(impl), nil, nil))
 served := make(chan error, 1)
 go func() {
  served <- transport.Serve(listener)
 }()
 t.Cleanup(func() {
  transport.Stop()
  require.NoError(t, <-served)
 })
 connection, err := grpc.NewClient("passthrough:///admission", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
  return listener.Dial()
 }), grpc.WithTransportCredentials(insecure.NewCredentials()))
 require.NoError(t, err)
 t.Cleanup(func() {
  require.NoError(t, connection.Close())
 })
 c := client.NewClient(connection)
 ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
 defer cancel()
 result, err := c.Echo()(ctx, invalid())
 require.Error(t, err)
 require.Nil(t, result)
 require.Zero(t, impl.unary.Load())
 result, err = c.Echo()(ctx, valid())
 require.NoError(t, err)
 require.Equal(t, valid(), result)
 require.EqualValues(t, 1, impl.unary.Load())
 // Element conversion failures retain the existing Any conversion fault contract.
 _, anyError := c.Any()(ctx, nil)
 require.Error(t, anyError)
 require.Equal(t, codes.Unknown, status.Code(anyError))
 for _, tc := range []struct {
  endpoint loom.Endpoint
  message string
 }{
  {c.Bad(), "gRPC conversion: nil message collection element"},
  {c.Fail(), "internal server error"},
 } {
  result, err := tc.endpoint(ctx, nil)
  require.Error(t, err)
  require.Nil(t, result)
  require.Equal(t, status.Code(anyError), status.Code(err))
  require.Equal(t, tc.message, status.Convert(err).Message())
 }
 result, err = c.Stream()(ctx, nil)
 require.NoError(t, err)
 stream := result.(svc.StreamClientStream)
 require.NoError(t, stream.Send(valid()))
 first, err := stream.Recv()
 require.NoError(t, err)
 require.Equal(t, valid(), first)
 require.Error(t, stream.Send(invalid()))
 require.NoError(t, stream.Close())
 select {
 case <-impl.done:
 case <-ctx.Done():
  t.Fatal(ctx.Err())
 }
 require.EqualValues(t, 1, impl.stream.Load())
 result, err = c.Badstream()(ctx, nil)
 require.NoError(t, err)
 bad := result.(svc.BadstreamClientStream)
 first, err = bad.Recv()
 require.NoError(t, err)
 require.Equal(t, valid(), first)
 first, err = bad.Recv()
 require.Error(t, err)
 require.Nil(t, first)
 require.Equal(t, status.Code(anyError), status.Code(err))
 require.Equal(t, "internal server error", status.Convert(err).Message())
}
`
