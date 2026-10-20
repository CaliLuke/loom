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
			res, ok := v.({{ .Result.Ref }})
			if !ok {
				return loomhttp.ErrInvalidType("{{ .ServiceName }}", "{{ .Method.Name }}", "{{ .Result.Ref }}", v)
			}
		{{- end }}
		{{- range .Result.Responses }}
			{{- if .ContentType }}
				{{- if $.Method.SkipResponseBodyEncodeDecode }}
					w.Header().Set("Content-Type", "{{ .ContentType }}")
				{{- else }}
					ctx = context.WithValue(ctx, loomhttp.ContentTypeKey, "{{ .ContentType }}")
				{{- end }}
			{{- end }}
			{{- if .Tag }}
				if {{ .Tag.Match }} {
			{{- end -}}
			{{ template "partial_response" . }}
			{{- if .EncodePlan.HasBody }}
				return enc.Encode(body)
			{{- else }}
				return nil
			{{- end }}
			{{- if .Tag }}
				}
			{{- end }}
		{{- end }}
	{{- else }}
		{{- with (index .Result.Responses 0) }}
			{{- if and $.Method.SkipResponseBodyEncodeDecode .ContentType }}
				w.Header().Set("Content-Type", "{{ .ContentType }}")
			{{- end }}
			{{- if and (not $.Method.FileResponse) (or (not $.Method.SkipResponseBodyEncodeDecode) (ne .StatusCode "http.StatusOK")) }}
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
func {{ .ErrorEncoder }}(encoder func(context.Context, http.ResponseWriter) loomhttp.Encoder, formatter func(ctx context.Context, err error) loomhttp.StatusCoder) func(context.Context, http.ResponseWriter, error) error {
	encodeError := loomhttp.ErrorEncoder(encoder, formatter)
	return func(ctx context.Context, w http.ResponseWriter, v error) error {
		errorResponse := loom.ResolveErrorResponse(v)
		if errorResponse.Owner == nil {
			return encodeError(ctx, w, v)
		}
		switch errorResponse.Name {
	{{- range $gerr := .Errors }}
	{{- range $err := .Errors }}
		case {{ printf "%q" .Name }}:
			//nolint:errorlint // Response ownership is already resolved.
			{{ if .Response.EncodePlan.NeedsErrorSource }}res{{ else }}_{{ end }}, ok := errorResponse.Owner.({{ $err.Ref }})
			if !ok {
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
		{name: "response", source: `	{{- if .EncodePlan.NeedsProblemSource }}
	problem := loomhttp.NewProblemResponse(ctx, {{ if .ErrorHeader }}v{{ else }}res{{ end }}, {{ .Code }}, {{ .ProblemTypeOverride }}, {{ .ProblemTitleOverride }})
	{{- end }}
	{{- if .EncodePlan.HasBody }}
	enc := encoder(ctx, w)
		{{- if .ErrorHeader }}
	var body any
	if formatter != nil {
		body = formatter(ctx, v)
	} else {
		{{- end }}
		{{- if .EncodePlan.UseViewedBodySwitch }}
			{{- if not .ErrorHeader }}
	var body any
			{{- end }}
	switch res.View	{
			{{- range $.ViewedResult.Views }}
	case {{ printf "%q" .Name }}{{ if eq .Name "default" }}, ""{{ end }}:
			{{- $vsb := (viewedServerBody $.ServerBody .Name) }}
		{{ if and $vsb.Init $vsb.Init.ServerErrorAware }}convertedBody, conversionError := {{ index $.EncodePlan.BodyValues $vsb }}
		if conversionError != nil {
			return conversionError
		}
		body = convertedBody{{ else }}body = {{ index $.EncodePlan.BodyValues $vsb }}{{ end }}
			{{- end }}
	}
		{{- else }}
	{{ if and .EncodePlan.FirstBody.Init .EncodePlan.FirstBody.Init.ServerErrorAware }}convertedBody, conversionError := {{ index .EncodePlan.BodyValues .EncodePlan.FirstBody }}
	if conversionError != nil {
		return conversionError
	}
	body {{ if not .ErrorHeader}}:{{ end }}= convertedBody{{ else }}body {{ if not .ErrorHeader}}:{{ end }}= {{ index .EncodePlan.BodyValues .EncodePlan.FirstBody }}{{ end }}
		{{- end }}
		{{- if .ErrorHeader }}
	}
		{{- end }}
	{{- end }}
 {{- $source := "res" }}
 {{- if eq .SourceVar "problem" }}{{ $source = "problem" }}{{ else if .ViewedResult }}{{ $source = "res.Projected" }}{{ end }}
	{{- range .Headers }}
		{{- $initDef := and (or .FieldPointer .Slice) (or .DefaultValue (and .Slice (ne .DefaultValue nil))) (not $.Tag) }}
		{{- $checkNil := and (or .FieldPointer .Slice (eq .Type.Name "bytes") (eq .Type.Name "any") $initDef) (not $.Tag) }}
		{{- if $checkNil }}
	if {{ if eq $.SourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}{{ if .FieldName }}.{{ .FieldName }}{{ end }} != nil {
		{{- end }}

		{{- if and (eq .Type.Name "string") (not (isAliased .FieldType)) }}
	w.Header().Set("{{ .CanonicalName }}", {{ transportScalarValue .Type .FieldType (transportFieldReference .AttributeData $source) .IsTextUnmarshaler }})
		{{- else }}
{{- if not $checkNil }}
{
{{- end }}
	val := {{ if .FieldPointer }}*{{ end }}{{ if eq $.SourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}{{ if .FieldName }}.{{ .FieldName }}{{ end }}
	{{ renderHeaderConversion .Type .FieldType .Locals.Encoded "val" .IsTextUnmarshaler }}
	{{- if .Slice }}
	if err := loomhttp.SetHeaderValues(w.Header(), "{{ .CanonicalName }}", {{ .Locals.Encoded }}); err != nil {
		return err
	}
	{{- else }}
	w.Header().Set("{{ .CanonicalName }}", {{ .Locals.Encoded }})
	{{- end }}
{{- if not $checkNil }}
}
{{- end }}
		{{- end }}

		{{- if $initDef }}
	{{ if $checkNil }} } else { {{ else }}if {{ if eq $.SourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}.{{ .FieldName }} == nil { {{ end }}
		{{- if .Slice }}
		val := {{ printf "%#v" .DefaultValue }}
		{{ renderHeaderConversion .Type .Type .Locals.Encoded "val" false }}
		if err := loomhttp.SetHeaderValues(w.Header(), "{{ .CanonicalName }}", {{ .Locals.Encoded }}); err != nil {
			return err
		}
		{{- else }}
		w.Header().Set("{{ .CanonicalName }}", "{{ printValue .Type .DefaultValue }}")
		{{- end }}
		{{- end }}

		{{- if or $checkNil $initDef }}
	}
		{{- end }}

	{{- end }}

	{{- range .Cookies }}
		{{- $initDef := and (or .FieldPointer .Slice) .DefaultValue }}
		{{- $checkNil := and (or .FieldPointer .Slice (eq .Type.Name "bytes") (eq .Type.Name "any") $initDef) }}
		{{- if $checkNil }}
	if {{ if eq $.SourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}.{{ .FieldName }} != nil {
		{{- end }}

		{{- if and (eq .Type.Name "string") (not (isAliased .FieldType)) }}
	{{ .VarName }} := {{ transportScalarValue .Type .FieldType (transportFieldReference .AttributeData $source) .IsTextUnmarshaler }}
		{{- else }}
	{{ .Locals.EncodedRaw }} := {{ if .FieldPointer }}*{{ end }}{{ if eq $.SourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}{{ if .FieldName }}.{{ .FieldName }}{{ end }}
	{{ renderHeaderConversion .Type .FieldType .VarName .Locals.EncodedRaw .IsTextUnmarshaler }}
		{{- end }}

		{{- if $initDef }}
	{{ if $checkNil }} } else { {{ else }}if {{ if eq $.SourceVar "problem" }}problem{{ else }}res{{ if $.ViewedResult }}.Projected{{ end }}{{ end }}.{{ .FieldName }} == nil { {{ end }}
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

	{{- if and .ErrorHeader (not .ErrorNameFromStatus) }}
	w.Header().Set("loom-error", res.LoomErrorName())
	{{- end }}
		{{- if not .DeferStatus }}
	w.WriteHeader({{ .StatusCode }})
	{{- end }}`},
	}
)
