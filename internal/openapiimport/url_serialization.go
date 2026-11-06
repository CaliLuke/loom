package openapiimport

import (
	"maps"
	"slices"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"

	"github.com/CaliLuke/loom/internal/httpencoding"
	"github.com/CaliLuke/loom/internal/openapiref"
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
	var bindings []httpencoding.Binding
	for _, name := range slices.Sorted(maps.Keys(properties)) {
		_, keys, err := httpencoding.Resolve(httpencoding.FormField, name, p.urlShape(properties[name]), body.FormEncoding[name])
		if err != nil {
			p.unsupported("form-serialization", path+"/encoding/"+escapeJSONPointer(name), err.Error())
		} else {
			bindings = append(bindings, httpencoding.Binding{Name: name, Keys: keys})
		}
	}
	if err := httpencoding.CheckNamespaces(httpencoding.FormField, bindings); err != nil {
		p.unsupported("form-serialization", path+"/encoding", err.Error())
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
		name, err := openapiref.Name(schema.Ref, "#/components/schemas/")
		if err != nil || seen[name] {
			return nil
		}
		seen[name] = true
		schema = p.schemas[name].Schema
	}
	return schema
}
