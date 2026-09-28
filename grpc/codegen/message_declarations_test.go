package codegen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestRecursiveMessageDeclarationsUseCanonicalTypes(t *testing.T) {
	root := RunGRPCDSL(t, inlineObjectRecursiveGRPCDSL)
	sd := &ServiceData{Name: "Orchard", Scope: codegen.NewNameScope()}
	message := makeProtoBufMessage(&expr.AttributeExpr{Type: root.UserType("Branch")}, "Branch", sd)
	branch := message.Type.(expr.UserType)
	leaves := expr.AsObject(branch).Attribute("leaves")
	require.IsType(t, &expr.UserTypeExpr{}, leaves.Type, "inline object must become a message")
	branches := expr.AsObject(leaves.Type).Attribute("branches")
	require.Same(t, branch, expr.AsArray(branches.Type).ElemType.Type,
		"recursive references must reach the normalized canonical message")
}

func TestNestedCollectionMessageDeclarations(t *testing.T) {
	root := RunGRPCDSL(t, testdata.MessageDeclarationsDSL)
	services := CreateGRPCServices(root)
	for _, name := range []string{"wrappersfirst", "designfirst"} {
		t.Run(name, func(t *testing.T) {
			sd := services.Get(name)
			require.Len(t, sd.Messages, 6, "request, response, two wrappers and two design types")
			names := make(map[string]bool)
			arrayWrappers := 0
			for _, message := range sd.Messages {
				require.False(t, names[message.VarName], "duplicate emitted name %q", message.VarName)
				names[message.VarName] = true
				if strings.Contains(message.Def, "repeated string field = 1") {
					arrayWrappers++
				}
			}
			require.Equal(t, 1, arrayWrappers, "identical array wrappers remain shared")
		})
	}
}

func TestNestedExplicitMessageDeclarations(t *testing.T) {
	for _, tc := range []struct {
		name, explicit, nested, field, want string
	}{
		{"different fields", "B", "B", "y", "conflicting fields"},
		{"compatible fields", "B", "B", "x", ""},
		{"normalized implicit name", "NodeTree", "node_tree", "y", "conflicting fields"},
		{"different protobuf names with one Go name", "node_tree", "NodeTree", "y", "both map to Go type"},
	} {
		for _, reverse := range []bool{false, true} {
			order := "explicit first"
			if reverse {
				order = "nested first"
			}
			t.Run(tc.name+"/"+order, func(t *testing.T) {
				root := RunGRPCDSL(t, func() {
					a := Type("A", func() {
						Meta("struct:name:proto", tc.explicit)
						Field(1, "x", String)
					})
					b := Type(tc.nested, func() {
						Field(1, tc.field, String)
					})
					nested := func() {
						Field(1, "nested", b)
					}
					Service("svc", func() {
						Method("m", func() {
							if reverse {
								Payload(nested)
								Result(a)
							} else {
								Payload(a)
								Result(nested)
							}
							GRPC(func() {})
						})
					})
				})
				err := generationError(func() {
					ProtoFiles("gen", CreateGRPCServices(root))
				})
				if tc.want == "" {
					require.NoError(t, err)
					dir := t.TempDir()
					renderGRPCModule(t, dir, "example.com/sharednested", root, resolveGRPCLoomSource(t))
					runGRPCGoCommand(t, dir, "mod", "tidy")
					runGRPCGoCommand(t, dir, "vet", "./...")
				} else {
					require.ErrorContains(t, err, tc.want)
				}
			})
		}
	}
}

func TestNestedCollectionMessageDeclarationsGeneratedModule(t *testing.T) {
	runGeneratedRoundTrip(t, "example.com/declarations", testdata.MessageDeclarationsDSL, messageDeclarationsHarness)
}

const messageDeclarationsHarness = `package roundtrip

import (
	"testing"

	"github.com/stretchr/testify/require"

	last "%[1]s/gen/designfirst"
	lastclient "%[1]s/gen/grpc/designfirst/client"
	lastserver "%[1]s/gen/grpc/designfirst/server"
	firstclient "%[1]s/gen/grpc/wrappersfirst/client"
	firstserver "%[1]s/gen/grpc/wrappersfirst/server"
	first "%[1]s/gen/wrappersfirst"
)

func TestWrapperFirst(t *testing.T) {
	count, label := 42, "retained"
	payload := &first.MPayload{
		Matrix:  [][]string{{"a", "b"}, {}},
		Maps:    []map[string]int32{{"k": 17}},
		Again:   map[string][]string{"k": {"x", "y"}},
		Array:   &first.ArrayOfString{Count: &count},
		Mapping: &first.MapOfStringSint32{Label: &label},
	}
	require.Equal(t, payload, firstserver.NewMPayload(firstclient.NewProtoMRequest(payload)))
}

func TestDesignFirst(t *testing.T) {
	count, label := 42, "retained"
	payload := &last.MPayload{
		Matrix:  [][]string{{"a", "b"}, {}},
		Maps:    []map[string]int32{{"k": 17}},
		Again:   map[string][]string{"k": {"x", "y"}},
		Array:   &last.ArrayOfString{Count: &count},
		Mapping: &last.MapOfStringSint32{Label: &label},
	}
	require.Equal(t, payload, lastserver.NewMPayload(lastclient.NewProtoMRequest(payload)))
}
`
