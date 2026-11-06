package ir

import (
	"net/url"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

func cookieFormEquivalent31(codec httpencoding.Codec, shape httpencoding.BranchShape) bool {
	if codec.Style != httpencoding.CookieStyle {
		return true
	}
	if shape.Shape != httpencoding.Scalar && codec.Explode {
		return false
	}
	switch shape.Shape {
	case httpencoding.Scalar, httpencoding.Sequence:
		return shape.Domain.Unreserved
	case httpencoding.ScalarMap:
		return shape.Keys.Unreserved && shape.Domain.Unreserved
	case httpencoding.Object:
		for _, property := range shape.Properties {
			if !property.Domain.Unreserved || url.QueryEscape(property.WireName) != property.WireName {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func parameterCodecFields(location string, codec httpencoding.Codec) (string, *bool) {
	style := parameterStyleName(codec.Style)
	defaultStyle := "form"
	if location == "path" || location == "header" {
		defaultStyle = "simple"
	}
	var explode *bool
	if style != "deepObject" && codec.Explode != (style == "form" || style == "cookie") {
		explode = boolPtr(codec.Explode)
	}
	if style == defaultStyle {
		style = ""
	}
	return style, explode
}

func parameterStyleName(style httpencoding.Style) string {
	return map[httpencoding.Style]string{
		httpencoding.Simple: "simple", httpencoding.Matrix: "matrix", httpencoding.Label: "label",
		httpencoding.Form: "form", httpencoding.SpaceDelimited: "spaceDelimited", httpencoding.PipeDelimited: "pipeDelimited",
		httpencoding.DeepObject: "deepObject", httpencoding.CookieStyle: "cookie",
	}[style]
}

func applyParameterCodec(parameter *Parameter, source *transportir.Parameter) {
	if source.Codec.Style == httpencoding.JSON {
		parameter.Content = map[string]*MediaType{"application/json": {Schema: parameter.Schema, Example: parameter.Example, Examples: parameter.Examples}}
		parameter.Schema, parameter.Example, parameter.Examples = nil, nil, nil
		parameter.Style, parameter.Explode, parameter.AllowReserved, parameter.AllowEmptyValue = "", nil, false, false
		return
	}
	parameter.Style, parameter.Explode = parameterCodecFields(source.In, source.Codec)
	if source.In == "cookie" {
		parameter.CookieFormEquivalent31 = cookieFormEquivalent31(source.Codec, expr.HTTPValueShape(source.Attribute))
	}
}
