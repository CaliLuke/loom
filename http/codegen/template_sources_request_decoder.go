package codegen

var (
	requestDecoderSource = joinHTTPTemplateSource(`{{ printf "%s returns a decoder for requests sent to the %s %s endpoint." .RequestDecoder .ServiceName .Method.Name | comment }}
{{- $usesDecoder := or .MultipartRequestDecoder (and .Payload.Request.ServerBody (not .Payload.Request.MultipartGenerated) (not .Payload.Request.FormEncoded)) }}
func {{ .RequestDecoder }}(mux loomhttp.Muxer, {{ if $usesDecoder }}decoder{{ else }}_{{ end }} func(*http.Request) loomhttp.Decoder) func(*http.Request{{ if .Method.IsJSONRPC }}, *jsonrpc.RawRequest{{ end }}) ({{ .Payload.Ref }}, error) {
	return func(r *http.Request{{ if .Method.IsJSONRPC }}, req *jsonrpc.RawRequest{{ end }}) ({{ .Payload.Ref }}, error) {
	{{- if .Method.IsJSONRPC }}
		params := req.Params
		{{- if not (or .Payload.Request.OptionalBodyAttribute .Payload.Request.ExplicitPresenceBody (ne .Payload.Request.BodyDefaultValue nil)) }}
		if len(params) == 0 {
			params = []byte("{}")
		}
		{{- end }}
		r.Body = io.NopCloser(bytes.NewReader(params))
	{{- end }}
		var payload {{ .Payload.Ref }}
{{- if .MultipartRequestDecoder }}
		if err := decoder(r).Decode(&payload); err != nil {
			var gerr *loom.ServiceError
			if errors.As(err, &gerr) {
				return payload, gerr
			}
			return payload, loom.DecodePayloadError(loomhttp.SafeDecodePayloadMessage(err))
		}
{{- else if .Payload.Request.ServerBody }}
		var (
		{{- if .Payload.Request.OptionalObjectBody }}
			body = &{{ .Payload.Request.ServerBody.VarName }}{}
		{{- else if .Payload.Request.OptionalPrimitiveBody }}
			body = new({{ .Payload.Request.ServerBody.VarName }})
		{{- else if ne .Payload.Request.BodyDefaultValue nil }}
			body = {{ .Payload.Request.ServerBody.ValueRef }}({{ printf "%#v" .Payload.Request.BodyDefaultValue }})
		{{- else }}
			body {{ .Payload.Request.ServerBody.ValueRef }}
		{{- end }}
		{{- if .Payload.Request.NeedsServerErrorVar }}
			err  error
		{{- end }}
		)
	{{- if .Payload.Request.MultipartGenerated }}
		mr, multipartErr := r.MultipartReader()
		if multipartErr != nil {
			var gerr *loom.ServiceError
			if errors.As(multipartErr, &gerr) {
				return payload, gerr
			}
			return payload, loom.DecodePayloadError(loomhttp.SafeDecodePayloadMessage(multipartErr))
		}
		multipartForm, multipartErr := loomhttp.ReadMultipartFormWithLimit(mr, loomhttp.RequestBodyLimit(r.Context()))
		if multipartErr != nil {
			var gerr *loom.ServiceError
			if errors.As(multipartErr, &gerr) {
				return payload, gerr
			}
			return payload, loom.DecodePayloadError(loomhttp.SafeDecodePayloadMessage(multipartErr))
		}
		if len(multipartForm.Values) == 0 && len(multipartForm.Files) == 0 {
		{{- if .Payload.Request.MustHaveBody }}
			return payload, loom.MissingPayloadError()
		{{- end }}
		} else {
		{{- range .Payload.Request.MultipartFileFields }}
			switch files := multipartForm.Files["{{ .HTTPName }}"]; len(files) {
			case 0:
				// A missing required file is reported by the body validation
				// below, which independently detects the resulting nil field.
			case 1:
				multipartForm.Values.Set("{{ .HTTPName }}", string(files[0].Data))
			{{- if .PopulateFilename }}
				if _, ok := multipartForm.Values["filename"]; !ok && files[0].Filename != "" {
					multipartForm.Values.Set("filename", files[0].Filename)
				}
			{{- end }}
			{{- if .PopulateContentType }}
				if _, ok := multipartForm.Values["content_type"]; !ok && files[0].ContentType != "" {
					multipartForm.Values.Set("content_type", files[0].ContentType)
				}
			{{- end }}
			default:
				return payload, loom.DecodePayloadError("multiple multipart files provided for field {{ .HTTPName }}")
			}
		{{- end }}
			if _, multipartErr = loomhttp.DecodeFormValue(multipartForm.Values, "", &body); multipartErr != nil {
				var gerr *loom.ServiceError
				if errors.As(multipartErr, &gerr) {
					return payload, gerr
				}
				return payload, loom.DecodePayloadError(loomhttp.SafeDecodePayloadMessage(multipartErr))
			}
		}
	{{- else if .Payload.Request.FormEncoded }}
		if err = loomhttp.ParseFormWithLimit(r, loomhttp.RequestBodyLimit(r.Context())); err != nil {
			var gerr *loom.ServiceError
			if errors.As(err, &gerr) {
				return payload, gerr
			}
			return payload, loom.DecodePayloadError(loomhttp.SafeDecodePayloadMessage(err))
		}
		if len(r.PostForm) == 0 {
		{{- if .Payload.Request.MustHaveBody }}
			return payload, loom.MissingPayloadError()
		{{- else if .Payload.Request.OptionalObjectBody }}
			body = nil
		{{- end }}
		} else {
			if _, err = loomhttp.DecodeFormValue(r.PostForm, "", {{ if not .Payload.Request.OptionalObjectBody }}&{{ end }}body); err != nil {
				var gerr *loom.ServiceError
				if errors.As(err, &gerr) {
					return payload, gerr
				}
				return payload, loom.DecodePayloadError(loomhttp.SafeDecodePayloadMessage(err))
			}
		}
	{{- else }}
		err = decoder({{ if .Payload.Request.BodyAllowsNull }}r{{ else }}loomhttp.WithNonNullableBody(r){{ end }}).Decode({{ if not (or .Payload.Request.OptionalObjectBody .Payload.Request.OptionalPrimitiveBody) }}&{{ end }}body)
		if err != nil {
		{{- if .Payload.Request.MustHaveBody }}
			if errors.Is(err, io.EOF) {
				return payload, loom.MissingPayloadError()
			}
		{{- else }}
			if errors.Is(err, io.EOF) {
			{{- if or .Payload.Request.OptionalObjectBody .Payload.Request.OptionalPrimitiveBody }}
				body = nil
			{{- end }}
				err = nil
			} else {
		{{- end }}
			var gerr *loom.ServiceError
			if errors.As(err, &gerr) {
				return payload, gerr
			}
			return payload, loom.DecodePayloadError(loomhttp.SafeDecodePayloadMessage(err))
		{{- if not .Payload.Request.MustHaveBody }}
			}
		{{- end }}
		}
	{{- end }}
	{{- if .Payload.Request.ServerBody.ValidateRef }}
		{{- if .Payload.Request.OptionalObjectBody }}
		if body != nil {
		{{- end }}
		{{ .Payload.Request.ServerBody.ValidateRef }}
		if err != nil {
		{{- if .Payload.Request.MultipartGenerated }}
			if multipartErr != nil {
				err = loom.MergeErrors(multipartErr, err)
			}
		{{- end }}
			return payload, err
		}
		{{- if .Payload.Request.OptionalObjectBody }}
		}
		{{- end }}
	{{- end }}
	{{- if .Payload.Request.MultipartGenerated }}
		if multipartErr != nil {
			return payload, multipartErr
		}
	{{- end }}
{{- end }}
{{- if not .MultipartRequestDecoder }}
	{{- if .Payload.Request.DecodePlan.HasElements }}
		{{- template "partial_request_elements" .Payload.Request }}
	{{- end }}
	{{- if .Payload.Request.DecodePlan.MustValidate }}
		if err != nil {
			return payload, err
		}
	{{- end }}
	{{- if .Payload.Request.PayloadInit }}
	payload = {{ .Payload.Request.PayloadInit.Name }}({{ range .Payload.Request.PayloadInit.ServerArgs }}{{ .Ref }}, {{ end }})
	{{- else if .Payload.DecoderReturnValue }}
	payload = {{ .Payload.DecoderReturnValue }}
	{{- else }}
	payload = body
	{{- end }}
{{- end }}
{{- if .BasicScheme }}{{ with .BasicScheme }}
	user, pass, {{ if or .UsernameRequired .PasswordRequired }}ok{{ else }}_{{ end }} := r.BasicAuth()
		{{- if or .UsernameRequired .PasswordRequired}}
	if !ok {
		return payload, loom.MissingFieldError("Authorization", "header")
	}
		{{- end }}
	{{- if isAliased .UsernameType }}
 userValue := {{ credentialTypeRef .UsernameType $.ServicePkgName }}(user)
 {{- end }}
 {{- if isAliased .PasswordType }}
 passValue := {{ credentialTypeRef .PasswordType $.ServicePkgName }}(pass)
 {{- end }}
 payload.{{ .UsernameField }} = {{ if .UsernamePointer }}&{{ end }}{{ if isAliased .UsernameType }}userValue{{ else }}user{{ end }}
	payload.{{ .PasswordField }} = {{ if .PasswordPointer }}&{{ end }}{{ if isAliased .PasswordType }}passValue{{ else }}pass{{ end }}
{{- end }}{{ end }}
{{- range .HeaderSchemes }}
	{{- if not .CredRequired }}
	if payload.{{ .CredField }} != nil {
	{{- end }}
	{{- if ne .Type "APIKey" }}
	if strings.Contains({{ if isAliased .CredType }}string({{ end }}{{ if .CredPointer }}*{{ end }}payload.{{ .CredField }}{{ if isAliased .CredType }}){{ end }}, " ") {
		// Remove authorization scheme prefix (e.g. "Bearer")
		cred := {{ if isAliased .CredType }}{{ credentialTypeRef .CredType $.ServicePkgName }}({{ end }}strings.SplitN({{ if isAliased .CredType }}string({{ end }}{{ if .CredPointer }}*{{ end }}payload.{{ .CredField }}{{ if isAliased .CredType }}){{ end }}, " ", 2)[1]{{ if isAliased .CredType }}){{ end }}
		payload.{{ .CredField }} = {{ if .CredPointer }}&{{ end }}cred
	}
	{{- end }}
	{{- if not .CredRequired }}
	}
	{{- end }}
{{- end }}

	return payload, nil
	}
}
`, requestDecoderPartials...)
)
