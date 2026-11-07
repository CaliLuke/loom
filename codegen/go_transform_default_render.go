package codegen

import (
	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/expr"
)

// materialization renders the selected absent-branch operation in its owning
// representation. Emission and failure discovery share this exact decision.
func (p DefaultApplication) materialization(ta *TransformAttrs) (DefaultValueCode, error) {
	if p.err != nil {
		return DefaultValueCode{}, p.err
	}
	if p.admittedSource {
		return ta.SourceCtx.DefaultValue(concretePresenceAttribute(p.source), p.Value,
			DefaultValueOptions{MaterializeMembers: true})
	}
	return ta.TargetCtx.DefaultValue(concretePresenceAttribute(p.target), p.Value)
}

func (p DefaultApplication) assignment(targetVar string, presence PresenceKind, ta *TransformAttrs) (*jen.Statement, error) {
	if !p.Apply {
		return nil, nil
	}
	value, err := p.materialization(ta)
	if err != nil {
		return nil, err
	}
	target := concretePresenceAttribute(p.target)
	if !p.admittedSource {
		literal := value.Expression
		if presence != NativePresence {
			if expr.IsObject(target.Type) {
				literal = "*(" + literal + ")"
			}
			return Expr(targetVar + ".SetValue(" + literal + ")"), nil
		}
		if p.targetPointer && expr.IsPrimitive(target.Type) {
			return new(jen.Statement).Add(Expr("var tmp " + GoNativeTypeName(target.Type) + " = " + literal)).Line().Add(Expr(targetVar + " = &tmp")), nil
		}
		return Expr(targetVar + " = " + literal), nil
	}
	source, target := concreteTransformPair(p.source, p.target, ta)
	sourceVar := "default" + Goify(targetVar, true) + "Source"
	convertedVar := "default" + Goify(targetVar, true) + "Target"
	stmt := &jen.Statement{}
	if expr.IsPrimitive(source.Type) {
		stmt.Add(Expr("var " + sourceVar + " " + ta.SourceCtx.Scope.Ref(source, ta.SourceCtx.Pkg(source)) + " = " + value.Expression)).Line()
	} else {
		stmt.Add(Expr(sourceVar + " := " + value.Expression)).Line()
	}
	var conversion *jen.Statement
	if transformUsesHelper(source, target) {
		conversion = new(jen.Statement).Id(convertedVar).Op(":=").Id(transformHelperName(source, target, ta)).Call(transformCallArguments(source, target, sourceVar, ta)...)
	} else {
		conversion, err = transformAttributeStmt(source, target, sourceVar, convertedVar, true, ta)
	}
	if err != nil {
		return nil, err
	}
	stmt.Add(conversion).Line()
	if presence != NativePresence {
		constructor := "OptionalValue"
		if presence == NullablePresence {
			constructor = "NullableValue"
		}
		stmt.Add(presenceValueAssignment(target, convertedVar, targetVar, constructor, expr.IsObject(target.Type), ta.TargetCtx))
	} else {
		value := convertedVar
		if p.targetPointer && !concreteTransformProducesPointer(target, ta.TargetCtx) {
			value = "&" + value
		}
		stmt.Add(Expr(targetVar + " = " + value))
	}
	return stmt, nil
}
