package expr

import "testing"

func TestArrayIsCompatible(t *testing.T) {
	var (
		b  = true
		i  = 1
		ia = [2]int{1, 2}
		is = []int{3, 4}
	)
	cases := map[string]struct {
		typ      DataType
		values   []any
		expected bool
	}{
		"compatible": {
			typ:      Int,
			values:   []any{ia, is},
			expected: true,
		},
		"not array and slice": {
			typ:      String,
			values:   []any{b, i},
			expected: false,
		},
		"array but not compatible": {
			typ:      String,
			values:   []any{ia},
			expected: false,
		},
		"slice but not compatible": {
			typ:      String,
			values:   []any{is},
			expected: false,
		},
	}

	for k, tc := range cases {
		array := Array{
			ElemType: &AttributeExpr{
				Type: tc.typ,
			},
		}
		for _, value := range tc.values {
			if actual := array.IsCompatible(value); tc.expected != actual {
				t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
			}
		}
	}
}

func TestObjectRename(t *testing.T) {
	cases := map[string]struct {
		old, new string
		expected []string
	}{
		"renamed": {
			old:      "foo",
			new:      "qux",
			expected: []string{"qux", "bar"},
		},
		"unmatched": {
			old:      "baz",
			new:      "qux",
			expected: []string{"foo", "bar"},
		},
	}

	for k, tc := range cases {
		object := &Object{
			&NamedAttributeExpr{
				Name: "foo",
				Attribute: &AttributeExpr{
					Type: String,
				},
			},
			&NamedAttributeExpr{
				Name: "bar",
				Attribute: &AttributeExpr{
					Type: String,
				},
			},
		}
		object.Rename(tc.old, tc.new)
		for _, s := range tc.expected {
			if att := object.Attribute(s); att == nil {
				t.Errorf("%s: %s not found", k, s)
			}
		}
	}
}

func TestObjectIsCompatible(t *testing.T) {
	var (
		b = true
		i = 1
		s = struct {
			Foo string
		}{
			Foo: "foo",
		}
		m = map[int]string{}
	)
	cases := map[string]struct {
		values   []any
		expected bool
	}{
		"compatible": {
			values:   []any{s, m},
			expected: true,
		},
		"not comatible": {
			values:   []any{b, i},
			expected: false,
		},
	}

	object := Object{}
	for k, tc := range cases {
		for _, value := range tc.values {
			if actual := object.IsCompatible(value); tc.expected != actual {
				t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
			}
		}
	}
}

func TestMapIsCompatible(t *testing.T) {
	var (
		b   = true
		i   = 1
		ism = map[int]string{
			1: "foo",
		}
		ssm = map[string]string{
			"bar": "bar",
		}
		iim = map[int]int{
			2: 2,
		}
	)
	cases := map[string]struct {
		values   []any
		expected bool
	}{
		"compatible": {
			values:   []any{ism},
			expected: true,
		},
		"not comatible": {
			values:   []any{b, i},
			expected: false,
		},
		"map but not comatible": {
			values:   []any{ssm, iim},
			expected: false,
		},
	}

	m := Map{
		KeyType: &AttributeExpr{
			Type: Int,
		},
		ElemType: &AttributeExpr{
			Type: String,
		},
	}
	for k, tc := range cases {
		for _, value := range tc.values {
			if actual := m.IsCompatible(value); tc.expected != actual {
				t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
			}
		}
	}
}

func TestUnionGetTypeKey(t *testing.T) {
	cases := map[string]struct {
		typeKey  string
		expected string
	}{
		"default": {
			typeKey:  "",
			expected: "type",
		},
		"custom": {
			typeKey:  "kind",
			expected: "kind",
		},
		"discriminator": {
			typeKey:  "discriminator",
			expected: "discriminator",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			union := &Union{
				TypeName: "TestUnion",
				TypeKey:  tc.typeKey,
			}
			if actual := union.GetTypeKey(); actual != tc.expected {
				t.Errorf("got %q, expected %q", actual, tc.expected)
			}
		})
	}
}

func TestUnionGetValueKey(t *testing.T) {
	cases := map[string]struct {
		valueKey string
		expected string
	}{
		"default": {
			valueKey: "",
			expected: "value",
		},
		"custom": {
			valueKey: "data",
			expected: "data",
		},
		"payload": {
			valueKey: "payload",
			expected: "payload",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			union := &Union{
				TypeName: "TestUnion",
				ValueKey: tc.valueKey,
			}
			if actual := union.GetValueKey(); actual != tc.expected {
				t.Errorf("got %q, expected %q", actual, tc.expected)
			}
		})
	}
}

func TestUnionDupPreservesCustomKeys(t *testing.T) {
	original := &Union{
		TypeName:         "TestUnion",
		ExplicitTypeName: true,
		TypeKey:          "kind",
		ValueKey:         "data",
		Values: []*NamedAttributeExpr{
			{
				Name: "String",
				Attribute: &AttributeExpr{
					Type: String,
				},
			},
		},
	}

	duplicated := Dup(original)

	dup, ok := duplicated.(*Union)
	if !ok {
		t.Fatalf("expected *Union, got %T", duplicated)
	}

	if dup.TypeKey != original.TypeKey {
		t.Errorf("TypeKey: got %q, expected %q", dup.TypeKey, original.TypeKey)
	}
	if dup.ValueKey != original.ValueKey {
		t.Errorf("ValueKey: got %q, expected %q", dup.ValueKey, original.ValueKey)
	}
	if dup.ExplicitTypeName != original.ExplicitTypeName {
		t.Errorf("ExplicitTypeName: got %t, expected %t", dup.ExplicitTypeName, original.ExplicitTypeName)
	}
	if dup.GetTypeKey() != original.GetTypeKey() {
		t.Errorf("GetTypeKey(): got %q, expected %q", dup.GetTypeKey(), original.GetTypeKey())
	}
	if dup.GetValueKey() != original.GetValueKey() {
		t.Errorf("GetValueKey(): got %q, expected %q", dup.GetValueKey(), original.GetValueKey())
	}
}

func TestQualifiedTypeName(t *testing.T) {
	var (
		array = &Array{
			ElemType: &AttributeExpr{
				Type: String,
			},
		}
		mapStringString = &Map{
			KeyType: &AttributeExpr{
				Type: String,
			},
			ElemType: &AttributeExpr{
				Type: String,
			},
		}
		mapStringArray = &Map{
			KeyType: &AttributeExpr{
				Type: String,
			},
			ElemType: &AttributeExpr{
				Type: array,
			},
		}
		mapStringMap = &Map{
			KeyType: &AttributeExpr{
				Type: String,
			},
			ElemType: &AttributeExpr{
				Type: mapStringString,
			},
		}
	)
	cases := map[string]struct {
		t        DataType
		expected string
	}{
		"boolean": {
			t:        Boolean,
			expected: "boolean",
		},
		"int": {
			t:        Int,
			expected: "int",
		},
		"int32": {
			t:        Int32,
			expected: "int32",
		},
		"int64": {
			t:        Int64,
			expected: "int64",
		},
		"uint": {
			t:        UInt,
			expected: "uint",
		},
		"uint32": {
			t:        UInt32,
			expected: "uint32",
		},
		"uint64": {
			t:        UInt64,
			expected: "uint64",
		},
		"float32": {
			t:        Float32,
			expected: "float32",
		},
		"float64": {
			t:        Float64,
			expected: "float64",
		},
		"string": {
			t:        String,
			expected: "string",
		},
		"bytes": {
			t:        Bytes,
			expected: "bytes",
		},
		"any": {
			t:        Any,
			expected: "any",
		},
		"user type": {
			t: &UserTypeExpr{
				TypeName: "userType",
			},
			expected: "userType",
		},
		"result type": {
			t: &ResultTypeExpr{
				UserTypeExpr: &UserTypeExpr{
					TypeName: "resultType",
				},
			},
			expected: "resultType",
		},
		"object": {
			t:        &Object{},
			expected: "object",
		},
		"array": {
			t:        array,
			expected: "array<string>",
		},
		"map": {
			t:        mapStringString,
			expected: "map<string, string>",
		},
		"map contains array": {
			t:        mapStringArray,
			expected: "map<string, array<string>>",
		},
		"map contains map": {
			t:        mapStringMap,
			expected: "map<string, map<string, string>>",
		},
	}

	for k, tc := range cases {
		if actual := QualifiedTypeName(tc.t); tc.expected != actual {
			t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
		}
	}
}
