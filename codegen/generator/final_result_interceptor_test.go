package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/testingx"
	"github.com/stretchr/testify/require"
)

const jsonRPCFinalResultHarness = `package jsonrpcfinal_test

import (
	"context"
	svc "example.com/jsonrpcfinal/gen/files"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
	"testing"
)

type implementation struct{}
type audit struct {
	t     *testing.T
	calls int
}

func (implementation) HandleStream(ctx context.Context, stream svc.Stream) error {
	return stream.Recv(ctx)
}
func (implementation) Upload(_ context.Context, p *svc.Message) (*svc.Message, error) {
	return p, nil
}
func (i *audit) Audit(ctx context.Context, info *svc.AuditInfo, next loom.Endpoint) (any, error) {
	result, err := next(ctx, info.RawPayload())
	if err != nil {
		return nil, err
	}
	require.Equal(i.t, loom.InterceptorUnary, info.CallType())
	require.Equal(i.t, "original", info.Result(result).ID())
	info.Result(result).SetID("modified")
	i.calls++
	return result, nil
}
func TestUnaryBoundary(t *testing.T) {
	id := "original"
	interceptor := &audit{t: t}
	result, err := svc.NewEndpoints(implementation{}, interceptor).Upload(context.Background(), &svc.Message{ID: &id})
	require.NoError(t, err)
	require.Equal(t, "modified", *result.(*svc.Message).ID)
	require.Equal(t, 1, interceptor.calls)
}
`

// TestGRPCFinalResultInterceptorsCompile covers final result wrappers on gRPC
// client streams without an ordinary payload, including dynamic result views.
func TestGRPCFinalResultInterceptorsCompile(t *testing.T) {
	for _, viewed := range []bool{false, true} {
		name := "plain"
		if viewed {
			name = "viewed"
		}
		t.Run(name, func(t *testing.T) {
			dir := buildGeneratedModule(t, "example.com/grpcfinal", func() {
				dsl.Interceptor("audit", func() {
					dsl.ReadStreamingResult(func() {
						dsl.Attribute("count")
					})
					dsl.WriteStreamingResult(func() {
						dsl.Attribute("count")
					})
				})
				fields := func() {
					dsl.Field(1, "count", dsl.Int)
					dsl.Field(2, "note", dsl.String)
				}
				var result any
				if viewed {
					result = dsl.ResultType("application/vnd.finalreply", "Reply", func() {
						fields()
						dsl.View("default", func() {
							dsl.Attribute("count")
							dsl.Attribute("note")
						})
						dsl.View("tiny", func() {
							dsl.Attribute("count")
						})
					})
				} else {
					result = dsl.Type("Reply", fields)
				}
				dsl.Service("files", func() {
					dsl.ServerInterceptor("audit")
					dsl.ClientInterceptor("audit")
					dsl.Method("upload", func() {
						dsl.StreamingPayload(func() {
							dsl.Field(1, "chunk", dsl.String)
						})
						dsl.Result(result)
						dsl.GRPC(func() {})
					})
				})
			})
			output, err := testingx.RunCmd(dir, "go", "vet", "./...")
			require.NoError(t, err, output)
		})
	}
}

// TestJSONRPCClientStreamKeepsUnaryResult verifies the JSON-RPC request/reply
// boundary, where the service returns a value instead of calling SendAndClose.
func TestJSONRPCClientStreamKeepsUnaryResult(t *testing.T) {
	dir := buildGeneratedModule(t, "example.com/jsonrpcfinal", func() {
		dsl.API("final", func() {
			dsl.JSONRPC(func() {})
		})
		dsl.Interceptor("audit", func() {
			dsl.ReadResult(func() {
				dsl.Attribute("id")
			})
			dsl.WriteResult(func() {
				dsl.Attribute("id")
			})
		})
		message := dsl.Type("Message", func() {
			dsl.Attribute("id", dsl.String)
		})
		dsl.Service("files", func() {
			dsl.JSONRPC(func() {
				dsl.GET("/ws")
			})
			dsl.ServerInterceptor("audit")
			dsl.Method("upload", func() {
				dsl.StreamingPayload(message)
				dsl.Result(message)
				dsl.JSONRPC(func() {})
			})
		})
	})
	require.NoError(t, os.WriteFile(filepath.Join(dir, "boundary_test.go"), []byte(jsonRPCFinalResultHarness), 0o600))
	output, err := testingx.RunCmd(dir, "go", "mod", "tidy")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(dir, "go", "test", "-race", "./...")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(dir, "go", "vet", "./...")
	require.NoError(t, err, output)
}
