package openapiimport

import (
	"maps"
	"slices"
	"strings"

	"github.com/CaliLuke/loom/internal/httpencoding"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

func formSerialization(content *orderedmap.Map[string, *v3.MediaType]) map[string]httpencoding.Serialization {
	var fields map[string]httpencoding.Serialization
	if content == nil {
		return nil
	}
	for mediaType, media := range content.FromOldest() {
		if !isFormMediaType(mediaType) || media == nil || media.Encoding == nil {
			continue
		}
		fields = make(map[string]httpencoding.Serialization, media.Encoding.Len())
		for name, encoding := range media.Encoding.FromOldest() {
			if encoding == nil {
				continue
			}
			source := httpencoding.Serialization{MediaType: encoding.ContentType, Style: encoding.Style, Explode: encoding.Explode}
			if encoding.AllowReserved || encoding.GoLow() != nil && !encoding.GoLow().AllowReserved.IsEmpty() {
				source.AllowReserved = &encoding.AllowReserved
			}
			fields[name] = source
		}
	}
	return fields
}

func (p *documentPlanner) querySerialization(parameter Parameter, path string) {
	source := parameter.Serialization
	// Also validate normalized Documents assembled by callers using the public fields.
	if source.Style == "" {
		source.Style = parameter.Style
	}
	if parameter.AllowReserved {
		source.AllowReserved = &parameter.AllowReserved
	}
	if err := httpencoding.Match(httpencoding.NamedQuery, p.urlShape(parameter.Schema), source); err != nil {
		p.unsupported("parameter-serialization", path, err.Error())
	}
}

func (p *documentPlanner) formSerialization(body *RequestBody, path string) {
	renderer := renderer{schemas: p.schemas}
	raw, err := renderer.formBodyRequiresRaw(body.Schema, path+"/schema", make(map[string]struct{}))
	if err != nil {
		p.unsupported("form-serialization", path, err.Error())
		return
	}
	if raw && len(body.FormEncoding) == 0 {
		return // Existing raw-body imports delegate their wire codec to the application.
	}
	properties := make(map[string]*Schema)
	p.formProperties(body.Schema, properties, make(map[*Schema]bool))
	for _, name := range slices.Sorted(maps.Keys(properties)) {
		if err := httpencoding.Match(httpencoding.FormField, p.urlShape(properties[name]), body.FormEncoding[name]); err != nil {
			p.unsupported("form-serialization", path+"/encoding/"+escapeJSONPointer(name), err.Error())
		}
	}
	for _, name := range slices.Sorted(maps.Keys(body.FormEncoding)) {
		if _, exists := properties[name]; !exists {
			p.unsupported("form-serialization", path+"/encoding/"+escapeJSONPointer(name), "encoding must name a form property")
		}
	}
}

func (p *documentPlanner) formProperties(schema *Schema, properties map[string]*Schema, seen map[*Schema]bool) {
	schema = p.urlSchema(schema)
	if schema == nil || seen[schema] {
		return
	}
	seen[schema] = true
	for _, base := range schema.Bases {
		p.formProperties(base, properties, seen)
	}
	for _, property := range schema.Properties {
		properties[property.Name] = property.Schema
	}
}

func (p *documentPlanner) urlSchema(schema *Schema) *Schema {
	seen := make(map[string]bool)
	for schema != nil && schema.Ref != "" {
		if seen[schema.Ref] || !strings.HasPrefix(schema.Ref, "#/components/schemas/") {
			return nil
		}
		seen[schema.Ref] = true
		schema = p.schemas[strings.TrimPrefix(schema.Ref, "#/components/schemas/")].Schema
	}
	return schema
}

func (p *documentPlanner) urlShape(schema *Schema) httpencoding.Shape {
	schema = p.urlSchema(schema)
	if schema == nil {
		return httpencoding.UnsupportedShape
	}
	switch schema.Type {
	case "array":
		return httpencoding.Sequence
	case "object":
		if len(schema.Properties) == 0 && len(schema.Bases) == 0 && schema.AdditionalProperties != nil {
			return httpencoding.Map
		}
		return httpencoding.Object
	case "string", "number", "integer", "boolean":
		return httpencoding.Scalar
	default:
		if len(schema.Bases) > 0 || len(schema.OneOf) > 0 {
			return httpencoding.Object
		}
		return httpencoding.UnsupportedShape
	}
}
