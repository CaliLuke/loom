//nolint:errcheck // Generator helpers write only to in-memory buffers/builders.
package codegen

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func transformAttributeHelpers(source, target *expr.AttributeExpr, ta *transformAttrs, seen map[string]*codegen.TransformFunctionData) ([]*codegen.TransformFunctionData, error) {
	source, target, err := compatibleTransformAttributes(source, target, ta)
	if err != nil {
		return nil, err
	}
	switch {
	case expr.IsArray(source.Type):
		return transformArrayAttributeHelpers(source, target, ta, seen)
	case expr.IsMap(source.Type):
		return transformMapAttributeHelpers(source, target, ta, seen)
	case expr.IsUnion(source.Type):
		return transformUnionAttributeHelpers(source, target, ta, seen)
	case expr.IsObject(source.Type):
		return transformObjectAttributeHelpers(source, target, ta, seen)
	default:
		return nil, nil
	}
}

func compatibleTransformAttributes(source, target *expr.AttributeExpr, ta *transformAttrs) (*expr.AttributeExpr, *expr.AttributeExpr, error) {
	if err := codegen.IsCompatible(source.Type, target.Type, "", ""); err == nil {
		return source, target, nil
	}
	if ta.proto {
		target = unwrapAttr(expr.DupAtt(target))
	} else {
		source = unwrapAttr(expr.DupAtt(source))
	}
	if err := codegen.IsCompatible(source.Type, target.Type, "", ""); err != nil {
		return nil, nil, err
	}
	return source, target, nil
}

func transformArrayAttributeHelpers(source, target *expr.AttributeExpr, ta *transformAttrs, seen map[string]*codegen.TransformFunctionData) ([]*codegen.TransformFunctionData, error) {
	return transformAttributeHelpers(expr.AsArray(source.Type).ElemType, expr.AsArray(target.Type).ElemType, ta, seen)
}

func transformMapAttributeHelpers(source, target *expr.AttributeExpr, ta *transformAttrs, seen map[string]*codegen.TransformFunctionData) ([]*codegen.TransformFunctionData, error) {
	sm := expr.AsMap(source.Type)
	tm := expr.AsMap(target.Type)
	helpers, err := transformAttributeHelpers(sm.ElemType, tm.ElemType, ta, seen)
	if err != nil {
		return nil, err
	}
	other, err := transformAttributeHelpers(sm.KeyType, tm.KeyType, ta, seen)
	if err != nil {
		return nil, err
	}
	return append(helpers, other...), nil
}

func transformUnionAttributeHelpers(source, target *expr.AttributeExpr, ta *transformAttrs, seen map[string]*codegen.TransformFunctionData) ([]*codegen.TransformFunctionData, error) {
	srcAttrs := expr.AsUnion(source.Type)
	tgtAttrs := expr.AsUnion(target.Type)
	if len(srcAttrs.Values) != len(tgtAttrs.Values) {
		return nil, fmt.Errorf("cannot transform union attribute %s with %d types to union attribute %s with %d types",
			source.Type.Name(), len(srcAttrs.Values), target.Type.Name(), len(tgtAttrs.Values))
	}
	helpers := []*codegen.TransformFunctionData{}
	for i, srcAtt := range srcAttrs.Values {
		h, err := collectHelpers(srcAtt.Attribute, tgtAttrs.Values[i].Attribute, true, ta, seen)
		if err != nil {
			return nil, err
		}
		helpers = append(helpers, h...)
	}
	return helpers, nil
}

func transformObjectAttributeHelpers(source, target *expr.AttributeExpr, ta *transformAttrs, seen map[string]*codegen.TransformFunctionData) ([]*codegen.TransformFunctionData, error) {
	helpers := []*codegen.TransformFunctionData{}
	var walkErr error
	walkMatches(source, target, func(srcMatt, _ *expr.MappedAttributeExpr, srcc, tgtc *expr.AttributeExpr, n string) {
		if walkErr != nil {
			return
		}
		srcc, tgtc, walkErr = compatibleTransformAttributes(srcc, tgtc, ta)
		if walkErr != nil {
			return
		}
		var current []*codegen.TransformFunctionData
		current, walkErr = collectHelpers(srcc, tgtc, srcMatt.IsRequired(n), ta, seen)
		if walkErr != nil {
			return
		}
		helpers = append(helpers, current...)
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return helpers, nil
}

// collectHelpers recursively traverses the given attributes and return the
// transform helper functions required to generate the transform code.
func collectHelpers(source, target *expr.AttributeExpr, req bool, ta *transformAttrs, seen map[string]*codegen.TransformFunctionData) ([]*codegen.TransformFunctionData, error) {
	switch {
	case expr.IsArray(source.Type):
		return transformAttributeHelpers(expr.AsArray(source.Type).ElemType, expr.AsArray(target.Type).ElemType, ta, seen)
	case expr.IsMap(source.Type):
		return collectMapHelpers(source, target, ta, seen)
	case expr.IsUnion(source.Type):
		return collectUnionHelpers(source, target, ta, seen)
	case expr.IsObject(source.Type):
		return collectObjectHelpers(source, target, req, ta, seen)
	}
	return nil, nil
}

func collectMapHelpers(source, target *expr.AttributeExpr, ta *transformAttrs, seen map[string]*codegen.TransformFunctionData) ([]*codegen.TransformFunctionData, error) {
	data, err := transformAttributeHelpers(expr.AsMap(source.Type).KeyType, expr.AsMap(target.Type).KeyType, ta, seen)
	if err != nil {
		return nil, err
	}
	helpers, err := transformAttributeHelpers(expr.AsMap(source.Type).ElemType, expr.AsMap(target.Type).ElemType, ta, seen)
	if err != nil {
		return nil, err
	}
	return append(data, helpers...), nil
}

func collectUnionHelpers(source, target *expr.AttributeExpr, ta *transformAttrs, seen map[string]*codegen.TransformFunctionData) ([]*codegen.TransformFunctionData, error) {
	srcAttrs := expr.AsUnion(source.Type)
	tgtAttrs := expr.AsUnion(target.Type)
	if len(srcAttrs.Values) != len(tgtAttrs.Values) {
		return nil, fmt.Errorf("cannot transform union attribute %s with %d types to union attribute %s with %d types", source.Type.Name(), len(srcAttrs.Values), target.Type.Name(), len(tgtAttrs.Values))
	}
	var data []*codegen.TransformFunctionData
	for i, srcVal := range srcAttrs.Values {
		src := srcVal.Attribute
		tgt := tgtAttrs.Values[i].Attribute
		if ta.proto {
			tgt = unwrapAttr(tgt)
		} else {
			src = unwrapAttr(src)
		}
		helpers, err := collectHelpers(src, tgt, true, ta, seen)
		if err != nil {
			return nil, err
		}
		data = append(data, helpers...)
	}
	return data, nil
}

func collectObjectHelpers(source, target *expr.AttributeExpr, req bool, ta *transformAttrs, seen map[string]*codegen.TransformFunctionData) ([]*codegen.TransformFunctionData, error) {
	var data []*codegen.TransformFunctionData
	if ut, ok := source.Type.(expr.UserType); ok {
		tfd, stop, err := buildObjectHelper(source, target, req, ut, ta, seen)
		if err != nil {
			return nil, err
		}
		if stop {
			return nil, nil
		}
		if tfd != nil {
			data = append(data, tfd)
		}
	}
	helpers, err := collectObjectFieldHelpers(source, target, ta, seen)
	if err != nil {
		return nil, err
	}
	return append(data, helpers...), nil
}

func buildObjectHelper(source, target *expr.AttributeExpr, req bool, ut expr.UserType, ta *transformAttrs, seen map[string]*codegen.TransformFunctionData) (*codegen.TransformFunctionData, bool, error) {
	ta = objectHelperTransformAttrs(source, target, ta)
	name := transformHelperName(source, target, ta)
	if _, ok := seen[name]; ok {
		return nil, true, nil
	}
	code, err := transformAttribute(ut.Attribute(), target, "v", "res", true, ta)
	if err != nil {
		return nil, false, err
	}
	if !req {
		code = "if v == nil {\n\treturn nil\n}\n" + code
	}
	tfd := &codegen.TransformFunctionData{
		Name:          name,
		ParamTypeRef:  ta.SourceCtx.Scope.Ref(source, ta.SourceCtx.Pkg(source)),
		ResultTypeRef: ta.TargetCtx.Scope.Ref(target, ta.TargetCtx.Pkg(target)),
		Code:          code,
		ErrorAware:    ta.errorAware,
	}
	seen[name] = tfd
	return tfd, false, nil
}

func objectHelperTransformAttrs(source, target *expr.AttributeExpr, ta *transformAttrs) *transformAttrs {
	if ta.proto && isUnionMessage(target) {
		dup := dupTransformAttrs(ta)
		dup.message = dup.TargetCtx.Scope.Name(target, dup.TargetCtx.Pkg(target), false, false)
		return dup
	}
	if !ta.proto && isUnionMessage(source) {
		dup := dupTransformAttrs(ta)
		dup.message = dup.SourceCtx.Scope.Ref(source, dup.SourceCtx.Pkg(source))
		return dup
	}
	return ta
}

func collectObjectFieldHelpers(source, target *expr.AttributeExpr, ta *transformAttrs, seen map[string]*codegen.TransformFunctionData) ([]*codegen.TransformFunctionData, error) {
	var (
		data []*codegen.TransformFunctionData
		err  error
	)
	walkMatches(source, target, func(srcMatt, _ *expr.MappedAttributeExpr, srcc, tgtc *expr.AttributeExpr, n string) {
		if err != nil {
			return
		}
		srcc, tgtc, err = normalizedTransformAttrs(srcc, tgtc, ta)
		if err != nil {
			return
		}
		var helpers []*codegen.TransformFunctionData
		helpers, err = collectHelpers(srcc, tgtc, srcMatt.IsRequired(n), ta, seen)
		if err != nil {
			return
		}
		data = append(data, helpers...)
	})
	if err != nil {
		return nil, err
	}
	return data, nil
}

func normalizedTransformAttrs(srcc, tgtc *expr.AttributeExpr, ta *transformAttrs) (*expr.AttributeExpr, *expr.AttributeExpr, error) {
	if err := codegen.IsCompatible(srcc.Type, tgtc.Type, "", ""); err == nil {
		return srcc, tgtc, nil
	}
	if ta.proto {
		tgtc = unwrapAttr(tgtc)
	} else {
		srcc = unwrapAttr(srcc)
	}
	if err := codegen.IsCompatible(srcc.Type, tgtc.Type, "", ""); err != nil {
		return nil, nil, err
	}
	return srcc, tgtc, nil
}

// walkMatches iterates through the source attribute expression and executes
// the walker function.
func walkMatches(source, target *expr.AttributeExpr, walker func(src, tgt *expr.MappedAttributeExpr, srcc, tgtc *expr.AttributeExpr, n string)) {
	srcMatt := expr.NewMappedAttributeExpr(source)
	tgtMatt := expr.NewMappedAttributeExpr(target)
	srcObj := expr.AsObject(srcMatt.Type)
	tgtObj := expr.AsObject(tgtMatt.Type)
	for _, nat := range *srcObj {
		if att := tgtObj.Attribute(nat.Name); att != nil {
			walker(srcMatt, tgtMatt, nat.Attribute, att, nat.Name)
		}
	}
}

// transformHelperName returns the transformation function name to initialize a
// target user type from an instance of a source user type.
func transformHelperName(source, target *expr.AttributeExpr, ta *transformAttrs) string {
	var (
		sname  string
		tname  string
		prefix string
	)
	{
		// Do not consider package overrides for protogen generated types
		if ta.proto {
			target = expr.DupAtt(target)
			codegen.Walk(target, func(att *expr.AttributeExpr) error { // nolint: errcheck
				delete(att.Meta, "struct:pkg:path")
				return nil
			})
		} else {
			source = expr.DupAtt(source)
			codegen.Walk(source, func(att *expr.AttributeExpr) error { // nolint: errcheck
				delete(att.Meta, "struct:pkg:path")
				return nil
			})
		}
		sname = codegen.Goify(ta.SourceCtx.Scope.Name(source, ta.SourceCtx.Pkg(source), ta.TargetCtx.Pointer, ta.TargetCtx.UseDefault), true)
		tname = codegen.Goify(ta.TargetCtx.Scope.Name(target, ta.TargetCtx.Pkg(target), ta.TargetCtx.Pointer, ta.TargetCtx.UseDefault), true)
		prefix = ta.Prefix
	}
	return codegen.Goify(prefix+sname+"To"+tname, false)
}

// unAlias returns the base AttributeExpr of an aliased one.
func unAlias(at *expr.AttributeExpr) *expr.AttributeExpr {
	if prim := getPrimitive(at); prim != nil {
		return prim
	}
	return at
}

// isUnionMessage returns true if the given attribute is a union message.
func isUnionMessage(at *expr.AttributeExpr) bool {
	ut, ok := at.Type.(expr.UserType)
	if !ok {
		return false
	}
	obj := expr.AsObject(ut.Attribute().Type)
	if obj == nil {
		return false
	}
	for _, nat := range *obj {
		if expr.IsUnion(nat.Attribute.Type) {
			return true
		}
	}
	return false
}

// dupTransformAttrs returns a shallow copy of the given transformAttrs.
func dupTransformAttrs(ta *transformAttrs) *transformAttrs {
	return &transformAttrs{
		TransformAttrs: ta.TransformAttrs,
		proto:          ta.proto,
		targetInit:     ta.targetInit,
		wrapped:        ta.wrapped,
		message:        ta.message,
	}
}

func formatGoLiteral(v any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%#v", v)
	return b.String()
}

func protobufDefaultLiteral(att *expr.AttributeExpr, value any, ta *transformAttrs) string {
	literal, ok := protobufTypedDefaultLiteral(att, value, ta)
	if !ok {
		return formatGoLiteral(value)
	}
	return literal
}

func protobufTypedDefaultLiteral(att *expr.AttributeExpr, value any, ta *transformAttrs) (string, bool) {
	if typeName, _ := codegen.GetMetaType(att); typeName == "json.RawMessage" {
		actual := reflect.ValueOf(value)
		if actual.IsValid() && actual.Kind() == reflect.Slice && actual.Type().Elem().Kind() == reflect.Uint8 {
			literal := fmt.Sprintf("%#v", actual.Bytes())
			return typeName + strings.TrimPrefix(literal, "[]byte"), true
		}
	}
	switch actual := att.Type.(type) {
	case *expr.Array:
		return protobufArrayDefaultLiteral(actual, value, ta)
	case *expr.Map:
		return protobufMapDefaultLiteral(actual, value, ta)
	default:
		return formatGoLiteral(value), true
	}
}

func protobufArrayDefaultLiteral(arr *expr.Array, value any, ta *transformAttrs) (string, bool) {
	actual := reflect.ValueOf(value)
	if !actual.IsValid() || actual.Kind() != reflect.Slice {
		return "", false
	}
	elemRef := protobufDefaultTypeRef(arr.ElemType, ta)
	items := make([]string, 0, actual.Len())
	for i := 0; i < actual.Len(); i++ {
		literal, ok := protobufTypedDefaultLiteral(arr.ElemType, actual.Index(i).Interface(), ta)
		if !ok {
			return "", false
		}
		items = append(items, literal)
	}
	return "[]" + elemRef + "{" + strings.Join(items, ", ") + "}", true
}

func protobufMapDefaultLiteral(m *expr.Map, value any, ta *transformAttrs) (string, bool) {
	actual := reflect.ValueOf(value)
	if !actual.IsValid() || actual.Kind() != reflect.Map {
		return "", false
	}
	keys := actual.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface())
	})
	keyRef := protobufDefaultTypeRef(m.KeyType, ta)
	elemRef := protobufDefaultTypeRef(m.ElemType, ta)
	items := make([]string, 0, len(keys))
	for _, key := range keys {
		keyLiteral, ok := protobufTypedDefaultLiteral(m.KeyType, key.Interface(), ta)
		if !ok {
			return "", false
		}
		elemLiteral, ok := protobufTypedDefaultLiteral(m.ElemType, actual.MapIndex(key).Interface(), ta)
		if !ok {
			return "", false
		}
		items = append(items, keyLiteral+": "+elemLiteral)
	}
	return "map[" + keyRef + "]" + elemRef + "{" + strings.Join(items, ", ") + "}", true
}

func protobufDefaultTypeRef(att *expr.AttributeExpr, ta *transformAttrs) string {
	if ta.proto && expr.IsPrimitive(att.Type) {
		return protoBufNativeGoTypeName(att.Type)
	}
	return ta.TargetCtx.Scope.Ref(att, ta.TargetCtx.Pkg(att))
}
