package service

import (
	"github.com/CaliLuke/loom/expr"
)

type (
	// RequirementsData is the list of security requirements.
	RequirementsData []*RequirementData

	// SchemesData is the list of security schemes.
	SchemesData []*SchemeData

	// RequirementData lists the schemes and scopes defined by a single
	// security requirement.
	RequirementData struct {
		// Schemes list the requirement schemes.
		Schemes []*SchemeData
		// Scopes list the required scopes.
		Scopes []string
	}

	// SchemeData describes a single security scheme.
	SchemeData struct {
		// Kind is the type of scheme, one of "Basic", "APIKey", "JWT"
		// or "OAuth2".
		Type string
		// SchemeName is the name of the scheme.
		SchemeName string
		// Name refers to a header or parameter name, based on In's
		// value.
		Name string
		// UsernameField is the name of the payload field that should be
		// initialized with the basic auth username if any.
		UsernameField string
		// UsernamePointer is true if the username field is a pointer.
		UsernamePointer bool
		// UsernameAttr is the name of the attribute that contains the
		// username.
		UsernameAttr string
		// UsernameRequired specifies whether the attribute that
		// contains the username is required.
		UsernameRequired bool
		// PasswordField is the name of the payload field that should be
		// initialized with the basic auth password if any.
		PasswordField string
		// PasswordPointer is true if the password field is a pointer.
		PasswordPointer bool
		// PasswordAttr is the name of the attribute that contains the
		// password.
		PasswordAttr string
		// PasswordRequired specifies whether the attribute that
		// contains the password is required.
		PasswordRequired bool
		// CredField contains the name of the payload field that should
		// be initialized with the API key, the JWT token or the OAuth2
		// access token.
		CredField string
		// CredPointer is true if the credential field is a pointer.
		CredPointer bool
		// CredRequired specifies if the key is a required attribute.
		CredRequired bool
		// KeyAttr is the name of the attribute that contains
		// the security tag (for APIKey, OAuth2, and JWT schemes).
		KeyAttr string
		// Scopes lists the scopes that apply to the scheme.
		Scopes []string
		// Flows describes the OAuth2 flows.
		Flows []*expr.FlowExpr
		// In indicates the request element that holds the credential.
		In string
		// TransportOwned is true when the credential is supplied by transport
		// state rather than a payload field.
		TransportOwned bool
	}
)

// Scheme returns the scheme data with the given scheme name.
func (r RequirementsData) Scheme(name string) *SchemeData {
	for _, req := range r {
		for _, s := range req.Schemes {
			if s.SchemeName == name {
				return s
			}
		}
	}
	return nil
}

// Dup creates a copy of the scheme data.
func (s *SchemeData) Dup() *SchemeData {
	return &SchemeData{
		Type:             s.Type,
		SchemeName:       s.SchemeName,
		Name:             s.Name,
		UsernameField:    s.UsernameField,
		UsernamePointer:  s.UsernamePointer,
		UsernameAttr:     s.UsernameAttr,
		UsernameRequired: s.UsernameRequired,
		PasswordField:    s.PasswordField,
		PasswordPointer:  s.PasswordPointer,
		PasswordAttr:     s.PasswordAttr,
		PasswordRequired: s.PasswordRequired,
		CredField:        s.CredField,
		CredPointer:      s.CredPointer,
		CredRequired:     s.CredRequired,
		KeyAttr:          s.KeyAttr,
		Scopes:           s.Scopes,
		Flows:            s.Flows,
		In:               s.In,
		TransportOwned:   s.TransportOwned,
	}
}

// Append appends a scheme data to schemes only if it doesn't exist.
func (s SchemesData) Append(d *SchemeData) SchemesData {
	if d == nil {
		return s
	}
	found := false
	for _, se := range s {
		if se.SchemeName == d.SchemeName {
			found = true
			break
		}
	}
	if found {
		return s
	}
	return append(s, d)
}

// DedupeByType returns a new SchemesData slice that is deduplicated by scheme
// type.
func (s SchemesData) DedupeByType() SchemesData {
	seen := make(map[string]struct{})
	uniqueSchemes := SchemesData{}
	for _, s := range s {
		if _, ok := seen[s.Type]; !ok {
			seen[s.Type] = struct{}{}
			uniqueSchemes = append(uniqueSchemes, s)
		}
	}

	return uniqueSchemes
}
