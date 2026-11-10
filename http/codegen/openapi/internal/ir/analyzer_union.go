package ir

import (
	"strings"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
)

func (a *Analyzer) ensureUnionBranchSchema(union *expr.Union, val *expr.NamedAttributeExpr) string {
	// Synthetic envelopes retain component annotation authority. Their payloads
	// are not inline positions in the enclosing message's acquisition scope.
	acquisition := a.asyncAcquisition
	a.asyncAcquisition = nil
	defer func() {
		a.asyncAcquisition = acquisition
	}()
	logical := a.unionBranchSchemaKey(union, val)
	projection := a.byteProjection(val.Attribute, valuePlanBranchNode(a.plan, expr.UnionVariantTag(val)))
	key := logical + projection
	if name, ok := a.unionBranchSchemas[key]; ok {
		return toRef(name)
	}

	desired := deterministicUnionBranchSchemaName(union, val)
	name := desired
	fingerprint := fingerprintString(key)
	name = a.Uniquify(name, fingerprint)
	a.unionBranchSchemas[key] = name
	setAcquisitionEntry(a, a.schemaNames, name, representationComponentName{desired: desired, logical: logical, baseline: logical, json: projection != ""})

	restore := a.componentAnnotationScope(logical, logical, exampleContext("component", desired))
	defer restore()
	branchSchema := &Schema{
		Type:        string(openapi.Object),
		Description: syntheticUnionBranchSchemaDescription(val),
		Properties: map[string]*Schema{
			union.GetTypeKey(): {
				Type: string(openapi.String),
				Enum: []any{expr.UnionVariantTag(val)},
			},
			union.GetValueKey(): a.analyzeSchemaPositions(
				val.Attribute,
				childExampleContext(exampleContext("component", desired), "property", union.GetValueKey()),
				valuePlanBranchNode(a.plan, expr.UnionVariantTag(val)),
				valuePlanBranchNode(a.examplePlan, expr.UnionVariantTag(val)),
			),
		},
		Required: []string{union.GetTypeKey(), union.GetValueKey()},
	}
	setAcquisitionEntry(a, a.constructions, branchSchema, schemaConstruction{slots: []schemaSlot{{location: schemaProperty, name: union.GetValueKey()}}})
	setAcquisitionEntry(a, a.schemaFingerprints, name, fingerprint)
	setAcquisitionEntry(a, a.schemas, name, branchSchema)
	return toRef(name)
}

func (a *Analyzer) unionBranchSchemaKey(union *expr.Union, val *expr.NamedAttributeExpr) string {
	fingerprint := a.FingerprintAttribute(val.Attribute)
	return strings.Join([]string{
		union.GetTypeKey(),
		union.GetValueKey(),
		expr.UnionVariantTag(val),
		fingerprint,
	}, ":")
}
