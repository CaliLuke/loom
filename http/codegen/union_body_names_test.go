package codegen

import (
	"bytes"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"path/filepath"
	"sort"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
)

// unionBodyNameCase is a design whose method payloads and results are unions,
// with the Go names expected for its transport union types and the OpenAPI
// component names that the design must keep.
type unionBodyNameCase struct {
	Name string
	DSL  func()
	// Unions lists the transport union types with their branch field types.
	Unions map[string][]string
	// Components lists the OpenAPI component schema names.
	Components []string
}

// unionBodyNameCases covers anonymous and named unions used as the payload,
// the result or both, a named union used as an explicit request body and as
// the result, and one union shared by the results of two methods.
var unionBodyNameCases = []unionBodyNameCase{
	{
		Name: "anonymous-payload-result",
		DSL:  unionBodyDSL(func(l, o expr.UserType) (any, any) { return OneOf(l, o), OneOf(l, o) }),
		Unions: map[string][]string{
			"PickRequestBody":  {"*LeafRequestBody", "*OtherRequestBody"},
			"PickResponseBody": {"*LeafResponse", "*OtherResponse"},
		},
		Components: []string{"Leaf", "LeafOrOtherLeafEnvelope", "LeafOrOtherOtherEnvelope", "Other"},
	},
	{
		Name: "named-payload-result",
		DSL: unionBodyDSL(func(l, o expr.UserType) (any, any) {
			u := Type("Choice", OneOf(l, o))
			return u, u
		}),
		Unions: map[string][]string{
			"PickRequestBody":  {"*LeafRequestBody", "*OtherRequestBody"},
			"PickResponseBody": {"*LeafResponse", "*OtherResponse"},
		},
		Components: []string{"ChoiceLeafEnvelope", "ChoiceOtherEnvelope", "Leaf", "Other"},
	},
	{
		Name: "named-payload",
		DSL: unionBodyDSL(func(l, o expr.UserType) (any, any) {
			return Type("Choice", OneOf(l, o)), nil
		}),
		Unions: map[string][]string{
			"PickRequestBody": {"*LeafRequestBody", "*OtherRequestBody"},
		},
		Components: []string{"ChoiceLeafEnvelope", "ChoiceOtherEnvelope", "Leaf", "Other"},
	},
	{
		Name: "named-result",
		DSL: unionBodyDSL(func(l, o expr.UserType) (any, any) {
			return nil, Type("Choice", OneOf(l, o))
		}),
		Unions: map[string][]string{
			"PickResponseBody": {"*LeafResponse", "*OtherResponse"},
		},
		Components: []string{"ChoiceLeafEnvelope", "ChoiceOtherEnvelope", "Leaf", "Other"},
	},
	{
		Name: "explicit-body",
		DSL: func() {
			leaf, other := unionBodyBranchTypes()
			choice := Type("Choice", OneOf(leaf, other))
			Service("svc", func() {
				Method("pick", func() {
					Payload(func() {
						Attribute("q", String)
						Attribute("u", choice)
					})
					Result(choice)
					HTTP(func() {
						POST("/pick")
						Param("q")
						Body("u")
					})
				})
			})
		},
		Unions: map[string][]string{
			// The explicit body is renamed twice by the design, see
			// TestUnionHTTPBodyBranchesAreSuffixed.
			"PickRequestBody":  {"*LeafRequestBodyRequestBody", "*OtherRequestBodyRequestBody"},
			"PickResponseBody": {"*LeafResponse", "*OtherResponse"},
		},
		Components: []string{"Choice", "ChoiceLeafEnvelope", "ChoiceOtherEnvelope", "Leaf", "Other"},
	},
	{
		Name: "shared-result",
		DSL: func() {
			leaf, other := unionBodyBranchTypes()
			Service("svc", func() {
				Method("first", func() {
					Result(OneOf(leaf, other))
					HTTP(func() {
						GET("/first")
					})
				})
				Method("second", func() {
					Result(OneOf(leaf, other))
					HTTP(func() {
						GET("/second")
					})
				})
			})
		},
		Unions: map[string][]string{
			"FirstResponseBody":  {"*LeafResponse", "*OtherResponse"},
			"SecondResponseBody": {"*LeafResponse", "*OtherResponse"},
		},
		Components: []string{"Leaf", "LeafOrOtherLeafEnvelope", "LeafOrOtherOtherEnvelope", "Other"},
	},
}

// TestUnionBodyTypeNames checks that every union request and response body
// has its own transport type named after the endpoint and the body, that the
// request and response branches use the request and response layouts, and
// that every body the generated code declares refers to one of those types.
func TestUnionBodyTypeNames(t *testing.T) {
	for _, c := range unionBodyNameCases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.DSL)
			sd := CreateHTTPServices(root).Get("svc")
			unions := make(map[string][]string, len(sd.UnionTypes))
			for _, union := range sd.UnionTypes {
				fields := make([]string, 0, len(union.Fields))
				for _, field := range union.Fields {
					fields = append(fields, field.FieldType)
				}
				unions[union.Name] = fields
			}
			assert.Equal(t, c.Unions, unions)

			var bodies []string
			for _, endpoint := range sd.Endpoints {
				if request := endpoint.Payload.Request; request.ServerBody != nil {
					bodies = append(bodies, request.ServerBody.VarName, request.ClientBody.VarName)
				}
				if endpoint.Result == nil {
					continue
				}
				for _, response := range endpoint.Result.Responses {
					for _, body := range response.ServerBody {
						bodies = append(bodies, body.VarName)
					}
					if response.ClientBody != nil {
						bodies = append(bodies, response.ClientBody.VarName)
					}
				}
			}
			require.NotEmpty(t, bodies)
			for _, body := range bodies {
				assert.Contains(t, unions, body, "body type %s is not a declared union", body)
			}
		})
	}
}

// TestUnionBodyOpenAPIComponents checks that the transport names of union
// bodies do not reach the OpenAPI document: it keeps the service type names
// of the unions and their branches, and it parses with libopenapi.
func TestUnionBodyOpenAPIComponents(t *testing.T) {
	for _, c := range unionBodyNameCases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.DSL)
			files, err := openapiv3.Files(root)
			require.NoError(t, err)
			var spec []byte
			for _, file := range files {
				if filepath.Ext(file.Path) != ".json" {
					continue
				}
				var buf bytes.Buffer
				require.NoError(t, file.AllSections()[0].Write(&buf))
				spec = buf.Bytes()
			}
			require.NotEmpty(t, spec)
			parsed, err := libopenapi.NewDocument(spec)
			require.NoError(t, err)
			_, err = parsed.BuildV3Model()
			require.NoError(t, err)

			var document struct {
				Components struct {
					Schemas map[string]jsontext.Value `json:"schemas"`
				} `json:"components"`
			}
			require.NoError(t, json.Unmarshal(spec, &document))
			names := make([]string, 0, len(document.Components.Schemas))
			for name := range document.Components.Schemas {
				names = append(names, name)
			}
			sort.Strings(names)
			assert.Equal(t, c.Components, names)
		})
	}
}

// unionBodyDSL returns a design whose "pick" method uses the payload and the
// result that types builds from the two branch types. A nil payload or
// result is omitted.
func unionBodyDSL(types func(leaf, other expr.UserType) (payload, result any)) func() {
	return func() {
		leaf, other := unionBodyBranchTypes()
		payload, result := types(leaf, other)
		Service("svc", func() {
			Method("pick", func() {
				if payload != nil {
					Payload(payload)
				}
				if result != nil {
					Result(result)
				}
				HTTP(func() {
					POST("/pick")
				})
			})
		})
	}
}

// unionBodyBranchTypes defines the object types used as union branches.
func unionBodyBranchTypes() (expr.UserType, expr.UserType) {
	leaf := Type("Leaf", func() {
		Attribute("name", String)
		Required("name")
	})
	other := Type("Other", func() {
		Attribute("count", Int)
	})
	return leaf, other
}
