package expr

import (
	"fmt"
	"math"
	"strings"

	"github.com/CaliLuke/loom/eval"
)

// Validate tests whether the attribute required fields exist.  Since attributes
// are unaware of their context, additional context information can be provided
// to be used in error messages.  The parent definition context is automatically
// added to error messages.
func (a *AttributeExpr) Validate(ctx string, parent eval.Expression) *eval.ValidationErrors {
	if validated[a] {
		return nil
	}
	validated[a] = true
	verr := new(eval.ValidationErrors)
	if err := a.resolveTypeRef(); err != nil {
		verr.Add(parent, "%s", err.Error())
		return verr
	}
	if a.Type == nil {
		verr.Add(parent, "attribute type is nil")
		return verr
	}
	if ctx != "" {
		ctx += " - "
	}
	verr.Merge(a.validateEnumDefault(ctx, parent))
	if v := a.Validation; v != nil {
		verr.Merge(v.Validate(ctx, parent))
		verr.Merge(a.validateNumericBounds(ctx, parent))
	}
	verr.Merge(a.validateExamples(ctx, parent))
	verr.Merge(a.validatePresence(ctx, parent))
	verr.Merge(a.validateChildTypes(ctx, parent))
	verr.Merge(a.validateViewReference(ctx, parent))

	return verr
}

func (a *AttributeExpr) validatePresence(ctx string, parent eval.Expression) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	jsonTag, hasJSONTag := attributeJSONTag(a)
	if hasJSONTag && jsonTagHasOption(jsonTag, "string") {
		verr.Add(parent, "%sJSON ,string is not supported", ctx)
	}
	if AllowsNull(a) {
		if _, hasType := a.Meta["struct:field:type"]; hasType && !legacyNullableWrapper(a) {
			verr.Add(parent, "%snull-admitting attributes conflict with custom field type metadata", ctx)
		}
		if hasJSONTag && strings.Split(jsonTag, ",")[0] == "-" {
			verr.Add(parent, "%snull-admitting attributes conflict with a JSON tag that omits the field", ctx)
		}
	}
	if value, ok := a.Meta.Last("openapi:nullable"); ok && value == "false" && legacyNullableWrapper(a) {
		verr.Add(parent, "%snullable metadata conflicts with Loom Nullable field type", ctx)
	}
	if value, ok := a.Meta.Last("openapi:nullable"); ok && value == "true" {
		if _, hasType := a.Meta["struct:field:type"]; hasType && !legacyNullableWrapper(a) {
			verr.Add(parent, "%snullable metadata conflicts with custom field type", ctx)
		}
	}
	if array := AsArray(a.Type); array != nil && array.NonNullableElems && IsNullable(array.ElemType) {
		verr.Add(parent, "%sarray elements cannot be both nullable and required", ctx)
	}
	if mapping := AsMap(a.Type); mapping != nil && IsNullable(mapping.KeyType) {
		verr.Add(parent, "%smap keys cannot be nullable", ctx)
	}
	return verr
}

func (a *AttributeExpr) validateNumericBounds(ctx string, parent eval.Expression) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if !isIntegerKind(a.Type.Kind()) {
		return verr
	}
	for _, bound := range []*float64{
		a.Validation.Minimum,
		a.Validation.Maximum,
		a.Validation.ExclusiveMinimum,
		a.Validation.ExclusiveMaximum,
	} {
		if bound != nil && math.Trunc(*bound) != *bound {
			verr.Add(parent, "%sinteger bounds must be whole numbers", ctx)
			return verr
		}
	}
	return verr
}

func isIntegerKind(kind Kind) bool {
	switch kind {
	case IntKind, Int32Kind, Int64Kind, UIntKind, UInt32Kind, UInt64Kind:
		return true
	default:
		return false
	}
}

// Prepare resolves any deferred named type references before validation.
func (a *AttributeExpr) Prepare() {
	a.prepareTypeRefs(make(map[*AttributeExpr]struct{}))
	a.normalizeLegacyPresence(make(map[*AttributeExpr]struct{}))
	a.preparePkgPath(a.pkgPath(), a.Type, make(map[*AttributeExpr]struct{}))
}

func (a *AttributeExpr) normalizeLegacyPresence(seen map[*AttributeExpr]struct{}) {
	if a == nil {
		return
	}
	if _, ok := seen[a]; ok {
		return
	}
	seen[a] = struct{}{}
	if legacyNullableWrapper(a) {
		if value, ok := a.Meta.Last("openapi:nullable"); !ok || value != "false" {
			a.Nullable = true
			delete(a.Meta, "openapi:nullable")
			if legacyWrapperCanUseSemanticType(a) {
				delete(a.Meta, "struct:field:type")
			}
		}
	} else if value, ok := a.Meta.Last("openapi:nullable"); ok && value == "true" {
		a.Nullable = true
		delete(a.Meta, "openapi:nullable")
	}
	switch actual := a.Type.(type) {
	case UserType:
		actual.Attribute().normalizeLegacyPresence(seen)
	case *Array:
		actual.ElemType.normalizeLegacyPresence(seen)
	case *Map:
		actual.KeyType.normalizeLegacyPresence(seen)
		actual.ElemType.normalizeLegacyPresence(seen)
	case *Object:
		for _, named := range *actual {
			named.Attribute.normalizeLegacyPresence(seen)
		}
	case *Union:
		for _, named := range actual.Values {
			named.Attribute.normalizeLegacyPresence(seen)
		}
	}
}

func legacyNullableWrapper(a *AttributeExpr) bool {
	if a == nil {
		return false
	}
	args, ok := a.Meta["struct:field:type"]
	return ok && len(args) == 3 && args[1] == "github.com/CaliLuke/loom/pkg" && args[2] == "loom" &&
		strings.HasPrefix(args[0], "loom.Nullable[") && strings.HasSuffix(args[0], "]")
}

func legacyWrapperCanUseSemanticType(a *AttributeExpr) bool {
	args := a.Meta["struct:field:type"]
	if len(args) != 3 {
		return false
	}
	want := ""
	switch a.Type {
	case Boolean:
		want = "bool"
	case Int:
		want = "int"
	case Int32:
		want = "int32"
	case Int64:
		want = "int64"
	case UInt:
		want = "uint"
	case UInt32:
		want = "uint32"
	case UInt64:
		want = "uint64"
	case Float32:
		want = "float32"
	case Float64:
		want = "float64"
	case String:
		want = "string"
	case Bytes:
		want = "[]byte"
	case Any:
		want = "any"
	}
	return want != "" && args[0] == "loom.Nullable["+want+"]"
}

func (a *AttributeExpr) preparePkgPath(pkgPath string, t DataType, seen map[*AttributeExpr]struct{}) {
	switch actual := t.(type) {
	case *Array:
		a.preparePkgPath(pkgPath, actual.ElemType.Type, seen)
	case *Map:
		a.preparePkgPath(pkgPath, actual.KeyType.Type, seen)
		a.preparePkgPath(pkgPath, actual.ElemType.Type, seen)
	case *Union:
		for _, nat := range actual.Values {
			if nat == nil || nat.Attribute == nil {
				continue
			}
			a.preparePkgPath(pkgPath, nat.Attribute.Type, seen)
		}
	case *Object:
		for _, nat := range *actual {
			if nat == nil || nat.Attribute == nil {
				continue
			}
			a.preparePkgPath(pkgPath, nat.Attribute.Type, seen)
		}
	}
	ut, ok := t.(UserType)
	if pkgPath == "" || !ok {
		return
	}
	att := ut.Attribute()
	if att == nil {
		return
	}
	if _, done := seen[att]; done {
		return
	}
	seen[att] = struct{}{}
	if _, ok := att.Meta.Last("struct:pkg:path"); !ok {
		att.AddMeta("struct:pkg:path", pkgPath)
	}
	a.preparePkgPath(pkgPath, att.Type, seen)
}

func (a *AttributeExpr) validatePkgPath(pkgPath string, t DataType) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if ar := AsArray(t); ar != nil {
		verr.Merge(a.validatePkgPath(pkgPath, ar.ElemType.Type))
	}
	if mp := AsMap(t); mp != nil {
		verr.Merge(a.validatePkgPath(pkgPath, mp.KeyType.Type))
		verr.Merge(a.validatePkgPath(pkgPath, mp.ElemType.Type))
	}
	if u := AsUnion(t); u != nil {
		for _, nat := range u.Values {
			if nat == nil || nat.Attribute == nil {
				continue
			}
			verr.Merge(a.validatePkgPath(pkgPath, nat.Attribute.Type))
		}
	}
	if ut, ok := t.(UserType); pkgPath != "" && ok {
		// This check ensures we error if a sub-type has a different custom package type set
		// or if two user types have different custom packages but share a sub-type (field that's a user type)
		if ut.Attribute().Meta != nil &&
			ut.Attribute().Meta["struct:pkg:path"] != nil &&
			ut.Attribute().Meta["struct:pkg:path"][0] != pkgPath {
			verr.Add(a, "type \"%s\" has conflicting packages %s and %s", ut.Name(), ut.Attribute().Meta["struct:pkg:path"][0], pkgPath)
		}
	}
	if len(verr.Errors) > 0 {
		return verr
	}
	return nil
}

func (a *AttributeExpr) validateChildTypes(ctx string, parent eval.Expression) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if o := AsObject(a.Type); o != nil {
		verr.Merge(a.validateObjectChildren(ctx, parent, o))
		return verr
	}
	if ar := AsArray(a.Type); ar != nil {
		verr.Merge(ar.ElemType.Validate(ctx, a))
		return verr
	}
	if mapping := AsMap(a.Type); mapping != nil {
		verr.Merge(mapping.KeyType.Validate(ctx, a))
		verr.Merge(mapping.ElemType.Validate(ctx, a))
		return verr
	}
	if u := AsUnion(a.Type); u != nil {
		if u.Untagged {
			for _, branch := range u.Values {
				userType, named := branch.Attribute.Type.(UserType)
				if !named || IsAlias(userType) || !IsObject(userType) {
					verr.Add(parent, "%suntagged OneOf branch %q must be a concrete named object type", ctx, branch.Name)
					continue
				}
				if additional, ok := userType.Attribute().Meta.Last("openapi:additionalProperties"); ok && additional != "false" {
					verr.Add(parent, "%suntagged OneOf branch %q must use the default open object or openapi:additionalProperties false", ctx, branch.Name)
				}
				wireNames := make(map[string]struct{})
				for _, field := range *AsObject(userType) {
					if !isUntaggedBranchFieldType(field.Attribute.Type) {
						verr.Add(parent, "%suntagged OneOf branch %q field %q must be primitive, a concrete named object type, or an array of either", ctx, branch.Name, field.Name)
					}
					wireName := JSONFieldName(field.Name, field.Attribute)
					if wireName == "" {
						verr.Add(parent, "%suntagged OneOf branch %q field %q cannot use an empty JSON tag name", ctx, branch.Name, field.Name)
						continue
					}
					if wireName == "-" {
						verr.Add(parent, "%suntagged OneOf branch %q field %q cannot use json tag %q", ctx, branch.Name, field.Name, wireName)
						continue
					}
					if _, duplicate := wireNames[wireName]; duplicate {
						verr.Add(parent, "%suntagged OneOf branch %q has duplicate JSON field name %q", ctx, branch.Name, wireName)
					}
					wireNames[wireName] = struct{}{}
				}
			}
		}
		for _, ut := range u.Values {
			verr.Merge(ut.Attribute.Validate(ctx, parent))
		}
	}
	return verr
}

func isUntaggedBranchFieldType(dataType DataType) bool {
	if IsPrimitive(dataType) {
		return true
	}
	if userType, named := dataType.(UserType); named {
		return !IsAlias(userType) && IsObject(userType)
	}
	array := AsArray(dataType)
	return array != nil && isUntaggedBranchFieldType(array.ElemType.Type)
}

// JSONFieldName returns the effective JSON member name for an object field.
func JSONFieldName(name string, attribute *AttributeExpr) string {
	if attribute == nil {
		return name
	}
	if tag, ok := attributeJSONTag(attribute); ok {
		return strings.Split(tag, ",")[0]
	}
	if tag, ok := attribute.Meta["struct:tag:json:name"]; ok && len(tag) > 0 {
		return strings.Split(strings.Join(tag, ","), ",")[0]
	}
	return name
}

func attributeJSONTag(attribute *AttributeExpr) (string, bool) {
	if attribute == nil {
		return "", false
	}
	values, ok := attribute.Meta["struct:tag:json"]
	return strings.Join(values, ","), ok
}
func (a *AttributeExpr) validateObjectChildren(ctx string, parent eval.Expression, obj *Object) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	for _, name := range a.AllRequired() {
		if a.Find(name) == nil {
			verr.Add(parent, `%srequired field %q does not exist in type %s`, ctx, name, a.Type.Name())
		}
	}
	pkgPath := a.pkgPath()
	wireNames := make(map[string]string, len(*obj))
	designNames := make(map[string]struct{}, len(*obj))
	for _, nat := range *obj {
		designNames[nat.Name] = struct{}{}
	}
	for _, nat := range *obj {
		verr.Merge(a.validatePkgPath(pkgPath, nat.Attribute.Type))
		fieldCtx := fmt.Sprintf("field %s", nat.Name)
		verr.Merge(nat.Attribute.Validate(fieldCtx, parent))
		wireName := JSONFieldName(nat.Name, nat.Attribute)
		if wireName == "-" {
			continue
		}
		if wireName == "" {
			verr.Add(parent, "%s cannot use an empty JSON tag name", fieldCtx)
			continue
		}
		if _, conflicts := designNames[wireName]; conflicts && wireName != nat.Name {
			verr.Add(parent, "%s JSON field name %q conflicts with another design field name", fieldCtx, wireName)
			continue
		}
		if first, exists := wireNames[wireName]; exists {
			verr.Add(parent, "%s duplicates JSON field name %q from field %s", fieldCtx, wireName, first)
			continue
		}
		wireNames[wireName] = nat.Name
	}
	return verr
}

func jsonTagHasOption(tag, option string) bool {
	for _, candidate := range strings.Split(tag, ",")[1:] {
		if candidate == option {
			return true
		}
	}
	return false
}

func (a *AttributeExpr) pkgPath() string {
	if meta, ok := a.Meta.Last("struct:pkg:path"); ok {
		return meta
	}
	return attributePkgPath(a.Type)
}

func attributePkgPath(dt DataType) string {
	ut, ok := dt.(UserType)
	if !ok {
		return ""
	}
	meta, ok := ut.Attribute().Meta["struct:pkg:path"]
	if !ok || len(meta) == 0 {
		return ""
	}
	return meta[0]
}

func (a *AttributeExpr) validateViewReference(ctx string, parent eval.Expression) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	view, ok := a.Meta.Last(ViewMetaKey)
	if !ok {
		return verr
	}
	rt, ok := a.Type.(*ResultTypeExpr)
	if !ok {
		verr.Add(parent, "%s uses view %q but %q is not a result type", ctx, view, a.Type.Name())
		return verr
	}
	if view == DefaultView || resultTypeHasView(rt, view) {
		return verr
	}
	verr.Add(parent, "%s: type %q does not define view %q", ctx, a.Type.Name(), view)
	return verr
}

func resultTypeHasView(rt *ResultTypeExpr, view string) bool {
	for _, candidate := range rt.Views {
		if candidate.Name == view {
			return true
		}
	}
	return false
}
