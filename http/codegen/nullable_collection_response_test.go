package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestNullableCollectionResponseBodyReferences(t *testing.T) {
	for _, tc := range []struct {
		name      string
		shape     func() expr.DataType
		clientRef string
		serverRef string
	}{
		{"array", func() expr.DataType {
			return ArrayOf(String)
		}, "loom.Nullable[[]loom.Optional[string]]", "loom.Nullable[[]string]"},
		{"map", func() expr.DataType {
			return MapOf(String, String)
		}, "loom.Nullable[map[string]loom.Optional[string]]", "loom.Nullable[map[string]string]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, streaming := range []bool{false, true} {
				root := RunHTTPDSL(t, func() {
					values := Type("Values", tc.shape(), func() {
						Nullable()
					})
					Service("collections", func() {
						Method("read", func() {
							if streaming {
								StreamingResult(values)
							} else {
								Result(values)
							}
							HTTP(func() {
								GET("/values")
							})
						})
					})
				})
				response := CreateHTTPServices(root).Get("collections").Endpoint("read").Result.Responses[0]
				require.Equal(t, tc.clientRef, response.ClientBody.Ref, "streaming=%t", streaming)
				require.Equal(t, tc.clientRef, response.ClientBody.ValueRef, "streaming=%t", streaming)
				require.Equal(t, tc.clientRef, response.ResultInit.ClientArgs[0].TypeRef, "streaming=%t", streaming)
				require.Equal(t, tc.serverRef, response.ServerBody[0].Ref, "streaming=%t", streaming)
			}
		})
	}
}
