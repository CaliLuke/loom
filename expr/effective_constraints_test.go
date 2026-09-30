package expr

import (
	"math"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

type constraintCodecValue struct {
	value string
}

func (constraintCodecValue) MarshalJSON() ([]byte, error) {
	return []byte(`"custom"`), nil
}

func TestEffectiveConstraintsEnumRefinement(t *testing.T) {
	tests := []struct {
		name      string
		values    []any
		want      []any
		wantError string
	}{
		{name: "equal", values: []any{1, 2}, want: []any{1, 2}},
		{name: "subset", values: []any{2}, want: []any{2}},
		{name: "partial overlap", values: []any{2, 3}, wantError: "enum member 3"},
		{name: "disjoint", values: []any{3}, wantError: "enum member 3"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			base := namedScalar("Base", Int, &ValidationExpr{Values: []any{1, 2}})
			derived := namedScalar("Derived", base, &ValidationExpr{Values: test.values})
			constraints, err := EffectiveConstraintsFor(&AttributeExpr{Type: derived})
			if test.wantError != "" {
				require.ErrorContains(t, err, test.wantError)
				require.ErrorContains(t, err, "Base")
				return
			}
			require.NoError(t, err)
			actual, present := constraints.EnumCandidates()
			require.True(t, present)
			require.Equal(t, test.want, actual)
		})
	}
}

func TestEffectiveConstraintsPreservePresentEmptyAuthoredEnum(t *testing.T) {
	base := namedScalar("EmptyBase", String, &ValidationExpr{Values: []any{}})
	inherited := namedScalar("EmptyInherited", base, nil)
	constraints, err := EffectiveConstraintsFor(&AttributeExpr{Type: inherited})
	require.NoError(t, err)
	candidates, present := constraints.EnumCandidates()
	require.True(t, present)
	require.Empty(t, candidates)
	require.Equal(t, [][]any{{}}, constraints.Validation().Lowered().Enums())

	widened := namedScalar("EmptyWidened", base, &ValidationExpr{Values: []any{"value"}})
	_, err = EffectiveConstraintsFor(&AttributeExpr{Type: widened})
	require.ErrorContains(t, err, `enum member "value"`)
	require.ErrorContains(t, err, `ancestor "EmptyBase"`)

	absent := namedScalar("AbsentBase", String, nil)
	constraints, err = EffectiveConstraintsFor(&AttributeExpr{Type: absent})
	require.NoError(t, err)
	_, present = constraints.EnumCandidates()
	require.False(t, present)
}

func TestEffectiveConstraintsEnumUsesDeclaredTypeEquality(t *testing.T) {
	tests := []struct {
		name       string
		typeName   string
		underlying DataType
		base       []any
		derived    []any
	}{
		{name: "Bytes text and slice", typeName: "Blob", underlying: Bytes, base: []any{[]byte("ok")}, derived: []any{"ok"}},
		{name: "Any deep value", typeName: "Anything", underlying: Any,
			base: []any{map[string]any{"count": int64(1)}}, derived: []any{map[string]any{"count": int64(1)}}},
		{name: "Float32 normalization", typeName: "Ratio", underlying: Float32,
			base: []any{float32(0.1)}, derived: []any{float64(0.1)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			base := namedScalar(test.typeName+"Base", test.underlying, &ValidationExpr{Values: test.base})
			derived := namedScalar(test.typeName+"Derived", base, &ValidationExpr{Values: test.derived})
			constraints, err := EffectiveConstraintsFor(&AttributeExpr{Type: derived})
			require.NoError(t, err)
			actual, present := constraints.EnumCandidates()
			require.True(t, present)
			require.Equal(t, test.derived, actual)
		})
	}
	stringBase := namedScalar("TextBase", String, &ValidationExpr{Values: []any{"ok"}})
	stringDerived := namedScalar("TextDerived", stringBase, &ValidationExpr{Values: []any{[]byte("ok")}})
	_, err := EffectiveConstraintsFor(&AttributeExpr{Type: stringDerived})
	require.ErrorContains(t, err, `enum member []byte{0x6f, 0x6b}`)
}

func TestAuthorizationValuesUseEffectiveEnum(t *testing.T) {
	base := namedScalar("AuthorizationBase", String, &ValidationExpr{Values: []any{"read", "write"}})
	derived := namedScalar("AuthorizationDerived", base, &ValidationExpr{Values: []any{"read"}})

	values, err := AuthorizationValues(&AttributeExpr{Type: derived})
	require.NoError(t, err)
	require.Equal(t, []string{"read"}, values)

	filtered := namedScalar("AuthorizationFiltered", base, &ValidationExpr{Pattern: "^admin$"})
	constraints, err := EffectiveConstraintsFor(&AttributeExpr{Type: filtered})
	require.NoError(t, err)
	candidates, present := constraints.EnumCandidates()
	require.True(t, present)
	require.Empty(t, candidates)
	values, err = AuthorizationValues(&AttributeExpr{Type: filtered})
	require.ErrorContains(t, err, "non-empty string enum")
	require.Empty(t, values)

	empty := namedScalar("AuthorizationEmpty", String, &ValidationExpr{EnumClauses: [][]any{{}}})
	values, err = AuthorizationValues(&AttributeExpr{Type: empty})
	require.ErrorContains(t, err, "non-empty string enum")
	require.Empty(t, values)
}

func TestEffectiveConstraintsDistinguishLocalContractFailureFromEnumWidening(t *testing.T) {
	minimum := 5.0
	base := namedScalar("Base", Int, nil)
	derived := namedScalar("Derived", base, &ValidationExpr{
		Minimum: &minimum,
		Values:  []any{1},
	})

	_, err := EffectiveConstraintsFor(&AttributeExpr{Type: derived})
	require.ErrorContains(t, err, `enum member 1 declared by "Derived" violates the effective contract for "Derived"`)
	require.NotContains(t, err.Error(), "not admitted by ancestor")
}

func TestOccurrenceRejectsAuthoredEnumWidening(t *testing.T) {
	base := namedScalar("Base", Int, &ValidationExpr{Values: []any{1, 2}})
	derived := namedScalar("Derived", base, &ValidationExpr{Values: []any{2, 3}})

	_, err := NewValueContext().NewOccurrence(&AttributeExpr{Type: derived})
	require.ErrorContains(t, err, "enum member 3")
}

func TestOccurrenceRejectsInheritedDefaultOutsideNarrowedEnum(t *testing.T) {
	base := namedScalar("Base", Int, &ValidationExpr{Values: []any{1, 2}})
	base.Attribute().DefaultValue = 1
	derived := namedScalar("Derived", base, &ValidationExpr{Values: []any{2}})

	_, err := NewValueContext().NewOccurrence(&AttributeExpr{Type: derived})
	require.ErrorContains(t, err, "default value 1")
}

func TestResolveAccumulatesNamedObjectRequirements(t *testing.T) {
	object := &Object{
		{Name: "first", Attribute: &AttributeExpr{Type: String}},
		{Name: "second", Attribute: &AttributeExpr{Type: String}},
	}
	base := &UserTypeExpr{TypeName: "Base", UID: "Base", AttributeExpr: &AttributeExpr{
		Type: object, Validation: &ValidationExpr{Required: []string{"first"}},
	}}
	derived := &UserTypeExpr{TypeName: "Derived", UID: "Derived", AttributeExpr: &AttributeExpr{
		Type: base, Validation: &ValidationExpr{Required: []string{"second"}},
	}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(&AttributeExpr{Type: derived})
	require.NoError(t, err)
	result := context.Resolve(
		occurrence,
		context.SupplyValue(ValueInput{Raw: map[string]any{"first": "present"}}),
		ValueRoleDefault,
	)
	require.Equal(t, ValueInvalid, result.Outcome())
}

func TestResolveConjoinsAllNumericPredicates(t *testing.T) {
	minimum, exclusiveMinimum := 5.0, 1.0
	attribute := &AttributeExpr{Type: Float64, Validation: &ValidationExpr{
		Minimum: &minimum, ExclusiveMinimum: &exclusiveMinimum,
	}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: 3.0}), ValueRoleDefault)
	require.Equal(t, ValueInvalid, result.Outcome())
}

func TestEffectiveConstraintsDefaultUsesNarrowedContract(t *testing.T) {
	base := namedScalar("Base", Int, &ValidationExpr{Values: []any{1, 2}})
	base.Attribute().DefaultValue = 1
	derived := namedScalar("Derived", base, &ValidationExpr{Values: []any{2}})

	_, err := EffectiveConstraintsFor(&AttributeExpr{Type: derived})
	require.ErrorContains(t, err, "default value 1")
	require.ErrorContains(t, err, `declared by "Base"`)
	require.ErrorContains(t, err, `effective contract for "Derived"`)

	derived.Attribute().DefaultValue = 2
	constraints, err := EffectiveConstraintsFor(&AttributeExpr{Type: derived})
	require.NoError(t, err)
	actual, present := constraints.Default()
	require.True(t, present)
	require.Equal(t, 2, actual)
}

func TestPreFinalDefaultSelectionMatchesEffectiveConstraints(t *testing.T) {
	base := namedScalar("DefaultBase", String, nil)
	base.Attribute().DefaultValue = "base"
	middle := namedScalar("DefaultMiddle", base, nil)
	inherited := &AttributeExpr{Type: namedScalar("DefaultInherited", middle, nil), Nullable: true}
	overriddenType := namedScalar("DefaultOverride", middle, nil)
	overriddenType.Attribute().DefaultValue = "override"

	for _, test := range []struct {
		name      string
		attribute *AttributeExpr
		want      any
		present   bool
	}{
		{name: "arbitrary depth inherited nullable", attribute: inherited, want: "base", present: true},
		{name: "nearest local override", attribute: &AttributeExpr{Type: overriddenType}, want: "override", present: true},
		{name: "absent", attribute: &AttributeExpr{Type: namedScalar("NoDefault", String, nil)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			constraints, err := EffectiveConstraintsFor(test.attribute)
			require.NoError(t, err)
			actual, present := constraints.Default()
			require.Equal(t, test.present, present)
			require.Equal(t, test.want, actual)
			require.Equal(t, actual, test.attribute.effectiveDefault())
		})
	}

	left := namedScalar("DefaultCycleLeft", String, nil)
	right := namedScalar("DefaultCycleRight", left, nil)
	left.Attribute().Type = right
	cyclic := &AttributeExpr{Type: left}
	require.Nil(t, cyclic.effectiveDefault())
	_, err := EffectiveConstraintsFor(cyclic)
	require.Error(t, err)
}

func TestEffectiveConstraintsConjoinBoundsAndRequiredFields(t *testing.T) {
	min, exclusiveMin, max, exclusiveMax := 1.0, 1.0, 10.0, 8.0
	baseNumber := namedScalar("BaseNumber", Float64, &ValidationExpr{Minimum: &min, Maximum: &max})
	derivedNumber := namedScalar("DerivedNumber", baseNumber, &ValidationExpr{
		ExclusiveMinimum: &exclusiveMin,
		ExclusiveMaximum: &exclusiveMax,
	})
	numberConstraints, err := EffectiveConstraintsFor(&AttributeExpr{Type: derivedNumber})
	require.NoError(t, err)
	numberValidation := numberConstraints.Validation().Lowered()
	require.Nil(t, numberValidation.Minimum)
	require.Equal(t, exclusiveMin, *numberValidation.ExclusiveMinimum)
	require.Nil(t, numberValidation.Maximum)
	require.Equal(t, exclusiveMax, *numberValidation.ExclusiveMaximum)

	object := &Object{
		{Name: "first:one", Attribute: &AttributeExpr{Type: String}},
		{Name: "second:two", Attribute: &AttributeExpr{Type: String}},
	}
	baseObject := &UserTypeExpr{TypeName: "BaseObject", UID: "BaseObject", AttributeExpr: &AttributeExpr{
		Type: object, Validation: &ValidationExpr{Required: []string{"first:one"}},
	}}
	derivedObject := &UserTypeExpr{TypeName: "DerivedObject", UID: "DerivedObject", AttributeExpr: &AttributeExpr{
		Type: baseObject, Validation: &ValidationExpr{Required: []string{"second:two"}},
	}}
	objectConstraints, err := EffectiveConstraintsFor(&AttributeExpr{Type: derivedObject})
	require.NoError(t, err)
	require.Equal(t, []string{"second:two", "first:one"}, objectConstraints.Validation().Lowered().Required)
	require.Equal(t, []string{"two", "one"}, []string{
		objectConstraints.Required()[0].WireName,
		objectConstraints.Required()[1].WireName,
	})
}

func TestEffectiveConstraintsAreOccurrenceLocal(t *testing.T) {
	base := namedScalar("Base", Int, &ValidationExpr{Values: []any{1, 2, 3}})
	left := &AttributeExpr{Type: base, Validation: &ValidationExpr{Values: []any{1}}}
	right := &AttributeExpr{Type: base, Validation: &ValidationExpr{Values: []any{2, 3}}}

	leftConstraints, err := EffectiveConstraintsFor(left)
	require.NoError(t, err)
	rightConstraints, err := EffectiveConstraintsFor(right)
	require.NoError(t, err)
	leftCandidates, leftPresent := leftConstraints.EnumCandidates()
	rightCandidates, rightPresent := rightConstraints.EnumCandidates()
	require.True(t, leftPresent)
	require.True(t, rightPresent)
	require.Equal(t, []any{1}, leftCandidates)
	require.Equal(t, []any{2, 3}, rightCandidates)
}

func TestEffectiveConstraintsValidateObjectEnumsAndDefaultsAfterSnapshotConstruction(t *testing.T) {
	object := &Object{
		{Name: "first", Attribute: &AttributeExpr{Type: String}},
		{Name: "second", Attribute: &AttributeExpr{Type: String}},
	}
	baseValue := map[string]any{"first": "one", "second": "two"}
	base := &UserTypeExpr{TypeName: "BaseObject", UID: "BaseObject", AttributeExpr: &AttributeExpr{
		Type: object, Validation: &ValidationExpr{Values: []any{baseValue}, Required: []string{"first"}},
		DefaultValue: baseValue,
	}}
	derived := &UserTypeExpr{TypeName: "DerivedObject", UID: "DerivedObject", AttributeExpr: &AttributeExpr{
		Type: base, Validation: &ValidationExpr{Values: []any{baseValue}, Required: []string{"second"}},
	}}
	constraints, err := EffectiveConstraintsFor(&AttributeExpr{Type: derived})
	require.NoError(t, err)
	require.Len(t, constraints.Required(), 2)

	base.Attribute().DefaultValue = map[string]any{"first": "one"}
	_, err = EffectiveConstraintsFor(&AttributeExpr{Type: derived})
	require.ErrorContains(t, err, `default value map[string]interface {}{"first":"one"}`)
}

func TestEffectiveConstraintsDoNotHideObjectFailuresBehindCustomCodecs(t *testing.T) {
	minimum, minLength := 5.0, 2
	tests := []struct {
		name  string
		field *AttributeExpr
		raw   any
	}{
		{name: "missing required", field: &AttributeExpr{Type: String}},
		{name: "numeric", field: &AttributeExpr{Type: Float64, Validation: &ValidationExpr{Minimum: &minimum}}, raw: 1.0},
		{name: "string", field: &AttributeExpr{Type: String, Validation: &ValidationExpr{MinLength: &minLength}}, raw: "x"},
		{name: "enum", field: &AttributeExpr{Type: String, Validation: &ValidationExpr{Values: []any{"allowed"}}}, raw: "rejected"},
		{name: "unsupported", field: &AttributeExpr{Type: Any}, raw: make(chan int)},
	}
	for _, test := range tests {
		for _, customFirst := range []bool{false, true} {
			for _, source := range []string{"enum", "default"} {
				name := test.name + "/" + source
				if customFirst {
					name += "/custom first"
				} else {
					name += "/custom last"
				}
				t.Run(name, func(t *testing.T) {
					invalidField := &NamedAttributeExpr{Name: "invalid", Attribute: test.field}
					customField := &NamedAttributeExpr{Name: "custom", Attribute: &AttributeExpr{Type: Any}}
					object := &Object{invalidField, customField}
					if customFirst {
						object = &Object{customField, invalidField}
					}
					raw := map[string]any{"custom": constraintCodecValue{}}
					if test.raw != nil {
						raw["invalid"] = test.raw
					}
					attribute := &AttributeExpr{
						Type:       object,
						Validation: &ValidationExpr{Required: []string{"invalid", "custom"}},
					}
					if source == "enum" {
						attribute.Validation.Values = []any{raw}
					} else {
						attribute.DefaultValue = raw
					}
					_, err := EffectiveConstraintsFor(attribute)
					require.Error(t, err)
				})
			}
		}
	}
}

func TestEffectiveConstraintsCheckAliasCollectionLengthsWithOpaqueElements(t *testing.T) {
	minimum := 2
	tests := []struct {
		name string
		typ  DataType
		raw  any
	}{
		{
			name: "array",
			typ:  &Array{ElemType: &AttributeExpr{Type: Any}},
			raw:  []any{constraintCodecValue{}},
		},
		{
			name: "map",
			typ: &Map{
				KeyType:  &AttributeExpr{Type: String},
				ElemType: &AttributeExpr{Type: Any},
			},
			raw: map[string]any{"one": constraintCodecValue{}},
		},
	}
	for _, test := range tests {
		for _, source := range []string{"enum", "default"} {
			t.Run(test.name+"/"+source, func(t *testing.T) {
				base := &UserTypeExpr{TypeName: "Base", UID: "Base", AttributeExpr: &AttributeExpr{Type: test.typ}}
				derived := &UserTypeExpr{TypeName: "Derived", UID: "Derived", AttributeExpr: &AttributeExpr{
					Type: base, Validation: &ValidationExpr{MinLength: &minimum},
				}}
				if source == "enum" {
					derived.Attribute().Validation.Values = []any{test.raw}
				} else {
					derived.Attribute().DefaultValue = test.raw
				}
				_, err := EffectiveConstraintsFor(&AttributeExpr{Type: derived})
				require.Error(t, err)
			})
		}
	}
}

func TestEffectiveConstraintsCompareKnownBranchesAroundOpaqueLeaves(t *testing.T) {
	exact := int64(9_007_199_254_740_993)
	object := &Object{
		{Name: "blob", Attribute: &AttributeExpr{Type: Bytes}},
		{Name: "exact", Attribute: &AttributeExpr{Type: Int64}},
		{Name: "ratio", Attribute: &AttributeExpr{Type: Float32}},
		{Name: "optional", Attribute: &AttributeExpr{Type: String, Nullable: true}},
		{Name: "custom", Attribute: &AttributeExpr{Type: Any}},
	}
	required := []string{"blob", "exact", "ratio", "optional", "custom"}
	baseValue := map[string]any{
		"blob": []byte("ok"), "exact": exact, "ratio": float32(0.1),
		"optional": nil, "custom": constraintCodecValue{value: "same"},
	}
	equivalent := map[string]any{
		"blob": "ok", "exact": exact, "ratio": float64(0.1),
		"optional": nil, "custom": constraintCodecValue{value: "same"},
	}
	base := &UserTypeExpr{TypeName: "Base", UID: "Base", AttributeExpr: &AttributeExpr{
		Type: object, Validation: &ValidationExpr{Values: []any{baseValue}, Required: required},
	}}
	derived := &UserTypeExpr{TypeName: "Derived", UID: "Derived", AttributeExpr: &AttributeExpr{
		Type: base, Validation: &ValidationExpr{Values: []any{equivalent}}, DefaultValue: equivalent,
	}}
	_, err := EffectiveConstraintsFor(&AttributeExpr{Type: derived})
	require.NoError(t, err)

	notEquivalent := map[string]any{
		"blob": "ok", "exact": exact, "ratio": float64(0.1),
		"optional": nil, "custom": constraintCodecValue{value: "different"},
	}
	derived.Attribute().Validation.Values = []any{notEquivalent}
	derived.Attribute().DefaultValue = nil
	_, err = EffectiveConstraintsFor(&AttributeExpr{Type: derived})
	require.ErrorContains(t, err, "enum member")
}

func TestEffectiveConstraintsTraverseOpaqueAliasAncestry(t *testing.T) {
	same := constraintCodecValue{value: "same"}
	different := constraintCodecValue{value: "different"}
	base := namedScalar("OpaqueBase", Any, &ValidationExpr{Values: []any{same}})
	middle := namedScalar("OpaqueMiddle", base, nil)

	for _, test := range []struct {
		name    string
		value   any
		wantErr bool
	}{
		{name: "equal opaque", value: same},
		{name: "different opaque", value: different, wantErr: true},
		{name: "known versus opaque", value: "same", wantErr: true},
	} {
		t.Run("enum/"+test.name, func(t *testing.T) {
			derived := namedScalar("OpaqueDerived", middle, &ValidationExpr{Values: []any{test.value}})
			_, err := EffectiveConstraintsFor(&AttributeExpr{Type: derived})
			if test.wantErr {
				require.ErrorContains(t, err, "enum member")
				return
			}
			require.NoError(t, err)
		})

		t.Run("local default/"+test.name, func(t *testing.T) {
			derived := namedScalar("OpaqueDefaultDerived", middle, nil)
			derived.Attribute().DefaultValue = test.value
			_, err := EffectiveConstraintsFor(&AttributeExpr{Type: derived})
			if test.wantErr {
				require.ErrorContains(t, err, "default value")
				return
			}
			require.NoError(t, err)
		})
	}

	inheritedDefaultBase := namedScalar("OpaqueDefaultBase", Any, &ValidationExpr{Values: []any{same, different}})
	inheritedDefaultBase.Attribute().DefaultValue = same
	inheritedDefaultMiddle := namedScalar("OpaqueDefaultMiddle", inheritedDefaultBase, nil)
	valid := namedScalar("OpaqueDefaultValid", inheritedDefaultMiddle, &ValidationExpr{Values: []any{same}})
	_, err := EffectiveConstraintsFor(&AttributeExpr{Type: valid})
	require.NoError(t, err)

	invalid := namedScalar("OpaqueDefaultInvalid", inheritedDefaultMiddle, &ValidationExpr{Values: []any{different}})
	_, err = EffectiveConstraintsFor(&AttributeExpr{Type: invalid})
	require.ErrorContains(t, err, "default value")
}

func TestEffectiveConstraintsCompareOpaqueCollectionMembers(t *testing.T) {
	same := constraintCodecValue{value: "same"}
	different := constraintCodecValue{value: "different"}
	for _, test := range []struct {
		name      string
		typ       DataType
		baseValue any
		equal     any
		different any
	}{
		{
			name:      "array element",
			typ:       &Array{ElemType: &AttributeExpr{Type: Any}},
			baseValue: []any{same},
			equal:     []any{same},
			different: []any{different},
		},
		{
			name:      "map value",
			typ:       &Map{KeyType: &AttributeExpr{Type: String}, ElemType: &AttributeExpr{Type: Any}},
			baseValue: map[string]any{"custom": same},
			equal:     map[string]any{"custom": same},
			different: map[string]any{"custom": different},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			base := &UserTypeExpr{TypeName: "OpaqueCollectionBase", UID: "OpaqueCollectionBase", AttributeExpr: &AttributeExpr{
				Type: test.typ, Validation: &ValidationExpr{Values: []any{test.baseValue}},
			}}
			equal := &UserTypeExpr{TypeName: "OpaqueCollectionEqual", UID: "OpaqueCollectionEqual", AttributeExpr: &AttributeExpr{
				Type: base, Validation: &ValidationExpr{Values: []any{test.equal}},
			}}
			_, err := EffectiveConstraintsFor(&AttributeExpr{Type: equal})
			require.NoError(t, err)

			different := &UserTypeExpr{TypeName: "OpaqueCollectionDifferent", UID: "OpaqueCollectionDifferent", AttributeExpr: &AttributeExpr{
				Type: base, Validation: &ValidationExpr{Values: []any{test.different}},
			}}
			_, err = EffectiveConstraintsFor(&AttributeExpr{Type: different})
			require.ErrorContains(t, err, "enum member")
		})
	}
}

func TestEffectiveConstraintsRejectUnsupportedNonCodecValues(t *testing.T) {
	channelOne := make(chan int)
	channelTwo := make(chan int)
	function := func() {}
	value := 1
	unsafePointer := unsafe.Pointer(&value)
	tests := []struct {
		name string
		raw  any
	}{
		{name: "channel", raw: channelOne},
		{name: "function", raw: function},
		{name: "complex", raw: complex(1, 2)},
		{name: "uintptr", raw: uintptr(1)},
		{name: "unsafe pointer", raw: unsafePointer},
		{name: "not a number", raw: math.NaN()},
		{name: "infinity", raw: math.Inf(1)},
	}
	for _, test := range tests {
		for _, source := range []string{"enum", "default"} {
			t.Run(test.name+"/"+source, func(t *testing.T) {
				attribute := &AttributeExpr{Type: Any}
				if source == "enum" {
					attribute.Validation = &ValidationExpr{Values: []any{test.raw}}
				} else {
					attribute.DefaultValue = test.raw
				}
				_, err := EffectiveConstraintsFor(attribute)
				require.Error(t, err)
			})
		}
	}

	base := namedScalar("Base", Any, &ValidationExpr{Values: []any{channelOne}})
	derived := namedScalar("Derived", base, &ValidationExpr{Values: []any{channelTwo}})
	_, err := EffectiveConstraintsFor(&AttributeExpr{Type: derived})
	require.Error(t, err)
	require.False(t, resolvedEnumEqual(ResolvedValue{}, ResolvedValue{}))
}

func TestEffectiveConstraintsRejectCyclicAuthoredValuesWithoutFormattingThem(t *testing.T) {
	cyclic := map[string]any{}
	cyclic["self"] = cyclic

	_, err := EffectiveConstraintsFor(&AttributeExpr{
		Type:       Any,
		Validation: &ValidationExpr{Values: []any{cyclic}},
	})
	require.ErrorContains(t, err, "invalid authored enum member")
	require.ErrorContains(t, err, "cyclic value")

	_, err = EffectiveConstraintsFor(&AttributeExpr{Type: Any, DefaultValue: cyclic})
	require.ErrorContains(t, err, "invalid authored default")
	require.ErrorContains(t, err, "cyclic value")
}

func TestEffectiveConstraintsValidateEnumClauseMemberShapes(t *testing.T) {
	cyclic := map[string]any{}
	cyclic["self"] = cyclic
	for _, test := range []struct {
		name string
		raw  any
	}{
		{name: "unsupported", raw: make(chan int)},
		{name: "cyclic", raw: cyclic},
	} {
		t.Run(test.name, func(t *testing.T) {
			attribute := &AttributeExpr{
				Type:       Any,
				Validation: &ValidationExpr{EnumClauses: [][]any{{test.raw}}},
			}
			_, err := EffectiveConstraintsFor(attribute)
			require.Error(t, err)
			errors := attribute.validateEffectiveConstraints("attribute - ", nil)
			require.NotEmpty(t, errors.Errors)
		})
	}

	for _, attribute := range []*AttributeExpr{
		{Type: Int, Validation: &ValidationExpr{Values: []any{int64(1)}}},
		{
			Type: &Map{
				KeyType:  &AttributeExpr{Type: Int},
				ElemType: &AttributeExpr{Type: String},
			},
			Validation: &ValidationExpr{Values: []any{map[int64]string{1: "value"}}},
		},
	} {
		_, err := EffectiveConstraintsFor(attribute)
		require.ErrorContains(t, err, "violates the effective contract")
		require.NotContains(t, err.Error(), "violates its declared type")
	}

	base := namedScalar("CarrierBase", String, &ValidationExpr{Values: []any{"ab", "ax"}})
	derived := namedScalar("CarrierDerived", base, &ValidationExpr{Pattern: "b$"})
	constraints, err := EffectiveConstraintsFor(&AttributeExpr{Type: derived})
	require.NoError(t, err)
	physical := &AttributeExpr{Type: String, Validation: constraints.Validation().Lowered()}
	physicalConstraints, err := EffectiveConstraintsFor(physical)
	require.NoError(t, err)
	candidates, present := physicalConstraints.EnumCandidates()
	require.True(t, present)
	require.Equal(t, []any{"ab"}, candidates)

	opaqueKey := constraintCodecValue{value: "same"}
	opaqueMap := map[constraintCodecValue]string{opaqueKey: "value"}
	opaqueAttribute := &AttributeExpr{
		Type: &Map{
			KeyType:  &AttributeExpr{Type: Any},
			ElemType: &AttributeExpr{Type: String},
		},
		Validation: &ValidationExpr{EnumClauses: [][]any{{opaqueMap}}},
	}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(opaqueAttribute)
	require.NoError(t, err)
	result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: opaqueMap}), ValueRoleEnum)
	require.Equal(t, ValueUnsupported, result.Outcome())
	require.False(t, result.checkableFailure)
}

func TestEffectiveConstraintsCompareOpaqueMapKeysByRawIdentity(t *testing.T) {
	key := constraintCodecValue{value: "same"}
	baseValue := map[constraintCodecValue]string{key: "value"}
	base := namedScalar("Base", Any, &ValidationExpr{Values: []any{baseValue}})

	equal := namedScalar("Equal", base, &ValidationExpr{
		Values: []any{map[constraintCodecValue]string{key: "value"}},
	})
	_, err := EffectiveConstraintsFor(&AttributeExpr{Type: equal})
	require.NoError(t, err)

	different := namedScalar("Different", base, &ValidationExpr{
		Values: []any{map[constraintCodecValue]string{{value: "different"}: "value"}},
	})
	_, err = EffectiveConstraintsFor(&AttributeExpr{Type: different})
	require.ErrorContains(t, err, "not admitted by ancestor")

	known := namedScalar("Known", base, &ValidationExpr{
		Values: []any{map[string]string{"same": "value"}},
	})
	_, err = EffectiveConstraintsFor(&AttributeExpr{Type: known})
	require.ErrorContains(t, err, "not admitted by ancestor")
}

func TestEffectiveConstraintsAllowRecursiveObjectDeclarations(t *testing.T) {
	node := &UserTypeExpr{TypeName: "Node", UID: "Node"}
	node.AttributeExpr = &AttributeExpr{Type: &Object{
		{Name: "name", Attribute: &AttributeExpr{Type: String}},
		{Name: "next", Attribute: &AttributeExpr{Type: node}},
	}, Validation: &ValidationExpr{Required: []string{"name"}}}

	constraints, err := EffectiveConstraintsFor(&AttributeExpr{Type: node})
	require.NoError(t, err)
	require.Equal(t, []string{"name"}, constraints.Validation().Lowered().Required)
}

func TestEffectiveValidationConjoinsPatternAndFormatClauses(t *testing.T) {
	base := namedScalar("PatternBase", String, &ValidationExpr{Pattern: "^a"})
	middle := namedScalar("PatternMiddle", base, &ValidationExpr{Pattern: "^a"})
	derived := namedScalar("PatternDerived", middle, &ValidationExpr{Pattern: "b$"})

	constraints, err := EffectiveConstraintsFor(&AttributeExpr{Type: derived})
	require.NoError(t, err)
	require.Equal(t, []EffectiveValidationClause{
		{Kind: EffectivePatternClause, Value: "b$", Provenance: EffectiveConstraintSource{Declaration: "PatternDerived"}},
		{Kind: EffectivePatternClause, Value: "^a", Provenance: EffectiveConstraintSource{Declaration: "PatternMiddle"}},
	}, clausesWithoutOccurrence(constraints.Validation().Clauses()))
	require.Equal(t, []string{"b$", "^a"}, constraints.Validation().Lowered().Patterns())

	context := NewValueContext()
	occurrence, err := context.NewOccurrence(&AttributeExpr{Type: derived})
	require.NoError(t, err)
	for value, outcome := range map[string]ValueOutcome{"ab": ValueResolved, "xb": ValueInvalid, "ax": ValueInvalid} {
		result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: value}), ValueRoleExample)
		require.Equal(t, outcome, result.Outcome(), value)
	}

	ip := namedScalar("IP", String, &ValidationExpr{Format: FormatIP})
	ipv4 := namedScalar("IPv4", ip, &ValidationExpr{Format: FormatIPv4})
	constraints, err = EffectiveConstraintsFor(&AttributeExpr{Type: ipv4, DefaultValue: "192.0.2.1"})
	require.NoError(t, err)
	require.Equal(t, []ValidationFormat{FormatIPv4, FormatIP}, constraints.Validation().Lowered().Formats())

	email := namedScalar("Email", String, &ValidationExpr{Format: FormatEmail})
	hostname := namedScalar("Hostname", email, &ValidationExpr{Format: FormatHostname})
	_, err = EffectiveConstraintsFor(&AttributeExpr{Type: hostname})
	require.NoError(t, err, "an empty conjunctive contract remains admissible")
	_, err = EffectiveConstraintsFor(&AttributeExpr{Type: hostname, DefaultValue: "example.com"})
	require.ErrorContains(t, err, `default value "example.com"`)
}

func TestEffectiveValidationRejectsValuesThatMissAnyClause(t *testing.T) {
	base := namedScalar("Base", String, &ValidationExpr{Pattern: "^a"})
	for _, test := range []struct {
		name      string
		configure func(*AttributeExpr)
	}{
		{name: "default", configure: func(attribute *AttributeExpr) { attribute.DefaultValue = "xb" }},
		{name: "enum", configure: func(attribute *AttributeExpr) {
			attribute.Validation = &ValidationExpr{Pattern: "b$", Values: []any{"xb"}}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			attribute := &AttributeExpr{Type: base, Validation: &ValidationExpr{Pattern: "b$"}}
			test.configure(attribute)
			_, err := EffectiveConstraintsFor(attribute)
			require.Error(t, err)
		})
	}
}

func TestEffectiveConstraintDiagnosticsPreserveOwnerFailure(t *testing.T) {
	tests := []struct {
		name      string
		attribute *AttributeExpr
		want      string
		reject    string
	}{
		{
			name: "default mismatch uses owner diagnostic",
			attribute: &AttributeExpr{
				Type:         String,
				DefaultValue: "missing",
				Validation:   &ValidationExpr{Values: []any{"present"}},
			},
			want: "default value \"missing\" declared by \"string\" violates the effective contract for \"string\"",
		},
		{
			name: "matching default does not mask invalid enum",
			attribute: &AttributeExpr{
				Type:         String,
				DefaultValue: "ab",
				Validation:   &ValidationExpr{Pattern: "^a", Values: []any{"xb", "ab"}},
			},
			want:   "enum member \"xb\"",
			reject: "default value \"ab\" is not one of",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errors := test.attribute.validateEffectiveConstraints("attribute - ", nil)
			require.Len(t, errors.Errors, 1)
			require.ErrorContains(t, errors.Errors[0], test.want)
			if test.reject != "" {
				require.NotContains(t, errors.Errors[0].Error(), test.reject)
			}
		})
	}

	bytes := &AttributeExpr{
		Type:         Bytes,
		DefaultValue: []byte("ok"),
		Validation:   &ValidationExpr{Values: []any{"ok"}},
	}
	constraints, err := EffectiveConstraintsFor(bytes)
	require.NoError(t, err)
	actual, present := constraints.Default()
	require.True(t, present)
	require.Equal(t, []byte("ok"), actual)
}

func TestEffectiveConstraintDiagnosticsUseDeclaredEnumEquality(t *testing.T) {
	minimum := 2
	attribute := &AttributeExpr{
		Type:         Bytes,
		DefaultValue: "x",
		Validation: &ValidationExpr{
			Values:    []any{[]byte("x")},
			MinLength: &minimum,
		},
	}

	errors := attribute.validateEffectiveConstraints("attribute - ", nil)
	require.Len(t, errors.Errors, 1)
	require.ErrorContains(t, errors.Errors[0], "enum member")
	require.ErrorContains(t, errors.Errors[0], "violates the effective contract")
	require.NotContains(t, errors.Errors[0].Error(), "default value")
	require.NotContains(t, errors.Errors[0].Error(), "not one of the accepted values")
}

func TestInheritedEnumAndDerivedPredicatesRemainConjunctive(t *testing.T) {
	base := namedScalar("Base", String, &ValidationExpr{Values: []any{"ab", "ax"}})
	derived := namedScalar("Derived", base, &ValidationExpr{Pattern: "b$"})
	attribute := &AttributeExpr{Type: derived}
	constraints, err := EffectiveConstraintsFor(attribute)
	require.NoError(t, err)
	candidates, present := constraints.EnumCandidates()
	require.True(t, present)
	require.Equal(t, []any{"ab"}, candidates)
	require.Equal(t, []any{"ab", "ax"}, constraints.Validation().Lowered().Enums()[0])

	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	accepted := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: "ab"}), ValueRoleExample)
	require.Equal(t, ValueResolved, accepted.Outcome())
	rejected := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: "ax"}), ValueRoleExample)
	require.Equal(t, ValueInvalid, rejected.Outcome())
	schema := mustInlineJSONSchema(t, attribute)
	require.Equal(t, "b$", schema["pattern"])
	require.Equal(t, []any{"ab", "ax"}, schema["allOf"].([]any)[0].(map[string]any)["enum"])

	explicit := namedScalar("Explicit", base, &ValidationExpr{Pattern: "b$", Values: []any{"ab", "ax"}})
	_, err = EffectiveConstraintsFor(&AttributeExpr{Type: explicit})
	require.ErrorContains(t, err, `enum member "ax" declared by "Explicit" violates the effective contract for "Explicit"`)

	inheritedDefault := namedScalar("DefaultBase", String, &ValidationExpr{Values: []any{"ab", "ax"}})
	inheritedDefault.Attribute().DefaultValue = "ax"
	invalidDefault := namedScalar("InvalidDefault", inheritedDefault, &ValidationExpr{Pattern: "b$"})
	_, err = EffectiveConstraintsFor(&AttributeExpr{Type: invalidDefault})
	require.ErrorContains(t, err, `default value "ax" declared by "DefaultBase" violates the effective contract for "InvalidDefault"`)
	validDefault := namedScalar("ValidDefault", inheritedDefault, &ValidationExpr{Pattern: "b$"})
	validDefault.Attribute().DefaultValue = "ab"
	_, err = EffectiveConstraintsFor(&AttributeExpr{Type: validDefault})
	require.NoError(t, err)
}

func TestEffectiveValidationLoweringIsDetachedAndExplicit(t *testing.T) {
	base := namedScalar("Base", String, &ValidationExpr{Pattern: "^a", Format: FormatIP})
	derived := namedScalar("Derived", base, &ValidationExpr{Pattern: "b$", Format: FormatIPv4})
	validation, err := EffectiveConstraintsFor(&AttributeExpr{Type: derived})
	require.NoError(t, err)
	snapshot := validation.Validation()
	lowered := snapshot.Lowered()
	require.Empty(t, lowered.Pattern)
	require.Empty(t, lowered.Format)
	require.Equal(t, []string{"b$", "^a"}, lowered.PatternClauses)
	require.Equal(t, []ValidationFormat{FormatIPv4, FormatIP}, lowered.FormatClauses)

	lowered.PatternClauses[0] = "^x"
	lowered.FormatClauses[0] = FormatHostname
	lowered.Pattern = "z$"
	lowered.Format = FormatEmail
	require.Equal(t, []string{"b$", "^a"}, snapshot.Lowered().Patterns())
	require.Equal(t, []ValidationFormat{FormatIPv4, FormatIP}, snapshot.Lowered().Formats())

	duplicate := lowered.Dup()
	duplicate.PatternClauses[0] = "^y"
	duplicate.FormatClauses[0] = FormatURI
	require.Equal(t, []string{"z$", "^x", "^a"}, lowered.Patterns())
	require.Equal(t, []ValidationFormat{FormatEmail, FormatHostname, FormatIP}, lowered.Formats())

	roundTrip, err := EffectiveConstraintsFor(&AttributeExpr{Type: String, Validation: lowered})
	require.NoError(t, err)
	require.Equal(t, lowered.Patterns(), roundTrip.Validation().Lowered().Patterns())
	require.Equal(t, lowered.Formats(), roundTrip.Validation().Lowered().Formats())
	for _, clause := range roundTrip.Validation().Clauses() {
		require.Equal(t, "string", clause.Provenance.Declaration, "lowering creates a synthetic physical declaration")
	}
}

func clausesWithoutOccurrence(clauses []EffectiveValidationClause) []EffectiveValidationClause {
	for index := range clauses {
		clauses[index].Provenance.Occurrence = ValueIdentity{}
	}
	return clauses
}

func namedScalar(name string, underlying DataType, validation *ValidationExpr) *UserTypeExpr {
	return &UserTypeExpr{
		TypeName: name,
		UID:      name,
		AttributeExpr: &AttributeExpr{
			Type:       underlying,
			Validation: validation,
		},
	}
}
