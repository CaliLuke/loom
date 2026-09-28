package codegen

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

// TestBodyTypeNamesSeparateEndpointAndNestedType covers an endpoint whose
// synthesized body name is also the transport name of a nested user type.
func TestBodyTypeNamesSeparateEndpointAndNestedType(t *testing.T) {
	for _, union := range []bool{false, true} {
		name := "object"
		if union {
			name = "union"
		}
		t.Run(name, func(t *testing.T) {
			root := RunHTTPDSL(t, bodyTypeNameCollisionDSL(union))
			sd := CreateHTTPServices(root).Get("svc")
			request := sd.Endpoints[0].Payload.Request
			for _, body := range []*TypeData{request.ServerBody, request.ClientBody} {
				require.NotEqual(t, "OtherRequestBody", body.VarName)
				require.Equal(t, body.VarName, strings.TrimPrefix(body.Ref, "*"))
			}
		})
	}
}

// TestBodyNameAllocation reserves types inside anonymous collections and
// existing suffixes before assigning body names, and retains copy identity.
func TestBodyNameAllocation(t *testing.T) {
	for _, collection := range []bool{false, true} {
		for _, occupiedSuffix := range []bool{false, true} {
			t.Run(fmt.Sprintf("collection=%t/suffix=%t", collection, occupiedSuffix), func(t *testing.T) {
				user := func(name, id string) *expr.UserTypeExpr {
					return &expr.UserTypeExpr{TypeName: name, UID: id, AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}}}
				}
				nested := user("OtherRequestBody", "authored-other")
				body := user("OtherRequestBody", "endpoint-other")
				body.Type = &expr.Object{{Name: "other", Attribute: &expr.AttributeExpr{Type: nested}}}
				att := &expr.AttributeExpr{Type: body}
				roots := []*expr.AttributeExpr{att, expr.DupAtt(att)}
				if collection {
					body.Type = &expr.Object{}
					roots[1] = expr.DupAtt(att)
					roots = append(roots, &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: nested}}})
				}
				want := "OtherRequestBody2"
				if occupiedSuffix {
					roots = append(roots, &expr.AttributeExpr{Type: user(want, "another-endpoint")})
					want = "OtherRequestBody3"
				}
				endpoints := make([]*transportir.Endpoint, len(roots))
				for i, root := range roots {
					endpoints[i] = &transportir.Endpoint{
						Request:  &transportir.Request{Body: root},
						Response: &transportir.Response{},
					}
				}
				nameBodyTypes(endpoints)
				require.Equal(t, want, roots[0].Type.Name())
				require.Equal(t, want, roots[1].Type.Name(), "copies must share their declaration")
				require.Equal(t, "OtherRequestBody", nested.Name())
			})
		}
	}
}

// TestBodyNamesPreserveOpenAPI checks that allocating Go identifiers does not
// rename authored schemas or change the wire contract.
func TestBodyNamesPreserveOpenAPI(t *testing.T) {
	root := RunHTTPDSL(t, testdata.TypeIdentityDSL)
	render := func() []byte {
		files, err := openapiv3.Files(root)
		require.NoError(t, err)
		for _, file := range files {
			if filepath.Ext(file.Path) == ".json" {
				var buf bytes.Buffer
				require.NoError(t, file.AllSections()[0].Write(&buf))
				return buf.Bytes()
			}
		}
		t.Fatal("missing OpenAPI JSON")
		return nil
	}
	before := render()
	services := CreateHTTPServices(root)
	for _, service := range root.API.HTTP.Services {
		services.Get(service.Name())
	}
	require.Equal(t, string(before), string(render()))
	document, err := libopenapi.NewDocument(before)
	require.NoError(t, err)
	_, err = document.BuildV3Model()
	require.NoError(t, err)
}

// TestBodyNameAllocationCycles checks that recursive references retain their
// target while endpoint wrappers and nested types receive distinct names.
func TestBodyNameAllocationCycles(t *testing.T) {
	root := &expr.UserTypeExpr{TypeName: "OtherRequestBody", UID: "endpoint", AttributeExpr: &expr.AttributeExpr{}}
	nested := &expr.UserTypeExpr{TypeName: "OtherRequestBody", UID: "authored", AttributeExpr: &expr.AttributeExpr{}}
	root.Type = &expr.Object{
		{Name: "self", Attribute: &expr.AttributeExpr{Type: root}},
		{Name: "other", Attribute: &expr.AttributeExpr{Type: nested}},
	}
	nested.Type = &expr.Object{{Name: "self", Attribute: &expr.AttributeExpr{Type: nested}}}
	endpoint := &transportir.Endpoint{
		Request:  &transportir.Request{Body: &expr.AttributeExpr{Type: root}},
		Response: &transportir.Response{},
	}
	nameBodyTypes([]*transportir.Endpoint{endpoint})
	require.Equal(t, "OtherRequestBody2", root.Name())
	require.Equal(t, "OtherRequestBody", nested.Name())
	require.Same(t, root, root.Find("self").Type)
	require.Same(t, nested, nested.Find("self").Type)
}

// TestBodyNameAllocationDistinctCopies keeps a nested definition distinct even
// when its public ID and name equal those of the synthesized wrapper. The
// expression copier separates them using its private wrapper category.
func TestBodyNameAllocationDistinctCopies(t *testing.T) {
	nested := &expr.UserTypeExpr{TypeName: "OtherRequestBody", UID: "shared", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}}}
	root := &expr.UserTypeExpr{TypeName: nested.Name(), UID: nested.ID(), AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Object{{Name: "other", Attribute: &expr.AttributeExpr{Type: nested}}},
	}}
	endpoint := &transportir.Endpoint{
		Request:  &transportir.Request{Body: &expr.AttributeExpr{Type: root}},
		Response: &transportir.Response{},
	}
	nameBodyTypes([]*transportir.Endpoint{endpoint})
	require.Equal(t, "OtherRequestBody2", root.Name())
	require.Equal(t, "OtherRequestBody", nested.Name())
}

func bodyTypeNameCollisionDSL(union bool) func() {
	return func() {
		leaf, other := unionBodyBranchTypes()
		Service("svc", func() {
			Method("other", func() {
				if union {
					Payload(OneOf(leaf, other))
					Result(OneOf(leaf, other))
				} else {
					Payload(func() {
						Attribute("other", other)
						Required("other")
					})
					Result(func() {
						Attribute("other", other)
						Required("other")
					})
				}
				HTTP(func() {
					POST("/other")
				})
			})
		})
	}
}

// TestBodyTypeNamesGeneratedCompile checks declarations, references, validators
// and conversion helpers together in generated client and server packages.
func TestBodyTypeNamesGeneratedCompile(t *testing.T) {
	for _, union := range []bool{false, true} {
		name := "object"
		if union {
			name = "union"
		}
		t.Run(name, func(t *testing.T) {
			root := RunHTTPDSL(t, bodyTypeNameCollisionDSL(union))
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/bodynames", root)
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "test", "./...")
		})
	}
}

// TestBodyTypeIdentityPreservesOpenAPIExample checks a branch whose authored
// identity equals the synthesized body ID, with a canonical schema name.
func TestBodyTypeIdentityPreservesOpenAPIExample(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		API("bodyprobe", func() {})
		branch := Type("svc#Other", func() {
			Attribute("value", String)
			Required("value")
		})
		leaf := Type("Leaf", func() {
			Attribute("name", String)
			Required("name")
		})
		choice := Type("Choice", OneOf(branch, leaf))
		Service("svc", func() {
			Method("other", func() {
				Payload(choice, func() {
					Meta("openapi:typename", "Choice")
				})
				Result(choice)
				HTTP(func() {
					POST("/")
				})
			})
		})
	})
	spec := renderOpenAPIJSON(t, openapiv3.Files, root)
	parseOpenAPIV3Document(t, spec)
	media := postRequestMediaTypeFromSpec(t, spec, "/", "application/json")
	example, ok := media["example"].(map[string]any)
	require.True(t, ok, "body identity must not suppress the generated example")
	require.Equal(t, "Svc#Other", example["type"])
	value, ok := example["value"].(map[string]any)
	require.True(t, ok)
	require.IsType(t, "", value["value"])
}
