package valuecontract

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/byteschema"
	"github.com/CaliLuke/loom/internal/encodingmeta"
)

type byteAliasBound struct {
	Minimum *int `json:"minimum"`
	Maximum *int `json:"maximum"`
}

// captureByteAliasLengths walks authored constraints independently of resolution
// and schema generation. This narrow lowering rejects other local constraints;
// it does not silently promote byte-length evidence into general alias evidence.
func captureByteAliasLengths(attribute *expr.AttributeExpr) ([]byteAliasBound, error) {
	var bounds []byteAliasBound
	seen := make(map[*expr.AttributeExpr]bool)
	for attribute != nil {
		if seen[attribute] {
			return nil, fmt.Errorf("cyclic byte alias declaration")
		}
		seen[attribute] = true
		if encodingmeta.Replacement(attribute.Meta) != "" || encodingmeta.SchemaOverride(attribute.Meta) {
			return nil, fmt.Errorf("explicit byte representation requires separate correspondence")
		}
		bound := byteAliasBound{}
		if validation := attribute.Validation; validation != nil {
			local := *validation
			local.MinLength, local.MaxLength = nil, nil
			if !reflect.DeepEqual(local, expr.ValidationExpr{}) {
				return nil, fmt.Errorf("non-length alias constraint requires separate reference lowering")
			}
			if validation.MinLength != nil {
				value := *validation.MinLength
				bound.Minimum = &value
			}
			if validation.MaxLength != nil {
				value := *validation.MaxLength
				bound.Maximum = &value
			}
		}
		bounds = append(bounds, bound)
		if named, ok := attribute.Type.(expr.UserType); ok {
			attribute = named.Attribute()
			continue
		}
		if attribute.Type != expr.Bytes {
			return nil, fmt.Errorf("alias-length lowering requires built-in Bytes")
		}
		return bounds, nil
	}
	return nil, fmt.Errorf("missing byte declaration")
}

func checkByteAliasLengthConformance(t *testing.T, executable string) {
	t.Helper()
	ptr := func(value int) *int {
		return &value
	}
	for _, tc := range []struct {
		name    string
		bounds  []byteAliasBound
		minimum *int
		maximum *int
	}{
		{"unconstrained", []byteAliasBound{{}, {}, {}}, nil, nil},
		{"intersect", []byteAliasBound{{Minimum: ptr(3)}, {Maximum: ptr(4)}, {Minimum: ptr(2)}}, ptr(3), ptr(4)},
		{"negative_lower", []byteAliasBound{{Minimum: ptr(-3)}, {Maximum: ptr(2)}}, ptr(-3), ptr(2)},
		{"negative_upper", []byteAliasBound{{Maximum: ptr(-1)}, {}}, nil, ptr(-1)},
		{"empty", []byteAliasBound{{Minimum: ptr(3)}, {Maximum: ptr(2)}}, ptr(3), ptr(2)},
		{"zero", []byteAliasBound{{Minimum: ptr(0)}, {Maximum: ptr(0)}}, ptr(0), ptr(0)},
		{"repeated", []byteAliasBound{{Minimum: ptr(2)}, {Maximum: ptr(4)}, {Minimum: ptr(2)}, {Maximum: ptr(4)}}, ptr(2), ptr(4)},
		{"redundant_huge", []byteAliasBound{{Maximum: ptr(2)}, {Maximum: ptr(int(^uint(0) >> 1))}}, nil, ptr(2)},
		{"empty_huge", []byteAliasBound{{Maximum: ptr(2)}, {Minimum: ptr(int(^uint(0) >> 1))}}, ptr(int(^uint(0) >> 1)), ptr(2)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			attribute := &expr.AttributeExpr{Type: expr.Bytes}
			for index := len(tc.bounds) - 1; index >= 0; index-- {
				bound := tc.bounds[index]
				attribute = &expr.AttributeExpr{
					Type:       &expr.UserTypeExpr{TypeName: fmt.Sprintf("Bytes%d", index), AttributeExpr: attribute},
					Validation: &expr.ValidationExpr{MinLength: bound.Minimum, MaxLength: bound.Maximum},
				}
			}
			captured, err := captureByteAliasLengths(attribute)
			require.NoError(t, err)
			require.Len(t, captured, len(tc.bounds)+1, "every alias-local gate and base are retained")
			lengths := []int{0, 1, 2, 3, 4, 5, 6, 7, 8}
			command := referenceConstructor("aliasLengths", map[string]any{"constraints": captured, "lengths": lengths})
			var result struct {
				Effective byteAliasBound `json:"effective"`
				Accepted  []bool         `json:"accepted"`
			}
			result = referenceDecode[struct {
				Effective byteAliasBound `json:"effective"`
				Accepted  []bool         `json:"accepted"`
			}](t, runReference(t, executable, []any{command})[0])
			require.Equal(t, byteAliasBound{Minimum: tc.minimum, Maximum: tc.maximum}, result.Effective)
			locals := make([]byteschema.Bounds, len(captured))
			for index, bound := range captured {
				locals[index] = byteschema.Bounds{Minimum: bound.Minimum, Maximum: bound.Maximum}
			}
			effective := byteschema.Intersect(locals)
			require.Equal(t, result.Effective, byteAliasBound{Minimum: effective.Minimum, Maximum: effective.Maximum}, "production intersection must agree with the independent reference")
			require.Len(t, result.Accepted, len(lengths))
			context := expr.NewValueContext()
			occurrence, err := context.NewOccurrence(attribute)
			require.NoError(t, err)
			projectionSource := expr.DupAtt(attribute)
			for current := projectionSource; ; {
				current.Validation = nil
				if named, ok := current.Type.(expr.UserType); ok {
					current = named.Attribute()
				} else {
					break
				}
			}
			projectionTarget := expr.DupAtt(projectionSource)
			current := projectionTarget
			for _, bound := range captured {
				current.Validation = &expr.ValidationExpr{MinLength: bound.Minimum, MaxLength: bound.Maximum}
				if named, ok := current.Type.(expr.UserType); ok {
					current = named.Attribute()
				}
			}
			unconstrained, err := context.NewOccurrence(projectionSource)
			require.NoError(t, err)
			plan, err := context.NewValuePlan(unconstrained, expr.ValuePlanRequest{
				Target: projectionTarget, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanRuntime,
			})
			require.NoError(t, err)
			for index, length := range lengths {
				for _, role := range []expr.ValueRole{expr.ValueRoleExample, expr.ValueRoleEnum, expr.ValueRoleDefault} {
					resolved := context.Resolve(occurrence, context.SupplyValue(expr.ValueInput{Raw: make([]byte, length)}), role)
					require.Equal(t, result.Accepted[index], resolved.Outcome() == expr.ValueResolved, "length=%d role=%d", length, role)
					validSource := context.Resolve(unconstrained, context.SupplyValue(expr.ValueInput{Raw: make([]byte, length)}), role)
					require.Equal(t, expr.ValueResolved, validSource.Outcome())
					projected := context.ProjectJSON(validSource, plan)
					require.Equal(t, result.Accepted[index], projected.Outcome() == expr.ProjectionEmitted, "target length=%d role=%d: %v", length, role, projected.Diagnostics())
				}
			}
		})
	}
}

func TestByteAliasLengthCaptureRejectsUnmodeledConstraints(t *testing.T) {
	for _, attribute := range []*expr.AttributeExpr{
		{Type: expr.String},
		{Type: expr.Bytes, Validation: &expr.ValidationExpr{Values: []any{[]byte("hi")}}},
		{Type: expr.Bytes, Validation: &expr.ValidationExpr{Pattern: "hi"}},
		{Type: expr.Bytes, Meta: expr.MetaExpr{"struct:field:type": {"custom.Blob"}}},
		{Type: expr.Bytes, Meta: expr.MetaExpr{"openapi:format": {""}}},
	} {
		_, err := captureByteAliasLengths(attribute)
		require.Error(t, err)
	}
}

func TestByteAliasLengthPresenceBoundary(t *testing.T) {
	minimum := 2
	for _, nullable := range []bool{false, true} {
		attribute := &expr.AttributeExpr{
			Type: &expr.UserTypeExpr{TypeName: "Blob", AttributeExpr: &expr.AttributeExpr{
				Type: expr.Bytes, Validation: &expr.ValidationExpr{MinLength: &minimum},
			}},
			Nullable: nullable,
		}
		context := expr.NewValueContext()
		occurrence, err := context.NewOccurrence(attribute)
		require.NoError(t, err)
		null := context.Resolve(occurrence, context.SupplyValue(expr.ValueInput{Raw: nil}), expr.ValueRoleExample)
		require.Equal(t, nullable, null.Outcome() == expr.ValueResolved, "null bypasses length only under nullable presence")
		nilBytes := context.Resolve(occurrence, context.SupplyValue(expr.ValueInput{Raw: []byte(nil)}), expr.ValueRoleExample)
		require.Equal(t, expr.ValueInvalid, nilBytes.Outcome(), "typed nil bytes retain decoded length zero")
	}
}
