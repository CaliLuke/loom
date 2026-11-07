package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/internal/representation"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/expr"
)

func transformObjectPresenceField(
	sourceParent, targetParent *expr.MappedAttributeExpr,
	source, target *expr.AttributeExpr,
	sourceVar, targetVar, name string,
	ta *TransformAttrs,
) (*jen.Statement, bool, error) {
	sourcePresence := ta.SourceCtx.FieldPresence(sourceParent, name, source)
	targetPresence := ta.TargetCtx.FieldPresence(targetParent, name, target)
	samePresence := sourcePresence == targetPresence
	if samePresence {
		switch sourcePresence {
		case NativePresence:
			return nil, false, nil
		case NullablePresence:
			code, err := transformNullablePresence(source, target, sourceVar, targetVar, false, ta.DefaultForField(sourceParent, targetParent, name), ta)
			return code, true, err
		}
	}
	if !samePresence && isAnyPresenceAttribute(source) && isAnyPresenceAttribute(target) {
		code, err := transformRawAnyPresence(source, target, sourceVar, targetVar, false, targetPresence == NullablePresence, ta)
		return code, true, err
	}
	if sourcePresence == NullablePresence || targetPresence == NullablePresence {
		return nil, true, fmt.Errorf("cannot transform nullable field %s to a non-nullable representation", name)
	}
	if sourcePresence == OptionalPresence {
		code, err := transformOptionalField(sourceParent, targetParent, source, target, sourceVar, targetVar, name, ta)
		return code, true, err
	}
	code, err := transformNativeToOptional(sourceParent, targetParent, source, target, sourceVar, targetVar, name, ta)
	return code, true, err
}

func transformOptionalField(
	sourceParent, targetParent *expr.MappedAttributeExpr,
	source, target *expr.AttributeExpr,
	sourceVar, targetVar, name string,
	ta *TransformAttrs,
) (*jen.Statement, error) {
	sourceValue, targetValue := concreteTransformPair(source, target, ta)
	temp := presenceTempName(targetVar)
	var conversion *jen.Statement
	var err error
	if transformUsesHelper(sourceValue, targetValue) {
		conversion = new(jen.Statement).
			Add(Expr(temp)).Op(":=").
			Id(transformHelperName(sourceValue, targetValue, ta)).
			Call(transformCallArguments(sourceValue, targetValue, "&actual", ta)...)
	} else {
		conversion, err = transformAttributeStmt(sourceValue, targetValue, "actual", temp, true, ta)
	}
	if err != nil {
		return nil, err
	}
	assignment := Expr(targetVar + " = " + temp)
	if ta.TargetCtx.FieldPresence(targetParent, name, target) == OptionalPresence {
		assignment = presenceValueAssignment(targetValue, temp, targetVar, "OptionalValue", transformUsesHelper(sourceValue, targetValue), ta.TargetCtx)
	} else if nativeObjectFieldPointer(targetParent, name, target, ta.TargetCtx) && !concreteTransformProducesPointer(targetValue, ta.TargetCtx) {
		assignment = Expr(targetVar + " = &" + temp)
	}
	stmt := &jen.Statement{}
	stmt.If(Expr("actual, ok := " + sourceVar + ".Value(); ok")).BlockFunc(func(group *jen.Group) {
		group.Add(conversion).Line()
		group.Add(assignment)
	})
	assignment, err = ta.DefaultForField(sourceParent, targetParent, name).assignment(targetVar, ta.TargetCtx.FieldPresence(targetParent, name, target), ta)
	if err != nil {
		return nil, err
	}
	if assignment != nil {
		stmt.Else().Block(assignment)
	}
	return stmt, nil
}

func transformNativeToOptional(
	sourceParent, targetParent *expr.MappedAttributeExpr,
	source, target *expr.AttributeExpr,
	sourceVar, targetVar, name string,
	ta *TransformAttrs,
) (*jen.Statement, error) {
	sourceValue, targetValue := concreteTransformPair(source, target, ta)
	temp := presenceTempName(targetVar)
	actual := sourceVar
	guard := ""
	if nativeObjectFieldPointer(sourceParent, name, source, ta.SourceCtx) {
		guard = sourceVar + " != nil"
		actual = "*" + sourceVar
	} else if defaultSourceUsesNil(source) {
		guard = sourceVar + " != nil"
	}
	objectTarget := expr.IsObject(targetValue.Type)
	userObject := transformUsesHelper(sourceValue, targetValue)
	var conversion *jen.Statement
	var err error
	if userObject {
		conversion = new(jen.Statement).
			Add(Expr(temp)).Op(":=").
			Id(transformHelperName(sourceValue, targetValue, ta)).
			Call(transformCallArguments(sourceValue, targetValue, sourceVar, ta)...)
	} else {
		conversion, err = transformAttributeStmt(sourceValue, targetValue, actual, temp, true, ta)
	}
	if objectTarget && !userObject {
		conversion, err = transformObjectValueIntoExisting(sourceValue, targetValue, actual, temp, ta)
	}
	if err != nil {
		return nil, err
	}
	body := &jen.Statement{}
	if objectTarget && !userObject {
		body.Add(Expr(temp + ", _ := " + targetVar + ".Value()")).Line()
	}
	body.Add(conversion).Line()
	body.Add(presenceValueAssignment(targetValue, temp, targetVar, "OptionalValue", userObject, ta.TargetCtx))
	if guard == "" {
		return body, nil
	}
	stmt := &jen.Statement{}
	stmt.If(Expr(guard)).Block(body)
	assignment, err := ta.DefaultForField(sourceParent, targetParent, name).assignment(targetVar, OptionalPresence, ta)
	if err != nil {
		return nil, err
	}
	if assignment != nil {
		stmt.Else().Block(assignment)
	}
	return stmt, nil
}

func transformNullablePresence(source, target *expr.AttributeExpr, sourceVar, targetVar string, newVar bool, plan DefaultApplication, ta *TransformAttrs) (*jen.Statement, error) {
	collectionLayoutDiffers := ta.SourceCtx.CollectionElementPresence != ta.TargetCtx.CollectionElementPresence &&
		expr.ContainsNonNullableCollectionElement(source)
	if plan.Value == nil && ta.failures.sourceDefaultNode(source, target, ta) == nil && !collectionLayoutDiffers && nullablePhysicalTypeRef(source, ta.SourceCtx) == nullablePhysicalTypeRef(target, ta.TargetCtx) {
		return directAssignment(sourceVar, targetVar, newVar), nil
	}
	sourceValue, targetValue := concreteTransformPair(source, target, ta)
	temp := presenceTempName(targetVar)
	objectTarget := expr.IsObject(targetValue.Type)
	userObject := transformUsesHelper(sourceValue, targetValue)
	var conversion *jen.Statement
	var err error
	if userObject {
		conversion = new(jen.Statement).
			Add(Expr(temp)).Op(":=").
			Id(transformHelperName(sourceValue, targetValue, ta)).
			Call(transformCallArguments(sourceValue, targetValue, "&actual", ta)...)
	} else {
		conversion, err = transformAttributeStmt(sourceValue, targetValue, "actual", temp, true, ta)
	}
	if objectTarget && !userObject {
		conversion, err = transformObjectValueIntoExisting(sourceValue, targetValue, "actual", temp, ta)
	}
	if err != nil {
		return nil, err
	}
	targetValueRef := presenceValueTypeRef(target, ta.TargetCtx)
	stmt := &jen.Statement{}
	if newVar {
		stmt.Var().Add(Expr(targetVar)).Add(TypeRef(nullablePhysicalTypeRef(target, ta.TargetCtx))).Line()
	}
	nullAssignment := targetVar + " = loom.NullValue[" + targetValueRef + "]()"
	if objectTarget {
		nullAssignment = targetVar + ".SetNull()"
	}
	stmt.If(Expr(sourceVar + ".IsNull()")).Block(
		Expr(nullAssignment),
	).Else().If(Expr("actual, ok := " + sourceVar + ".Value(); ok")).BlockFunc(func(group *jen.Group) {
		if objectTarget && !userObject {
			group.Add(Expr(temp + ", _ := " + targetVar + ".Value()")).Line()
		}
		group.Add(conversion).Line()
		group.Add(presenceValueAssignment(targetValue, temp, targetVar, "NullableValue", userObject, ta.TargetCtx))
	})
	assignment, err := plan.assignment(targetVar, NullablePresence, ta)
	if err != nil {
		return nil, err
	}
	if assignment != nil {
		stmt.Else().Block(assignment)
	}
	return stmt, nil
}

// presenceValueAssignment preserves the target's declared type when wrapping a
// converted value. Generic constructor inference would use an unnamed collection.
func presenceValueAssignment(target *expr.AttributeExpr, value, targetVar, constructor string, userObject bool, context *AttributeContext) *jen.Statement {
	if expr.IsObject(target.Type) || isNamedCollection(target) {
		if userObject {
			value = "*" + value
		}
		return Expr(targetVar + ".SetValue(" + value + ")")
	}
	if concreteTransformProducesPointer(target, context) {
		value = "*" + value
	}
	return Expr(targetVar + " = loom." + constructor + "(" + value + ")")
}

// isNamedCollection reports whether a presence wrapper needs the declared
// collection type instead of the unnamed slice or map produced by conversion.
func isNamedCollection(attribute *expr.AttributeExpr) bool {
	_, named := attribute.Type.(expr.UserType)
	return named && (expr.IsArray(attribute.Type) || expr.IsMap(attribute.Type))
}

func transformRawAnyPresence(source, target *expr.AttributeExpr, sourceVar, targetVar string, newVar, targetNullable bool, ta *TransformAttrs) (*jen.Statement, error) {
	stmt := &jen.Statement{}
	if targetNullable {
		sourceValue, targetValue := concreteTransformPair(source, target, ta)
		temp := presenceTempName(targetVar)
		conversion, err := transformAttributeStmt(sourceValue, targetValue, sourceVar, temp, true, ta)
		if err != nil {
			return nil, err
		}
		stmt.Add(conversion).Line()
		if newVar {
			stmt.Var().Add(Expr(targetVar)).Add(TypeRef(nullablePhysicalTypeRef(target, ta.TargetCtx))).Line()
		}
		stmt.If(Expr(temp + " == nil")).Block(
			Expr(targetVar + " = loom.NullValue[" + presenceValueTypeRef(target, ta.TargetCtx) + "]()"),
		).Else().Block(
			Expr(targetVar + " = loom.NullableValue(" + temp + ")"),
		)
		return stmt, nil
	}
	if newVar {
		stmt.Var().Add(Expr(targetVar)).Add(TypeRef(ta.TargetCtx.Scope.Ref(target, ta.TargetCtx.Pkg(target)))).Line()
	}
	sourceValue, targetValue := concreteTransformPair(source, target, ta)
	temp := presenceTempName(targetVar)
	conversion, err := transformAttributeStmt(sourceValue, targetValue, "actual", temp, true, ta)
	if err != nil {
		return nil, err
	}
	stmt.If(Expr(sourceVar + ".IsNull()")).Block(
		Expr(targetVar + " = nil"),
	).Else().If(Expr("actual, ok := " + sourceVar + ".Value(); ok")).BlockFunc(func(group *jen.Group) {
		group.Add(conversion).Line()
		group.Add(Expr(targetVar + " = " + temp))
	})
	return stmt, nil
}

func presenceTempName(targetVar string) string {
	return Goify(targetVar+"Value", false)
}

func transformObjectValueIntoExisting(source, target *expr.AttributeExpr, sourceVar, targetVar string, ta *TransformAttrs) (*jen.Statement, error) {
	initFields, postInitCode, err := buildTransformObjectInit(source, target, sourceVar, targetVar, ta)
	if err != nil {
		return nil, err
	}
	stmt := &jen.Statement{}
	for _, field := range initFields {
		stmt.Add(Expr(targetVar)).Dot(field.Name).Op("=").Add(field.Expr).Line()
	}
	fieldCode, err := buildTransformObjectFieldCode(source, target, sourceVar, targetVar, ta)
	if err != nil {
		return nil, err
	}
	appendStatementsWithSpacing(stmt, postInitCode, fieldCode)
	return stmt, nil
}

func nullablePhysicalTypeRef(attribute *expr.AttributeExpr, context *AttributeContext) string {
	return "loom.Nullable[" + presenceValueTypeRef(attribute, context) + "]"
}

func presenceValueTypeRef(attribute *expr.AttributeExpr, context *AttributeContext) string {
	concrete := representation.ConcreteAttribute(attribute)
	if expr.IsArray(concrete.Type) || expr.IsMap(concrete.Type) {
		// Render the collection as its declaring type does so that element
		// pointers, defaults and inline struct tags match.
		return context.CollectionElementRef(concrete, false)
	}
	return context.Scope.Name(concrete, context.Pkg(concrete), context)
}

func concreteTransformProducesPointer(attribute *expr.AttributeExpr, context *AttributeContext) bool {
	if expr.IsObject(attribute.Type) {
		return true
	}
	if expr.IsUnion(attribute.Type) {
		return strings.HasPrefix(context.Scope.Ref(attribute, context.Pkg(attribute)), "*")
	}
	return false
}

func nativeObjectFieldPointer(parent *expr.MappedAttributeExpr, name string, attribute *expr.AttributeExpr, context *AttributeContext) bool {
	if parent == nil {
		return expr.IsObject(attribute.Type)
	}
	return expr.ServiceFieldLayout(parent.AttributeExpr, name, attribute, context.Pointer, context.UseDefault).Pointer
}

func isAnyPresenceAttribute(attribute *expr.AttributeExpr) bool {
	return attribute != nil && expr.IsAny(attribute.Type)
}
