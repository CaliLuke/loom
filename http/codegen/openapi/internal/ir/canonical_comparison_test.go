package ir

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestAnalyzerCanonicalRecursiveClaims(t *testing.T) {
	if os.Getenv("LOOM_CANONICAL_RECURSION_TEST") == "" {
		executable, err := os.Executable()
		require.NoError(t, err)
		command := exec.Command(executable, "-test.run=^TestAnalyzerCanonicalRecursiveClaims$", "-test.count=1")
		command.Env = append(os.Environ(), "LOOM_CANONICAL_RECURSION_TEST=1")
		output, err := command.CombinedOutput()
		if len(output) > 4000 {
			output = output[:4000]
		}
		require.NoError(t, err, string(output))
		return
	}
	// Keep a regression in recursive acquisition from exhausting a gigabyte of
	// stack or terminating unrelated package tests.
	debug.SetMaxStack(2 << 20)
	for _, same := range []bool{true, false} {
		t.Run(map[bool]string{true: "equivalent", false: "conflicting"}[same], func(t *testing.T) {
			node := func(uid string, value expr.DataType) *expr.AttributeExpr {
				named := &expr.UserTypeExpr{TypeName: "Shared", UID: uid, AttributeExpr: &expr.AttributeExpr{Meta: expr.MetaExpr{"openapi:typename:canonical": {"true"}}}}
				named.Type = &expr.Object{
					{Name: "next", Attribute: &expr.AttributeExpr{Type: named}},
					{Name: "value", Attribute: &expr.AttributeExpr{Type: value}},
				}
				return &expr.AttributeExpr{Type: named}
			}
			analyzer := NewAnalyzer(expr.NewRandom("canonical-recursion"), false, WithExampleValue(openAPIExampleValueForTest))
			first := analyzer.AnalyzeSchema(node("first", expr.String))
			kind := expr.DataType(expr.String)
			if !same {
				kind = expr.Int
			}
			if same {
				second := analyzer.AnalyzeSchema(node("second", kind))
				require.Equal(t, first.Ref, second.Ref)
				require.Len(t, analyzer.Components(), 1)
			} else {
				require.PanicsWithValue(t, "openapi: explicit component name \"Shared\" is claimed by multiple different schemas; use distinct Meta(\"openapi:typename\", ...) values", func() {
					analyzer.AnalyzeSchema(node("second", kind))
				})
			}
		})
	}
	root := recursiveSourceOverlayDSL(t)
	document, err := BuildDocument(root.API, root.Types, root.ResultTypes, WithExampleSuppression(func(*expr.AttributeExpr, bool) bool {
		return true
	}))
	require.NoError(t, err)
	require.Len(t, document.Components.Schemas, 2)
	require.NotNil(t, document.Components.Schemas["Node"])
	require.NotNil(t, document.Components.Schemas["NodeChildrenItem"])
}

func recursiveSourceOverlayDSL(t *testing.T) *expr.RootExpr {
	return codegen.RunDSL(t, func() {
		node := dsl.Type("Node", func() {
			dsl.Meta("openapi:typename:canonical", "true")
			dsl.Attribute("children", dsl.ArrayOf("NodeChildrenItem"))
			dsl.Attribute("name", dsl.String, func() {
				dsl.MinLength(2)
			})
			dsl.Required("name")
		})
		dsl.Type("NodeChildrenItem", dsl.MapOf(dsl.String, dsl.ArrayOf(node, func() {
			dsl.Nullable()
			dsl.Meta("http:encoding:sourceType", "absent")
		}), func() {
			dsl.Elem(func() {
				dsl.MinLength(1)
			})
		}), func() {
			dsl.Meta("openapi:typename:canonical", "true")
		})
		dsl.API("Recursive", func() {
			dsl.Meta("openapi:example", "false")
		})
		dsl.Service("Recursive", func() {
			dsl.Method("GetNodes", func() {
				dsl.Result(node)
				dsl.HTTP(func() {
					dsl.GET("/nodes")
					dsl.Response(200)
				})
			})
		})
	})
}

func TestCanonicalComparisonMutualUnfoldings(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(fmt.Sprint(reverse), func(t *testing.T) {
			cycle := func(size int) []*expr.AttributeExpr {
				nodes := make([]*expr.UserTypeExpr, size)
				for index := range nodes {
					name := []string{"First", "Second"}[index%2]
					nodes[index] = &expr.UserTypeExpr{TypeName: name, UID: fmt.Sprintf("%s-%d-%d", name, size, index), AttributeExpr: &expr.AttributeExpr{Meta: expr.MetaExpr{"openapi:typename": {name}, "openapi:typename:canonical": {"true"}}}}
				}
				for index, node := range nodes {
					node.Type = &expr.Object{{Name: "next", Attribute: &expr.AttributeExpr{Type: nodes[(index+1)%size]}}}
				}
				return []*expr.AttributeExpr{{Type: nodes[0]}, {Type: nodes[1]}}
			}
			base, unfolded := cycle(2), cycle(4)
			start := 0
			if reverse {
				start = 1
			}
			analyzer := NewAnalyzer(expr.NewRandom("recursive-controls"), false, WithExampleValue(openAPIExampleValueForTest), WithExampleSuppression(func(*expr.AttributeExpr, bool) bool {
				return true
			}))
			first := analyzer.AnalyzeSchema(base[start])
			second := analyzer.AnalyzeSchema(unfolded[start])
			require.Equal(t, first.Ref, second.Ref)
			require.Len(t, analyzer.Components(), 2)
			require.Empty(t, analyzer.canonicalComparisons)
			require.Equal(t, toRef("Second"), analyzer.Components()["First"].Properties["next"].Ref)
			require.Equal(t, toRef("First"), analyzer.Components()["Second"].Properties["next"].Ref)
		})
	}
}

func TestCanonicalComparisonPreservesFirstErrorAndPriorReferences(t *testing.T) {
	analyzer := NewAnalyzer(nil, false)
	firstError := errors.New("nested schema projection failed")
	analyzer.err = firstError
	prior := schemaRef{ref: toRef("Valid"), explicitName: "Valid"}
	acquired := schemaRef{ref: toRef("Candidate"), explicitName: "Candidate"}
	later := schemaRef{ref: toRef("Later"), explicitName: "Later"}
	analyzer.schemasByFingerprint["shared"] = []schemaRef{prior, acquired, later}
	analyzer.canonicalComparisons = []*canonicalComparison{{name: "Candidate", fingerprint: "shared", registration: acquired, added: true, candidate: &Schema{Type: "integer"}}}
	analyzer.finishCanonicalComparisons(true)
	require.ErrorIs(t, analyzer.Err(), firstError)
	require.Equal(t, []schemaRef{prior, later}, analyzer.schemasByFingerprint["shared"])
	require.Empty(t, analyzer.canonicalComparisons)
}

func TestRecursiveOccurrencePoliciesRemainDistinct(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(fmt.Sprint(reverse), func(t *testing.T) {
			base := recursiveFingerprintAttribute("Node")
			closed := &expr.AttributeExpr{Type: base.Type, Meta: expr.MetaExpr{"openapi:additionalProperties": {"false"}}}
			attributes := []*expr.AttributeExpr{base, closed}
			if reverse {
				attributes[0], attributes[1] = attributes[1], attributes[0]
			}
			analyzer := NewAnalyzer(expr.NewRandom("recursive-controls"), false, WithExampleValue(openAPIExampleValueForTest), WithExampleSuppression(func(*expr.AttributeExpr, bool) bool {
				return true
			}))
			schemas := map[bool]*Schema{}
			for _, attribute := range attributes {
				schema := analyzer.AnalyzeSchema(attribute)
				name, ok := schemaComponentName(schema.Ref)
				require.True(t, ok)
				schemas[attribute == closed] = analyzer.Components()[name]
			}
			require.Len(t, analyzer.Components(), 2)
			require.Nil(t, schemas[false].AdditionalProperties)
			require.Equal(t, boolPtr(false), schemas[true].AdditionalProperties.Bool)
			required := &expr.AttributeExpr{Type: base.Type, Validation: &expr.ValidationExpr{Required: []string{"next"}}}
			overlay := analyzer.AnalyzeSchema(required)
			require.Equal(t, []string{"next"}, overlay.Required)
			require.NotEmpty(t, overlay.Ref)
			require.Nil(t, base.Type.(expr.UserType).Attribute().Validation)
		})
	}
}

func TestCanonicalComparisonPreservesProjectionPanic(t *testing.T) {
	failure := errors.New("nested example codec panic")
	analyzer := NewAnalyzer(expr.NewRandom("panic"), false, WithExampleValue(func(*expr.AttributeExpr, any) (any, bool) {
		panic(failure)
	}))
	registration := schemaRef{ref: toRef("Pending"), explicitName: "Pending"}
	analyzer.schemasByFingerprint["pending"] = []schemaRef{registration}
	analyzer.canonicalComparisons = []*canonicalComparison{{name: "Pending", fingerprint: "pending", registration: registration, added: true, candidate: &Schema{}}}
	require.PanicsWithValue(t, failure, func() {
		analyzer.AnalyzeSchema(&expr.AttributeExpr{Type: expr.String, UserExamples: []*expr.ExampleExpr{{Value: "value"}}})
	})
	require.Zero(t, analyzer.schemaDepth)
	require.Empty(t, analyzer.canonicalComparisons)
	require.NotContains(t, analyzer.schemasByFingerprint, "pending")
}
