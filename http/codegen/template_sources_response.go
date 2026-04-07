package codegen

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
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body = io.NopCloser(bytes.NewBuffer(b))
			defer func() {
				resp.Body = io.NopCloser(bytes.NewBuffer(b))
			}()
		{{- if not .Method.SkipResponseBodyEncodeDecode }} } else {
			defer resp.Body.Close()
		{{- end }}
		}
		switch resp.StatusCode {
	{{- range .Result.Responses }}
		case {{ .StatusCode }}:
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
			res := {{ $.ServicePkgName }}.{{ $.Method.ViewedResult.ResultInit.Name }}(vres)
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
			body, _ := io.ReadAll(resp.Body)
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
			body, _ := io.ReadAll(resp.Body)
			return nil, loomhttp.ErrInvalidResponse({{ printf "%q" .ServiceName }}, {{ printf "%q" .Method.Name }}, resp.StatusCode, string(body))
		}
	}
}
`,
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
		templateSource{name: "query_type_conversion", source: `	{{- if eq .Type.Name "bytes" }}
		{{ .VarName }} = []byte({{.VarName}}Raw)
	{{- else if eq .Type.Name "int" }}
		v, err2 := strconv.ParseInt({{ .VarName }}Raw, 10, strconv.IntSize)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "integer"))
		}
		{{- if .Pointer }}
		pv := {{ if .TypeRef }}{{slice .TypeRef 1 (len .TypeRef)}}{{ else }}int{{ end }}(v)
		{{ .VarName }} = &pv
		{{- else }}
		{{ .VarName }} = {{ if .TypeRef }}{{ .TypeRef }}{{ else }}int{{ end }}(v)
		{{- end }}
	{{- else if eq .Type.Name "int32" }}
		v, err2 := strconv.ParseInt({{ .VarName }}Raw, 10, 32)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "integer"))
		}
		{{- if .Pointer }}
		pv := {{ if .TypeRef }}{{ slice .TypeRef 1 (len .TypeRef) }}{{ else }}int32{{ end }}(v)
		{{ .VarName }} = &pv
		{{- else }}
		{{ .VarName }} = {{ if .TypeRef }}{{ .TypeRef }}{{ else }}int32{{ end }}(v)
		{{- end }}
	{{- else if eq .Type.Name "int64" }}
		v, err2 := strconv.ParseInt({{ .VarName }}Raw, 10, 64)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "integer"))
		}
		{{ if and (ne .TypeRef nil) (and (ne .TypeRef "int64") (ne .TypeRef "*int64")) }}{{ .VarName }} = ({{.TypeRef}})({{ if .Pointer }}&{{ end }}v){{ else }}{{ .VarName }} = {{ if .Pointer }}&{{ end }}v{{ end }}
	{{- else if eq .Type.Name "uint" }}
		v, err2 := strconv.ParseUint({{ .VarName }}Raw, 10, strconv.IntSize)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "unsigned integer"))
		}
		{{- if .Pointer }}
		pv := {{ if .TypeRef }}{{ slice .TypeRef 1 (len .TypeRef) }}{{ else }}uint{{ end }}(v)
		{{ .VarName }} = &pv
		{{- else }}
		{{ .VarName }} = {{ if .TypeRef }}{{ .TypeRef }}{{ else }}uint{{ end }}(v)
		{{- end }}
	{{- else if eq .Type.Name "uint32" }}
		v, err2 := strconv.ParseUint({{ .VarName }}Raw, 10, 32)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "unsigned integer"))
		}
		{{- if .Pointer }}
		pv := {{ if .TypeRef }}{{ slice .TypeRef 1 (len .TypeRef) }}{{ else }}uint32{{ end }}(v)
		{{ .VarName }} = &pv
		{{- else }}
		{{ .VarName }} = {{ if .TypeRef }}{{ .TypeRef }}{{ else }}uint32{{ end }}(v)
		{{- end }}
	{{- else if eq .Type.Name "uint64" }}
		v, err2 := strconv.ParseUint({{ .VarName }}Raw, 10, 64)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "unsigned integer"))
		}
		{{ if and (ne .TypeRef nil) (and (ne .TypeRef "uint64") (ne .TypeRef "*uint64")) }}{{ .VarName }} = ({{.TypeRef}})({{ if .Pointer }}&{{ end }}v){{ else }}{{ .VarName }} = {{ if .Pointer }}&{{ end }}v{{ end }}
	{{- else if eq .Type.Name "float32" }}
		v, err2 := strconv.ParseFloat({{ .VarName }}Raw, 32)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "float"))
		}
		{{- if .Pointer }}
		pv := {{ if .TypeRef }}{{ slice .TypeRef 1 (len .TypeRef) }}{{ else }}float32{{ end }}(v)
		{{ .VarName }} = &pv
		{{- else }}
		{{ .VarName }} = {{ if .TypeRef }}{{ .TypeRef }}{{ else }}float32{{ end }}(v)
		{{- end }}
	{{- else if eq .Type.Name "float64" }}
		v, err2 := strconv.ParseFloat({{ .VarName }}Raw, 64)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "float"))
		}
		{{ if and (ne .TypeRef nil) (and (ne .TypeRef "float64") (ne .TypeRef "*float64")) }}{{ .VarName }} = ({{.TypeRef}})({{ if .Pointer }}&{{ end }}v){{ else }}{{ .VarName }} = {{ if .Pointer }}&{{ end }}v{{ end }}
	{{- else if eq .Type.Name "boolean" }}
		v, err2 := strconv.ParseBool({{ .VarName }}Raw)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "boolean"))
		}
		{{ if and (ne .TypeRef nil) (and (ne .TypeRef "bool") (ne .TypeRef "*bool")) }}{{ .VarName }} = ({{.TypeRef}})({{ if .Pointer }}&{{ end }}v){{ else }}{{ .VarName }} = {{ if .Pointer }}&{{ end }}v{{ end }}
	{{- else }}
		// unsupported type {{ .Type.Name }} for var {{ .VarName }}
	{{- end }}`},
		templateSource{name: "element_slice_conversion", source: `	{{ .VarName }} = make({{ goTypeRef .Type }}, len({{ .VarName }}Raw))
	for i, rv := range {{ .VarName }}Raw {
		{{- template "partial_slice_item_conversion" . }}
	}`},
		templateSource{name: "slice_item_conversion", source: `		{{- if eq .Type.ElemType.Type.Name "string" }}
			{{ .VarName }}[i] = rv
		{{- else if eq .Type.ElemType.Type.Name "bytes" }}
			{{ .VarName }}[i] = []byte(rv)
		{{- else if eq .Type.ElemType.Type.Name "int" }}
			v, err2 := strconv.ParseInt(rv, 10, strconv.IntSize)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of integers"))
			}
			{{ .VarName }}[i] = int(v)
		{{- else if eq .Type.ElemType.Type.Name "int32" }}
			v, err2 := strconv.ParseInt(rv, 10, 32)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of integers"))
			}
			{{ .VarName }}[i] = int32(v)
		{{- else if eq .Type.ElemType.Type.Name "int64" }}
			v, err2 := strconv.ParseInt(rv, 10, 64)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of integers"))
			}
			{{ .VarName }}[i] = v
		{{- else if eq .Type.ElemType.Type.Name "uint" }}
			v, err2 := strconv.ParseUint(rv, 10, strconv.IntSize)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of unsigned integers"))
			}
			{{ .VarName }}[i] = uint(v)
		{{- else if eq .Type.ElemType.Type.Name "uint32" }}
			v, err2 := strconv.ParseUint(rv, 10, 32)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of unsigned integers"))
			}
			{{ .VarName }}[i] = uint32(v)
		{{- else if eq .Type.ElemType.Type.Name "uint64" }}
			v, err2 := strconv.ParseUint(rv, 10, 64)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of unsigned integers"))
			}
			{{ .VarName }}[i] = v
		{{- else if eq .Type.ElemType.Type.Name "float32" }}
			v, err2 := strconv.ParseFloat(rv, 32)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of floats"))
			}
			{{ .VarName }}[i] = float32(v)
		{{- else if eq .Type.ElemType.Type.Name "float64" }}
			v, err2 := strconv.ParseFloat(rv, 64)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of floats"))
			}
			{{ .VarName }}[i] = v
		{{- else if eq .Type.ElemType.Type.Name "boolean" }}
			v, err2 := strconv.ParseBool(rv)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of booleans"))
			}
			{{ .VarName }}[i] = v
		{{- else if eq .Type.ElemType.Type.Name "any" }}
			{{ .VarName }}[i] = rv
		{{- else }}
			// unsupported slice type {{ .Type.ElemType.Type.Name }} for var {{ .VarName }}
		{{- end }}`},
	)

	responseEncoderSource = joinHTTPTemplateSource(`{{ printf "%s returns an encoder for responses returned by the %s %s endpoint." .ResponseEncoder .ServiceName .Method.Name | comment }}
func {{ .ResponseEncoder }}(encoder func(context.Context, http.ResponseWriter) loomhttp.Encoder) func(context.Context, http.ResponseWriter, any) error {
	return func(ctx context.Context, w http.ResponseWriter, v any) error {
	{{- if .Result.MustInit }}
		{{- if .Method.ViewedResult }}
			res := v.({{ .Method.ViewedResult.FullRef }})
			{{- if not .Method.ViewedResult.ViewName }}
				w.Header().Set("loom-view", res.View)
			{{- end }}
		{{- else }}
			res, _ := v.({{ .Result.Ref }})
		{{- end }}
		{{- range .Result.Responses }}
			{{- if .ContentType }}
				ctx = context.WithValue(ctx, loomhttp.ContentTypeKey, "{{ .ContentType }}")
			{{- end }}
			{{- if .TagName }}
				{{- if .TagPointer }}
					if res.{{ if .ViewedResult }}Projected.{{ end }}{{ .TagName }} != nil && *res.{{ if .ViewedResult }}Projected.{{ end }}{{ .TagName }} == {{ printf "%q" .TagValue }} {
				{{- else }}
					if {{ if .ViewedResult }}*{{ end }}res.{{ if .ViewedResult }}Projected.{{ end }}{{ .TagName }} == {{ printf "%q" .TagValue }} {
				{{- end }}
			{{- end -}}
			{{ template "partial_response" . }}
			{{- if .ServerBody }}
				return enc.Encode(body)
			{{- else }}
				return nil
			{{- end }}
			{{- if .TagName }}
				}
			{{- end }}
		{{- end }}
	{{- else }}
		{{- with (index .Result.Responses 0) }}
			{{- if not $.Method.SkipResponseBodyEncodeDecode }}
				w.WriteHeader({{ .StatusCode }})
			{{- end }}
			return nil
		{{- end }}
	{{- end }}
	}
}
`,
		templateSource{name: "response", source: `	{{- $servBodyLen := len .ServerBody }}
	{{- if gt $servBodyLen 0 }}
	enc := encoder(ctx, w)
		{{- if and (gt $servBodyLen 1) $.ViewedResult }}
	var body any
	switch res.View	{
			{{- range $.ViewedResult.Views }}
	case {{ printf "%q" .Name }}{{ if eq .Name "default" }}, ""{{ end }}:
		{{- $vsb := (viewedServerBody $.ServerBody .Name) }}
		body = {{ $vsb.Init.Name }}({{ range $vsb.Init.ServerArgs }}{{ .Ref }}, {{ end }})
			{{- end }}
	}
		{{- else if (index .ServerBody 0).Init }}
			{{- if .ErrorHeader }}
	var body any
	if formatter != nil {
		body = formatter(ctx, {{ (index (index .ServerBody 0).Init.ServerArgs 0).Ref }})
	} else {
			{{- end }}
	body {{ if not .ErrorHeader}}:{{ end }}= {{ (index .ServerBody 0).Init.Name }}({{ range (index .ServerBody 0).Init.ServerArgs }}{{ .Ref }}, {{ end }})
			{{- if .ErrorHeader }}
	}
			{{- end }}
		{{- else }}
	body := res{{ if $.ViewedResult }}.Projected{{ end }}{{ if .ResultAttr }}.{{ .ResultAttr }}{{ end }}
		{{- end }}
	{{- end }}
	{{- if and (eq .HeaderSourceVar "problem") (or .Headers .Cookies) }}
	problem := loomhttp.NewProblemResponse(ctx, res, {{ .Code }}, {{ .ProblemTypeOverride }}, {{ .ProblemTitleOverride }})
	{{- end }}
	{{- range .Headers }}
		{{- $initDef := and (or .FieldPointer .Slice) .DefaultValue (not $.TagName) }}
		{{- $checkNil := and (or .FieldPointer .Slice (eq .Type.Name "bytes") (eq .Type.Name "any") $initDef) (not $.TagName) }}
		{{- if $checkNil }}
	if {{ if eq $.HeaderSourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}{{ if .FieldName }}.{{ .FieldName }}{{ end }} != nil {
		{{- end }}

		{{- if and (eq .Type.Name "string") (not (isAliased .FieldType)) }}
	w.Header().Set("{{ .CanonicalName }}", {{ if or .FieldPointer $.ViewedResult }}*{{ end }}{{ if eq $.HeaderSourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}{{ if .FieldName }}.{{ .FieldName }}{{ end }})
		{{- else }}
{{- if not $checkNil }}
{
{{- end }}
			{{- if isAliased .FieldType }}
	val := {{ goTypeRef .Type }}({{ if .FieldPointer }}*{{ end }}{{ if eq $.HeaderSourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}{{ if .FieldName }}.{{ .FieldName }}{{ end }})
	{{ template "partial_header_conversion" (headerConversionData .Type (printf "%ss" .VarName) true "val") }}
			{{- else }}
	val := {{ if eq $.HeaderSourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}{{ if .FieldName }}.{{ .FieldName }}{{ end }}
	{{ template "partial_header_conversion" (headerConversionData .Type (printf "%ss" .VarName) (not .FieldPointer) "val") }}
			{{- end }}
	w.Header().Set("{{ .CanonicalName }}", {{ .VarName }}s)
{{- if not $checkNil }}
}
{{- end }}
		{{- end }}

		{{- if $initDef }}
	{{ if $checkNil }} } else { {{ else }}if {{ if eq $.HeaderSourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}.{{ .FieldName }} == nil { {{ end }}
		w.Header().Set("{{ .CanonicalName }}", "{{ printValue .Type .DefaultValue }}")
		{{- end }}

		{{- if or $checkNil $initDef }}
	}
		{{- end }}

	{{- end }}

	{{- range .Cookies }}
		{{- $initDef := and (or .FieldPointer .Slice) .DefaultValue }}
		{{- $checkNil := and (or .FieldPointer .Slice (eq .Type.Name "bytes") (eq .Type.Name "any") $initDef) }}
		{{- if $checkNil }}
	if {{ if eq $.HeaderSourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}.{{ .FieldName }} != nil {
		{{- end }}

		{{- if eq .Type.Name "string" }}
	{{ .VarName }} := {{ if or .FieldPointer $.ViewedResult }}*{{ end }}{{ if eq $.HeaderSourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}{{ if .FieldName }}.{{ .FieldName }}{{ end }}
		{{- else }}
			{{- if isAliased .FieldType }}
	{{ .VarName }}raw := {{ goTypeRef .Type }}({{ if .FieldPointer }}*{{ end }}{{ if eq $.HeaderSourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}{{ if .FieldName }}.{{ .FieldName }}{{ end }})
	{{ template "partial_header_conversion" (headerConversionData .Type (printf "%sraw" .VarName) true .VarName) }}
			{{- else }}
	{{ .VarName }}raw := {{ if eq $.HeaderSourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}{{ if .FieldName }}.{{ .FieldName }}{{ end }}
	{{ template "partial_header_conversion" (headerConversionData .Type (printf "%sraw" .VarName) (not .FieldPointer) .VarName) }}
			{{- end }}
		{{- end }}

		{{- if $initDef }}
	{{ if $checkNil }} } else { {{ else }}if {{ if eq $.HeaderSourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}.{{ .FieldName }} == nil { {{ end }}
		{{ .VarName }} := "{{ printValue .Type .DefaultValue }}"
		{{- end }}
		http.SetCookie(w, &http.Cookie{
			Name: {{ printf "%q" .HTTPName }},
			Value: {{ .VarName }},
			{{- if .MaxAge }}
			MaxAge: {{ .MaxAge }},
			{{- end }}
			{{- if .Path }}
			Path: {{ printf "%q" .Path }},
			{{- end }}
			{{- if .Domain }}
			Domain: {{ printf "%q" .Domain }},
			{{- end }}
			{{- if .Secure }}
			Secure: true,
			{{- end }}
			{{- if .HTTPOnly }}
			HttpOnly: true,
			{{- end }}
			{{- if .SameSite }}
			SameSite: {{ .SameSite }},
			{{- end }}
		})
		{{- if or $checkNil $initDef }}
	}
		{{- end }}

	{{- end }}

	{{- if .ErrorHeader }}
	w.Header().Set("loom-error", res.LoomErrorName())
	{{- end }}
	w.WriteHeader({{ .StatusCode }})`},
		templateSource{name: "header_conversion", source: `	{{- if eq .Type.Name "boolean" -}}
		{{ .VarName }} := strconv.FormatBool({{ if not .Required }}*{{ end }}{{ .Target }})
	{{- else if eq .Type.Name "int" -}}
		{{ .VarName }} := strconv.Itoa({{ if not .Required }}*{{ end }}{{ .Target }})
	{{- else if eq .Type.Name "int32" -}}
		{{ .VarName }} := strconv.FormatInt(int64({{ if not .Required }}*{{ end }}{{ .Target }}), 10)
	{{- else if eq .Type.Name "int64" -}}
		{{ .VarName }} := strconv.FormatInt({{ if not .Required }}*{{ end }}{{ .Target }}, 10)
	{{- else if eq .Type.Name "uint" -}}
		{{ .VarName }} := strconv.FormatUint(uint64({{ if not .Required }}*{{ end }}{{ .Target }}), 10)
	{{- else if eq .Type.Name "uint32" -}}
		{{ .VarName }} := strconv.FormatUint(uint64({{ if not .Required }}*{{ end }}{{ .Target }}), 10)
	{{- else if eq .Type.Name "uint64" -}}
		{{ .VarName }} := strconv.FormatUint({{ if not .Required }}*{{ end }}{{ .Target }}, 10)
	{{- else if eq .Type.Name "float32" -}}
		{{ .VarName }} := strconv.FormatFloat(float64({{ if not .Required }}*{{ end }}{{ .Target }}), 'f', -1, 32)
	{{- else if eq .Type.Name "float64" -}}
		{{ .VarName }} := strconv.FormatFloat({{ if not .Required }}*{{ end }}{{ .Target }}, 'f', -1, 64)
	{{- else if eq .Type.Name "string" -}}
		{{ .VarName }} := {{ .Target }}
	{{- else if eq .Type.Name "bytes" -}}
		{{ .VarName }} := string({{ .Target }})
	{{- else if eq .Type.Name "any" -}}
		{{ .VarName }} := fmt.Sprintf("%v", {{ .Target }})
	{{- else if eq .Type.Name "array" -}}
		{{- if eq .Type.ElemType.Type.Name "string" -}}
		{{ .VarName }} := strings.Join({{ .Target }}, ", ")
		{{- else -}}
		{{ .VarName }}Slice := make([]string, len({{ .Target }}))
		for i, e := range {{ .Target }}  {
			{{ template "partial_header_conversion" (headerConversionData .Type.ElemType.Type "es" true "e") }}
			{{ .VarName }}Slice[i] = es
		}
		{{ .VarName }} := strings.Join({{ .VarName }}Slice, ", ")
		{{- end }}
	{{- else }}
		// unsupported type {{ .Type.Name }} for header field {{ .FieldName }}
	{{- end }}`},
	)

	errorEncoderSource = joinHTTPTemplateSource(`{{ printf "%s returns an encoder for errors returned by the %s %s endpoint." .ErrorEncoder .Method.Name .ServiceName | comment }}
func {{ .ErrorEncoder }}(encoder func(context.Context, http.ResponseWriter) loomhttp.Encoder, formatter func(ctx context.Context, err error) loomhttp.Statuser) func(context.Context, http.ResponseWriter, error) error {
	encodeError := loomhttp.ErrorEncoder(encoder, formatter)
	return func(ctx context.Context, w http.ResponseWriter, v error) error {
		var en loom.LoomErrorNamer
		if !errors.As(v, &en) {
			return encodeError(ctx, w, v)
		}
		switch en.LoomErrorName() {
	{{- range $gerr := .Errors }}
	{{- range $err := .Errors }}
		case {{ printf "%q" .Name }}:
			var res {{ $err.Ref }}
			errors.As(v, &res)
			{{- with .Response}}
				{{- if .ContentType }}
					ctx = context.WithValue(ctx, loomhttp.ContentTypeKey, "{{ .ContentType }}")
				{{- end }}
				{{- template "partial_response" . }}
				{{- if .ServerBody }}
				return enc.Encode(body)
				{{- else }}
				return nil
				{{- end }}
			{{- end }}
	{{- end }}
	{{- end }}
		default:
			return encodeError(ctx, w, v)
		}
	}
}
`,
		templateSource{name: "response", source: `	{{- $servBodyLen := len .ServerBody }}
	{{- if gt $servBodyLen 0 }}
	enc := encoder(ctx, w)
		{{- if and (gt $servBodyLen 1) $.ViewedResult }}
	var body any
	switch res.View	{
			{{- range $.ViewedResult.Views }}
	case {{ printf "%q" .Name }}{{ if eq .Name "default" }}, ""{{ end }}:
		{{- $vsb := (viewedServerBody $.ServerBody .Name) }}
		body = {{ $vsb.Init.Name }}({{ range $vsb.Init.ServerArgs }}{{ .Ref }}, {{ end }})
			{{- end }}
	}
		{{- else if (index .ServerBody 0).Init }}
			{{- if .ErrorHeader }}
	var body any
	if formatter != nil {
		body = formatter(ctx, {{ (index (index .ServerBody 0).Init.ServerArgs 0).Ref }})
	} else {
			{{- end }}
	body {{ if not .ErrorHeader}}:{{ end }}= {{ (index .ServerBody 0).Init.Name }}({{ range (index .ServerBody 0).Init.ServerArgs }}{{ .Ref }}, {{ end }})
			{{- if .ErrorHeader }}
	}
			{{- end }}
		{{- else }}
	body := res{{ if $.ViewedResult }}.Projected{{ end }}{{ if .ResultAttr }}.{{ .ResultAttr }}{{ end }}
		{{- end }}
	{{- end }}
	{{- if and (eq .HeaderSourceVar "problem") (or .Headers .Cookies) }}
	problem := loomhttp.NewProblemResponse(ctx, res, {{ .Code }}, {{ .ProblemTypeOverride }}, {{ .ProblemTitleOverride }})
	{{- end }}
	{{- range .Headers }}
		{{- $initDef := and (or .FieldPointer .Slice) .DefaultValue (not $.TagName) }}
		{{- $checkNil := and (or .FieldPointer .Slice (eq .Type.Name "bytes") (eq .Type.Name "any") $initDef) (not $.TagName) }}
		{{- if $checkNil }}
	if {{ if eq $.HeaderSourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}{{ if .FieldName }}.{{ .FieldName }}{{ end }} != nil {
		{{- end }}

		{{- if and (eq .Type.Name "string") (not (isAliased .FieldType)) }}
	w.Header().Set("{{ .CanonicalName }}", {{ if or .FieldPointer $.ViewedResult }}*{{ end }}{{ if eq $.HeaderSourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}{{ if .FieldName }}.{{ .FieldName }}{{ end }})
		{{- else }}
{{- if not $checkNil }}
{
{{- end }}
			{{- if isAliased .FieldType }}
	val := {{ goTypeRef .Type }}({{ if .FieldPointer }}*{{ end }}{{ if eq $.HeaderSourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}{{ if .FieldName }}.{{ .FieldName }}{{ end }})
	{{ template "partial_header_conversion" (headerConversionData .Type (printf "%ss" .VarName) true "val") }}
			{{- else }}
	val := {{ if eq $.HeaderSourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}{{ if .FieldName }}.{{ .FieldName }}{{ end }}
	{{ template "partial_header_conversion" (headerConversionData .Type (printf "%ss" .VarName) (not .FieldPointer) "val") }}
			{{- end }}
	w.Header().Set("{{ .CanonicalName }}", {{ .VarName }}s)
{{- if not $checkNil }}
}
{{- end }}
		{{- end }}

		{{- if $initDef }}
	{{ if $checkNil }} } else { {{ else }}if {{ if eq $.HeaderSourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}.{{ .FieldName }} == nil { {{ end }}
		w.Header().Set("{{ .CanonicalName }}", "{{ printValue .Type .DefaultValue }}")
		{{- end }}

		{{- if or $checkNil $initDef }}
	}
		{{- end }}

	{{- end }}

	{{- range .Cookies }}
		{{- $initDef := and (or .FieldPointer .Slice) .DefaultValue }}
		{{- $checkNil := and (or .FieldPointer .Slice (eq .Type.Name "bytes") (eq .Type.Name "any") $initDef) }}
		{{- if $checkNil }}
	if {{ if eq $.HeaderSourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}.{{ .FieldName }} != nil {
		{{- end }}

		{{- if eq .Type.Name "string" }}
	{{ .VarName }} := {{ if or .FieldPointer $.ViewedResult }}*{{ end }}{{ if eq $.HeaderSourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}{{ if .FieldName }}.{{ .FieldName }}{{ end }}
		{{- else }}
			{{- if isAliased .FieldType }}
	{{ .VarName }}raw := {{ goTypeRef .Type }}({{ if .FieldPointer }}*{{ end }}{{ if eq $.HeaderSourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}{{ if .FieldName }}.{{ .FieldName }}{{ end }})
	{{ template "partial_header_conversion" (headerConversionData .Type (printf "%sraw" .VarName) true .VarName) }}
			{{- else }}
	{{ .VarName }}raw := {{ if eq $.HeaderSourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}{{ if .FieldName }}.{{ .FieldName }}{{ end }}
	{{ template "partial_header_conversion" (headerConversionData .Type (printf "%sraw" .VarName) (not .FieldPointer) .VarName) }}
			{{- end }}
		{{- end }}

		{{- if $initDef }}
	{{ if $checkNil }} } else { {{ else }}if {{ if eq $.HeaderSourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}.{{ .FieldName }} == nil { {{ end }}
		{{ .VarName }} := "{{ printValue .Type .DefaultValue }}"
		{{- end }}
		http.SetCookie(w, &http.Cookie{
			Name: {{ printf "%q" .HTTPName }},
			Value: {{ .VarName }},
			{{- if .MaxAge }}
			MaxAge: {{ .MaxAge }},
			{{- end }}
			{{- if .Path }}
			Path: {{ printf "%q" .Path }},
			{{- end }}
			{{- if .Domain }}
			Domain: {{ printf "%q" .Domain }},
			{{- end }}
			{{- if .Secure }}
			Secure: true,
			{{- end }}
			{{- if .HTTPOnly }}
			HttpOnly: true,
			{{- end }}
			{{- if .SameSite }}
			SameSite: {{ .SameSite }},
			{{- end }}
		})
		{{- if or $checkNil $initDef }}
	}
		{{- end }}

	{{- end }}

	{{- if .ErrorHeader }}
	w.Header().Set("loom-error", res.LoomErrorName())
	{{- end }}
	w.WriteHeader({{ .StatusCode }})`},
		templateSource{name: "header_conversion", source: `	{{- if eq .Type.Name "boolean" -}}
		{{ .VarName }} := strconv.FormatBool({{ if not .Required }}*{{ end }}{{ .Target }})
	{{- else if eq .Type.Name "int" -}}
		{{ .VarName }} := strconv.Itoa({{ if not .Required }}*{{ end }}{{ .Target }})
	{{- else if eq .Type.Name "int32" -}}
		{{ .VarName }} := strconv.FormatInt(int64({{ if not .Required }}*{{ end }}{{ .Target }}), 10)
	{{- else if eq .Type.Name "int64" -}}
		{{ .VarName }} := strconv.FormatInt({{ if not .Required }}*{{ end }}{{ .Target }}, 10)
	{{- else if eq .Type.Name "uint" -}}
		{{ .VarName }} := strconv.FormatUint(uint64({{ if not .Required }}*{{ end }}{{ .Target }}), 10)
	{{- else if eq .Type.Name "uint32" -}}
		{{ .VarName }} := strconv.FormatUint(uint64({{ if not .Required }}*{{ end }}{{ .Target }}), 10)
	{{- else if eq .Type.Name "uint64" -}}
		{{ .VarName }} := strconv.FormatUint({{ if not .Required }}*{{ end }}{{ .Target }}, 10)
	{{- else if eq .Type.Name "float32" -}}
		{{ .VarName }} := strconv.FormatFloat(float64({{ if not .Required }}*{{ end }}{{ .Target }}), 'f', -1, 32)
	{{- else if eq .Type.Name "float64" -}}
		{{ .VarName }} := strconv.FormatFloat({{ if not .Required }}*{{ end }}{{ .Target }}, 'f', -1, 64)
	{{- else if eq .Type.Name "string" -}}
		{{ .VarName }} := {{ .Target }}
	{{- else if eq .Type.Name "bytes" -}}
		{{ .VarName }} := string({{ .Target }})
	{{- else if eq .Type.Name "any" -}}
		{{ .VarName }} := fmt.Sprintf("%v", {{ .Target }})
	{{- else if eq .Type.Name "array" -}}
		{{- if eq .Type.ElemType.Type.Name "string" -}}
		{{ .VarName }} := strings.Join({{ .Target }}, ", ")
		{{- else -}}
		{{ .VarName }}Slice := make([]string, len({{ .Target }}))
		for i, e := range {{ .Target }}  {
			{{ template "partial_header_conversion" (headerConversionData .Type.ElemType.Type "es" true "e") }}
			{{ .VarName }}Slice[i] = es
		}
		{{ .VarName }} := strings.Join({{ .VarName }}Slice, ", ")
		{{- end }}
	{{- else }}
		// unsupported type {{ .Type.Name }} for header field {{ .FieldName }}
	{{- end }}`},
	)
)
