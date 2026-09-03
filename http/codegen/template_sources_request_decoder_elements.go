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
		{{- if and .MustValidate (not .ServerBody) }}
			err error
		{{- end }}
		{{- if .DecodePlan.HasPathParams }}

			params = mux.Vars(r)
		{{- end }}
		)

	{{- range .PathParams }}
		{{- if .IsTextUnmarshaler }}
			{
				{{ .VarName }}Raw := params["{{ .HTTPName }}"]
				{{- template "partial_path_conversion" . }}
				{{- if .Validate }}
				{{ .Validate }}
				{{- end }}
			}

	{{- else if and (or (eq .Type.Name "string") (eq .Type.Name "any")) }}
		{{ .VarName }} = params["{{ .HTTPName }}"]

		{{- else }}{{/* not string and not any */}}
			{
				{{ .VarName }}Raw := params["{{ .HTTPName }}"]
				{{- template "partial_path_conversion" . }}
			}

		{{- end }}
			{{- if and .Validate (not .IsTextUnmarshaler) }}
			{{ .Validate }}
			{{- end }}
	{{- end }}

{{- $qpVar := "r.URL.Query()" }}
{{- if gt (len .QueryParams) 1 }}
{{- $qpVar = "qp" }}
qp := r.URL.Query()
{{- end }}
{{- range .QueryParams }}
	{{- if .IsTextUnmarshaler }}
		{{ .VarName }}Raw := {{$qpVar}}.Get("{{ .HTTPName }}")
		{{- if .Required }}
		if {{ .VarName }}Raw == "" {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "query string"))
		}
		{{- else if .DefaultValue }}
		if {{ .VarName }}Raw == "" {
			{{ .VarName }}Raw = {{ printf "%q" .DefaultValue }}
		}
		{{- end }}
		{{- if or .DefaultValue .Required }}
		{{- template "partial_query_type_conversion" . }}
			{{- if .Validate }}
			{{ .Validate }}
			{{- end }}
		{{- else }}
		if {{ .VarName }}Raw != "" {
			{{- template "partial_query_type_conversion" . }}
			{{- if .Validate }}
			{{ .Validate }}
			{{- end }}
		}
		{{- end }}

	{{- else if and (or (eq .Type.Name "string") (eq .Type.Name "any")) .Required }}
		{{ .VarName }} = {{$qpVar}}.Get("{{ .HTTPName }}")
		if {{ .VarName }} == "" {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "query string"))
		}

	{{- else if (or (eq .Type.Name "string") (eq .Type.Name "any")) }}
	{
		rawValues, present := {{$qpVar}}["{{ .HTTPName }}"]
		if present {
			raw := ""
			if len(rawValues) > 0 {
				raw = rawValues[0]
			}
			{{ .VarName }} = {{ if and (eq .Type.Name "string") .Pointer }}&{{ end }}raw
		}
		{{- if .DefaultValue }} else {
			{{ .VarName }} = {{ if eq .Type.Name "string" }}{{ printf "%q" .DefaultValue }}{{ else }}{{ printf "%#v" .DefaultValue }}{{ end }}
		}
		{{- end }}
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
		{{ .VarName }}Raw := {{$qpVar}}["{{ .HTTPName }}"]
		{{- if .Required }}
		if {{ .VarName }}Raw == nil {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "query string"))
		}
		{{- else if .DefaultValue }}
		if {{ .VarName }}Raw == nil {
			{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
		}
		{{- end }}

		{{- if .DefaultValue }}else {
		{{- else if not .Required }}
		if {{ .VarName }}Raw != nil {
		{{- end }}
		{{- template "partial_element_slice_conversion" . }}
		{{- if or .DefaultValue (not .Required) }}
		}
		{{- end }}
	}

	{{- else if .Map }}
	{
		{{ .VarName }}Raw := {{$qpVar}}
		{{ .VarName }}HasValues := false
		for keyRaw := range {{ .VarName }}Raw {
			if strings.HasPrefix(keyRaw, "{{ .HTTPName }}[") {
				{{ .VarName }}HasValues = true
				break
			}
		}
		{{- if .Required }}
		if !{{ .VarName }}HasValues {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "query string"))
		}
		{{- else if .DefaultValue }}
		if !{{ .VarName }}HasValues {
			{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
		}
		{{- end }}

		{{- if .DefaultValue }}else {
		{{- else if not .Required }}
		if {{ .VarName }}HasValues {
		{{- end }}
		for keyRaw, valRaw := range {{ .VarName }}Raw {
			if strings.HasPrefix(keyRaw, "{{ .HTTPName }}[") {
				{{- template "partial_query_map_conversion" (mapQueryDecodeData .Type .VarName 0) }}
			}
		}
		{{- if or .DefaultValue (not .Required) }}
		}
		{{- end }}
	}

	{{- else if .MapQueryParams }}
	{
		{{ .VarName }}Raw := {{$qpVar}}
		{{- if .Required }}
		if len({{ .VarName }}Raw) == 0 {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "query string"))
		}
		{{- else if .DefaultValue }}
		if len({{ .VarName }}Raw) == 0 {
			{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
		}
		{{- end }}

		{{- if .DefaultValue }}else {
		{{- else if not .Required }}
		if len({{ .VarName }}Raw) != 0 {
		{{- end }}
		if {{ .VarName }} == nil {
			{{ .VarName }} = make({{ goTypeRef .Type }})
		}
		for keyRaw, valRaw := range {{ .VarName }}Raw {
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
						{{- template "partial_element_slice_conversion" (conversionData "val" "query" .Type.ElemType.Type) }}
					}
					{{ .VarName }}[key] = val
				{{- end }}
			{{- else if eq .Type.ElemType.Type.Name "map" }}
				{{- template "partial_query_map_conversion" (mapQueryDecodeData .Type.ElemType.Type (printf "%s[key]" .VarName) 1) }}
			{{- else }}
				var val{{ .Loop }} {{ goTypeRef .Type.ElemType.Type }}
				{
					val{{ .Loop }}Raw := valRaw[0]
					{{- template "partial_query_type_conversion" (conversionData (printf "val%s" .Loop) "query" .Type.ElemType.Type) }}
				}
				{{ .VarName }}[key] = val{{ .Loop }}
			{{- end }}
		}
		{{- if or .DefaultValue (not .Required) }}
		}
		{{- end }}
	}

	{{- else }}{{/* not string, not any, not slice and not map */}}
	{
		{{ .VarName }}Raw := {{$qpVar}}.Get("{{ .HTTPName }}")
		{{- if .Required }}
		if {{ .VarName }}Raw == "" {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "query string"))
		}
		{{- else if .DefaultValue }}
		if {{ .VarName }}Raw == "" {
			{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
		}
		{{- end }}

		{{- if .DefaultValue }}else {
		{{- else if not .Required }}
		if {{ .VarName }}Raw != "" {
		{{- end }}
		{{- template "partial_query_type_conversion" . }}
		{{- if or .DefaultValue (not .Required) }}
		}
		{{- end }}
	}

	{{- end }}
			{{- if and .Validate (not .IsTextUnmarshaler) }}
			{{ .Validate }}
			{{- end }}
	{{- end }}

{{- range .Headers }}
	{{- if .IsTextUnmarshaler }}
		{{ .VarName }}Raw := r.Header.Get("{{ .HTTPName }}")
		{{- if .Required }}
		if {{ .VarName }}Raw == "" {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "header"))
		}
		{{- else if .DefaultValue }}
		if {{ .VarName }}Raw == "" {
			{{ .VarName }}Raw = {{ printf "%q" .DefaultValue }}
		}
		{{- end }}
		{{- if or .DefaultValue .Required }}
		{{- template "partial_query_type_conversion" . }}
			{{- if .Validate }}
			{{ .Validate }}
			{{- end }}
		{{- else }}
		if {{ .VarName }}Raw != "" {
			{{- template "partial_query_type_conversion" . }}
			{{- if .Validate }}
			{{ .Validate }}
			{{- end }}
		}
		{{- end }}

	{{- else if and (or (eq .Type.Name "string") (eq .Type.Name "any")) .Required }}
		{{ .VarName }} = r.Header.Get("{{ .HTTPName }}")
		if {{ .VarName }} == "" {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "header"))
		}

	{{- else if (or (eq .Type.Name "string") (eq .Type.Name "any")) }}
		{{ .VarName }}Raw := r.Header.Get("{{ .HTTPName }}")
		if {{ .VarName }}Raw != "" {
			{{ .VarName }} = {{ if and (eq .Type.Name "string") .Pointer }}&{{ end }}{{ .VarName }}Raw
		}
		{{- if .DefaultValue }} else {
			{{ .VarName }} = {{ if eq .Type.Name "string" }}{{ printf "%q" .DefaultValue }}{{ else }}{{ printf "%#v" .DefaultValue }}{{ end }}
		}
		{{- end }}

	{{- else if .StringSlice }}
		{{ .VarName }} = r.Header["{{ .CanonicalName }}"]
		{{- if .Required }}
		if {{ .VarName }} == nil {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "header"))
		}
		{{- else if .DefaultValue }}
		if {{ .VarName }} == nil {
			{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
		}
		{{- end }}

	{{- else if .Slice }}
	{
		{{ .VarName }}Raw := r.Header["{{ .CanonicalName }}"]
		{{ if .Required }}if {{ .VarName }}Raw == nil {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "header"))
		}
		{{- else if .DefaultValue }}
		if {{ .VarName }}Raw == nil {
			{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
		}
		{{- end }}

		{{- if .DefaultValue }}else {
		{{- else if not .Required }}
		if {{ .VarName }}Raw != nil {
		{{- end }}
		{{- template "partial_element_slice_conversion" . }}
		{{- if or .DefaultValue (not .Required) }}
		}
		{{- end }}
	}

	{{- else }}{{/* not string, not any and not slice */}}
	{
		{{ .VarName }}Raw := r.Header.Get("{{ .HTTPName }}")
		{{- if .Required }}
		if {{ .VarName }}Raw == "" {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "header"))
		}
		{{- else if .DefaultValue }}
		if {{ .VarName }}Raw == "" {
			{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
		}
		{{- end }}

		{{- if .DefaultValue }}else {
		{{- else if not .Required }}
		if {{ .VarName }}Raw != "" {
		{{- end }}
		{{- template "partial_query_type_conversion" . }}
		{{- if or .DefaultValue (not .Required) }}
		}
		{{- end }}
	}
	{{- end }}
		{{- if and .Validate (not .IsTextUnmarshaler) }}
			{{ .Validate }}
		{{- end }}
	{{- end }}

{{- range .Cookies }}
		{
			c, cookieErr := r.Cookie("{{ .HTTPName }}")
			if cookieErr != nil {
				if errors.Is(cookieErr, http.ErrNoCookie) {
					{{- if .Required }}
					err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "cookie"))
					{{- end }}
				} else {
					return payload, cookieErr
				}
			}
		{{- if .IsTextUnmarshaler }}
			var {{ .VarName }}Raw string
			if c != nil {
				{{ .VarName }}Raw = c.Value
			}
			{{- if .DefaultValue }}
			if {{ .VarName }}Raw == "" {
				{{ .VarName }}Raw = {{ printf "%q" .DefaultValue }}
			}
			{{- end }}
			{{- if or .DefaultValue .Required }}
			{{- template "partial_query_type_conversion" . }}
				{{- if .Validate }}
				{{ .Validate }}
				{{- end }}
			{{- else }}
			if {{ .VarName }}Raw != "" {
				{{- template "partial_query_type_conversion" . }}
				{{- if .Validate }}
				{{ .Validate }}
				{{- end }}
			}
			{{- end }}

		{{- else if and (or (eq .Type.Name "string") (eq .Type.Name "any")) .Required }}
			if c != nil {
				{{ .VarName }} = c.Value
			}

		{{- else if (or (eq .Type.Name "string") (eq .Type.Name "any")) }}
			var {{ .VarName }}Raw string
			if c != nil {
				{{ .VarName }}Raw = c.Value
			}
			if {{ .VarName }}Raw != "" {
				{{ .VarName }} = {{ if and (eq .Type.Name "string") .Pointer }}&{{ end }}{{ .VarName }}Raw
			}
			{{- if .DefaultValue }} else {
				{{ .VarName }} = {{ if eq .Type.Name "string" }}{{ printf "%q" .DefaultValue }}{{ else }}{{ printf "%#v" .DefaultValue }}{{ end }}
			}
			{{- end }}

		{{- else }}{{/* not string and not any */}}
			var {{ .VarName }}Raw string
			if c != nil {
				{{ .VarName }}Raw = c.Value
		}
		{{- if .Required }}
		if {{ .VarName }}Raw == "" {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "cookie"))
		}
		{{- else if .DefaultValue }}
		if {{ .VarName }}Raw == "" {
			{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
		}
		{{- end }}

		{{- if .DefaultValue }}else {
		{{- else if not .Required }}
		if {{ .VarName }}Raw != "" {
		{{- end }}
		{{- template "partial_query_type_conversion" . }}
			{{- if or .DefaultValue (not .Required) }}
			}
			{{- end }}
		{{- end }}
		}
			{{- if and .Validate (not .IsTextUnmarshaler) }}
				{{ .Validate }}
			{{- end }}
{{- end }}
{{- end }}`},
}
