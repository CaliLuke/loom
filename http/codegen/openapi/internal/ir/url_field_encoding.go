package ir

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

// exportedFieldEncoding pairs one captured physical field with its authored
// context. One OpenAPI Encoding Object must describe every occurrence of its
// name, even when the body contains nested root alternatives.
type exportedFieldEncoding struct {
	name      string
	shape     httpencoding.BranchShape
	source    httpencoding.Serialization
	form      *httpencoding.Codec
	multipart *httpencoding.MultipartCodec
}

func validateURLFieldEncoding(endpoint *transportir.Endpoint) error {
	for _, group := range []struct {
		location   httpencoding.Location
		parameters []*transportir.Parameter
	}{
		{httpencoding.Path, endpoint.Request.PathParams},
		{httpencoding.NamedQuery, endpoint.Request.QueryParams},
		{httpencoding.Header, endpoint.Request.Headers},
		{httpencoding.Cookie, endpoint.Request.Cookies},
	} {
		for _, parameter := range group.parameters {
			if parameter.MapQueryParams != nil {
				continue
			}
			_, _, err := httpencoding.Resolve(group.location, parameter.HTTPName, expr.HTTPValueShape(parameter.Attribute), expr.HTTPParameterSerialization(parameter.Attribute, "openapi:"))
			if err != nil {
				return fmt.Errorf("%s parameter %q: %w", parameter.In, parameter.HTTPName, err)
			}
		}
	}
	request := endpoint.Request
	if !request.FormEncoded || request.SkipBodyEncode || request.DocumentBody != nil {
		return nil
	}
	body := expr.UnwrapInlineHTTPBody(request.Body)
	if body == nil {
		return nil
	}
	_, err := expr.HTTPFormBodyEncoding(body)
	return err
}

func applyMediaFieldEncoding(request *transportir.Request, media *MediaType) error {
	if request.SkipBodyEncode || request.DocumentBody != nil || (!request.FormEncoded && !request.Multipart) {
		return nil
	}
	var fields []exportedFieldEncoding
	if request.Form != nil {
		fields = formExportFields(request.Form)
	}
	if request.MultipartEncoding != nil {
		fields = multipartExportFields(request.MultipartEncoding)
	}
	if request.Form == nil && request.MultipartEncoding == nil || request.MultipartEncoding != nil && request.MultipartEncoding.Custom {
		return nil
	}
	grouped := make(map[string][]exportedFieldEncoding)
	for _, field := range fields {
		grouped[field.name] = append(grouped[field.name], field)
	}
	metadata := maps.Clone(media.Metadata)
	if metadata == nil {
		metadata = make(map[string][]string)
	}
	// All field slots are rebuilt from the captured context, after full-tag
	// renames and omission. This also removes semantic names from the wire map.
	for key := range metadata {
		if strings.HasPrefix(key, "openapi:encoding:") {
			delete(metadata, key)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(grouped)) {
		alternatives := grouped[name]
		source, err := reconcileFieldEncoding(alternatives)
		if err != nil {
			return fmt.Errorf("field %q: %w", name, err)
		}
		writeFieldEncoding(metadata, name, source)
	}
	media.Metadata = metadata
	projectTaggedFormMediaExamples(media, request.Form)
	return nil
}

func formExportFields(root *expr.HTTPFormEncoding) []exportedFieldEncoding {
	var result []exportedFieldEncoding
	for _, branch := range root.Branches {
		result = append(result, formExportFields(branch)...)
	}
	for _, branch := range root.TaggedBranches {
		if branch.Object != nil {
			result = append(result, formExportFields(branch.Object)...)
		} else {
			codec := branch.Value.Codec
			result = append(result, exportedFieldEncoding{name: branch.Value.Name, shape: expr.HTTPValueShape(branch.Body), form: &codec})
		}
	}
	scoped := *root.Body
	scoped.Meta = root.Metadata
	for _, binding := range slices.Sorted(maps.Keys(root.Fields)) {
		field := root.Fields[binding]
		if field.Omitted {
			continue
		}
		attribute := expr.AsObject(root.Body.Type).Attribute(binding)
		codec := field.Codec
		result = append(result, exportedFieldEncoding{name: field.Name, shape: expr.HTTPValueShape(attribute), source: expr.HTTPParameterSerialization(&scoped, "openapi:encoding:"+expr.ElementName(binding)+":"), form: &codec})
	}
	return result
}

func multipartExportFields(root *expr.HTTPMultipartEncoding) []exportedFieldEncoding {
	var result []exportedFieldEncoding
	for _, branch := range root.Branches {
		result = append(result, multipartExportFields(branch)...)
	}
	scoped := *root.Body
	scoped.Meta = root.Metadata
	for _, binding := range slices.Sorted(maps.Keys(root.Fields)) {
		field := root.Fields[binding]
		if field.Omitted {
			continue
		}
		attribute := expr.AsObject(root.Body.Type).Attribute(binding)
		codec := field.Codec
		result = append(result, exportedFieldEncoding{name: field.Name, shape: expr.HTTPValueShape(attribute), source: expr.HTTPParameterSerialization(&scoped, "openapi:encoding:"+expr.ElementName(binding)+":"), multipart: &codec})
	}
	return result
}

func reconcileFieldEncoding(fields []exportedFieldEncoding) (httpencoding.Serialization, error) {
	// Try authored contracts before the implicit default so explicit media labels
	// survive whenever one label describes the entire field. Resolver comparison
	// determines equivalence; the exporter never invents codec policy.
	candidates := make([]httpencoding.Serialization, 0, len(fields)+1)
	for _, field := range fields {
		source := field.source
		if source.Style != "" || source.Explode != nil || source.AllowReserved != nil {
			source.MediaType = ""
			// allowReserved:false is omitted by the OpenAPI renderer. An explicit
			// default style retains the flag's presence-dependent codec selection.
			if source.Style == "" {
				source.Style = "form"
			}
		}
		if source != (httpencoding.Serialization{}) {
			candidates = append(candidates, source)
		}
	}
	if len(candidates) == 0 && len(fields) > 0 && fields[0].form != nil && fields[0].form.Style == httpencoding.JSON {
		candidates = append(candidates, httpencoding.Serialization{MediaType: "application/json"})
	}
	candidates = append(candidates, httpencoding.Serialization{})
	for _, candidate := range candidates {
		matches := true
		for _, field := range fields {
			if field.form != nil {
				codec, _, err := httpencoding.Resolve(httpencoding.FormField, field.name, field.shape, candidate)
				matches = err == nil && reflect.DeepEqual(codec, *field.form)
			} else {
				codec, err := httpencoding.ResolveMultipart(field.name, field.shape, candidate)
				matches = err == nil && reflect.DeepEqual(codec, *field.multipart)
			}
			if !matches {
				break
			}
		}
		if matches {
			return candidate, nil
		}
	}
	return httpencoding.Serialization{}, fmt.Errorf("root alternatives require incompatible Encoding Objects; OpenAPI provides only one encoding per physical field name")
}

func writeFieldEncoding(metadata map[string][]string, name string, source httpencoding.Serialization) {
	prefix := "openapi:encoding:" + name + ":"
	if source.MediaType != "" {
		metadata[prefix+"contentType"] = []string{source.MediaType}
	}
	if source.Style != "" {
		metadata[prefix+"style"] = []string{source.Style}
	}
	if source.Explode != nil {
		metadata[prefix+"explode"] = []string{strconv.FormatBool(*source.Explode)}
	}
	if source.AllowReserved != nil {
		metadata[prefix+"allowReserved"] = []string{strconv.FormatBool(*source.AllowReserved)}
	}
}
