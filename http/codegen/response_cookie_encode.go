package codegen

import (
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type responseCookieEncoding struct {
	*ResponseData
	// Source is the expression for the native response value.
	Source string
	// Target is the destination HTTP header expression.
	Target string
	// Context is the actual request context used by cookie policies.
	Context string
}

func renderResponseCookies(response *ResponseData, source, target, context string) (string, error) {
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
	section := codegen.NewTextTemplateSection("response-cookies", responseCookieSource, funcs, responseCookieEncoding{response, source, target, context})
	var out strings.Builder
	if err := section.Write(&out); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

const responseCookieSource = `{{- $source := .Source }}
	{{- range .Cookies }}
		{{- $initDef := and (or .FieldPointer .Slice) .DefaultValue }}
		{{- $checkNil := and (or .FieldPointer .Slice (eq .Type.Name "bytes") (eq .Type.Name "any") $initDef) }}
		{{- if $checkNil }}
	if {{ $.Source }}.{{ .FieldName }} != nil {
		{{- end }}

		{{- if and (eq .Type.Name "string") (not (isAliased .FieldType)) }}
	{{ .VarName }} := {{ transportScalarValue .Type .FieldType (transportFieldReference .AttributeData $source) .IsTextUnmarshaler }}
		{{- else }}
	{{ .Locals.EncodedRaw }} := {{ if .FieldPointer }}*{{ end }}{{ $.Source }}{{ if .FieldName }}.{{ .FieldName }}{{ end }}
	{{ if isIntegerValue .AttributeData }}{{ renderIntegerValue .AttributeData .VarName .Locals.EncodedRaw "" }}{{ else }}{{ renderHeaderConversion .Type .FieldType .VarName .Locals.EncodedRaw .IsTextUnmarshaler }}{{ end }}
		{{- end }}

		{{- if $initDef }}
	{{ if $checkNil }} } else { {{ else }}if {{ $.Source }}.{{ .FieldName }} == nil { {{ end }}
		{{ .VarName }} := "{{ printValue .Type .DefaultValue }}"
		{{- end }}
		{{ .Locals.Encoded }}, err := loomhttp.PrepareResponseCookie({{ $.Context }}, &http.Cookie{
			Name: {{ printf "%q" .HTTPName }},
			Value: {{ .VarName }},
			{{- if .MaxAge }}
			MaxAge: {{ .MaxAge }},
			{{- end }}
			{{- if .Path }}
			Path: {{ printf "%q" .Path }},
			{{- end }}
			{{- if .Domain }}
			Domain: {{ printf "%q" .Domain }},
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
		if err != nil {
			return err
		}
		{{ $.Target }}.Add("Set-Cookie", {{ .Locals.Encoded }})
		{{- if or $checkNil $initDef }}
	}
		{{- end }}

	{{- end }}
`
