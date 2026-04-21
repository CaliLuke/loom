package ir

import (
	"strings"

	"github.com/CaliLuke/loom/expr"
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
	key := userType.Hash()
	if _, ok := seen[key]; ok {
		return false
	}
	seen[key] = struct{}{}
	changed := pruneAttributeForSchemaUsage(userType.Attribute(), usage, seen)
	delete(seen, key)
	if !changed {
		return false
	}
	clearUsageSpecificTypeMetadata(attr)
	clearUsageSpecificTypeMetadata(userType.Attribute())
	synchronizeRequiredAttributes(attr, userType.Attribute().Type)
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
		changed  bool
		filtered expr.Object
	)
	for _, named := range *object {
		if shouldOmitAttributeForSchemaUsage(named.Attribute, usage) {
			changed = true
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
	if attr.Validation != nil && len(attr.Validation.Required) > 0 {
		synchronizeRequiredAttributes(attr, attr.Type)
	}
	clearAttributeExamples(attr)
	return true
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

func synchronizeRequiredAttributes(attr *expr.AttributeExpr, dataType expr.DataType) {
	if attr == nil || attr.Validation == nil || len(attr.Validation.Required) == 0 {
		return
	}
	object := expr.AsObject(dataType)
	if object == nil {
		return
	}
	required := attr.Validation.Required[:0]
	for _, name := range attr.Validation.Required {
		if object.Attribute(name) != nil {
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
