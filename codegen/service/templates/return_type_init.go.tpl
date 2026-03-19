{{ if or .ToResult .ToViewed }}
	var {{ .ReturnVar }} {{ .ReturnTypeRef }}
	switch {{ .ViewExpr }} {
		{{- range .Views }}
		case {{ printf "%q" .Name }}{{ if eq .Name "default" }}, ""{{ end }}:
			{{- if $.ToViewed }}
				p := {{ $.InitName }}{{ if ne .Name "default" }}{{ goify .Name true }}{{ end }}({{ $.ArgVar }})
				{{ $.ReturnVar }} = {{ if not $.IsCollection }}&{{ end }}{{ $.TargetType }}{Projected: p, View: {{ printf "%q" .Name }} }
			{{- else }}
				{{ $.ReturnVar }} = {{ $.InitName }}{{ if ne .Name "default" }}{{ goify .Name true }}{{ end }}({{ $.ArgVar }}.Projected)
			{{- end }}
		{{- end }}
		default:
			panic(goa.InvalidEnumValueError("view", {{ .ViewExpr }}, []any{
				{{- range .Views }}
				{{ printf "%q" .Name }},
				{{- end }}
			}))
	}
	return {{ .ReturnVar }}
{{- else if .IsCollection -}}
	{{ .ReturnVar }} := make({{ .TargetType }}, len({{ .ArgVar }}))
	for i, n := range {{ .ArgVar }} {
		{{ .ReturnVar }}[i] = {{ .InitName }}(n)
	}
	return {{ .ReturnVar }}
{{- else -}}
	{{ .Code }}
	{{- range .Fields }}
		if {{ $.Source }}.{{ .VarName }} != nil {
			{{ $.Target }}.{{ .VarName }} = {{ .FieldInit }}({{ $.Source }}.{{ .VarName }})
		}
	{{- end }}
	return {{ .ReturnVar }}
{{- end -}}
