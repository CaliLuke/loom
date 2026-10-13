package codegen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetadataDefaultPresenceOwner(t *testing.T) {
	root := RunGRPCDSL(t, metadataDefaultsDSL)
	endpoint := CreateGRPCServices(root).Get("mdsvc").Endpoint("echo")
	for _, fields := range [][]*MetadataData{endpoint.Request.Metadata, endpoint.Response.Headers, endpoint.Response.Trailers} {
		for _, field := range fields {
			t.Run(field.Name, func(t *testing.T) {
				code := renderGRPCMetadataDecode(field, "md", false)
				require.Equal(t, 1, strings.Count(code, "md.Get("))
				if field.DefaultValue != nil && !field.Required {
					require.Contains(t, code, "} else {")
				}
				if field.Required {
					require.Contains(t, code, "loom.MissingFieldError")
					require.NotContains(t, code, "required default")
				}
			})
		}
	}
}

func TestMetadataDefaultLocalOccurrence(t *testing.T) {
	root := RunGRPCDSL(t, metadataLocalDefaultsDSL)
	field := CreateGRPCServices(root).Get("local").Endpoint("read").Request.Metadata[0]
	require.True(t, field.Pointer)
	require.Equal(t, "7", field.DefaultLiteral)
	require.Contains(t, renderGRPCMetadataDecode(field, "md", false), "new(int32)")
	runMetadataDefaultsModule(t, root, map[string]string{"local_test.go": metadataLocalDefaultHarness})
}

const metadataLocalDefaultHarness = `package rt

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"example.com/mdprobe/gen/local"
	"example.com/mdprobe/gen/grpc/local/server"
)

func TestLocalDefault(t *testing.T) {
	for _, tc := range []struct {
		name string
		values metadata.MD
		want int32
	}{
		{"absent", nil, 7},
		{"zero", metadata.Pairs("value", "0"), 0},
		{"ordinary", metadata.Pairs("value", "9"), 9},
	} {
		t.Run(tc.name, func(t *testing.T) {
			decoded, err := server.DecodeReadRequest(context.Background(), nil, tc.values)
			require.NoError(t, err)
			value := decoded.(*local.ReadPayload).Value
			require.NotNil(t, value)
			require.Equal(t, tc.want, *value)
		})
	}
}
`
