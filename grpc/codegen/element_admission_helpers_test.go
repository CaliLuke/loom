package codegen

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestElementAdmissionSharedHelpers(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		name := "plain first"
		if reverse {
			name = "checked first"
		}
		t.Run(name, func(t *testing.T) {
			root := RunGRPCDSL(t, func() {
				leaf := Type("Leaf", func() {
					Field(1, "name", String)
				})
				var node expr.UserType
				node = Type("Node", func() {
					Field(1, "children", ArrayOf(node))
				})
				plain := Type("Plain", func() {
					Field(1, "leaf", leaf)
				})
				checked := Type("Checked", func() {
					Field(1, "leaf", leaf)
					Field(2, "items", ArrayOf(leaf))
					Field(3, "node", node)
				})
				Service("shared", func() {
					methods := []func(){
						func() {
							Method("plain", func() {
								Payload(plain)
								Result(plain)
								GRPC(func() {})
							})
						},
						func() {
							Method("checked", func() {
								Payload(checked)
								Result(checked)
								GRPC(func() {})
							})
						},
					}
					if reverse {
						methods[0], methods[1] = methods[1], methods[0]
					}
					for _, method := range methods {
						method()
					}
				})
			})
			runWrapperContractModule(t, root, elementAdmissionHelpersHarness)
		})
	}
}

const elementAdmissionHelpersHarness = `package probe

import (
 "testing"

 "github.com/stretchr/testify/require"
 svc "example.com/wrapperprobe/gen/shared"
 "example.com/wrapperprobe/gen/grpc/shared/client"
 "example.com/wrapperprobe/gen/grpc/shared/server"
)

func TestLocalHelperContracts(t *testing.T) {
 // Plain optional Leaf helpers remain infallible and preserve nil in either order.
 plain := &svc.Plain{}
 require.Equal(t, plain, server.NewPlainPayload(client.NewProtoPlainRequest(plain)))
 for _, value := range []*svc.Checked{
  {},
  {Leaf: &svc.Leaf{}},
  {Items: []*svc.Leaf{{}}},
  {Node: &svc.Node{Children: []*svc.Node{{}}}},
 } {
  request, err := client.NewProtoCheckedRequest(value)
  require.NoError(t, err)
  payload, err := server.NewCheckedPayload(request)
  require.NoError(t, err)
  require.Equal(t, value, payload)
  response, err := server.NewProtoCheckedResponse(value)
  require.NoError(t, err)
  result, err := client.NewCheckedResult(response)
  require.NoError(t, err)
  require.Equal(t, value, result)
 }
 for _, value := range []*svc.Checked{
  {Items: []*svc.Leaf{nil}},
  {Node: &svc.Node{Children: []*svc.Node{nil}}},
 } {
  request, err := client.NewProtoCheckedRequest(value)
  require.Error(t, err)
  require.Nil(t, request)
  response, err := server.NewProtoCheckedResponse(value)
  require.Error(t, err)
  require.Nil(t, response)
 }
}
`
