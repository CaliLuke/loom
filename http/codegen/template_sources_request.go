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
		{{- if and .Payload.Request.ClientPayloadAbsent (not .Method.IsJSONRPC) }}
		if {{ .Payload.Request.ClientPayloadAbsent }} {
			{{- if .Payload.Request.ClientPayloadOptional }}
			return nil
			{{- else }}
			return loom.MissingPayloadError()
			{{- end }}
		}
		{{- end }}
	{{- range .Payload.Request.Headers }}
{{- if .Styled }}
{{ renderStyledEncode .Element "p" "header" }}
{{- else }}
			{{- if .FieldPointer }}
		if p{{ if .FieldName }}.{{ .FieldName }}{{ end }} != nil {
			{{- else }}
			{
			{{- end }}
			head := {{ if .IsTextUnmarshaler }}{{ transportScalarValue .Type .FieldType (transportFieldReference .AttributeData "p") true }}{{ else }}{{ transportFieldReference .AttributeData "p" }}{{ end }}
   {{- if .Bearer.Framed }}
   req.Header.Set({{ printf "%q" .HTTPName }}, "Bearer "+{{ if isAlias .FieldType }}string(head){{ else }}head{{ end }})
   {{- else }}
			{{- if eq .Type.Name "array" }}
			{{ renderHeaderConversion .Type .FieldType "values" "head" false }}
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
	{{- end }}
	{{- range .Payload.Request.Cookies }}
{{- if .Styled }}
{{ renderStyledEncode .Element "p" "cookie" }}
{{- else }}
			{{- if .FieldPointer }}
		if p{{ if .FieldName }}.{{ .FieldName }}{{ end }} != nil {
			{{- else }}
			{
			{{- end }}
			v{{ if or (not (eq .Type.Name "string")) (and (isAlias .FieldType) (not .IsTextUnmarshaler)) }}raw{{ end }} := {{ if .IsTextUnmarshaler }}{{ transportScalarValue .Type .FieldType (transportFieldReference .AttributeData "p") true }}{{ else }}{{ transportFieldReference .AttributeData "p" }}{{ end }}
			{{- if or (not (eq .Type.Name "string")) (and (isAlias .FieldType) (not .IsTextUnmarshaler)) }}
			{{ template "partial_client_type_conversion" (typeConversionData .Type .FieldType "v" "vraw") }}
			{{- end }}
			if err := loomhttp.AddRequestCookie(req, {{ printf "%q" .HTTPName }}, v); err != nil {
				return loomhttp.ErrEncodingError("{{ $.ServiceName }}", "{{ $.Method.Name }}", err)
			}
		}
	{{- end }}
	{{- end }}
	{{- if or .Payload.Request.QueryParams }}
		values := req.URL.Query()
{{- if hasStyledQuery .Payload.Request.QueryParams }}
var parameterBuilder loomhttp.RawPairsBuilder
{{- end }}
	{{- end }}
	{{- range .Payload.Request.QueryParams }}
{{- if .Styled }}
{{ renderStyledEncode .Element "p" "query" }}
{{- else if .MapQueryParams }}
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
		if _, err := loomhttp.EncodeJSONField(values, {{ printf "%q" .HTTPName }}, p{{ if .FieldName }}.{{ .FieldName }}{{ end }}, loomhttp.FieldValue); err != nil {
			return err
		}
		{{- else if .FieldName }}
			{{- if .FieldPointer }}
		if p.{{ .FieldName }} != nil {
			{{- end }}
		values.Add("{{ .HTTPName }}",
            {{- if .IsTextUnmarshaler }} {{ transportScalarValue .Type .FieldType (transportFieldReference .AttributeData "p") true }}
            {{- else }}
			{{- if eq .Type.Name "any" }} loom.JSONValueString(
			{{- else if or (eq .Type.Name "bytes") (and (isAlias .FieldType) (eq (underlyingType .FieldType).Name "string")) }} string(
			{{- else if not (eq .Type.Name "string") }} fmt.Sprintf("%v",
			{{- end }}
			{{- if .FieldPointer }}*{{ end }}p.{{ .FieldName }}
			{{- if or (eq .Type.Name "bytes") (not (eq .Type.Name "string")) (and (isAlias .FieldType) (eq (underlyingType .FieldType).Name "string")) }})
			{{- end }}{{- end }})
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
		req.URL.RawQuery = {{ if hasStyledQuery .Payload.Request.QueryParams }}parameterBuilder.Encode(values){{ else }}values.Encode(){{ end }}
	{{- end }}
	{{- if and .Payload.Request.ClientBodyAbsent (not .Payload.Request.OptionalBodyAttribute) }}
		if {{ .Payload.Request.ClientBodyAbsent }} {
			return loom.MissingPayloadError()
		}
	{{- end }}
	{{- if .MultipartRequestEncoder }}
		if err := encoder(req).Encode(p); err != nil {
			return loomhttp.ErrEncodingError("{{ .ServiceName }}", "{{ .Method.Name }}", err)
		}
	{{- else if or .Payload.Request.ClientBody .Payload.Request.JSONRPCID .Method.IsJSONRPC }}
		{{- if and .Method.IsJSONRPC (or .Payload.Request.OptionalBodyAttribute (not .Payload.Request.ClientBody)) }}
		body := &jsonrpc.Request{
			JSONRPC: "2.0",
			Method:  "{{ .Method.Name }}",
		}
		{{- if .Payload.Request.ClientBody }}
		{{- if .Payload.Request.ClientBodyAbsent }}
		if !({{ .Payload.Request.ClientBodyAbsent }}) {
		{{- end }}
			{{ if and .Payload.Request.ClientBody.Init .Payload.Request.ClientBody.Init.ClientErrorAware }}convertedBody, conversionError := {{ .Payload.Request.ClientBodyValue }}
			if conversionError != nil {
				return loomhttp.ErrEncodingError("{{ .ServiceName }}", "{{ .Method.Name }}", conversionError)
			}
			body.Params = convertedBody{{ else }}body.Params = {{ .Payload.Request.ClientBodyValue }}{{ end }}
		{{- if .Payload.Request.ClientBodyAbsent }}
		}
		{{- end }}
		{{- end }}
		{{- else }}
		{{- if and .Payload.Request.OptionalBodyAttribute .Payload.Request.ClientBodyAbsent }}
		if !({{ .Payload.Request.ClientBodyAbsent }}) {
		{{- end }}
		{{- if .Method.IsJSONRPC }}
		b{{ if and .Payload.Request.ClientBody.Init .Payload.Request.ClientBody.Init.ClientErrorAware }}, conversionError{{ end }} := {{ .Payload.Request.ClientBodyValue }}
		{{- if and .Payload.Request.ClientBody.Init .Payload.Request.ClientBody.Init.ClientErrorAware }}
		if conversionError != nil {
			return loomhttp.ErrEncodingError("{{ .ServiceName }}", "{{ .Method.Name }}", conversionError)
		}
		{{- end }}
		{{- else }}
		{{- if clientBodyValueUsed .Payload.Request }}
		body{{ if and .Payload.Request.ClientBody.Init .Payload.Request.ClientBody.Init.ClientErrorAware }}, conversionError{{ end }} := {{ .Payload.Request.ClientBodyValue }}
		{{- else if and .Payload.Request.ClientBody.Init .Payload.Request.ClientBody.Init.ClientErrorAware }}
		_, conversionError := {{ .Payload.Request.ClientBodyValue }}
		{{- else if .Payload.Request.ClientBody.Init }}
		{{ .Payload.Request.ClientBodyValue }}
		{{- end }}
		{{- if and .Payload.Request.ClientBody.Init .Payload.Request.ClientBody.Init.ClientErrorAware }}
		if conversionError != nil {
			return loomhttp.ErrEncodingError("{{ .ServiceName }}", "{{ .Method.Name }}", conversionError)
		}
		{{- end }}
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
        {{- if and .Payload.Request.ClientBody .Payload.Request.ClientBody.Media }}
        {
            mediaIndex := -1
            var mediaValue any
            switch string(body.Kind()) {
            {{- range $i, $variant := .Payload.Request.ClientBody.Media.Variants }}
            case {{ printf "%q" .Tag }}:
                mediaIndex, mediaValue = {{ $i }}, body.{{ .Field }}
{{- if .Multipart }}
mediaContext := req.Context()
defer func() {
 *req = *req.WithContext(mediaContext)
}()
*req = *req.WithContext({{ renderMediaMultipartEncode . "req.Context()" (printf "body.%s" .Field) }})
{{- else if mediaFormCodec . }}
mediaContext := req.Context()
defer func() {
 *req = *req.WithContext(mediaContext)
}()
*req = *req.WithContext({{ renderMediaFormEncode . "req.Context()" (printf "body.%s" .Field) (not $.Payload.Request.MustHaveBody) }})
{{- end }}
            {{- end }}
            }
            if err := loomhttp.EncodeMediaRequest(req, mediaIndex, {{ .Payload.Request.ClientBody.Media.Contracts }}, mediaValue, nil, encoder); err != nil {
                return loomhttp.ErrEncodingError("{{ .ServiceName }}", "{{ .Method.Name }}", err)
            }
        }
        {{- else if .Payload.Request.MultipartGenerated }}
{{ renderMultipartRequestEncode .Payload.Request }}
        {{- else if .Payload.Request.FormEncoded }}
{{- if formRootRequest .Payload.Request }}
{{ renderFormRootRequestEncode .Payload.Request }}
{{- else if or .Payload.Request.StyledFormFields .Payload.Request.OrdinaryFormFields }}
{{ renderStyledFormEncode .Payload.Request }}
{{- else }}
		if err := loomhttp.SetFormRequest(req, &body); err != nil {
			return loomhttp.ErrEncodingError("{{ .ServiceName }}", "{{ .Method.Name }}", err)
		}
{{- end }}
		{{- else }}
		{{- if .Payload.Request.ContentType }}
		req.Header.Set("Content-Type", {{ printf "%q" .Payload.Request.ContentType }})
		{{- end }}
		if err := encoder(req).Encode(&body); err != nil {
			return loomhttp.ErrEncodingError("{{ .ServiceName }}", "{{ .Method.Name }}", err)
		}
		{{- end }}
		{{- if and .Payload.Request.OptionalBodyAttribute .Payload.Request.ClientBodyAbsent (not .Method.IsJSONRPC) }}
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
		templateSource{name: "client_type_conversion", source: `{{- .VarName }} := {{ transportScalarValue .Type .FieldType .Target false }}`},
	)
)
