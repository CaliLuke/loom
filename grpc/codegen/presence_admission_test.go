package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

const localAnyPresenceHarness = `package localany_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
	client "example.com/localany/gen/grpc/local_any/client"
	server "example.com/localany/gen/grpc/local_any/server"
	svc "example.com/localany/gen/local_any"
)

func TestRoundTrip(t *testing.T) {
	for _, value := range []loom.JSONValue{nil, loom.JSONValue("null"), loom.JSONValue("42")} {
		message, err := client.NewProtoSendRequest(&svc.SendPayload{Value: value})
		require.NoError(t, err)
		got, err := server.NewSendPayload(message)
		require.NoError(t, err)
		require.Equal(t, value, got.Value)
	}
}
`

func TestGeneratedLocalNullableAnyPresence(t *testing.T) {
	root := RunGRPCDSL(t, func() {
		Service("local-any", func() {
			Method("send", func() {
				Payload(func() {
					Field(1, "value", Any)
				})
				GRPC(func() {
					Message(func() {
						Attribute("value", func() {
							Nullable()
						})
					})
				})
			})
		})
	})
	dir := t.TempDir()
	renderGRPCModule(t, dir, "example.com/localany", root, resolveGRPCLoomSource(t))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "presence_test.go"), []byte(localAnyPresenceHarness), 0o600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "build", "./...")
	runGRPCGoCommand(t, dir, "vet", "./...")
	runGRPCGoCommand(t, dir, "test", ".")
}
