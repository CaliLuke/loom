package openapiimport

import (
	"fmt"
	"maps"
	"slices"
	"strconv"

	"github.com/CaliLuke/loom/internal/httpencoding"
)

func parameterMetadata(parameter renderedParameter) ([]renderedMetadata, error) {
	var metadata []renderedMetadata
	if parameter.componentName != "" {
		metadata = append(metadata, renderedMetadata{name: "openapi:component:parameter", value: parameter.componentName})
	}
	if parameter.parameter.In == "query" {
		metadata = append(metadata, renderedMetadata{name: "openapi:allowEmptyValue", value: strconv.FormatBool(parameter.parameter.AllowEmptyValue)})
	}
	metadata = append(metadata, parameterCodecMetadata(parameter.parameter.In, parameter.codec)...)
	if value := parameterSerialization(parameter.parameter).AllowReserved; value != nil {
		metadata = append(metadata, renderedMetadata{name: "openapi:allowReserved", value: strconv.FormatBool(*value)})
	}
	extensions, err := renderedExtensions("parameter", parameter.parameter.Extensions)
	return append(metadata, extensions...), err
}

func parameterCodecMetadata(location string, codec httpencoding.Codec) []renderedMetadata {
	if codec.Style == httpencoding.JSON {
		return nil
	}
	style := parameterStyleName(codec.Style)
	defaultStyle := "form"
	if location == "path" || location == "header" {
		defaultStyle = "simple"
	}
	var metadata []renderedMetadata
	// Cookie intent is explicit because OpenAPI and Loom have different defaults.
	// Query maps likewise need form to override the native JSON default.
	if style != defaultStyle || location == "cookie" || location == "query" && (codec.Shape == httpencoding.ScalarMap || codec.Shape == httpencoding.Nested) {
		metadata = append(metadata, renderedMetadata{name: "openapi:style", value: style})
	}
	if codec.Style != httpencoding.DeepObject && codec.Explode != (style == "form" || style == "cookie") {
		metadata = append(metadata, renderedMetadata{name: "openapi:explode", value: strconv.FormatBool(codec.Explode)})
	}
	return metadata
}

func parameterStyleName(style httpencoding.Style) string {
	return map[httpencoding.Style]string{
		httpencoding.Simple: "simple", httpencoding.Matrix: "matrix", httpencoding.Label: "label",
		httpencoding.Form: "form", httpencoding.SpaceDelimited: "spaceDelimited", httpencoding.PipeDelimited: "pipeDelimited",
		httpencoding.DeepObject: "deepObject", httpencoding.CookieStyle: "cookie",
	}[style]
}

func (r *renderer) formEncodingMetadata(body RequestBody) ([]renderedMetadata, error) {
	if len(body.FormEncoding) == 0 || body.Schema != nil && len(body.Schema.MediaTypes) > 0 {
		return nil, nil
	}
	planner := documentPlanner{schemas: r.schemas}
	properties := make(map[string]*Schema)
	planner.formProperties(body.Schema, properties, make(map[*Schema]bool))
	var metadata []renderedMetadata
	for _, name := range slices.Sorted(maps.Keys(body.FormEncoding)) {
		source := body.FormEncoding[name]
		codec, _, err := httpencoding.Resolve(httpencoding.FormField, name, planner.urlShape(properties[name]), source)
		if err != nil {
			return nil, fmt.Errorf("form encoding %q: %w", name, err)
		}
		prefix := "openapi:encoding:" + name + ":"
		if codec.Style == httpencoding.JSON {
			metadata = append(metadata, renderedMetadata{name: prefix + "contentType", value: "application/json"})
			continue
		}
		style := parameterStyleName(codec.Style)
		// Explicit style prevents the Encoding Object's composite JSON default.
		metadata = append(metadata, renderedMetadata{name: prefix + "style", value: style})
		if codec.Style != httpencoding.DeepObject && codec.Explode != (style == "form") {
			metadata = append(metadata, renderedMetadata{name: prefix + "explode", value: strconv.FormatBool(codec.Explode)})
		}
		if source.AllowReserved != nil {
			metadata = append(metadata, renderedMetadata{name: prefix + "allowReserved", value: strconv.FormatBool(*source.AllowReserved)})
		}
	}
	return metadata, nil
}
