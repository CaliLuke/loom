package ir

import (
	"fmt"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
)

type (
	schemaFingerprintInput struct {
		attribute  *expr.AttributeExpr
		occurrence expr.ValueOccurrence
	}
	schemaFingerprintAnalysis struct {
		context *expr.ValueContext
		err     error
	}
	schemaFingerprintBranch struct {
		branch *expr.NamedAttributeExpr
		input  schemaFingerprintInput
	}
)

func (e *canonicalSchemaEncoder) captureInput(attribute *expr.AttributeExpr) schemaFingerprintInput {
	input := schemaFingerprintInput{attribute: attribute}
	if e.checked == nil || e.checked.err != nil || attribute == nil {
		return input
	}
	input.occurrence, e.checked.err = e.checked.context.NewOccurrence(attribute)
	if e.checked.err == nil {
		// Representation admission is mandatory even without annotation values.
		_, e.checked.err = e.checked.plan(input)
	}
	return input
}

func (e *canonicalSchemaEncoder) childInput(attribute *expr.AttributeExpr, occurrence expr.ValueOccurrence) schemaFingerprintInput {
	if e.checked != nil && e.checked.err == nil && occurrence == (expr.ValueOccurrence{}) {
		e.checked.err = fmt.Errorf("missing captured fingerprint occurrence")
	}
	return schemaFingerprintInput{attribute: attribute, occurrence: occurrence}
}

func (e *canonicalSchemaEncoder) objectMembers(object *expr.Object, input schemaFingerprintInput) []expr.ValueMember {
	if e.checked == nil || e.checked.err != nil {
		return nil
	}
	members := input.occurrence.Members()
	if len(members) != len(*object) {
		e.checked.err = fmt.Errorf("captured fingerprint object shape differs")
		return nil
	}
	for index, member := range *object {
		if members[index].Name != member.Name {
			e.checked.err = fmt.Errorf("captured fingerprint object member differs")
			return nil
		}
	}
	return members
}

func (e *canonicalSchemaEncoder) unionInputs(union *expr.Union, input schemaFingerprintInput) []schemaFingerprintBranch {
	inputs := make([]schemaFingerprintBranch, len(union.Values))
	var branches []expr.ValueBranch
	if e.checked != nil && e.checked.err == nil {
		branches = input.occurrence.Branches()
		if len(branches) != len(union.Values) {
			e.checked.err = fmt.Errorf("captured fingerprint union shape differs")
		}
	}
	for index, branch := range union.Values {
		inputs[index].branch = branch
		inputs[index].input.attribute = branch.Attribute
		if e.checked != nil && e.checked.err == nil {
			if branches[index].Name != branch.Name {
				e.checked.err = fmt.Errorf("captured fingerprint union branch differs")
				continue
			}
			inputs[index].input.occurrence = branches[index].Occurrence
		}
	}
	return inputs
}

func (c *schemaFingerprintAnalysis) plan(input schemaFingerprintInput) (expr.ValuePlan, error) {
	return c.context.NewValuePlan(input.occurrence, representation.DocumentationPlanRequest(input.attribute,
		expr.ValuePlanRequest{TargetOccurrence: input.occurrence}))
}

func (c *schemaFingerprintAnalysis) annotations(input schemaFingerprintInput) [][]any {
	if c.err != nil {
		return nil
	}
	if input.occurrence == (expr.ValueOccurrence{}) {
		c.err = fmt.Errorf("missing captured fingerprint annotation occurrence")
		return nil
	}
	constraints := input.occurrence.EffectiveConstraints()
	if constraints.DefaultSource().Occurrence == (expr.ValueIdentity{}) && constraints.EnumSource().Occurrence == (expr.ValueIdentity{}) {
		return nil
	}
	plan, err := c.plan(input)
	if err != nil {
		c.err = err
		return nil
	}
	annotations, err := c.context.PrepareJSONSchemaAnnotations(input.occurrence, plan)
	if err != nil {
		c.err = err
		return nil
	}
	return schemaAnnotationEnums(annotations.Enums)
}
