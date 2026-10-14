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
		if node := low.GetReferenceNode(); node != nil {
			for index := 0; index+1 < len(node.Content); index += 2 {
				if node.Content[index].Value == "description" {
					value := node.Content[index+1].Value
					parameter.ReferenceDescription = &value
					break
				}
			}
		}
	}
	return parameter
}
