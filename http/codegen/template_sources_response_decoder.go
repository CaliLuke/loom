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
		if restoreBody {
			b, err := loomhttp.ReadResponseBody(resp)
			if err != nil {
				return nil, err
			}
			resp.Body = io.NopCloser(bytes.NewBuffer(b))
			defer func() {
				resp.Body = io.NopCloser(bytes.NewBuffer(b))
			}()
		{{- if not (or .Method.SkipResponseBodyEncodeDecode .Method.FileResponse) }} } else {
			defer resp.Body.Close()
		{{- end }}
		}
		switch resp.StatusCode {
	{{- range .Result.Responses }}
		case {{ .StatusCode }}{{ if and $.Method.FileResponse (eq .Code 200) }}, http.StatusPartialContent, http.StatusNotModified{{ end }}:
			{{- template "partial_single_response" (buildResponseData . $.ServiceName $.Method) }}
		{{- if .ResultInit }}
			{{- if .ViewedResult }}
			p := {{ .ResultInit.Name }}({{ range $i, $arg := .ResultInit.ClientArgs }}{{ if $i }}, {{ end }}{{ $arg.Ref }}{{ end }})
				{{- if .TagName }}
				tmp := {{ printf "%q" .TagValue }}
				p.{{ .TagName }} = &tmp
				{{- end }}
				{{- if $.Method.ViewedResult.ViewName }}
			view := {{ printf "%q" $.Method.ViewedResult.ViewName }}
				{{- else }}
			view := resp.Header.Get("loom-view")
				{{- end }}
			vres := {{ if not $.Method.ViewedResult.IsCollection }}&{{ end }}{{ $.Method.ViewedResult.ViewsPkg}}.{{ $.Method.ViewedResult.VarName }}{Projected: p, View: view}
				{{- if .ClientBody }}
				if err = {{ $.Method.ViewedResult.ViewsPkg}}.Validate{{ $.Method.Result }}(vres); err != nil {
					return nil, loomhttp.ErrValidationError("{{ $.ServiceName }}", "{{ $.Method.Name }}", err)
				}
				{{- end }}
			res, err := {{ $.ServicePkgName }}.{{ $.Method.ViewedResult.ResultInit.Name }}(vres)
			if err != nil {
				return nil, loomhttp.ErrValidationError("{{ $.ServiceName }}", "{{ $.Method.Name }}", err)
			}
			{{- else }}
			res := {{ .ResultInit.Name }}({{ range $i, $arg := .ResultInit.ClientArgs }}{{ if $i }}, {{ end }}{{ $arg.Ref }}{{ end }})
			{{- end }}
			{{- if and .TagName (not .ViewedResult) }}
				{{- if .TagPointer }}
					tmp := {{ printf "%q" .TagValue }}
					res.{{ .TagName }} = &tmp
				{{- else }}
					res.{{ .TagName }} = {{ printf "%q" .TagValue }}
				{{- end }}
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
			return nil, {{ .ResultInit.Name }}({{ range $i, $arg := .ResultInit.ClientArgs }}{{ if $i }}, {{ end }}{{ $arg.Ref }}{{ end }})
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
			return nil, {{ .ResultInit.Name }}({{ range $i, $arg := .ResultInit.ClientArgs }}{{ if $i }}, {{ end }}{{ $arg.Ref }}{{ end }})
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
	}
}
`,
		append([]templateSource{
			templateSource{name: "single_response", source: `{{- with .Data }}
	{{- if .ClientBody }}
			var (
				body {{ .ClientBody.VarName }}
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

		{{- if (or (eq .Type.Name "string") (eq .Type.Name "any")) }}
			{{ .VarName }}Raw := resp.Header.Get("{{ .CanonicalName }}")
			{{- if .Required }}
				if {{ .VarName }}Raw == "" {
					err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "header"))
				}
				{{ .VarName }} = {{ if and (eq .Type.Name "string") .Pointer }}&{{ end }}{{ .VarName }}Raw
			{{- else }}
				if {{ .VarName }}Raw != "" {
					{{ .VarName }} = {{ if and (eq .Type.Name "string") .Pointer }}&{{ end }}{{ .VarName }}Raw
				}
				{{- if .DefaultValue }} else {
					{{ .VarName }} = {{ if eq .Type.Name "string" }}{{ printf "%q" .DefaultValue }}{{ else }}{{ printf "%#v" .DefaultValue }}{{ end }}
				}
				{{- end }}
			{{- end }}

		{{- else if .StringSlice }}
			{{ .VarName }} = resp.Header["{{ .CanonicalName }}"]
			{{ if .Required }}
			if {{ .VarName }} == nil {
				err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "header"))
			}
			{{- else if .DefaultValue }}
			if {{ .VarName }} == nil {
				{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
			}
			{{- end }}

		{{- else if .Slice }}
		{
			{{ .VarName }}Raw := resp.Header["{{ .CanonicalName }}"]
				{{ if .Required }} if {{ .VarName }}Raw == nil {
				return nil, loomhttp.ErrValidationError("{{ $.ServiceName }}", "{{ $.Method.Name }}", loom.MissingFieldError("{{ .Name }}", "header"))
			}
			{{- else if .DefaultValue }}
			if {{ .VarName }}Raw == nil {
				{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
			}
			{{- end }}

			{{- if .DefaultValue }}else {
			{{- else if not .Required }}
			if {{ .VarName }}Raw != nil {
			{{- end }}
				{{- template "partial_element_slice_conversion" . }}
			{{- if or .DefaultValue (not .Required) }}
			}
			{{- end }}
		}

		{{- else }}{{/* not string, not any and not slice */}}
		{
			{{ .VarName }}Raw := resp.Header.Get("{{ .CanonicalName }}")
			{{- if .Required }}
			if {{ .VarName }}Raw == "" {
				return nil, loomhttp.ErrValidationError("{{ $.ServiceName }}", "{{ $.Method.Name }}", loom.MissingFieldError("{{ .Name }}", "header"))
			}
			{{- else if .DefaultValue }}
			if {{ .VarName }}Raw == "" {
				{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
			}
			{{- end }}

			{{- if .DefaultValue }}else {
				{{- else if not .Required }}
			if {{ .VarName }}Raw != "" {
			{{- end }}
				{{- template "partial_query_type_conversion" . }}
			{{- if or .DefaultValue (not .Required) }}
			}
			{{- end }}
		}
		{{- end }}
		{{- if .Validate }}
			{{ .Validate }}
		{{- end }}
		{{- end }}{{/* range .Headers */}}
	{{- end }}

	{{- if .Cookies }}
			var (
		{{- range .Cookies }}
				{{ .VarName }}    {{ .TypeRef }}
				{{ .VarName }}Raw string
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
				{{ .VarName }}Raw = c.Value
		{{- end }}
			}
		}
		{{- range .Cookies }}

		{{- if (or (eq .Type.Name "string") (eq .Type.Name "any")) }}
			{{- if .Required }}
				if {{ .VarName }}Raw == "" {
					err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "cookie"))
				}
				{{ .VarName }} = {{ if and (eq .Type.Name "string") .Pointer }}&{{ end }}{{ .VarName }}Raw
			{{- else }}
				if {{ .VarName }}Raw != "" {
					{{ .VarName }} = {{ if and (eq .Type.Name "string") .Pointer }}&{{ end }}{{ .VarName }}Raw
				}
				{{- if .DefaultValue }} else {
					{{ .VarName }} = {{ if eq .Type.Name "string" }}{{ printf "%q" .DefaultValue }}{{ else }}{{ printf "%#v" .DefaultValue }}{{ end }}
				}
				{{- end }}
			{{- end }}

		{{- else }}{{/* not string and not any */}}
		{
			{{- if .Required }}
			if {{ .VarName }}Raw == "" {
				return nil, loomhttp.ErrValidationError("{{ $.ServiceName }}", "{{ $.Method.Name }}", loom.MissingFieldError("{{ .Name }}", "cookie"))
			}
			{{- else if .DefaultValue }}
			if {{ .VarName }}Raw == "" {
				{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
			}
			{{- end }}

			{{- if .DefaultValue }}else {
				{{- else if not .Required }}
			if {{ .VarName }}Raw != "" {
			{{- end }}
				{{- template "partial_query_type_conversion" . }}
			{{- if or .DefaultValue (not .Required) }}
			}
			{{- end }}
		}
		{{- end }}
		{{- if .Validate }}
			{{ .Validate }}
		{{- end }}
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
