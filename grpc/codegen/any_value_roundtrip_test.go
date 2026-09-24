package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestGeneratedAnyValueRoundTrip serves a generated module whose payloads and
// results carry an Any value in a OneOf branch and in a field typed with an
// alias of Any. It sends an object, a scalar and a nil Any through the
// generated client over an in-memory gRPC connection and checks the JSON
// bytes the client decodes from the echoed result.
func TestGeneratedAnyValueRoundTrip(t *testing.T) {
	const modulePath = "example.com/grpcanyroundtrip"
	root := RunGRPCDSL(t, anyValueRoundTripDSL)
	dir := t.TempDir()
	renderGRPCModule(t, dir, modulePath, root, resolveGRPCLoomSource(t))
	testDir := filepath.Join(dir, "internal", "roundtrip")
	require.NoError(t, os.MkdirAll(testDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "roundtrip_test.go"), []byte(anyValueRoundTripHarness), 0o600))

	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "build", "./...")
	runGRPCGoCommand(t, dir, "vet", "./...")
	runGRPCGoCommand(t, dir, "test", "./internal/roundtrip")
}

func anyValueRoundTripDSL() {
	var Picked = Type("Picked", func() {
		OneOf("pick", func() {
			Field(1, "text", String)
			Field(2, "value", Any)
		})
	})
	var Blob = Type("Blob", Any)
	var Boxed = Type("Boxed", func() {
		Field(1, "value", Blob)
	})
	Service("anyecho", func() {
		Method("echo", func() {
			Payload(Picked)
			Result(Picked)
			GRPC(func() {})
		})
		Method("box", func() {
			Payload(Boxed)
			Result(Boxed)
			GRPC(func() {})
		})
	})
}

var anyValueRoundTripHarness = fmt.Sprintf(`package roundtrip

import (
	"context"
	"net"
	"testing"

	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	anyecho "%[1]s/gen/anyecho"
	"%[1]s/gen/grpc/anyecho/client"
	pb "%[1]s/gen/grpc/anyecho/pb"
	"%[1]s/gen/grpc/anyecho/server"
)

type service struct{}

func (service) Echo(_ context.Context, p *anyecho.Picked) (*anyecho.Picked, error) {
	return p, nil
}

func (service) Box(_ context.Context, p *anyecho.Boxed) (*anyecho.Boxed, error) {
	return p, nil
}

func dial(t *testing.T) *grpc.ClientConn {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	pb.RegisterAnyechoServer(srv, server.New(anyecho.NewEndpoints(service{}), nil))
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
	return conn
}

func TestAnyRoundTrip(t *testing.T) {
	c := client.NewClient(dial(t))
	ctx := context.Background()
	// A nil Any in a OneOf branch selects the branch and comes back as JSON
	// null. A nil Any field is absent, as for every other optional field.
	// Objects have a single member because google.protobuf.Value does not
	// keep member order.
	cases := []struct {
		Name       string
		Sent       loom.JSONValue
		WantBranch string
		WantField  string
	}{
		{"object", loom.JSONValue(`+"`"+`{"a":[1,"x",true,null]}`+"`"+`), `+"`"+`{"a":[1,"x",true,null]}`+"`"+`, `+"`"+`{"a":[1,"x",true,null]}`+"`"+`},
		{"scalar", loom.JSONValue(`+"`"+`42`+"`"+`), `+"`"+`42`+"`"+`, `+"`"+`42`+"`"+`},
		{"nil", nil, `+"`"+`null`+"`"+`, ""},
	}
	for _, tc := range cases {
		t.Run("oneof-branch/"+tc.Name, func(t *testing.T) {
			sent := &anyecho.Picked{Pick: &anyecho.Pick{}}
			sent.Pick.SetValue(tc.Sent)
			res, err := c.Echo()(ctx, sent)
			require.NoError(t, err)
			got := res.(*anyecho.Picked)
			require.NotNil(t, got.Pick)
			value, ok := got.Pick.AsValue()
			require.True(t, ok)
			require.Equal(t, tc.WantBranch, string(value))
		})
		t.Run("alias-field/"+tc.Name, func(t *testing.T) {
			res, err := c.Box()(ctx, &anyecho.Boxed{Value: tc.Sent})
			require.NoError(t, err)
			got := res.(*anyecho.Boxed)
			require.Equal(t, tc.WantField, string(got.Value))
		})
	}
}
`, "example.com/grpcanyroundtrip")
