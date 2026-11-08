package codegen

var (
	responseEncoderSource = joinHTTPTemplateSource(`{{ printf "%s returns an encoder for responses returned by the %s %s endpoint." .ResponseEncoder .ServiceName .Method.Name | comment }}
func {{ .ResponseEncoder }}(encoder func(context.Context, http.ResponseWriter) loomhttp.Encoder) func(context.Context, http.ResponseWriter, any{{ if .Method.HasResponseMetadata }}, loomhttp.SSEHandshake{{ end }}) error {
	return func(ctx context.Context, w http.ResponseWriter, v any{{ if .Method.HasResponseMetadata }}, metadata loomhttp.SSEHandshake{{ end }}) error {
	{{- if .NoSuccess }}
		return &loomhttp.StatusViolationError{}
	{{- else if .Result.MustInit }}
		{{- if .Method.ViewedResult }}
			res, ok := v.({{ .Method.ViewedResult.FullRef }})
			if !ok {
				return loomhttp.ErrInvalidType("{{ .ServiceName }}", "{{ .Method.Name }}", "{{ .Method.ViewedResult.FullRef }}", v)
			}
		{{- else }}
			res, ok := v.({{ .Result.Ref }})
			if !ok {
				return loomhttp.ErrInvalidType("{{ .ServiceName }}", "{{ .Method.Name }}", "{{ .Result.Ref }}", v)
			}
		{{- end }}
		{{- if .SelectsNativeStatus }}
		{{ renderFinalSuccessSelection . }}
		{{- end }}
		{{- range responsePayloads . }}
			{{- if $.SelectsNativeStatus }}
			if selected == ({{ responseSelectorValue . }}) {
			{{- end }}
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
			{{- if $.Method.HasResponseMetadata }}
			for name, values := range metadata.Header {
				w.Header()[name] = append([]string(nil), values...)
			}
			w.WriteHeader(status)
			{{- end }}
			{{- if .EncodePlan.HasBody }}
				return enc.Encode(body)
			{{- else }}
				return nil
			{{- end }}
			{{- if or .Tag $.SelectsNativeStatus }}
				}
			{{- end }}
		{{- end }}
		{{- if .SelectsNativeStatus }}
		return &loomhttp.StatusViolationError{Code: status, Selector: selected}
		{{- end }}
	{{- else }}
		{{- with (index .Result.Responses 0) }}
			{{- if and $.Method.SkipResponseBodyEncodeDecode .ContentType }}
				w.Header().Set("Content-Type", "{{ .ContentType }}")
			{{- end }}
			{{- if and (not $.Method.FileResponse) (or (not $.Method.SkipResponseBodyEncodeDecode) (ne .LiteralStatus "http.StatusOK")) }}
				w.WriteHeader({{ responseWireStatus . }})
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
			{{ if .Response.ErrorPayload }}errorOwner{{ else if .Response.EncodePlan.NeedsErrorSource }}res{{ else }}_{{ end }}, ok := errorResponse.Owner.({{ $err.Ref }})
			if !ok {
				return encodeError(ctx, w, v)
			}
			{{- with .Response}}
				{{- if .ErrorPayload }}
				{{ if or .EncodePlan.HasBody .Headers .Cookies .HasStatusBinding }}res{{ else }}_{{ end }}, payloadError := errorOwner.Payload()
				if payloadError != nil {
					return payloadError
				}
				{{- end }}
				{{- if .HasStatusBinding }}
				{{ renderFinalErrorStatus . }}
				{{- end }}
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
		{name: "response", source: `	{{- $errorVariable := "res" }}
	{{- if .ErrorHeader }}{{ $errorVariable = "v" }}{{ end }}
	{{- $prepareProblem := .ProblemSourcePreparation $errorVariable }}
	{{- if $prepareProblem }}
{{ $prepareProblem }}
	{{- end }}
	{{- $bodyCondition := "" }}
	{{- if and .ErrorHeader .EncodePlan.HasBody }}{{ $bodyCondition = "formatter == nil" }}{{ end }}
	{{- $prepareSource := .EncodePlan.SourcePreparation nil $bodyCondition }}
	{{- if $prepareSource }}
{{ $prepareSource }}
	{{- end }}
	{{- if and .ViewedResult (not .ViewedResult.ViewName) (not .HasStatusBinding) }}
	w.Header().Set("loom-view", res.View)
	{{- end }}
 {{- $source := "res" }}
 {{- if eq .SourceVar "problem" }}{{ $source = "problem" }}{{ else if .ViewedResult }}{{ $source = "res.Projected" }}{{ end }}
 {{- if .EncodePlan.FieldSource }}{{ $source = .EncodePlan.FieldSource }}{{ end }}
 {{- if .HasStatusBinding }}
 {{ renderFinalMetadataPreparation . }}
 {{- if and .ViewedResult (not .ViewedResult.ViewName) }}
 preparedHeaders.Set("loom-view", res.View)
 {{- end }}
 {{- end }}
	{{- if .EncodePlan.HasBody }}
    {{- if not .EncodePlan.FirstBody.Media }}
	enc := encoder(ctx, w)
    {{- end }}
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
 {{- if not .HasStatusBinding }}
 {{- if .EncodePlan.MetadataGuard }}
 if {{ .EncodePlan.MetadataGuard }} {
 {{- end }}
	{{- with renderResponseHeaders . $source "w.Header()" }}
	{{ . }}
	{{- end }}
	{{- if .Cookies }}
	{{ renderResponseCookies . $source "w.Header()" "ctx" }}
	{{- end }}
 {{- if .EncodePlan.MetadataGuard }}
 }
 {{- end }}
 {{- end }}

	{{- if and .ErrorHeader (not .ErrorNameFromStatus) }}
	{{ if .HasStatusBinding }}preparedHeaders{{ else }}w.Header(){{ end }}.Set("loom-error", {{ if .ErrorPayload }}{{ printf "%q" .ErrorHeader }}{{ else }}res.LoomErrorName(){{ end }})
	{{- end }}
    {{- if and .EncodePlan.HasBody .EncodePlan.FirstBody.Media }}
    {{- if .ErrorHeader }}
    var enc loomhttp.Encoder
    if formatter != nil {
        enc = encoder(ctx, w)
    } else {
    {{- end }}
    mediaIndex := -1
    var mediaValue any
    switch mediaBody := any(body).(type) {
    {{- range .ServerBody }}
    case {{ .Ref }}:
        switch string(mediaBody.Kind()) {
        {{- range $i, $variant := .Media.Variants }}
        case {{ printf "%q" .Tag }}:
            mediaIndex, mediaValue = {{ $i }}, mediaBody.{{ .Field }}
{{- if .Multipart }}
ctx = {{ renderMediaMultipartEncode . "ctx" (printf "mediaBody.%s" .Field) }}
{{- else if mediaFormCodec . }}
ctx = {{ renderMediaFormEncode . "ctx" (printf "mediaBody.%s" .Field) false }}
{{- end }}
        {{- end }}
        }
    {{- end }}
    }
    mediaEncoder, mediaErr := loomhttp.MediaResponseEncoder(ctx, w, mediaIndex, {{ .EncodePlan.FirstBody.Media.Contracts }}, mediaValue, nil, encoder)
    if mediaErr != nil {
        return mediaErr
    }
    enc {{ if .ErrorHeader }}={{ else }}:={{ end }} mediaEncoder
    {{- if .ErrorHeader }}
    }
    {{- end }}
    {{- end }}
	{{- if .HasStatusBinding }}
 for name, values := range preparedHeaders {
  w.Header()[name] = append([]string(nil), values...)
 }
 {{- end }}
		{{- if not .DeferStatus }}
	w.WriteHeader({{ responseWireStatus . }})
	{{- end }}`},
	}
)
