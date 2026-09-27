package naming

import (
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

func TestGoifyAttribute(t *testing.T) {
	for _, tc := range []struct {
		name     string
		metadata map[string][]string
		upper    bool
		want     string
	}{
		{"foo_bar", nil, true, "FooBar"},
		{"foo_bar", nil, false, "fooBar"},
		{"foo_bar:x", nil, true, "FooBar"},
		{"foo_bar", map[string][]string{"struct:field:name": {}}, true, "FooBar"},
		{"foo_bar", map[string][]string{"struct:field:name": {"other_field", "ignored"}}, true, "OtherField"},
		{"foo_bar", map[string][]string{"struct:field:name": {"other_field"}}, false, "otherField"},
	} {
		require.Equal(t, tc.want, GoifyAttribute(tc.name, tc.metadata, tc.upper))
	}
}
