package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

// TestAliasClashingImports checks that only the packages whose names clash
// with other imports of a file take an alias.
func TestAliasClashingImports(t *testing.T) {
	const (
		security = "example.com/app/gen/types/security"
		log      = "example.com/app/gen/types/log"
		common   = "example.com/app/gen/types/common"
		other    = "example.com/app/gen/other/common"
		sec2     = "example.com/app/gen/types/security2"
	)
	framework := []*ImportSpec{
		{Path: "context"},
		{Path: "mime/multipart"},
		{Path: "encoding/json/v2", Name: "json"},
		{Path: "github.com/gorilla/websocket"},
		LoomImport(""),
		LoomImport("security"),
	}
	cases := []struct {
		Name    string
		Imports []*ImportSpec
		Paths   []string
		Want    map[string]string
	}{
		{
			Name:    "no-clash",
			Imports: append(framework, &ImportSpec{Name: "log", Path: log}, &ImportSpec{Name: "common", Path: common}),
			Paths:   []string{common, log},
			Want:    map[string]string{},
		},
		{
			Name:    "clash-with-unnamed-import",
			Imports: append(framework, &ImportSpec{Name: "security", Path: security}),
			Paths:   []string{security},
			Want:    map[string]string{security: "security2"},
		},
		{
			Name:    "clash-with-named-import",
			Imports: append(framework, &ImportSpec{Name: "loom", Path: "example.com/app/gen/types/loom"}),
			Paths:   []string{"example.com/app/gen/types/loom"},
			Want:    map[string]string{"example.com/app/gen/types/loom": "loom2"},
		},
		{
			Name:    "clash-with-inferred-name",
			Imports: append(framework, &ImportSpec{Name: "websocket", Path: "example.com/app/gen/types/websocket"}),
			Paths:   []string{"example.com/app/gen/types/websocket"},
			Want:    map[string]string{"example.com/app/gen/types/websocket": "websocket2"},
		},
		{
			Name:    "alias-skips-used-names",
			Imports: append(framework, &ImportSpec{Name: "security", Path: security}, &ImportSpec{Name: "security2", Path: sec2}),
			Paths:   []string{security, sec2},
			Want:    map[string]string{security: "security3"},
		},
		{
			Name:    "user-type-packages-sharing-a-name",
			Imports: []*ImportSpec{{Path: "context"}, {Name: "common", Path: other}, {Name: "common", Path: common}},
			Paths:   []string{other, common},
			Want:    map[string]string{common: "common2"},
		},
		{
			Name:    "absent-path",
			Imports: framework,
			Paths:   []string{security},
			Want:    map[string]string{},
		},
		{
			Name:    "duplicate-import",
			Imports: append(framework, &ImportSpec{Name: "security", Path: security}, &ImportSpec{Name: "security", Path: security}),
			Paths:   []string{security, security},
			Want:    map[string]string{security: "security2"},
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			assert.Equal(t, c.Want, AliasClashingImports(c.Imports, c.Paths))
		})
	}
}

// TestNameScopePackageNames checks that the type references and definitions
// of a scope created with package names qualify the types of the renamed
// struct:pkg:path packages with their names at any depth, and that the other
// scopes keep the package names.
func TestNameScopePackageNames(t *testing.T) {
	item := &expr.UserTypeExpr{
		AttributeExpr: &expr.AttributeExpr{
			Type: &expr.Object{{Name: "id", Attribute: &expr.AttributeExpr{Type: expr.String}}},
			Meta: expr.MetaExpr{"struct:pkg:path": []string{"types/security"}},
		},
		TypeName: "Item",
		UID:      "item",
	}
	wrapper := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "item", Attribute: &expr.AttributeExpr{Type: item}},
		{Name: "items", Attribute: &expr.AttributeExpr{Type: &expr.Map{
			KeyType:  &expr.AttributeExpr{Type: expr.String},
			ElemType: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: item}}},
		}}},
	}}
	names := map[string]string{"types/security": "security2"}
	nested := &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: item}}}
	cases := []struct {
		Name                string
		Scope               *NameScope
		Package, Ref, Array string
		Def                 string
	}{
		{Name: "package-names", Scope: NewNameScope(), Package: "security", Ref: "*security.Item", Array: "[]*security.Item", Def: "security.Item"},
		{Name: "renamed", Scope: NewNameScopeWithPackageNames(names), Package: "security2", Ref: "*security2.Item", Array: "[]*security2.Item", Def: "security2.Item"},
		{Name: "like-renamed", Scope: NewNameScopeLike(NewNameScopeWithPackageNames(names)), Package: "security2", Ref: "*security2.Item", Array: "[]*security2.Item", Def: "security2.Item"},
		{Name: "like-nil", Scope: NewNameScopeLike(nil), Package: "security", Ref: "*security.Item", Array: "[]*security.Item", Def: "security.Item"},
		{Name: "without-package-names", Scope: NewNameScopeWithPackageNames(names).WithoutPackageNames(), Package: "security", Ref: "*security.Item", Array: "[]*security.Item", Def: "security.Item"},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			loc := UserTypeLocation(item)
			assert.Equal(t, c.Package, c.Scope.PackageName(loc))
			assert.Equal(t, c.Ref, c.Scope.GoFullTypeRef(&expr.AttributeExpr{Type: item}, c.Scope.PackageName(loc)))
			assert.Equal(t, c.Array, c.Scope.GoFullTypeRef(nested, "svc"))
			assert.Equal(t, c.Array, c.Scope.GoValueTypeName(nested))
			def := c.Scope.GoTypeDef(wrapper, false, false)
			assert.Contains(t, def, "Item *"+c.Def)
			assert.Contains(t, def, "Items map[string][]*"+c.Def)
		})
	}
	assert.Empty(t, NewNameScope().PackageName(nil))
	var nilScope *NameScope
	assert.Equal(t, "security", nilScope.PackageName(UserTypeLocation(item)))
}

// TestNameScopeWithoutPackageNamesSharesNames checks that the scope returned
// by WithoutPackageNames allocates names in the scope it is derived from.
func TestNameScopeWithoutPackageNamesSharesNames(t *testing.T) {
	scope := NewNameScopeWithPackageNames(map[string]string{"types/security": "security2"})
	require.Equal(t, "Item", scope.WithoutPackageNames().Unique("Item"))
	assert.Equal(t, "Item2", scope.Unique("Item"))
}

// TestAttributeContextPkgUsesScopePackageNames checks that transforms qualify
// the types of a renamed struct:pkg:path package with its name.
func TestAttributeContextPkgUsesScopePackageNames(t *testing.T) {
	item := &expr.UserTypeExpr{
		AttributeExpr: &expr.AttributeExpr{
			Type: &expr.Object{{Name: "id", Attribute: &expr.AttributeExpr{Type: expr.String}}},
			Meta: expr.MetaExpr{"struct:pkg:path": []string{"types/security"}},
		},
		TypeName: "Item",
		UID:      "item",
	}
	att := &expr.AttributeExpr{Type: item}
	renamed := NewAttributeContext(false, false, true, "svc", NewNameScopeWithPackageNames(map[string]string{"types/security": "security2"}))
	assert.Equal(t, "security2", renamed.Pkg(att))
	assert.Equal(t, "security", NewAttributeContext(false, false, true, "svc", NewNameScope()).Pkg(att))
	assert.Equal(t, "security", renamed.NamePkg(att))
	assert.Equal(t, "svc", renamed.NamePkg(&expr.AttributeExpr{Type: expr.String}))
	source := NewAttributeContext(false, false, true, "svc", NewNameScope())
	name := transformHelperName(att, att, &TransformAttrs{SourceCtx: source, TargetCtx: renamed})
	assert.Equal(t, "transformSecurityItemToSecurityItem", name, "helper names do not depend on package aliases")
}

// TestInitStructFieldsPackageNames checks that the conversions of aliased
// primitive fields qualify the types of struct:pkg:path packages with the
// names of the given scope.
func TestInitStructFieldsPackageNames(t *testing.T) {
	code := &expr.UserTypeExpr{
		AttributeExpr: &expr.AttributeExpr{
			Type: expr.String,
			Meta: expr.MetaExpr{"struct:pkg:path": []string{"types/strconv"}},
		},
		TypeName: "Code",
		UID:      "code",
	}
	args := []*InitArgData{{Name: "code", Type: expr.String, FieldName: "Code", FieldType: code}}
	cases := []struct {
		Name  string
		Scope *NameScope
		Want  string
	}{
		{Name: "package-names", Want: "v.Code = strconv.Code(code)"},
		{Name: "renamed", Scope: NewNameScopeWithPackageNames(map[string]string{"types/strconv": "strconv2"}), Want: "v.Code = strconv2.Code(code)"},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			got, _, err := InitStructFields(args, "v", "", "svc", c.Scope)
			require.NoError(t, err)
			assert.Contains(t, got, c.Want)
		})
	}
}

// TestFinalizeGoSourceAliasesRegexpOnClash checks that the regexp import of
// compiled pattern variables takes an alias when another import of the file,
// such as a struct:pkg:path package, is named regexp, and keeps its name
// otherwise.
func TestFinalizeGoSourceAliasesRegexpOnClash(t *testing.T) {
	const validate = `
func validate(a string) (err error) {
	err = loom.MergeErrors(err, loom.ValidatePattern("a", a, "^[a]+$"))
	return err
}
`
	cases := []struct {
		Name    string
		Imports string
		Wants   []string
	}{
		{
			Name:    "clash",
			Imports: "import (\n\tregexp \"example.com/app/gen/types/regexp\"\n\tloom \"github.com/CaliLuke/loom/pkg\"\n)\n\nvar _ *regexp.Item\n",
			Wants:   []string{`regexp2 "regexp"`, `regexp "example.com/app/gen/types/regexp"`, `var loomPatternGenerated0 = regexp2.MustCompile("^[a]+$")`},
		},
		{
			Name:    "clash-with-used-name",
			Imports: "import (\n\tregexp \"example.com/app/gen/types/regexp\"\n\tregexp2 \"example.com/app/gen/types/regexp2\"\n\tloom \"github.com/CaliLuke/loom/pkg\"\n)\n\nvar _ *regexp.Item\n\nvar _ regexp2.Other\n",
			Wants:   []string{`regexp3 "regexp"`, `regexp3.MustCompile`},
		},
		{
			Name:    "regexp-already-imported",
			Imports: "import (\n\tre \"regexp\"\n\tloom \"github.com/CaliLuke/loom/pkg\"\n)\n\nvar _ = re.QuoteMeta\n",
			Wants:   []string{`re "regexp"`, `re.MustCompile`},
		},
		{
			Name:    "no-clash",
			Imports: "import loom \"github.com/CaliLuke/loom/pkg\"\n",
			Wants:   []string{"\t\"regexp\"", `regexp.MustCompile`},
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			finalized, err := finalizeGoSource("generated.go", []byte("package generated\n\n"+c.Imports+validate))
			require.NoError(t, err)
			for _, want := range c.Wants {
				assert.Contains(t, string(finalized), want)
			}
		})
	}
}
