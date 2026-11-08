package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
)

func renderResponseMetadataFields(response *ResponseData, serviceName string, method *service.MethodData, data *ServiceData, headers, cookies string) (string, error) {
	funcs := map[string]any{
		"goTypeRef": func(typ expr.DataType) string {
			return data.Service.Scope.GoTypeRef(&expr.AttributeExpr{Type: typ})
		},
		"mapQueryDecodeData":   mapQueryDecodeData,
		"conversionData":       conversionData,
		"isIntegerValue":       isIntegerValue,
		"integerSliceAssign":   integerSliceAssign,
		"integerPointerAssign": integerPointerAssign,
		"integerElementText":   integerElementText,
		"isStoredIntegerValue": isStoredIntegerValue,
		"integerDecodeCheck":   integerDecodeCheck,
		"renderIntegerValue":   renderIntegerValue,

		"scalarElementData":    scalarElementData,
		"renderStyledDecode":   renderStyledDecode,
		"renderElementDefault": renderElementDefault,
		"defaultErrorReturn": func(expression string) string {
			return fmt.Sprintf("return nil, loomhttp.ErrValidationError(%q, %q, %s)", serviceName, method.Name, expression)
		},
	}
	values := buildResponseData(response, serviceName, method)
	values["HeaderSource"] = headers
	values["CookieSource"] = cookies
	section := codegen.NewTextTemplateSection("response-headers-decode", joinHTTPTemplateSource(responseHeaderDecodeSource+responseCookieDecodeSource, httpDecoderConversionPartials...), funcs, values)
	var out strings.Builder
	if err := section.Write(&out); err != nil {
		return "", err
	}
	return strings.TrimLeft(out.String(), "\n\t "), nil
}

const responseHeaderDecodeSource = `{{- with .Data }}
	{{- if .Headers }}
			var (
		{{- range .Headers }}
				{{ .VarName }} {{ .TypeRef }}
		{{- end }}
		{{- if not .ClientBody }}
			{{- if .MustValidate }}
				err error
			{{- end }}
		{{- end }}
			)
		{{- range .Headers }}

 {{- $scalar := or .IsTextUnmarshaler (not .Slice) }}
 {{- if .Styled }}
 if parameterError := func() error {
 {{ renderStyledDecode .Element "response-header" $.HeaderSource }}
 }(); parameterError != nil {
 return nil, loomhttp.ErrValidationError("{{ $.ServiceName }}", "{{ $.Method.Name }}", parameterError)
 }
 {{- else if $scalar }}
 {
  {{ .Locals.Raw }} := {{ $.HeaderSource }}.Get("{{ .CanonicalName }}")
  {{ .Locals.Present }} := len({{ $.HeaderSource }}.Values("{{ .CanonicalName }}")) > 0
  {{- template "partial_scalar_element" (scalarElementData .Element "header") }}
 }
		{{- else if .StringSlice }}
			{{ .VarName }} = loomhttp.HeaderValues({{ $.HeaderSource }}, "{{ .CanonicalName }}")
			{{ if .Required }}
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
			{{ .Locals.Raw }} := loomhttp.HeaderValues({{ $.HeaderSource }}, "{{ .CanonicalName }}")
				{{ if .Required }} if {{ .Locals.Raw }} == nil {
				return nil, loomhttp.ErrValidationError("{{ $.ServiceName }}", "{{ $.Method.Name }}", loom.MissingFieldError("{{ .Name }}", "header"))
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
		{{- if and .Validate (not $scalar) (not .Styled) }}
			{{ .Validate }}
		{{- end }}
		{{- end }}{{/* range .Headers */}}
	{{- end }}

{{- end }}`

const responseCookieDecodeSource = `{{- with .Data }}
	{{- if .Cookies }}
			var (
		{{- range .Cookies }}
				{{ .VarName }}    {{ .TypeRef }}
				{{ .Locals.Raw }} string
            {{ .Locals.Present }} bool
		{{- end }}

				cookies = {{ $.CookieSource }}
		{{- if not .ClientBody }}
			{{- if .MustValidate }}
				{{- if not .Headers}}
					err error
				{{- end }}
			{{- end }}
		{{- end }}
			)
        for _, c := range cookies {
			switch c.Name {
		{{- range .Cookies }}
			case {{ printf "%q" .HTTPName }}:
				{{ .Locals.Raw }} = c.Value
                {{ .Locals.Present }} = true
		{{- end }}
			}
		}
		{{- range .Cookies }}

  {{- template "partial_scalar_element" (scalarElementData .Element "cookie") }}
		{{- end }}{{/* range .Cookies */}}
	{{- end }}

{{- end }}`
