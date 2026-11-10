package ir

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/openapiversion"
)

type discriminatorHintContext struct {
	types         map[string]expr.UserType
	attribute     *expr.AttributeExpr
	owner         expr.UserType
	declarationID string
	compatibility bool
	// parentHint reports that the current named root admitted a parent hint.
	parentHint bool
	// parentRoots holds the named roots that admitted a parent hint, however
	// their own schema is wrapped by composition or other root constructions.
	parentRoots map[*Schema]struct{}
}

// initializeDiscriminatorHints retains declaration identity for ParentHint
// targets independently of whichever components happen to be analyzed first.
func (a *Analyzer) initializeDiscriminatorHints(api *expr.APIExpr, types []expr.UserType, results []*expr.ResultTypeExpr) {
	value, _ := api.Meta.Last("openapi:version")
	target, err := openapiversion.Parse(value)
	if err != nil {
		a.err = err
		return
	}
	a.discriminatorHints.compatibility = target == openapiversion.Target31
	for _, typ := range types {
		a.discriminatorHints.types[typ.Name()] = typ
	}
	for _, typ := range results {
		a.discriminatorHints.types[typ.Name()] = typ
	}
}

// analyzeNamedUnderlyingSchema carries the declaration that owns this exact
// detached root. Nested inline objects never inherit that declaration identity.
func (a *Analyzer) analyzeNamedUnderlyingSchema(attribute *expr.AttributeExpr, owner expr.UserType, context string, noRef ...bool) *Schema {
	state := a.discriminatorHints
	previousAttribute, previousOwner, previousID, previousParent := state.attribute, state.owner, state.declarationID, state.parentHint
	state.attribute, state.owner, state.parentHint = attribute, owner, false
	state.declarationID = a.plan.TargetDeclarationID()
	if state.declarationID == "" {
		state.declarationID = owner.ID()
	}
	defer func() {
		state.attribute, state.owner, state.declarationID, state.parentHint = previousAttribute, previousOwner, previousID, previousParent
	}()
	schema := a.analyzeUnderlyingSchema(attribute, context, noRef...)
	if state.parentHint && schema != nil {
		if state.parentRoots == nil {
			state.parentRoots = make(map[*Schema]struct{})
		}
		state.parentRoots[schema] = struct{}{}
	}
	return schema
}

func (a *Analyzer) applyDiscriminatorHint(schema *Schema, attribute *expr.AttributeExpr, context string) error {
	property, present := attribute.Meta.Last("openapi:discriminator:propertyName")
	union := expr.AsUnion(attribute.Type)
	if !present {
		if union != nil && !union.Untagged && schema.Discriminator != nil {
			return a.discriminatorBranchMappings(schema, union)
		}
		return nil
	}
	if property == "" {
		return fmt.Errorf("OpenAPI discriminator propertyName must not be empty")
	}
	if union != nil && !union.Untagged {
		return fmt.Errorf("OpenAPI discriminator propertyName metadata is not allowed on a tagged union")
	}
	if _, present := attribute.Meta["openapi:discriminator:optional"]; present {
		return fmt.Errorf("OpenAPI discriminator optional is derived from hinted schemas")
	}
	if union == nil {
		projected, kept := a.projectedDiscriminatorProperty(attribute, property)
		if !kept {
			return nil
		}
		property = projected
	}
	schema.Discriminator = &Discriminator{PropertyName: property, Mapping: make(map[string]string), Untagged: union != nil, Object: union == nil}
	if union != nil {
		for _, branch := range union.Values {
			if !discriminatorPropertyRequired(branch.Attribute, property) {
				schema.Discriminator.Optional = true
			}
		}
		return a.discriminatorBranchMappings(schema, union)
	}
	if expr.AsObject(attribute.Type) == nil || a.discriminatorHints.attribute != attribute || a.discriminatorHints.owner == nil {
		return fmt.Errorf("OpenAPI discriminator parent hint requires a named object type")
	}
	a.discriminatorHints.parentHint = true
	schema.Discriminator.Optional = !discriminatorPropertyRequired(attribute, property)
	return a.parentHintMappings(schema, attribute, context)
}

// parentHintMappings resolves the explicit key=Type mappings of a parent hint
// to the direct children they name, in the representation of attribute.
func (a *Analyzer) parentHintMappings(schema *Schema, attribute *expr.AttributeExpr, context string) error {
	for _, mapping := range attribute.Meta["openapi:discriminator:mapping"] {
		separator := strings.LastIndexByte(mapping, '=')
		if separator < 0 || separator == len(mapping)-1 {
			return fmt.Errorf("OpenAPI discriminator parent mapping must use key=Type")
		}
		key, name := mapping[:separator], mapping[separator+1:]
		target := a.discriminatorHints.types[name]
		if target == nil {
			return fmt.Errorf("OpenAPI discriminator mapping %q does not name a declared type", mapping)
		}
		if !directDiscriminatorChild(target, a.discriminatorHints.declarationID) {
			return fmt.Errorf("OpenAPI discriminator mapping type %q does not extend parent %q", name, a.discriminatorHints.owner.Name())
		}
		if _, exists := schema.Discriminator.Mapping[key]; exists {
			return fmt.Errorf("duplicate discriminator mapping key %q", key)
		}
		a.recordProjectedMapping(schema.Discriminator, key, attribute, target, context)
	}
	return nil
}

func directDiscriminatorChild(child expr.UserType, parentID string) bool {
	for _, base := range child.Attribute().CompositionBases() {
		if named, ok := base.(expr.UserType); ok && named.ID() == parentID {
			return true
		}
	}
	return false
}

func discriminatorPropertyRequired(attribute *expr.AttributeExpr, property string) bool {
	object := expr.AsObject(attribute.Type)
	if object == nil || expr.AllowsNull(attribute) {
		return false
	}
	for _, member := range *object {
		if expr.JSONFieldName(expr.ElementName(member.Name), member.Attribute) == property {
			return attribute.IsRequired(member.Name) || attribute.IsRequired(expr.AttributeName(member.Name))
		}
	}
	return slices.Contains(attribute.RequiredAdditionalMembers(), property)
}

func (a *Analyzer) discriminatorBranchMappings(schema *Schema, union *expr.Union) error {
	for index, branch := range sortedUnionValues(union) {
		for _, key := range branch.Attribute.Meta["openapi:discriminator:mapping"] {
			if _, exists := schema.Discriminator.Mapping[key]; exists {
				if !union.Untagged {
					for _, value := range union.Values {
						if expr.UnionVariantTag(value) == key {
							return fmt.Errorf("discriminator mapping key %q conflicts with branch tag", key)
						}
					}
				}
				return fmt.Errorf("duplicate discriminator mapping key %q", key)
			}
			ref := schema.OneOf[index].Ref
			if ref == "" {
				ref = a.discriminatorInlineBranch(union, branch, schema.OneOf[index])
				schema.OneOf[index] = &Schema{Ref: ref}
			}
			schema.Discriminator.Mapping[key] = ref
		}
	}
	return nil
}

func (a *Analyzer) discriminatorInlineBranch(union *expr.Union, branch *expr.NamedAttributeExpr, schema *Schema) string {
	fingerprint := a.FingerprintAttribute(branch.Attribute)
	desired := normalizedOpenAPINamePart(union.TypeName) + normalizedOpenAPINamePart(branch.Name) + "Variant"
	name := a.Uniquify(desired, fingerprint)
	setAcquisitionEntry(a, a.schemas, name, schema)
	setAcquisitionEntry(a, a.schemaFingerprints, name, fingerprint)
	setAcquisitionEntry(a, a.schemaNames, name, representationComponentName{desired: desired, logical: fingerprint, baseline: fingerprint})
	return toRef(name)
}

// Hint metadata participates in provisional component identity; final graph
// equivalence still compares complete rendered discriminator bodies.
func (e *canonicalSchemaEncoder) writeDiscriminatorHintMetadata(meta expr.MetaExpr) {
	if !hasDiscriminatorHintMetadata(meta) {
		return
	}
	var keys []string
	for key := range meta {
		if strings.HasPrefix(key, "openapi:discriminator:") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	e.writeString("discriminator-hint")
	e.writeInt(len(keys))
	for _, key := range keys {
		e.writeString(key)
		e.writeInt(len(meta[key]))
		for _, value := range meta[key] {
			e.writeString(value)
		}
	}
}

func hasDiscriminatorHintMetadata(meta expr.MetaExpr) bool {
	for key := range meta {
		if key == "openapi:discriminator:propertyName" || strings.HasPrefix(key, "openapi:discriminator:extension:") {
			return true
		}
	}
	return false
}
