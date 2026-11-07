package openapiimport

import (
	"fmt"

	"github.com/CaliLuke/loom/internal/httpencoding"
)

type (
	schemaEncodingRole uint8

	schemaEncodingOccurrence struct {
		schema *Schema
		role   schemaEncodingRole
	}
)

const (
	schemaEncodedValue schemaEncodingRole = iota
	schemaRawBody
)

// schemaEncoding admits string representations at actual wire occurrences.
// Declarations alone do not select a codec; references preserve the occurrence
// role, while structural children use their own role. The path-local guard keeps
// recursive references finite without granting a raw occurrence's permission to
// a separate encoded occurrence of the same declaration.
func (p *documentPlanner) schemaEncoding(schema *Schema, path string, role schemaEncodingRole, active map[schemaEncodingOccurrence]bool) {
	if schema == nil {
		return
	}
	occurrence := schemaEncodingOccurrence{schema, role}
	if active[occurrence] {
		return
	}
	active[occurrence] = true
	defer delete(active, occurrence)
	if schema.Ref != "" {
		renderer := renderer{schemas: p.schemas}
		resolved, err := renderer.resolveSchemaReference(schema, path)
		if err != nil {
			return // Existing schema admission reports invalid references.
		}
		p.schemaEncoding(resolved, path, role, active)
	}
	if schema.Type == "string" && stringRepresentation(schema.Format).raw && role != schemaRawBody {
		p.unsupported("schema-binary-location", path, "binary represents raw octets and requires a whole raw response body or multipart part")
	}
	for _, property := range schema.Properties {
		p.schemaEncoding(property.Schema, path+"/properties/"+escapeJSONPointer(property.Name), schemaEncodedValue, active)
	}
	p.schemaEncoding(schema.Items, path+"/items", schemaEncodedValue, active)
	if schema.AdditionalProperties != nil {
		p.schemaEncoding(schema.AdditionalProperties.Schema, path+"/additionalProperties", schemaEncodedValue, active)
	}
	for index, base := range schema.Bases {
		p.schemaEncoding(base, fmt.Sprintf("%s/allOf/%d", path, index), role, active)
	}
	for index, branch := range schema.OneOf {
		p.schemaEncoding(branch, fmt.Sprintf("%s/oneOf/%d", path, index), role, active)
	}
}

// multipartSchemaEncoding applies the resolved field contract to schema wire
// occurrences. Only actual raw parts receive raw permission: Items consumes one
// array level, and an inner JSON array retains ordinary encoded-value checks.
func (p *documentPlanner) multipartSchemaEncoding(schema *Schema, path string, codec httpencoding.MultipartCodec) {
	if codec.Application == httpencoding.Items {
		resolved := p.urlSchema(schema)
		if resolved == nil {
			return
		}
		if codec.Kind == httpencoding.BranchPart {
			for index, branch := range resolved.OneOf {
				p.multipartSchemaEncoding(branch, fmt.Sprintf("%s/oneOf/%d", path, index), codec.Branches[index])
			}
			return
		}
		schema = resolved.Items
		path += "/items"
	}
	role := schemaEncodedValue
	if codec.Kind == httpencoding.BinaryPart {
		role = schemaRawBody
	}
	p.schemaEncoding(schema, path, role, make(map[schemaEncodingOccurrence]bool))
}
