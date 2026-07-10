package codegen

// httpDecoderConversionPartials holds conversion helpers shared by request and
// response decoder templates.
var httpDecoderConversionPartials = []templateSource{
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
	{name: "query_type_conversion", source: `	{{- if .IsTextUnmarshaler }}
		{{- if .Pointer }}
		var {{ .VarName }}Val {{ .TypeName }}
		if err2 := {{ .VarName }}Val.UnmarshalText([]byte({{ .VarName }}Raw)); err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName }}Raw, {{ printf "%q" .TypeName }}))
		} else {
			{{ .VarName }} = &{{ .VarName }}Val
		}
		{{- else }}
		if err2 := {{ .VarName }}.UnmarshalText([]byte({{ .VarName }}Raw)); err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName }}Raw, {{ printf "%q" .TypeName }}))
		}
		{{- end }}
	{{- else if eq .Type.Name "bytes" }}
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
	var key{{ .Loop }}Err error
	{
		openIdx := strings.IndexRune(keyRaw, '[')
		closeIdx := strings.IndexRune(keyRaw, ']')
		if openIdx == -1 || closeIdx == -1 || closeIdx <= openIdx {
			key{{ .Loop }}Err = loom.DecodePayloadError("invalid query string: malformed brackets")
			err = loom.MergeErrors(err, key{{ .Loop }}Err)
		} else {
	{{- if eq .Type.KeyType.Type.Name "string" }}
		key{{ .Loop }} = keyRaw[openIdx+1 : closeIdx]
	{{- else }}
		key{{ .Loop }}Raw := keyRaw[openIdx+1 : closeIdx]
		{{- if eq .Type.KeyType.Type.Name "int" }}
		v, err2 := strconv.ParseInt(key{{ .Loop }}Raw, 10, strconv.IntSize)
		if err2 != nil {
			key{{ .Loop }}Err = loom.InvalidFieldTypeError("query", key{{ .Loop }}Raw, "integer")
			err = loom.MergeErrors(err, key{{ .Loop }}Err)
		}
		key{{ .Loop }} = {{ goTypeRef .Type.KeyType.Type }}(v)
		{{- else if eq .Type.KeyType.Type.Name "int32" }}
		v, err2 := strconv.ParseInt(key{{ .Loop }}Raw, 10, 32)
		if err2 != nil {
			key{{ .Loop }}Err = loom.InvalidFieldTypeError("query", key{{ .Loop }}Raw, "integer")
			err = loom.MergeErrors(err, key{{ .Loop }}Err)
		}
		key{{ .Loop }} = {{ goTypeRef .Type.KeyType.Type }}(v)
		{{- else if eq .Type.KeyType.Type.Name "int64" }}
		v, err2 := strconv.ParseInt(key{{ .Loop }}Raw, 10, 64)
		if err2 != nil {
			key{{ .Loop }}Err = loom.InvalidFieldTypeError("query", key{{ .Loop }}Raw, "integer")
			err = loom.MergeErrors(err, key{{ .Loop }}Err)
		}
		key{{ .Loop }} = {{ goTypeRef .Type.KeyType.Type }}(v)
		{{- else if eq .Type.KeyType.Type.Name "uint" }}
		v, err2 := strconv.ParseUint(key{{ .Loop }}Raw, 10, strconv.IntSize)
		if err2 != nil {
			key{{ .Loop }}Err = loom.InvalidFieldTypeError("query", key{{ .Loop }}Raw, "unsigned integer")
			err = loom.MergeErrors(err, key{{ .Loop }}Err)
		}
		key{{ .Loop }} = {{ goTypeRef .Type.KeyType.Type }}(v)
		{{- else if eq .Type.KeyType.Type.Name "uint32" }}
		v, err2 := strconv.ParseUint(key{{ .Loop }}Raw, 10, 32)
		if err2 != nil {
			key{{ .Loop }}Err = loom.InvalidFieldTypeError("query", key{{ .Loop }}Raw, "unsigned integer")
			err = loom.MergeErrors(err, key{{ .Loop }}Err)
		}
		key{{ .Loop }} = {{ goTypeRef .Type.KeyType.Type }}(v)
		{{- else if eq .Type.KeyType.Type.Name "uint64" }}
		v, err2 := strconv.ParseUint(key{{ .Loop }}Raw, 10, 64)
		if err2 != nil {
			key{{ .Loop }}Err = loom.InvalidFieldTypeError("query", key{{ .Loop }}Raw, "unsigned integer")
			err = loom.MergeErrors(err, key{{ .Loop }}Err)
		}
		key{{ .Loop }} = {{ goTypeRef .Type.KeyType.Type }}(v)
		{{- else if eq .Type.KeyType.Type.Name "float32" }}
		v, err2 := strconv.ParseFloat(key{{ .Loop }}Raw, 32)
		if err2 != nil {
			key{{ .Loop }}Err = loom.InvalidFieldTypeError("query", key{{ .Loop }}Raw, "float")
			err = loom.MergeErrors(err, key{{ .Loop }}Err)
		}
		key{{ .Loop }} = {{ goTypeRef .Type.KeyType.Type }}(v)
		{{- else if eq .Type.KeyType.Type.Name "float64" }}
		v, err2 := strconv.ParseFloat(key{{ .Loop }}Raw, 64)
		if err2 != nil {
			key{{ .Loop }}Err = loom.InvalidFieldTypeError("query", key{{ .Loop }}Raw, "float")
			err = loom.MergeErrors(err, key{{ .Loop }}Err)
		}
		key{{ .Loop }} = {{ goTypeRef .Type.KeyType.Type }}(v)
		{{- else if eq .Type.KeyType.Type.Name "boolean" }}
		v, err2 := strconv.ParseBool(key{{ .Loop }}Raw)
		if err2 != nil {
			key{{ .Loop }}Err = loom.InvalidFieldTypeError("query", key{{ .Loop }}Raw, "boolean")
			err = loom.MergeErrors(err, key{{ .Loop }}Err)
		}
		key{{ .Loop }} = {{ goTypeRef .Type.KeyType.Type }}(v)
		{{- else }}
		key{{ .Loop }}Err = loom.InvalidFieldTypeError("query", key{{ .Loop }}Raw, {{ printf "%q" .Type.KeyType.Type.Name }})
		err = loom.MergeErrors(err, key{{ .Loop }}Err)
		{{- end }}
	{{- end }}
		{{- if gt .Depth 0 }}
			keyRaw = keyRaw[closeIdx+1:]
		{{- end }}
		}
	}
	if key{{ .Loop }}Err != nil {
		continue
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

func requestDecoderPartialSources() []templateSource {
	partials := make([]templateSource, 0, len(requestDecoderElementPartials)+len(httpDecoderConversionPartials))
	partials = append(partials, requestDecoderElementPartials...)
	return append(partials, httpDecoderConversionPartials...)
}

var requestDecoderPartials = requestDecoderPartialSources()
