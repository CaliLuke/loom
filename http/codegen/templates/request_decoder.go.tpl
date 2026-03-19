{{ printf "%s returns a decoder for requests sent to the %s %s endpoint." .RequestDecoder .ServiceName .Method.Name | comment }}
func {{ .RequestDecoder }}(mux goahttp.Muxer, decoder func(*http.Request) goahttp.Decoder) func(*http.Request) ({{ .Payload.Ref }}, error) {
	return func(r *http.Request) ({{ .Payload.Ref }}, error) {
		var payload {{ .Payload.Ref }}
{{- if .MultipartRequestDecoder }}
		if err := decoder(r).Decode(&payload); err != nil {
			var gerr *goa.ServiceError
			if errors.As(err, &gerr) {
				return payload, gerr
			}
			return payload, goa.DecodePayloadError(err.Error())
		}
{{- else if .Payload.Request.ServerBody }}
		var (
			body {{ .Payload.Request.ServerBody.VarName }}
			err  error
		)
	{{- if .Payload.Request.MultipartGenerated }}
		mr, multipartErr := r.MultipartReader()
		if multipartErr != nil {
			var gerr *goa.ServiceError
			if errors.As(multipartErr, &gerr) {
				return payload, gerr
			}
			return payload, goa.DecodePayloadError(multipartErr.Error())
		}
		multipartForm, multipartErr := goahttp.ReadMultipartForm(mr)
		if multipartErr != nil {
			var gerr *goa.ServiceError
			if errors.As(multipartErr, &gerr) {
				return payload, gerr
			}
			return payload, goa.DecodePayloadError(multipartErr.Error())
		}
		if len(multipartForm.Values) == 0 && len(multipartForm.Files) == 0 {
		{{- if .Payload.Request.MustHaveBody }}
			return payload, goa.MissingPayloadError()
		{{- end }}
		} else {
		{{- range .Payload.Request.MultipartFileFields }}
			files := multipartForm.Files["{{ .HTTPName }}"]
			switch len(files) {
			case 0:
			{{- if .Required }}
				err = goa.MergeErrors(err, goa.MissingFieldError("{{ .Name }}", "body"))
			{{- end }}
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
				return payload, goa.DecodePayloadError("multiple multipart files provided for field {{ .HTTPName }}")
			}
		{{- end }}
			if _, multipartErr = goahttp.DecodeFormValue(multipartForm.Values, "", &body); multipartErr != nil {
				var gerr *goa.ServiceError
				if errors.As(multipartErr, &gerr) {
					return payload, gerr
				}
				return payload, goa.DecodePayloadError(multipartErr.Error())
			}
		}
	{{- else if .Payload.Request.FormEncoded }}
		if err = r.ParseForm(); err != nil {
			return payload, goa.DecodePayloadError(err.Error())
		}
		if len(r.PostForm) == 0 {
		{{- if .Payload.Request.MustHaveBody }}
			return payload, goa.MissingPayloadError()
		{{- end }}
		} else {
			if _, err = goahttp.DecodeFormValue(r.PostForm, "", &body); err != nil {
				var gerr *goa.ServiceError
				if errors.As(err, &gerr) {
					return payload, gerr
				}
				return payload, goa.DecodePayloadError(err.Error())
			}
		}
	{{- else }}
		err = decoder(r).Decode(&body)
		if err != nil {
		{{- if .Payload.Request.MustHaveBody }}
			if errors.Is(err, io.EOF) {
				return payload, goa.MissingPayloadError()
			}
		{{- else }}
			if errors.Is(err, io.EOF) {
				err = nil
			} else {
		{{- end }}
			var gerr *goa.ServiceError
			if errors.As(err, &gerr) {
				return payload, gerr
			}
			return payload, goa.DecodePayloadError(err.Error())
		{{- if not .Payload.Request.MustHaveBody }}
			}
		{{- end }}
		}
	{{- end }}
	{{- if .Payload.Request.ServerBody.ValidateRef }}
		{{ .Payload.Request.ServerBody.ValidateRef }}
		if err != nil {
		{{- if .Payload.Request.MultipartGenerated }}
			if multipartErr != nil {
				err = goa.MergeErrors(multipartErr, err)
			}
		{{- end }}
			return payload, err
		}
	{{- end }}
	{{- if .Payload.Request.MultipartGenerated }}
		if multipartErr != nil {
			return payload, multipartErr
		}
	{{- end }}
{{- end }}
{{- if not .MultipartRequestDecoder }}
	{{- template "partial_request_elements" .Payload.Request }}
	{{- if .Payload.Request.MustValidate }}
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
		return payload, goa.MissingFieldError("Authorization", "header")
	}
		{{- end }}
	payload.{{ .UsernameField }} = {{ if .UsernamePointer }}&{{ end }}user
	payload.{{ .PasswordField }} = {{ if .PasswordPointer }}&{{ end }}pass
{{- end }}{{ end }}
{{- range .HeaderSchemes }}
	{{- if not .CredRequired }}
	if payload.{{ .CredField }} != nil {
	{{- end }}
	if strings.Contains({{ if .CredPointer }}*{{ end }}payload.{{ .CredField }}, " ") {
		// Remove authorization scheme prefix (e.g. "Bearer")
		cred := strings.SplitN({{ if .CredPointer }}*{{ end }}payload.{{ .CredField }}, " ", 2)[1]
		payload.{{ .CredField }} = {{ if .CredPointer }}&{{ end }}cred
	}
	{{- if not .CredRequired }}
	}
	{{- end }}
{{- end }}

	return payload, nil
	}
}
