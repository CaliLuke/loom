package codegen

// requestDecoderElementPartials holds the request element decoder template.
// It is split from the conversion helpers so both files stay below the
// generator file-size warning threshold.
var requestDecoderElementPartials = []templateSource{
	{name: "request_elements", source: `{{- if .DecodePlan.HasElements }}
{{- if .ServerBody }}{{/* we want a newline only if there was code before */}}
{{ end }}
		var (
		{{- range .PathParams }}
			{{ .VarName }} {{ .TypeRef }}
		{{- end }}
		{{- range .QueryParams }}
			{{ .VarName }} {{ .TypeRef }}
		{{- end }}
		{{- range .Headers }}
			{{ .VarName }} {{ .TypeRef }}
		{{- end }}
		{{- range .Cookies }}
			{{ .VarName }} {{ .TypeRef }}
		{{- end }}
		{{- if .DecodePlan.HasDecodedPathParams }}

			{{ .DecodePlan.PathValuesVar }} = mux.Vars(r)
		{{- end }}
		)

	{{- $pathValues := .DecodePlan.PathValuesVar }}
	{{- range .PathParams }}
		{{- if .IsTextUnmarshaler }}
			{
				{{ .Locals.Raw }} := {{ if eq .Type.Name "array" }}mux.RawVars(r){{ else }}{{ $pathValues }}{{ end }}["{{ .HTTPName }}"]
				{{- template "partial_path_conversion" . }}
				{{- if .Validate }}
				{{ .Validate }}
				{{- end }}
			}

	{{- else if and (or (eq .Type.Name "string") (eq .Type.Name "any")) }}
		{{ .VarName }} = {{ if eq .Type.Name "any" }}loom.JSONValueFromString({{ end }}{{ $pathValues }}["{{ .HTTPName }}"]{{ if eq .Type.Name "any" }}){{ end }}

		{{- else }}{{/* not string and not any */}}
			{
				{{ .Locals.Raw }} := {{ if eq .Type.Name "array" }}mux.RawVars(r){{ else }}{{ $pathValues }}{{ end }}["{{ .HTTPName }}"]
				{{- template "partial_path_conversion" . }}
			}

		{{- end }}
			{{- if and .Validate (not .IsTextUnmarshaler) }}
			{{ .Validate }}
			{{- end }}
	{{- end }}

{{- if .QueryParams }}
{{ .DecodePlan.QueryValuesVar }}, {{ .DecodePlan.QueryErrorVar }} := url.ParseQuery(r.URL.RawQuery)
if {{ .DecodePlan.QueryErrorVar }} != nil {
	{{ requestErrorReturn "loom.DecodePayloadError(\"invalid query string\")" }}
}
{{- end }}
{{- $qpVar := .DecodePlan.QueryValuesVar }}
{{- range .QueryParams }}
 {{- $scalar := or .IsTextUnmarshaler (and (not .Slice) (not .Map) (not .MapQueryParams) (not .JSONField)) }}
 {{- if $scalar }}
 {
  {{ .Locals.Raw }} := {{$qpVar}}.Get("{{ .HTTPName }}")
  {{ .Locals.Present }} := {{$qpVar}}.Has("{{ .HTTPName }}")
  {{- template "partial_scalar_element" (scalarElementData .Element "query string") }}
 }
	{{- else if .StringSlice }}
		{{ .VarName }} = {{$qpVar}}["{{ .HTTPName }}"]
		{{- if .Required }}
		if {{ .VarName }} == nil {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "query string"))
		}
		{{- else if .DefaultValue }}
		if {{ .VarName }} == nil {
			{{ .VarName }} = []string{
                {{- range $i, $v := .DefaultValue }}
                    {{- if $i }}{{ print ", " }}{{ end }}
                    {{- printf "%q" $v -}}
                {{- end -}} }
		}
		{{- end }}

	{{- else if .Slice }}
	{
		{{ .Locals.Raw }} := {{$qpVar}}["{{ .HTTPName }}"]
		{{- if .Required }}
		if {{ .Locals.Raw }} == nil {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "query string"))
		}
		{{- else if .DefaultValue }}
		if {{ .Locals.Raw }} == nil {
			{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
		}
		{{- end }}

		{{- if .DefaultValue }}else {
		{{- else if not .Required }}
		if {{ .Locals.Raw }} != nil {
		{{- end }}
		{{- template "partial_element_slice_conversion" . }}
		{{- if or .DefaultValue (not .Required) }}
		}
		{{- end }}
	}

	{{- else if .JSONField }}
	{
		var decoded {{ .JSONDecodeType }}
		present, decodeErr := loomhttp.DecodeJSONField({{$qpVar}}, {{ printf "%q" .HTTPName }}, &decoded)
		if decodeErr != nil {
			{{ requestErrorReturn "loom.DecodePayloadError(loomhttp.SafeDecodePayloadMessage(decodeErr))" }}
		}
		if present {
			{{ .JSONDecodeCode }}
		}
		{{- if .Required }}
		if !present {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "query string"))
		}
		{{- else if .DefaultValue }}
		if !present {
			{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
		}
		{{- end }}
	}

	{{- else if .MapQueryParams }}
	{
		{{ .Locals.Raw }} := {{$qpVar}}
		{{- if .Required }}
		if len({{ .Locals.Raw }}) == 0 {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "query string"))
		}
		{{- else if .DefaultValue }}
		if len({{ .Locals.Raw }}) == 0 {
			{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
		}
		{{- end }}

		{{- if .DefaultValue }}else {
		{{- else if not .Required }}
		if len({{ .Locals.Raw }}) != 0 {
		{{- end }}
		if {{ .VarName }} == nil {
			{{ .VarName }} = make({{ goTypeRef .Type }})
		}
		for keyRaw, valRaw := range {{ .Locals.Raw }} {
			var key {{ goTypeRef .Type.KeyType.Type }}
			var keyErr error
			{{- if eq .Type.KeyType.Type.Name "string" }}
			key = keyRaw
			{{- else }}
				{{- if eq .Type.KeyType.Type.Name "int" }}
			v, err2 := strconv.ParseInt(keyRaw, 10, strconv.IntSize)
			if err2 != nil {
				keyErr = loom.InvalidFieldTypeError("query", keyRaw, "integer")
				err = loom.MergeErrors(err, keyErr)
			}
			key = {{ goTypeRef .Type.KeyType.Type }}(v)
				{{- else if eq .Type.KeyType.Type.Name "int32" }}
			v, err2 := strconv.ParseInt(keyRaw, 10, 32)
			if err2 != nil {
				keyErr = loom.InvalidFieldTypeError("query", keyRaw, "integer")
				err = loom.MergeErrors(err, keyErr)
			}
			key = {{ goTypeRef .Type.KeyType.Type }}(v)
				{{- else if eq .Type.KeyType.Type.Name "int64" }}
			v, err2 := strconv.ParseInt(keyRaw, 10, 64)
			if err2 != nil {
				keyErr = loom.InvalidFieldTypeError("query", keyRaw, "integer")
				err = loom.MergeErrors(err, keyErr)
			}
			key = {{ goTypeRef .Type.KeyType.Type }}(v)
				{{- else if eq .Type.KeyType.Type.Name "uint" }}
			v, err2 := strconv.ParseUint(keyRaw, 10, strconv.IntSize)
			if err2 != nil {
				keyErr = loom.InvalidFieldTypeError("query", keyRaw, "unsigned integer")
				err = loom.MergeErrors(err, keyErr)
			}
			key = {{ goTypeRef .Type.KeyType.Type }}(v)
				{{- else if eq .Type.KeyType.Type.Name "uint32" }}
			v, err2 := strconv.ParseUint(keyRaw, 10, 32)
			if err2 != nil {
				keyErr = loom.InvalidFieldTypeError("query", keyRaw, "unsigned integer")
				err = loom.MergeErrors(err, keyErr)
			}
			key = {{ goTypeRef .Type.KeyType.Type }}(v)
				{{- else if eq .Type.KeyType.Type.Name "uint64" }}
			v, err2 := strconv.ParseUint(keyRaw, 10, 64)
			if err2 != nil {
				keyErr = loom.InvalidFieldTypeError("query", keyRaw, "unsigned integer")
				err = loom.MergeErrors(err, keyErr)
			}
			key = {{ goTypeRef .Type.KeyType.Type }}(v)
				{{- else if eq .Type.KeyType.Type.Name "float32" }}
			v, err2 := strconv.ParseFloat(keyRaw, 32)
			if err2 != nil {
				keyErr = loom.InvalidFieldTypeError("query", keyRaw, "float")
				err = loom.MergeErrors(err, keyErr)
			}
			key = {{ goTypeRef .Type.KeyType.Type }}(v)
				{{- else if eq .Type.KeyType.Type.Name "float64" }}
			v, err2 := strconv.ParseFloat(keyRaw, 64)
			if err2 != nil {
				keyErr = loom.InvalidFieldTypeError("query", keyRaw, "float")
				err = loom.MergeErrors(err, keyErr)
			}
			key = {{ goTypeRef .Type.KeyType.Type }}(v)
				{{- else if eq .Type.KeyType.Type.Name "boolean" }}
			v, err2 := strconv.ParseBool(keyRaw)
			if err2 != nil {
				keyErr = loom.InvalidFieldTypeError("query", keyRaw, "boolean")
				err = loom.MergeErrors(err, keyErr)
			}
			key = {{ goTypeRef .Type.KeyType.Type }}(v)
				{{- else }}
			keyErr = loom.InvalidFieldTypeError("query", keyRaw, {{ printf "%q" .Type.KeyType.Type.Name }})
			err = loom.MergeErrors(err, keyErr)
				{{- end }}
			{{- end }}
			if keyErr != nil {
				continue
			}
			{{- if eq .Type.ElemType.Type.Name "string" }}
				{{ .VarName }}[key] = valRaw[0]
			{{- else if eq .Type.ElemType.Type.Name "array" }}
				{{- if eq .Type.ElemType.Type.ElemType.Type.Name "string" }}
					{{ .VarName }}[key] = valRaw
				{{- else }}
					var val {{ goTypeRef .Type.ElemType.Type }}
					{
						{{- template "partial_element_slice_conversion" (conversionData "val" "valRaw" "query" .Type.ElemType.Type) }}
					}
					{{ .VarName }}[key] = val
				{{- end }}
			{{- else if eq .Type.ElemType.Type.Name "map" }}
				{{- template "partial_query_map_conversion" (mapQueryDecodeData .Type.ElemType.Type (printf "%s[key]" .VarName) 1) }}
			{{- else }}
				var val{{ .Loop }} {{ goTypeRef .Type.ElemType.Type }}
				{
					val{{ .Loop }}Raw := valRaw[0]
					{{- template "partial_query_type_conversion" (conversionData (printf "val%s" .Loop) (printf "val%sRaw" .Loop) "query" .Type.ElemType.Type) }}
				}
				{{ .VarName }}[key] = val{{ .Loop }}
			{{- end }}
		}
		{{- if or .DefaultValue (not .Required) }}
		}
		{{- end }}
	}


	{{- end }}
			{{- if and .Validate (not $scalar) }}
			{{ .Validate }}
			{{- end }}
	{{- end }}

{{- range .Headers }}
 {{- $scalar := and (not .Bearer.Framed) (or .IsTextUnmarshaler (not .Slice)) }}
 {{- if .Bearer.Framed }}
 {
  if {{ .Locals.Raw }} := r.Header.Values("{{ .HTTPName }}"); {{ if .Required }}len({{ .Locals.Raw }}) == 0 {
   err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "header"))
  } else { {{ else }}len({{ .Locals.Raw }}) > 0 { {{ end }}
   {{ .Locals.Val }}, ok := security.DecodeBearer({{ .Locals.Raw }}[0])
   if !ok{{ if .Bearer.AllowOtherScheme }} && !security.IsOtherAuthorizationScheme({{ .Locals.Raw }}[0], "Bearer"){{ end }} {
    {{ requestErrorReturn "loom.DecodePayloadError(\"invalid Authorization bearer credential\")" }}
   }
   {{- if .Bearer.AllowOtherScheme }}
   if ok {
   {{- end }}
   {{ .VarName }} = {{ if .Pointer }}&{{ end }}{{ .Locals.Val }}
   {{- if .Bearer.AllowOtherScheme }}
   }
   {{- end }}
  }
 }
 {{- else if $scalar }}
 {
  {{ .Locals.Raw }} := r.Header.Get("{{ .HTTPName }}")
  {{ .Locals.Present }} := len(r.Header.Values("{{ .HTTPName }}")) > 0
  {{- template "partial_scalar_element" (scalarElementData .Element "header") }}
 }
	{{- else if .StringSlice }}
		{{ .VarName }} = loomhttp.HeaderValues(r.Header, "{{ .CanonicalName }}")
		{{- if .Required }}
		if {{ .VarName }} == nil {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "header"))
		}
		{{- else if (ne .DefaultValue nil) }}
		if {{ .VarName }} == nil {
			{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
		}
		{{- end }}

	{{- else if .Slice }}
	{
		{{ .Locals.Raw }} := loomhttp.HeaderValues(r.Header, "{{ .CanonicalName }}")
		{{ if .Required }}if {{ .Locals.Raw }} == nil {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "header"))
		}
		{{- else if (ne .DefaultValue nil) }}
		if {{ .Locals.Raw }} == nil {
			{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
		}
		{{- end }}

		{{- if (ne .DefaultValue nil) }}else {
		{{- else if not .Required }}
		if {{ .Locals.Raw }} != nil {
		{{- end }}
		{{- template "partial_element_slice_conversion" . }}
		{{- if or (ne .DefaultValue nil) (not .Required) }}
		}
		{{- end }}
	}


	{{- end }}
		{{- if and .Validate (not $scalar) }}
			{{ .Validate }}
		{{- end }}
	{{- end }}

{{- range .Cookies }}
 {
  c, cookieErr := r.Cookie("{{ .HTTPName }}")
  if cookieErr != nil && !errors.Is(cookieErr, http.ErrNoCookie) {
   {{ requestErrorReturn "cookieErr" }}
  }
  var {{ .Locals.Raw }} string
  {{ .Locals.Present }} := c != nil
  if {{ .Locals.Present }} {
   {{ .Locals.Raw }} = c.Value
  }
  {{- template "partial_scalar_element" (scalarElementData .Element "cookie") }}
 }
{{- end }}
{{- end }}`},
}
