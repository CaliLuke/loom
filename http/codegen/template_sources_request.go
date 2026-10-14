package codegen

var (
	requestEncoderSource = joinHTTPTemplateSource(`{{ printf "%s returns an encoder for requests sent to the %s %s server." .RequestEncoder .ServiceName .Method.Name | comment }}
func {{ .RequestEncoder }}(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) {{ if .Payload.Request.FinalizeEncoding }}(err error){{ else }}error{{ end }} {
		{{- if .Payload.Request.FinalizeEncoding }}
		defer func() {
			err = loomhttp.FinalizeRequestEncoding(req, "{{ .ServiceName }}", "{{ .Method.Name }}", err)
		}()
		{{- end }}
		{{- if .Method.SkipRequestBodyEncodeDecode }}
		data, ok := v.(*{{ .ServicePkgName }}.{{ .Method.RequestStruct }})
		if !ok {
			return loomhttp.ErrInvalidType("{{ .ServiceName }}", "{{ .Method.Name }}", "*{{ .ServicePkgName }}.{{ .Method.RequestStruct }}", v)
		}
		p := data.Payload
		{{- else }}
		p, ok := v.({{ .Payload.Ref }})
		if !ok {
			return loomhttp.ErrInvalidType("{{ .ServiceName }}", "{{ .Method.Name }}", "{{ .Payload.Ref }}", v)
		}
		{{- end }}
	{{- range .Payload.Request.Headers }}
			{{- if .FieldPointer }}
		if p{{ if .FieldName }}.{{ .FieldName }}{{ end }} != nil {
			{{- else }}
			{
			{{- end }}
			head := {{ if .IsTextUnmarshaler }}fmt.Sprint({{ end }}{{ if .FieldPointer }}*{{ end }}p{{ if .FieldName }}.{{ .FieldName }}{{ end }}{{ if .IsTextUnmarshaler }}){{ end }}
   {{- if .Bearer.Framed }}
   req.Header.Set({{ printf "%q" .HTTPName }}, "Bearer "+{{ if isAlias .FieldType }}string(head){{ else }}head{{ end }})
   {{- else }}
			{{- if eq .Type.Name "array" }}
			{{ renderHeaderConversion .Type .FieldType "values" "head" }}
			if err := loomhttp.SetHeaderValues(req.Header, {{ printf "%q" .HTTPName }}, values); err != nil {
				return err
			}
			{{- else if (and (isAlias .FieldType) (eq (underlyingType .FieldType).Name "string")) }}
			req.Header.Set({{ printf "%q" .HTTPName }}, string(head))
			{{- else if eq .Type.Name "string" }}
			req.Header.Set({{ printf "%q" .HTTPName }}, head)
			{{- else }}
			{{ template "partial_client_type_conversion" (typeConversionData .Type .FieldType "headStr" "head") }}
			req.Header.Set({{ printf "%q" .HTTPName }}, headStr)
			{{- end }}
			{{- end }}
		}
	{{- end }}
	{{- range .Payload.Request.Cookies }}
			{{- if .FieldPointer }}
		if p{{ if .FieldName }}.{{ .FieldName }}{{ end }} != nil {
			{{- else }}
			{
			{{- end }}
			v{{ if or (not (eq .Type.Name "string")) (and (isAlias .FieldType) (not .IsTextUnmarshaler)) }}raw{{ end }} := {{ if .IsTextUnmarshaler }}fmt.Sprint({{ end }}{{ if .FieldPointer }}*{{ end }}p{{ if .FieldName }}.{{ .FieldName }}{{ end }}{{ if .IsTextUnmarshaler }}){{ end }}
			{{- if or (not (eq .Type.Name "string")) (and (isAlias .FieldType) (not .IsTextUnmarshaler)) }}
			{{ template "partial_client_type_conversion" (typeConversionData .Type .FieldType "v" "vraw") }}
			{{- end }}
			if err := loomhttp.AddRequestCookie(req, {{ printf "%q" .HTTPName }}, v); err != nil {
				return loomhttp.ErrEncodingError("{{ $.ServiceName }}", "{{ $.Method.Name }}", err)
			}
		}
	{{- end }}
	{{- if or .Payload.Request.QueryParams }}
		values := req.URL.Query()
	{{- end }}
	{{- range .Payload.Request.QueryParams }}
		{{- if .MapQueryParams }}
		for key, value := range p{{ if .FieldName }}.{{ .FieldName }}{{ end }} {
			{{ template "partial_client_type_conversion" (typeConversionData .Type.KeyType.Type (aliasedType .FieldType).KeyType.Type "keyStr" "key") }}
			{{- if eq .Type.ElemType.Type.Name "array" }}
			for _, val := range value {
				{{ template "partial_client_type_conversion" (typeConversionData .Type.ElemType.Type.ElemType.Type (aliasedType (aliasedType .FieldType).ElemType.Type).ElemType.Type "valStr" "val") }}
				values.Add(keyStr, valStr)
			}
			{{- else }}
			{{ template "partial_client_type_conversion" (typeConversionData .Type.ElemType.Type (aliasedType .FieldType).ElemType.Type "valueStr" "value") }}
			values.Add(keyStr, valueStr)
			{{- end }}
    }
		{{- else if .StringSlice }}
			for _, value := range p{{ if .FieldName }}.{{ .FieldName }}{{ end }} {
				values.Add("{{ .HTTPName }}", value)
			}
		{{- else if .Slice }}
			for _, value := range p{{ if .FieldName }}.{{ .FieldName }}{{ end }} {
				{{ template "partial_client_type_conversion" (typeConversionData .Type.ElemType.Type (aliasedType .FieldType).ElemType.Type "valueStr" "value") }}
				values.Add("{{ .HTTPName }}", valueStr)
			}
		{{- else if .JSONField }}
		if _, err := loomhttp.EncodeJSONField(values, {{ printf "%q" .HTTPName }}, p{{ if .FieldName }}.{{ .FieldName }}{{ end }}); err != nil {
			return err
		}
		{{- else if .FieldName }}
			{{- if .FieldPointer }}
		if p.{{ .FieldName }} != nil {
			{{- end }}
		values.Add("{{ .HTTPName }}",
			{{- if eq .Type.Name "any" }} loom.JSONValueString(
			{{- else if or (eq .Type.Name "bytes") (and (isAlias .FieldType) (eq (underlyingType .FieldType).Name "string")) }} string(
			{{- else if or .IsTextUnmarshaler (not (eq .Type.Name "string")) }} fmt.Sprintf("%v",
			{{- end }}
			{{- if .FieldPointer }}*{{ end }}p.{{ .FieldName }}
			{{- if or .IsTextUnmarshaler (eq .Type.Name "bytes") (not (eq .Type.Name "string")) (and (isAlias .FieldType) (eq (underlyingType .FieldType).Name "string")) }})
			{{- end }})
			{{- if .FieldPointer }}
		}
			{{- end }}
		{{- else }}
			{{- if eq .Type.Name "string" }}
				values.Add("{{ .HTTPName }}", p)
			{{- else if (and (isAlias .Type) (eq (underlyingType .Type).Name "string")) }}
				values.Add("{{ .HTTPName }}", string(p))
			{{- else }}
				{{ template "partial_client_type_conversion" (typeConversionData .Type .FieldType "pStr" "p") }}
				values.Add("{{ .HTTPName }}", pStr)
			{{- end }}
		{{- end }}
	{{- end }}
	{{- if .Payload.Request.QueryParams }}
		req.URL.RawQuery = values.Encode()
	{{- end }}
	{{- if .MultipartRequestEncoder }}
		if err := encoder(req).Encode(p); err != nil {
			return loomhttp.ErrEncodingError("{{ .ServiceName }}", "{{ .Method.Name }}", err)
		}
	{{- else if or .Payload.Request.ClientBody .Payload.Request.JSONRPCID }}
		{{- if and .Method.IsJSONRPC (or .Payload.Request.OptionalBodyAttribute (not .Payload.Request.ClientBody)) }}
		body := &jsonrpc.Request{
			JSONRPC: "2.0",
			Method:  "{{ .Method.Name }}",
		}
		{{- if .Payload.Request.ClientBody }}
		if p.{{ .Payload.Request.PayloadAttr }}{{ if .Payload.Request.OptionalBodyNullable }}.Present(){{ else }} != nil{{ end }} {
			body.Params = {{ .Payload.Request.ClientBodyValue }}
		}
		{{- end }}
		{{- else }}
		{{- if .Payload.Request.OptionalBodyAttribute }}
		if p.{{ .Payload.Request.PayloadAttr }}{{ if .Payload.Request.OptionalBodyNullable }}.Present(){{ else }} != nil{{ end }} {
		{{- end }}
		{{- if .Method.IsJSONRPC }}
		b := {{ .Payload.Request.ClientBodyValue }}
		{{- else }}
		body := {{ .Payload.Request.ClientBodyValue }}
		{{- end }}
		{{- if .Method.IsJSONRPC }}
		body := &jsonrpc.Request{
			JSONRPC: "2.0",
			Method:  "{{ .Method.Name }}",
			Params:  b,
		}
		{{- end }}
		{{- end }}
		{{- if .Method.IsJSONRPC }}
		{{- if .Payload.Request.JSONRPCID }}
 if err := jsonrpc.SetClientRequestID(req, body, p.{{ .Payload.Request.JSONRPCID.FieldName }}, {{ .Payload.Request.JSONRPCID.ClientPresence }}); err != nil {
  return loomhttp.ErrEncodingError("{{ .ServiceName }}", "{{ .Method.Name }}", err)
 }
		{{- else }}
		// No ID field in payload - always send as a request with generated ID
		id, err := jsonrpc.NewRequestID()
		if err != nil {
			return loomhttp.ErrEncodingError("{{ .ServiceName }}", "{{ .Method.Name }}", err)
		}
		body.ID = id
		{{- end }}
		{{- end }}
		{{- if .Payload.Request.FormEncoded }}
		if err := loomhttp.SetFormRequest(req, &body); err != nil {
			return loomhttp.ErrEncodingError("{{ .ServiceName }}", "{{ .Method.Name }}", err)
		}
		{{- else }}
		{{- if .Payload.Request.ContentType }}
		req.Header.Set("Content-Type", {{ printf "%q" .Payload.Request.ContentType }})
		{{- end }}
		if err := encoder(req).Encode(&body); err != nil {
			return loomhttp.ErrEncodingError("{{ .ServiceName }}", "{{ .Method.Name }}", err)
		}
		{{- end }}
		{{- if and .Payload.Request.OptionalBodyAttribute (not .Method.IsJSONRPC) }}
		}
		{{- end }}
	{{- end }}
	{{- if .BasicScheme }}{{ with .BasicScheme }}
		{{- if not .AlwaysSend }}
		if p.{{ .UsernameField }} != nil || p.{{ .PasswordField }} != nil {
		{{- end }}
		{{- if or .UsernamePointer .PasswordPointer }}
		{
			var user, pass string
			{{- if .UsernamePointer }}
			if p.{{ .UsernameField }} != nil {
			{{- end }}
			user = {{ if isAlias .UsernameType }}string({{ end }}{{ if .UsernamePointer }}*{{ end }}p.{{ .UsernameField }}{{ if isAlias .UsernameType }}){{ end }}
			{{- if .UsernamePointer }}
			}
			{{- end }}
			{{- if .PasswordPointer }}
			if p.{{ .PasswordField }} != nil {
			{{- end }}
			pass = {{ if isAlias .PasswordType }}string({{ end }}{{ if .PasswordPointer }}*{{ end }}p.{{ .PasswordField }}{{ if isAlias .PasswordType }}){{ end }}
			{{- if .PasswordPointer }}
			}
			{{- end }}
			req.SetBasicAuth(user, pass)
		}
		{{- else }}
		req.SetBasicAuth({{ if isAlias .UsernameType }}string({{ end }}p.{{ .UsernameField }}{{ if isAlias .UsernameType }}){{ end }}, {{ if isAlias .PasswordType }}string({{ end }}p.{{ .PasswordField }}{{ if isAlias .PasswordType }}){{ end }})
		{{- end }}
		{{- if not .AlwaysSend }}
		}
		{{- end }}
	{{- end }}{{ end }}
		return nil
	}
}
`,
		templateSource{name: "client_type_conversion", source: `{{- .VarName }} := {{ transportScalarValue .Type .FieldType .Target }}`},
	)
)
