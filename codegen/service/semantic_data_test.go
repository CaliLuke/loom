package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestSemanticDataSharesValuesAndIsolatesBackends(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.Service("values", func() {
			dsl.Method("show", func() {
				dsl.Payload(dsl.String, func() {
					dsl.Example("before")
				})
				dsl.Result(dsl.String)
			})
		})
	})
	semantics := NewSemanticData(root)
	left, right := semantics.Services(), semantics.Services()
	l, r := left.Get("values"), right.Get("values")
	lm, rm := l.Method("show"), r.Method("show")
	require.NotSame(t, l.Scope, r.Scope)
	require.NotSame(t, lm.PayloadValue, rm.PayloadValue)
	require.Equal(t, lm.PayloadValue.Occurrence.ID(), rm.PayloadValue.Occurrence.ID())
	require.Same(t, lm.PayloadValue.Context, rm.PayloadValue.Context)
	require.NotEqual(t, lm.PayloadValue.Occurrence.ID(), lm.ResultValue.Occurrence.ID())
	documentation := semantics.Method(root.Services[0].Methods[0])
	require.Equal(t, lm.PayloadValue.Occurrence.ID(), documentation.PayloadValue.Occurrence.ID())
	require.Zero(t, documentation.PayloadValue.Example.Outcome())
	lm.PayloadValue.Example = expr.ValueResult{}
	raw, _ := rm.PayloadValue.Example.LegacyValue()
	require.Equal(t, "before", raw)
	l.Scope.Unique("OnlyInLeft")
	require.Equal(t, "OnlyInLeft", r.Scope.Unique("OnlyInLeft"))
}

func TestSemanticDataCapturesBeforeBackendMutation(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.Service("values", func() {
			dsl.Method("show", func() {
				dsl.Payload(dsl.String, func() {
					dsl.Example("before")
				})
			})
		})
	})
	semantics := NewSemanticData(root)
	method := root.Services[0].Methods[0]
	method.Payload.UserExamples[0].Value = "after"
	data := semantics.Services().Get("values").Method("show").PayloadValue
	raw, _ := data.Example.LegacyValue()
	require.Equal(t, "before", raw)
	other := NewSemanticData(root).Services().Get("values").Method("show").PayloadValue
	require.NotEqual(t, data.Occurrence.ID(), other.Occurrence.ID())
	raw, _ = other.Example.LegacyValue()
	require.Equal(t, "after", raw)
}

func TestSemanticDataRejectsForeignMethod(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.Service("values", func() {
			dsl.Method("show", func() {
				dsl.Payload(dsl.String)
			})
		})
	})
	semantics := NewSemanticData(root)
	method := *root.Services[0].Methods[0]
	require.Nil(t, semantics.Method(&method))
	require.NotNil(t, semantics.Method(root.Services[0].Methods[0]))
}

func TestSemanticDataCaptureErrorKeepsDSLAttribution(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.Service("values", func() {
			dsl.Method("show", func() {
				dsl.Payload(dsl.String)
			})
		})
	})
	root.Services[0].Methods[0].Payload.DefaultValue = 17
	var recovered any
	func() {
		defer func() {
			recovered = recover()
		}()
		NewSemanticData(root)
	}()
	err, ok := recovered.(error)
	require.True(t, ok, "expected invalid finalized default to fail capture")
	var attributed *codegen.Error
	require.ErrorAs(t, err, &attributed)
	require.Equal(t, "values", attributed.Service)
	require.Equal(t, "show", attributed.Method)
}

func TestSemanticDataPreservesBackendSampling(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.Service("values", func() {
			dsl.Method("show", func() {
				dsl.Payload(dsl.String)
			})
		})
	})
	random := &backendRandomizer{Randomizer: expr.NewDeterministicRandomizer()}
	root.API.ExampleGenerator = &expr.ExampleGenerator{Randomizer: random}
	want := make([]any, 0, 2)
	for range 2 {
		want = append(want, NewServicesData(root).Get("values").Method("show").PayloadEx)
	}
	random.draws = 0
	owner := NewSemanticData(root)
	got := make([]any, 0, 2)
	for range 2 {
		got = append(got, owner.Services().Get("values").Method("show").PayloadEx)
	}
	require.Equal(t, want, got)
	require.Equal(t, 2, random.draws)
}

type backendRandomizer struct {
	expr.Randomizer
	draws int
}

func (r *backendRandomizer) String() string {
	r.draws++
	return fmt.Sprint(r.draws)
}
