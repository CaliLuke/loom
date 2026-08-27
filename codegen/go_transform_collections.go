//nolint:errcheck // Generator helpers write only to in-memory buffers/builders.
package codegen

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/expr"
)

func transformArray(source, target *expr.Array, sourceVar, targetVar string, newVar bool, ta *TransformAttrs) (*jen.Statement, error) {
	if err := IsCompatible(source.ElemType.Type, target.ElemType.Type, sourceVar+"[0]", targetVar+"[0]"); err != nil {
		return nil, err
	}
	data := transformArrayRenderData{
		ElemTypeRef:    ta.TargetCtx.Scope.Ref(target.ElemType, ta.TargetCtx.Pkg(target.ElemType)),
		SourceElem:     source.ElemType,
		TargetElem:     target.ElemType,
		SourceVar:      sourceVar,
		TargetVar:      targetVar,
		NewVar:         newVar,
		TransformAttrs: ta,
		LoopVar:        string(rune(105 + strings.Count(targetVar, "["))),
		IsStruct:       expr.IsObject(target.ElemType.Type) && !expr.AllowsNull(target.ElemType),
		SourcePresence: ta.SourceCtx.CollectionElementPresence && !expr.ArrayElementsAllowNull(source),
	}
	return renderTransformGoArray(data)
}

// transformMap generates Go code to transform source map to target map.
func transformMap(source, target *expr.Map, sourceVar, targetVar string, newVar bool, ta *TransformAttrs) (*jen.Statement, error) {
	if err := IsCompatible(source.KeyType.Type, target.KeyType.Type, sourceVar+"[key]", targetVar+"[key]"); err != nil {
		return nil, err
	}
	if err := IsCompatible(source.ElemType.Type, target.ElemType.Type, sourceVar+"[*]", targetVar+"[*]"); err != nil {
		return nil, err
	}
	data := transformMapRenderData{
		KeyTypeRef:     ta.TargetCtx.Scope.Ref(target.KeyType, ta.TargetCtx.Pkg(target.KeyType)),
		ElemTypeRef:    ta.TargetCtx.Scope.Ref(target.ElemType, ta.TargetCtx.Pkg(target.ElemType)),
		SourceKey:      source.KeyType,
		TargetKey:      target.KeyType,
		SourceElem:     source.ElemType,
		TargetElem:     target.ElemType,
		SourceVar:      sourceVar,
		TargetVar:      targetVar,
		NewVar:         newVar,
		TransformAttrs: ta,
		IsKeyStruct:    expr.IsObject(target.KeyType.Type) && !expr.AllowsNull(target.KeyType),
		IsElemStruct:   expr.IsObject(target.ElemType.Type) && !expr.AllowsNull(target.ElemType),
	}
	if depth := MapDepth(target); depth > 0 {
		data.LoopVar = string(rune(97 + depth))
	}
	return renderTransformGoMap(data)
}

// transformUnion generates Go code to transform source union to target union.
//
// Note: transport to/from service transforms are always object to union or
// union to object. The only case a transform is union to union is when
// converting a projected type from/to a service type.
func transformUnion(
	source *expr.AttributeExpr,
	target *expr.AttributeExpr,
	sourceVar string,
	targetVar string,
	newVar bool,
	targetPointerOverride *bool,
	ta *TransformAttrs,
) (*jen.Statement, error) {
	srcUnion, tgtUnion, err := validateTransformUnion(source, target, sourceVar, targetVar)
	if err != nil {
		return nil, err
	}
	unionPkg := ta.TargetCtx.Pkg(target)
	typeRef := ta.TargetCtx.Scope.Ref(target, unionPkg)
	tempVarName := transformUnionTempVarName(targetVar)

	cases, err := buildTransformUnionCases(srcUnion, tgtUnion, unionPkg, tempVarName, ta)
	if err != nil {
		return nil, err
	}
	targetIsPointer := strings.HasPrefix(typeRef, "*")
	if targetPointerOverride != nil {
		targetIsPointer = *targetPointerOverride
	}
	data := transformUnionRenderData{
		SourceVar:       sourceVar,
		TargetVar:       targetVar,
		NewVar:          newVar,
		TypeRef:         typeRef,
		TargetIsPointer: targetIsPointer,
		ValueTypeRef:    strings.TrimPrefix(typeRef, "*"),
		TempVarName:     tempVarName,
		Cases:           cases,
		TransformAttrs:  ta,
	}

	return renderTransformGoUnion(data)
}

func validateTransformUnion(source, target *expr.AttributeExpr, sourceVar, targetVar string) (*expr.Union, *expr.Union, error) {
	if !expr.IsUnion(target.Type) {
		return nil, nil, fmt.Errorf("cannot transform union %s to non-union %s", source.Type.Name(), target.Type.Name())
	}
	srcUnion, tgtUnion := expr.AsUnion(source.Type), expr.AsUnion(target.Type)
	if len(srcUnion.Values) != len(tgtUnion.Values) {
		return nil, nil, fmt.Errorf("cannot transform union: number of union types differ (%s has %d, %s has %d)",
			source.Type.Name(), len(srcUnion.Values), target.Type.Name(), len(tgtUnion.Values))
	}
	for i, st := range srcUnion.Values {
		if err := IsCompatible(st.Attribute.Type, tgtUnion.Values[i].Attribute.Type, sourceVar, targetVar); err != nil {
			return nil, nil, fmt.Errorf("cannot transform union %s to %s: type at index %d: %w",
				source.Type.Name(), target.Type.Name(), i, err)
		}
	}
	return srcUnion, tgtUnion, nil
}

func transformUnionTempVarName(targetVar string) string {
	if strings.HasPrefix(targetVar, "obj.") {
		return "tmp"
	}
	return "obj"
}

func buildTransformUnionCases(srcUnion, tgtUnion *expr.Union, unionPkg, tempVarName string, ta *TransformAttrs) ([]transformUnionRenderCase, error) {
	cases := make([]transformUnionRenderCase, 0, len(srcUnion.Values))
	for i, srcValue := range srcUnion.Values {
		targetValue, ok := matchingTransformUnionValue(srcValue, tgtUnion, i)
		if !ok {
			continue
		}
		c, err := transformUnionCaseData(srcValue, targetValue, unionPkg, tempVarName, ta)
		if err != nil {
			return nil, err
		}
		cases = append(cases, c)
	}
	return cases, nil
}

func matchingTransformUnionValue(srcValue *expr.NamedAttributeExpr, tgtUnion *expr.Union, index int) (*expr.NamedAttributeExpr, bool) {
	if srcValue == nil || srcValue.Attribute == nil || index >= len(tgtUnion.Values) {
		return nil, false
	}
	targetValue := tgtUnion.Values[index]
	if targetValue == nil || targetValue.Attribute == nil {
		return nil, false
	}
	return targetValue, true
}

func transformUnionCaseData(srcValue, targetValue *expr.NamedAttributeExpr, unionPkg, tempVarName string, ta *TransformAttrs) (transformUnionRenderCase, error) {
	c := transformUnionRenderCase{
		CaseName:        srcValue.Name,
		CaseTag:         expr.UnionVariantTag(srcValue),
		SourceFieldName: Goify(srcValue.Name, true),
		TargetFieldName: Goify(targetValue.Name, true),
		SourceAttr:      srcValue.Attribute,
		TargetAttr:      targetValue.Attribute,
		TargetCastType:  ta.TargetCtx.Scope.Ref(targetValue.Attribute, transformUnionCastPkg(targetValue.Attribute, unionPkg, ta)),
		UseHelper:       transformUnionUsesHelper(srcValue.Attribute, targetValue.Attribute),
		HelperName:      transformHelperName(srcValue.Attribute, targetValue.Attribute, ta),
	}
	if c.UseHelper {
		return c, nil
	}
	code, err := transformAttributeStmt(c.SourceAttr, c.TargetAttr, "actual", tempVarName, true, ta)
	if err != nil {
		return transformUnionRenderCase{}, err
	}
	c.TransformCode = code
	return c, nil
}

func transformUnionCastPkg(targetAttr *expr.AttributeExpr, unionPkg string, ta *TransformAttrs) string {
	castPkg := ta.TargetCtx.Pkg(targetAttr)
	if castPkg == ta.TargetCtx.DefaultPkg && unionPkg != "" && unionPkg != ta.TargetCtx.DefaultPkg {
		return unionPkg
	}
	return castPkg
}

func transformUnionUsesHelper(sourceAttr, targetAttr *expr.AttributeExpr) bool {
	_, srcIsUserType := sourceAttr.Type.(expr.UserType)
	_, tgtIsUserType := targetAttr.Type.(expr.UserType)
	return srcIsUserType && expr.IsObject(sourceAttr.Type) && tgtIsUserType && expr.IsObject(targetAttr.Type)
}

func renderTransformGoArray(data transformArrayRenderData) (*jen.Statement, error) {
	assign := "="
	if data.NewVar {
		assign = ":="
	}
	typeName := "[]" + data.ElemTypeRef
	if data.TypeAliasName != "" {
		typeName = data.TypeAliasName
	}
	var elemCode *jen.Statement
	sourceElement := "val"
	if data.SourcePresence {
		sourceElement = "actual"
	}
	if !data.IsStruct {
		var err error
		elemCode, err = transformAttributeStmt(data.SourceElem, data.TargetElem, sourceElement, data.TargetVar+"["+data.LoopVar+"]", false, data.TransformAttrs)
		if err != nil {
			return nil, err
		}
	}
	stmt := &jen.Statement{}
	stmt.Add(Expr(data.TargetVar)).Op(assign).Make(TypeRef(typeName), jen.Len(Expr(data.SourceVar))).Line()
	stmt.For(
		jen.List(Expr(data.LoopVar), jen.Id("val")).Op(":=").Range().Add(Expr(data.SourceVar)),
	).BlockFunc(func(group *jen.Group) {
		if data.SourcePresence {
			group.List(jen.Id("actual"), jen.Id("ok")).Op(":=").Id("val").Dot("Value").Call()
			group.If(jen.Op("!").Id("ok")).Block(jen.Continue())
		}
		if data.IsStruct {
			group.If(Expr(sourceElement + " == nil")).BlockFunc(func(ifGroup *jen.Group) {
				ifGroup.Add(Expr(data.TargetVar)).Index(Expr(data.LoopVar)).Op("=").Nil()
				ifGroup.Continue()
			})
			group.Add(Expr(data.TargetVar)).
				Index(Expr(data.LoopVar)).
				Op("=").
				Id(transformHelperName(data.SourceElem, data.TargetElem, data.TransformAttrs)).
				Call(Expr(sourceElement))
			return
		}
		group.Add(elemCode)
	})
	return stmt, nil
}

func renderTransformGoMap(data transformMapRenderData) (*jen.Statement, error) {
	assign := "="
	if data.NewVar {
		assign = ":="
	}
	typeName := "map[" + data.KeyTypeRef + "]" + data.ElemTypeRef
	if data.TypeAliasName != "" {
		typeName = data.TypeAliasName
	}
	var keyCode *jen.Statement
	if !data.IsKeyStruct {
		var err error
		keyCode, err = transformAttributeStmt(data.SourceKey, data.TargetKey, "key", "tk", true, data.TransformAttrs)
		if err != nil {
			return nil, err
		}
	}
	var elemCode *jen.Statement
	if !data.IsElemStruct {
		var err error
		temp := "tv" + data.LoopVar
		elemCode, err = transformAttributeStmt(data.SourceElem, data.TargetElem, "val", temp, true, data.TransformAttrs)
		if err != nil {
			return nil, err
		}
	}
	stmt := &jen.Statement{}
	stmt.Add(Expr(data.TargetVar)).Op(assign).Make(TypeRef(typeName), jen.Len(Expr(data.SourceVar))).Line()
	stmt.For(
		jen.List(jen.Id("key"), jen.Id("val")).Op(":=").Range().Add(Expr(data.SourceVar)),
	).BlockFunc(func(group *jen.Group) {
		if data.IsKeyStruct {
			group.Id("tk").Op(":=").Id(transformHelperName(data.SourceKey, data.TargetKey, data.TransformAttrs)).Call(Expr("key"))
		} else {
			group.Add(keyCode)
		}
		if data.IsElemStruct {
			group.If(Expr("val == nil")).BlockFunc(func(ifGroup *jen.Group) {
				ifGroup.Add(Expr(data.TargetVar)).Index(Expr("tk")).Op("=").Nil()
				ifGroup.Continue()
			})
			group.Add(Expr(data.TargetVar)).
				Index(Expr("tk")).
				Op("=").
				Id(transformHelperName(data.SourceElem, data.TargetElem, data.TransformAttrs)).
				Call(Expr("val"))
			return
		}
		temp := "tv" + data.LoopVar
		group.Add(elemCode)
		group.Add(Expr(data.TargetVar)).Index(Expr("tk")).Op("=").Add(Expr(temp))
	})
	return stmt, nil
}

func renderTransformGoUnion(data transformUnionRenderData) (*jen.Statement, error) {
	stmt := &jen.Statement{}
	if data.NewVar {
		stmt.Var().Add(Expr(data.TargetVar)).Add(TypeRef(data.TypeRef)).Line()
	} else {
		stmt.Line()
	}
	stmt.Switch(jen.String().Call(Expr(data.SourceVar).Dot("Kind").Call())).BlockFunc(func(group *jen.Group) {
		for _, c := range data.Cases {
			group.Case(jen.Lit(c.CaseTag)).BlockFunc(func(caseGroup *jen.Group) {
				caseGroup.List(jen.Id("actual"), jen.Id("_")).Op(":=").Add(Expr(data.SourceVar)).Dot("As" + c.SourceFieldName).Call()
				if c.UseHelper {
					caseGroup.Id(data.TempVarName).Op(":=").Id(c.HelperName).Call(Expr("actual"))
					caseGroup.Line()
				} else {
					caseGroup.Add(c.TransformCode)
				}
				if data.NewVar {
					caseGroup.Var().Id("u").Add(TypeRef(data.ValueTypeRef))
					caseGroup.Id("u").Dot("Set" + c.TargetFieldName).Call(
						jen.Parens(TypeRef(c.TargetCastType)).Call(Expr(data.TempVarName)),
					)
					if data.TargetIsPointer {
						caseGroup.Add(Expr(data.TargetVar)).Op("=").Op("&").Id("u")
					} else {
						caseGroup.Add(Expr(data.TargetVar)).Op("=").Id("u")
					}
					return
				}
				if data.TargetIsPointer {
					caseGroup.Var().Id("u").Add(TypeRef(data.ValueTypeRef))
					caseGroup.If(Expr(data.TargetVar).Op("!=").Nil()).Block(
						jen.Id("u").Op("=").Op("*").Add(Expr(data.TargetVar)),
					)
				} else {
					caseGroup.Id("u").Op(":=").Add(Expr(data.TargetVar))
				}
				caseGroup.Id("u").Dot("Set" + c.TargetFieldName).Call(
					jen.Parens(TypeRef(c.TargetCastType)).Call(Expr(data.TempVarName)),
				)
				if data.TargetIsPointer {
					caseGroup.Add(Expr(data.TargetVar)).Op("=").Op("&").Id("u")
				} else {
					caseGroup.Add(Expr(data.TargetVar)).Op("=").Id("u")
				}
			})
		}
	})
	return stmt, nil
}

func renderJenniferSnippet(stmt *jen.Statement) string {
	var buf bytes.Buffer
	_ = stmt.Render(&buf)
	rendered := buf.String()
	if rendered != "" && !strings.HasSuffix(rendered, "\n") {
		rendered += "\n"
	}
	return rendered
}

func formatGoLiteral(v any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%#v", v)
	return b.String()
}

func formatAttributeGoLiteral(att *expr.AttributeExpr, value any) string {
	if typeName, _ := GetMetaType(att); typeName == "json.RawMessage" {
		actual := reflect.ValueOf(value)
		if actual.IsValid() && actual.Kind() == reflect.Slice && actual.Type().Elem().Kind() == reflect.Uint8 {
			literal := fmt.Sprintf("%#v", actual.Bytes())
			return typeName + strings.TrimPrefix(literal, "[]byte")
		}
	}
	return formatGoLiteral(value)
}

func typedDefaultLiteral(att *expr.AttributeExpr, value any, ta *TransformAttrs) (string, bool) {
	switch actual := att.Type.(type) {
	case *expr.Map:
		return typedMapDefaultLiteral(actual, value, ta)
	default:
		return formatAttributeGoLiteral(att, value), true
	}
}

func typedMapDefaultLiteral(m *expr.Map, value any, ta *TransformAttrs) (string, bool) {
	items, ok := mapDefaultLiteralItems(m, value, ta)
	if !ok {
		return "", false
	}
	keyRef := ta.TargetCtx.Scope.Ref(m.KeyType, ta.TargetCtx.Pkg(m.KeyType))
	elemRef := ta.TargetCtx.Scope.Ref(m.ElemType, ta.TargetCtx.Pkg(m.ElemType))
	return "map[" + keyRef + "]" + elemRef + "{" + strings.Join(items, ", ") + "}", true
}

func mapDefaultLiteralItems(m *expr.Map, value any, ta *TransformAttrs) ([]string, bool) {
	actual := reflect.ValueOf(value)
	if !actual.IsValid() || actual.Kind() != reflect.Map {
		return nil, false
	}
	keys := actual.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface())
	})
	items := make([]string, 0, len(keys))
	for _, key := range keys {
		keyLiteral, ok := typedDefaultValueLiteral(m.KeyType, key.Interface(), ta)
		if !ok {
			return nil, false
		}
		elemLiteral, ok := typedDefaultValueLiteral(m.ElemType, actual.MapIndex(key).Interface(), ta)
		if !ok {
			return nil, false
		}
		items = append(items, keyLiteral+": "+elemLiteral)
	}
	return items, true
}

func typedDefaultValueLiteral(att *expr.AttributeExpr, value any, ta *TransformAttrs) (string, bool) {
	if expr.IsMap(att.Type) {
		return typedMapDefaultLiteral(expr.AsMap(att.Type), value, ta)
	}
	return formatAttributeGoLiteral(att, value), true
}

// transformAttributeHelpers returns the Go transform functions and their definitions
// that may be used in code produced by Transform. It returns an error if source and
// target are incompatible (different types, fields of different type etc).
// transformAttributeHelpers recurses through the attribute types and calls
// collectHelpers for each child attribute. collectHelpers actually produces the
// transform helper functions for the given attribute.
//
// source, target are the source and target attributes used in transformation
//
// ta holds the transform attributes
//
// seen keeps track of generated transform functions to avoid infinite recursion.
func transformAttributeHelpers(source, target *expr.AttributeExpr, ta *TransformAttrs, seen map[string]*TransformFunctionData) (helpers []*TransformFunctionData, err error) {
	if expr.IsNullable(source) || expr.IsNullable(target) {
		concreteSource := concretePresenceAttribute(source)
		concreteTarget := concretePresenceAttribute(target)
		if nullablePhysicalTypeRef(source, ta.SourceCtx) != nullablePhysicalTypeRef(target, ta.TargetCtx) &&
			presenceUserObjectPair(concreteSource, concreteTarget) {
			helper, helperErr := generateHelper(concreteSource, concreteTarget, true, ta, seen)
			if helperErr != nil {
				return nil, helperErr
			}
			if helper != nil {
				helpers = append(helpers, helper)
			}
		}
	}
	// Non-nullable top-level user types are transformed inline and do not need
	// their own helper. Nullable named object roots use a helper inside the
	// presence transform and are collected above.
	nested, err := appendNestedHelpers(source, target, true, ta, seen)
	return append(helpers, nested...), err
}

// collectHelpers recurses through the given attributes and returns the transform
// helper functions required to generate the transform code. If the attributes type
// is array or map then the recursion is done via transformAttributeHelpers so that
// the tope level conversion function is skipped as the generate code does not make
// use of it (since it inlines that top-level transformation).
func collectHelpers(source, target *expr.AttributeExpr, req bool, ta *TransformAttrs, seen map[string]*TransformFunctionData) (helpers []*TransformFunctionData, err error) {
	if expr.IsNullable(source) || expr.IsNullable(target) {
		source = concretePresenceAttribute(source)
		target = concretePresenceAttribute(target)
	}
	name := transformHelperName(source, target, ta)
	if _, ok := seen[name]; ok {
		return helpers, err
	}
	if _, ok := source.Type.(expr.UserType); ok && expr.IsObject(source.Type) {
		var h *TransformFunctionData
		h, err = generateHelper(source, target, req, ta, seen)
		if err != nil {
			return helpers, err
		}
		if h != nil {
			helpers = append(helpers, h)
		}
	}
	var other []*TransformFunctionData
	if other, err = appendNestedHelpers(source, target, req, ta, seen); err != nil {
		return helpers, err
	}
	helpers = append(helpers, other...)
	return helpers, err
}

func appendNestedHelpers(source, target *expr.AttributeExpr, req bool, ta *TransformAttrs, seen map[string]*TransformFunctionData) (helpers []*TransformFunctionData, err error) {
	if matchingExplicitType(source, target) {
		return helpers, nil
	}
	var other []*TransformFunctionData
	switch {
	case expr.IsArray(source.Type):
		if other, err = collectHelpers(expr.AsArray(source.Type).ElemType, expr.AsArray(target.Type).ElemType, req, ta, seen); err == nil {
			helpers = append(helpers, other...)
		}
	case expr.IsMap(source.Type):
		sm, tm := expr.AsMap(source.Type), expr.AsMap(target.Type)
		if other, err = collectHelpers(sm.ElemType, tm.ElemType, req, ta, seen); err == nil {
			helpers = append(helpers, other...)
			if other, err = collectHelpers(sm.KeyType, tm.KeyType, req, ta, seen); err == nil {
				helpers = append(helpers, other...)
			}
		}
	case expr.IsUnion(source.Type):
		tt := expr.AsUnion(target.Type)
		if tt == nil {
			return helpers, err
		}
		for i, st := range expr.AsUnion(source.Type).Values {
			if other, err = collectHelpers(st.Attribute, tt.Values[i].Attribute, req, ta, seen); err == nil {
				helpers = append(helpers, other...)
			}
		}
	case expr.IsObject(source.Type):
		if expr.IsUnion(target.Type) {
			return helpers, err
		}
		walkMatches(source, target, func(srcMatt, _ *expr.MappedAttributeExpr, srcc, tgtc *expr.AttributeExpr, n string) {
			if err != nil {
				return
			}
			if other, err = collectHelpers(srcc, tgtc, srcMatt.IsRequired(n), ta, seen); err == nil {
				helpers = append(helpers, other...)
			}
		})
	}
	return helpers, err
}

// generateHelper generates the code that transform instances of source into
// target. Both source and targe must be user types or generateHelper panics.
// generateHelper returns nil if a helper has already been generated for the
// pair source, target.
func generateHelper(source, target *expr.AttributeExpr, req bool, ta *TransformAttrs, seen map[string]*TransformFunctionData) (*TransformFunctionData, error) {
	name := transformHelperName(source, target, ta)
	if _, ok := seen[name]; ok {
		return nil, nil
	}
	nested := *ta
	nested.SourceCtx = ta.SourceCtx.Dup()
	nested.TargetCtx = ta.TargetCtx.Dup()
	if attributeUsesJSONPresence(source, nested.SourceCtx) {
		nested.SourceCtx.JSONPresence = true
		nested.SourceCtx.CollectionElementPresence = true
	}
	if attributeUsesJSONPresence(target, nested.TargetCtx) {
		nested.TargetCtx.JSONPresence = true
		nested.TargetCtx.CollectionElementPresence = true
	}
	ta = &nested

	// When transforming into a user type defined in an external package, assume
	// nested anonymous types (e.g., union sum types) belong to the same target
	// package unless they explicitly specify a different location.
	prevDefaultPkg := ta.TargetCtx.DefaultPkg
	prevSamePackageConversion := ta.TargetCtx.SamePackageConversion
	if pkg := ta.TargetCtx.Pkg(target); pkg != "" && pkg != prevDefaultPkg {
		ta.TargetCtx.DefaultPkg = pkg
		ta.TargetCtx.SamePackageConversion = false
		defer func() {
			ta.TargetCtx.DefaultPkg = prevDefaultPkg
			ta.TargetCtx.SamePackageConversion = prevSamePackageConversion
		}()
	}

	code, err := transformAttribute(source, target, "v", "res", true, ta)
	if err != nil {
		return nil, err
	}
	if ta.SourceCtx.Pointer && !expr.IsPrimitive(source.Type) {
		code = "if v == nil {\n\treturn nil\n}\n" + code
	} else if !req && !expr.IsPrimitive(source.Type) {
		code = "if v == nil {\n\treturn nil\n}\n" + code
	}
	tfd := &TransformFunctionData{
		Name:          name,
		ParamTypeRef:  ta.SourceCtx.Scope.Ref(source, ta.SourceCtx.Pkg(source)),
		ResultTypeRef: ta.TargetCtx.Scope.Ref(target, ta.TargetCtx.Pkg(target)),
		Code:          code,
	}
	seen[name] = tfd
	return tfd, nil
}

func attributeUsesJSONPresence(attribute *expr.AttributeExpr, context *AttributeContext) bool {
	if attribute == nil || context == nil || len(context.JSONPresenceTypes) == 0 {
		return false
	}
	if userType, ok := attribute.Type.(expr.UserType); ok && context.JSONPresenceTypes[userType.ID()] {
		return true
	}
	name := context.Scope.Name(attribute, context.Pkg(attribute), false, context.UseDefault)
	return context.JSONPresenceTypes[name]
}

// walkMatches iterates through the attributes of source and looks for
// attributes with identical names in target. walkMatches calls the walker
// function for each pair of matched attributes. Both source and target must be
// objects or else walkMatches panics.
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
func transformHelperName(source, target *expr.AttributeExpr, ta *TransformAttrs) string {
	var (
		sname  string
		tname  string
		prefix string
	)
	{
		sname = Goify(ta.SourceCtx.Scope.Name(source, ta.SourceCtx.Pkg(source), ta.SourceCtx.Pointer, ta.SourceCtx.UseDefault), true)
		tname = Goify(ta.TargetCtx.Scope.Name(target, ta.TargetCtx.Pkg(target), ta.TargetCtx.Pointer, ta.TargetCtx.UseDefault), true)
		prefix = ta.Prefix
		if prefix == "" {
			prefix = "transform"
		}
	}
	return Goify(prefix+sname+"To"+tname, false)
}
