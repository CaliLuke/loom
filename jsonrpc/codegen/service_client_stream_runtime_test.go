package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServiceClientWebSocketStreamRuntime(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcWebSocketPendingOrderDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcwspending", root)
	harness := strings.Replace(jsonRPCWebSocketPendingOrderHarness, `"github.com/CaliLuke/loom/jsonrpc"`, `"github.com/CaliLuke/loom/jsonrpc"
 loom "github.com/CaliLuke/loom/pkg"`, 1)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "contract_test.go"), []byte(harness+serviceClientWebSocketHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-run", "^TestTypedServiceStream$", "-count=1", ".")
}

const serviceClientWebSocketHarness = `
type contractKey struct{}
type clientAudit struct {
	t              *testing.T
	sent, received int
}

func (a *clientAudit) Audit(ctx context.Context, info *echo.AuditInfo, next loom.Endpoint) (any, error) {
	if ctx.Value(contractKey{}) != "opening" {
		a.t.Errorf("lost operation context")
	}
	if info.CallType() == loom.InterceptorStreamingSend {
		a.sent++
		text := info.ClientStreamingPayload().Text()
		if text != "slow" {
			text += "!"
		}
		return next(ctx, &echo.Note{Text: text})
	}
	a.received++
	result, err := next(ctx, nil)
	if err == nil {
		text := info.ClientStreamingResult(result).Text()
		info.ClientStreamingResult(result).SetText(text + "?")
	}
	return result, err
}

func TestTypedServiceStream(t *testing.T) {
	for _, intercepted := range []bool{false, true} {
		t.Run(fmt.Sprint(intercepted), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.WithValue(context.Background(), contractKey{}, "opening"))
			defer cancel()
			tc := newClient(t, newService(), newGatedConn(), nil)
			defer func() {
				if err := tc.Close(); err != nil {
					t.Error(err)
				}
			}()
			audit := &clientAudit{t: t}
			var ci echo.ClientInterceptors
			if intercepted {
				ci = audit
			}
			c := echo.NewClient(tc.Talk(), ci)
			stream, err := c.Talk(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			peer, err := c.Talk(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := peer.Close(); err != nil {
					t.Error(err)
				}
			}()
			if err := stream.Send(&echo.Note{Text: "one"}); err != nil {
				t.Fatal(err)
			}
			got, err := stream.Recv()
			want := "one"
			if intercepted {
				want = "one!?"
			}
			if err != nil || got.Text != want {
				t.Fatalf("got %v, %v", got, err)
			}
			if err := stream.Close(); err != nil {
				t.Fatal(err)
			}
			if _, err := stream.Recv(); err == nil {
				t.Fatal("closed stream received")
			}
			if err := peer.SendWithContext(ctx, &echo.Note{Text: "two"}); err != nil {
				t.Fatal(err)
			}
			got, err = peer.RecvWithContext(ctx)
			want = "two"
			if intercepted {
				want = "two!?"
			}
			if err != nil || got.Text != want {
				t.Fatalf("peer got %v, %v", got, err)
			}
			if err := peer.Send(&echo.Note{Text: "slow"}); err != nil {
				t.Fatal(err)
			}
			call, stop := context.WithCancel(ctx)
			stop()
			if _, err := peer.RecvWithContext(call); !errors.Is(err, context.Canceled) {
				t.Fatalf("operation cancellation: %v", err)
			}
			cancel()
			if _, err := peer.Recv(); !errors.Is(err, context.Canceled) {
				t.Fatalf("opening cancellation: %v", err)
			}
			if intercepted && (audit.sent != 3 || audit.received != 5) {
				t.Fatalf("interception counts %+v", audit)
			}
		})
	}
}
`
