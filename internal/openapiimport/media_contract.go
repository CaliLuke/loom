package openapiimport

import (
	"fmt"
	"maps"
	"mime"
	"slices"
	"strconv"
	"strings"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"

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
		p.mediaBranchSchema(branch, media, mediaPath, false, mediaSchemaContext{})
		if !isMultipartMediaType(media) {
			role := schemaEncodedValue
			if !isJSONMediaType(media) && !isFormMediaType(media) {
				role = schemaRawBody
			}
			p.schemaEncoding(branch, mediaPath+"/schema", role, make(map[schemaEncodingOccurrence]bool))
		}
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
	if sourceType := schemaEncodingType(schema, r.schemas); sourceType != "" {
		r.line("Meta(%q, %q)", "http:encoding:sourceType", sourceType)
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

// mediaBranchSchema admits native roots independently of media selection and
// union nesting. customMultipartMap preserves the ordinary single-media custom
// codec seam; it never propagates into generated untagged alternatives.
func (p *documentPlanner) mediaBranchSchema(branch *Schema, media, path string, customMultipartMap bool, context mediaSchemaContext) {
	resolved := p.urlSchema(branch)
	if resolved == nil {
		return
	}
	base, _, err := mime.ParseMediaType(media)
	if err != nil {
		return // The media map validator reports malformed keys.
	}
	p.mediaScalarSchema(resolved, base, path)
	if !isFormMediaType(media) && !isMultipartMediaType(media) {
		return
	}
	if context.active == nil {
		context.active = make(map[*Schema]bool)
	}
	if context.active[resolved] {
		return // The expression grammar owns unconsumed-cycle diagnostics.
	}
	context.active[resolved] = true
	defer delete(context.active, resolved)
	local := p.validatedMediaEncoding(branch, path)
	context.encoding = mergeMediaEncoding(context.encoding, local)
	if p.urlShape(branch).Nullable {
		p.unsupported("media-schema", path, "form and multipart roots cannot encode null")
		return
	}
	if len(resolved.OneOf) > 0 && len(resolved.MediaTypes) == 0 {
		for index, alternative := range resolved.OneOf {
			p.mediaBranchSchema(alternative, media, fmt.Sprintf("%s/oneOf/%d", path, index), false, context)
		}
		return
	}
	p.mediaObjectSchema(branch, resolved, media, path, customMultipartMap, context.encoding)
}

func (p *documentPlanner) mediaScalarSchema(resolved *Schema, base, path string) {
	if httpcodec.RequestContentType(base).Kind == httpcodec.Text && (resolved.Type != "string" || resolved.Nullable) {
		p.unsupported("media-schema", path, "text media requires a non-nullable string or byte schema")
	}
	if base == "application/octet-stream" && (resolved.Type != "string" || resolved.Format != "binary" || resolved.Nullable) {
		p.unsupported("media-schema", path, "octet-stream media requires a non-nullable binary string schema")
	}
}

func (p *documentPlanner) validatedMediaEncoding(branch *Schema, path string) map[string]httpencoding.Serialization {
	local := p.mediaEncodingFor(branch)
	if len(local) > 0 {
		names := make(map[string]bool)
		p.mediaPropertyNames(branch, names, make(map[*Schema]bool))
		for _, name := range slices.Sorted(maps.Keys(local)) {
			if !names[name] {
				p.unsupported("media-encoding", path+"/encoding/"+escapeJSONPointer(name), "encoding must name a body property")
			}
		}
	}
	return local
}

func (p *documentPlanner) mediaObjectSchema(branch, resolved *Schema, media, path string, customMultipartMap bool, encoding map[string]httpencoding.Serialization) {
	if resolved.Nullable || resolved.Type != "object" {
		p.unsupported("media-schema", path, "form and multipart roots require non-nullable object or map branches")
		return
	}
	representation := classifyObject(resolved)
	if representation == objectTypedRecord || representation == objectFreeMap || !isFormMediaType(media) && !customMultipartMap && representation == objectTypedMap {
		p.unsupported("media-schema", path, "this object member policy has no native form or multipart root codec")
		return
	}
	if representation == objectTypedMap {
		if isFormMediaType(media) {
			// Map entries are dynamically named form fields. Resolve the value
			// with the same default contract as expr.HTTPFormBodyEncoding;
			// the enclosing map's URL-parameter shape is unrelated.
			shape := p.urlShape(resolved.AdditionalProperties.Schema)
			if _, _, err := httpencoding.Resolve(httpencoding.FormField, "", shape, httpencoding.Serialization{}); err != nil {
				p.unsupported("media-schema", path, "form map value: "+err.Error())
			}
		} else if shape := p.urlShape(resolved); shape.Shape != httpencoding.ScalarMap {
			p.unsupported("media-schema", path, "custom multipart root map values require scalar codecs")
		}
		return
	}
	p.mediaFieldEncodings(branch, media, path, encoding)
}

func (p *documentPlanner) mediaFieldEncodings(branch *Schema, media, path string, encoding map[string]httpencoding.Serialization) {
	properties := make(map[string]*Schema)
	p.formProperties(branch, properties, make(map[*Schema]bool))
	var bindings []httpencoding.Binding
	for _, name := range slices.Sorted(maps.Keys(properties)) {
		property := p.urlSchema(properties[name])
		shape := p.urlShape(properties[name])
		value := encoding[name]
		fieldPath := path + "/encoding/" + escapeJSONPointer(name)
		if property == nil {
			continue
		}
		if isMultipartMediaType(media) {
			if codec, err := httpencoding.ResolveMultipart(name, shape, value); err != nil {
				p.unsupported("media-encoding", fieldPath, err.Error())
			} else {
				bindings = append(bindings, httpencoding.Binding{Name: name, Keys: codec.Keys})
				p.multipartSchemaEncoding(properties[name], path+"/schema/properties/"+escapeJSONPointer(name), codec)
			}
			continue
		}
		if _, keys, err := httpencoding.Resolve(httpencoding.FormField, name, shape, value); err != nil {
			p.unsupported("media-encoding", fieldPath, err.Error())
		} else {
			bindings = append(bindings, httpencoding.Binding{Name: name, Keys: keys})
		}
	}
	if err := httpencoding.CheckNamespaces(httpencoding.FormField, bindings); err != nil {
		p.unsupported("media-encoding", path+"/encoding", err.Error())
	}
}
