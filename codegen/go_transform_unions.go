//nolint:errcheck // Generator helpers write only to in-memory buffers/builders.
package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/expr"
)

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
		typeRef = strings.TrimPrefix(typeRef, "*")
		if targetIsPointer {
			typeRef = "*" + typeRef
		}
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

// transformUnionTempVarName returns the name of the variable that holds the
// converted branch value in each case of a union transform. A case assigns
// the target after it declares the variable, so the variable must not shadow
// the variable that the target expression starts with, such as the variable
// of an enclosing union whose branch is a union.
func transformUnionTempVarName(targetVar string) string {
	root := targetVar
	if i := strings.IndexAny(targetVar, ".["); i >= 0 {
		root = targetVar[:i]
	}
	if root == "obj" {
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
	// Nullable carriers must reach attribute dispatch before selecting a concrete
	// object helper; that owner unwraps values and preserves absent/null states.
	c := transformUnionRenderCase{
		CaseName:        srcValue.Name,
		CaseTag:         expr.UnionVariantTag(srcValue),
		SourceFieldName: Goify(srcValue.Name, true),
		TargetFieldName: Goify(targetValue.Name, true),
		SourceAttr:      srcValue.Attribute,
		TargetAttr:      targetValue.Attribute,
		TargetCastType:  ta.TargetCtx.Scope.Ref(targetValue.Attribute, transformUnionCastPkg(targetValue.Attribute, unionPkg, ta)),
		UseHelper: transformUsesHelper(srcValue.Attribute, targetValue.Attribute) &&
			!expr.IsNullable(srcValue.Attribute) && !expr.IsNullable(targetValue.Attribute),
		HelperName: transformHelperName(srcValue.Attribute, targetValue.Attribute, ta),
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
					caseGroup.Id(data.TempVarName).Op(":=").Id(c.HelperName).Call(transformCallArguments(c.SourceAttr, c.TargetAttr, "actual", data.TransformAttrs)...)
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
