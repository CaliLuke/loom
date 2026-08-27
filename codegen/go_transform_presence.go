package codegen

import (
	"fmt"
	"strings"

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
	if sourcePresence == NativePresence && targetPresence == NativePresence {
		return nil, false, nil
	}
	if sourcePresence == NullablePresence || targetPresence == NullablePresence {
		if sourcePresence != NullablePresence || targetPresence != NullablePresence {
			return nil, true, fmt.Errorf("cannot transform nullable field %s to a non-nullable representation", name)
		}
		code, err := transformNullablePresence(source, target, sourceVar, targetVar, false, targetParent.GetDefault(name), ta)
		return code, true, err
	}
	if sourcePresence == OptionalPresence {
		code, err := transformOptionalToNative(targetParent, source, target, sourceVar, targetVar, name, ta)
		return code, true, err
	}
	code, err := transformNativeToOptional(sourceParent, source, target, sourceVar, targetVar, name, ta)
	return code, true, err
}

func transformOptionalToNative(
	targetParent *expr.MappedAttributeExpr,
	source, target *expr.AttributeExpr,
	sourceVar, targetVar, name string,
	ta *TransformAttrs,
) (*jen.Statement, error) {
	sourceValue := concretePresenceAttribute(source)
	targetValue := concretePresenceAttribute(target)
	temp := presenceTempName(targetVar)
	var conversion *jen.Statement
	var err error
	if presenceUserObjectPair(sourceValue, targetValue) {
		conversion = new(jen.Statement).
			Add(Expr(temp)).Op(":=").
			Id(transformHelperName(sourceValue, targetValue, ta)).
			Call(Expr("&actual"))
	} else {
		conversion, err = transformAttributeStmt(sourceValue, targetValue, "actual", temp, true, ta)
	}
	if err != nil {
		return nil, err
	}
	assignment := Expr(targetVar + " = " + temp)
	if nativeObjectFieldPointer(targetParent, name, target, ta.TargetCtx) && !concreteTransformProducesPointer(targetValue, ta.TargetCtx) {
		assignment = Expr(targetVar + " = &" + temp)
	}
	stmt := &jen.Statement{}
	stmt.If(Expr("actual, ok := " + sourceVar + ".Value(); ok")).BlockFunc(func(group *jen.Group) {
		group.Add(conversion).Line()
		group.Add(assignment)
	})
	if defaultValue := targetParent.GetDefault(name); defaultValue != nil {
		stmt.Else().Block(Expr(targetVar + " = " + formatAttributeGoLiteral(targetValue, defaultValue)))
	}
	return stmt, nil
}

func transformNativeToOptional(
	sourceParent *expr.MappedAttributeExpr,
	source, target *expr.AttributeExpr,
	sourceVar, targetVar, name string,
	ta *TransformAttrs,
) (*jen.Statement, error) {
	sourceValue := concretePresenceAttribute(source)
	targetValue := concretePresenceAttribute(target)
	temp := presenceTempName(targetVar)
	actual := sourceVar
	guard := ""
	if nativeObjectFieldPointer(sourceParent, name, source, ta.SourceCtx) {
		guard = sourceVar + " != nil"
		actual = "*" + sourceVar
	} else if expr.IsArray(source.Type) || expr.IsMap(source.Type) {
		guard = sourceVar + " != nil"
	}
	objectTarget := expr.IsObject(targetValue.Type)
	userObject := presenceUserObjectPair(sourceValue, targetValue)
	var conversion *jen.Statement
	var err error
	if userObject {
		conversion = new(jen.Statement).
			Add(Expr(temp)).Op(":=").
			Id(transformHelperName(sourceValue, targetValue, ta)).
			Call(Expr(sourceVar))
	} else {
		conversion, err = transformAttributeStmt(sourceValue, targetValue, actual, temp, true, ta)
	}
	if objectTarget && !userObject {
		conversion, err = transformObjectValueIntoExisting(sourceValue, targetValue, actual, temp, ta)
	}
	if err != nil {
		return nil, err
	}
	wrapped := temp
	if concreteTransformProducesPointer(targetValue, ta.TargetCtx) {
		wrapped = "*" + temp
	}
	body := &jen.Statement{}
	if objectTarget && !userObject {
		body.Add(Expr(temp + ", _ := " + targetVar + ".Value()")).Line()
	}
	body.Add(conversion).Line()
	if objectTarget {
		value := temp
		if userObject {
			value = "*" + temp
		}
		body.Add(Expr(targetVar + ".SetValue(" + value + ")"))
	} else {
		body.Add(Expr(targetVar + " = loom.OptionalValue(" + wrapped + ")"))
	}
	if guard == "" {
		return body, nil
	}
	stmt := &jen.Statement{}
	stmt.If(Expr(guard)).Block(body)
	return stmt, nil
}

func transformNullablePresence(source, target *expr.AttributeExpr, sourceVar, targetVar string, newVar bool, defaultValue any, ta *TransformAttrs) (*jen.Statement, error) {
	collectionLayoutDiffers := ta.SourceCtx.CollectionElementPresence != ta.TargetCtx.CollectionElementPresence &&
		expr.ContainsNonNullableArrayElement(source)
	if defaultValue == nil && !collectionLayoutDiffers && nullablePhysicalTypeRef(source, ta.SourceCtx) == nullablePhysicalTypeRef(target, ta.TargetCtx) {
		return directAssignment(sourceVar, targetVar, newVar), nil
	}
	sourceValue := concretePresenceAttribute(source)
	targetValue := concretePresenceAttribute(target)
	temp := presenceTempName(targetVar)
	objectTarget := expr.IsObject(targetValue.Type)
	userObject := presenceUserObjectPair(sourceValue, targetValue)
	var conversion *jen.Statement
	var err error
	if userObject {
		conversion = new(jen.Statement).
			Add(Expr(temp)).Op(":=").
			Id(transformHelperName(sourceValue, targetValue, ta)).
			Call(Expr("&actual"))
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
	wrapped := temp
	if concreteTransformProducesPointer(targetValue, ta.TargetCtx) {
		wrapped = "*" + temp
	}
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
		if objectTarget {
			value := temp
			if userObject {
				value = "*" + temp
			}
			group.Add(Expr(targetVar + ".SetValue(" + value + ")"))
		} else {
			group.Add(Expr(targetVar + " = loom.NullableValue(" + wrapped + ")"))
		}
	})
	if defaultValue != nil {
		stmt.Else().Block(Expr(targetVar + " = loom.NullableValue(" + formatAttributeGoLiteral(targetValue, defaultValue) + ")"))
	}
	return stmt, nil
}

func transformRawAnyPresence(source, target *expr.AttributeExpr, sourceVar, targetVar string, newVar bool, ta *TransformAttrs) (*jen.Statement, error) {
	stmt := &jen.Statement{}
	if expr.IsNullable(target) {
		sourceValue := concretePresenceAttribute(source)
		targetValue := concretePresenceAttribute(target)
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
	sourceValue := concretePresenceAttribute(source)
	targetValue := concretePresenceAttribute(target)
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
	concrete := concretePresenceAttribute(attribute)
	return context.Scope.Name(concrete, context.Pkg(concrete), false, context.UseDefault)
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
	if expr.IsObject(attribute.Type) {
		return true
	}
	if expr.IsUnion(attribute.Type) {
		return parent != nil && !parent.IsRequired(name)
	}
	return expr.IsPrimitive(attribute.Type) && parent != nil && context.IsPrimitivePointer(name, parent.AttributeExpr)
}

func presenceUserObjectPair(source, target *expr.AttributeExpr) bool {
	_, sourceUser := source.Type.(expr.UserType)
	_, targetUser := target.Type.(expr.UserType)
	return sourceUser && targetUser && expr.IsObject(source.Type) && expr.IsObject(target.Type)
}

func isAnyPresenceAttribute(attribute *expr.AttributeExpr) bool {
	concrete := concretePresenceAttribute(attribute)
	concrete.Nullable = false
	return expr.AllowsNull(concrete)
}
