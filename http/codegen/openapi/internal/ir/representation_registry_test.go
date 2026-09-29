package ir

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestRepresentationRegistryReservesPublicNames(t *testing.T) {
	a := NewAnalyzer(nil, false)
	a.schemas["json"] = &Schema{Type: "string", ContentEncoding: "base64"}
	a.schemas["raw"] = &Schema{Type: "string", Format: "binary"}
	a.schemaNames["json"] = representationComponentName{desired: "Blob", logical: "blob", explicit: true, json: true}
	a.schemaNames["raw"] = representationComponentName{desired: "Blob", logical: "blob", explicit: true}
	reserved := "Blob_" + completeSchemaFingerprints(a.schemas)["raw"][:16]
	a.schemas[reserved] = &Schema{Type: "integer"}
	a.schemaNames[reserved] = representationComponentName{desired: reserved, logical: "authored", explicit: true}
	jsonRoot, rawRoot := &Schema{Ref: toRef("json")}, &Schema{Ref: toRef("raw")}
	a.occurrences = []schemaOccurrenceAnalysis{{jsonRoot, a}, {rawRoot, a}}
	a.finalizeRepresentations()
	require.Equal(t, toRef("Blob"), jsonRoot.Ref)
	require.Equal(t, toRef(reserved+"_2"), rawRoot.Ref)
	require.Equal(t, "integer", a.schemas[reserved].Type)
}

func TestRepresentationRegistrySharesRecursiveEquivalentSchemas(t *testing.T) {
	a := NewAnalyzer(nil, false)
	for _, name := range []string{"first", "second"} {
		a.schemas[name] = &Schema{Type: "object", Properties: map[string]*Schema{"next": {Ref: toRef(name)}, "data": {Type: "string", ContentEncoding: "base64"}}}
		a.schemaNames[name] = representationComponentName{desired: "Node", logical: "node", explicit: true, json: true}
	}
	first, second := &Schema{Ref: toRef("first")}, &Schema{Ref: toRef("second")}
	a.occurrences = []schemaOccurrenceAnalysis{{first, a}, {second, a}}
	a.finalizeRepresentations()
	require.Len(t, a.schemas, 1)
	require.Equal(t, toRef("Node"), first.Ref)
	require.Equal(t, first.Ref, second.Ref)
	require.Equal(t, first.Ref, a.schemas["Node"].Properties["next"].Ref)
}

func TestRepresentationRegistrySharesEquivalentRecursiveUnfoldings(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(strconv.FormatBool(reverse), func(t *testing.T) {
			a := NewAnalyzer(nil, false)
			edges := []struct {
				name string
				next string
			}{
				{"self", "self"},
				{"prefix", "self"},
				{"mutualFirst", "mutualSecond"},
				{"mutualSecond", "mutualFirst"},
			}
			if reverse {
				for i, j := 0, len(edges)-1; i < j; i, j = i+1, j-1 {
					edges[i], edges[j] = edges[j], edges[i]
				}
			}
			for _, edge := range edges {
				a.schemas[edge.name] = &Schema{Type: "object", Properties: map[string]*Schema{
					"next": {Ref: toRef(edge.next)},
					"data": {Type: "string", ContentEncoding: "base64"},
				}}
				a.schemaNames[edge.name] = representationComponentName{desired: "Node", logical: "node", explicit: true, json: true}
				a.occurrences = append(a.occurrences, schemaOccurrenceAnalysis{&Schema{Ref: toRef(edge.name)}, a})
			}
			a.finalizeRepresentations()
			require.Len(t, a.schemas, 1, "finite graph unfolding must not create a representation conflict")
			require.NotNil(t, a.schemas["Node"])
			require.Equal(t, toRef("Node"), a.schemas["Node"].Properties["next"].Ref)
			for _, occurrence := range a.occurrences {
				require.Equal(t, toRef("Node"), occurrence.root.Ref)
			}
		})
	}
}

func TestRepresentationRegistryKeepsRecursiveContractDifferences(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Schema)
	}{
		{"constraint", func(schema *Schema) {
			maximum := 2
			schema.Properties["data"].MaxLength = &maximum
		}},
		{"annotation", func(schema *Schema) {
			schema.Properties["data"].Description = "An independently authored annotation."
		}},
		{"example", func(schema *Schema) {
			schema.Example = map[string]any{"data": "aGk="}
		}},
		{"external_reference", func(schema *Schema) {
			schema.Properties["data"].Ref = "https://example.com/custom.json"
		}},
		{"codec_metadata", func(schema *Schema) {
			schema.Properties["data"].ContentEncoding = "custom"
		}},
		{"asserting_reference_sibling", func(schema *Schema) {
			schema.Properties["next"].Not = &Schema{}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, reverse := range []bool{false, true} {
				t.Run(strconv.FormatBool(reverse), func(t *testing.T) {
					a := NewAnalyzer(nil, false)
					names := []string{"first", "second"}
					if reverse {
						names[0], names[1] = names[1], names[0]
					}
					for _, name := range names {
						schema := &Schema{Type: "object", Properties: map[string]*Schema{
							"next": {Ref: toRef(name)},
							"data": {Type: "string", ContentEncoding: "base64"},
						}}
						if name == "second" {
							test.change(schema)
						}
						a.schemas[name] = schema
						a.schemaNames[name] = representationComponentName{desired: "Node", logical: "node", explicit: true, json: true}
						a.occurrences = append(a.occurrences, schemaOccurrenceAnalysis{&Schema{Ref: toRef(name)}, a})
					}
					a.finalizeRepresentations()
					require.Len(t, a.schemas, 2)
					require.NotEqual(t, a.occurrences[0].root.Ref, a.occurrences[1].root.Ref)
				})
			}
		})
	}
}

func TestByteRepresentationWithoutJSONKeepsCanonicalContract(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		blob := dsl.Type("Blob", dsl.Bytes, func() {
			dsl.Meta("openapi:typename", "PublicBlob")
			dsl.Meta("openapi:typename:canonical", "true")
			dsl.Meta("type:generate:force")
			dsl.MaxLength(2)
		})
		dsl.Service("upload", func() {
			dsl.Method("send", func() {
				dsl.Payload(func() {
					dsl.Attribute("data", blob)
				})
				dsl.HTTP(func() {
					dsl.POST("/")
					dsl.MultipartRequest()
				})
			})
		})
	})
	doc, err := BuildDocument(root.API, root.Types, root.ResultTypes)
	require.NoError(t, err)
	require.Equal(t, "binary", doc.Components.Schemas["PublicBlob"].Format)
	require.Empty(t, doc.Components.Schemas["PublicBlob"].ContentEncoding)
	require.Equal(t, 2, *doc.Components.Schemas["PublicBlob"].MaxLength)
}

func TestByteRepresentationReusesExampleProjection(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		blob := dsl.Type("Blob", dsl.Bytes, func() {
			dsl.Meta("openapi:typename:canonical", "true")
			dsl.Example([]byte("hi"))
		})
		dsl.Service("upload", func() {
			for _, name := range []string{"first", "second"} {
				dsl.Method(name, func() {
					dsl.Payload(func() {
						dsl.Attribute("data", blob)
					})
					dsl.HTTP(func() {
						dsl.POST("/" + name)
					})
				})
			}
		})
	})
	calls := 0
	BuildBodyTypes(root.API, root.Types, root.ResultTypes, WithExampleValue(func(attribute *expr.AttributeExpr, value any) (any, bool) {
		if attribute.Type == expr.Bytes {
			calls++
		}
		return OpenAPIExampleValue(attribute, value)
	}))
	require.Equal(t, 1, calls, "a shared declaration must not resample/materialize its example per representation")
}

func TestTaggedUnionRepresentationOwnership(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(strconv.FormatBool(reverse), func(t *testing.T) {
			attr := &expr.AttributeExpr{Type: &expr.Union{TypeName: "Envelope", TypeKey: "kind", ValueKey: "value", Values: []*expr.NamedAttributeExpr{{Name: "blob", Attribute: &expr.AttributeExpr{Type: expr.Bytes}}}}}
			c := expr.NewValueContext()
			occurrence, err := c.NewOccurrence(attr)
			require.NoError(t, err)
			makePlan := func(codec expr.ValueCodec) expr.ValuePlanNode {
				p, err := c.NewValuePlan(occurrence, expr.ValuePlanRequest{Target: attr, Codec: codec, Use: expr.ValuePlanSchema})
				require.NoError(t, err)
				return p.Root()
			}
			a := NewAnalyzer(nil, false)
			codecs := []expr.ValueCodec{expr.ValueCodecRaw, expr.ValueCodecJSON}
			if reverse {
				codecs = []expr.ValueCodec{expr.ValueCodecJSON, expr.ValueCodecRaw}
			}
			roots := map[expr.ValueCodec]*Schema{}
			for _, codec := range codecs {
				roots[codec] = a.analyzeOccurrence(attr, "review", makePlan(codec))
			}
			a.finalizeRepresentations()
			for codec, root := range roots {
				branch := a.schemas[strings.TrimPrefix(root.OneOf[0].Ref, "#/components/schemas/")]
				val := branch.Properties["value"]
				if codec == expr.ValueCodecJSON {
					require.Equal(t, "base64", val.ContentEncoding)
				} else {
					require.Empty(t, val.ContentEncoding)
				}
			}
		})
	}
}

func TestJSONOnlyNoPhantomLegacyVariant(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		blob := dsl.Type("Blob", dsl.Bytes, func() {
			dsl.Meta("openapi:typename", "PublicBlob")
			dsl.Meta("openapi:typename:canonical", "true")
		})
		dsl.Service("storage", func() {
			dsl.Method("put", func() {
				dsl.Payload(func() {
					dsl.Attribute("data", blob)
				})
				dsl.HTTP(func() {
					dsl.POST("/")
				})
			})
		})
	})
	doc, err := BuildDocument(root.API, root.Types, root.ResultTypes)
	require.NoError(t, err)
	require.NotNil(t, doc.Components.Schemas["PublicBlob"])
	require.Equal(t, "base64", doc.Components.Schemas["PublicBlob"].ContentEncoding)
	for name, schema := range doc.Components.Schemas {
		if strings.HasPrefix(name, "PublicBlob") {
			t.Logf("%s format=%s contentEncoding=%s", name, schema.Format, schema.ContentEncoding)
			require.Equal(t, "PublicBlob", name, "no excluded occurrence requires a legacy variant")
		}
	}
}

func TestForcedUnusedBytesKeepsLegacyContract(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.Type("Unused", dsl.Bytes, func() {
			dsl.Meta("type:generate:force")
			dsl.Meta("openapi:typename", "Unused")
			dsl.Meta("openapi:typename:canonical", "true")
		})
	})
	bodies := BuildBodyTypes(root.API, root.Types, root.ResultTypes)
	require.Len(t, bodies.Components, 1)
	require.Equal(t, "binary", bodies.Components["Unused"].Format)
	require.Empty(t, bodies.Components["Unused"].ContentEncoding)
}

func TestExplicitShapeConflictMustRemainError(t *testing.T) {
	for _, scalar := range []expr.DataType{expr.String, expr.Bytes} {
		t.Run(scalar.Name(), func(t *testing.T) {
			named := &expr.UserTypeExpr{TypeName: "Result", AttributeExpr: &expr.AttributeExpr{
				Meta: expr.MetaExpr{"openapi:typename": {"Public"}, "openapi:typename:canonical": {"true"}},
				Type: &expr.Object{{Name: "first", Attribute: &expr.AttributeExpr{Type: scalar}}, {Name: "second", Attribute: &expr.AttributeExpr{Type: expr.String}}},
			}}
			original := &expr.AttributeExpr{Type: named}
			narrowed := expr.DupAtt(original)
			narrowed.Type.(*expr.UserTypeExpr).TypeName = "ResultView"
			object := expr.AsObject(narrowed.Type.(expr.UserType).Attribute().Type)
			*object = (*object)[:1]
			context := expr.NewValueContext()
			occurrence, err := context.NewOccurrence(original)
			require.NoError(t, err)
			plan := func(target *expr.AttributeExpr) expr.ValuePlanNode {
				result, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{Target: target, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanSchema})
				require.NoError(t, err)
				return result.Root()
			}
			analyzer := NewAnalyzer(nil, false)
			analyzer.analyzeOccurrence(original, "first", plan(original))
			require.Panics(t, func() {
				analyzer.analyzeOccurrence(narrowed, "second", plan(narrowed))
				analyzer.finalizeRepresentations()
			}, "incompatible projected shapes cannot use the byte representation exception")
		})
	}
}
