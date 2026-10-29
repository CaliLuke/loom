package openapiimport

import (
	"fmt"
	"maps"
	"mime"
	"slices"
	"strconv"
	"strings"

	v3 "github.com/CaliLuke/libopenapi/datamodel/high/v3"
	"github.com/CaliLuke/libopenapi/orderedmap"

	"github.com/CaliLuke/loom/internal/httpcodec"
	"github.com/CaliLuke/loom/internal/httpencoding"
	"github.com/CaliLuke/loom/internal/openapiref"
)

func (a *analyzer) mediaEncoding(media *v3.MediaType, path string) map[string]httpencoding.Serialization {
	result := make(map[string]httpencoding.Serialization, orderedmap.Len(media.Encoding))
	for name, encoding := range media.Encoding.FromOldest() {
		if encoding == nil {
			continue
		}
		if orderedmap.Len(encoding.Headers) > 0 {
			a.unsupported("media-encoding-headers", path+"/encoding/"+escapeJSONPointer(name)+"/headers", "custom part headers are not in the strict import subset")
		}
		value := httpencoding.Serialization{MediaType: encoding.ContentType, Style: encoding.Style, Explode: encoding.Explode}
		if encoding.AllowReserved || encoding.GoLow() != nil && !encoding.GoLow().AllowReserved.IsEmpty() {
			value.AllowReserved = &encoding.AllowReserved
		}
		result[name] = value
	}
	return result
}

func (p *documentPlanner) mediaContracts(schema *Schema, path string) {
	if len(schema.MediaTypes) != len(schema.OneOf) {
		p.unsupported("media-contracts", path, "each media type requires one schema branch")
		return
	}
	if err := httpcodec.ValidateMediaContracts(schema.MediaTypes); err != nil {
		p.unsupported("media-contracts", path, err.Error())
	}
	for index, media := range schema.MediaTypes {
		branch := schema.OneOf[index]
		mediaPath := path + "/content/" + escapeJSONPointer(media)
		if branch == nil {
			p.unsupported("content-schema", mediaPath+"/schema", "content type has no schema")
			continue
		}
		p.mediaBranchSchema(branch, media, mediaPath)
		role := schemaEncodedValue
		if isMultipartMediaType(media) {
			role = schemaMultipartBody
		} else if !isJSONMediaType(media) && !isFormMediaType(media) {
			role = schemaRawBody
		}
		p.schemaEncoding(branch, mediaPath+"/schema", role, make(map[schemaEncodingOccurrence]bool))
		p.schema(branch, mediaPath+"/schema", objectField)
	}
}

func (r *renderer) mediaTypeMappings(schema *Schema) error {
	for index, media := range schema.MediaTypes {
		branch := schema.OneOf[index]
		name, err := openapiref.Name(branch.Ref, "#/components/schemas/")
		if err != nil {
			return fmt.Errorf("render media contract %q: %w", media, err)
		}
		r.line("MediaType(%q, %q)", media, name)
	}
	return nil
}

func (r *renderer) mediaEncodingMetadata(schema *Schema) {
	for _, name := range slices.Sorted(maps.Keys(schema.MediaEncoding)) {
		encoding := schema.MediaEncoding[name]
		prefix := "openapi:encoding:" + name + ":"
		if encoding.MediaType != "" {
			r.line("Meta(%q, %q)", prefix+"contentType", encoding.MediaType)
		}
		if encoding.Style != "" {
			r.line("Meta(%q, %q)", prefix+"style", encoding.Style)
		}
		if encoding.Explode != nil {
			r.line("Meta(%q, %q)", prefix+"explode", strconv.FormatBool(*encoding.Explode))
		}
		if encoding.AllowReserved != nil {
			r.line("Meta(%q, %q)", prefix+"allowReserved", strconv.FormatBool(*encoding.AllowReserved))
		}
	}
}

func mediaRangeSchema(schema *Schema) bool {
	if schema == nil {
		return false
	}
	for _, media := range schema.MediaTypes {
		if strings.Contains(strings.SplitN(media, ";", 2)[0], "*") {
			return true
		}
	}
	return false
}

func withMediaContentTypeParameter(parameters []Parameter) []Parameter {
	for _, parameter := range parameters {
		if parameter.In == "header" && strings.EqualFold(parameter.Name, "Content-Type") {
			return parameters
		}
	}
	return append(slices.Clone(parameters), Parameter{Name: "Content-Type", In: "header", Schema: &Schema{Type: "string"}})
}

func withMediaContentTypeHeader(headers []NamedHeader) []NamedHeader {
	for _, header := range headers {
		if strings.EqualFold(header.Name, "Content-Type") {
			return headers
		}
	}
	return append(slices.Clone(headers), NamedHeader{Name: "Content-Type", Header: Header{Schema: &Schema{Type: "string"}}})
}

func (p *documentPlanner) mediaEncodingFor(schema *Schema) map[string]httpencoding.Serialization {
	seen := make(map[*Schema]bool)
	for schema != nil && !seen[schema] {
		seen[schema] = true
		if schema.MediaEncoding != nil {
			return schema.MediaEncoding
		}
		if schema.Ref == "" {
			return nil
		}
		name, err := openapiref.Name(schema.Ref, "#/components/schemas/")
		if err != nil {
			return nil
		}
		schema = p.schemas[name].Schema
	}
	return nil
}

func (p *documentPlanner) mediaBranchSchema(branch *Schema, media, path string) {
	resolved := p.urlSchema(branch)
	if resolved == nil {
		return
	}
	base, _, err := mime.ParseMediaType(media)
	if err != nil {
		return // The media map validator reports malformed keys.
	}
	if httpcodec.RequestContentType(base).Kind == httpcodec.Text && (resolved.Type != "string" || resolved.Nullable) {
		p.unsupported("media-schema", path, "text media requires a non-nullable string or byte schema")
	}
	if base == "application/octet-stream" && (resolved.Type != "string" || resolved.Format != "binary" || resolved.Nullable) {
		p.unsupported("media-schema", path, "octet-stream media requires a non-nullable binary string schema")
	}
	if !isFormMediaType(media) && !isMultipartMediaType(media) {
		return
	}
	representation, representationErr := classifyObject(resolved)
	if resolved.Type != "object" || representationErr != nil || representation != objectStructured || !p.mediaFormSchemaSupported(branch, make(map[*Schema]bool)) {
		p.unsupported("media-schema", path, "form and multipart media require object schemas without nullable values or untagged unions")
		return
	}
	properties := make(map[string]*Schema)
	p.formProperties(branch, properties, make(map[*Schema]bool))
	encoding := p.mediaEncodingFor(branch)
	for _, name := range slices.Sorted(maps.Keys(encoding)) {
		if _, exists := properties[name]; !exists {
			p.unsupported("media-encoding", path+"/encoding/"+escapeJSONPointer(name), "encoding must name a body property")
		}
	}
	for _, name := range slices.Sorted(maps.Keys(properties)) {
		property := p.urlSchema(properties[name])
		shape := p.urlShape(property)
		value := encoding[name]
		fieldPath := path + "/encoding/" + escapeJSONPointer(name)
		if property == nil {
			continue
		}
		if isMultipartMediaType(media) {
			if _, err := httpencoding.MultipartContentType(shape, property.Type == "string" && stringRepresentation(property.Format).raw, value.MediaType); err != nil {
				p.unsupported("media-encoding", fieldPath, err.Error())
			}
			value.MediaType = ""
		}
		if err := httpencoding.Match(httpencoding.FormField, shape, value); err != nil {
			p.unsupported("media-encoding", fieldPath, err.Error())
		}
	}
}

// mediaFormSchemaSupported mirrors the evaluated design's recursive form
// admission, including alias occurrences and nested JSON-valued fields.
func (p *documentPlanner) mediaFormSchemaSupported(schema *Schema, seen map[*Schema]bool) bool {
	if schema == nil || seen[schema] {
		return true
	}
	seen[schema] = true
	if schema.Nullable || len(schema.OneOf) > 0 {
		return false
	}
	if schema.Ref != "" {
		name, err := openapiref.Name(schema.Ref, "#/components/schemas/")
		if err == nil && !p.mediaFormSchemaSupported(p.schemas[name].Schema, seen) {
			return false
		}
	}
	for _, property := range schema.Properties {
		if !p.mediaFormSchemaSupported(property.Schema, seen) {
			return false
		}
	}
	for _, base := range schema.Bases {
		if !p.mediaFormSchemaSupported(base, seen) {
			return false
		}
	}
	if schema.AdditionalProperties != nil && !p.mediaFormSchemaSupported(schema.AdditionalProperties.Schema, seen) {
		return false
	}
	return p.mediaFormSchemaSupported(schema.Items, seen)
}
