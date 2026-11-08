package codegen

import (
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type responseHeaderEncoding struct {
	*ResponseData
	// Source is the actual native response value expression.
	Source string
	// Target is the destination http.Header expression.
	Target string
}

func renderResponseHeaders(response *ResponseData, source, target string) (string, error) {
	funcs := map[string]any{
		"isAliased": func(dt expr.DataType) bool {
			_, ok := dt.(expr.UserType)
			return ok
		},
		"renderHeaderConversion":  renderHeaderConversion,
		"isIntegerValue":          isIntegerValue,
		"renderIntegerValue":      renderIntegerValue,
		"renderStyledEncode":      renderStyledEncode,
		"transportFieldReference": transportFieldReference,
		"transportScalarValue":    transportScalarValue,
		"printValue":              printValue,
	}
	section := codegen.NewTextTemplateSection("response-headers", responseHeaderSource, funcs, responseHeaderEncoding{response, source, target})
	var out strings.Builder
	if err := section.Write(&out); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

const responseHeaderSource = `{{- $source := .Source }}
	{{- range .Headers }}
{{- if .Styled }}
{{ renderStyledEncode .Element $source "response-header" $.Target }}
{{- else }}
		{{- $initDef := and (or .FieldPointer .Slice) (or .DefaultValue (and .Slice (ne .DefaultValue nil))) (not $.Tag) }}
		{{- $checkNil := and (or .FieldPointer .Slice (eq .Type.Name "bytes") (eq .Type.Name "any") $initDef) (not $.Tag) }}
		{{- if $checkNil }}
	if {{ $source }}{{ if .FieldName }}.{{ .FieldName }}{{ end }} != nil {
		{{- end }}

		{{- if and (eq .Type.Name "string") (not (isAliased .FieldType)) }}
	{{ $.Target }}.Set("{{ .CanonicalName }}", {{ transportScalarValue .Type .FieldType (transportFieldReference .AttributeData $source) .IsTextUnmarshaler }})
		{{- else }}
{{- if not $checkNil }}
{
{{- end }}
	val := {{ if .FieldPointer }}*{{ end }}{{ $source }}{{ if .FieldName }}.{{ .FieldName }}{{ end }}
	{{ if isIntegerValue .AttributeData }}{{ renderIntegerValue .AttributeData .Locals.Encoded "val" "" }}{{ else }}{{ renderHeaderConversion .Type .FieldType .Locals.Encoded "val" .IsTextUnmarshaler }}{{ end }}
	{{- if .Slice }}
	if err := loomhttp.SetHeaderValues({{ $.Target }}, "{{ .CanonicalName }}", {{ .Locals.Encoded }}); err != nil {
		return err
	}
	{{- else }}
	{{ $.Target }}.Set("{{ .CanonicalName }}", {{ .Locals.Encoded }})
	{{- end }}
{{- if not $checkNil }}
}
{{- end }}
		{{- end }}

		{{- if $initDef }}
	{{ if $checkNil }} } else { {{ else }}if {{ $source }}.{{ .FieldName }} == nil { {{ end }}
		{{- if .Slice }}
		val := {{ printf "%#v" .DefaultValue }}
		{{ renderHeaderConversion .Type .Type .Locals.Encoded "val" false }}
		if err := loomhttp.SetHeaderValues({{ $.Target }}, "{{ .CanonicalName }}", {{ .Locals.Encoded }}); err != nil {
			return err
		}
		{{- else }}
		{{ $.Target }}.Set("{{ .CanonicalName }}", "{{ printValue .Type .DefaultValue }}")
		{{- end }}
		{{- end }}

		{{- if or $checkNil $initDef }}
	}
		{{- end }}

	{{- end }}
	{{- end }}
`
