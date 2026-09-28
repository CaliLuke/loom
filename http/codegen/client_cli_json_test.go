package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
)

func TestContainsBooleanMapKeys(t *testing.T) {
	booleanMap := &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.Boolean}, ElemType: &expr.AttributeExpr{Type: expr.String}}
	alias := &expr.UserTypeExpr{TypeName: "Flags", AttributeExpr: &expr.AttributeExpr{Type: booleanMap}}
	keyAlias := &expr.UserTypeExpr{TypeName: "Flag", AttributeExpr: &expr.AttributeExpr{Type: expr.Boolean}}
	recursive := &expr.UserTypeExpr{TypeName: "Node", AttributeExpr: &expr.AttributeExpr{}}
	recursive.Type = &expr.Object{{Name: "next", Attribute: &expr.AttributeExpr{Type: recursive}}}
	for _, tc := range []struct {
		name     string
		datatype expr.DataType
		want     bool
	}{
		{"primitive", expr.Boolean, false},
		{"string-keys", &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.Boolean}}, false},
		{"boolean-keys", booleanMap, true},
		{"array", &expr.Array{ElemType: &expr.AttributeExpr{Type: booleanMap}}, true},
		{"alias", alias, true},
		{"named-key", &expr.Map{KeyType: &expr.AttributeExpr{Type: keyAlias}, ElemType: &expr.AttributeExpr{Type: expr.String}}, true},
		{"recursive-without-map", recursive, false},
		{"recursive-with-map", &expr.Object{{Name: "tree", Attribute: &expr.AttributeExpr{Type: recursive}}, {Name: "flags", Attribute: &expr.AttributeExpr{Type: booleanMap}}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, containsBooleanMapKeys(tc.datatype))
		})
	}
}

func TestCLIJSONImportAliasReservedForRequestVariables(t *testing.T) {
	for _, tc := range []struct {
		name    string
		service string
		imports []*codegen.ImportSpec
		want    string
	}{
		{name: "base", service: "svc", want: "loomhttpcli"},
		{name: "service", service: "loomhttpcli", want: "loomhttpcli2"},
		{name: "user type", service: "svc", imports: []*codegen.ImportSpec{{Path: "example.com/custom", Name: "loomhttpcli"}}, want: "loomhttpcli2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := &ServiceData{Service: &service.Data{PkgName: tc.service, PathName: tc.service, UserTypeImports: tc.imports}}
			require.Equal(t, tc.want, cliJSONPackageName(data))
			allocated, _ := newRequestVarScope(data).allocate(tc.want, nil)
			require.NotEqual(t, tc.want, allocated)
		})
	}
}
