package codegen

import "fmt"

func formRootRequest(request *RequestData) bool {
	return request.form != nil && (len(request.form.Branches) != 0 || request.form.Element != nil)
}

func mediaFormCodec(variant *MediaVariantData) bool {
	return len(variant.StyledFormFields) != 0 || len(variant.OrdinaryFormFields) != 0 || variant.form != nil && (len(variant.form.Branches) != 0 || variant.form.Element != nil)
}

func renderFormRootRequestEncode(request *RequestData) string {
	encoding := ""
	if formMapBody(request) {
		encoding = renderFormMapEncode(request.ClientBody, request.form, "body")
	} else {
		encoding = renderFormUnionEncode(request.ClientBody, request.form, !request.MustHaveBody)
	}
	return "{\nparameterEncoded, parameterError := func() (string, error) {\n" + encoding + "}()\nif parameterError != nil {\nreturn parameterError\n}\nif parameterError := loomhttp.SetRawFormRequest(req, parameterEncoded); parameterError != nil {\nreturn parameterError\n}\n}\n"
}

func renderFormRootRequestDecode(request *RequestData) string {
	pairs := "parameterFormPairs"
	if request.MustHaveBody && !formRootRequest(request) && len(request.StyledFormFields) == 0 {
		pairs = "_"
	}
	code := pairs + ", parseError := loomhttp.ParseRawFormWithLimit(r, loomhttp.RequestBodyLimit(r.Context()))\nerr = parseError\nif err != nil {\nvar serviceError *loom.ServiceError\nif errors.As(err, &serviceError) {\nreturn payload, serviceError\n}\nreturn payload, loom.DecodePayloadError(loomhttp.SafeDecodePayloadMessage(err))\n}\n"
	if !request.MustHaveBody {
		code += "if parameterFormPairs.HasInput() {\n"
	}
	switch {
	case !formRootRequest(request):
		if len(request.StyledFormFields)+len(request.OrdinaryFormFields) != 0 {
			code += "if parameterError := func() error {\n" + renderStyledFormDecode(request) + "}(); parameterError != nil {\nreturn payload, loom.DecodePayloadError(loomhttp.SafeDecodePayloadMessage(parameterError))\n}\n"
		} else {
			target := "&body"
			if request.OptionalObjectBody {
				target = "body"
			}
			code += "if _, err = loomhttp.DecodeFormValue(r.PostForm, \"\", " + target + "); err != nil {\nvar serviceError *loom.ServiceError\nif errors.As(err, &serviceError) {\nreturn payload, serviceError\n}\nreturn payload, loom.DecodePayloadError(loomhttp.SafeDecodePayloadMessage(err))\n}\n"
		}
	case formMapBody(request):
		code += "if parameterError := func(parameterFormPairs loomhttp.RawPairs, parameterForm map[string][]string) error {\n" + renderFormMapDecode(request.ServerBody, request.form, "body") + "return nil\n}(parameterFormPairs, r.PostForm); parameterError != nil {\nreturn payload, loom.DecodePayloadError(loomhttp.SafeDecodePayloadMessage(parameterError))\n}\n"
	default:
		code += "parameterForm := r.PostForm\n" + renderFormUnionMatcher(request.ServerBody, request.form)
		code += "matched, _, parameterError := matchFormUnion()\nif parameterError != nil {\nreturn payload, loom.DecodePayloadError(loomhttp.SafeDecodePayloadMessage(parameterError))\n}\nbody = matched\n"
	}
	if !request.MustHaveBody {
		if request.OptionalObjectBody {
			code += "} else {\nbody = nil\n"
		}
		code += "}\n"
	}
	return code
}

func renderFormUnionMediaDecode(variant *MediaVariantData, context, target string) string {
	return fmt.Sprintf("loomhttp.WithMediaFormCodec(%s, loomhttp.MediaFormCodec{Decode: func(parameterFormPairs loomhttp.RawPairs, parameterForm map[string][]string) error {\n%smatched, _, err := matchFormUnion()\nif err != nil {\nreturn err\n}\n%s = &matched\nreturn nil\n}})", context, renderFormUnionMatcher(variant.formBody, variant.form), target)
}

func formUnionResponse(response *ResponseData) bool {
	for _, body := range response.ServerBody {
		if body.Media == nil {
			continue
		}
		for _, variant := range body.Media.Variants {
			if variant.form != nil && len(variant.form.Branches) > 0 {
				return true
			}
		}
	}
	return false
}
