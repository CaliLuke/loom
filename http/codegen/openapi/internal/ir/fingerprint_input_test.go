package ir

import (
	"errors"
	"sync/atomic"
	"testing"

	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

type fingerprintCodec struct {
	calls atomic.Int32
}

func TestCheckedFingerprintSnapshotAllocationScaling(t *testing.T) {
	allocations := func(depth int) float64 {
		attribute := nestedFingerprintInput(depth)
		return testing.AllocsPerRun(3, func() {
			analyzer := &Analyzer{}
			analyzer.FingerprintAttribute(attribute)
			if analyzer.err != nil {
				panic(analyzer.err)
			}
		})
	}
	small, large := allocations(16), allocations(64)
	t.Logf("depths 16/64 allocate %.0f/%.0f objects", small, large)
	require.Less(t, large/small, float64(7), "structural traversal must not analyze every remaining subtree")
}

func TestCheckedFingerprintInputContract(t *testing.T) {
	expected := map[string]string{
		"empty enum": "0657819a727e16581bce31a65daa620814009b8ec3ad9778d2247a9c8bfd7305", "bytes base": "7b6ccdbdaea6b427dbb1dd808c79e75f16b7d5a25450a0a3c25e3cb34d4b3822", "union": "91004ff1ce54a3971d59f687fe0867067c1fa0c3729a1e17c5a11b4cd424edba", "result view": "0fbeaa8c8a97edd0d3f5f1f8321649b69a572f1bd8661e809442d18d1a2e2442",
		"nil":               "531d957745373893eca05cc06c9bfb538e1e7b6ba9d6bb6df6b0bff8af175d42",
		"primitive default": "aa96cb3cb12860b2d4833d8b18d7a9898144c39dc283852d9ab7abbf64782091",
		"enum":              "fa318165dc68f414680fefa93a915a4d92ce326b7043363af6f2cf0d6cb0ef43",
		"recursive":         "9b47ec3963cfa5cc8102ff3b78d99cfce9cbbbd824a2d2580189b35f8d14f89c",
		"shared":            "25d209ee640152796aadb2a6ae406ee2c9924bb9dafd3bd8acb239a6171d7589",
		"title alias":       "b5cf2e2662e046bc16960c6d70a3f12e7fcae7a494a2e782e05716c493cdb7ae",
		"hidden alias":      "20507a2eb2002b12a15c0a8120e7ac411d5d28e0653d15839ab4b02b233b9c33",
		"overlay alias":     "2b8fac27e6ad20b0ee8e38deeb4aede72dad916459060d16ce6c7537fcea4bc4",
		"nullable alias":    "fd677ca4a34d0e475e37c6420d75864f6389cb7b5bfb1eeaaa1c5a3e2accd244",
		"map":               "d5fcaec0cbe6268751c5d537c39b44aa075f8042de4bbae2e111e317cbd81326",
	}

	for _, tc := range []struct {
		name      string
		attribute *expr.AttributeExpr
	}{
		{"nil", nil},
		{"primitive default", &expr.AttributeExpr{Type: expr.String, DefaultValue: "value"}},
		{"enum", &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: []any{"b", "a", "b"}}}},
		{"recursive", recursiveFingerprintAttribute("Recursive")},
		{"shared", sharedFingerprintInput(false, false)},
		{"title alias", sharedFingerprintInput(true, false)},
		{"hidden alias", sharedFingerprintInput(true, true)},
		{"overlay alias", &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Base", UID: "Base", AttributeExpr: &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: []any{"a", "b"}}}}, Validation: &expr.ValidationExpr{Values: []any{"a"}}}},
		{"nullable alias", &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Base", UID: "Base", AttributeExpr: &expr.AttributeExpr{Type: expr.String}}, Nullable: true}},
		{"empty enum", &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: []any{}}}},
		{"bytes base", &expr.AttributeExpr{Type: expr.Bytes, Bases: []expr.DataType{&expr.UserTypeExpr{TypeName: "Binary", UID: "Binary", AttributeExpr: &expr.AttributeExpr{Type: expr.Bytes}}}}},
		{"union", &expr.AttributeExpr{Type: &expr.Union{Values: []*expr.NamedAttributeExpr{
			{Name: "right", Attribute: &expr.AttributeExpr{Type: expr.Int}},
			{Name: "left", Attribute: &expr.AttributeExpr{Type: expr.String, DefaultValue: "value"}},
		}}}},
		{"result view", resultFingerprintInput()},
		{"map", &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.Bytes}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			analyzer := &Analyzer{}
			fingerprint := analyzer.FingerprintAttribute(tc.attribute)
			require.NoError(t, analyzer.err)
			require.Len(t, fingerprint, 64)
			require.Equal(t, expected[tc.name], fingerprint)
		})
	}
}

func TestCheckedFingerprintRejectsInvalidInput(t *testing.T) {
	for _, tc := range []struct {
		name      string
		attribute *expr.AttributeExpr
	}{
		{"missing type", &expr.AttributeExpr{}},
		{"invalid default", &expr.AttributeExpr{Type: expr.String, DefaultValue: 42}},
		{"duplicate wire", fingerprintObject(
			&expr.NamedAttributeExpr{Name: "left", Attribute: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:tag:json": {"same"}}}},
			&expr.NamedAttributeExpr{Name: "right", Attribute: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:tag:json": {"same"}}}},
		)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			analyzer := &Analyzer{}
			fingerprint := analyzer.FingerprintAttribute(tc.attribute)
			require.Error(t, analyzer.err)
			require.Len(t, fingerprint, 64)
		})
	}
}

func TestCheckedFingerprintObservesNextCallMutation(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: []any{"a"}}}
	analyzer := &Analyzer{}
	first := analyzer.FingerprintAttribute(attribute)
	require.NoError(t, analyzer.err)
	attribute.Validation.Values = []any{"b"}
	second := analyzer.FingerprintAttribute(attribute)
	require.NoError(t, analyzer.err)
	require.NotEqual(t, first, second)
}

func TestCheckedFingerprintPreservesAnnotationEffects(t *testing.T) {
	codec := &fingerprintCodec{}
	attribute := fingerprintObject(
		&expr.NamedAttributeExpr{Name: "left", Attribute: &expr.AttributeExpr{Type: expr.Any, DefaultValue: codec}},
		&expr.NamedAttributeExpr{Name: "right", Attribute: &expr.AttributeExpr{Type: expr.Any, DefaultValue: codec}},
	)
	attribute.DefaultValue = map[string]any{"left": codec, "right": codec}
	analyzer := &Analyzer{}
	fingerprint := analyzer.FingerprintAttribute(attribute)
	require.NoError(t, analyzer.err)
	require.Len(t, fingerprint, 64)
	require.EqualValues(t, 2, codec.calls.Load())
	require.Equal(t, "b730230d467b87b9fb5c371231e8f7b484005744bdc130397dbb468c24537bda", fingerprint)
}

func TestCheckedFingerprintPreservesEarlierAnalyzerError(t *testing.T) {
	prior := errors.New("earlier analysis failure")
	analyzer := &Analyzer{err: prior}
	require.Len(t, analyzer.FingerprintAttribute(&expr.AttributeExpr{Type: expr.String, DefaultValue: 42}), 64)
	require.ErrorIs(t, analyzer.err, prior)
}

func BenchmarkCheckedFingerprintDepth(b *testing.B) {
	for _, tc := range []struct {
		name  string
		depth int
	}{{"16", 16}, {"64", 64}, {"256", 256}} {
		b.Run(tc.name, func(b *testing.B) {
			attribute := nestedFingerprintInput(tc.depth)
			b.ReportAllocs()
			for b.Loop() {
				analyzer := &Analyzer{}
				analyzer.FingerprintAttribute(attribute)
				if analyzer.err != nil {
					b.Fatal(analyzer.err)
				}
			}
		})
	}
}

func BenchmarkRawFingerprintDepth(b *testing.B) {
	for _, tc := range []struct {
		name  string
		depth int
	}{{"16", 16}, {"64", 64}, {"256", 256}} {
		b.Run(tc.name, func(b *testing.B) {
			attribute := nestedFingerprintInput(tc.depth)
			b.ReportAllocs()
			for b.Loop() {
				fingerprintAttribute(attribute, false)
			}
		})
	}
}

func (c *fingerprintCodec) MarshalJSON() ([]byte, error) {
	c.calls.Add(1)
	return []byte(`"value"`), nil
}

func nestedFingerprintInput(depth int) *expr.AttributeExpr {
	attribute := &expr.AttributeExpr{Type: expr.String}
	for range depth {
		attribute = fingerprintObject(&expr.NamedAttributeExpr{Name: "child", Attribute: attribute})
	}
	return attribute
}

func sharedFingerprintInput(title, hidden bool) *expr.AttributeExpr {
	base := fingerprintObject(
		&expr.NamedAttributeExpr{Name: "shown", Attribute: &expr.AttributeExpr{Type: expr.String}},
		&expr.NamedAttributeExpr{Name: "hidden", Attribute: &expr.AttributeExpr{Type: expr.String}},
	)
	base.DefaultValue = map[string]any{"shown": "yes", "hidden": "secret"}
	if hidden {
		(*expr.AsObject(base.Type))[1].Attribute.Meta = expr.MetaExpr{"openapi:generate": {"false"}}
	}
	named := &expr.UserTypeExpr{TypeName: "Shared", UID: "Shared", AttributeExpr: base}
	left := &expr.AttributeExpr{Type: named}
	if title {
		left.Title = "Display"
	}
	return fingerprintObject(
		&expr.NamedAttributeExpr{Name: "left", Attribute: left},
		&expr.NamedAttributeExpr{Name: "right", Attribute: &expr.AttributeExpr{Type: named}},
	)
}

func resultFingerprintInput() *expr.AttributeExpr {
	base := fingerprintObject(&expr.NamedAttributeExpr{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}})
	result := &expr.ResultTypeExpr{
		Identifier:   "application/vnd.record",
		UserTypeExpr: &expr.UserTypeExpr{TypeName: "Record", UID: "Record", AttributeExpr: base},
	}
	result.Views = []*expr.ViewExpr{{Name: "default", AttributeExpr: base, Parent: result}}
	return &expr.AttributeExpr{Type: result, Meta: expr.MetaExpr{expr.ViewMetaKey: {"default"}}}
}
