package openapiimport

import "fmt"

type (
	objectRepresentation  uint8
	objectConstructorRole uint8
)

const (
	objectNotApplicable objectRepresentation = iota
	objectStructured
	objectFreeMap
	objectTypedMap
)

const (
	objectDeclaration objectConstructorRole = iota
	objectField
	objectMapValue
	objectArrayItem
	objectUnionBranch
)

// classifyObject selects the shared object carrier without changing authored
// schema information. It preserves the supported record and additional-property
// forms; the shared admission pass checks their remaining schema constraints.
func classifyObject(schema *Schema) (objectRepresentation, error) {
	if schema == nil || schema.Ref != "" || schema.Type != "object" {
		return objectNotApplicable, nil
	}
	additional := schema.AdditionalProperties
	members := len(schema.Properties) > 0 || len(schema.Required) > 0 || len(schema.Bases) > 0
	if additional == nil {
		if members {
			return objectStructured, nil
		}
		return objectFreeMap, nil
	}
	if additional.Schema != nil {
		if len(schema.Properties) > 0 {
			return objectNotApplicable, fmt.Errorf("object properties with schema-valued additionalProperties are not renderable")
		}
		return objectTypedMap, nil
	}
	if additional.Allowed != nil && *additional.Allowed {
		if members {
			return objectNotApplicable, fmt.Errorf("object members with additionalProperties true are not renderable")
		}
		return objectFreeMap, nil
	}
	return objectStructured, nil
}

// objectNeedsName identifies occurrences whose DSL constructor cannot establish
// the object itself. Array and union roles retain their existing promotion rule.
func objectNeedsName(schema *Schema, role objectConstructorRole) bool {
	if schema == nil || schema.Ref != "" || schema.Type != "object" {
		return false
	}
	if role == objectArrayItem || role == objectUnionBranch {
		return true
	}
	representation, err := classifyObject(schema)
	if err != nil || representation != objectStructured {
		return false
	}
	return role == objectMapValue || role == objectField && len(schema.Properties) == 0 && len(schema.Bases) == 0
}

// requestObjectRole reflects the JSON payload field; raw documentation and
// structured form/multipart bodies establish their own object callbacks.
func requestObjectRole(body RequestBody) objectConstructorRole {
	if len(body.ContentTypes) == 1 && isJSONMediaType(body.ContentTypes[0]) {
		return objectField
	}
	return objectDeclaration
}

// responseObjectRole supplies concrete empty JSON objects for either a selected
// field or an explicit whole response body. Raw documentation keeps callbacks.
func responseObjectRole(response Response, allowNonJSON bool) objectConstructorRole {
	if !allowNonJSON || isJSONMediaType(response.ContentType) {
		return objectField
	}
	return objectDeclaration
}
