package codegen

// FileResponse endpoints add http.StatusPartialContent (206) and
// http.StatusNotModified (304) to the 200 success case below because
// net/http.ServeContent (see http/file_response.go) treats both as
// successful outcomes of the same file transfer. The OpenAPI output for
// these endpoints additionally documents 412 (Precondition Failed) and 416
// (Range Not Satisfiable) as real protocol responses (see
// addFileResponseProtocolResponses in
// http/codegen/openapi/internal/ir/document_file_response.go), since
// ServeContent can produce them too. Those two are intentionally NOT added
// to the success case: a failed precondition or unsatisfiable range is not a
// successful file transfer, so treating them as success would hand the
// caller a Result with a body that doesn't match what was asked for. They
// fall through to the decoder's generic unexpected-status branch below,
// which already returns a typed loomhttp.ClientError carrying the real
// status code and response body — a sound, if generic, mapping for a
// declared-but-not-successful response.
var (
	responseDecoderSource = joinHTTPTemplateSource(`{{ printf "%s returns a decoder for responses returned by the %s %s endpoint. restoreBody controls whether the response body should be restored after having been read." .ResponseDecoder .ServiceName .Method.Name | comment }}
{{- if .Errors }}
{{ printf "%s may return the following errors:" .ResponseDecoder | comment }}
	{{- range $gerr := .Errors }}
	{{- range $errors := .Errors }}
//	- {{ printf "%q" .Name }} (type {{ .Ref }}): {{ if $.HasDynamicFinalSelectors }}{{ .Response.Selector.OpenAPIKey }}{{ else }}{{ .Response.LiteralStatus }}{{ end }}{{ if .Response.Description }}, {{ .Response.Description }}{{ end }}
	{{- end }}
	{{- end }}
//	- error: internal error
{{- end }}
func {{ .ResponseDecoder }}(decoder func(*http.Response) loomhttp.Decoder, restoreBody bool) func(*http.Response) (any, error) {
	return func(resp *http.Response) (any, error) {
        return loomhttp.DecodeResponse(resp, restoreBody, {{ if or .Method.SkipResponseBodyEncodeDecode .Method.FileResponse }}false{{ range .Result.Responses }} || resp.StatusCode == {{ .LiteralStatus }}{{ if and $.Method.FileResponse (eq .LiteralCode 200) }} || resp.StatusCode == http.StatusPartialContent || resp.StatusCode == http.StatusNotModified{{ end }}{{ end }}{{ else }}false{{ end }}, func(resp *http.Response) (any, error) {
		{{- if .HasDynamicFinalSelectors }}
		selected, found := loomhttp.SelectFinal([]loomhttp.StatusSelector{ {{ finalSelectorValues . }} }, resp.StatusCode)
		if !found {
			return nil, &loomhttp.StatusViolationError{Code: resp.StatusCode}
		}
		switch selected {
		{{- else }}
		switch resp.StatusCode {
		{{- end }}
	{{- range responsePayloads . }}
		case {{ if $.HasDynamicFinalSelectors }}{{ responseSelectorValue . }}{{ else }}{{ .LiteralStatus }}{{ end }}{{ if and $.Method.FileResponse (eq .LiteralCode 200) }}, http.StatusPartialContent, http.StatusNotModified{{ end }}:
			{{- if $.Method.HasResponseMetadata }}
			metadata, metadataError := {{ $.SSE.DecodeHandshake }}({{ responseSelectorValue . }}, resp)
			if metadataError != nil {
				return nil, metadataError
			}
			{{- end }}
			{{- template "partial_single_response" (buildResponseData . $.ServiceName $.Method) }}
		{{- if .ResultInit }}
			{{- if .ViewedResult }}
			p{{ if .ResultInit.ClientErrorAware }}, err{{ end }} := {{ .ResultInit.Name }}({{ range $i, $ref := responseConstructorArgRefs .ResultInit.ClientArgs "resp.StatusCode" }}{{ if $i }}, {{ end }}{{ $ref }}{{ end }})
			{{- if .ResultInit.ClientErrorAware }}
			if err != nil {
				return nil, loomhttp.ErrValidationError("{{ $.ServiceName }}", "{{ $.Method.Name }}", err)
			}
			{{- end }}
				{{- if .Tag }}
				{{ .Tag.Assignment }}
				{{- end }}
				{{- if $.Method.ViewedResult.ViewName }}
			view := {{ printf "%q" $.Method.ViewedResult.ViewName }}
				{{- else }}
			view := resp.Header.Get("loom-view")
				{{- end }}
			vres := {{ if not $.Method.ViewedResult.IsCollection }}&{{ end }}{{ $.Method.ViewedResult.ViewsPkg}}.{{ $.Method.ViewedResult.VarName }}{Projected: p, View: view}
				{{- if .ClientBody }}
				if err = {{ $.Method.ViewedResult.ViewsPkg}}.{{ $.Method.ViewedResult.Validate.Name }}(vres); err != nil {
					return nil, loomhttp.ErrValidationError("{{ $.ServiceName }}", "{{ $.Method.Name }}", err)
				}
				{{- end }}
			res, err := {{ $.ServicePkgName }}.{{ $.Method.ViewedResult.ResultInit.Name }}(vres)
			if err != nil {
				return nil, loomhttp.ErrValidationError("{{ $.ServiceName }}", "{{ $.Method.Name }}", err)
			}
			{{- else }}
			res{{ if .ResultInit.ClientErrorAware }}, err{{ end }} := {{ .ResultInit.Name }}({{ range $i, $ref := responseConstructorArgRefs .ResultInit.ClientArgs "resp.StatusCode" }}{{ if $i }}, {{ end }}{{ $ref }}{{ end }})
			{{- if .ResultInit.ClientErrorAware }}
			if err != nil {
				return nil, loomhttp.ErrValidationError("{{ $.ServiceName }}", "{{ $.Method.Name }}", err)
			}
			{{- end }}
			{{- end }}
			{{- if and .Tag (not .ViewedResult) }}
			{{ .Tag.Assignment }}
			{{- end }}
			{{- if .HasStatusBinding }}
			{{ renderDecodedStatusValidation . "res" }}
			{{- end }}
			{{ if $.Method.HasResponseMetadata }}return &{{ $.ServicePkgName }}.{{ $.Method.ResponseStruct }}{Result: res, Metadata: metadata}, nil{{ else }}return res, nil{{ end }}
		{{- else if .ClientBody }}
			{{ if $.Method.HasResponseMetadata }}return &{{ $.ServicePkgName }}.{{ $.Method.ResponseStruct }}{Result: body, Metadata: metadata}, nil{{ else }}return body, nil{{ end }}
		{{- else if .Headers }}
			return {{ (index .Headers 0).VarName }}, nil
		{{- else if .Cookies }}
			return {{ (index .Cookies 0).VarName }}, nil
		{{- else }}
			return nil, nil
		{{- end }}
	{{- end }}
	{{- range .Errors }}
		case {{ if $.HasDynamicFinalSelectors }}{{ responseSelectorValue (index .Errors 0).Response }}{{ else }}{{ (index .Errors 0).Response.LiteralStatus }}{{ end }}:
		{{- if gt (len .Errors) 1 }}
		en := resp.Header.Get("loom-error")
		switch en {
			{{- range .Errors }}
		case {{ printf "%q" .Name }}:
				{{- with .Response }}
					{{- template "partial_single_response" (buildResponseData . $.ServiceName $.Method) }}
					{{- if .ResultInit }}
			{{ if or .ResultInit.ClientErrorAware .HasStatusBinding }}result{{ if .ResultInit.ClientErrorAware }}, conversionError{{ end }} := {{ .ResultInit.Name }}({{ range $i, $ref := responseConstructorArgRefs .ResultInit.ClientArgs "resp.StatusCode" }}{{ if $i }}, {{ end }}{{ $ref }}{{ end }})
			{{- if .ResultInit.ClientErrorAware }}
			if conversionError != nil {
				return nil, loomhttp.ErrValidationError("{{ $.ServiceName }}", "{{ $.Method.Name }}", conversionError)
			}
			{{- end }}
			{{- if .HasStatusBinding }}
			{{ renderDecodedStatusValidation . "result" }}
			{{- end }}
			return nil, result{{ else }}return nil, {{ .ResultInit.Name }}({{ range $i, $ref := responseConstructorArgRefs .ResultInit.ClientArgs "resp.StatusCode" }}{{ if $i }}, {{ end }}{{ $ref }}{{ end }}){{ end }}
					{{- else if .ClientBody }}
			return nil, body
					{{- else }}
			return nil, nil
					{{- end }}
				{{- end }}
			{{- end }}
		default:
			body, err := loomhttp.ReadUnexpectedResponseBody(resp)
			if err != nil {
				return nil, err
			}
			return nil, loomhttp.ErrInvalidResponse({{ printf "%q" $.ServiceName }}, {{ printf "%q" $.Method.Name }}, resp.StatusCode, string(body))
		}
		{{- else }}
			{{- with (index .Errors 0).Response }}
				{{- template "partial_single_response" (buildResponseData . $.ServiceName $.Method) }}
				{{- if .ResultInit }}
			{{ if or .ResultInit.ClientErrorAware .HasStatusBinding }}result{{ if .ResultInit.ClientErrorAware }}, conversionError{{ end }} := {{ .ResultInit.Name }}({{ range $i, $ref := responseConstructorArgRefs .ResultInit.ClientArgs "resp.StatusCode" }}{{ if $i }}, {{ end }}{{ $ref }}{{ end }})
			{{- if .ResultInit.ClientErrorAware }}
			if conversionError != nil {
				return nil, loomhttp.ErrValidationError("{{ $.ServiceName }}", "{{ $.Method.Name }}", conversionError)
			}
			{{- end }}
			{{- if .HasStatusBinding }}
			{{ renderDecodedStatusValidation . "result" }}
			{{- end }}
			return nil, result{{ else }}return nil, {{ .ResultInit.Name }}({{ range $i, $ref := responseConstructorArgRefs .ResultInit.ClientArgs "resp.StatusCode" }}{{ if $i }}, {{ end }}{{ $ref }}{{ end }}){{ end }}
				{{- else if .ClientBody }}
			return nil, body
				{{- else }}
			return nil, nil
				{{- end }}
			{{- end }}
		{{- end }}
	{{- end }}
		default:
			body, err := loomhttp.ReadUnexpectedResponseBody(resp)
			if err != nil {
				return nil, err
			}
			return nil, loomhttp.ErrInvalidResponse({{ printf "%q" .ServiceName }}, {{ printf "%q" .Method.Name }}, resp.StatusCode, string(body))
		}
        })
	}
}
`,
		append([]templateSource{
			templateSource{name: "single_response", source: `{{- with .Data }}
	{{- if .ClientBody }}
			var (
				body {{ .ClientBody.ValueRef }}
				err error
			)
            {{- if .ClientBody.Media }}
            {
                mediaIndex, mediaErr := loomhttp.SelectMediaType(resp.Header.Get("Content-Type"), {{ .ClientBody.Media.Contracts }})
                if mediaErr != nil {
                    err = mediaErr
                } else {
                    switch mediaIndex {
                    {{- range $i, $variant := .ClientBody.Media.Variants }}
                    case {{ $i }}:
                        var mediaValue {{ .TypeRef }}
{{- if .Multipart }}
mediaResponse := *resp
mediaRequest := resp.Request
if mediaRequest == nil {
 mediaRequest = new(http.Request)
}
mediaResponse.Request = mediaRequest.WithContext({{ renderMediaMultipartDecode . "mediaRequest.Context()" "mediaValue" }})
err = loomhttp.DecodeMediaResponse(&mediaResponse, &mediaValue, {{ .AllowsNull }}, nil, decoder)
{{- else if mediaFormCodec . }}
mediaResponse := *resp
mediaRequest := resp.Request
if mediaRequest == nil {
 mediaRequest = new(http.Request)
}
mediaResponse.Request = mediaRequest.WithContext({{ renderMediaFormDecode . "mediaRequest.Context()" "mediaValue" }})
err = loomhttp.DecodeMediaResponse(&mediaResponse, &mediaValue, {{ .AllowsNull }}, nil, decoder)
{{- else }}
                        err = loomhttp.DecodeMediaResponse(resp, &mediaValue, {{ .AllowsNull }}, nil, decoder)
{{- end }}
                        if err == nil {
                            body.Set{{ .Method }}(mediaValue)
                        }
                    {{- end }}
                    }
                }
            }
            {{- else }}
			err = decoder({{ if .ClientBody.AllowsNull }}resp{{ else }}loomhttp.WithNonNullableResponseBody(resp){{ end }}).Decode(&body)
            {{- end }}
			if err != nil {
				return nil, loomhttp.ErrDecodingError("{{ $.ServiceName }}", "{{ $.Method.Name }}", err)
			}
		{{- if .ClientBody.ValidateRef }}
			{{ .ClientBody.ValidateRef }}
			if err != nil {
				return nil, loomhttp.ErrValidationError("{{ $.ServiceName }}", "{{ $.Method.Name }}", err)
			}
		{{- end }}
	{{- end }}

{{- if or .Headers .Cookies }}
{{ renderResponseMetadataFields . $.ServiceName $.Method "resp.Header" "resp.Cookies()" }}
{{- end }}

	{{- if .MustValidate }}
			if err != nil {
				return nil, loomhttp.ErrValidationError("{{ $.ServiceName }}", "{{ $.Method.Name }}", err)
			}
	{{- end }}
{{- end }}`},
		}, httpDecoderConversionPartials...)...)
)
