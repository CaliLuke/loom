package openapiimport

func (r *renderer) parameterMappings(parameters []renderedParameter, path string) error {
	for _, parameter := range parameters {
		if parameter.securityScheme != "" && parameter.securityKind != "apiKey" {
			continue
		}
		if err := r.parameterMapping(parameter, path); err != nil {
			return err
		}
	}
	return nil
}
