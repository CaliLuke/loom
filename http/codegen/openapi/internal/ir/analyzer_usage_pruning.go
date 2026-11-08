package ir

import (
	"slices"
	"strings"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
)

// usageOmittedMembersMeta lists the JSON names of the members that read/write
// usage pruning removed from an object. The analyzer records it on the pruned
// copy of the object attribute so that schemas, fingerprints and examples of
// the pruned object can tell an omitted member from an undeclared one.
const (
	usageOmittedMembersMeta = "openapi:usage:omitted"
	// usageOmittedAttributesMeta retains declared field identity separately from
	// wire names, so inherited requirements cannot revive a pruned member or
	// mistake it for a required literal additional key.
	usageOmittedAttributesMeta = "openapi:usage:omitted-attributes"
)

func attributeForSchemaUsage(attr *expr.AttributeExpr, usage schemaUsage) *expr.AttributeExpr {
	if attr == nil || attr.Type == expr.Empty || usage == schemaUsageNeutral {
		return attr
	}
	cloned := expr.DupAtt(attr)
	if !pruneAttributeForSchemaUsage(cloned, usage, map[string]struct{}{}) {
		return attr
	}
	return cloned
}

func pruneAttributeForSchemaUsage(attr *expr.AttributeExpr, usage schemaUsage, seen map[string]struct{}) bool {
	if attr == nil || attr.Type == nil || attr.Type == expr.Empty {
		return false
	}
	switch actual := attr.Type.(type) {
	case *expr.Array:
		return pruneAttributeForSchemaUsage(actual.ElemType, usage, seen)
	case *expr.Map:
		return pruneAttributeForSchemaUsage(actual.ElemType, usage, seen)
	case expr.UserType:
		return pruneUserTypeForSchemaUsage(attr, actual, usage, seen)
	case *expr.Object:
		return pruneObjectForSchemaUsage(attr, actual, usage, seen)
	default:
		return false
	}
}

func pruneUserTypeForSchemaUsage(attr *expr.AttributeExpr, userType expr.UserType, usage schemaUsage, seen map[string]struct{}) bool {
	if hasCanonicalOpenAPITypeName(attr.Meta) || hasCanonicalOpenAPITypeName(userType.Attribute().Meta) {
		return false
	}
	key := userType.Hash()
	if _, ok := seen[key]; ok {
		return false
	}
	seen[key] = struct{}{}
	original := expr.AsObject(userType.Attribute().Type)
	changed := pruneAttributeForSchemaUsage(userType.Attribute(), usage, seen)
	delete(seen, key)
	if !changed {
		return false
	}
	clearUsageSpecificTypeMetadata(attr)
	clearUsageSpecificTypeMetadata(userType.Attribute())
	synchronizeRequiredAttributes(attr, original, userType.Attribute())
	clearAttributeExamples(attr)
	clearAttributeExamples(userType.Attribute())
	renameSchemaUsageType(userType, usage)
	return true
}

func pruneObjectForSchemaUsage(attr *expr.AttributeExpr, object *expr.Object, usage schemaUsage, seen map[string]struct{}) bool {
	if object == nil {
		return false
	}
	var (
		changed           bool
		filtered          expr.Object
		omitted           []string
		omittedAttributes []string
	)
	for _, named := range *object {
		if shouldOmitAttributeForSchemaUsage(named.Attribute, usage) {
			changed = true
			omittedAttributes = append(omittedAttributes, named.Name)
			if name := expr.JSONFieldName(expr.ElementName(named.Name), named.Attribute); name != "-" && openapi.MustGenerate(named.Attribute.Meta) {
				omitted = append(omitted, name)
			}
			continue
		}
		if pruneAttributeForSchemaUsage(named.Attribute, usage, seen) {
			changed = true
		}
		filtered = append(filtered, named)
	}
	if !changed {
		return false
	}
	attr.Type = &filtered
	recordUsageOmittedMembers(attr, omitted, omittedAttributes)
	synchronizeRequiredAttributes(attr, object, attr)
	clearAttributeExamples(attr)
	return true
}

// recordUsageOmittedMembers retains both wire names and declared field keys on
// the pruned object. The former forbid removed wire properties; the latter
// remove inherited requirements without conflating literal additional keys.
func recordUsageOmittedMembers(attr *expr.AttributeExpr, omitted, attributes []string) {
	for _, entry := range []struct {
		key    string
		values []string
	}{{usageOmittedMembersMeta, omitted}, {usageOmittedAttributesMeta, attributes}} {
		if len(entry.values) == 0 {
			continue
		}
		if attr.Meta == nil {
			attr.Meta = expr.MetaExpr{}
		}
		names := attr.Meta[entry.key]
		for _, name := range entry.values {
			if !slices.Contains(names, name) {
				names = append(names, name)
			}
		}
		slices.Sort(names)
		attr.Meta[entry.key] = names
	}
}

// usageOmittedMembers returns the sorted JSON names of the members that usage
// pruning removed from the object attr defines, together with the members a
// transport projection removed from a typed body, such as attributes bound to
// path parameters. The service type still declares both, so a typed schema
// forbids them.
func usageOmittedMembers(attr *expr.AttributeExpr) []string {
	if attr == nil {
		return nil
	}
	names := slices.Clone(attr.Meta[usageOmittedMembersMeta])
	for _, member := range attr.OmittedMembers() {
		name := expr.JSONFieldName(expr.ElementName(member.Name), member.Attribute)
		if name != "-" && !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

func shouldOmitAttributeForSchemaUsage(attr *expr.AttributeExpr, usage schemaUsage) bool {
	if attr == nil {
		return false
	}
	switch usage {
	case schemaUsageRequest:
		if value, ok := attr.Meta.Last("openapi:readOnly"); ok {
			return metaBoolValue(value)
		}
	case schemaUsageResponse:
		if value, ok := attr.Meta.Last("openapi:writeOnly"); ok {
			return metaBoolValue(value)
		}
	}
	return false
}

func clearUsageSpecificTypeMetadata(attr *expr.AttributeExpr) {
	if attr == nil || attr.Meta == nil {
		return
	}
	delete(attr.Meta, "openapi:typename")
	delete(attr.Meta, "openapi:typename:canonical")
	delete(attr.Meta, "name:original")
}

func clearAttributeExamples(attr *expr.AttributeExpr) {
	if attr == nil {
		return
	}
	attr.UserExamples = nil
	attr.DefaultValue = nil
}

// synchronizeRequiredAttributes removes from the required names of attr the
// members that pruning omitted from owner, the attribute that defines the
// object. original is the object before pruning. A name that original does not
// declare names a typed additional member when owner has AdditionalAttributes;
// pruning never omits those, so they stay required.
func synchronizeRequiredAttributes(attr *expr.AttributeExpr, original *expr.Object, owner *expr.AttributeExpr) {
	if attr == nil || attr.Validation == nil || len(attr.Validation.Required) == 0 || owner == nil {
		return
	}
	object := expr.AsObject(owner.Type)
	if object == nil {
		return
	}
	typed := owner.AdditionalAttributes != nil
	required := make([]string, 0, len(attr.Validation.Required))
	for _, name := range attr.Validation.Required {
		declared := object.Attribute(name) != nil
		undeclared := original == nil || original.Attribute(name) == nil
		if declared || typed && undeclared {
			required = append(required, name)
		}
	}
	attr.Validation.Required = required
}

func renameSchemaUsageType(userType expr.UserType, usage schemaUsage) {
	if userType == nil {
		return
	}
	suffix := schemaUsageTypeSuffix(usage)
	if suffix == "" {
		return
	}
	switch actual := userType.(type) {
	case *expr.ResultTypeExpr:
		if !strings.HasSuffix(actual.TypeName, suffix) {
			actual.TypeName += suffix
		}
		if actual.UID != "" && !strings.HasSuffix(actual.UID, "#"+suffix) {
			actual.UID += "#" + suffix
		}
	case *expr.UserTypeExpr:
		if !strings.HasSuffix(actual.TypeName, suffix) {
			actual.TypeName += suffix
		}
		if actual.UID != "" && !strings.HasSuffix(actual.UID, "#"+suffix) {
			actual.UID += "#" + suffix
		}
	}
}

func schemaUsageTypeSuffix(usage schemaUsage) string {
	switch usage {
	case schemaUsageRequest:
		return "Request"
	case schemaUsageResponse:
		return "Response"
	default:
		return ""
	}
}

// usagePrunedExample removes from an example value the members that usage
// pruning omitted from attr's schemas. Prepared examples are projected from the
// unpruned value plan, so without this they would carry members that a closed
// or typed pruned schema rejects. Values the pruning did not reach are
// returned unchanged; changed containers are copied.
func usagePrunedExample(value any, attr *expr.AttributeExpr) any {
	if attr == nil || attr.Type == nil {
		return value
	}
	switch actual := attr.Type.(type) {
	case expr.UserType:
		return usagePrunedExample(value, actual.Attribute())
	case *expr.Array:
		elements, ok := value.([]any)
		if !ok {
			return value
		}
		pruned := make([]any, len(elements))
		for index, element := range elements {
			pruned[index] = usagePrunedExample(element, actual.ElemType)
		}
		return pruned
	case *expr.Map:
		members, ok := value.(map[string]any)
		if !ok {
			return value
		}
		pruned := make(map[string]any, len(members))
		for key, member := range members {
			pruned[key] = usagePrunedExample(member, actual.ElemType)
		}
		return pruned
	case *expr.Object:
		members, ok := value.(map[string]any)
		if !ok {
			return value
		}
		return usagePrunedObjectExample(members, attr, actual)
	default:
		return value
	}
}

func usagePrunedObjectExample(members map[string]any, attr *expr.AttributeExpr, object *expr.Object) map[string]any {
	declared := make(map[string]*expr.AttributeExpr, len(*object))
	for _, named := range *object {
		declared[expr.JSONFieldName(expr.ElementName(named.Name), named.Attribute)] = named.Attribute
	}
	omitted := usageOmittedMembers(attr)
	pruned := make(map[string]any, len(members))
	for key, member := range members {
		if slices.Contains(omitted, key) {
			continue
		}
		if child, ok := declared[key]; ok {
			pruned[key] = usagePrunedExample(member, child)
			continue
		}
		pruned[key] = usagePrunedExample(member, attr.AdditionalAttributes)
	}
	return pruned
}
