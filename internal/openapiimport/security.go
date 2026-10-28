package openapiimport

import (
	"fmt"
	"go/token"
	"sort"
	"strings"

	"github.com/CaliLuke/libopenapi/datamodel/high/base"
	v3 "github.com/CaliLuke/libopenapi/datamodel/high/v3"
	"github.com/CaliLuke/libopenapi/orderedmap"
	"github.com/CaliLuke/loom/internal/openapiref"
	"github.com/pb33f/go-yaml"
)

// securitySchemeDescriptions resolves each alias chain from authored reference occurrences.
func (a *analyzer) securitySchemeDescriptions(source *orderedmap.Map[string, *v3.SecurityScheme]) map[string]string {
	sources := make(map[string]*v3.SecurityScheme, orderedmap.Len(source))
	for name, scheme := range source.FromOldest() {
		sources[name] = scheme
	}

	descriptions := make(map[string]string, len(sources))
	active := make(map[string]bool, len(sources))
	for name := range source.FromOldest() {
		a.securitySchemeDescription(name, sources, descriptions, active)
	}
	return descriptions
}

func (a *analyzer) securitySchemeDescription(
	name string,
	sources map[string]*v3.SecurityScheme,
	descriptions map[string]string,
	active map[string]bool,
) string {
	if description, exists := descriptions[name]; exists {
		return description
	}
	source := sources[name]
	if source == nil {
		return ""
	}
	if active[name] {
		return source.Description
	}
	active[name] = true
	defer delete(active, name)

	description := source.Description
	low := source.GoLow()
	if low != nil && low.GetReference() != "" {
		if targetName, err := openapiref.Name(low.GetReference(), "#/components/securitySchemes/"); err == nil {
			if _, exists := sources[targetName]; exists {
				description = a.securitySchemeDescription(targetName, sources, descriptions, active)
			}
		}
		if !strings.HasPrefix(a.version, "3.0.") {
			if override := referenceDescriptionOverride(low.GetReferenceNode()); override != nil {
				description = *override
			}
		}
	}
	descriptions[name] = description
	return description
}

func (a *analyzer) securityScheme(
	name string,
	source *v3.SecurityScheme,
	path string,
	description string,
) (SecurityScheme, bool) {
	if source == nil {
		a.unsupported("security-scheme", path, "security scheme is nil")
		return SecurityScheme{}, false
	}
	if source.Reference != "" {
		a.unsupported("security-scheme", path, "security scheme references are not in the strict import subset")
		return SecurityScheme{}, false
	}
	scheme := SecurityScheme{
		Name: name, Type: source.Type, Scheme: strings.ToLower(source.Scheme), Description: description,
		In: source.In, ParameterName: source.Name, Deprecated: source.Deprecated,
		OAuth2MetadataURL: source.OAuth2MetadataUrl, Extensions: a.extensions(path, source.Extensions),
	}
	if !a.openAPI32() && (source.Deprecated || source.OAuth2MetadataUrl != "") {
		a.unsupported("versioned-field", path, "security deprecated and oauth2MetadataUrl require OpenAPI 3.2")
		return SecurityScheme{}, false
	}
	switch source.Type {
	case "apiKey":
		if !a.supportedAPIKeyScheme(source, path) {
			return SecurityScheme{}, false
		}
	case "http":
		if !a.supportedHTTPScheme(source, path) {
			return SecurityScheme{}, false
		}
	case "oauth2":
		flows, ok := a.oauthFlows(source.Flows, path+"/flows")
		if !ok || source.Scheme != "" || source.BearerFormat != "" || source.In != "" || source.Name != "" || source.OpenIdConnectUrl != "" {
			a.unsupported("security-scheme", path, "OAuth2 scheme contains fields outside the strict import subset")
			return SecurityScheme{}, false
		}
		scheme.OAuthFlows = flows
	default:
		a.unsupported("security-scheme", path, fmt.Sprintf("security scheme type %q is not in the strict import subset", source.Type))
		return SecurityScheme{}, false
	}
	return scheme, true
}

func (a *analyzer) supportedAPIKeyScheme(source *v3.SecurityScheme, path string) bool {
	if source.In != "header" && source.In != "query" && source.In != "cookie" {
		a.unsupported("security-scheme", path+"/in", fmt.Sprintf("API key location %q is not supported", source.In))
		return false
	}
	if strings.TrimSpace(source.Name) == "" {
		a.unsupported("security-scheme", path+"/name", "API key name must not be empty")
		return false
	}
	if strings.Contains(source.Name, ":") {
		a.unsupported("security-scheme", path+"/name", "API key names containing ':' are not renderable")
		return false
	}
	if source.Scheme != "" || source.BearerFormat != "" || source.Flows != nil || source.OpenIdConnectUrl != "" || source.OAuth2MetadataUrl != "" {
		a.unsupported("security-scheme", path, "API key scheme contains fields outside the strict import subset")
		return false
	}
	return true
}

func (a *analyzer) supportedHTTPScheme(source *v3.SecurityScheme, path string) bool {
	scheme := strings.ToLower(source.Scheme)
	if scheme != "basic" && scheme != "bearer" {
		a.unsupported("security-scheme", path+"/scheme", fmt.Sprintf("HTTP security scheme %q is not supported", source.Scheme))
		return false
	}
	if source.BearerFormat != "" {
		a.unsupported("security-scheme", path+"/bearerFormat", "bearerFormat has no lossless Loom DSL representation")
		return false
	}
	if source.In != "" || source.Name != "" || source.Flows != nil || source.OpenIdConnectUrl != "" || source.OAuth2MetadataUrl != "" {
		a.unsupported("security-scheme", path, "HTTP security scheme contains fields outside the strict import subset")
		return false
	}
	return true
}

func (a *analyzer) oauthFlows(source *v3.OAuthFlows, path string) ([]OAuthFlow, bool) {
	if source == nil {
		a.unsupported("security-scheme", path, "OAuth2 security scheme must define flows")
		return nil, false
	}
	if source.Device != nil || oauthDeviceAuthorizationPresent(source) {
		a.unsupported("security-scheme", path+"/deviceAuthorization", "OAuth device authorization import is blocked until the parser exposes the official field")
		return nil, false
	}
	if orderedmap.Len(source.Extensions) > 0 {
		a.unsupported("security-scheme", path, "OAuth flow-container extensions are not in the strict import subset")
		return nil, false
	}
	types := []struct {
		kind string
		flow *v3.OAuthFlow
	}{
		{kind: "authorizationCode", flow: source.AuthorizationCode},
		{kind: "implicit", flow: source.Implicit},
		{kind: "password", flow: source.Password},
		{kind: "clientCredentials", flow: source.ClientCredentials},
	}
	var result []OAuthFlow
	for _, candidate := range types {
		if candidate.flow == nil {
			continue
		}
		flowPath := path + "/" + candidate.kind
		if orderedmap.Len(candidate.flow.Extensions) > 0 {
			a.unsupported("security-scheme", flowPath, "OAuth flow extensions are not in the strict import subset")
			return nil, false
		}
		if !a.validOAuthFlow(candidate.kind, candidate.flow, flowPath) {
			return nil, false
		}

		result = append(result, OAuthFlow{
			Kind: candidate.kind, AuthorizationURL: candidate.flow.AuthorizationUrl,
			TokenURL: candidate.flow.TokenUrl, RefreshURL: candidate.flow.RefreshUrl,
			Scopes: oauthScopes(candidate.flow.Scopes),
		})
	}
	if len(result) == 0 {
		a.unsupported("security-scheme", path, "OAuth2 security scheme must define at least one supported flow")
		return nil, false
	}
	return result, true
}

func oauthDeviceAuthorizationPresent(flows *v3.OAuthFlows) bool {
	if flows == nil || flows.GoLow() == nil {
		return false
	}
	root := flows.GoLow().GetRootNode()
	if root == nil || root.Kind != yaml.MappingNode {
		return false
	}
	for index := 0; index+1 < len(root.Content); index += 2 {
		if root.Content[index].Value == "deviceAuthorization" {
			return true
		}
	}
	return false
}

func oauthScopes(source *orderedmap.Map[string, string]) []SecurityScope {
	result := make([]SecurityScope, 0, orderedmap.Len(source))
	for name, description := range source.FromOldest() {
		result = append(result, SecurityScope{Name: name, Description: description})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

func (a *analyzer) validOAuthFlow(kind string, flow *v3.OAuthFlow, path string) bool {
	if flow.Scopes == nil {
		a.unsupported("security-scheme", path+"/scopes", "OAuth flow must define a scopes map")
		return false
	}
	needsAuthorizationURL := kind == "authorizationCode" || kind == "implicit"
	if needsAuthorizationURL && strings.TrimSpace(flow.AuthorizationUrl) == "" {
		a.unsupported("security-scheme", path+"/authorizationUrl", "OAuth flow must define an authorization URL")
		return false
	}
	needsTokenURL := kind == "authorizationCode" || kind == "password" || kind == "clientCredentials"
	if needsTokenURL && strings.TrimSpace(flow.TokenUrl) == "" {
		a.unsupported("security-scheme", path+"/tokenUrl", "OAuth flow must define a token URL")
		return false
	}
	return true
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
			scheme := securitySchemeByName(components.SecuritySchemes, name)
			if scheme == nil {
				a.unsupported("security-requirement", schemePath, fmt.Sprintf("security scheme %q is not a supported component", name))
				continue
			}

			normalizedScopes := append([]string(nil), scopes...)
			sort.Strings(normalizedScopes)
			requirement.Schemes = append(requirement.Schemes, SecurityRequirementScheme{
				Name:   name,
				Scopes: normalizedScopes,
			})
		}
		requirements = append(requirements, requirement)
	}
	return requirements
}

func securityScopeDefined(scheme SecurityScheme, name string) bool {
	for _, flow := range scheme.OAuthFlows {
		for _, scope := range flow.Scopes {
			if scope.Name == name {
				return true
			}
		}
	}
	return false
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
	constructor := ""
	switch {
	case scheme.Type == "apiKey":
		constructor = "APIKeySecurity"
	case scheme.Type == "http" && scheme.Scheme == "basic":
		constructor = "BasicAuthSecurity"
	case scheme.Type == "http" && scheme.Scheme == "bearer":
		constructor = "JWTSecurity"
	case scheme.Type == "oauth2":
		constructor = "OAuth2Security"
	default:
		return fmt.Errorf("render OpenAPI design: security scheme %q has unsupported type %q and scheme %q", scheme.Name, scheme.Type, scheme.Scheme)
	}
	r.open("var %s = %s(%q, func()", scheme.GoName, constructor, scheme.Name)
	if scheme.Description != "" {
		r.line("Description(%q)", scheme.Description)
	}
	if scheme.Deprecated {
		r.line("Meta(%q, %q)", "openapi:deprecated", "true")
	}
	if scheme.OAuth2MetadataURL != "" {
		r.line("Meta(%q, %q)", "openapi:oauth2MetadataUrl", scheme.OAuth2MetadataURL)
	}
	for _, flow := range scheme.OAuthFlows {
		switch flow.Kind {
		case "authorizationCode":
			r.open("AuthorizationCodeFlow(%q, %q, %q, func()", flow.AuthorizationURL, flow.TokenURL, flow.RefreshURL)
		case "implicit":
			r.open("ImplicitFlow(%q, %q, func()", flow.AuthorizationURL, flow.RefreshURL)
		case "password":
			r.open("PasswordFlow(%q, %q, func()", flow.TokenURL, flow.RefreshURL)
		case "clientCredentials":
			r.open("ClientCredentialsFlow(%q, %q, func()", flow.TokenURL, flow.RefreshURL)
		default:
			return fmt.Errorf("render OpenAPI design: security scheme %q has unsupported OAuth flow %q", scheme.Name, flow.Kind)
		}
		for _, scope := range flow.Scopes {
			r.line("Scope(%q, %q)", scope.Name, scope.Description)
		}
		r.close()
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
		if err := validateSecurityRequirement(requirement, r.document.Components, r.document.OpenAPIVersion); err != nil {
			return fmt.Errorf("render OpenAPI design: %w", err)
		}
		for _, reference := range requirement.Schemes {
			scheme := securitySchemeByName(r.document.Components.SecuritySchemes, reference.Name)
			if scheme == nil {
				return fmt.Errorf("render OpenAPI design: security scheme %q is not defined", reference.Name)
			}
			arguments = append(arguments, scheme.GoName)
		}
		hasNames := false
		for _, reference := range requirement.Schemes {
			hasNames = hasNames || len(reference.Scopes) > 0
		}
		if !hasNames {
			r.line("Security(%s)", strings.Join(arguments, ", "))
			continue
		}
		r.open("Security(%s, func()", strings.Join(arguments, ", "))
		for _, reference := range requirement.Schemes {
			scheme := securitySchemeByName(r.document.Components.SecuritySchemes, reference.Name)
			call := "RoleFor"
			if scheme.Type == "oauth2" {
				call = "ScopeFor"
			}
			for _, name := range reference.Scopes {
				r.line("%s(%s, %q)", call, scheme.GoName, name)
			}
		}
		r.close()
	}
	return nil
}

// validateSecurityRequirement also checks normalized documents supplied directly
// to Render. Role names have no protocol-defined inventory.
func validateSecurityRequirement(requirement SecurityRequirement, components Components, version string) error {
	seen := make(map[string]bool)
	for _, reference := range requirement.Schemes {
		scheme := securitySchemeByName(components.SecuritySchemes, reference.Name)
		if scheme == nil {
			return fmt.Errorf("security scheme %q is not defined", reference.Name)
		}
		if seen[reference.Name] {
			return fmt.Errorf("security scheme %q occurs twice in one requirement", reference.Name)
		}
		seen[reference.Name] = true
		if scheme.Type != "oauth2" {
			if strings.HasPrefix(version, "3.0.") && len(reference.Scopes) > 0 {
				return fmt.Errorf("non-OAuth security requirement arrays must be empty in OpenAPI 3.0")
			}
			continue
		}
		for _, scope := range reference.Scopes {
			if !securityScopeDefined(*scheme, scope) {
				return fmt.Errorf("OAuth scope %q is not defined by scheme %q", scope, reference.Name)
			}
		}
	}
	return nil
}

func (p *documentPlanner) securityRequirements(requirements SecurityRequirements, path string) {
	for index, requirement := range requirements {
		if err := validateSecurityRequirement(requirement, p.document.Components, p.document.OpenAPIVersion); err != nil {
			p.unsupported("security-requirement", fmt.Sprintf("%s/%d", path, index), err.Error())
		}
	}
}
