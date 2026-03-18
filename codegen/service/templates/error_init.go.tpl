{{ printf "%s builds a %s from an error." .Name .TypeName |  comment }}
func {{ .Name }}(err error) {{ .TypeRef }} {
	{{- if or .RemedyCode .SafeMessage .RetryHint }}
	serr := goa.NewServiceError(err, {{ printf "%q" .ErrName }}, {{ printf "%v" .Timeout }}, {{ printf "%v" .Temporary}}, {{ printf "%v" .Fault}})
	goa.WithErrorRemedy(serr, &goa.ErrorRemedy{
		Code:        {{ printf "%q" .RemedyCode }},
		SafeMessage: {{ printf "%q" .SafeMessage }},
		RetryHint:   {{ printf "%q" .RetryHint }},
	})
	return serr
	{{- else }}
	return goa.NewServiceError(err, {{ printf "%q" .ErrName }}, {{ printf "%v" .Timeout }}, {{ printf "%v" .Temporary}}, {{ printf "%v" .Fault}})
	{{- end }}
}
