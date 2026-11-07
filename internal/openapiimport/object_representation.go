package openapiimport

import "github.com/CaliLuke/loom/internal/openapiref"

type (
	objectRepresentation  uint8
	objectConstructorRole uint8
)

const (
	objectNotApplicable objectRepresentation = iota
	objectStructured
	objectFreeMap
	objectTypedMap
	// objectTypedRecord declares members and admits other members that
	// satisfy its additionalProperties, which may also supply the values of
	// required members it does not declare.
	objectTypedRecord
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
func classifyObject(schema *Schema) objectRepresentation {
	if schema == nil || schema.Ref != "" || schema.Type != "object" {
		return objectNotApplicable
	}
	additional := schema.AdditionalProperties
	members := len(schema.Properties) > 0 || len(schema.Required) > 0 || len(schema.Bases) > 0
	open := additional == nil || additional.Allowed != nil && *additional.Allowed
	switch {
	case !members && open:
		return objectFreeMap
	case !members && additional.Schema != nil:
		return objectTypedMap
	case additional != nil && additional.Schema != nil:
		return objectTypedRecord
	case open && additional != nil:
		// Members with additionalProperties true admit any other member.
		return objectTypedRecord
	case open && len(schema.Bases) == 0 && requiresUndeclared(schema):
		// An open object requires members that only its additional
		// properties describe.
		return objectTypedRecord
	}
	return objectStructured
}

// requiresUndeclared reports whether a schema without bases requires a member
// that its properties do not declare.
func requiresUndeclared(schema *Schema) bool {
	declared := make(map[string]bool, len(schema.Properties))
	for _, property := range schema.Properties {
		declared[property.Name] = true
	}
	for _, name := range schema.Required {
		if !declared[name] {
			return true
		}
	}
	return false
}

// objectAdmitsTypedMembers reports whether the effective object carries a
// policy for members it does not declare: its own additionalProperties policy,
// or one inherited from a reference or allOf base that no closed source
// overrides. Extend keeps an inherited Any policy (true or an unconstrained
// schema) as the carrier, so it counts too. Such objects have a carrier for
// undeclared members and may require members they do not declare.
func (r *renderer) objectAdmitsTypedMembers(schema *Schema) bool {
	return objectAdmitsTypedMembers(schema, r.schemas)
}

// objectAdmitsTypedMembers resolves component references through schemas.
func objectAdmitsTypedMembers(schema *Schema, schemas map[string]NamedSchema) bool {
	typed, closed := objectMemberPolicy(schema, schemas, make(map[*Schema]bool))
	return typed && !closed
}

// objectMemberPolicy reports whether the object's effective policy admits
// typed undeclared members and whether any source closes it.
func objectMemberPolicy(schema *Schema, schemas map[string]NamedSchema, seen map[*Schema]bool) (typed, closed bool) {
	if schema == nil || seen[schema] {
		return false, false
	}
	seen[schema] = true
	if schema.Ref != "" {
		name, err := openapiref.Name(schema.Ref, "#/components/schemas/")
		if err != nil {
			return false, false
		}
		typed, closed = objectMemberPolicy(schemas[name].Schema, schemas, seen)
	}
	if additional := schema.AdditionalProperties; additional != nil {
		if additional.Allowed != nil && !*additional.Allowed {
			return false, true
		}
		typed = true
	} else if classifyObject(schema) == objectTypedRecord {
		typed = true
	}
	for _, base := range schema.Bases {
		baseTyped, baseClosed := objectMemberPolicy(base, schemas, seen)
		typed, closed = typed || baseTyped, closed || baseClosed
	}
	return typed, closed
}

// objectNeedsName identifies occurrences whose DSL constructor cannot establish
// the object itself. Array and union roles retain their existing promotion rule.
func objectNeedsName(schema *Schema, role objectConstructorRole) bool {
	if schema == nil || schema.Ref != "" || schema.Type != "object" {
		return false
	}
	if role == objectArrayItem {
		return true
	}
	if representation := classifyObject(schema); representation != objectStructured && representation != objectTypedRecord {
		return false
	}
	return role == objectUnionBranch || role == objectMapValue || role == objectField && len(schema.Properties) == 0 && len(schema.Bases) == 0
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

// typedSchema returns the schema that undeclared members satisfy, or nil when
// additionalProperties is absent or a boolean.
func (a *AdditionalProperties) typedSchema() *Schema {
	if a == nil {
		return nil
	}
	return a.Schema
}
