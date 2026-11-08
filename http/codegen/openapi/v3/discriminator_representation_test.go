package openapiv3_test

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/internal/byteschema"
	"github.com/stretchr/testify/require"
)

// TestDiscriminatorParentHintRepresentation covers a hinted parent whose JSON
// body projects byte fields. The emitted parent is that JSON representation,
// so its children, mapping and default targets are published in the same
// representation and no raw duplicate of the parent is emitted.
func TestDiscriminatorParentHintRepresentation(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		mapping, fallback        string
		refined, childFirst      bool
		wantMapping, wantDefault string
	}{
		{name: "implicit child"},
		{name: "child declared first", childFirst: true},
		{name: "refined child", refined: true},
		{name: "explicit mapping", mapping: "child=Child", wantMapping: "#/components/schemas/Child"},
		{name: "default mapping", fallback: "Child", wantDefault: "#/components/schemas/Child"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			schemas := representationSchemas(t, representationParentDSL(true, tc.mapping, tc.fallback, tc.refined, tc.childFirst))
			require.Equal(t, []string{"Child", "Parent"}, slices.Sorted(maps.Keys(schemas)), "no raw duplicate representation")
			parent := requireMap(t, schemas["Parent"], "parent")
			requireBase64Field(t, parent, "blob")
			hint := requireMap(t, parent["discriminator"], "parent hint")
			if tc.wantMapping != "" {
				require.Equal(t, map[string]any{"child": tc.wantMapping}, hint["mapping"])
			} else {
				require.NotContains(t, hint, "mapping")
			}
			if tc.wantDefault != "" {
				require.Equal(t, tc.wantDefault, hint["defaultMapping"])
			}
			child := requireMap(t, schemas["Child"], "child")
			allOf := requireSlice(t, child["allOf"], "child allOf")
			require.Equal(t, "#/components/schemas/Parent", requireMap(t, allOf[0], "child base")["$ref"], "child extends the emitted parent")
			member := requireMap(t, allOf[1], "child member")
			requireBase64Field(t, member, "blob")
			parentBlob := requireMap(t, requireMap(t, parent["properties"], "properties")["blob"], "parent blob")
			childBlob := requireMap(t, requireMap(t, member["properties"], "properties")["blob"], "child blob")
			if tc.refined {
				require.NotEqual(t, parentBlob["anyOf"], childBlob["anyOf"], "the child's own member carries its refinement")
			} else {
				require.Equal(t, parentBlob["anyOf"], childBlob["anyOf"])
			}
		})
	}
}

// TestDiscriminatorParentHintPlainRepresentation is the ordinary control: a
// parent without byte content has one representation.
func TestDiscriminatorParentHintPlainRepresentation(t *testing.T) {
	schemas := representationSchemas(t, representationParentDSL(false, "", "Child", false, false))
	require.Equal(t, []string{"Child", "Parent"}, slices.Sorted(maps.Keys(schemas)))
	child := requireMap(t, schemas["Child"], "child")
	require.Equal(t, "#/components/schemas/Parent", requireMap(t, requireSlice(t, child["allOf"], "child allOf")[0], "child base")["$ref"])
	require.Equal(t, "#/components/schemas/Child", requireMap(t, requireMap(t, schemas["Parent"], "parent")["discriminator"], "hint")["defaultMapping"])
}

// TestDiscriminatorParentHintDistinctRepresentations publishes the same hinted
// parent through a JSON body and a form body that is the parent itself, whose
// fields are form fields. Each emitted parent representation retains a child
// in that representation, and every child extends exactly the parent
// representation whose byte encoding it inherits. A parent-typed form field
// would be JSON content and share the JSON representation instead.
func TestDiscriminatorParentHintDistinctRepresentations(t *testing.T) {
	for _, childUsed := range []bool{false, true} {
		t.Run(fmt.Sprintf("child used %v", childUsed), func(t *testing.T) {
			requireDistinctRepresentations(t, childUsed)
		})
	}
}

func requireDistinctRepresentations(t *testing.T, childUsed bool) {
	t.Helper()
	schemas := representationSchemas(t, func() {
		representationParentDSL(true, "", "", false, false)()
		Service("forms", func() {
			Method("submit", func() {
				Payload(expr.Root.UserType("Parent"))
				HTTP(func() {
					POST("/form")
					FormRequest()
				})
			})
			if childUsed {
				// The child's own JSON occurrence must not discard the raw child
				// that the form representation of the parent publishes.
				Method("child", func() {
					Result(expr.Root.UserType("Child"))
					HTTP(func() {
						GET("/child")
					})
				})
			}
		})
	})
	parents := map[string]string{}
	children := map[string]string{}
	for name, value := range schemas {
		schema := requireMap(t, value, name)
		if allOf, ok := schema["allOf"].([]any); ok {
			children[name] = requireString(t, requireMap(t, allOf[0], "child base")["$ref"], "child base ref")
			continue
		}
		if _, hinted := schema["discriminator"]; hinted {
			parents["#/components/schemas/"+name] = name
		}
	}
	require.Len(t, parents, 2, "JSON and form parent representations")
	require.Len(t, children, 2, "one child per parent representation")
	encodings := map[string]bool{}
	for child, base := range children {
		parent, published := parents[base]
		require.True(t, published, "child %s extends a published parent", child)
		member := requireMap(t, requireSlice(t, requireMap(t, schemas[child], child)["allOf"], "allOf")[1], "member")
		inherited := requireMap(t, requireMap(t, member["properties"], "properties")["blob"], "blob")
		own := requireMap(t, requireMap(t, requireMap(t, schemas[parent], parent)["properties"], "properties")["blob"], "blob")
		require.Equal(t, own["contentEncoding"], inherited["contentEncoding"], "child %s inherits its parent's representation", child)
		encodings[fmt.Sprint(own["contentEncoding"])] = true
	}
	require.Len(t, encodings, 2, "the representations differ")
}

// TestDiscriminatorPublicationCodecContext publishes a hinted parent without
// byte fields through a JSON body. The JSON codec context still reaches the
// children, mapping and default targets the parent publishes, so their byte
// fields use the JSON base64 representation with projected decoded bounds.
func TestDiscriminatorPublicationCodecContext(t *testing.T) {
	for _, tc := range []struct {
		name, mapping, fallback string
		target                  string
	}{
		{name: "implicit child", target: "Child"},
		{name: "explicit mapping", mapping: "sample=Sample", target: "Sample"},
		{name: "default mapping", fallback: "Fallback", target: "Fallback"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			schemas := representationSchemas(t, publicationContextDSL(tc.mapping, tc.fallback, false))
			names := slices.Sorted(maps.Keys(schemas))
			require.Contains(t, names, tc.target)
			for _, name := range names {
				require.NotContains(t, name, "_", "no second representation in a JSON-only document")
			}
			parent := requireMap(t, schemas["Parent"], "parent")
			hint := requireMap(t, parent["discriminator"], "hint")
			if tc.mapping != "" {
				require.Equal(t, map[string]any{"sample": "#/components/schemas/Sample"}, hint["mapping"], "authored keys only")
			} else {
				require.NotContains(t, hint, "mapping")
			}
			if tc.fallback != "" {
				require.Equal(t, "#/components/schemas/Fallback", hint["defaultMapping"])
			}
			child := requireMap(t, schemas["Child"], "child")
			require.Equal(t, "#/components/schemas/Parent", requireMap(t, requireSlice(t, child["allOf"], "child allOf")[0], "child base")["$ref"])
			target := requireMap(t, schemas[tc.target], tc.target)
			if allOf, ok := target["allOf"].([]any); ok {
				target = requireMap(t, allOf[len(allOf)-1], "member")
			}
			requireDecodedBytesField(t, target, "thumb")
		})
	}
}

// TestDiscriminatorPublicationMixedCodecContext publishes the byte-free
// parent through a JSON body and a form body that is the parent itself. The
// JSON representation publishes the JSON child and the form representation
// the raw child; the dependency closure keeps the two parent representations
// apart.
func TestDiscriminatorPublicationMixedCodecContext(t *testing.T) {
	schemas := representationSchemas(t, publicationContextDSL("", "", true))
	parents := map[string]string{}
	children := map[string]string{}
	for name, value := range schemas {
		schema := requireMap(t, value, name)
		if allOf, ok := schema["allOf"].([]any); ok {
			children[name] = requireString(t, requireMap(t, allOf[0], "child base")["$ref"], "child base ref")
			continue
		}
		if _, hinted := schema["discriminator"]; hinted {
			parents["#/components/schemas/"+name] = name
		}
	}
	require.Len(t, parents, 2, "JSON and form parent representations")
	require.Len(t, children, 2, "one child per parent representation")
	require.Contains(t, children, "Child")
	require.Equal(t, "#/components/schemas/Parent", children["Child"], "the public parent publishes the JSON child")
	requireDecodedBytesField(t, requireMap(t, requireSlice(t, requireMap(t, schemas["Child"], "child")["allOf"], "allOf")[1], "member"), "thumb")
	for child, base := range children {
		require.Contains(t, parents, base, "child %s extends a published parent", child)
		if child == "Child" {
			continue
		}
		member := requireMap(t, requireSlice(t, requireMap(t, schemas[child], child)["allOf"], "allOf")[1], "member")
		thumb := requireMap(t, requireMap(t, member["properties"], "properties")["thumb"], "thumb")
		require.Equal(t, "binary", thumb["format"], "the form representation keeps raw bytes")
	}
}

func publicationContextDSL(mapping, fallback string, form bool) func() {
	return func() {
		API("publication context", func() {
			Meta("openapi:example", "false")
		})
		thumb := func() {
			Attribute("thumb", Bytes, func() {
				MinLength(3)
			})
		}
		parent := Type("Parent", func() {
			Meta("openapi:discriminator:propertyName", "kind")
			if mapping != "" {
				Meta("openapi:discriminator:mapping", mapping)
			}
			if fallback != "" {
				Meta("openapi:discriminator:defaultMapping", fallback)
			}
			Attribute("kind", String)
			Required("kind")
		})
		Type("Child", func() {
			Extend(parent)
			Meta("openapi:allOf:reference", "true")
			Attribute("name", String)
			if mapping == "" && fallback == "" {
				thumb()
			}
		})
		// A flat extension is published only through its explicit mapping.
		Type("Sample", func() {
			Extend(parent)
			thumb()
		})
		Type("Fallback", func() {
			Attribute("kind", String)
			thumb()
		})
		Service("parents", func() {
			Method("show", func() {
				Payload(parent)
				Result(parent)
				HTTP(func() {
					POST("/parent")
				})
			})
			if form {
				Method("submit", func() {
					Payload(parent)
					HTTP(func() {
						POST("/form")
						FormRequest()
					})
				})
			}
		})
	}
}

// TestDiscriminatorUnionHintFallbackCodecContext keeps a union hint's separate
// byte-bearing fallback in the JSON representation even though no branch has
// byte fields.
func TestDiscriminatorUnionHintFallbackCodecContext(t *testing.T) {
	root := httpgen.RunHTTPDSL(t, func() {
		API("union fallback", func() {
			Meta("openapi:example", "false")
		})
		cat := Type("Cat", func() {
			Attribute("kind", String)
			Required("kind")
		})
		dog := Type("Dog", func() {
			Attribute("kind", String)
			Required("kind")
		})
		Type("Fallback", func() {
			Attribute("kind", String)
			Attribute("thumb", Bytes, func() {
				MinLength(3)
			})
		})
		choice := Type("Choice", OneOf(func() {
			Untagged()
			Attribute("Cat", cat, func() {
				Meta("openapi:discriminator:mapping", "cat")
			})
			Attribute("Dog", dog)
		}), func() {
			Meta("openapi:discriminator:propertyName", "kind")
			Meta("openapi:discriminator:defaultMapping", "Fallback")
		})
		Service("choices", func() {
			Method("show", func() {
				Result(choice)
				HTTP(func() {
					GET("/choice")
				})
			})
		})
	})
	spec := representationSpec(t, root)
	schemas := requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
	require.Equal(t, []string{"Cat", "Dog", "Fallback"}, slices.Sorted(maps.Keys(schemas)))
	requireDecodedBytesField(t, requireMap(t, schemas["Fallback"], "fallback"), "thumb")
	media := requireResponseMediaType(t, requireOperation(t, spec, "/choice", "get"), "application/json")
	hint := requireMap(t, resolveCardinalitySchema(t, spec, requireMap(t, media["schema"], "choice"))["discriminator"], "hint")
	require.Equal(t, map[string]any{"cat": "#/components/schemas/Cat"}, hint["mapping"])
	require.Equal(t, "#/components/schemas/Fallback", hint["defaultMapping"])
}

// TestDiscriminatorUnionHintRepresentation keeps alias and default references
// of a projected union hint on their projected branch, including an alias that
// shadows another branch's name.
func TestDiscriminatorUnionHintRepresentation(t *testing.T) {
	root := httpgen.RunHTTPDSL(t, func() {
		API("union", func() {
			Meta("openapi:example", "false")
		})
		cat := Type("Cat", func() {
			Attribute("kind", String)
			Attribute("photo", Bytes, func() {
				MinLength(1)
			})
			Required("kind")
		})
		dog := Type("Dog", func() {
			Attribute("kind", String)
			Required("kind")
		})
		choice := Type("Choice", OneOf(func() {
			Untagged()
			Attribute("Cat", cat, func() {
				Meta("openapi:discriminator:mapping", "cat", "kitty")
			})
			Attribute("Dog", dog, func() {
				Meta("openapi:discriminator:mapping", "Cat")
			})
		}), func() {
			Meta("openapi:discriminator:propertyName", "kind")
			Meta("openapi:discriminator:defaultMapping", "Cat")
		})
		Service("choices", func() {
			Method("show", func() {
				Result(choice)
				HTTP(func() {
					GET("/choice")
				})
			})
		})
	})
	spec := representationSpec(t, root)
	schemas := requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
	require.Equal(t, []string{"Cat", "Dog"}, slices.Sorted(maps.Keys(schemas)), "no raw duplicate branch")
	requireBase64Field(t, requireMap(t, schemas["Cat"], "cat"), "photo")
	media := requireResponseMediaType(t, requireOperation(t, spec, "/choice", "get"), "application/json")
	choice := resolveCardinalitySchema(t, spec, requireMap(t, media["schema"], "choice"))
	hint := requireMap(t, choice["discriminator"], "hint")
	require.Equal(t, map[string]any{"cat": "#/components/schemas/Cat", "kitty": "#/components/schemas/Cat", "Cat": "#/components/schemas/Dog"}, hint["mapping"])
	require.Equal(t, "#/components/schemas/Cat", hint["defaultMapping"])
}

func representationParentDSL(bytes bool, mapping, fallback string, refined, childFirst bool) func() {
	return func() {
		API("representation", func() {
			Meta("openapi:example", "false")
		})
		var parent expr.UserType
		child := func() {
			Type("Child", func() {
				Extend(parent)
				Meta("openapi:allOf:reference", "true")
				Attribute("name", String)
				if refined {
					Attribute("blob", Bytes, func() {
						MinLength(4)
					})
				}
			})
		}
		if childFirst {
			child()
		}
		parent = Type("Parent", func() {
			Meta("openapi:discriminator:propertyName", "kind")
			if mapping != "" {
				Meta("openapi:discriminator:mapping", mapping)
			}
			if fallback != "" {
				Meta("openapi:discriminator:defaultMapping", fallback)
			}
			Attribute("kind", String)
			if bytes {
				Attribute("blob", Bytes, func() {
					MinLength(2)
				})
			}
			Required("kind")
		})
		if !childFirst {
			child()
		}
		Service("parents", func() {
			Method("show", func() {
				Payload(parent)
				Result(parent)
				HTTP(func() {
					POST("/parent")
				})
			})
		})
	}
}

func representationSchemas(t *testing.T, design func()) map[string]any {
	t.Helper()
	spec := representationSpec(t, httpgen.RunHTTPDSL(t, design))
	return requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
}

func representationSpec(t *testing.T, root *expr.RootExpr) map[string]any {
	t.Helper()
	var spec map[string]any
	for _, target := range []string{"3.1", "3.2"} {
		root.API.Meta["openapi:version"] = []string{target}
		files, err := openapiv3.Files(root)
		require.NoError(t, err)
		for _, file := range files {
			buffer := renderSection(t, file.AllSections()[0])
			raw := buffer.Bytes()
			if target == "3.1" {
				validateOpenAPIVersion(t, raw, "3.1.1")
			} else {
				validateOpenAPIVersion(t, raw, "3.2.0")
			}
			if target == "3.2" && file.Path[len(file.Path)-5:] == ".json" {
				spec = decodeOpenAPIJSON(t, raw)
			}
		}
	}
	require.NotNil(t, spec)
	return spec
}

func requireBase64Field(t *testing.T, object map[string]any, name string) {
	t.Helper()
	field := requireMap(t, requireMap(t, object["properties"], "properties")[name], name)
	require.Equal(t, "base64", field["contentEncoding"], name)
	require.NotContains(t, field, "format", name)
}

// requireDecodedBytesField checks the JSON representation of a Bytes field:
// base64 content and the encoded-length gate projected from its decoded
// minimum.
func requireDecodedBytesField(t *testing.T, object map[string]any, name string) {
	t.Helper()
	requireBase64Field(t, object, name)
	field := requireMap(t, requireMap(t, object["properties"], "properties")[name], name)
	minimum := 3
	constraint, err := byteschema.Project(&minimum, nil)
	require.NoError(t, err)
	branches := requireSlice(t, field["anyOf"], name+" gate")
	require.Len(t, branches, len(constraint.Branches))
	for index, expected := range constraint.Branches {
		branch := requireMap(t, branches[index], "gate branch")
		require.Equal(t, expected.Pattern, branch["pattern"])
		require.Equal(t, float64(expected.MinLength), branch["minLength"])
	}
}
