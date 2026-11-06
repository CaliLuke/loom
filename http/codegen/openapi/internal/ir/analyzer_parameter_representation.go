package ir

import "github.com/CaliLuke/loom/expr"

// parameterSchemaTarget consumes the same owned target as parameter examples.
// Retain the original declaration at each plan position: canonical names may
// have JSON and text variants, but unrelated declarations must still collide.
func (a *Analyzer) parameterSchemaTarget(attribute *expr.AttributeExpr, plan expr.ValuePlanNode, codec expr.ValueCodec) (*expr.AttributeExpr, func()) {
	if !plan.Valid() || codec != expr.ValueCodecText && codec != expr.ValueCodecForm {
		return attribute, func() {}
	}
	previous := a.parameterSources
	a.parameterSources = make(map[expr.ValuePlanNode]*expr.AttributeExpr)
	a.captureParameterSources(attribute, plan)
	return plan.Attribute(), func() { a.parameterSources = previous }
}

func (a *Analyzer) captureParameterSources(attribute *expr.AttributeExpr, plan expr.ValuePlanNode) {
	if attribute == nil || !plan.Valid() || a.parameterSources[plan] != nil {
		return
	}
	a.parameterSources[plan] = attribute
	switch typ := attribute.Type.(type) {
	case expr.UserType:
		a.captureParameterSources(typ.Attribute(), plan.Underlying())
	case *expr.Object:
		for _, member := range plan.Members() {
			for _, field := range *typ {
				if field.Name == member.Name {
					a.captureParameterSources(field.Attribute, member.Node)
				}
			}
		}
	case *expr.Array:
		a.captureParameterSources(typ.ElemType, plan.Element())
	case *expr.Map:
		a.captureParameterSources(typ.ElemType, plan.Element())
	case *expr.Union:
		for _, branch := range plan.Branches() {
			for _, field := range typ.Values {
				if expr.UnionVariantTag(field) == branch.Tag {
					a.captureParameterSources(field.Attribute, branch.Node)
				}
			}
		}
	}
}

func (a *Analyzer) componentBaselineFingerprint(attribute *expr.AttributeExpr, typ expr.UserType) string {
	if source := a.parameterSources[a.plan]; source != nil {
		if original, ok := source.Type.(expr.UserType); ok {
			return a.componentFingerprint(source, original)
		}
	}
	return a.componentFingerprint(attribute, typ)
}
