package codegen

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
)

func TestNilCollectionConversionReadiness(t *testing.T) {
	root := RunGRPCDSL(t, nilCollectionDSL)
	runWrapperContractModule(t, root, nilCollectionReadinessHarness)
}

func nilCollectionDSL() {
	leaf := Type("Leaf", func() {
		Field(1, "name", String, func() {
			Default("fallback")
		})
		Field(2, "must", String)
		Required("must")
	})
	node := Type("Node", func() {
		Field(1, "id", String)
		Field(2, "leaves", MapOf(String, leaf))
		Required("id")
	})
	data := Type("Data", func() {
		Field(1, "nodes", ArrayOf(node))
		Field(2, "leaves", ArrayOf(leaf))
	})
	Service("nilitems", func() {
		Method("echo", func() {
			Payload(data)
			Result(data)
			GRPC(func() {
			})
		})
	})
}

const nilCollectionReadinessHarness = `package probe

import (
	"testing"
 "strings"

	"github.com/stretchr/testify/require"

	"example.com/wrapperprobe/gen/nilitems"
	"example.com/wrapperprobe/gen/grpc/nilitems/client"
	"example.com/wrapperprobe/gen/grpc/nilitems/server"
)

func TestNilObjects(t *testing.T) {
	for _, tc := range []struct {
		name string
		value *nilitems.Data
	}{
		{"nil outer object", &nilitems.Data{Nodes: []*nilitems.Node{nil}}},
		{"nil leaf object", &nilitems.Data{Leaves: []*nilitems.Leaf{nil}}},
		{"nil map object", &nilitems.Data{Nodes: []*nilitems.Node{{ID: "id", Leaves: map[string]*nilitems.Leaf{"key": nil}}}}},
		{"valid populated control", &nilitems.Data{Leaves: []*nilitems.Leaf{{Must: "present", Name: "name"}}}},
		{"empty control", &nilitems.Data{Leaves: []*nilitems.Leaf{}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("request", func(t *testing.T) {
				value, err := client.NewProtoEchoRequest(tc.value)
                if strings.HasPrefix(tc.name, "nil ") {
                    require.Error(t, err)
                    require.Nil(t, value)
                } else {
                    require.NoError(t, err)
                    require.NotNil(t, value)
                }
			})
			t.Run("response", func(t *testing.T) {
				value, err := server.NewProtoEchoResponse(tc.value)
                if strings.HasPrefix(tc.name, "nil ") {
                    require.Error(t, err)
                    require.Nil(t, value)
                } else {
                    require.NoError(t, err)
                    require.NotNil(t, value)
                }
			})
		})
	}
}
`
