//nolint:errcheck // Generator helpers write only to in-memory buffers/builders.
package codegen

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/expr"
)

type (
	objectInitField struct {
		Name string
		Expr *jen.Statement
	}

	transformArrayRenderData struct {
		ElemTypeRef    string
		SourceElem     *expr.AttributeExpr
		TargetElem     *expr.AttributeExpr
		SourceVar      string
		TargetVar      string
		NewVar         bool
		TransformAttrs *TransformAttrs
		LoopVar        string
		IsStruct       bool
		TypeAliasName  string
	}

	transformMapRenderData struct {
		KeyTypeRef     string
		ElemTypeRef    string
		SourceKey      *expr.AttributeExpr
		TargetKey      *expr.AttributeExpr
		SourceElem     *expr.AttributeExpr
		TargetElem     *expr.AttributeExpr
		SourceVar      string
		TargetVar      string
		NewVar         bool
		TransformAttrs *TransformAttrs
		LoopVar        string
		IsKeyStruct    bool
		IsElemStruct   bool
		TypeAliasName  string
	}

	transformUnionRenderCase struct {
		CaseName        string
		CaseTag         string
		SourceFieldName string
		TargetFieldName string
		SourceAttr      *expr.AttributeExpr
		TargetAttr      *expr.AttributeExpr
		TargetCastType  string
		UseHelper       bool
		HelperName      string
	}

	transformUnionRenderData struct {
		SourceVar       string
		TargetVar       string
		NewVar          bool
		TypeRef         string
		TargetIsPointer bool
		ValueTypeRef    string
		TempVarName     string
		Cases           []transformUnionRenderCase
		TransformAttrs  *TransformAttrs
	}
)

// GoTransform produces Go code that initializes the data structure defined
// by target from an instance of the data structure described by source.
// The data structures can be objects, arrays or maps. The algorithm
// matches object fields by name and ignores object fields in target that
// don't have a match in source. The matching and generated code leverage
// mapped attributes so that attribute names may use the "name:elem"
// syntax to define the name of the design attribute and the name of the
// corresponding generated Go struct field. The object field may also differ
// in that they may be pointers in one case and not the other. The function
// returns an error if target is not compatible with source (different type,
// fields of different type etc).
//
// As a special case GoTransform can map union types from and to object types
// with two attributes, one called "Value" which stores the value and one called
// "Type" which is of type string and contains the value type name (union types
// are otherwise implemented as a struct containing a single field: the current
// value - however having the kind explicitly stored is required to serialize to
// JSON for example).
//
// source and target are the attributes used in the transformation
//
// sourceVar and targetVar are the variable names used in the transformation
//
// sourceCtx and targetCtx are the attribute contexts for the source and target
// attributes
//
// prefix is the transformation helper function prefix
//
// newVar if true initializes a target variable with the generated Go code
// using `:=` operator. If false, it assigns Go code to the target variable
// using `=`.
func GoTransform(source, target *expr.AttributeExpr, sourceVar, targetVar string, sourceCtx, targetCtx *AttributeContext, prefix string, newVar bool) (string, []*TransformFunctionData, error) {
	ta := &TransformAttrs{
		SourceCtx: sourceCtx,
		TargetCtx: targetCtx,
		Prefix:    prefix,
	}

	code, err := transformAttribute(source, target, sourceVar, targetVar, newVar, ta)
	if err != nil {
		return "", nil, err
	}

	funcs, err := transformAttributeHelpers(source, target, ta, make(map[string]*TransformFunctionData))
	if err != nil {
		return "", nil, err
	}

	return strings.TrimRight(code, "\n"), funcs, nil
}

// transformAttribute returns the code to transform source attribute to target
// attribute. It returns an error if source and target are not compatible for
// transformation.
func transformAttribute(source, target *expr.AttributeExpr, sourceVar, targetVar string, newVar bool, ta *TransformAttrs) (code string, err error) {
	stmt, err := transformAttributeStmt(source, target, sourceVar, targetVar, newVar, ta)
	if err != nil {
		return "", err
	}
	return renderJenniferSnippet(stmt), nil
}

func transformAttributeStmt(source, target *expr.AttributeExpr, sourceVar, targetVar string, newVar bool, ta *TransformAttrs) (*jen.Statement, error) {
	if err := IsCompatible(source.Type, target.Type, sourceVar, targetVar); err != nil {
		return nil, err
	}
	switch {
	case expr.IsArray(source.Type):
		return transformArray(expr.AsArray(source.Type), expr.AsArray(target.Type), sourceVar, targetVar, newVar, ta)
	case expr.IsMap(source.Type):
		return transformMap(expr.AsMap(source.Type), expr.AsMap(target.Type), sourceVar, targetVar, newVar, ta)
	case expr.IsUnion(source.Type):
		return transformUnion(source, target, sourceVar, targetVar, newVar, ta)
	case expr.IsObject(source.Type):
		return transformObject(source, target, sourceVar, targetVar, newVar, ta)
	default:
		return transformPrimitive(source, target, sourceVar, targetVar, newVar, ta)
	}
}

// transformPrimitive returns the code to transform source primtive type to
// target primitive type. It returns an error if source and target are not
// compatible for transformation.
func transformPrimitive(source, target *expr.AttributeExpr, sourceVar, targetVar string, newVar bool, ta *TransformAttrs) (*jen.Statement, error) {
	if err := IsCompatible(source.Type, target.Type, sourceVar, targetVar); err != nil {
		return nil, err
	}
	assign := "="
	if newVar {
		assign = ":="
	}

	srcRef := ta.SourceCtx.Scope.Ref(source, ta.SourceCtx.Pkg(source))
	tgtRef := ta.TargetCtx.Scope.Ref(target, ta.TargetCtx.Pkg(target))
	stmt := &jen.Statement{}
	if srcRef != tgtRef {
		stmt.Add(Expr(targetVar)).Op(assign).Add(Expr(tgtRef + "(" + sourceVar + ")"))
		return stmt, nil
	}
	stmt.Add(Expr(targetVar)).Op(assign).Add(Expr(sourceVar))
	return stmt, nil
}

// transformObject generates Go code to transform source object to target
// object.
func transformObject(source, target *expr.AttributeExpr, sourceVar, targetVar string, newVar bool, ta *TransformAttrs) (*jen.Statement, error) {
	initFields, postInitCode, err := buildTransformObjectInit(source, target, sourceVar, targetVar, ta)
	if err != nil {
		return nil, err
	}

	deref := "&"
	assign := "="
	if newVar {
		assign = ":="
	}
	name := ta.TargetCtx.Scope.Name(target, ta.TargetCtx.Pkg(target), ta.TargetCtx.Pointer, ta.TargetCtx.UseDefault)
	stmt := &jen.Statement{}
	stmt.Add(Expr(targetVar)).Op(assign).Add(buildObjectLiteralExpr(name, deref == "&", initFields))

	fieldCode, err := buildTransformObjectFieldCode(source, target, sourceVar, targetVar, ta)
	if err != nil {
		return nil, err
	}
	appendStatementsWithSpacing(stmt, postInitCode, fieldCode)
	return stmt, nil
}

func appendStatementsWithSpacing(dst *jen.Statement, groups ...[]*jen.Statement) {
	for _, group := range groups {
		for _, stmt := range group {
			if stmt == nil {
				continue
			}
			dst.Line()
			dst.Add(stmt)
		}
	}
}

func buildTransformObjectInit(source, target *expr.AttributeExpr, sourceVar, targetVar string, ta *TransformAttrs) ([]objectInitField, []*jen.Statement, error) {
	var (
		initFields   []objectInitField
		postInitCode []*jen.Statement
		err          error
	)
	walkMatches(source, target, func(srcMatt, tgtMatt *expr.MappedAttributeExpr, srcc, tgtc *expr.AttributeExpr, n string) {
		if !expr.IsPrimitive(srcc.Type) {
			return
		}
		if err = IsCompatible(srcc.Type, tgtc.Type, sourceVar, targetVar); err != nil {
			return
		}
		exp, postAssign := transformObjectPrimitiveInitExpression(srcMatt, tgtMatt, srcc, tgtc, sourceVar, targetVar, n, ta)
		if postAssign != nil {
			postInitCode = append(postInitCode, postAssign)
		}
		if exp == nil {
			return
		}
		tgtField := GoifyAtt(tgtc, tgtMatt.ElemName(n), true)
		initFields = append(initFields, objectInitField{Name: tgtField, Expr: exp})
	})
	return initFields, postInitCode, err
}

func transformObjectPrimitiveInitExpression(srcMatt, tgtMatt *expr.MappedAttributeExpr, srcc, tgtc *expr.AttributeExpr, sourceVar, targetVar, name string, ta *TransformAttrs) (*jen.Statement, *jen.Statement) {
	srcPtr := ta.SourceCtx.IsPrimitivePointer(name, srcMatt.AttributeExpr)
	tgtPtr := ta.TargetCtx.IsPrimitivePointer(name, tgtMatt.AttributeExpr)
	srcField := sourceVar + "." + GoifyAtt(srcc, srcMatt.ElemName(name), true)
	tgtField := GoifyAtt(tgtc, tgtMatt.ElemName(name), true)
	_, isSrcUT := srcc.Type.(expr.UserType)
	_, isTgtUT := tgtc.Type.(expr.UserType)

	switch {
	case isSrcUT || isTgtUT:
		baseExpr := srcField
		if srcPtr {
			baseExpr = "*" + srcField
		}
		exp := Expr(ta.TargetCtx.Scope.Ref(tgtc, ta.TargetCtx.Pkg(tgtc)) + "(" + baseExpr + ")")
		if srcPtr && !srcMatt.IsRequired(name) {
			return nil, buildConditionalPrimitiveAssignmentStmt(srcField, targetVar, tgtField, exp, tgtPtr, Goify(tgtMatt.ElemName(name), false))
		}
		if tgtPtr {
			return nil, buildPointerPrimitiveAssignmentStmt(targetVar, tgtField, exp, Goify(tgtMatt.ElemName(name), false))
		}
		return exp, nil
	case srcPtr && !tgtPtr:
		exp := Expr("*" + srcField)
		if srcMatt.IsRequired(name) {
			return exp, nil
		}
		return nil, buildConditionalPrimitiveAssignmentStmt(srcField, targetVar, tgtField, exp, false, "")
	case !srcPtr && tgtPtr:
		return Expr("&" + srcField), nil
	default:
		return Expr(srcField), nil
	}
}

func buildTransformObjectFieldCode(source, target *expr.AttributeExpr, sourceVar, targetVar string, ta *TransformAttrs) ([]*jen.Statement, error) {
	var (
		stmts []*jen.Statement
		err   error
	)
	walkMatches(source, target, func(srcMatt, tgtMatt *expr.MappedAttributeExpr, srcc, tgtc *expr.AttributeExpr, n string) {
		if err != nil {
			return
		}
		code, codeErr := transformObjectFieldCode(srcMatt, tgtMatt, srcc, tgtc, target, sourceVar, targetVar, n, ta)
		if codeErr != nil {
			err = codeErr
			return
		}
		if code != nil {
			stmts = append(stmts, code)
		}
	})
	return stmts, err
}

func transformObjectFieldCode(srcMatt, tgtMatt *expr.MappedAttributeExpr, srcc, tgtc, target *expr.AttributeExpr, sourceVar, targetVar, name string, ta *TransformAttrs) (*jen.Statement, error) {
	if err := IsCompatible(srcc.Type, tgtc.Type, sourceVar, targetVar); err != nil {
		return nil, err
	}

	srcFieldVar := sourceVar + "." + GoifyAtt(srcc, srcMatt.ElemName(name), true)
	tgtFieldVar := targetVar + "." + GoifyAtt(tgtc, tgtMatt.ElemName(name), true)
	code, err := transformObjectFieldAssignment(srcc, tgtc, target, tgtMatt, srcFieldVar, tgtFieldVar, targetVar, name, ta)
	if err != nil {
		return nil, err
	}
	code = wrapTransformObjectFieldCode(code, srcc, tgtc, srcMatt, srcFieldVar, tgtFieldVar, name, ta)
	code = appendTransformStatements(code, transformObjectDefaultValueCode(srcc, tgtc, srcMatt, tgtMatt, srcFieldVar, tgtFieldVar, name, ta))
	return code, nil
}

func transformObjectFieldAssignment(srcc, tgtc, target *expr.AttributeExpr, tgtMatt *expr.MappedAttributeExpr, srcVar, tgtVar, targetVar, name string, ta *TransformAttrs) (*jen.Statement, error) {
	_, isUserType := srcc.Type.(expr.UserType)
	switch {
	case expr.IsArray(srcc.Type):
		return transformArray(expr.AsArray(srcc.Type), expr.AsArray(tgtc.Type), srcVar, tgtVar, false, ta)
	case expr.IsMap(srcc.Type):
		return transformMap(expr.AsMap(srcc.Type), expr.AsMap(tgtc.Type), srcVar, tgtVar, false, ta)
	case expr.IsUnion(srcc.Type):
		return transformUnion(srcc, tgtc, srcVar, tgtVar, false, ta)
	case isUserType:
		if ta.TargetCtx.IsInterface {
			ref := ta.TargetCtx.Scope.Ref(target, ta.TargetCtx.Pkg(target))
			tgtVar = targetVar + ".(" + ref + ")." + GoifyAtt(tgtc, tgtMatt.ElemName(name), true)
		}
		if expr.IsPrimitive(srcc.Type) {
			return nil, nil
		}
		stmt := &jen.Statement{}
		stmt.Add(Expr(tgtVar)).Op("=").Id(transformHelperName(srcc, tgtc, ta)).Call(Expr(srcVar))
		return stmt, nil
	case expr.IsObject(srcc.Type):
		return transformAttributeStmt(srcc, tgtc, srcVar, tgtVar, false, ta)
	default:
		return nil, nil
	}
}

func wrapTransformObjectFieldCode(code *jen.Statement, srcc, tgtc *expr.AttributeExpr, srcMatt *expr.MappedAttributeExpr, srcVar, tgtVar, name string, ta *TransformAttrs) *jen.Statement {
	if code == nil || !shouldWrapTransformObjectField(srcc, srcMatt, name, ta) {
		return code
	}
	condition := Expr(srcVar + " != nil")
	if expr.IsUnion(srcc.Type) {
		condition = Expr(srcVar + `.Kind() != ""`)
	}
	stmt := &jen.Statement{}
	stmt.If(condition).BlockFunc(func(group *jen.Group) {
		group.Add(code)
	})
	if expr.IsArray(srcc.Type) && srcMatt.IsRequired(name) {
		elemRef := ta.TargetCtx.Scope.Ref(expr.AsArray(tgtc.Type).ElemType, ta.TargetCtx.Pkg(expr.AsArray(tgtc.Type).ElemType))
		stmt.Else().BlockFunc(func(group *jen.Group) {
			group.Add(Expr(tgtVar)).Op("=").Add(Expr("[]" + elemRef + "{}"))
		})
	}
	return stmt
}

func shouldWrapTransformObjectField(srcc *expr.AttributeExpr, srcMatt *expr.MappedAttributeExpr, name string, ta *TransformAttrs) bool {
	isRef := !expr.IsPrimitive(srcc.Type) && !srcMatt.IsRequired(name) || ta.SourceCtx.IsPrimitivePointer(name, srcMatt.AttributeExpr) && expr.IsPrimitive(srcc.Type)
	marshalNonPrimitive := !expr.IsPrimitive(srcc.Type) && ta.SourceCtx.UseDefault && ta.TargetCtx.UseDefault
	return isRef || marshalNonPrimitive
}

func transformObjectDefaultValueCode(srcc, tgtc *expr.AttributeExpr, srcMatt, tgtMatt *expr.MappedAttributeExpr, srcVar, tgtVar, name string, ta *TransformAttrs) *jen.Statement {
	tdef := tgtMatt.GetDefault(name)
	if tdef == nil || !ta.TargetCtx.UseDefault || ta.TargetCtx.Pointer || srcMatt.IsRequired(name) {
		return nil
	}

	switch {
	case ta.SourceCtx.IsPrimitivePointer(name, srcMatt.AttributeExpr) || !expr.IsPrimitive(srcc.Type):
		stmt := &jen.Statement{}
		stmt.If(Expr(srcVar + " == nil")).BlockFunc(func(group *jen.Group) {
			switch {
			case ta.TargetCtx.IsPrimitivePointer(name, tgtMatt.AttributeExpr) && expr.IsPrimitive(tgtc.Type):
				group.Add(Expr("var tmp " + GoNativeTypeName(tgtc.Type) + " = " + formatGoLiteral(tdef)))
				group.Add(Expr(tgtVar)).Op("=").Op("&").Id("tmp")
			case expr.IsArray(tgtc.Type):
				group.Add(transformObjectArrayDefaultValueCode(tgtc, tgtVar, tdef, ta))
			default:
				group.Add(Expr(tgtVar)).Op("=").Add(Expr(formatGoLiteral(tdef)))
			}
		})
		return stmt
	case expr.IsPrimitive(srcc.Type) && srcMatt.HasDefaultValue(name) && ta.SourceCtx.UseDefault:
		stmt := &jen.Statement{}
		stmt.BlockFunc(func(group *jen.Group) {
			zeroType := ""
			nilable := false
			if typeName, _ := GetMetaType(tgtc); typeName != "" {
				nilable = typeStringIsNilable(typeName)
				if !nilable {
					zeroType = typeName
				}
			} else if _, ok := tgtc.Type.(expr.UserType); ok {
				zeroType = ta.TargetCtx.Scope.Ref(tgtc, ta.TargetCtx.Pkg(tgtc))
			} else {
				zeroType = GoNativeTypeName(tgtc.Type)
			}
			if zeroType != "" {
				group.Var().Id("zero").Id(zeroType)
			}
			condition := tgtVar + " == zero"
			if typeName, _ := GetMetaType(tgtc); typeName != "" && typeStringIsNilable(typeName) {
				condition = tgtVar + " == nil"
			}
			group.If(Expr(condition)).BlockFunc(func(ifGroup *jen.Group) {
				ifGroup.Add(Expr(tgtVar)).Op("=").Add(Expr(formatGoLiteral(tdef)))
			})
		})
		return stmt
	default:
		return nil
	}
}

func transformObjectArrayDefaultValueCode(tgtc *expr.AttributeExpr, tgtVar string, tdef any, ta *TransformAttrs) *jen.Statement {
	arr := expr.AsArray(tgtc.Type)
	stmt := &jen.Statement{}
	if !expr.IsAlias(arr.ElemType.Type) {
		stmt.Add(Expr(tgtVar)).Op("=").Add(Expr(formatGoLiteral(tdef)))
		return stmt
	}

	elemRef := ta.TargetCtx.Scope.Ref(arr.ElemType, ta.TargetCtx.Pkg(arr.ElemType))
	var items []string
	appendItems := func(values []any) {
		for _, value := range values {
			items = append(items, elemRef+"("+formatGoLiteral(value)+")")
		}
	}

	switch dv := tdef.(type) {
	case expr.ArrayVal:
		appendItems([]any(dv))
	case []any:
		appendItems(dv)
	case []string:
		for _, de := range dv {
			items = append(items, elemRef+"("+formatGoLiteral(de)+")")
		}
	case []int:
		for _, de := range dv {
			items = append(items, elemRef+"("+formatGoLiteral(de)+")")
		}
	case []int32:
		for _, de := range dv {
			items = append(items, elemRef+"("+formatGoLiteral(de)+")")
		}
	case []int64:
		for _, de := range dv {
			items = append(items, elemRef+"("+formatGoLiteral(de)+")")
		}
	case []uint:
		for _, de := range dv {
			items = append(items, elemRef+"("+formatGoLiteral(de)+")")
		}
	case []uint32:
		for _, de := range dv {
			items = append(items, elemRef+"("+formatGoLiteral(de)+")")
		}
	case []uint64:
		for _, de := range dv {
			items = append(items, elemRef+"("+formatGoLiteral(de)+")")
		}
	case []float32:
		for _, de := range dv {
			items = append(items, elemRef+"("+formatGoLiteral(de)+")")
		}
	case []float64:
		for _, de := range dv {
			items = append(items, elemRef+"("+formatGoLiteral(de)+")")
		}
	case []bool:
		for _, de := range dv {
			items = append(items, elemRef+"("+formatGoLiteral(de)+")")
		}
	default:
		stmt.Add(Expr(tgtVar)).Op("=").Add(Expr(formatGoLiteral(tdef)))
		return stmt
	}

	if len(items) == 0 {
		return nil
	}
	stmt.Add(Expr(tgtVar)).Op("=").Add(Expr("[]" + elemRef + "{" + strings.Join(items, ", ") + "}"))
	return stmt
}

func buildConditionalPrimitiveAssignmentStmt(sourceField, targetVar, targetField string, expression *jen.Statement, targetPointer bool, tempVar string) *jen.Statement {
	stmt := &jen.Statement{}
	stmt.If(Expr(sourceField + " != nil")).BlockFunc(func(group *jen.Group) {
		if targetPointer {
			group.Add(buildPointerPrimitiveAssignmentStmt(targetVar, targetField, expression, tempVar))
			return
		}
		group.Add(Expr(targetVar)).Dot(targetField).Op("=").Add(expression)
	})
	return stmt
}

func buildPointerPrimitiveAssignmentStmt(targetVar, targetField string, expression *jen.Statement, tempVar string) *jen.Statement {
	stmt := &jen.Statement{}
	stmt.Id(tempVar).Op(":=").Add(expression).Line()
	stmt.Add(Expr(targetVar)).Dot(targetField).Op("=").Op("&").Id(tempVar)
	return stmt
}

func appendTransformStatements(left, right *jen.Statement) *jen.Statement {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}
	stmt := &jen.Statement{}
	stmt.Add(left).Line()
	stmt.Add(right)
	return stmt
}

func buildObjectLiteralExpr(name string, pointer bool, fields []objectInitField) *jen.Statement {
	prefix := ""
	if pointer {
		prefix = "&"
	}
	if len(fields) == 0 {
		return Expr(prefix + name + "{}")
	}
	stmt := &jen.Statement{}
	if pointer {
		stmt.Op("&")
	}
	stmt.Id(name).CustomFunc(jen.Options{
		Open:      "{",
		Close:     "}",
		Separator: ",",
		Multi:     true,
	}, func(group *jen.Group) {
		for _, field := range fields {
			group.Id(field.Name).Op(":").Add(field.Expr)
		}
	})
	return stmt
}

// typeStringIsNilable takes a go type as a string and checks for a '[]' or
// 'map[' prefix to see if it's a nilable primitive type.
func typeStringIsNilable(typeName string) bool {
	return strings.HasPrefix(typeName, "[]") || strings.HasPrefix(typeName, "map[")
}

// transformArray generates Go code to transform source array to target array.
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
		IsStruct:       expr.IsObject(target.ElemType.Type),
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
		IsKeyStruct:    expr.IsObject(target.KeyType.Type),
		IsElemStruct:   expr.IsObject(target.ElemType.Type),
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
func transformUnion(source, target *expr.AttributeExpr, sourceVar, targetVar string, newVar bool, ta *TransformAttrs) (*jen.Statement, error) {
	srcUnion, tgtUnion, err := validateTransformUnion(source, target, sourceVar, targetVar)
	if err != nil {
		return nil, err
	}
	unionPkg := ta.TargetCtx.Pkg(target)
	typeRef := ta.TargetCtx.Scope.Ref(target, unionPkg)

	data := transformUnionRenderData{
		SourceVar:       sourceVar,
		TargetVar:       targetVar,
		NewVar:          newVar,
		TypeRef:         typeRef,
		TargetIsPointer: strings.HasPrefix(typeRef, "*"),
		ValueTypeRef:    strings.TrimPrefix(typeRef, "*"),
		TempVarName:     transformUnionTempVarName(targetVar),
		Cases:           buildTransformUnionCases(srcUnion, tgtUnion, unionPkg, ta),
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

func buildTransformUnionCases(srcUnion, tgtUnion *expr.Union, unionPkg string, ta *TransformAttrs) []transformUnionRenderCase {
	cases := make([]transformUnionRenderCase, 0, len(srcUnion.Values))
	for i, srcValue := range srcUnion.Values {
		targetValue, ok := matchingTransformUnionValue(srcValue, tgtUnion, i)
		if !ok {
			continue
		}
		cases = append(cases, transformUnionCaseData(srcValue, targetValue, unionPkg, ta))
	}
	return cases
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

func transformUnionCaseData(srcValue, targetValue *expr.NamedAttributeExpr, unionPkg string, ta *TransformAttrs) transformUnionRenderCase {
	return transformUnionRenderCase{
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
	stmt := &jen.Statement{}
	stmt.Add(Expr(data.TargetVar)).Op(assign).Make(TypeRef(typeName), jen.Len(Expr(data.SourceVar))).Line()
	stmt.For(
		jen.List(Expr(data.LoopVar), jen.Id("val")).Op(":=").Range().Add(Expr(data.SourceVar)),
	).BlockFunc(func(group *jen.Group) {
		if data.IsStruct {
			group.If(Expr("val == nil")).BlockFunc(func(ifGroup *jen.Group) {
				ifGroup.Add(Expr(data.TargetVar)).Index(Expr(data.LoopVar)).Op("=").Nil()
				ifGroup.Continue()
			})
			group.Add(Expr(data.TargetVar)).
				Index(Expr(data.LoopVar)).
				Op("=").
				Id(transformHelperName(data.SourceElem, data.TargetElem, data.TransformAttrs)).
				Call(Expr("val"))
			return
		}
		code, err := transformAttributeStmt(data.SourceElem, data.TargetElem, "val", data.TargetVar+"["+data.LoopVar+"]", false, data.TransformAttrs)
		if err != nil {
			group.Add(Expr(`panic("unreachable transform render error")`))
			return
		}
		group.Add(code)
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
	stmt := &jen.Statement{}
	stmt.Add(Expr(data.TargetVar)).Op(assign).Make(TypeRef(typeName), jen.Len(Expr(data.SourceVar))).Line()
	stmt.For(
		jen.List(jen.Id("key"), jen.Id("val")).Op(":=").Range().Add(Expr(data.SourceVar)),
	).BlockFunc(func(group *jen.Group) {
		if data.IsKeyStruct {
			group.Id("tk").Op(":=").Id(transformHelperName(data.SourceKey, data.TargetKey, data.TransformAttrs)).Call(Expr("key"))
		} else {
			code, err := transformAttributeStmt(data.SourceKey, data.TargetKey, "key", "tk", true, data.TransformAttrs)
			if err != nil {
				group.Add(Expr(`panic("unreachable transform render error")`))
				return
			}
			group.Add(code)
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
		code, err := transformAttributeStmt(data.SourceElem, data.TargetElem, "val", temp, true, data.TransformAttrs)
		if err != nil {
			group.Add(Expr(`panic("unreachable transform render error")`))
			return
		}
		group.Add(code)
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
					code, err := transformAttributeStmt(c.SourceAttr, c.TargetAttr, "actual", data.TempVarName, true, data.TransformAttrs)
					if err != nil {
						caseGroup.Add(Expr(`panic("unreachable transform render error")`))
						return
					}
					caseGroup.Add(code)
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
				caseGroup.Id("u").Op(":=").Add(Expr(data.TargetVar))
				caseGroup.Id("u").Dot("Set" + c.TargetFieldName).Call(
					jen.Parens(TypeRef(c.TargetCastType)).Call(Expr(data.TempVarName)),
				)
				caseGroup.Add(Expr(data.TargetVar)).Op("=").Id("u")
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
	// Do not generate a transform function for the top most user type.
	return appendNestedHelpers(source, target, true, ta, seen)
}

// collectHelpers recurses through the given attributes and returns the transform
// helper functions required to generate the transform code. If the attributes type
// is array or map then the recursion is done via transformAttributeHelpers so that
// the tope level conversion function is skipped as the generate code does not make
// use of it (since it inlines that top-level transformation).
func collectHelpers(source, target *expr.AttributeExpr, req bool, ta *TransformAttrs, seen map[string]*TransformFunctionData) (helpers []*TransformFunctionData, err error) {
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

	// Reset need for type assertion for union types because we are
	// generating the code to transform the concrete type.
	ta.TargetCtx.IsInterface = false
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
