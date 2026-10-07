package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
)

func TestUntaggedArrayKeepsProtobufOneof(t *testing.T) {
	design := func(untagged bool) func() {
		return func() {
			item := Type("Item", func() {
				Field(1, "name", String)
			})
			items := Type("Items", ArrayOf(item))
			page := Type("Page", func() {
				Field(1, "total", Int)
			})
			Service("listing", func() {
				Method("list", func() {
					Result(func() {
						Field(1, "choice", OneOf(items, page), func() {
							if untagged {
								Untagged()
							}
						})
					})
					GRPC(func() {

					})
				})
			})
		}
	}
	tagged := protoFileCode(t, design(false))
	untagged := protoFileCode(t, design(true))
	require.Equal(t, tagged, untagged, "JSON encoding must not change protobuf identity or wrappers")
	require.Contains(t, untagged, "oneof choice")
	require.Contains(t, untagged, "repeated Item")
	require.NoError(t, protoc(defaultProtocCmd, codegen.CreateTempFile(t, untagged), nil))
}
