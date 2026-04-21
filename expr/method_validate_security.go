package expr

import (
	"github.com/CaliLuke/loom/eval"
)

// validateRequirements validates the security requirements.
func (m *MethodExpr) validateRequirements() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	requirements := m.validationRequirements()
	flags := newRequirementFlags()
	for _, r := range requirements {
		verr.Merge(m.validateRequirementSchemes(r, flags))
		verr.Merge(m.validateRequirementScopes(r))
	}
	verr.Merge(m.validateMissingRequirementTags(flags))
	return verr
}

type requirementFlags struct {
	hasBasicAuth bool
	hasAPIKey    bool
	hasJWT       bool
	hasOAuth     bool
}

func newRequirementFlags() *requirementFlags {
	return &requirementFlags{}
}

func (m *MethodExpr) validateRequirementSchemes(r *SecurityExpr, flags *requirementFlags) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	for _, s := range r.Schemes {
		verr.Merge(s.Validate())
		verr.Merge(m.validateRequirementScheme(s, flags))
	}
	return verr
}

func (m *MethodExpr) validateRequirementScheme(s *SchemeExpr, flags *requirementFlags) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	switch s.Kind {
	case BasicAuthKind:
		flags.hasBasicAuth = true
		verr.Merge(m.validateBasicAuthTags())
	case APIKeyKind:
		flags.hasAPIKey = true
		verr.Merge(m.validateAPIKeyTags(s))
	case JWTKind:
		flags.hasJWT = true
		verr.Merge(m.validateSecurityTag("security:token", "payload of method %q of service %q does not define a JWT attribute, use Token to define one"))
	case OAuth2Kind:
		flags.hasOAuth = true
		verr.Merge(m.validateSecurityTag("security:accesstoken", "payload of method %q of service %q does not define a OAuth2 access token attribute, use AccessToken to define one"))
	}
	return verr
}

func (m *MethodExpr) validateBasicAuthTags() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	verr.Merge(m.validateSecurityTag("security:username", "payload of method %q of service %q does not define a username attribute, use Username to define one"))
	verr.Merge(m.validateSecurityTag("security:password", "payload of method %q of service %q does not define a password attribute, use Password to define one"))
	return verr
}

func (m *MethodExpr) validateAPIKeyTags(s *SchemeExpr) *eval.ValidationErrors {
	if m.isTransportOwnedCookieScheme(s) {
		return nil
	}
	return m.validateSecurityTag("security:apikey:"+s.SchemeName, "payload of method %q of service %q does not define an API key attribute, use APIKey to define one")
}

func (m *MethodExpr) validateSecurityTag(tag, msg string) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if !hasTag(m.Payload, tag) {
		verr.Add(m, msg, m.Name, m.Service.Name)
	}
	return verr
}

func (m *MethodExpr) validateRequirementScopes(r *SecurityExpr) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	for _, scope := range r.Scopes {
		if requirementHasScope(r, scope) {
			continue
		}
		verr.Add(m, "security scope %q not found in any of the security schemes.", scope)
	}
	return verr
}

func requirementHasScope(r *SecurityExpr, scope string) bool {
	for _, s := range r.Schemes {
		if !schemeSupportsScopes(s) {
			continue
		}
		for _, se := range s.Scopes {
			if se.Name == scope {
				return true
			}
		}
	}
	return false
}

func schemeSupportsScopes(s *SchemeExpr) bool {
	switch s.Kind {
	case BasicAuthKind, APIKeyKind, OAuth2Kind, JWTKind:
		return true
	default:
		return false
	}
}

func (m *MethodExpr) validateMissingRequirementTags(flags *requirementFlags) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if !flags.hasBasicAuth {
		verr.Merge(m.validateUnexpectedSecurityTag("security:username", "payload of method %q of service %q defines a username attribute, but no basic auth security scheme exist"))
		verr.Merge(m.validateUnexpectedSecurityTag("security:password", "payload of method %q of service %q defines a password attribute, but no basic auth security scheme exist"))
	}
	if !flags.hasAPIKey {
		verr.Merge(m.validateUnexpectedSecurityTagPrefix("security:apikey", "payload of method %q of service %q defines an API key attribute, but no APIKey security scheme exist"))
	}
	if !flags.hasJWT {
		verr.Merge(m.validateUnexpectedSecurityTag("security:token", "payload of method %q of service %q defines a JWT token attribute, but no JWT auth security scheme exist"))
	}
	if !flags.hasOAuth {
		verr.Merge(m.validateUnexpectedSecurityTag("security:accesstoken", "payload of method %q of service %q defines a OAuth2 access token attribute, but no OAuth2 security scheme exist"))
	}
	return verr
}

func (m *MethodExpr) validateUnexpectedSecurityTag(tag, msg string) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if hasTag(m.Payload, tag) {
		verr.Add(m, msg, m.Name, m.Service.Name)
	}
	return verr
}

func (m *MethodExpr) validateUnexpectedSecurityTagPrefix(prefix, msg string) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if hasTagPrefix(m.Payload, prefix) {
		verr.Add(m, msg, m.Name, m.Service.Name)
	}
	return verr
}

func (m *MethodExpr) isTransportOwnedCookieScheme(s *SchemeExpr) bool {
	if m == nil || s == nil || s.Kind != APIKeyKind {
		return false
	}
	for _, sessionAuth := range m.validationSessionAuths() {
		for _, transport := range sessionAuth.Transports {
			if transport == nil || transport.Kind != SessionCookieTransportKind || transport.PayloadOwned() {
				continue
			}
			if transport.Scheme != nil && transport.Scheme.SchemeName == s.SchemeName {
				return true
			}
		}
	}
	return false
}

