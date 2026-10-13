package openapiimport

func (r *renderer) parameterMappings(parameters []renderedParameter, path string) error {
	for _, parameter := range parameters {
		if len(parameter.securityRoles) > 0 && parameter.parameter.In == "" {
			continue
		}
		if err := r.parameterMapping(parameter, path); err != nil {
			return err
		}
	}
	return nil
}
