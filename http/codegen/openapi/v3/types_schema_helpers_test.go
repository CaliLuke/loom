package openapiv3

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	"github.com/CaliLuke/loom/http/codegen/openapi/v3/testdata/dsls"
)

func TestMapTypes(t *testing.T) {
	svcName := "test-service"

	testCases := []struct {
		Name     string
		DSL      func()
		Expected typ
	}{
		{
			Name: "map_int_array_string",
			DSL:  dsls.MapIntKeyBodyDSL(svcName, "map_int_array_string"),
			Expected: typ{
				Type: "object",
				Props: []attr{{Name: "intmap", Val: typ{
					Type: "object",
					AdditionalProperties: &additionalPropsType{
						Type:  "array",
						Items: &additionalPropsType{Type: "string"},
					},
				}}},
			},
		},
		{
			Name: "map_int_array_object",
			DSL:  dsls.MapIntKeyObjectBodyDSL(svcName, "map_int_array_object"),
			Expected: typ{
				Type: "object",
				Props: []attr{{Name: "intmap", Val: typ{
					Type: "object",
					AdditionalProperties: &additionalPropsType{
						Type:  "array",
						Items: &additionalPropsType{Ref: "#/components/schemas/MapData"},
					},
				}}},
			},
		},
		{
			Name: "map_int_string",
			DSL:  dsls.MapIntKeyStringBodyDSL(svcName, "map_int_string"),
			Expected: typ{
				Type: "object",
				Props: []attr{{Name: "intmap", Val: typ{
					Type:                 "object",
					AdditionalProperties: &additionalPropsType{Type: "string"},
				}}},
			},
		},
		{
			Name: "map_int_object_direct",
			DSL:  dsls.MapIntKeyObjectDirectBodyDSL(svcName, "map_int_object_direct"),
			Expected: typ{
				Type: "object",
				Props: []attr{{Name: "intmap", Val: typ{
					Type:                 "object",
					AdditionalProperties: &additionalPropsType{Ref: "#/components/schemas/MapData"},
				}}},
			},
		},
		{
			Name: "map_string_int",
			DSL:  dsls.MapStringKeyIntBodyDSL(svcName, "map_string_int"),
			Expected: typ{
				Type: "object",
				Props: []attr{{Name: "stringmap", Val: typ{
					Type:                 "object",
					AdditionalProperties: &additionalPropsType{Type: "integer"},
				}}},
			},
		},
		{
			Name: "map_string_object_direct",
			DSL:  dsls.MapStringKeyObjectDirectBodyDSL(svcName, "map_string_object_direct"),
			Expected: typ{
				Type: "object",
				Props: []attr{{Name: "stringmap", Val: typ{
					Type:                 "object",
					AdditionalProperties: &additionalPropsType{Ref: "#/components/schemas/MapData"},
				}}},
			},
		},
		{
			Name: "map_string_array_object",
			DSL:  dsls.MapStringKeyArrayObjectBodyDSL(svcName, "map_string_array_object"),
			Expected: typ{
				Type: "object",
				Props: []attr{{Name: "stringmap", Val: typ{
					Type: "object",
					AdditionalProperties: &additionalPropsType{
						Type:  "array",
						Items: &additionalPropsType{Ref: "#/components/schemas/MapData"},
					},
				}}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			root := codegen.RunDSL(t, tc.DSL)
			bodies, types := buildBodyTypes(root.API, root.Types, root.ResultTypes)

			svcBodies, ok := bodies[svcName]
			if !ok {
				t.Fatalf("Could not find service %s in bodies", svcName)
			}

			methodBody, ok := svcBodies[tc.Name]
			if !ok {
				t.Fatalf("Could not find method %s in service bodies", tc.Name)
			}

			requestBodyRef := methodBody.RequestBody.Ref
			if requestBodyRef == "" {
				t.Fatal("Expected request body reference")
			}

			requestBodyTypeName := nameFromRef(requestBodyRef)
			requestBodySchema, ok := types[requestBodyTypeName]
			if !ok {
				t.Fatalf("Could not find request body type %s", requestBodyTypeName)
			}

			matchesSchema(t, tc.Name, requestBodySchema, types, tc.Expected)
		})
	}
}

func validateAdditionalProperties(t *testing.T, ctx string, addProps any, types map[string]*openapi.Schema, expected *additionalPropsType, prefix string) {
	if addProps == nil {
		t.Errorf("%s: %sexpected additionalProperties to be set", ctx, prefix)
		return
	}

	schema, ok := addProps.(*openapi.Schema)
	if !ok {
		t.Errorf("%s: %sexpected additionalProperties to be schema, got %T", ctx, prefix, addProps)
		return
	}

	validateAdditionalPropsSchema(t, ctx, schema, types, expected, prefix+"additionalProperties: ")
}

func validateAdditionalPropsSchema(t *testing.T, ctx string, schema *openapi.Schema, types map[string]*openapi.Schema, expected *additionalPropsType, prefix string) {
	if expected.Ref != "" {
		if schema.Ref == "" {
			t.Errorf("%s: %sexpected reference to %s, but got inline schema", ctx, prefix, expected.Ref)
			return
		}
		if schema.Ref != expected.Ref {
			t.Errorf("%s: %sexpected reference %s, got %s", ctx, prefix, expected.Ref, schema.Ref)
		}
		return
	}

	if schema.Ref != "" {
		typeName := nameFromRef(schema.Ref)
		resolvedSchema, ok := types[typeName]
		if !ok {
			t.Errorf("%s: %scould not resolve reference %s", ctx, prefix, schema.Ref)
			return
		}
		schema = resolvedSchema
	}

	if string(schema.Type) != expected.Type {
		t.Errorf("%s: %sexpected type %s, got %s", ctx, prefix, expected.Type, schema.Type)
	}

	if expected.Items != nil {
		if schema.Items == nil {
			t.Errorf("%s: %sexpected array items to be set", ctx, prefix)
		} else {
			validateAdditionalPropsSchema(t, ctx, schema.Items, types, expected.Items, prefix+"items: ")
		}
	}
}

func TestTypesOnlyDifferByEnum(t *testing.T) {
	root := codegen.RunDSL(t, dsls.StringEnumBodyDSL())

	bodies, types := buildBodyTypes(root.API, root.Types, root.ResultTypes)

	svc1, ok := bodies["svc_enum_1"]
	if !ok {
		t.Errorf("bodies does not contain details for service %q", "svc_enum_1")
		return
	}
	svc2, ok := bodies["svc_enum_2"]
	if !ok {
		t.Errorf("bodies does not contain details for service %q", "svc_enum_2")
		return
	}

	svc1MethodRB := svc1["method_enum"].RequestBody.Ref
	svc2MethodRB := svc2["method_enum"].RequestBody.Ref

	if svc1MethodRB == svc2MethodRB {
		t.Errorf("expected different refs, got %q", svc1MethodRB)

		name := nameFromRef(svc1MethodRB)
		derefed := types[name]
		jsoned, _ := json.Marshal(derefed)
		t.Errorf("shared referenced type (%s) was: %v", name, string(jsoned))
		return
	}
}

func TestSchemafierUniquifyUsesStableHashSuffix(t *testing.T) {
	sf := newSchemafier(expr.NewRandom("test"))
	sf.schemas["CreateThreadRequest"] = openapi.NewSchema()
	sf.schemas["CreateThreadRequest2"] = openapi.NewSchema()

	name := sf.uniquify("CreateThreadRequest", "000000001234abcd000000000000000000000000000000000000000000000000")
	if name != "CreateThreadRequest_000000001234abcd" {
		t.Fatalf("got %q, expected deterministic hash suffix", name)
	}

	name = sf.uniquify("CreateThreadRequest2", "0000000000000099000000000000000000000000000000000000000000000000")
	if name != "CreateThreadRequest2_0000000000000099" {
		t.Fatalf("got %q, expected original trailing digits to be preserved", name)
	}

	name = sf.uniquify("FreshName", "000000000000beef000000000000000000000000000000000000000000000000")
	if name != "FreshName" {
		t.Fatalf("got %q, expected unsuffixed fresh name", name)
	}
}

func TestClaimExplicitNamePanicsOnConflictingSchema(t *testing.T) {
	sf := newSchemafier(expr.NewRandom("test"))
	sf.schemaFingerprints["AuthSessionResponseBody"] = "first"

	require.PanicsWithValue(t,
		"openapi: explicit component name \"AuthSessionResponseBody\" is claimed by multiple different schemas; use distinct Meta(\"openapi:typename\", ...) values",
		func() {
			sf.claimExplicitName("AuthSessionResponseBody", "second")
		},
	)
}

func TestFingerprintAttribute(t *testing.T) {
	type (
		testAttr struct {
			name string
			att  *expr.AttributeExpr
		}

		fingerprintBehavior int

		testGroup struct {
			name     string
			attrs    []testAttr
			behavior fingerprintBehavior
		}
	)

	const (
		uniqueFingerprints fingerprintBehavior = iota
		identicalFingerprints
	)

	var (
		metaNotGenerate = expr.MetaExpr{"openapi:generate": []string{"false"}}
		metaEmpty       = expr.MetaExpr{}
	)

	cases := []testGroup{
		{
			name:     "Distinct OpenAPI primitive types",
			behavior: uniqueFingerprints,
			attrs: []testAttr{
				{name: "bool", att: &expr.AttributeExpr{Type: expr.Boolean}},
				{name: "int", att: &expr.AttributeExpr{Type: expr.Int}},
				{name: "int32", att: &expr.AttributeExpr{Type: expr.Int32}},
				{name: "float32", att: &expr.AttributeExpr{Type: expr.Float32}},
				{name: "float64", att: &expr.AttributeExpr{Type: expr.Float64}},
				{name: "string", att: &expr.AttributeExpr{Type: expr.String}},
				{name: "bytes", att: &expr.AttributeExpr{Type: expr.Bytes}},
				{name: "any", att: &expr.AttributeExpr{Type: expr.Any}},
			},
		}, {
			name:     "Equivalent OpenAPI integer types",
			behavior: identicalFingerprints,
			attrs: []testAttr{
				{name: "int", att: &expr.AttributeExpr{Type: expr.Int}},
				{name: "int64", att: &expr.AttributeExpr{Type: expr.Int64}},
				{name: "uint", att: &expr.AttributeExpr{Type: expr.UInt}},
				{name: "uint64", att: &expr.AttributeExpr{Type: expr.UInt64}},
			},
		}, {
			name:     "Equivalent OpenAPI int32 types",
			behavior: identicalFingerprints,
			attrs: []testAttr{
				{name: "int32", att: &expr.AttributeExpr{Type: expr.Int32}},
				{name: "uint32", att: &expr.AttributeExpr{Type: expr.UInt32}},
			},
		}, {
			name:     "Collection types",
			behavior: uniqueFingerprints,
			attrs: []testAttr{
				{name: "array-bool", att: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Boolean}}}},
				{name: "array-int", att: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Int}}}},
				{name: "map-str-int", att: &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.Int}}}},
				{name: "map-str-str", att: &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}}},
			},
		}, {
			name:     "Objects with validation rules",
			behavior: uniqueFingerprints,
			attrs: []testAttr{
				{name: "no-validation", att: newObj("foo", false)},
				{name: "required-validation", att: newObj("foo", true)},
				{name: "pattern-validation", att: &expr.AttributeExpr{
					Type: expr.String,
					Validation: &expr.ValidationExpr{
						Pattern: "^[a-z]+$",
					},
				}},
				{name: "enum-validation", att: &expr.AttributeExpr{
					Type: expr.String,
					Validation: &expr.ValidationExpr{
						Values: []any{"foo", "bar"},
					},
				}},
			},
		}, {
			name:     "Result types with different views",
			behavior: uniqueFingerprints,
			attrs: []testAttr{
				{name: "no-view", att: newRT("id", newObj("foo", true))},
				{name: "default-view", att: newRTWithView("id", newObj("foo", true), "default")},
				{name: "tiny-view", att: newRTWithView("id", newObj("foo", true), "tiny")},
			},
		}, {
			name:     "Objects with openapi:generate:false metadata",
			behavior: identicalFingerprints,
			attrs: []testAttr{
				{name: "obj-with-skipped-field", att: newObj2Meta("foo", "bar", expr.String, expr.String, metaEmpty, metaNotGenerate)},
				{name: "obj-without-skipped-field", att: newObj("foo", false)},
			},
		}, {
			name:     "Complex map types",
			behavior: uniqueFingerprints,
			attrs: []testAttr{
				{name: "map-int-array", att: &expr.AttributeExpr{Type: &expr.Map{
					KeyType:  &expr.AttributeExpr{Type: expr.Int},
					ElemType: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}},
				}}},
				{name: "map-array-int", att: &expr.AttributeExpr{Type: &expr.Map{
					KeyType:  &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}},
					ElemType: &expr.AttributeExpr{Type: expr.Int},
				}}},
			},
		}, {
			name:     "Nested user types",
			behavior: uniqueFingerprints,
			attrs: []testAttr{
				{name: "single-nest", att: newUserType("foo", newObj("bar", false))},
				{name: "double-nest", att: newUserType("foo", newUserType("bar", newObj("baz", false)))},
			},
		}, {
			name:     "Recursive types",
			behavior: identicalFingerprints,
			attrs: []testAttr{
				{name: "recursive-1", att: newRecursiveType("foo")},
				{name: "recursive-2", att: newRecursiveType("foo")},
			},
		},
	}

	sf := newSchemafier(expr.NewRandom("test"))

	for _, group := range cases {
		t.Run(group.name, func(t *testing.T) {
			seen := make(map[string][]string)

			for _, attr := range group.attrs {
				fingerprint := sf.fingerprintAttribute(attr.att)
				seen[fingerprint] = append(seen[fingerprint], attr.name)
			}

			switch group.behavior {
			case uniqueFingerprints:
				for fingerprint, names := range seen {
					if len(names) > 1 {
						t.Errorf("expected unique fingerprints but got collision between %v (fingerprint: %s)",
							names, fingerprint)
					}
				}
			case identicalFingerprints:
				if len(seen) > 1 {
					t.Errorf("expected identical fingerprints but got different ones: %v", seen)
				}
			}
		})
	}
}

func newObj(n string, req bool) *expr.AttributeExpr {
	attr := &expr.AttributeExpr{
		Type:       &expr.Object{{Name: n, Attribute: &expr.AttributeExpr{Type: expr.String}}},
		Validation: &expr.ValidationExpr{},
	}
	if req {
		attr.Validation.Required = []string{n}
	}
	return attr
}

func newObj2Meta(n, o string, t, u expr.DataType, l, m expr.MetaExpr, reqs ...string) *expr.AttributeExpr {
	attr := &expr.AttributeExpr{
		Type: &expr.Object{
			{Name: n, Attribute: &expr.AttributeExpr{Type: t, Meta: l}},
			{Name: o, Attribute: &expr.AttributeExpr{Type: u, Meta: m}},
		},
		Validation: &expr.ValidationExpr{},
	}
	attr.Validation.Required = append(attr.Validation.Required, reqs...)
	return attr
}

func newRT(id string, att *expr.AttributeExpr) *expr.AttributeExpr {
	return &expr.AttributeExpr{
		Type: &expr.ResultTypeExpr{
			Identifier: id,
			UserTypeExpr: &expr.UserTypeExpr{
				AttributeExpr: att,
			},
		},
	}
}

func newRTWithView(id string, att *expr.AttributeExpr, view string) *expr.AttributeExpr {
	rt := newRT(id, att)
	rt.Type.(*expr.ResultTypeExpr).Meta = expr.MetaExpr{
		expr.ViewMetaKey: []string{view},
	}
	return rt
}

func newUserType(name string, att *expr.AttributeExpr) *expr.AttributeExpr {
	return &expr.AttributeExpr{
		Type: &expr.UserTypeExpr{
			AttributeExpr: att,
			TypeName:      name,
		},
	}
}

func newRecursiveType(name string) *expr.AttributeExpr {
	ut := &expr.UserTypeExpr{
		TypeName: name,
	}
	att := &expr.AttributeExpr{
		Type: &expr.Object{
			&expr.NamedAttributeExpr{
				Name: "self",
				Attribute: &expr.AttributeExpr{
					Type: ut,
				},
			},
		},
	}
	ut.AttributeExpr = att
	return &expr.AttributeExpr{Type: ut}
}

func nameFromRef(ref string) string {
	elems := strings.Split(ref, "/")
	return elems[len(elems)-1]
}

func countEnvelopeSchemas(types map[string]*openapi.Schema) []string {
	var names []string
	for name := range types {
		if strings.Contains(name, "Envelope") {
			names = append(names, name)
		}
	}
	return names
}
