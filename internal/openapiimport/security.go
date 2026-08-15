package openapiimport

import (
	"fmt"
	"go/token"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

func (a *analyzer) securityScheme(name string, source *v3.SecurityScheme, path string) (SecurityScheme, bool) {
	if source == nil {
		a.unsupported("security-scheme", path, "security scheme is nil")
		return SecurityScheme{}, false
	}
	if source.Reference != "" {
		a.unsupported("security-scheme", path, "security scheme references are not in the strict import subset")
		return SecurityScheme{}, false
	}
	if source.Type != "apiKey" {
		a.unsupported("security-scheme", path, fmt.Sprintf("security scheme type %q is not in the strict import subset", source.Type))
		return SecurityScheme{}, false
	}
	if source.In != "header" && source.In != "query" && source.In != "cookie" {
		a.unsupported("security-scheme", path+"/in", fmt.Sprintf("API key location %q is not supported", source.In))
		return SecurityScheme{}, false
	}
	if strings.TrimSpace(source.Name) == "" {
		a.unsupported("security-scheme", path+"/name", "API key name must not be empty")
		return SecurityScheme{}, false
	}
	if strings.Contains(source.Name, ":") {
		a.unsupported("security-scheme", path+"/name", "API key names containing ':' are not renderable")
		return SecurityScheme{}, false
	}
	if source.Scheme != "" || source.BearerFormat != "" || source.Flows != nil || source.OpenIdConnectUrl != "" ||
		source.OAuth2MetadataUrl != "" || source.Deprecated {
		a.unsupported("security-scheme", path, "API key scheme contains fields outside the strict import subset")
		return SecurityScheme{}, false
	}
	return SecurityScheme{
		Name:          name,
		Type:          source.Type,
		Description:   source.Description,
		In:            source.In,
		ParameterName: source.Name,
		Extensions:    a.extensions(path, source.Extensions),
	}, true
}

func (a *analyzer) securityRequirements(source []*base.SecurityRequirement, path string, components Components) SecurityRequirements {
	requirements := make(SecurityRequirements, 0, len(source))
	for index, sourceRequirement := range source {
		requirementPath := fmt.Sprintf("%s/%d", path, index)
		if sourceRequirement == nil {
			a.unsupported("security-requirement", requirementPath, "security requirement is nil")
			continue
		}
		requirement := SecurityRequirement{}
		for name, scopes := range sourceRequirement.Requirements.FromOldest() {
			schemePath := requirementPath + "/" + escapeJSONPointer(name)
			if securitySchemeByName(components.SecuritySchemes, name) == nil {
				a.unsupported("security-requirement", schemePath, fmt.Sprintf("security scheme %q is not a supported component", name))
				continue
			}
			if len(scopes) > 0 {
				a.unsupported("security-requirement", schemePath, "API key security requirement scopes must be empty")
			}
			requirement.Schemes = append(requirement.Schemes, SecurityRequirementScheme{
				Name:   name,
				Scopes: append([]string(nil), scopes...),
			})
		}
		requirements = append(requirements, requirement)
	}
	return requirements
}

func securitySchemeByName(schemes []SecurityScheme, name string) *SecurityScheme {
	for index := range schemes {
		if schemes[index].Name == name {
			return &schemes[index]
		}
	}
	return nil
}

func (r *renderer) securityScheme(scheme SecurityScheme) error {
	if scheme.GoName == "" || !token.IsIdentifier(scheme.GoName) || token.Lookup(scheme.GoName).IsKeyword() {
		return fmt.Errorf("render OpenAPI design: security scheme %q has invalid Go name %q", scheme.Name, scheme.GoName)
	}
	r.open("var %s = APIKeySecurity(%q, func()", scheme.GoName, scheme.Name)
	if scheme.Description != "" {
		r.line("Description(%q)", scheme.Description)
	}
	if err := r.emitExtensions("", scheme.Extensions); err != nil {
		return err
	}
	r.close()
	r.line("")
	return nil
}

func (r *renderer) securityRequirements(requirements SecurityRequirements) error {
	for _, requirement := range requirements {
		if len(requirement.Schemes) == 0 {
			r.line("Security()")
			continue
		}
		arguments := make([]string, 0, len(requirement.Schemes))
		for _, reference := range requirement.Schemes {
			scheme := securitySchemeByName(r.document.Components.SecuritySchemes, reference.Name)
			if scheme == nil {
				return fmt.Errorf("render OpenAPI design: security scheme %q is not defined", reference.Name)
			}
			arguments = append(arguments, scheme.GoName)
		}
		r.line("Security(%s)", strings.Join(arguments, ", "))
	}
	return nil
}

func (r *renderer) securityParameters(operation *Operation, parameters []renderedParameter, path string) ([]renderedParameter, error) {
	requirements := r.document.Security
	if operation.SecurityDefined {
		requirements = operation.Security
	} else if !r.document.SecurityDefined {
		requirements = nil
	}
	if len(requirements) == 0 {
		return parameters, nil
	}
	usedFields := make(map[string]int, len(parameters)+len(requirements))
	for _, parameter := range parameters {
		uniqueName(parameter.field, usedFields)
	}
	seen := make(map[string]struct{})
	for requirementIndex, requirement := range requirements {
		for schemeIndex, reference := range requirement.Schemes {
			if _, ok := seen[reference.Name]; ok {
				continue
			}
			seen[reference.Name] = struct{}{}
			scheme := securitySchemeByName(r.document.Components.SecuritySchemes, reference.Name)
			if scheme == nil {
				return nil, fmt.Errorf(
					"render OpenAPI design: %s/%d/%d references undefined security scheme %q",
					path,
					requirementIndex,
					schemeIndex,
					reference.Name,
				)
			}
			field := uniqueName(codegen.Goify(scheme.Name+" credential", false), usedFields)
			parameters = append(parameters, renderedParameter{
				parameter: Parameter{
					Name:   scheme.ParameterName,
					In:     scheme.In,
					Schema: &Schema{Type: "string"},
				},
				field:          field,
				securityScheme: scheme.Name,
			})
		}
	}
	return parameters, nil
}
