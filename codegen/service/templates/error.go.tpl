// Error returns an error description.
func (e {{ .Ref }}) Error() string {
	return {{ printf "%q" .Description }}
}

// ErrorName returns the error name.
//
// Deprecated: Use GoaErrorName - https://github.com/goadesign/goa/issues/3105
func (e {{ .Ref }}) ErrorName() string {
	return e.GoaErrorName()
}

// GoaErrorName returns the error name.
func (e {{ .Ref }}) GoaErrorName() string {
	return {{ errorName . }}
}
{{- if or .RemedyCode .SafeMessage .RetryHint }}

// GoaErrorRemedy returns the remediation guidance for the error.
func (e {{ .Ref }}) GoaErrorRemedy() *goa.ErrorRemedy {
	return &goa.ErrorRemedy{
		Code:        {{ printf "%q" .RemedyCode }},
		SafeMessage: {{ printf "%q" .SafeMessage }},
		RetryHint:   {{ printf "%q" .RetryHint }},
	}
}
{{- end }}
