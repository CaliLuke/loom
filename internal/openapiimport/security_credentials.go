package openapiimport

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

type (
	// credentialRole retains the semantic scheme annotation independently from
	// the physical field carrying it. Requirements retain the auth check identities.
	credentialRole struct {
		scheme string
		kind   string
	}
	// credentialSlot identifies one component of a wire operation. Basic uses
	// separate username/password components; header names compare without case.
	credentialSlot struct {
		location  string
		name      string
		codec     string
		component string
	}
	credentialSource struct {
		slot      credentialSlot
		role      credentialRole
		suffix    string
		parameter Parameter
	}
)

// securityParameters allocates each physical source once before assigning Go
// names. Attribute roles and transport mappings consume these same records.
func (r *renderer) securityParameters(operation *Operation, parameters []renderedParameter, path string) ([]renderedParameter, error) {
	requirements := r.document.Security
	if operation.SecurityDefined {
		requirements = operation.Security
	} else if !r.document.SecurityDefined {
		requirements = nil
	}
	usedFields := make(map[string]int, len(parameters)+len(requirements))
	for _, parameter := range parameters {
		uniqueName(codegen.Goify(parameter.field, false), usedFields)
	}
	seen := make(map[string]bool)
	slots := make(map[credentialSlot]int)
	for requirementIndex, requirement := range requirements {
		if err := validateSecurityRequirement(requirement, r.document.Components, r.document.OpenAPIVersion); err != nil {
			return nil, fmt.Errorf("render OpenAPI design: %s/%d: %w", path, requirementIndex, err)
		}
		for schemeIndex, reference := range requirement.Schemes {
			if seen[reference.Name] {
				continue
			}
			seen[reference.Name] = true
			scheme := securitySchemeByName(r.document.Components.SecuritySchemes, reference.Name)
			if scheme == nil {
				return nil, fmt.Errorf("render OpenAPI design: %s/%d/%d references undefined security scheme %q", path, requirementIndex, schemeIndex, reference.Name)
			}
			for _, source := range securityCredentialSources(*scheme) {
				if index, ok := slots[source.slot]; ok {
					parameters[index].securityRoles = append(parameters[index].securityRoles, source.role)
					continue
				}
				slots[source.slot] = len(parameters)
				parameters = append(parameters, renderedParameter{
					parameter:     source.parameter,
					field:         uniqueName(codegen.Goify(scheme.Name+" "+source.suffix, false), usedFields),
					securityRoles: []credentialRole{source.role},
				})
			}
		}
	}
	return parameters, nil
}

// securityCredentialSources translates the supported OpenAPI source encodings.
// Expr remains responsible for admitting coexistence of distinct operations.
func securityCredentialSources(scheme SecurityScheme) []credentialSource {
	source := func(suffix, kind, codec, component string, parameter Parameter) credentialSource {
		location, name := "header", "authorization"
		if scheme.Type == "apiKey" {
			location, name = scheme.In, scheme.ParameterName
			if location == "header" {
				name = strings.ToLower(name)
			}
		}
		return credentialSource{
			slot: credentialSlot{location: location, name: name, codec: codec, component: component},
			role: credentialRole{scheme: scheme.Name, kind: kind}, suffix: suffix, parameter: parameter,
		}
	}
	switch {
	case scheme.Type == "apiKey":
		return []credentialSource{source("credential", "apiKey", "raw", "value", Parameter{Name: scheme.ParameterName, In: scheme.In, Schema: &Schema{Type: "string"}})}
	case scheme.Type == "http" && scheme.Scheme == "basic":
		return []credentialSource{
			source("username", "username", "basic", "username", Parameter{Schema: &Schema{Type: "string"}}),
			source("password", "password", "basic", "password", Parameter{Schema: &Schema{Type: "string"}}),
		}
	case scheme.Type == "http" && scheme.Scheme == "bearer":
		return []credentialSource{source("token", "token", "bearer", "value", Parameter{Schema: &Schema{Type: "string"}})}
	case scheme.Type == "oauth2":
		return []credentialSource{source("access token", "accessToken", "bearer", "value", Parameter{Schema: &Schema{Type: "string"}})}
	default:
		return nil
	}
}

func (r *renderer) securityCredential(parameter renderedParameter) error {
	role := parameter.securityRoles[0]
	var call string
	switch role.kind {
	case "apiKey":
		call = fmt.Sprintf("APIKey(%q, %q, String", role.scheme, parameter.field)
	case "username":
		call = fmt.Sprintf("Username(%q, String", parameter.field)
	case "password":
		call = fmt.Sprintf("Password(%q, String", parameter.field)
	case "token":
		call = fmt.Sprintf("Token(%q, String", parameter.field)
	case "accessToken":
		call = fmt.Sprintf("AccessToken(%q, String", parameter.field)
	default:
		return fmt.Errorf("render OpenAPI design: unsupported security credential kind %q", role.kind)
	}
	seen := map[string]bool{role.tag(): true}
	var additional []credentialRole
	for _, other := range parameter.securityRoles[1:] {
		if !seen[other.tag()] {
			additional = append(additional, other)
			seen[other.tag()] = true
		}
	}
	if len(additional) == 0 {
		r.line("%s)", call)
		return nil
	}
	r.open("%s, func()", call)
	for _, other := range additional {
		if other.kind == "apiKey" {
			r.line("Meta(%q, %q)", other.tag(), other.scheme)
		} else {
			r.line("Meta(%q)", other.tag())
		}
	}
	r.close()
	return nil
}

func (r credentialRole) tag() string {
	if r.kind == "apiKey" {
		return "security:apikey:" + r.scheme
	}
	return "security:" + strings.ToLower(r.kind)
}
