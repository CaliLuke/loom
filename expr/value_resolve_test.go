package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValueResolveSourceContract(t *testing.T) {
	object := &AttributeExpr{Type: &Object{{Name: "name", Attribute: &AttributeExpr{Type: String}}}, Validation: &ValidationExpr{Required: []string{"name"}}}
	for _, tc := range []struct {
		name      string
		attribute *AttributeExpr
		raw       any
		role      ValueRole
		outcome   ValueOutcome
		presence  ValuePresence
	}{
		{"bytes literal", &AttributeExpr{Type: Bytes}, "aGk=", ValueRoleExample, ValueResolved, ValuePresent},
		{"bytes nil", &AttributeExpr{Type: Bytes}, []byte(nil), ValueRoleExample, ValueResolved, ValuePresent},
		{"string rejects bytes", &AttributeExpr{Type: String}, []byte("hi"), ValueRoleExample, ValueInvalid, ValueAbsent},
		{"partial object", object, map[string]any{}, ValueRoleExample, ValueIncomplete, ValuePresent},
		{"default complete", object, map[string]any{}, ValueRoleDefault, ValueInvalid, ValueAbsent},
		{"null is not missing", object, map[string]any{"name": nil}, ValueRoleExample, ValueInvalid, ValueAbsent},
		{"nullable null", &AttributeExpr{Type: String, Nullable: true}, nil, ValueRoleExample, ValueResolved, ValueNull},
		{"nil array", &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String}}}, []string(nil), ValueRoleExample, ValueResolved, ValueNil},
		{"required Any element", &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: Any}, NonNullableElems: true}}, []any{nil}, ValueRoleExample, ValueInvalid, ValueAbsent},
		{"Any typed map self enum", &AttributeExpr{Type: Any, Validation: &ValidationExpr{Values: []any{map[int]string{1: "one"}}}}, map[int]string{1: "one"}, ValueRoleExample, ValueResolved, ValuePresent},
		{"Any typed map distinct host enum", &AttributeExpr{Type: Any, Validation: &ValidationExpr{Values: []any{map[int]string{1: "one"}}}}, map[int64]string{1: "one"}, ValueRoleExample, ValueInvalid, ValueAbsent},
		{"map spelling collision", &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: Any}, ElemType: &AttributeExpr{Type: String}}}, map[any]string{1: "a", "1": "b"}, ValueRoleExample, ValueInvalid, ValueAbsent},
		{"map mixed width spelling", &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: Any}, ElemType: &AttributeExpr{Type: String}}}, map[any]string{float32(.1): "a", float64(float32(.1)): "b"}, ValueRoleExample, ValueResolved, ValuePresent},
		{"Any mixed width map", &AttributeExpr{Type: Any}, map[any]string{float32(.1): "a", float64(float32(.1)): "b"}, ValueRoleExample, ValueResolved, ValuePresent},
		{"Any cross width enum", &AttributeExpr{Type: Any, Validation: &ValidationExpr{Values: []any{float32(.1)}}}, float64(float32(.1)), ValueRoleExample, ValueResolved, ValuePresent},
		{"whole array enum", &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String}}, Validation: &ValidationExpr{Values: []any{[]string{"one"}}}}, []string{"two"}, ValueRoleExample, ValueInvalid, ValueAbsent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			context := NewValueContext()
			occurrence, err := context.NewOccurrence(tc.attribute)
			require.NoError(t, err)
			result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: tc.raw}), tc.role)
			require.Equal(t, tc.outcome, result.Outcome(), "%v", result.Diagnostics())
			value, present := result.Value()
			require.Equal(t, tc.outcome == ValueResolved || tc.outcome == ValueIncomplete, present)
			if present {
				require.Equal(t, tc.presence, value.Presence())
			}
			if tc.name == "bytes literal" {
				scalar, ok := value.Scalar()
				require.True(t, ok)
				require.Equal(t, []byte("aGk="), scalar)
			}
		})
	}
}

func TestValueResolveAliasesAndUnionRanking(t *testing.T) {
	object := func(name string, typ DataType) *AttributeExpr {
		return &AttributeExpr{Type: &Object{{Name: name, Attribute: &AttributeExpr{Type: typ}}}, Validation: &ValidationExpr{Required: []string{name}}}
	}
	partial := &Union{Values: []*NamedAttributeExpr{{Name: "A", Attribute: object("a", String)}, {Name: "B", Attribute: object("b", Boolean)}}}
	for _, tc := range []struct {
		name      string
		attribute *AttributeExpr
		raw       any
		outcome   ValueOutcome
	}{
		{"partial ambiguity", &AttributeExpr{Type: partial}, map[string]any{}, ValueAmbiguous},
		{"complete preferred", &AttributeExpr{Type: partial}, map[string]any{"a": "hi"}, ValueResolved},
		{"losing alias invalid", &AttributeExpr{Type: &Object{{Name: "name:wire", Attribute: &AttributeExpr{Type: String}}}}, map[string]any{"name:wire": 1, "wire": "valid"}, ValueInvalid},
		{"cross member overlap", &AttributeExpr{Type: &Object{{Name: "a:b", Attribute: &AttributeExpr{Type: String}}, {Name: "b:c", Attribute: &AttributeExpr{Type: Int}}}}, map[string]any{"b": 1}, ValueInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			context := NewValueContext()
			occurrence, err := context.NewOccurrence(tc.attribute)
			require.NoError(t, err)
			result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: tc.raw}), ValueRoleExample)
			require.Equal(t, tc.outcome, result.Outcome(), "%v", result.Diagnostics())
			if tc.outcome == ValueResolved {
				value, ok := result.Value()
				require.True(t, ok)
				union, branch, _, selected := value.Union()
				require.True(t, selected)
				require.True(t, union == occurrence.ID())
				require.False(t, branch == ValueIdentity{})
			}
		})
	}
}

func TestValueResolveRecursiveOccurrencesAndOwnership(t *testing.T) {
	recursive := &UserTypeExpr{TypeName: "Recursive", AttributeExpr: &AttributeExpr{}}
	recursive.Type = &Object{
		{Name: "name", Attribute: &AttributeExpr{Type: String}},
		{Name: "next", Attribute: &AttributeExpr{Type: recursive, Nullable: true}},
	}
	recursive.Validation = &ValidationExpr{Required: []string{"name"}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(&AttributeExpr{Type: recursive})
	require.NoError(t, err)
	raw := map[string]any{"name": "root", "next": map[string]any{"name": "child", "next": nil}}
	source := context.SupplyValue(ValueInput{Raw: raw})
	result := context.Resolve(occurrence, source, ValueRoleExample)
	require.Equal(t, ValueResolved, result.Outcome(), "%v", result.Diagnostics())
	value, ok := result.Value()
	require.True(t, ok)
	fields := value.Fields()
	fields[0].Name = "mutated"
	require.Equal(t, "name", value.Fields()[0].Name)
	legacy, present := result.LegacyValue()
	require.True(t, present)
	legacy.(map[string]any)["name"] = "mutated"
	unchanged, present := result.LegacyValue()
	require.True(t, present)
	require.Equal(t, "root", unchanged.(map[string]any)["name"])
	require.Equal(t, ValueInvalid, NewValueContext().Resolve(occurrence, source, ValueRoleExample).Outcome())
	cycle := map[string]any{"name": "root"}
	cycle["next"] = cycle
	cyclic := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: cycle}), ValueRoleExample)
	require.Equal(t, ValueInvalid, cyclic.Outcome())
	require.Equal(t, "cyclic", cyclic.Diagnostics()[0].Code)
}

func TestValueResolveValidatesEveryAliasAndRetainsMissingPaths(t *testing.T) {
	child := &AttributeExpr{Type: &Object{{Name: "required", Attribute: &AttributeExpr{Type: String}}}, Validation: &ValidationExpr{Required: []string{"required"}}}
	outer := &AttributeExpr{Type: &Object{{Name: "child", Attribute: child}}, Validation: &ValidationExpr{Required: []string{"child"}}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(outer)
	require.NoError(t, err)
	result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: map[string]any{"child": map[string]any{}}}), ValueRoleExample)
	require.Equal(t, ValueIncomplete, result.Outcome())
	require.Len(t, result.Missing(), 1)
	require.Len(t, result.Missing()[0], 2)
	copy := result.Missing()
	copy[0][0] = ValueIdentity{}
	require.False(t, result.Missing()[0][0] == ValueIdentity{})
}

func TestValueResolveDeclaredPrecisionAndLegacyRaw(t *testing.T) {
	for _, role := range []ValueRole{ValueRoleExample, ValueRoleEnum, ValueRoleDefault} {
		context := NewValueContext()
		occurrence, err := context.NewOccurrence(&AttributeExpr{Type: Float64})
		require.NoError(t, err)
		result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: float32(.1)}), role)
		require.Equal(t, ValueResolved, result.Outcome())
		value, ok := result.Value()
		require.True(t, ok)
		scalar, ok := value.Scalar()
		require.True(t, ok)
		require.Equal(t, float64(.1), scalar)
		legacy, ok := result.LegacyValue()
		require.True(t, ok)
		require.Equal(t, float32(.1), legacy)
	}
}

func TestValueResolveCycleDoesNotPreemptLength(t *testing.T) {
	limit := 0
	attribute := &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: Any}}, Validation: &ValidationExpr{MaxLength: &limit}}
	raw := make([]any, 1)
	raw[0] = raw
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: raw}), ValueRoleExample)
	require.Equal(t, ValueInvalid, result.Outcome())
	require.Equal(t, "length", result.Diagnostics()[0].Code)
}

func TestValueResolveByteHostShapes(t *testing.T) {
	array := &Array{ElemType: &AttributeExpr{Type: UInt}}
	union := &Union{Values: []*NamedAttributeExpr{
		{Name: "Bytes", Attribute: &AttributeExpr{Type: Bytes}},
		{Name: "Array", Attribute: &AttributeExpr{Type: array}},
	}}
	for _, tc := range []struct {
		name    string
		typ     DataType
		raw     any
		outcome ValueOutcome
		legacy  bool
		branch  string
	}{
		{"slice Any", Any, []byte("hi"), ValueResolved, true, ""},
		{"array Any", Any, [2]byte{104, 105}, ValueResolved, true, ""},
		{"slice Bytes", Bytes, []byte("hi"), ValueResolved, true, ""},
		{"array Bytes", Bytes, [2]byte{104, 105}, ValueInvalid, false, ""},
		{"slice Array", array, []byte("hi"), ValueResolved, true, ""},
		{"array Array", array, [2]byte{104, 105}, ValueResolved, true, ""},
		{"slice union", union, []byte("hi"), ValueAmbiguous, false, ""},
		{"array union", union, [2]byte{104, 105}, ValueResolved, true, "Array"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			attribute := &AttributeExpr{Type: tc.typ}
			require.Equal(t, tc.legacy, exampleMatchesAttribute(attribute, tc.raw), "legacy source matching")
			context := NewValueContext()
			occurrence, err := context.NewOccurrence(attribute)
			require.NoError(t, err)
			result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: tc.raw}), ValueRoleExample)
			require.Equal(t, tc.outcome, result.Outcome(), "%v", result.Diagnostics())
			if tc.branch != "" {
				value, present := result.Value()
				require.True(t, present)
				_, branch, _, selected := value.Union()
				require.True(t, selected)
				require.Equal(t, occurrence.node.declaration.branches[1].id, branch.index)
				require.Equal(t, tc.branch, occurrence.node.declaration.branches[1].name)
			}
		})
	}
}

func TestValueResolvedRawAnyOwnsHostSnapshot(t *testing.T) {
	type numbers map[int64][]int32
	type outer []numbers
	original := outer{{1: {2, 3}}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(&AttributeExpr{Type: Any})
	require.NoError(t, err)
	result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: original}), ValueRoleExample)
	require.Equal(t, ValueResolved, result.Outcome())
	value, present := result.Value()
	require.True(t, present)
	first, present := value.RawAny()
	require.True(t, present)
	require.IsType(t, outer{}, first)
	first.(outer)[0][1][0] = 99
	original[0][1][1] = 88
	second, present := value.RawAny()
	require.True(t, present)
	require.Equal(t, outer{{1: {2, 3}}}, second)
	payload, present := value.Any()
	require.True(t, present)
	_, present = payload.RawAny()
	require.False(t, present)
}
