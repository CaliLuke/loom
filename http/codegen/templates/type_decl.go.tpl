{{ comment .Description }}
type {{ .VarName }} {{ .Def }}
{{- if .FlatFormUnionField }}

// MarshalFormValues marshals the synthetic request body wrapper using the
// wrapped union field at the top level.
func (body {{ .VarName }}) MarshalFormValues(values url.Values, prefix string) error {
	return body.{{ .FlatFormUnionField }}.MarshalFormValues(values, prefix)
}

// UnmarshalFormValues unmarshals the synthetic request body wrapper using the
// wrapped union field at the top level.
func (body *{{ .VarName }}) UnmarshalFormValues(values url.Values, prefix string) error {
	return (&body.{{ .FlatFormUnionField }}).UnmarshalFormValues(values, prefix)
}
{{- end }}
