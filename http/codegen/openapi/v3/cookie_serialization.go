package openapiv3

import "fmt"

// lowerOpenAPI31Cookies preserves the actual codec when removing the 3.2-only
// cookie style. Schema-domain equivalence is established by shared IR analysis.
func lowerOpenAPI31Cookies(spec *OpenAPI) error {
	lower := func(parameters []*ParameterRef) error {
		for _, ref := range parameters {
			if ref == nil || ref.Value == nil {
				continue
			}
			parameter := ref.Value
			if parameter.In != "cookie" || parameter.Style != "cookie" {
				continue
			}
			if !parameter.CookieFormEquivalent31 {
				return fmt.Errorf("cookie %q uses raw cookie serialization that OpenAPI 3.1 cannot represent; select OpenAPI 3.2 or explicitly use form serialization", parameter.Name)
			}
			parameter.Style = ""
		}
		return nil
	}
	for _, name := range sortedStringMapKeys(spec.Paths) {
		path := spec.Paths[name]
		if path == nil {
			continue
		}
		if err := lower(path.Parameters); err != nil {
			return err
		}
		for _, operation := range pathItemOperations(path) {
			if operation != nil {
				if err := lower(operation.Parameters); err != nil {
					return err
				}
			}
		}
	}
	if spec.Components != nil {
		for _, name := range sortedStringMapKeys(spec.Components.Parameters) {
			if err := lower([]*ParameterRef{spec.Components.Parameters[name]}); err != nil {
				return err
			}
		}
	}
	return nil
}
