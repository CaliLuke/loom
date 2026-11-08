package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
)

func TestComponentDefaultPreparationPhases(t *testing.T) {
	codec := &fingerprintCodec{}
	attribute := &expr.AttributeExpr{Type: expr.Any, DefaultValue: codec, Meta: expr.MetaExpr{"openapi:example": {"false"}}}
	generator := expr.NewRandom("annotation-phases")
	control := expr.NewRandom("annotation-phases")
	target, err := representation.PrepareStandaloneValues(attribute, generator, false)
	require.NoError(t, err)
	require.True(t, target.AnnotationsPrepared)
	require.False(t, target.ExamplesPrepared)
	require.Empty(t, target.ExampleSets)
	require.Equal(t, control.Int(), generator.Int(), "annotation preparation must not synthesize examples")
	calls := codec.calls.Load()
	annotation := target.Annotations[target.Plan.Root()]
	require.True(t, annotation.HasDefault)
	require.NoError(t, representation.PrepareTargetValues(target, attribute, generator, false))
	require.NoError(t, representation.PrepareTargetValues(target, attribute, generator, true))
	require.True(t, target.ExamplesPrepared)
	require.Equal(t, annotation, target.Annotations[target.Plan.Root()])
	require.Equal(t, calls, codec.calls.Load(), "the second phase must not repeat annotation codecs")
	require.NoError(t, representation.PrepareTargetValues(target, attribute, generator, true))
	require.Equal(t, calls, codec.calls.Load())
}

func TestComponentDefaultPreparedIdentity(t *testing.T) {
	for _, test := range []struct {
		name   string
		values []any
		opaque bool
	}{
		{"builtin", []any{"left", "right"}, false},
		{"opaque", []any{defaultAppendKey(1), defaultAppendKey(2)}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			analyzer := NewAnalyzer(nil, false)
			identities := make([]string, 0, len(test.values))
			for _, value := range test.values {
				attribute := &expr.AttributeExpr{Type: expr.Any, DefaultValue: value}
				if test.opaque {
					attribute.Type = &expr.Object{{Name: "opaque", Attribute: &expr.AttributeExpr{Type: expr.Any}}}
					attribute.DefaultValue = map[string]any{"opaque": value}
				}
				target, err := representation.PrepareStandaloneValues(attribute, nil, false)
				require.NoError(t, err)
				if test.opaque {
					require.Equal(t, expr.ProjectionUnsupported, target.Annotations[target.Plan.Root()].Default.Projection().Outcome())
				}
				analyzer.preparedExamples = target
				first := analyzer.componentAnnotationFingerprint(attribute, target.Plan.Root())
				require.Equal(t, first, analyzer.componentAnnotationFingerprint(attribute, target.Plan.Root()))
				require.NoError(t, analyzer.Err())
				identities = append(identities, first)
			}
			require.NotEqual(t, identities[0], identities[1], "different authored defaults must not share acquisition")
			t.Logf("identity %s: %q", test.name, identities)
		})
	}
}

func TestComponentDefaultMissingPreparation(t *testing.T) {
	codec := &fingerprintCodec{}
	analyzer := NewAnalyzer(nil, false)
	analyzer.componentAnnotationFingerprint(&expr.AttributeExpr{Type: expr.Any, DefaultValue: codec}, expr.ValuePlanNode{})
	require.ErrorContains(t, analyzer.Err(), "no checked annotation")
	require.Zero(t, codec.calls.Load(), "missing preparation must never invoke a raw codec fallback")
}

func TestCustomExamplesRetainCheckedAnnotations(t *testing.T) {
	for _, role := range []string{"default", "enum"} {
		for _, valid := range []bool{false, true} {
			t.Run(role+"/"+map[bool]string{false: "invalid", true: "valid"}[valid], func(t *testing.T) {
				attribute := &expr.AttributeExpr{Type: expr.String, UserExamples: []*expr.ExampleExpr{{Value: "custom-input"}}}
				value := any(23)
				if valid {
					value = "value"
				}
				if role == "default" {
					attribute.DefaultValue = value
				} else {
					attribute.Validation = &expr.ValidationExpr{Values: []any{value}}
				}
				calls := 0
				analyzer := NewAnalyzer(expr.NewRandom("custom-annotations"), false, WithExampleValue(func(_ *expr.AttributeExpr, raw any) (any, bool) {
					calls++
					require.Equal(t, "custom-input", raw)
					return "custom-output", true
				}))
				schema := analyzer.AnalyzeSchema(attribute)
				if !valid {
					require.Error(t, analyzer.Err())
					require.Zero(t, calls)
					return
				}
				require.NoError(t, analyzer.Err())
				require.Equal(t, 1, calls)
				require.Equal(t, "custom-output", schema.Example)
				if role == "default" {
					require.Equal(t, "value", schema.DefaultValue)
				} else {
					require.Equal(t, []any{"value"}, schema.Enum)
				}
			})
		}
	}
}

func TestComponentDefaultAdditionalPreparation(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: &expr.Object{}, AdditionalAttributes: &expr.AttributeExpr{Type: expr.String, DefaultValue: "extra"}}
	target, err := representation.PrepareStandaloneValues(attribute, nil, false)
	require.NoError(t, err)
	annotation, found := target.Annotations[target.Plan.Root().Additional()]
	require.True(t, found, "typed additional properties need exact prepared annotations")
	require.True(t, annotation.HasDefault)
}

func TestComponentDefaultTaggedBranchIdentity(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: &expr.Union{TypeName: "Choice", Values: []*expr.NamedAttributeExpr{
		{Name: "left", Attribute: &expr.AttributeExpr{Type: expr.String, DefaultValue: "left-default", Meta: expr.MetaExpr{"oneof:type:tag": {"left-wire"}}}},
		{Name: "right", Attribute: &expr.AttributeExpr{Type: expr.Int, DefaultValue: 7}},
	}}}
	target, err := representation.PrepareStandaloneValues(attribute, nil, false)
	require.NoError(t, err)
	analyzer := NewAnalyzer(nil, false)
	analyzer.preparedExamples = target
	require.NotEmpty(t, analyzer.componentAnnotationFingerprint(attribute, target.Plan.Root()))
	require.NoError(t, analyzer.Err(), "branch identity follows the captured wire tag, not the authored field name")
}

func TestComponentOpaqueDefaultOwnerAcrossTargets(t *testing.T) {
	value := map[string]any{"opaque": defaultAppendKey(1)}
	original := &expr.AttributeExpr{Type: expr.Any, DefaultValue: value}
	copied := expr.DupAtt(original)
	distinct := &expr.AttributeExpr{Type: expr.Any, DefaultValue: value}
	override := expr.DupAtt(original)
	override.SetDefault(value)
	analyzer := NewAnalyzer(nil, false)
	identity := func(attribute *expr.AttributeExpr) string {
		target, err := representation.PrepareStandaloneValues(attribute, nil, false)
		require.NoError(t, err)
		require.Equal(t, expr.ProjectionUnsupported, target.Annotations[target.Plan.Root()].Default.Projection().Outcome())
		analyzer.preparedExamples = target
		key := analyzer.componentAnnotationFingerprint(attribute, target.Plan.Root())
		require.NoError(t, analyzer.Err())
		return key
	}
	first := identity(original)
	require.Equal(t, first, identity(original))
	require.Equal(t, first, identity(copied))
	require.NotEqual(t, first, identity(distinct))
	require.NotEqual(t, first, identity(override))
	t.Logf("authored owner identities: %q %q %q", first, identity(distinct), identity(override))
}

func TestReferenceAnnotationPreparationCached(t *testing.T) {
	codec := &fingerprintCodec{}
	declaration := &expr.UserTypeExpr{TypeName: "DefaultCodec", UID: "default-codec", AttributeExpr: &expr.AttributeExpr{Type: expr.Any, DefaultValue: codec}}
	analyzer := NewAnalyzer(nil, false)
	first := analyzer.analyzeReferenceSchema(declaration, "first")
	require.NoError(t, analyzer.Err())
	require.NotNil(t, first)
	calls := codec.calls.Load()
	require.Positive(t, calls)
	target := analyzer.referenceAnnotationTargets[declaration]
	require.NotNil(t, target)
	require.True(t, target.AnnotationsPrepared)
	require.False(t, target.ExamplesPrepared)
	second := analyzer.analyzeReferenceSchema(declaration, "second")
	require.NoError(t, analyzer.Err())
	require.Equal(t, first.Ref, second.Ref)
	require.Same(t, target, analyzer.referenceAnnotationTargets[declaration])
	require.Equal(t, calls, codec.calls.Load(), "cached reference acquisition cannot replay annotation codecs")
	require.Empty(t, target.ExampleSets, "reference annotation preparation never draws examples")
}

func TestTemplateBodyDefaultOwner(t *testing.T) {
	for _, override := range []bool{false, true} {
		t.Run(map[bool]string{false: "inherited", true: "override"}[override], func(t *testing.T) {
			var payload expr.UserType
			root := codegen.RunDSL(t, func() {
				payload = dsl.Type("Payload", func() {
					dsl.Attribute("v", dsl.Any, func() {
						dsl.Default(map[string]any{"k": defaultAppendKey(1)})
					})
				})
				dsl.Service("probe", func() {
					dsl.Method("run", func() {
						dsl.Payload(payload)
						dsl.HTTP(func() {
							dsl.POST("/")
							dsl.Body(func() {
								dsl.Attribute("v", func() {
									if override {
										dsl.Default(map[string]any{"k": defaultAppendKey(2)})
									}
								})
							})
						})
					})
				})
			})
			source := expr.AsObject(payload.Attribute().Type).Attribute("v")
			body := expr.AsObject(root.API.HTTP.Service("probe").Endpoint("run").Body.Type).Attribute("v")
			analyzer := NewAnalyzer(nil, false)
			fingerprint := func(a *expr.AttributeExpr) string {
				target, err := representation.PrepareStandaloneValues(a, nil, false)
				require.NoError(t, err)
				require.Equal(t, expr.ProjectionUnsupported, target.Annotations[target.Plan.Root()].Default.Projection().Outcome())
				analyzer.preparedExamples = target
				result := analyzer.componentAnnotationFingerprint(a, target.Plan.Root())
				require.NoError(t, analyzer.Err())
				return result
			}
			left, right := fingerprint(source), fingerprint(body)
			t.Logf("source default=%#v body default=%#v; keys source=%s body=%s", source.DefaultValue, body.DefaultValue, left, right)
			if override {
				require.NotEqual(t, left, right, "a distinct authored opaque Body Default cannot reuse the payload default's acquisition owner")
			} else {
				require.Equal(t, left, right)
			}
		})
	}
}

func TestComposedTemplateDefaultOwner(t *testing.T) {
	for _, local := range []bool{false, true} {
		t.Run(map[bool]string{false: "inherited", true: "equal local override"}[local], func(t *testing.T) {
			var base, derived expr.UserType
			root := codegen.RunDSL(t, func() {
				base = dsl.Type("Base", func() {
					dsl.Attribute("v", dsl.Any, func() {
						dsl.Default(map[string]any{"k": defaultAppendKey(1)})
					})
				})
				derived = dsl.Type("Derived", func() {
					dsl.Extend(base)
					dsl.Attribute("v", dsl.Any, func() {
						if local {
							dsl.Default(map[string]any{"k": defaultAppendKey(1)})
						}
					})
				})
				dsl.Service("probe", func() {
					dsl.Method("run", func() {
						dsl.Payload(derived)
						dsl.HTTP(func() {
							dsl.POST("/")
							dsl.Body(func() {
								dsl.Attribute("v")
							})
						})
					})
				})
			})
			owner := func(attribute *expr.AttributeExpr) expr.DefaultValueOwner {
				target, err := representation.PrepareStandaloneValues(attribute, nil, false)
				require.NoError(t, err)
				annotation := target.Annotations[target.Plan.Root()]
				require.True(t, annotation.HasDefault)
				return annotation.Default.DefaultOwner()
			}
			baseOwner := owner(base.Attribute().Find("v"))
			derivedOwner := baseOwner
			if local {
				derivedOwner = owner(derived.Attribute().Find("v"))
			}
			body := root.API.HTTP.Service("probe").Endpoint("run").Body.Find("v")
			require.True(t, owner(body) == derivedOwner, "template publication retains the composition-selected default owner")
			require.True(t, owner(expr.DupAtt(body)) == derivedOwner)
			require.Equal(t, local, baseOwner != derivedOwner, "local declaration owns a new assignment even with equal raw value")
		})
	}
}
