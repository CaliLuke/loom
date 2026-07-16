package codegen

var (
	requestEncoderSource = joinHTTPTemplateSource(`{{ printf "%s returns an encoder for requests sent to the %s %s server." .RequestEncoder .ServiceName .Method.Name | comment }}
func {{ .RequestEncoder }}(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		{{- if .Method.SkipRequestBodyEncodeDecode }}
		data, ok := v.(*{{ requestStructPkg .Method .ServicePkgName }}.{{ .Method.RequestStruct }})
		if !ok {
			return loomhttp.ErrInvalidType("{{ .ServiceName }}", "{{ .Method.Name }}", "*{{ requestStructPkg .Method .ServicePkgName }}.{{ .Method.RequestStruct }}", v)
		}
		p := data.Payload
		{{- else }}
		p, ok := v.({{ .Payload.Ref }})
		if !ok {
			return loomhttp.ErrInvalidType("{{ .ServiceName }}", "{{ .Method.Name }}", "{{ .Payload.Ref }}", v)
		}
		{{- end }}
	{{- range .Payload.Request.Headers }}
		{{- if .FieldName }}
			{{- if .FieldPointer }}
		if p.{{ .FieldName }} != nil {
			{{- else }}
			{
			{{- end }}
			head := {{ if .FieldPointer }}*{{ end }}p.{{ .FieldName }}
			{{- if (and (eq .HTTPName "Authorization") (isBearer $.HeaderSchemes)) }}
		if !strings.Contains(head, " ") {
			req.Header.Set({{ printf "%q" .HTTPName }}, "Bearer "+head)
		} else {
			{{- end }}
			{{- if eq .Type.Name "array" }}
			for _, val := range head {
				{{- if eq .Type.ElemType.Type.Name "string" }}
				req.Header.Add({{ printf "%q" .HTTPName }}, val)
				{{- else if (and (isAlias .Type.ElemType.Type) (eq (underlyingType .Type.ElemType.Type).Name "string")) }}
				req.Header.Set({{ printf "%q" .HTTPName }}, string(val))
				{{- else }}
				{{ template "partial_client_type_conversion" (typeConversionData .Type.ElemType.Type (aliasedType .FieldType).ElemType.Type "valStr" "val") }}
				req.Header.Add({{ printf "%q" .HTTPName }}, valStr)
				{{- end }}
			}
			{{- else if (and (isAlias .FieldType) (eq (underlyingType .FieldType).Name "string")) }}
			req.Header.Set({{ printf "%q" .HTTPName }}, string(head))
			{{- else if eq .Type.Name "string" }}
			req.Header.Set({{ printf "%q" .HTTPName }}, head)
			{{- else }}
			{{ template "partial_client_type_conversion" (typeConversionData .Type .FieldType "headStr" "head") }}
			req.Header.Set({{ printf "%q" .HTTPName }}, headStr)
			{{- end }}
			{{- if (and (eq .HTTPName "Authorization") (isBearer $.HeaderSchemes)) }}
		}
			{{- end }}
		}
		{{- end }}
	{{- end }}
	{{- range .Payload.Request.Cookies }}
		{{- if .FieldName }}
			{{- if .FieldPointer }}
		if p.{{ .FieldName }} != nil {
			{{- else }}
			{
			{{- end }}
			v{{ if not (eq .Type.Name "string") }}raw{{ end }} := {{ if .FieldPointer }}*{{ end }}p.{{ .FieldName }}
			{{- if not (eq .Type.Name "string" ) }}
			{{ template "partial_client_type_conversion" (typeConversionData .Type .FieldType "v" "vraw") }}
			{{- end }}
			req.AddCookie(&http.Cookie{
				Name: {{ printf "%q" .HTTPName }},
				Value: v,
				{{- if .MaxAge }}
				MaxAge: {{ .MaxAge }},
				{{- end }}
				{{- if .Path }}
				Path: {{ .Path }},
				{{- end }}
				{{- if .Domain }}
				Domain: {{ .Domain }},
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
		}
		{{- end }}
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
		{{- else if .Map }}
			{{- template "partial_client_map_conversion" (mapConversionData .Type .FieldType .HTTPName "p" .FieldName true) }}
		{{- else if .FieldName }}
			{{- if .FieldPointer }}
		if p.{{ .FieldName }} != nil {
			{{- end }}
		values.Add("{{ .HTTPName }}",
			{{- if or (eq .Type.Name "bytes") (and (isAlias .FieldType) (eq (underlyingType .FieldType).Name "string")) }} string(
			{{- else if not (eq .Type.Name "string") }} fmt.Sprintf("%v",
			{{- end }}
			{{- if .FieldPointer }}*{{ end }}p.{{ .FieldName }}
			{{- if or (eq .Type.Name "bytes") (not (eq .Type.Name "string")) (and (isAlias .FieldType) (eq (underlyingType .FieldType).Name "string")) }})
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
	{{- else if .Payload.Request.ClientBody }}
		{{- if .Payload.Request.ClientBody.Init }}
		{{- if .Method.IsJSONRPC }}
		b := {{ .Payload.Request.ClientBody.Init.Name }}({{ range $i, $arg := .Payload.Request.ClientBody.Init.ClientArgs }}{{ if $i }}, {{ end }}{{ if $arg.FieldPointer }}&{{ end }}{{ $arg.VarName }}{{ end }})
		{{- else }}
		body := {{ .Payload.Request.ClientBody.Init.Name }}({{ range $i, $arg := .Payload.Request.ClientBody.Init.ClientArgs }}{{ if $i }}, {{ end }}{{ if $arg.FieldPointer }}&{{ end }}{{ $arg.VarName }}{{ end }})
		{{- end }}
		{{- else }}
		{{- if .Method.IsJSONRPC }}
		b := p{{ if .Payload.Request.PayloadAttr }}.{{ .Payload.Request.PayloadAttr }}{{ end }}
		{{- else }}
		body := p{{ if .Payload.Request.PayloadAttr }}.{{ .Payload.Request.PayloadAttr }}{{ end }}
		{{- end }}
		{{- end }}
		{{- if .Method.IsJSONRPC }}
		body := &jsonrpc.Request{
			JSONRPC: "2.0",
			Method:  "{{ .Method.Name }}",
			Params:  b,
		}
		{{- if .Payload.IDAttribute }}
			{{- if .Payload.IDAttributeRequired }}
		if p.{{ .Payload.IDAttribute }} != "" {
			body.ID = p.{{ .Payload.IDAttribute }}
		}
		// If ID is empty, this is a notification - no ID field
			{{- else }}
		if p.{{ .Payload.IDAttribute }} != nil && *p.{{ .Payload.IDAttribute }} != "" {
			body.ID = p.{{ .Payload.IDAttribute }}
		}
		// If ID is nil or empty, this is a notification - no ID field
			{{- end }}
		{{- else }}
		// No ID field in payload - always send as a request with generated ID
		id := uuid.New().String()
		body.ID = id
		{{- end }}
		{{- end }}
		{{- if .Payload.Request.FormEncoded }}
		if err := loomhttp.SetFormRequest(req, &body); err != nil {
			return loomhttp.ErrEncodingError("{{ .ServiceName }}", "{{ .Method.Name }}", err)
		}
		{{- else }}
		if err := encoder(req).Encode(&body); err != nil {
			return loomhttp.ErrEncodingError("{{ .ServiceName }}", "{{ .Method.Name }}", err)
		}
		{{- end }}
	{{- end }}
	{{- if .BasicScheme }}{{ with .BasicScheme }}
		{{- if not .UsernameRequired }}
		if p.{{ .UsernameField }} != nil {
		{{- end }}
		{{- if not .PasswordRequired }}
		if p.{{ .PasswordField }} != nil {
		{{- end }}
		req.SetBasicAuth({{ if .UsernamePointer }}*{{ end }}p.{{ .UsernameField }}, {{ if .PasswordPointer }}*{{ end }}p.{{ .PasswordField }})
		{{- if not .UsernameRequired }}
		}
		{{- end }}
		{{- if not .PasswordRequired }}
		}
		{{- end }}
	{{- end }}{{ end }}
		return nil
	}
}
`,
		templateSource{name: "client_type_conversion", source: `  {{- if eq .Type.Name "boolean" -}}
    {{ .VarName }} := strconv.FormatBool({{ if .IsAliased }}bool({{ end }}{{ .Target }}{{ if .IsAliased }}){{ end }})
  {{- else if eq .Type.Name "int" -}}
    {{ .VarName }} := strconv.Itoa({{ if .IsAliased }}int({{ end }}{{ .Target }}{{ if .IsAliased }}){{ end }})
  {{- else if eq .Type.Name "int32" -}}
    {{ .VarName }} := strconv.FormatInt(int64({{ .Target }}), 10)
  {{- else if eq .Type.Name "int64" -}}
    {{ .VarName }} := strconv.FormatInt({{ if .IsAliased }}int64({{ end }}{{ .Target }}{{ if .IsAliased }}){{ end }}, 10)
  {{- else if eq .Type.Name "uint" -}}
    {{ .VarName }} := strconv.FormatUint(uint64({{ .Target }}), 10)
  {{- else if eq .Type.Name "uint32" -}}
    {{ .VarName }} := strconv.FormatUint(uint64({{ .Target }}), 10)
  {{- else if eq .Type.Name "uint64" -}}
    {{ .VarName }} := strconv.FormatUint({{ if .IsAliased }}uint64({{ end }}{{ .Target }}{{ if .IsAliased }}){{ end }}, 10)
  {{- else if eq .Type.Name "float32" -}}
    {{ .VarName }} := strconv.FormatFloat(float64({{ .Target }}), 'f', -1, 32)
  {{- else if eq .Type.Name "float64" -}}
    {{ .VarName }} := strconv.FormatFloat({{ if .IsAliased }}float64({{ end }}{{ .Target }}{{ if .IsAliased }}){{ end }}, 'f', -1, 64)
	{{- else if eq .Type.Name "string" -}}
    {{ .VarName }} := {{ if .IsAliased }}string({{ end }}{{ .Target }}{{ if .IsAliased }}){{ end }}
  {{- else if eq .Type.Name "bytes" -}}
    {{ .VarName }} := string({{ .Target }})
  {{- else if eq .Type.Name "any" -}}
    {{ .VarName }} := fmt.Sprintf("%v", {{ .Target }})
  {{- else }}
    // unsupported type {{ .Type.Name }} for field {{ .FieldName }}
  {{- end }}`},
		templateSource{name: "client_map_conversion", source: `for k{{ if not (eq .Type.KeyType.Type.Name "string") }}Raw{{ end }}, value := range {{ .SourceVar }}{{ if .SourceField }}.{{ .SourceField }}{{ end }} {
		{{- if not (eq .Type.KeyType.Type.Name "string") }}
			{{ template "partial_client_type_conversion" (typeConversionData .Type.KeyType.Type .FieldType.KeyType.Type "k" "kRaw") }}
		{{- end }}
		key {{ if .NewVar }}:={{ else }}={{ end }} fmt.Sprintf("{{ .VarName }}[%s]", {{ if not .NewVar }}key, {{ end }}k)
		{{- if eq .Type.ElemType.Type.Name "string" }}
			values.Add(key, {{ if (isAlias .FieldType.ElemType.Type) }}string({{ end }}value{{ if (isAlias .FieldType.ElemType.Type) }}){{ end }})
		{{- else if eq .Type.ElemType.Type.Name "map" }}
			{{- template "partial_client_map_conversion" (mapConversionData .Type.ElemType.Type .FieldType.ElemType.Type "%s" "value" "" false) }}
		{{- else if eq .Type.ElemType.Type.Name "array" }}
			{{- if and (eq .Type.ElemType.Type.ElemType.Type.Name "string") (not (isAlias .FieldType.ElemType.Type.ElemType.Type)) }}
				values[key] = value
			{{- else }}
				for _, val := range value {
					{{ template "partial_client_type_conversion" (typeConversionData .Type.ElemType.Type.ElemType.Type (aliasedType .FieldType.ElemType.Type).ElemType.Type "valStr" "val") }}
					values.Add(key, valStr)
				}
			{{- end }}
		{{- else }}
			{{ template "partial_client_type_conversion" (typeConversionData .Type.ElemType.Type .FieldType.ElemType.Type "valueStr" "value") }}
			values.Add(key, valueStr)
		{{- end }}
	}`},
	)
)
