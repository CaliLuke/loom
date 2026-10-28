package service

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestEndpointRequiredScopeOwnership(t *testing.T) {
	root := codegen.RunDSL(t, testdata.SecurityRequiredScopesDSL)
	service := NewServicesData(root).Get("scopes")
	for _, tc := range []struct {
		method string
		want   map[string][][]string
	}{
		{"isolated", map[string][][]string{"oauth": {{"shared"}}, "second": {{"oauth"}}, "key": {{"admin"}}}},
		{"compound", map[string][][]string{"scope_basic": {{"shared", "basic"}}, "key": {{"shared", "key"}}, "scope_jwt": {{"shared", "read"}}, "oauth": {{"shared", "oauth"}}}},
		{"alternative", map[string][][]string{"scope_jwt": {{"read"}, {"write"}}, "unscoped": {{}, {}}}},
	} {
		t.Run(tc.method, func(t *testing.T) {
			var method *EndpointMethodData
			for _, candidate := range endpointData(service).Methods {
				if candidate.Name == tc.method {
					method = candidate
				}
			}
			require.NotNil(t, method)
			source := codegen.SectionCode(t, endpointMethodSection(method))
			parsed, err := parser.ParseFile(token.NewFileSet(), "endpoint.go", "package scopes\n"+source, 0)
			require.NoError(t, err)
			got := map[string][][]string{}
			ast.Inspect(parsed, func(node ast.Node) bool {
				literal, ok := node.(*ast.CompositeLit)
				if !ok {
					return true
				}
				name := ""
				var scopes []string
				hasScopes := false
				for _, element := range literal.Elts {
					pair, ok := element.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					key, ok := pair.Key.(*ast.Ident)
					if !ok {
						continue
					}
					switch key.Name {
					case "Name":
						value, ok := pair.Value.(*ast.BasicLit)
						if ok {
							name, err = strconv.Unquote(value.Value)
							require.NoError(t, err)
						}
					case "RequiredScopes":
						hasScopes = true
						scopes = []string{}
						for _, item := range pair.Value.(*ast.CompositeLit).Elts {
							value, err := strconv.Unquote(item.(*ast.BasicLit).Value)
							require.NoError(t, err)
							scopes = append(scopes, value)
						}
					}
				}
				if hasScopes {
					got[name] = append(got[name], scopes)
				}
				return true
			})
			require.Equal(t, tc.want, got)
		})
	}
}

func TestRequiredScopeDataAlternativesAndCopy(t *testing.T) {
	root := codegen.RunDSL(t, testdata.SecurityRequiredScopesDSL)
	method := NewServicesData(root).Get("scopes").Method("alternative")
	require.NotNil(t, method)
	first := method.Requirements[0].Schemes[0]
	second := method.Requirements[1].Schemes[0]
	require.Equal(t, "scope_jwt", first.SchemeName)
	require.Equal(t, "scope_jwt", second.SchemeName)
	require.NotSame(t, first, second)
	require.Equal(t, []string{"read"}, first.RequiredScopes)
	require.Equal(t, []string{"write"}, second.RequiredScopes)
	copied := first.Dup()
	require.Equal(t, first.RequiredScopes, copied.RequiredScopes)
	copied.RequiredScopes[0] = "changed"
	require.Equal(t, []string{"read"}, first.RequiredScopes)
	first.RequiredScopes[0] = "changed"
	require.Equal(t, []string{"write"}, second.RequiredScopes)
}
