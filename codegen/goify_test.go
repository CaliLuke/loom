package codegen

import (
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixReservedGo(t *testing.T) {
	cases := map[string]struct {
		w    string
		want string
	}{
		"predeclared type":           {w: "bool", want: "bool_"},
		"predeclared constant":       {w: "true", want: "true_"},
		"predeclared zero value":     {w: "nil", want: "nil_"},
		"predeclared function":       {w: "append", want: "append_"},
		"non predeclared identifier": {w: "foo", want: "foo"},
		"package":                    {w: "fmt", want: "fmt_"},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			assert.Equal(t, tc.want, fixReservedGo(tc.w))
		})
	}
}

func TestGoifyExportedIdentifiers(t *testing.T) {
	cases := map[string]struct {
		name       string
		firstUpper bool
		want       string
	}{
		"titlecase first letter":     {name: "ǅemal", firstUpper: true, want: "Ǆemal"},
		"caseless script exported":   {name: "日本語", firstUpper: true, want: "Val日本語"},
		"caseless script unexported": {name: "日本語", firstUpper: false, want: "日本語"},
		"leading digit exported":     {name: "123foo", firstUpper: true, want: "Val123foo"},
		"leading digit unexported":   {name: "123foo", firstUpper: false, want: "val123foo"},
		"non ascii leading digit":    {name: "١٢٣", firstUpper: true, want: "Val١٢٣"},
		"ascii exported unchanged":   {name: "user_id", firstUpper: true, want: "UserID"},
		"reserved word unexported":   {name: "type", firstUpper: false, want: "type_"},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			got := Goify(tc.name, tc.firstUpper)
			require.Equal(t, tc.want, got)
			require.Equal(t, tc.firstUpper, token.IsExported(got))
		})
	}
}
