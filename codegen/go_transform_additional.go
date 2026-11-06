package codegen

import (
	"strconv"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/expr"
)

// additionalMembersPath names the object whose undeclared members collide in
// the error of the generated check: the name of the target or else the source
// user type, or else the field path of the inline object within the value the
// transform converts. Generated variable names do not appear in it.
func additionalMembersPath(source, target *expr.AttributeExpr, sourceVar string) string {
	for _, attribute := range []*expr.AttributeExpr{target, source} {
		if named, ok := attribute.Type.(expr.UserType); ok {
			return named.Name()
		}
	}
	if _, field, ok := strings.Cut(sourceVar, "."); ok && field != "" {
		return field
	}
	return "object"
}

// transformAdditionalMembers converts the map that carries the undeclared
// members of typed objects, then rejects members that reuse a declared or
// synthesized member's wire name. The check is a failure of the conversion,
// so no output carrying the collision is published.
func transformAdditionalMembers(source, target *expr.AttributeExpr, sourceVar, targetVar string, ta *TransformAttrs) (*jen.Statement, error) {
	srcCarrier, tgtCarrier := source.AdditionalMembersCarrier(), target.AdditionalMembersCarrier()
	if srcCarrier == nil || tgtCarrier == nil {
		return nil, nil
	}
	srcVar := sourceVar + "." + expr.AdditionalAttributesFieldName
	tgtVar := targetVar + "." + expr.AdditionalAttributesFieldName
	code, err := transformAttributeStmt(srcCarrier, tgtCarrier, srcVar, tgtVar, false, ta)
	if err != nil {
		return nil, err
	}
	stmt := &jen.Statement{}
	stmt.Add(code)
	names := target.PhysicalWireNames()
	if len(names) == 0 {
		return stmt, nil
	}
	arguments := make([]string, 0, len(names)+2)
	arguments = append(arguments, srcVar, strconv.Quote(additionalMembersPath(source, target, sourceVar)))
	for _, name := range names {
		arguments = append(arguments, strconv.Quote(name))
	}
	stmt.Line().If(Expr("*transformErr == nil")).Block(
		Expr("*transformErr = loom.CheckAdditionalMembers(" + strings.Join(arguments, ", ") + ")"),
	)
	return stmt, nil
}
