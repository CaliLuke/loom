package codegen

// requestDecoderPartials holds the helper templates composed into
// requestDecoderSource. Declared in a separate file to keep the main
// template source file under the file-size ceiling.
var requestDecoderPartials = []templateSource{
	{name: "request_elements", source: `{{- if or .PathParams .QueryParams .Headers .Cookies }}
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
		{{- if .Cookies }}
			c *http.Cookie
		{{- end }}
		{{- if .PathParams }}

			params = mux.Vars(r)
		{{- end }}
		)

{{- range .PathParams }}
	{{- if and (or (eq .Type.Name "string") (eq .Type.Name "any")) }}
		{{ .VarName }} = params["{{ .HTTPName }}"]

	{{- else }}{{/* not string and not any */}}
		{
			{{ .VarName }}Raw := params["{{ .HTTPName }}"]
			{{- template "partial_path_conversion" . }}
		}

	{{- end }}
		{{- if .Validate }}
		{{ .Validate }}
		{{- end }}
{{- end }}

{{- $qpVar := "r.URL.Query()" }}
{{- if gt (len .QueryParams) 1 }}
{{- $qpVar = "qp" }}
qp := r.URL.Query()
{{- end }}
{{- range .QueryParams }}
	{{- if and (or (eq .Type.Name "string") (eq .Type.Name "any")) .Required }}
		{{ .VarName }} = {{$qpVar}}.Get("{{ .HTTPName }}")
		if {{ .VarName }} == "" {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "query string"))
		}

	{{- else if (or (eq .Type.Name "string") (eq .Type.Name "any")) }}
		{{ .VarName }}Raw := {{$qpVar}}.Get("{{ .HTTPName }}")
		if {{ .VarName }}Raw != "" {
			{{ .VarName }} = {{ if and (eq .Type.Name "string") .Pointer }}&{{ end }}{{ .VarName }}Raw
		}
		{{- if .DefaultValue }} else {
			{{ .VarName }} = {{ if eq .Type.Name "string" }}{{ printf "%q" .DefaultValue }}{{ else }}{{ printf "%#v" .DefaultValue }}{{ end }}
		}
		{{- end }}

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
			{{- if eq .Type.KeyType.Type.Name "string" }}
			key = keyRaw
			{{- else }}
				{{- template "partial_query_type_conversion" (conversionData "key" "query" .Type.KeyType.Type) }}
			{{- end }}
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
		{{- if .Validate }}
		{{ .Validate }}
		{{- end }}
{{- end }}

{{- range .Headers }}
	{{- if and (or (eq .Type.Name "string") (eq .Type.Name "any")) .Required }}
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
	{{- if .Validate }}
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
		{{- if and (or (eq .Type.Name "string") (eq .Type.Name "any")) .Required }}
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
		{{- if .Validate }}
			{{ .Validate }}
		{{- end }}
{{- end }}
{{- end }}`},
	{name: "slice_item_conversion", source: `		{{- if eq .Type.ElemType.Type.Name "string" }}
			{{ .VarName }}[i] = rv
		{{- else if eq .Type.ElemType.Type.Name "bytes" }}
			{{ .VarName }}[i] = []byte(rv)
		{{- else if eq .Type.ElemType.Type.Name "int" }}
			v, err2 := strconv.ParseInt(rv, 10, strconv.IntSize)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of integers"))
			}
			{{ .VarName }}[i] = int(v)
		{{- else if eq .Type.ElemType.Type.Name "int32" }}
			v, err2 := strconv.ParseInt(rv, 10, 32)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of integers"))
			}
			{{ .VarName }}[i] = int32(v)
		{{- else if eq .Type.ElemType.Type.Name "int64" }}
			v, err2 := strconv.ParseInt(rv, 10, 64)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of integers"))
			}
			{{ .VarName }}[i] = v
		{{- else if eq .Type.ElemType.Type.Name "uint" }}
			v, err2 := strconv.ParseUint(rv, 10, strconv.IntSize)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of unsigned integers"))
			}
			{{ .VarName }}[i] = uint(v)
		{{- else if eq .Type.ElemType.Type.Name "uint32" }}
			v, err2 := strconv.ParseUint(rv, 10, 32)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of unsigned integers"))
			}
			{{ .VarName }}[i] = uint32(v)
		{{- else if eq .Type.ElemType.Type.Name "uint64" }}
			v, err2 := strconv.ParseUint(rv, 10, 64)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of unsigned integers"))
			}
			{{ .VarName }}[i] = v
		{{- else if eq .Type.ElemType.Type.Name "float32" }}
			v, err2 := strconv.ParseFloat(rv, 32)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of floats"))
			}
			{{ .VarName }}[i] = float32(v)
		{{- else if eq .Type.ElemType.Type.Name "float64" }}
			v, err2 := strconv.ParseFloat(rv, 64)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of floats"))
			}
			{{ .VarName }}[i] = v
		{{- else if eq .Type.ElemType.Type.Name "boolean" }}
			v, err2 := strconv.ParseBool(rv)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of booleans"))
			}
			{{ .VarName }}[i] = v
		{{- else if eq .Type.ElemType.Type.Name "any" }}
			{{ .VarName }}[i] = rv
		{{- else }}
			// unsupported slice type {{ .Type.ElemType.Type.Name }} for var {{ .VarName }}
		{{- end }}`},
	{name: "element_slice_conversion", source: `	{{ .VarName }} = make({{ goTypeRef .Type }}, len({{ .VarName }}Raw))
	for i, rv := range {{ .VarName }}Raw {
		{{- template "partial_slice_item_conversion" . }}
	}`},
	{name: "query_slice_conversion", source: `	{{- if eq . "string" }} url.QueryEscape(v)
	{{- else if eq . "int" "int32" }} strconv.FormatInt(int64(v), 10)
	{{- else if eq . "int64" }} strconv.FormatInt(v, 10)
	{{- else if eq . "uint" "uint32" }} strconv.FormatUint(uint64(v), 10)
	{{- else if eq . "uint64" }} strconv.FormatUint(v, 10)
	{{- else if eq . "float32" }} strconv.FormatFloat(float64(v), 'f', -1, 32)
	{{- else if eq . "float64" }} strconv.FormatFloat(v, 'f', -1, 64)
	{{- else if eq . "boolean" }} strconv.FormatBool(v)
	{{- else if eq . "bytes" }} url.QueryEscape(string(v))
	{{- else }} url.QueryEscape(fmt.Sprintf("%v", v))
	{{- end }}`},
	{name: "query_type_conversion", source: `	{{- if eq .Type.Name "bytes" }}
		{{ .VarName }} = []byte({{.VarName}}Raw)
	{{- else if eq .Type.Name "int" }}
		v, err2 := strconv.ParseInt({{ .VarName }}Raw, 10, strconv.IntSize)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "integer"))
		}
		{{- if .Pointer }}
		pv := {{ if .TypeRef }}{{slice .TypeRef 1 (len .TypeRef)}}{{ else }}int{{ end }}(v)
		{{ .VarName }} = &pv
		{{- else }}
		{{ .VarName }} = {{ if .TypeRef }}{{ .TypeRef }}{{ else }}int{{ end }}(v)
		{{- end }}
	{{- else if eq .Type.Name "int32" }}
		v, err2 := strconv.ParseInt({{ .VarName }}Raw, 10, 32)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "integer"))
		}
		{{- if .Pointer }}
		pv := {{ if .TypeRef }}{{ slice .TypeRef 1 (len .TypeRef) }}{{ else }}int32{{ end }}(v)
		{{ .VarName }} = &pv
		{{- else }}
		{{ .VarName }} = {{ if .TypeRef }}{{ .TypeRef }}{{ else }}int32{{ end }}(v)
		{{- end }}
	{{- else if eq .Type.Name "int64" }}
		v, err2 := strconv.ParseInt({{ .VarName }}Raw, 10, 64)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "integer"))
		}
		{{ if and (ne .TypeRef nil) (and (ne .TypeRef "int64") (ne .TypeRef "*int64")) }}{{ .VarName }} = ({{.TypeRef}})({{ if .Pointer }}&{{ end }}v){{ else }}{{ .VarName }} = {{ if .Pointer }}&{{ end }}v{{ end }}
	{{- else if eq .Type.Name "uint" }}
		v, err2 := strconv.ParseUint({{ .VarName }}Raw, 10, strconv.IntSize)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "unsigned integer"))
		}
		{{- if .Pointer }}
		pv := {{ if .TypeRef }}{{ slice .TypeRef 1 (len .TypeRef) }}{{ else }}uint{{ end }}(v)
		{{ .VarName }} = &pv
		{{- else }}
		{{ .VarName }} = {{ if .TypeRef }}{{ .TypeRef }}{{ else }}uint{{ end }}(v)
		{{- end }}
	{{- else if eq .Type.Name "uint32" }}
		v, err2 := strconv.ParseUint({{ .VarName }}Raw, 10, 32)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "unsigned integer"))
		}
		{{- if .Pointer }}
		pv := {{ if .TypeRef }}{{ slice .TypeRef 1 (len .TypeRef) }}{{ else }}uint32{{ end }}(v)
		{{ .VarName }} = &pv
		{{- else }}
		{{ .VarName }} = {{ if .TypeRef }}{{ .TypeRef }}{{ else }}uint32{{ end }}(v)
		{{- end }}
	{{- else if eq .Type.Name "uint64" }}
		v, err2 := strconv.ParseUint({{ .VarName }}Raw, 10, 64)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "unsigned integer"))
		}
		{{ if and (ne .TypeRef nil) (and (ne .TypeRef "uint64") (ne .TypeRef "*uint64")) }}{{ .VarName }} = ({{.TypeRef}})({{ if .Pointer }}&{{ end }}v){{ else }}{{ .VarName }} = {{ if .Pointer }}&{{ end }}v{{ end }}
	{{- else if eq .Type.Name "float32" }}
		v, err2 := strconv.ParseFloat({{ .VarName }}Raw, 32)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "float"))
		}
		{{- if .Pointer }}
		pv := {{ if .TypeRef }}{{ slice .TypeRef 1 (len .TypeRef) }}{{ else }}float32{{ end }}(v)
		{{ .VarName }} = &pv
		{{- else }}
		{{ .VarName }} = {{ if .TypeRef }}{{ .TypeRef }}{{ else }}float32{{ end }}(v)
		{{- end }}
	{{- else if eq .Type.Name "float64" }}
		v, err2 := strconv.ParseFloat({{ .VarName }}Raw, 64)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "float"))
		}
		{{ if and (ne .TypeRef nil) (and (ne .TypeRef "float64") (ne .TypeRef "*float64")) }}{{ .VarName }} = ({{.TypeRef}})({{ if .Pointer }}&{{ end }}v){{ else }}{{ .VarName }} = {{ if .Pointer }}&{{ end }}v{{ end }}
	{{- else if eq .Type.Name "boolean" }}
		v, err2 := strconv.ParseBool({{ .VarName }}Raw)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "boolean"))
		}
		{{ if and (ne .TypeRef nil) (and (ne .TypeRef "bool") (ne .TypeRef "*bool")) }}{{ .VarName }} = ({{.TypeRef}})({{ if .Pointer }}&{{ end }}v){{ else }}{{ .VarName }} = {{ if .Pointer }}&{{ end }}v{{ end }}
	{{- else }}
		// unsupported type {{ .Type.Name }} for var {{ .VarName }}
	{{- end }}`},
	{name: "query_map_conversion", source: `	if {{ .VarName }} == nil {
		{{ .VarName }} = make({{ goTypeRef .Type }})
	}
	var key{{ .Loop }} {{ goTypeRef .Type.KeyType.Type }}
	{
		openIdx := strings.IndexRune(keyRaw, '[')
		closeIdx := strings.IndexRune(keyRaw, ']')
		if openIdx == -1 || closeIdx == -1 || closeIdx <= openIdx {
			err = loom.MergeErrors(err, loom.DecodePayloadError("invalid query string: malformed brackets"))
		} else {
	{{- if eq .Type.KeyType.Type.Name "string" }}
		key{{ .Loop }} = keyRaw[openIdx+1 : closeIdx]
	{{- else }}
		key{{ .Loop }}Raw := keyRaw[openIdx+1 : closeIdx]
		{{- template "partial_query_type_conversion" (conversionData (printf "key%s" .Loop) "query" .Type.KeyType.Type) }}
	{{- end }}
		{{- if gt .Depth 0 }}
			keyRaw = keyRaw[closeIdx+1:]
		{{- end }}
		}
	}
	{{- if eq .Type.ElemType.Type.Name "string" }}
		{{ .VarName }}[key{{ .Loop }}] = valRaw[0]
	{{- else if eq .Type.ElemType.Type.Name "array" }}
		{{- if eq .Type.ElemType.Type.ElemType.Type.Name "string" }}
			{{ .VarName }}[key{{ .Loop }}] = valRaw
		{{- else }}
			var val {{ goTypeRef .Type.ElemType.Type }}
			{
				{{- template "partial_element_slice_conversion" (conversionData "val" "query" .Type.ElemType.Type) }}
			}
			{{ .VarName }}[key{{ .Loop }}] = val
		{{- end }}
	{{- else if eq .Type.ElemType.Type.Name "map" }}
		{{- template "partial_query_map_conversion" (mapQueryDecodeData .Type.ElemType.Type (printf "%s[key%s]" .VarName .Loop) 1) }}
	{{- else }}
		var val{{ .Loop }} {{ goTypeRef .Type.ElemType.Type }}
		{
			val{{ .Loop }}Raw := valRaw[0]
			{{- template "partial_query_type_conversion" (conversionData (printf "val%s" .Loop) "query" .Type.ElemType.Type) }}
		}
		{{ .VarName }}[key{{ .Loop }}] = val{{ .Loop }}
	{{- end }}`},
	{name: "path_conversion", source: `	{{- if eq .Type.Name "array" }}
		{{ .VarName }}RawSlice := strings.Split({{ .VarName }}Raw, ",")
		{{ .VarName }} = make({{ goTypeRef .Type }}, len({{ .VarName }}RawSlice))
		for i, rv := range {{ .VarName }}RawSlice {
			{{- template "partial_slice_item_conversion" . }}
		}
	{{- else }}
		{{- template "partial_query_type_conversion" . }}
	{{- end }}`},
}
