//nolint:errcheck // Generator helpers write only to in-memory buffers/builders.
package codegen

import (
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
		TransformCode   *jen.Statement
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
		return transformUnion(source, target, sourceVar, targetVar, newVar, nil, ta)
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
		code, codeErr := transformObjectFieldCode(srcMatt, tgtMatt, srcc, tgtc, sourceVar, targetVar, n, ta)
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

func transformObjectFieldCode(srcMatt, tgtMatt *expr.MappedAttributeExpr, srcc, tgtc *expr.AttributeExpr, sourceVar, targetVar, name string, ta *TransformAttrs) (*jen.Statement, error) {
	if err := IsCompatible(srcc.Type, tgtc.Type, sourceVar, targetVar); err != nil {
		return nil, err
	}

	srcFieldVar := sourceVar + "." + GoifyAtt(srcc, srcMatt.ElemName(name), true)
	tgtFieldVar := targetVar + "." + GoifyAtt(tgtc, tgtMatt.ElemName(name), true)
	var code *jen.Statement
	var err error
	if expr.IsUnion(srcc.Type) {
		targetIsPointer := !tgtMatt.IsRequired(name)
		code, err = transformUnion(srcc, tgtc, srcFieldVar, tgtFieldVar, false, &targetIsPointer, ta)
	} else {
		code, err = transformObjectFieldAssignment(srcc, tgtc, srcFieldVar, tgtFieldVar, ta)
	}
	if err != nil {
		return nil, err
	}
	code = wrapTransformObjectFieldCode(code, srcc, tgtc, srcMatt, srcFieldVar, tgtFieldVar, name, ta)
	code = appendTransformStatements(code, transformObjectDefaultValueCode(srcc, tgtc, srcMatt, tgtMatt, srcFieldVar, tgtFieldVar, name, ta))
	return code, nil
}

func transformObjectFieldAssignment(srcc, tgtc *expr.AttributeExpr, srcVar, tgtVar string, ta *TransformAttrs) (*jen.Statement, error) {
	_, isUserType := srcc.Type.(expr.UserType)
	switch {
	case expr.IsArray(srcc.Type):
		return transformArray(expr.AsArray(srcc.Type), expr.AsArray(tgtc.Type), srcVar, tgtVar, false, ta)
	case expr.IsMap(srcc.Type):
		return transformMap(expr.AsMap(srcc.Type), expr.AsMap(tgtc.Type), srcVar, tgtVar, false, ta)
	case expr.IsUnion(srcc.Type):
		return transformUnion(srcc, tgtc, srcVar, tgtVar, false, nil, ta)
	case isUserType:
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
		if srcMatt.IsRequired(name) {
			condition = Expr(srcVar + `.Kind() != ""`)
		} else {
			condition = Expr(srcVar + ` != nil && ` + srcVar + `.Kind() != ""`)
		}
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
			case expr.IsMap(tgtc.Type):
				group.Add(transformObjectMapDefaultValueCode(tgtc, tgtVar, tdef, ta))
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
	items, ok := aliasDefaultArrayItems(elemRef, tdef)
	if !ok {
		stmt.Add(Expr(tgtVar)).Op("=").Add(Expr(formatGoLiteral(tdef)))
		return stmt
	}

	if len(items) == 0 {
		return nil
	}
	stmt.Add(Expr(tgtVar)).Op("=").Add(Expr("[]" + elemRef + "{" + strings.Join(items, ", ") + "}"))
	return stmt
}

func transformObjectMapDefaultValueCode(tgtc *expr.AttributeExpr, tgtVar string, tdef any, ta *TransformAttrs) *jen.Statement {
	literal, ok := typedDefaultLiteral(tgtc, tdef, ta)
	stmt := &jen.Statement{}
	if !ok {
		stmt.Add(Expr(tgtVar)).Op("=").Add(Expr(formatGoLiteral(tdef)))
		return stmt
	}
	stmt.Add(Expr(tgtVar)).Op("=").Add(Expr(literal))
	return stmt
}

func aliasDefaultArrayItems(elemRef string, tdef any) ([]string, bool) {
	switch dv := tdef.(type) {
	case expr.ArrayVal:
		return formatAliasArrayItems(elemRef, []any(dv)), true
	case []any:
		return formatAliasArrayItems(elemRef, dv), true
	case []string:
		return formatAliasTypedArrayItems(elemRef, dv), true
	case []int:
		return formatAliasTypedArrayItems(elemRef, dv), true
	case []int32:
		return formatAliasTypedArrayItems(elemRef, dv), true
	case []int64:
		return formatAliasTypedArrayItems(elemRef, dv), true
	case []uint:
		return formatAliasTypedArrayItems(elemRef, dv), true
	case []uint32:
		return formatAliasTypedArrayItems(elemRef, dv), true
	case []uint64:
		return formatAliasTypedArrayItems(elemRef, dv), true
	case []float32:
		return formatAliasTypedArrayItems(elemRef, dv), true
	case []float64:
		return formatAliasTypedArrayItems(elemRef, dv), true
	case []bool:
		return formatAliasTypedArrayItems(elemRef, dv), true
	default:
		return nil, false
	}
}

func formatAliasArrayItems(elemRef string, values []any) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		items = append(items, elemRef+"("+formatGoLiteral(value)+")")
	}
	return items
}

func formatAliasTypedArrayItems[T any](elemRef string, values []T) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		items = append(items, elemRef+"("+formatGoLiteral(value)+")")
	}
	return items
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
