package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/CaliLuke/loom/expr"
)

// TestNewProtoMessageNames checks the names that the fields, oneofs and oneof
// fields of a message take: regular fields keep their names, a oneof avoids
// its own branch names and the names already taken with "_oneof", and a oneof
// field whose name is taken takes the name of its union field as a prefix.
func TestNewProtoMessageNames(t *testing.T) {
	leaf := &expr.UserTypeExpr{TypeName: "Leaf", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}}}
	other := &expr.UserTypeExpr{TypeName: "Other", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}}}
	union := func(branches ...*expr.NamedAttributeExpr) *expr.NamedAttributeExpr {
		return &expr.NamedAttributeExpr{Attribute: &expr.AttributeExpr{Type: &expr.Union{TypeName: "U", Values: branches}}}
	}
	named := func(name string, nat *expr.NamedAttributeExpr) *expr.NamedAttributeExpr {
		nat.Name = name
		return nat
	}
	branch := func(name string, t expr.DataType) *expr.NamedAttributeExpr {
		return &expr.NamedAttributeExpr{Name: name, Attribute: &expr.AttributeExpr{Type: t}}
	}
	cases := []struct {
		name     string
		obj      *expr.Object
		fields   map[string]string
		branches map[string][]string
		goFields map[string]string
		goBranch map[string][]string
		conflict *protoNameConflict
	}{
		{
			name: "distinct names",
			obj: &expr.Object{
				branch("id", expr.String),
				named("pick", union(branch("Leaf", leaf), branch("Other", other))),
			},
			fields:   map[string]string{"id": "id", "pick": "pick"},
			branches: map[string][]string{"pick": {"leaf", "other"}},
			goFields: map[string]string{"id": "Id", "pick": "Pick"},
			goBranch: map[string][]string{"pick": {"Leaf", "Other"}},
		},
		{
			name: "shared primitive branch",
			obj: &expr.Object{
				named("a", union(branch("String", expr.String), branch("Int64", expr.Int64))),
				named("b", union(branch("Boolean", expr.Boolean), branch("Int64", expr.Int64))),
				named("c", union(branch("Int64", expr.Int64), branch("Leaf", leaf))),
			},
			fields:   map[string]string{"a": "a", "b": "b", "c": "c"},
			branches: map[string][]string{"a": {"string_", "int64_"}, "b": {"boolean", "b_int64"}, "c": {"c_int64", "leaf"}},
			goBranch: map[string][]string{"a": {"String_", "Int64_"}, "b": {"Boolean", "BInt64"}, "c": {"CInt64", "Leaf"}},
		},
		{
			name: "regular field keeps its name",
			obj: &expr.Object{
				named("pick", union(branch("Leaf", leaf), branch("Other", other))),
				branch("leaf", expr.String),
			},
			fields:   map[string]string{"pick": "pick", "leaf": "leaf"},
			branches: map[string][]string{"pick": {"pick_leaf", "other"}},
			goBranch: map[string][]string{"pick": {"PickLeaf", "Other"}},
		},
		{
			name: "prefix applied until unique",
			obj: &expr.Object{
				named("a", union(branch("Int64", expr.Int64))),
				named("b", union(branch("Int64", expr.Int64))),
				branch("b_int64", expr.String),
			},
			branches: map[string][]string{"a": {"int64_"}, "b": {"b_b_int64"}},
			goBranch: map[string][]string{"b": {"BBInt64"}},
		},
		{
			name: "oneof named like its branch and a regular field",
			obj: &expr.Object{
				named("leaf", union(branch("Leaf", leaf), branch("Other", other))),
				branch("leaf_oneof", expr.String),
			},
			fields:   map[string]string{"leaf": "leaf_oneof_oneof", "leaf_oneof": "leaf_oneof"},
			branches: map[string][]string{"leaf": {"leaf", "other"}},
			goFields: map[string]string{"leaf": "LeafOneofOneof"},
		},
		{
			name: "oneof named like a branch of another oneof",
			obj: &expr.Object{
				named("u", union(branch("Leaf", leaf), branch("Other", other))),
				named("w", union(branch("u", leaf), branch("v", other))),
			},
			fields:   map[string]string{"u": "u", "w": "w"},
			branches: map[string][]string{"u": {"leaf", "other"}, "w": {"w_u", "v"}},
			goBranch: map[string][]string{"w": {"WU", "V"}},
		},
		{
			name: "branch named like a later oneof",
			obj: &expr.Object{
				named("w", union(branch("u", leaf), branch("v", other))),
				named("u", union(branch("Leaf", leaf), branch("Other", other))),
			},
			fields:   map[string]string{"w": "w", "u": "u_oneof"},
			branches: map[string][]string{"w": {"u", "v"}, "u": {"leaf", "other"}},
			goFields: map[string]string{"u": "UOneof"},
		},
		{
			name: "branch names that differ only in case",
			obj: &expr.Object{
				named("a", union(branch("fooBar", expr.String))),
				named("b", union(branch("FooBar", expr.Int))),
			},
			branches: map[string][]string{"a": {"foo_bar"}, "b": {"b_foo_bar"}},
		},
		{
			name: "mapped attribute names",
			obj: &expr.Object{
				branch("leaf:l", expr.String),
				named("pick:p", union(branch("Leaf", leaf), branch("Other", other))),
			},
			fields:   map[string]string{"leaf": "leaf", "pick": "pick"},
			branches: map[string][]string{"pick": {"pick_leaf", "other"}},
			goFields: map[string]string{"pick": "Pick", "pick:p": "Pick"},
			goBranch: map[string][]string{"pick": {"PickLeaf", "Other"}, "pick:p": {"PickLeaf", "Other"}},
		},
		{
			name: "two regular fields",
			obj: &expr.Object{
				branch("fooBar", expr.String),
				branch("foo_bar", expr.String),
			},
			conflict: &protoNameConflict{name: "foo_bar", first: `attribute "fooBar"`, second: `attribute "foo_bar"`},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			names := newProtoMessageNames(c.obj)
			for field, want := range c.fields {
				assert.Equal(t, want, names.fields[field], "field %q", field)
			}
			for field, want := range c.branches {
				assert.Equal(t, want, names.branches[field], "branches of %q", field)
			}
			for field, want := range c.goFields {
				assert.Equal(t, want, names.goField(field), "Go field %q", field)
			}
			for field, want := range c.goBranch {
				assert.Equal(t, want, names.goBranches(field), "Go branches of %q", field)
			}
			assert.Equal(t, c.conflict, names.conflict)
		})
	}
}

// TestProtoGoName checks that protoGoName returns the Go names that
// protoc-gen-go generates for protocol buffer field names.
func TestProtoGoName(t *testing.T) {
	cases := map[string]string{
		"leaf":             "Leaf",
		"leaf_oneof":       "LeafOneof",
		"leaf_oneof_oneof": "LeafOneofOneof",
		"string_":          "String_",
		"int64_":           "Int64_",
		"b_int64":          "BInt64",
		"w_u":              "WU",
		"embedded_one_of":  "EmbeddedOneOf",
		"a1b":              "A1B",
		"field_2":          "Field_2",
		"_x":               "XX",
	}
	for name, want := range cases {
		assert.Equal(t, want, protoGoName(name), "protoGoName(%q)", name)
	}
}
