package ir

import (
	"fmt"
	"maps"
	"slices"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

func validateURLFieldEncoding(endpoint *transportir.Endpoint) error {
	for _, parameter := range endpoint.Request.QueryParams {
		if endpoint.Request.MapQueryParams != nil {
			break
		}
		source := fieldSerialization(parameter.Attribute.Meta, "openapi:")
		if parameter.Encoding == httpencoding.JSON {
			source.MediaType = "application/json"
		}
		if err := httpencoding.Match(httpencoding.NamedQuery, transportir.URLShape(parameter.Attribute), source); err != nil {
			return fmt.Errorf("query parameter %q: %w", parameter.HTTPName, err)
		}
	}
	request := endpoint.Request
	if !request.FormEncoded || request.SkipBodyEncode || request.DocumentBody != nil {
		return nil
	}
	for _, name := range slices.Sorted(maps.Keys(request.FormFieldModes)) {
		mode := request.FormFieldModes[name]
		shape := httpencoding.Scalar
		switch mode {
		case httpencoding.JSON:
			shape = httpencoding.Map
		case httpencoding.Repeated:
			shape = httpencoding.Sequence
		}
		source := fieldSerialization(request.Body.Meta, "openapi:encoding:"+name+":")
		if err := httpencoding.Match(httpencoding.FormField, shape, source); err != nil {
			return fmt.Errorf("form field %q: %w", name, err)
		}
	}
	return nil
}

func fieldSerialization(meta expr.MetaExpr, prefix string) httpencoding.Serialization {
	source := httpencoding.Serialization{Style: metaValue(meta, prefix+"style"), MediaType: metaValue(meta, prefix+"contentType")}
	if value, exists := meta.Last(prefix + "explode"); exists {
		source.Explode = boolPtr(value != "false")
	}
	if value, exists := meta.Last(prefix + "allowReserved"); exists {
		source.AllowReserved = boolPtr(value != "false")
	}
	return source
}

func applyFormFieldEncoding(request *transportir.Request, media *MediaType) {
	if !request.FormEncoded || request.SkipBodyEncode || request.DocumentBody != nil {
		return
	}
	for name, mode := range request.FormFieldModes {
		if mode != httpencoding.JSON {
			continue
		}
		if media.Metadata == nil {
			media.Metadata = make(map[string][]string)
		}
		media.Metadata["openapi:encoding:"+name+":contentType"] = []string{"application/json"}
	}
}
