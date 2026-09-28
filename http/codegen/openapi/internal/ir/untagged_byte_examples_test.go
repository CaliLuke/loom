package ir

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestUntaggedByteExamples(t *testing.T) {
	bytes := &expr.AttributeExpr{Type: expr.Bytes}
	data := &expr.AttributeExpr{
		Type:       &expr.Object{{Name: "data", Attribute: bytes}},
		Validation: &expr.ValidationExpr{Required: []string{"data"}},
	}
	other := &expr.AttributeExpr{
		Type:       &expr.Object{{Name: "count", Attribute: &expr.AttributeExpr{Type: expr.Int}}},
		Validation: &expr.ValidationExpr{Required: []string{"count"}},
	}
	union := &expr.AttributeExpr{Type: &expr.Union{Untagged: true, Values: []*expr.NamedAttributeExpr{
		{Name: "data", Attribute: data}, {Name: "other", Attribute: other},
	}}}
	for _, raw := range [][]byte{{}, []byte("hi"), {0, 255}, []byte("aGk=")} {
		encoded := base64.StdEncoding.EncodeToString(raw)
		for _, value := range []any{raw, string(raw)} {
			object := map[string]any{"data": value}
			want := map[string]any{"data": encoded}
			for _, tc := range []struct {
				name      string
				attribute *expr.AttributeExpr
				input     any
				want      any
			}{
				{"union", union, object, want},
				{"array", &expr.AttributeExpr{Type: &expr.Array{ElemType: union}}, []any{object}, []any{want}},
				{"map", &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: union}}, map[string]any{"key": object}, map[string]any{"key": want}},
			} {
				t.Run(tc.name+"/"+encoded, func(t *testing.T) {
					got, ok := OpenAPIExampleValue(tc.attribute, tc.input)
					require.True(t, ok)
					require.Equal(t, tc.want, got)
					require.Equal(t, value, object["data"], "projection must not mutate authored input")
				})
			}
		}
	}
	value, ok := OpenAPIExampleValue(union, map[string]any{"count": 3})
	require.True(t, ok)
	require.Equal(t, map[string]any{"count": 3}, value)
}

func TestUntaggedByteExampleBranchSelection(t *testing.T) {
	for _, tc := range []struct {
		name     string
		dataType expr.DataType
		input    any
		want     any
	}{
		{"alias", &expr.UserTypeExpr{TypeName: "Blob", AttributeExpr: &expr.AttributeExpr{Type: expr.Bytes}}, "hi", "aGk="},
		{"array", &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Bytes}}, []any{"hi"}, []any{"aGk="}},
		{"map", &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.Bytes}}, map[string]any{"key": "hi"}, map[string]any{"key": "aGk="}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			branch := &expr.AttributeExpr{Type: &expr.Object{{Name: "data", Attribute: &expr.AttributeExpr{Type: tc.dataType}}}, Validation: &expr.ValidationExpr{Required: []string{"data"}}}
			union := &expr.Union{Untagged: true, Values: []*expr.NamedAttributeExpr{{Name: "data", Attribute: branch}}}
			attribute := &expr.AttributeExpr{Type: union}
			value, ok := OpenAPIExampleValue(attribute, map[string]any{"data": tc.input})
			require.True(t, ok)
			require.Equal(t, map[string]any{"data": tc.want}, value)
			_, ok = OpenAPIExampleValue(attribute, map[string]any{"data": 42})
			require.False(t, ok, "wrong field types must remain rejected")
			union.Values = append(union.Values, &expr.NamedAttributeExpr{Name: "duplicate", Attribute: expr.DupAtt(branch)})
			_, ok = OpenAPIExampleValue(attribute, map[string]any{"data": tc.input})
			require.False(t, ok, "ambiguous wire shapes must remain rejected")
		})
	}
}

func FuzzUntaggedByteExampleRoundTrip(f *testing.F) {
	for _, value := range [][]byte{{}, []byte("hi"), {0, 255}, []byte("aGk=")} {
		f.Add(value)
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		field := &expr.AttributeExpr{Type: expr.Bytes}
		branch := &expr.AttributeExpr{
			Type:       &expr.Object{{Name: "data", Attribute: field}},
			Validation: &expr.ValidationExpr{Required: []string{"data"}},
		}
		attribute := &expr.AttributeExpr{Type: &expr.Union{Untagged: true, Values: []*expr.NamedAttributeExpr{{Name: "data", Attribute: branch}}}}
		for _, validation := range []*expr.ValidationExpr{nil, {Values: []any{raw}}} {
			field.Validation = validation
			for _, input := range []any{raw, string(raw)} {
				value, ok := OpenAPIExampleValue(attribute, map[string]any{"data": input})
				require.True(t, ok)
				object, ok := value.(map[string]any)
				require.True(t, ok)
				encoded, ok := object["data"].(string)
				require.True(t, ok)
				decoded, err := base64.StdEncoding.DecodeString(encoded)
				require.NoError(t, err)
				require.Equal(t, string(raw), string(decoded))
			}
		}
	})
}

func TestUntaggedByteExampleEnums(t *testing.T) {
	for _, enum := range []any{"hi", []byte("hi")} {
		field := &expr.AttributeExpr{
			Type:       expr.Bytes,
			Validation: &expr.ValidationExpr{Values: []any{enum}},
		}
		branch := &expr.AttributeExpr{
			Type:       &expr.Object{{Name: "data", Attribute: field}},
			Validation: &expr.ValidationExpr{Required: []string{"data"}},
		}
		attribute := &expr.AttributeExpr{Type: &expr.Union{
			Untagged: true,
			Values:   []*expr.NamedAttributeExpr{{Name: "data", Attribute: branch}},
		}}
		value, ok := OpenAPIExampleValue(attribute, map[string]any{"data": "hi"})
		require.True(t, ok)
		require.Equal(t, map[string]any{"data": "aGk="}, value)
		_, ok = OpenAPIExampleValue(attribute, map[string]any{"data": "different"})
		require.False(t, ok)
		require.Equal(t, []any{enum}, field.Validation.Values)
	}
}

func TestUntaggedByteAndTextEnumBranches(t *testing.T) {
	branch := func(datatype expr.DataType) *expr.AttributeExpr {
		return &expr.AttributeExpr{
			Type: &expr.Object{{Name: "data", Attribute: &expr.AttributeExpr{
				Type:       datatype,
				Validation: &expr.ValidationExpr{Values: []any{"hi"}},
			}}},
			Validation: &expr.ValidationExpr{Required: []string{"data"}},
		}
	}
	attribute := &expr.AttributeExpr{Type: &expr.Union{
		Untagged: true,
		Values: []*expr.NamedAttributeExpr{
			{Name: "bytes", Attribute: branch(expr.Bytes)},
			{Name: "text", Attribute: branch(expr.String)},
		},
	}}
	value, ok := OpenAPIExampleValue(attribute, map[string]any{"data": []byte("hi")})
	require.True(t, ok)
	require.Equal(t, map[string]any{"data": "aGk="}, value)
}
