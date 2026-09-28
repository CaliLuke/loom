package ir

import (
	"cmp"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

type (
	// securityBinding identifies one authored scheme at one HTTP location.
	securityBinding struct {
		scheme string
		kind   expr.SchemeKind
		in     string
		name   string
	}

	// securityBindings allocates all names before operations reference them.
	securityBindings struct {
		names   map[securityBinding]string
		schemes map[string]*expr.SchemeExpr
	}
)

func newSecurityBindings(requirements ...[]*expr.SecurityExpr) (*securityBindings, error) {
	definitions := make(map[securityBinding]*expr.SchemeExpr)
	counts := make(map[string]int)
	for _, group := range requirements {
		for _, requirement := range group {
			for _, scheme := range requirement.Schemes {
				if scheme.Kind == expr.NoKind {
					continue
				}
				key := securityBindingFor(scheme)
				if _, ok := definitions[key]; ok {
					continue
				}
				definition := expr.DupScheme(scheme)
				definition.In, definition.Name = key.in, key.name
				definitions[key] = definition
				counts[key.scheme]++
			}
		}
	}
	keys := make([]securityBinding, 0, len(definitions))
	for key := range definitions {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b securityBinding) int {
		return cmp.Or(cmp.Compare(a.scheme, b.scheme), cmp.Compare(a.kind, b.kind),
			cmp.Compare(a.in, b.in), cmp.Compare(a.name, b.name))
	})
	bindings := &securityBindings{
		names:   make(map[securityBinding]string),
		schemes: make(map[string]*expr.SchemeExpr),
	}
	for _, key := range keys {
		if err := validateSecurityBinding(key); err != nil {
			return nil, err
		}
		name := key.scheme
		if counts[name] > 1 {
			base := name + "_" + key.in + "_" + securityComponentToken(key.name)
			name = base
			for suffix := 2; counts[name] > 0 || bindings.schemes[name] != nil; suffix++ {
				name = base + "_" + strconv.Itoa(suffix)
			}
		}
		bindings.names[key] = name
		bindings.schemes[name] = definitions[key]
	}
	return bindings, nil
}

func securityBindingFor(scheme *expr.SchemeExpr) securityBinding {
	in, name := scheme.In, scheme.Name
	if in == "" {
		in = "header"
	}
	if name == "" {
		name = "Authorization"
	}
	if in == "header" {
		name = http.CanonicalHeaderKey(name)
	}
	return securityBinding{scheme: scheme.Hash(), kind: scheme.Kind, in: in, name: name}
}

func validateSecurityBinding(binding securityBinding) error {
	if binding.in != "header" && binding.in != "query" && binding.in != "cookie" {
		return fmt.Errorf("OpenAPI security scheme %q uses unsupported %s credential %q; map the credential to a header, query parameter, or cookie, or exclude the endpoint from OpenAPI", binding.scheme, binding.in, binding.name)
	}
	if binding.kind == expr.OAuth2Kind && (binding.in != "header" || binding.name != "Authorization") {
		return fmt.Errorf("OpenAPI OAuth2 security scheme %q cannot describe %s credential %q while preserving OAuth flows and scopes; use the Authorization header or exclude the endpoint from OpenAPI", binding.scheme, binding.in, binding.name)
	}
	return nil
}

func securityComponentToken(name string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '.' || r == '-' {
			return r
		}
		return '_'
	}, name)
}

// apiSecurityRequirements resolves default HTTP locations before endpoint
// mappings are applied. Session transports retain their declared cookie name.
func apiSecurityRequirements(api *expr.APIExpr) []*expr.SecurityExpr {
	requirements := make([]*expr.SecurityExpr, 0, len(api.Requirements))
	for _, requirement := range api.Requirements {
		requirements = append(requirements, expr.DupRequirement(requirement))
	}
	for _, session := range api.SessionAuths {
		for _, transport := range session.Transports {
			if transport == nil || transport.Scheme == nil {
				continue
			}
			scheme := expr.DupScheme(transport.Scheme)
			scheme.In, scheme.Name = "header", "Authorization"
			if transport.Kind == expr.SessionCookieTransportKind {
				scheme.In, scheme.Name = "cookie", transport.HTTPName
				if scheme.Name == "" {
					scheme.Name = transport.TransportAttributeName()
				}
			}
			requirements = append(requirements, &expr.SecurityExpr{Schemes: []*expr.SchemeExpr{scheme}})
		}
	}
	return requirements
}

func (bindings *securityBindings) requirements(requirements []*expr.SecurityExpr) []map[string][]string {
	if len(requirements) == 0 {
		return nil
	}
	result := make([]map[string][]string, len(requirements))
	for i, requirement := range requirements {
		result[i] = make(map[string][]string)
		for _, scheme := range requirement.Schemes {
			if scheme.Kind == expr.NoKind {
				continue
			}
			scopes := []string{}
			if scheme.Kind == expr.OAuth2Kind && len(requirement.Scopes) > 0 {
				scopes = slices.Clone(requirement.Scopes)
			}
			result[i][bindings.names[securityBindingFor(scheme)]] = scopes
		}
	}
	return result
}
