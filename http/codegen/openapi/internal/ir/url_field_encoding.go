package ir

import (
	"fmt"
	"strconv"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

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
	if object := expr.AsObject(body.Type); object != nil {
		for _, field := range *object {
			name := expr.ElementName(field.Name)
			_, _, err := httpencoding.Resolve(httpencoding.FormField, name, expr.HTTPValueShape(field.Attribute), expr.HTTPParameterSerialization(body, "openapi:encoding:"+name+":"))
			if err != nil {
				return fmt.Errorf("form field %q: %w", name, err)
			}
		}
	}
	return nil
}

func applyFormFieldEncoding(request *transportir.Request, media *MediaType) {
	if !request.FormEncoded || request.SkipBodyEncode || request.DocumentBody != nil {
		return
	}
	if media.Metadata == nil {
		media.Metadata = make(map[string][]string)
	}
	for name, codec := range request.FormFieldCodecs {
		prefix := "openapi:encoding:" + name + ":"
		if codec.Style == httpencoding.JSON {
			media.Metadata[prefix+"contentType"] = []string{"application/json"}
			delete(media.Metadata, prefix+"style")
			delete(media.Metadata, prefix+"explode")
			delete(media.Metadata, prefix+"allowReserved")
			continue
		}
		if codec.Style == httpencoding.Root {
			continue
		}
		// Explicit style owns form encoding even if a contentType was authored.
		// Objects/maps need style:form to override the Encoding Object JSON default.
		delete(media.Metadata, prefix+"contentType")
		if codec.Shape == httpencoding.Object || codec.Shape == httpencoding.ScalarMap || codec.Style != httpencoding.Form || codec.Explode != true {
			media.Metadata[prefix+"style"] = []string{parameterStyleName(codec.Style)}
		} else {
			delete(media.Metadata, prefix+"style")
		}
		_, explode := parameterCodecFields("query", codec)
		if explode == nil {
			delete(media.Metadata, prefix+"explode")
		} else {
			media.Metadata[prefix+"explode"] = []string{strconv.FormatBool(*explode)}
		}
	}
}
