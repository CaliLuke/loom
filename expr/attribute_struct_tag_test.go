package expr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAttributeExprValidateJSONWireNames(t *testing.T) {
	cases := map[string]struct {
		field   string
		meta    MetaExpr
		invalid string
	}{
		"plain name":            {field: "user_id"},
		"space and newline":     {field: "a b\nc"},
		"unicode":               {field: "日本語"},
		"json name override":    {field: "x", meta: MetaExpr{"struct:tag:json:name": {"custom"}}},
		"json tag with options": {field: "x", meta: MetaExpr{"struct:tag:json": {"custom,omitempty"}}},
		"comma in field name":   {field: "a,b", invalid: "a,b"},
		"quote in field name":   {field: `a"b`, invalid: `a"b`},
		"single quote":          {field: "it's", invalid: "it's"},
		"backslash":             {field: `a\b`, invalid: `a\b`},
		"backtick":              {field: "a`b", invalid: "a`b"},
		"invalid utf8":          {field: "a\xffb", invalid: "a\xffb"},
		"quote in json name":    {field: "x", meta: MetaExpr{"struct:tag:json:name": {`c"d`}}, invalid: `c"d`},
		"quote in json tag":     {field: "x", meta: MetaExpr{"struct:tag:json": {`c"d,omitempty`}}, invalid: `c"d`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			attribute := &AttributeExpr{Type: &Object{{Name: tc.field, Attribute: &AttributeExpr{Type: String, Meta: tc.meta}}}}
			verr := attribute.Validate("ctx", nil)
			if tc.invalid == "" {
				require.Empty(t, verr.Errors)
				return
			}
			require.Len(t, verr.Errors, 1)
			require.Contains(t, verr.Errors[0].Error(), fmt.Sprintf("JSON name %q cannot contain", tc.invalid))
		})
	}
}

func TestUnionValidateJSONWireKeys(t *testing.T) {
	cases := map[string]struct {
		typeKey  string
		valueKey string
		invalid  []string
	}{
		"default keys":     {},
		"custom keys":      {typeKey: "kind", valueKey: "data"},
		"newline key":      {typeKey: "ki\nnd"},
		"quote type key":   {typeKey: `ty"pe`, invalid: []string{`ty"pe`}},
		"comma value key":  {valueKey: "val,ue", invalid: []string{"val,ue"}},
		"backtick both":    {typeKey: "ty`pe", valueKey: "val`ue", invalid: []string{"ty`pe", "val`ue"}},
		"backslash, quote": {typeKey: `ty\pe`, valueKey: "it's", invalid: []string{`ty\pe`, "it's"}},
		"dash type key":    {typeKey: "-", invalid: []string{"-"}},
		"dash value key":   {valueKey: "-", invalid: []string{"-"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			union := &Union{TypeName: "U", TypeKey: tc.typeKey, ValueKey: tc.valueKey, Values: []*NamedAttributeExpr{{Name: "S", Attribute: &AttributeExpr{Type: String}}}}
			verr := (&AttributeExpr{Type: union}).Validate("ctx", nil)
			require.Len(t, verr.Errors, len(tc.invalid))
			for i, key := range tc.invalid {
				want := fmt.Sprintf("JSON name %q cannot contain", key)
				if key == "-" {
					want = `OneOf discriminator JSON name "-" would omit the member from JSON`
				}
				require.Contains(t, verr.Errors[i].Error(), want)
			}
		})
	}
}

func TestAttributeExprValidateStructTagKeys(t *testing.T) {
	cases := map[string]struct {
		key     string
		invalid bool
	}{
		"plain key":          {key: "db"},
		"json override":      {key: "json"},
		"json name":          {key: "json:name"},
		"dashed key":         {key: "x-custom"},
		"empty key":          {key: "", invalid: true},
		"space in key":       {key: "bad key", invalid: true},
		"colon in key":       {key: "a:b", invalid: true},
		"quote in key":       {key: `a"b`, invalid: true},
		"control char":       {key: "a\tb", invalid: true},
		"delete char in key": {key: "a\x7fb", invalid: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			attribute := &AttributeExpr{Type: String, Meta: MetaExpr{"struct:tag:" + tc.key: {"value"}}}
			verr := attribute.Validate("ctx", nil)
			if !tc.invalid {
				require.Empty(t, verr.Errors)
				return
			}
			require.Len(t, verr.Errors, 1)
			require.Contains(t, verr.Errors[0].Error(), fmt.Sprintf("ctx - struct tag key %q in metadata %q must be a non-empty run of", tc.key, "struct:tag:"+tc.key))
		})
	}
}
