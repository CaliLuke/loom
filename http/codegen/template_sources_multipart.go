package codegen

var (
	multipartRequestDecoderSource = joinHTTPTemplateSource(`{{ printf "%s returns a decoder to decode the multipart request for the %q service %q endpoint." .InitName .ServiceName .MethodName | comment }}
func {{ .InitName }}(mux loomhttp.Muxer, {{ .VarName }} {{ .FuncName }}) func(r *http.Request) loomhttp.Decoder {
	return func(r *http.Request) loomhttp.Decoder {
		return loomhttp.EncodingFunc(func(v any) error {
			mr, merr := r.MultipartReader()
			if merr != nil {
				return merr
			}
			p := v.(*{{ .Payload.Ref }})
			if err := {{ .VarName }}(mr, p); err != nil {
				return err
			}
			{{- if .Payload.Request.DecodePlan.ElementsMustValidate }}
			var err error
			{{- end }}
			{{- template "partial_request_elements" .Payload.Request }}
			{{- if .Payload.Request.DecodePlan.ElementsMustValidate }}
			if err != nil {
				return err
			}
			{{- end }}
			{{- if .Payload.Request.PayloadInit }}
				{{- range .Payload.Request.PayloadInit.ServerArgs }}
					{{- if .FieldName }}
			(*p).{{ .FieldName }} = {{ if and (not .Pointer) .FieldPointer }}&{{ end }}{{ .VarName }}
					{{- end }}
				{{- end }}
			{{- end }}
			return nil
		})
	}
}
`, requestDecoderPartials...)
)
