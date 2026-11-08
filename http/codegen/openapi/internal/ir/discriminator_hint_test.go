package ir

import (
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestDiscriminatorHintExport(t *testing.T) {
	const whitespaceAlias = " cat " // Mapping keys are exact authored strings.
	for _, tc := range []struct {
		name                        string
		tagged, optional, duplicate bool
		aliases                     [][]string
		meta                        expr.MetaExpr
		want                        map[string]string
		errorText                   string
	}{
		{name: "explicit keys only", aliases: [][]string{{"cat", "kitty"}, {"dog"}}, want: map[string]string{"cat": "#/components/schemas/Cat", "kitty": "#/components/schemas/Cat", "dog": "#/components/schemas/Dog"}},
		{name: "no derived keys", want: map[string]string{}},
		{name: "arbitrary explicit keys", aliases: [][]string{{"", "cat=old", whitespaceAlias}, nil}, want: map[string]string{"": "#/components/schemas/Cat", "cat=old": "#/components/schemas/Cat", whitespaceAlias: "#/components/schemas/Cat"}},
		{name: "key shadows other branch name", aliases: [][]string{nil, {"Cat"}}, want: map[string]string{"Cat": "#/components/schemas/Dog"}},
		{name: "duplicate alternative retained", duplicate: true, aliases: [][]string{{"cat"}, nil}, want: map[string]string{"cat": "#/components/schemas/Cat"}},
		{name: "duplicate explicit key", aliases: [][]string{{"same"}, {"same"}}, errorText: "duplicate discriminator mapping key"},
		{name: "tagged alias", tagged: true, aliases: [][]string{{"kitty"}, nil}, want: map[string]string{"Cat": "#/components/schemas/ChoiceCatEnvelope", "Dog": "#/components/schemas/ChoiceDogEnvelope", "kitty": "#/components/schemas/ChoiceCatEnvelope"}},
		{name: "tagged alias conflicts with tag", tagged: true, aliases: [][]string{{"Dog"}, nil}, errorText: "conflicts with branch tag"},
		{name: "propertyName on tagged", tagged: true, meta: expr.MetaExpr{"openapi:discriminator:propertyName": {"kind"}}, errorText: "propertyName"},
		{name: "hint optional metadata rejected", meta: expr.MetaExpr{"openapi:discriminator:optional": {"false"}}, errorText: "derived"},
		{name: "optional hint needs fallback", optional: true, errorText: "defaultMapping"},
		{name: "optional hint with fallback", optional: true, meta: expr.MetaExpr{"openapi:discriminator:defaultMapping": {"Cat"}}, want: map[string]string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				dsl.API("hint", func() {})
				cat := dsl.Type("Cat", func() {
					dsl.Attribute("kind", dsl.String, func() {
						dsl.Enum("cat")
					})
					dsl.Required("kind")
				})
				dog := dsl.Type("Dog", func() {
					dsl.Attribute("kind", dsl.String, func() {
						dsl.Enum("dog")
					})
					if !tc.optional {
						dsl.Required("kind")
					}
				})
				dsl.Type("Choice", dsl.OneOf(func() {
					if !tc.tagged {
						dsl.Untagged()
					}
					for index, branch := range []struct {
						name string
						typ  expr.UserType
					}{{"Cat", cat}, {"Dog", dog}} {
						dsl.Attribute(branch.name, branch.typ, func() {
							if index < len(tc.aliases) && len(tc.aliases[index]) > 0 {
								dsl.Meta("openapi:discriminator:mapping", tc.aliases[index]...)
							}
						})
					}
					if tc.duplicate {
						dsl.Attribute("CatAgain", cat)
					}
				}), func() {
					dsl.Meta("type:generate:force")
					if !tc.tagged {
						dsl.Meta("openapi:discriminator:propertyName", "kind")
					}
					for key, values := range tc.meta {
						dsl.Meta(key, values...)
					}
				})
			})
			bodies := BuildBodyTypes(root.API, root.Types, root.ResultTypes, WithExampleValue(nil))
			if tc.errorText != "" {
				require.ErrorContains(t, bodies.Err, tc.errorText)
				return
			}
			require.NoError(t, bodies.Err)
			schema := bodies.Components["Choice"]
			require.NotNil(t, schema)
			require.NotNil(t, schema.Discriminator)
			if tc.tagged {
				require.Equal(t, "type", schema.Discriminator.PropertyName)
			} else {
				require.Equal(t, "kind", schema.Discriminator.PropertyName)
				require.Empty(t, schema.Required)
			}
			require.Equal(t, tc.want, schema.Discriminator.Mapping)
			require.Equal(t, tc.optional, schema.Discriminator.Optional)
			branches := 2
			if tc.duplicate {
				branches++
			}
			require.Len(t, schema.OneOf, branches)
		})
	}
}

func TestDiscriminatorParentHintExport(t *testing.T) {
	for _, tc := range []struct{ name, mapping, errorText string }{
		{"child", "cat=Cat", ""},
		{"implicit child", "", ""},
		{"empty key", "=Cat", ""},
		{"delimiter in key", "cat=old=Cat", ""},
		{"whitespace key", " cat =Cat", ""},
		{"non child", "other=Other", "does not extend"},
		{"unknown child", "missing=Missing", "does not name"},
		{"malformed", "Cat", "key=Type"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				dsl.API("parent", func() {})
				parent := dsl.Type("Pet", func() {
					dsl.Attribute("kind", dsl.String)
					dsl.Required("kind")
					dsl.Meta("openapi:discriminator:propertyName", "kind")
					if tc.mapping != "" {
						dsl.Meta("openapi:discriminator:mapping", tc.mapping)
					}
				})
				dsl.Type("Cat", func() {
					dsl.Extend(parent)
					dsl.Meta("openapi:allOf:reference", "true")
					dsl.Attribute("name", dsl.String)
				})
				dsl.Service("parents", func() {
					dsl.Method("Show", func() {
						dsl.Payload(parent)
						dsl.HTTP(func() {
							dsl.POST("/parent")
						})
					})
				})
				dsl.Type("Other", func() {
					dsl.Attribute("kind", dsl.String)
				})
			})
			bodies := BuildBodyTypes(root.API, root.Types, root.ResultTypes, WithExampleValue(nil))
			if tc.errorText != "" {
				require.ErrorContains(t, bodies.Err, tc.errorText)
				return
			}
			require.NoError(t, bodies.Err)
			a, selected := annotationBodies(t, root, true, WithExampleValue(nil))
			endpoint := selected.Services["parents"]["Show"]
			family := capturePreferredFamily(t, a, endpoint.RequestBody, endpoint.prepared.Request.BodyValue)
			_, parent := annotationComponent(t, family.root, a.schemas)
			require.NotNil(t, parent.Discriminator)
			if tc.mapping == "" {
				require.Empty(t, parent.Discriminator.Mapping)
			} else {
				key := tc.mapping[:len(tc.mapping)-len("=Cat")]
				require.Equal(t, map[string]string{key: family.childRef}, parent.Discriminator.Mapping, "explicit keys select this occurrence's contributor")
			}
			owners := snapshotAnnotationOwners(a)
			a.finalizeRepresentations()
			require.NoError(t, a.Err())
			require.Equal(t, owners, snapshotAnnotationOwners(a))
			child := assertPreferredFamily(t, a, family)
			require.Contains(t, child.AllOf[1].Properties, "name", "hint-only target retains its own member")
			_, parent = annotationComponent(t, family.root, a.schemas)
			require.Equal(t, []string{"kind"}, parent.Required)
			require.Equal(t, a.schemas, bodies.Components, "public body analysis preserves the selected bindings")
			require.Equal(t, a.discriminatorChildren, bodies.DiscriminatorChildren)
		})
	}
}

func TestDiscriminatorHintFingerprint(t *testing.T) {
	base := &expr.AttributeExpr{Type: expr.String}
	object := &expr.Object{{Name: "kind", Attribute: base}}
	first := &expr.AttributeExpr{Type: object, Meta: expr.MetaExpr{"openapi:discriminator:propertyName": {"kind"}, "openapi:discriminator:mapping": {"cat=Cat"}}}
	for _, tc := range []struct{ key, value string }{
		{"openapi:discriminator:propertyName", "category"},
		{"openapi:discriminator:mapping", "dog=Dog"},
		{"openapi:discriminator:extension:x-audit", `{"v":1}`},
		{"openapi:discriminator:defaultMapping", "Fallback"},
	} {
		t.Run(tc.key, func(t *testing.T) {
			second := expr.DupAtt(first)
			second.Meta[tc.key] = []string{tc.value}
			require.NotEqual(t, fingerprintAttribute(first, false), fingerprintAttribute(second, false))
		})
	}
	union := &expr.AttributeExpr{Type: &expr.Union{TypeName: "Choice", Untagged: true, Values: []*expr.NamedAttributeExpr{{Name: "value", Attribute: base}}}}
	aliased := expr.DupAtt(union)
	expr.AsUnion(aliased.Type).Values[0].Attribute.Meta = expr.MetaExpr{"openapi:discriminator:mapping": {"alias"}}
	require.NotEqual(t, fingerprintAttribute(union, false), fingerprintAttribute(aliased, false))
	// A mapping belongs to its enclosing union, not to the payload component.
	require.Equal(t, fingerprintAttribute(base, false), fingerprintAttribute(expr.AsUnion(aliased.Type).Values[0].Attribute, false))
}

func TestDiscriminatorEnvelopeDescriptionPresence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values []string
		want   string
	}{
		{"synthesized", nil, `Synthetic wrapper for union variant "created".`},
		{"suppressed", []string{""}, ""},
		{"authored whitespace", []string{"  description  "}, "  description  "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			branch := &expr.NamedAttributeExpr{Name: "created", Attribute: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{}}}
			if tc.values != nil {
				branch.Attribute.Meta["openapi:description:unionEnvelope"] = tc.values
			}
			require.Equal(t, tc.want, syntheticUnionBranchSchemaDescription(branch))
		})
	}
}

func TestDiscriminatorParentHintHTTPBody(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.API("parent", func() {})
		parent := dsl.Type("Pet", func() {
			dsl.Attribute("kind", dsl.String)
			dsl.Required("kind")
			dsl.Meta("openapi:discriminator:propertyName", "kind")
			dsl.Meta("openapi:discriminator:mapping", "cat=Cat")
		})
		dsl.Type("Cat", func() {
			dsl.Extend(parent)
			dsl.Meta("openapi:allOf:reference", "true")
			dsl.Attribute("name", dsl.String)
		})
		dsl.Service("pets", func() {
			dsl.Method("Show", func() {
				dsl.Payload(parent)
				dsl.Result(parent)
				dsl.HTTP(func() {
					dsl.POST("/Show")
					dsl.Response(dsl.StatusOK)
				})
			})
		})
	})
	document, err := BuildDocument(root.API, root.Types, root.ResultTypes)
	require.NoError(t, err)
	body := hintRequestBody(t, document, "Show").Content["application/json"]
	parent := resolvedHintSchema(t, document, body.Schema)
	require.NotNil(t, parent.Discriminator)
	require.Equal(t, "kind", parent.Discriminator.PropertyName)
	require.Contains(t, parent.Discriminator.Mapping, "cat")
}
