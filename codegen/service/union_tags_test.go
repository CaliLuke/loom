package service

import (
	json "encoding/json/v2"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnionJSONBodiesQuoteDiscriminatorKeys(t *testing.T) {
	cases := map[string]struct {
		typeKey  string
		valueKey string
	}{
		"default keys":    {typeKey: "type", valueKey: "value"},
		"backtick keys":   {typeKey: "ty`pe", valueKey: "val`ue"},
		"quote keys":      {typeKey: `ty"pe`, valueKey: `val"ue`},
		"backslash, line": {typeKey: `ty\pe`, valueKey: "val\nue"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			data := &UnionTypeData{Name: "U", KindName: "UKind", TypeKey: tc.typeKey, ValueKey: tc.valueKey}
			for body, render := range map[string]func(*UnionTypeData) string{
				"marshal":   renderUnionMarshalJSONBody,
				"unmarshal": renderUnionUnmarshalJSONBody,
			} {
				tags := parseStructTagsInBody(t, render(data))
				require.Len(t, tags, 2, body)
				got, ok := tags[0].Lookup("json")
				require.True(t, ok, body)
				require.Equal(t, tc.typeKey, got, body)
				got, ok = tags[1].Lookup("json")
				require.True(t, ok, body)
				require.Equal(t, tc.valueKey, got, body)
			}
		})
	}
}

func TestUnionJSONBodiesDiscriminatorRoundTrip(t *testing.T) {
	cases := map[string]struct {
		typeKey  string
		valueKey string
	}{
		"default keys":  {typeKey: "type", valueKey: "value"},
		"space keys":    {typeKey: "the type", valueKey: "the value"},
		"newline keys":  {typeKey: "ty\npe", valueKey: "val\nue"},
		"colon unicode": {typeKey: "kind:日本", valueKey: "data:ü"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			data := &UnionTypeData{Name: "U", KindName: "UKind", TypeKey: tc.typeKey, ValueKey: tc.valueKey}
			for body, render := range map[string]func(*UnionTypeData) string{
				"marshal":   renderUnionMarshalJSONBody,
				"unmarshal": renderUnionUnmarshalJSONBody,
			} {
				tags := parseStructTagsInBody(t, render(data))
				require.Len(t, tags, 2, body)
				typ := reflect.StructOf([]reflect.StructField{
					{Name: "Type", Type: reflect.TypeFor[string](), Tag: tags[0]},
					{Name: "Value", Type: reflect.TypeFor[string](), Tag: tags[1]},
				})
				value := reflect.New(typ)
				value.Elem().Field(0).SetString("t")
				value.Elem().Field(1).SetString("v")
				encoded, err := json.Marshal(value.Interface(), json.Deterministic(true))
				require.NoError(t, err, body)
				var members map[string]string
				require.NoError(t, json.Unmarshal(encoded, &members), body)
				require.Equal(t, map[string]string{tc.typeKey: "t", tc.valueKey: "v"}, members, body)
				decoded := reflect.New(typ)
				require.NoError(t, json.Unmarshal(encoded, decoded.Interface()), body)
				require.Equal(t, "t", decoded.Elem().Field(0).String(), body)
				require.Equal(t, "v", decoded.Elem().Field(1).String(), body)
			}
		})
	}
}

// parseStructTagsInBody parses a generated function body and returns the
// struct field tags it declares, in source order.
func parseStructTagsInBody(t *testing.T, body string) []reflect.StructTag {
	t.Helper()
	src := "package p\n\nfunc f(u *U, data []byte) ([]byte, error) {\n" + body + "\n}\n"
	file, err := parser.ParseFile(token.NewFileSet(), "p.go", src, 0)
	require.NoError(t, err, src)
	var tags []reflect.StructTag
	ast.Inspect(file, func(n ast.Node) bool {
		field, ok := n.(*ast.Field)
		if !ok || field.Tag == nil {
			return true
		}
		raw, err := strconv.Unquote(field.Tag.Value)
		require.NoError(t, err)
		tags = append(tags, reflect.StructTag(raw))
		return true
	})
	return tags
}
