package codegen

import (
	"net/url"
	"testing"
	"time"

	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestOpaqueCompositeDefaultMaterialization(t *testing.T) {
	cases := []struct {
		name    string
		raw     any
		meta    []string
		want    string
		invalid bool
	}{
		{"imported slice", []url.Values{{"q": {"one"}}}, []string{"[]url.Values", "net/url"}, `[]url.Values{url.Values{"q": []string{"one"}}}`, false},
		{"aliased slice", []url.Values{{"q": {"one"}}}, []string{"[]web.Values", "net/url", "web"}, `[]web.Values{web.Values{"q": []string{"one"}}}`, false},
		{"empty imported slice", []url.Values{}, []string{"[]url.Values", "net/url"}, `[]url.Values{}`, false},
		{"nil aliased slice", []url.Values(nil), []string{"[]web.Values", "net/url", "web"}, `[]web.Values(nil)`, false},
		{"imported map", map[string]url.Values{"a": {"q": {"one"}}}, []string{"map[string]url.Values", "net/url"}, `map[string]url.Values{"a": url.Values{"q": []string{"one"}}}`, false},
		{"aliased map", map[string]url.Values{"a": {"q": {"one"}}}, []string{"map[string]web.Values", "net/url", "web"}, `map[string]web.Values{"a": web.Values{"q": []string{"one"}}}`, false},
		{"empty aliased map", map[string]url.Values{}, []string{"map[string]web.Values", "net/url", "web"}, `map[string]web.Values{}`, false},
		{"nil imported map", map[string]url.Values(nil), []string{"map[string]url.Values", "net/url"}, `map[string]url.Values(nil)`, false},
		{"imported array", [1]url.Values{{"q": {"one"}}}, []string{"[1]url.Values", "net/url"}, `[1]url.Values{url.Values{"q": []string{"one"}}}`, false},
		{"aliased array", [1]url.Values{{"q": {"one"}}}, []string{"[1]web.Values", "net/url", "web"}, `[1]web.Values{web.Values{"q": []string{"one"}}}`, false},
		{"empty imported array", [0]url.Values{}, []string{"[0]url.Values", "net/url"}, `[0]url.Values{}`, false},
		{"undeclared structural alias", []url.Values{}, []string{"[]web.Values", "net/url"}, "", true},
		{"structural wrong import", []url.Values{}, []string{"[]url.Values", "other/url"}, "", true},
		{"builtin bytes", []byte{1, 2}, []string{"[]byte"}, `[]byte{0x1, 0x2}`, false},
		{"builtin composite", map[string][]string{"q": {"one"}}, []string{"map[string][]string"}, `map[string][]string{"q": []string{"one"}}`, false},
		{"typed empty", url.Values{}, []string{"url.Values", "net/url"}, `url.Values{}`, false},
		{"typed nil", url.Values(nil), []string{"url.Values", "net/url"}, `url.Values(nil)`, false},
		{"private pointed state", url.User("name"), []string{"*url.Userinfo", "net/url"}, "", true},
		{"typed", url.Values{"q": {"one", "two"}}, []string{"url.Values", "net/url"}, `url.Values{"q": []string{"one", "two"}}`, false},
		{"import alias", url.Values{"q": {"one"}}, []string{"web.Values", "net/url", "web"}, `web.Values{"q": []string{"one"}}`, false},
		{"untyped", map[string]any{"q": []any{"one"}}, []string{"url.Values", "net/url"}, "", true},
		{"assignable but unproven", map[string][]string{"q": {"one"}}, []string{"url.Values", "net/url"}, "", true},
		{"wrong import", url.Values{}, []string{"url.Values", "other/url"}, "", true},
		{"unknown alias", url.Values{}, []string{"url.Query", "net/url"}, "", true},
		{"private state", time.Unix(0, 0), []string{"time.Time", "time"}, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			field := &expr.AttributeExpr{Type: expr.Any, Meta: expr.MetaExpr{"struct:field:type": tc.meta}}
			target := &expr.AttributeExpr{Type: &expr.Object{{Name: "query", Attribute: field}}}
			ctx := NewAttributeContext(false, false, true, "", NewNameScope())
			plan, err := ctx.DefaultValue(target, map[string]any{"query": tc.raw})
			if tc.invalid {
				require.ErrorContains(t, err, "default.query: unsupported opaque default materialization")
				return
			}
			require.NoError(t, err)
			require.Contains(t, plan.Expression, "Query: "+tc.want)
		})
	}
}
