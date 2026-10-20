package openapiimport

import (
	"strings"

	v3 "github.com/CaliLuke/libopenapi/datamodel/high/v3"
)

func (a *analyzer) parameterReference(source *v3.Parameter, reference string) Parameter {
	parameter := Parameter{Ref: reference}
	if strings.HasPrefix(a.version, "3.0.") {
		return parameter
	}
	// The high-level description may already come from the component. Read
	// the reference occurrence to distinguish absence from an empty override.
	if low := source.GoLow(); low != nil {
		parameter.ReferenceDescription = referenceDescriptionOverride(low.GetReferenceNode())
	}
	return parameter
}
