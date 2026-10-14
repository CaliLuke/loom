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
//	- {{ printf "%q" .Name }} (type {{ .Ref }}): {{ .Response.StatusCode }}{{ if .Response.Description }}, {{ .Response.Description }}{{ end }}
	{{- end }}
	{{- end }}
//	- error: internal error
{{- end }}
func {{ .ResponseDecoder }}(decoder func(*http.Response) loomhttp.Decoder, restoreBody bool) func(*http.Response) (any, error) {
	return func(resp *http.Response) (any, error) {
        return loomhttp.DecodeResponse(resp, restoreBody, {{ if or .Method.SkipResponseBodyEncodeDecode .Method.FileResponse }}false{{ range .Result.Responses }} || resp.StatusCode == {{ .StatusCode }}{{ if and $.Method.FileResponse (eq .Code 200) }} || resp.StatusCode == http.StatusPartialContent || resp.StatusCode == http.StatusNotModified{{ end }}{{ end }}{{ else }}false{{ end }}, func(resp *http.Response) (any, error) {
		switch resp.StatusCode {
	{{- range .Result.Responses }}
		case {{ .StatusCode }}{{ if and $.Method.FileResponse (eq .Code 200) }}, http.StatusPartialContent, http.StatusNotModified{{ end }}:
			{{- template "partial_single_response" (buildResponseData . $.ServiceName $.Method) }}
		{{- if .ResultInit }}
			{{- if .ViewedResult }}
			p{{ if .ResultInit.ClientErrorAware }}, err{{ end }} := {{ .ResultInit.Name }}({{ range $i, $arg := .ResultInit.ClientArgs }}{{ if $i }}, {{ end }}{{ $arg.Ref }}{{ end }})
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
			res{{ if .ResultInit.ClientErrorAware }}, err{{ end }} := {{ .ResultInit.Name }}({{ range $i, $arg := .ResultInit.ClientArgs }}{{ if $i }}, {{ end }}{{ $arg.Ref }}{{ end }})
			{{- if .ResultInit.ClientErrorAware }}
			if err != nil {
				return nil, loomhttp.ErrValidationError("{{ $.ServiceName }}", "{{ $.Method.Name }}", err)
			}
			{{- end }}
			{{- end }}
			{{- if and .Tag (not .ViewedResult) }}
			{{ .Tag.Assignment }}
			{{- end }}
			return res, nil
		{{- else if .ClientBody }}
			return body, nil
		{{- else if .Headers }}
			return {{ (index .Headers 0).VarName }}, nil
		{{- else if .Cookies }}
			return {{ (index .Cookies 0).VarName }}, nil
		{{- else }}
			return nil, nil
		{{- end }}
	{{- end }}
	{{- range .Errors }}
		case {{ .StatusCode }}:
		{{- if gt (len .Errors) 1 }}
		en := resp.Header.Get("loom-error")
		switch en {
			{{- range .Errors }}
		case {{ printf "%q" .Name }}:
				{{- with .Response }}
					{{- template "partial_single_response" (buildResponseData . $.ServiceName $.Method) }}
					{{- if .ResultInit }}
			{{ if .ResultInit.ClientErrorAware }}result, conversionError := {{ .ResultInit.Name }}({{ range $i, $arg := .ResultInit.ClientArgs }}{{ if $i }}, {{ end }}{{ $arg.Ref }}{{ end }})
			if conversionError != nil {
				return nil, loomhttp.ErrValidationError("{{ $.ServiceName }}", "{{ $.Method.Name }}", conversionError)
			}
			return nil, result{{ else }}return nil, {{ .ResultInit.Name }}({{ range $i, $arg := .ResultInit.ClientArgs }}{{ if $i }}, {{ end }}{{ $arg.Ref }}{{ end }}){{ end }}
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
			{{ if .ResultInit.ClientErrorAware }}result, conversionError := {{ .ResultInit.Name }}({{ range $i, $arg := .ResultInit.ClientArgs }}{{ if $i }}, {{ end }}{{ $arg.Ref }}{{ end }})
			if conversionError != nil {
				return nil, loomhttp.ErrValidationError("{{ $.ServiceName }}", "{{ $.Method.Name }}", conversionError)
			}
			return nil, result{{ else }}return nil, {{ .ResultInit.Name }}({{ range $i, $arg := .ResultInit.ClientArgs }}{{ if $i }}, {{ end }}{{ $arg.Ref }}{{ end }}){{ end }}
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
			err = decoder(resp).Decode(&body)
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

	{{- if .Headers }}
			var (
		{{- range .Headers }}
				{{ .VarName }} {{ .TypeRef }}
		{{- end }}
		{{- if not .ClientBody }}
			{{- if .MustValidate }}
				err error
			{{- end }}
		{{- end }}
			)
		{{- range .Headers }}

 {{- $scalar := or .IsTextUnmarshaler (not .Slice) }}
 {{- if $scalar }}
 {
  {{ .Locals.Raw }} := resp.Header.Get("{{ .CanonicalName }}")
  {{ .Locals.Present }} := len(resp.Header.Values("{{ .CanonicalName }}")) > 0
  {{- template "partial_scalar_element" (scalarElementData .Element "header") }}
 }
		{{- else if .StringSlice }}
			{{ .VarName }} = loomhttp.HeaderValues(resp.Header, "{{ .CanonicalName }}")
			{{ if .Required }}
			if {{ .VarName }} == nil {
				err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "header"))
			}
			{{- else if (ne .DefaultValue nil) }}
			if {{ .VarName }} == nil {
				{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
			}
			{{- end }}

		{{- else if .Slice }}
		{
			{{ .Locals.Raw }} := loomhttp.HeaderValues(resp.Header, "{{ .CanonicalName }}")
				{{ if .Required }} if {{ .Locals.Raw }} == nil {
				return nil, loomhttp.ErrValidationError("{{ $.ServiceName }}", "{{ $.Method.Name }}", loom.MissingFieldError("{{ .Name }}", "header"))
			}
			{{- else if (ne .DefaultValue nil) }}
			if {{ .Locals.Raw }} == nil {
				{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
			}
			{{- end }}

			{{- if (ne .DefaultValue nil) }}else {
			{{- else if not .Required }}
			if {{ .Locals.Raw }} != nil {
			{{- end }}
				{{- template "partial_element_slice_conversion" . }}
			{{- if or (ne .DefaultValue nil) (not .Required) }}
			}
			{{- end }}
		}


		{{- end }}
		{{- if and .Validate (not $scalar) }}
			{{ .Validate }}
		{{- end }}
		{{- end }}{{/* range .Headers */}}
	{{- end }}

	{{- if .Cookies }}
			var (
		{{- range .Cookies }}
				{{ .VarName }}    {{ .TypeRef }}
				{{ .Locals.Raw }} string
            {{ .Locals.Present }} bool
		{{- end }}

				cookies = resp.Cookies()
		{{- if not .ClientBody }}
			{{- if .MustValidate }}
				{{- if not .Headers}}
					err error
				{{- end }}
			{{- end }}
		{{- end }}
			)
        for _, c := range cookies {
			switch c.Name {
		{{- range .Cookies }}
			case {{ printf "%q" .HTTPName }}:
				{{ .Locals.Raw }} = c.Value
                {{ .Locals.Present }} = true
		{{- end }}
			}
		}
		{{- range .Cookies }}

  {{- template "partial_scalar_element" (scalarElementData .Element "cookie") }}
		{{- end }}{{/* range .Cookies */}}
	{{- end }}

	{{- if .MustValidate }}
			if err != nil {
				return nil, loomhttp.ErrValidationError("{{ $.ServiceName }}", "{{ $.Method.Name }}", err)
			}
	{{- end }}
{{- end }}`},
		}, httpDecoderConversionPartials...)...)
)
