package codegen

var (
	responseEncoderSource = joinHTTPTemplateSource(`{{ printf "%s returns an encoder for responses returned by the %s %s endpoint." .ResponseEncoder .ServiceName .Method.Name | comment }}
func {{ .ResponseEncoder }}(encoder func(context.Context, http.ResponseWriter) loomhttp.Encoder) func(context.Context, http.ResponseWriter, any) error {
	return func(ctx context.Context, w http.ResponseWriter, v any) error {
	{{- if .Result.MustInit }}
		{{- if .Method.ViewedResult }}
			res, ok := v.({{ .Method.ViewedResult.FullRef }})
			if !ok {
				return loomhttp.ErrInvalidType("{{ .ServiceName }}", "{{ .Method.Name }}", "{{ .Method.ViewedResult.FullRef }}", v)
			}
			{{- if not .Method.ViewedResult.ViewName }}
				w.Header().Set("loom-view", res.View)
			{{- end }}
		{{- else }}
			{{- if .Result.IsAny }}
			res := v
			{{- else }}
			res, ok := v.({{ .Result.Ref }})
			if !ok {
				return loomhttp.ErrInvalidType("{{ .ServiceName }}", "{{ .Method.Name }}", "{{ .Result.Ref }}", v)
			}
			{{- end }}
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
			{{- if .EncodePlan.HasBody }}
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
			{{- if not (or $.Method.SkipResponseBodyEncodeDecode $.Method.FileResponse) }}
				w.WriteHeader({{ .StatusCode }})
			{{- end }}
			return nil
		{{- end }}
	{{- end }}
	}
}
`,
		responseEncoderPartials...,
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
			if !errors.As(v, &res) {
				return encodeError(ctx, w, v)
			}
			{{- with .Response}}
				{{- if .ContentType }}
					ctx = context.WithValue(ctx, loomhttp.ContentTypeKey, "{{ .ContentType }}")
				{{- end }}
				{{- template "partial_response" . }}
				{{- if .EncodePlan.HasBody }}
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
		responseEncoderPartials...,
	)
	responseEncoderPartials = []templateSource{
		{name: "response", source: `	{{- if .EncodePlan.HasBody }}
	enc := encoder(ctx, w)
		{{- if .EncodePlan.UseViewedBodySwitch }}
	var body any
	switch res.View	{
			{{- range $.ViewedResult.Views }}
	case {{ printf "%q" .Name }}{{ if eq .Name "default" }}, ""{{ end }}:
		{{- $vsb := (viewedServerBody $.ServerBody .Name) }}
		body = {{ $vsb.Init.Name }}({{ range $vsb.Init.ServerArgs }}{{ .Ref }}, {{ end }})
			{{- end }}
	}
		{{- else if .EncodePlan.FirstBody.Init }}
			{{- if .ErrorHeader }}
	var body any
	if formatter != nil {
		body = formatter(ctx, {{ (index .EncodePlan.FirstBody.Init.ServerArgs 0).Ref }})
	} else {
			{{- end }}
	body {{ if not .ErrorHeader}}:{{ end }}= {{ .EncodePlan.FirstBody.Init.Name }}({{ range .EncodePlan.FirstBody.Init.ServerArgs }}{{ .Ref }}, {{ end }})
			{{- if .ErrorHeader }}
	}
			{{- end }}
		{{- else }}
	body := res{{ if $.ViewedResult }}.Projected{{ end }}{{ if .ResultAttr }}.{{ .ResultAttr }}{{ end }}
		{{- end }}
	{{- end }}
	{{- if .EncodePlan.NeedsProblemSource }}
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
		if err := loomhttp.SetResponseCookie(ctx, w, &http.Cookie{
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
		}); err != nil {
			return err
		}
		{{- if or $checkNil $initDef }}
	}
		{{- end }}

	{{- end }}

	{{- if .ErrorHeader }}
	w.Header().Set("loom-error", res.LoomErrorName())
	{{- end }}
		{{- if not .DeferStatus }}
	w.WriteHeader({{ .StatusCode }})
	{{- end }}`},
		{name: "header_conversion", source: `	{{- if eq .Type.Name "boolean" -}}
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
	}
)
