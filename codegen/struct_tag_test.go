package codegen

import (
	json "encoding/json/v2"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestAttributeTagsWithNameJSONRoundTrip(t *testing.T) {
	cases := map[string]struct {
		field string
		meta  expr.MetaExpr
		name  string
	}{
		"plain":          {field: "user_id", name: "user_id"},
		"space":          {field: "user id", name: "user id"},
		"newline":        {field: "user\nid", name: "user\nid"},
		"tab and colon":  {field: "a\tb:c", name: "a\tb:c"},
		"unicode":        {field: "日本語", name: "日本語"},
		"json name meta": {field: "x", meta: expr.MetaExpr{"struct:tag:json:name": {"new\nline"}}, name: "new\nline"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			att := &expr.AttributeExpr{Type: expr.String, Meta: tc.meta}
			parent := &expr.AttributeExpr{Type: &expr.Object{{Name: tc.field, Attribute: att}}}
			verr := parent.Validate("", nil)
			require.Empty(t, verr.Errors)
			tag, ok := parseRenderedStructTag(t, AttributeTagsWithName(parent, tc.field, att))
			require.True(t, ok)
			requireJSONFieldRoundTrip(t, tag, tc.name)
		})
	}
}

// requireJSONFieldRoundTrip marshals and unmarshals a one-field struct that
// carries tag with encoding/json/v2 and checks the member is named name.
func requireJSONFieldRoundTrip(t *testing.T, tag reflect.StructTag, name string) {
	t.Helper()
	typ := reflect.StructOf([]reflect.StructField{{Name: "F", Type: reflect.TypeFor[string](), Tag: tag}})
	value := reflect.New(typ)
	value.Elem().Field(0).SetString("v")
	encoded, err := json.Marshal(value.Interface(), json.Deterministic(true))
	require.NoError(t, err)
	quoted, err := json.Marshal(name)
	require.NoError(t, err)
	require.Equal(t, "{"+string(quoted)+`:"v"}`, string(encoded))
	decoded := reflect.New(typ)
	require.NoError(t, json.Unmarshal(encoded, decoded.Interface()))
	require.Equal(t, "v", decoded.Elem().Field(0).String())
}

func TestAttributeTagsWithNameQuotesDesignText(t *testing.T) {
	cases := map[string]struct {
		key      string
		value    string
		field    string
		rendered string
	}{
		"plain value keeps raw tag": {
			key: "db", value: "user_id", field: "id",
			rendered: " `db:\"user_id\" json:\"id,omitempty\"`",
		},
		"double quote in value": {
			key: "db", value: `quoted "value"`, field: "q",
			rendered: " `db:\"quoted \\\"value\\\"\" json:\"q,omitempty\"`",
		},
		"backslash in value": {
			key: "db", value: `back\slash`, field: "s",
			rendered: " `db:\"back\\\\slash\" json:\"s,omitempty\"`",
		},
		"newline in value": {
			key: "db", value: "new\nline", field: "n",
			rendered: " `db:\"new\\nline\" json:\"n,omitempty\"`",
		},
		"backtick in value": {
			key: "db", value: "back`tick", field: "b",
			rendered: ` "db:\"back` + "`" + `tick\" json:\"b,omitempty\""`,
		},
		// Design validation rejects the JSON names below; the renderer must still
		// emit Go that compiles if it is ever called with them.
		"backtick in field name": {
			key: "db", value: "v", field: "field`name",
			rendered: ` "db:\"v\" json:\"field` + "`" + `name,omitempty\""`,
		},
		"quote in field name": {
			key: "db", value: "v", field: `field"name`,
			rendered: " `db:\"v\" json:\"field\\\"name,omitempty\"`",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			att := &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:tag:" + tc.key: {tc.value}}}
			parent := &expr.AttributeExpr{Type: &expr.Object{{Name: tc.field, Attribute: att}}}
			rendered := AttributeTagsWithName(parent, tc.field, att)
			require.Equal(t, tc.rendered, rendered)
			tag, ok := parseRenderedStructTag(t, rendered)
			require.True(t, ok)
			got, found := tag.Lookup(tc.key)
			require.True(t, found)
			require.Equal(t, tc.value, got)
			got, found = tag.Lookup("json")
			require.True(t, found)
			require.Equal(t, tc.field+",omitempty", got)
		})
	}
}

// parseRenderedStructTag compiles a struct field carrying the rendered tag
// and returns its value as seen by the Go parser.
func parseRenderedStructTag(t *testing.T, rendered string) (reflect.StructTag, bool) {
	t.Helper()
	src := "package p\n\ntype T struct {\n\tF string" + rendered + "\n}\n"
	file, err := parser.ParseFile(token.NewFileSet(), "p.go", src, 0)
	if err != nil {
		t.Errorf("rendered tag %q does not compile: %v", rendered, err)
		return "", false
	}
	field := file.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec).Type.(*ast.StructType).Fields.List[0]
	if field.Tag == nil {
		return "", rendered == ""
	}
	raw, err := strconv.Unquote(field.Tag.Value)
	if err != nil {
		t.Errorf("rendered tag %q is not a Go string literal: %v", rendered, err)
		return "", false
	}
	return reflect.StructTag(raw), true
}
