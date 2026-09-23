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

func TestAttributeTagsJSONRoundTrip(t *testing.T) {
	cases := map[string]string{
		"plain":         "user_id",
		"space":         "user id",
		"newline":       "user\nid",
		"tab and colon": "a\tb:c",
		"unicode":       "日本語",
	}
	for name, field := range cases {
		t.Run(name, func(t *testing.T) {
			rendered := attributeTags(&expr.AttributeExpr{Type: expr.String}, field, false, false)
			src := "package p\n\ntype T struct {\n\tF string" + rendered + "\n}\n"
			file, err := parser.ParseFile(token.NewFileSet(), "p.go", src, 0)
			require.NoError(t, err, rendered)
			raw, err := strconv.Unquote(file.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec).Type.(*ast.StructType).Fields.List[0].Tag.Value)
			require.NoError(t, err)
			typ := reflect.StructOf([]reflect.StructField{{Name: "F", Type: reflect.TypeFor[string](), Tag: reflect.StructTag(raw)}})
			value := reflect.New(typ)
			value.Elem().Field(0).SetString("v")
			encoded, err := json.Marshal(value.Interface(), json.Deterministic(true))
			require.NoError(t, err)
			quoted, err := json.Marshal(field)
			require.NoError(t, err)
			require.Equal(t, "{"+string(quoted)+`:"v"}`, string(encoded))
			decoded := reflect.New(typ)
			require.NoError(t, json.Unmarshal(encoded, decoded.Interface()))
			require.Equal(t, "v", decoded.Elem().Field(0).String())
		})
	}
}

func TestAttributeTagsQuoteDesignText(t *testing.T) {
	cases := map[string]struct {
		att  *expr.AttributeExpr
		name string
		want map[string]string
	}{
		"plain name": {
			att:  &expr.AttributeExpr{Type: expr.String},
			name: "user_id",
			want: map[string]string{"form": "user_id,omitempty", "json": "user_id,omitempty", "xml": "user_id,omitempty"},
		},
		"backtick in name": {
			att:  &expr.AttributeExpr{Type: expr.String},
			name: "user`id",
			want: map[string]string{"form": "user`id,omitempty", "json": "user`id,omitempty", "xml": "user`id,omitempty"},
		},
		"quote in name": {
			att:  &expr.AttributeExpr{Type: expr.String},
			name: `user"id`,
			want: map[string]string{"form": `user"id,omitempty`, "json": `user"id,omitempty`, "xml": `user"id,omitempty`},
		},
		"quote and backtick in custom tag": {
			att:  &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:tag:db": {"a\"b`c"}}},
			name: "id",
			want: map[string]string{"db": "a\"b`c"},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			rendered := attributeTags(tc.att, tc.name, true, false)
			src := "package p\n\ntype T struct {\n\tF string" + rendered + "\n}\n"
			file, err := parser.ParseFile(token.NewFileSet(), "p.go", src, 0)
			require.NoError(t, err, rendered)
			field := file.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec).Type.(*ast.StructType).Fields.List[0]
			require.NotNil(t, field.Tag)
			raw, err := strconv.Unquote(field.Tag.Value)
			require.NoError(t, err)
			for key, want := range tc.want {
				got, ok := reflect.StructTag(raw).Lookup(key)
				require.True(t, ok, "tag %s lacks key %s", rendered, key)
				require.Equal(t, want, got, "tag %s key %s", rendered, key)
			}
		})
	}
}
