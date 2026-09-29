package service

import (
	"testing"

	"github.com/CaliLuke/loom/codegen"
	stest "github.com/CaliLuke/loom/codegen/service/testdata"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestValueDataPreservesLegacySampling(t *testing.T) {
	for _, attribute := range []*expr.AttributeExpr{
		{Type: expr.String},
		{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Int}}},
		{Type: &expr.Object{{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}}}},
		{Type: expr.String, UserExamples: []*expr.ExampleExpr{{Value: "authored"}}},
		{Type: expr.String, Meta: expr.MetaExpr{"openapi:example": {"false"}}},
	} {
		before, after := expr.NewRandom("same-value-source"), expr.NewRandom("same-value-source")
		want := attribute.Example(before)
		actual := newValueData(expr.NewValueContext(), attribute, after)
		got, _ := actual.Example.LegacyValue()
		require.Equal(t, want, got)
		require.Equal(t, before.Int(), after.Int(), "same source must consume the same random draws")
	}
}

func TestServiceValueDataSurvivesPackageReanalysis(t *testing.T) {
	root := codegen.RunDSL(t, stest.WithDefaultDSL)
	services := NewServicesData(root)
	base := services.Get("WithDefault").Method("A")
	renamed := services.WithPackageNames(map[string]string{"types/shared": "shared2"}).Get("WithDefault").Method("A")
	require.NotNil(t, base.PayloadValue)
	require.NotNil(t, base.ResultValue)
	require.Same(t, base.PayloadValue, renamed.PayloadValue)
	require.Same(t, base.ResultValue, renamed.ResultValue)
	require.False(t, base.PayloadValue.Occurrence.ID() == base.ResultValue.Occurrence.ID())
	require.Equal(t, base.PayloadEx, renamed.PayloadEx)
	require.Equal(t, base.ResultEx, renamed.ResultEx)
}

func TestServiceValueDataStreamsAndErrors(t *testing.T) {
	for _, test := range []struct {
		name    string
		design  func()
		service string
		method  string
		stream  bool
	}{
		{name: "stream", design: stest.StreamingPayloadMethodDSL, service: "StreamingPayloadService", method: "StreamingPayloadMethod", stream: true},
		{name: "error", design: stest.ErrorRemedyMethodDSL, service: "ErrorRemedyMethod", method: "Show"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := codegen.RunDSL(t, test.design)
			services := NewServicesData(root)
			method := services.Get(test.service).Method(test.method)
			renamed := services.WithPackageNames(map[string]string{"types/other": "other2"}).Get(test.service).Method(test.method)
			if test.stream {
				require.NotNil(t, method.StreamingPayloadValue)
				require.NotNil(t, method.StreamingResultValue)
				require.Same(t, method.ResultValue, method.StreamingResultValue)
				require.Same(t, method.StreamingPayloadValue, renamed.StreamingPayloadValue)
				require.False(t, method.PayloadValue.Occurrence.ID() == method.StreamingPayloadValue.Occurrence.ID())
				raw, _ := method.StreamingPayloadValue.Example.LegacyValue()
				require.Equal(t, raw, method.StreamingPayloadEx)
			} else {
				require.NotNil(t, method.ErrorValues["bad_request"])
				require.Same(t, method.ErrorValues["bad_request"], renamed.ErrorValues["bad_request"])
			}
		})
	}
}
