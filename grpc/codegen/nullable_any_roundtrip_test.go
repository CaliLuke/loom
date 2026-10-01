package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

const nullableAnyRootPayloadModule = "example.com/nullany"

var nullableAnyRootPayloadHarness = fmt.Sprintf(`package roundtrip

import (
	"testing"

	"github.com/stretchr/testify/require"
	structpb "google.golang.org/protobuf/types/known/structpb"

	client "%[1]s/gen/grpc/nullable_any/client"
	server "%[1]s/gen/grpc/nullable_any/server"
	loom "github.com/CaliLuke/loom/pkg"
)

func TestNullableAnyRequestConverters(t *testing.T) {
	var absent loom.Nullable[loom.JSONValue]
	null := loom.NullValue[loom.JSONValue]()
	value := loom.NullableValue(loom.JSONValue(%[2]q))

	for _, tc := range []struct {
		name string
		in loom.Nullable[loom.JSONValue]
		checkMessage func(*testing.T, *structpb.Value)
		checkOutput func(*testing.T, loom.Nullable[loom.JSONValue])
	}{
		{
			name: "absent",
			in: absent,
			checkMessage: func(t *testing.T, field *structpb.Value) {
				require.Nil(t, field)
			},
			checkOutput: func(t *testing.T, got loom.Nullable[loom.JSONValue]) {
				require.False(t, got.Present())
			},
		},
		{
			name: "null",
			in: null,
			checkMessage: func(t *testing.T, field *structpb.Value) {
				_, ok := field.GetKind().(*structpb.Value_NullValue)
				require.True(t, ok)
			},
			checkOutput: func(t *testing.T, got loom.Nullable[loom.JSONValue]) {
				require.True(t, got.IsNull())
			},
		},
		{
			name: "value",
			in: value,
			checkMessage: func(t *testing.T, field *structpb.Value) {
				require.NotNil(t, field.GetStructValue())
			},
			checkOutput: func(t *testing.T, got loom.Nullable[loom.JSONValue]) {
				actual, ok := got.Value()
				require.True(t, ok)
				require.JSONEq(t, %[2]q, string(actual))
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			message, err := client.NewProtoSendRequest(tc.in)
			require.NoError(t, err)
			tc.checkMessage(t, message.Field)
			got, err := server.NewSendPayload(message)
			require.NoError(t, err)
			tc.checkOutput(t, got)
		})
	}
}

func TestNullableAnyMalformedJSON(t *testing.T) {
	message, err := client.NewProtoSendRequest(loom.NullableValue(loom.JSONValue("{")))
	require.Error(t, err)
	require.Nil(t, message)
}
`, nullableAnyRootPayloadModule, `{"text":"kept"}`)

func TestGeneratedNullableAnyRootPayloadRoundTrip(t *testing.T) {
	root := RunGRPCDSL(t, nullableAnyRootPayloadDSL)
	dir := t.TempDir()
	renderGRPCModule(t, dir, nullableAnyRootPayloadModule, root, resolveGRPCLoomSource(t))
	testDir := filepath.Join(dir, "internal", "roundtrip")
	require.NoError(t, os.MkdirAll(testDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "roundtrip_test.go"), []byte(nullableAnyRootPayloadHarness), 0o600))

	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "build", "./...")
	runGRPCGoCommand(t, dir, "vet", "./...")
	runGRPCGoCommand(t, dir, "test", "./internal/roundtrip")
}

func nullableAnyRootPayloadDSL() {
	API("nullable-any", func() {
	})
	Service("nullable-any", func() {
		Method("send", func() {
			Payload(Any, func() {
				Nullable()
				Example(Null())
			})
			Result(Any)
			GRPC(func() {
			})
		})
	})
}
