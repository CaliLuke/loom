package expr

import (
	"fmt"
	"net/url"

	"github.com/CaliLuke/loom/eval"
)

// SchemeKind is a type of security scheme.
type SchemeKind int

const (
	// OAuth2Kind identifies a "OAuth2" security scheme.
	OAuth2Kind SchemeKind = iota + 1
	// BasicAuthKind means "basic" security scheme.
	BasicAuthKind
	// APIKeyKind means "apiKey" security scheme.
	APIKeyKind
	// JWTKind means an "JWT" security scheme, with support for
	// TokenPath and Scopes.
	JWTKind
	// NoKind means to have no security for this endpoint.
	NoKind
)

// SessionTransportKind identifies the transport used by a session auth contract.
type SessionTransportKind int

const (
	// SessionBearerTransportKind identifies a bearer token transport.
	SessionBearerTransportKind SessionTransportKind = iota + 1
	// SessionCookieTransportKind identifies a cookie transport.
	SessionCookieTransportKind
)

// FlowKind is a type of OAuth2 flow.
type FlowKind int

const (
	// AuthorizationCodeFlowKind identifies a OAuth2 authorization code
	// flow.
	AuthorizationCodeFlowKind FlowKind = iota + 1
	// ImplicitFlowKind identifiers a OAuth2 implicit flow.
	ImplicitFlowKind
	// PasswordFlowKind identifies a Resource Owner Password flow.
	PasswordFlowKind
	// ClientCredentialsFlowKind identifies a OAuth Client Credentials flow.
	ClientCredentialsFlowKind
	// DeviceAuthorizationFlowKind identifies an OAuth 2 device authorization flow.
	DeviceAuthorizationFlowKind
)

type (
	// SecurityHolder is an interface that allows expression types to receive
	// security requirements. Types implementing this interface can use the
	// Security() DSL function to add security schemes.
	SecurityHolder interface {
		AddSecurityRequirement(*SecurityExpr)
	}

	// SecurityExpr defines a security requirement.
	SecurityExpr struct {
		// Schemes is the list of security schemes used for this
		// requirement.
		Schemes []*SchemeExpr
		// Scopes list the required scopes if any.
		Scopes []string
	}

	// SessionAuthExpr defines a logical auth contract backed by one or more
	// transport alternatives.
	SessionAuthExpr struct {
		// Name is the session auth contract name.
		Name string
		// Description describes the session auth contract.
		Description string
		// Transports lists the accepted auth transports.
		Transports []*SessionTransportExpr
	}

	// SessionTransportExpr defines one accepted transport for a session auth
	// contract.
	SessionTransportExpr struct {
		// Kind identifies the transport kind.
		Kind SessionTransportKind
		// Scheme is the underlying security scheme used by this transport.
		Scheme *SchemeExpr
		// FieldName is the payload field name associated with the transport.
		FieldName string
		// HTTPName is the inferred HTTP transport element name if any.
		HTTPName string
		// Description describes the transport.
		Description string
	}

	// SchemeExpr defines a security scheme used to authenticate against the
	// method being designed.
	SchemeExpr struct {
		// Kind is the sort of security scheme this object represents.
		Kind SchemeKind
		// SchemeName is the name of the security scheme, e.g. "googAuth",
		// "my_big_token", "jwt".
		SchemeName string
		// Description describes the security scheme e.g. "Google OAuth2"
		Description string
		// In determines the location of the API key, one of "header" or
		// "query".
		In string
		// Name refers to a header or parameter name, based on In's
		// value.
		Name string
		// Scopes lists the Basic, APIKey, JWT or OAuth2 scopes.
		Scopes []*ScopeExpr
		// Flows determine the oauth2 flows supported by this scheme.
		Flows []*FlowExpr
		// Meta is a list of key/value pairs
		Meta MetaExpr
	}

	// FlowExpr describes a specific OAuth2 flow.
	FlowExpr struct {
		// Kind is the kind of flow.
		Kind FlowKind
		// AuthorizationURL to be used for implicit or authorizationCode
		// flows.
		AuthorizationURL string
		// TokenURL to be used for password, clientCredentials or
		// authorizationCode flows.
		TokenURL string
		// RefreshURL to be used for obtaining refresh token.
		RefreshURL string
		// DeviceAuthorizationURL starts a device authorization flow.
		DeviceAuthorizationURL string
	}

	// ScopeExpr defines a security scope.
	ScopeExpr struct {
		// Name of the scope.
		Name string
		// Description is the description of the scope.
		Description string
	}
)

// EvalName returns the generic definition name used in error messages.
func (s *SessionAuthExpr) EvalName() string {
	if s.Name == "" {
		return "unnamed session auth"
	}
	return fmt.Sprintf("session auth %q", s.Name)
}

// SetDescription sets the session auth description.
func (s *SessionAuthExpr) SetDescription(d string) {
	s.Description = d
}

// Validate validates the session auth contract.
func (s *SessionAuthExpr) Validate() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if len(s.Transports) == 0 {
		verr.Add(s, "session auth %q must define at least one transport", s.Name)
		return verr
	}
	seenKinds := make(map[SessionTransportKind]struct{}, len(s.Transports))
	seenFields := make(map[string]struct{}, len(s.Transports))
	for _, transport := range s.Transports {
		if transport == nil {
			continue
		}
		if _, ok := seenKinds[transport.Kind]; ok {
			verr.Add(s, "session auth %q defines duplicate %s transport", s.Name, transport.Kind)
		} else {
			seenKinds[transport.Kind] = struct{}{}
		}
		if transport.PayloadOwned() {
			if _, ok := seenFields[transport.FieldName]; ok {
				verr.Add(s, "session auth %q defines duplicate field name %q", s.Name, transport.FieldName)
			} else {
				seenFields[transport.FieldName] = struct{}{}
			}
		} else if transport.Kind != SessionCookieTransportKind {
			verr.Add(s, "session auth %q defines a %s transport with an empty field name", s.Name, transport.Kind)
		}
		if terr := transport.Validate(); terr != nil {
			for _, err := range terr.Errors {
				verr.Add(s, "%s", err.Error())
			}
		}
	}
	return verr
}

// EvalName returns the generic definition name used in error messages.
func (s *SessionTransportExpr) EvalName() string {
	return fmt.Sprintf("%s transport", s.Kind)
}

// SetDescription sets the transport description.
func (s *SessionTransportExpr) SetDescription(d string) {
	s.Description = d
}

// PayloadOwned returns true when the transport credential is modeled as a
// payload field.
func (s *SessionTransportExpr) PayloadOwned() bool {
	return s != nil && s.FieldName != ""
}

// TransportAttributeName returns the attribute name used for transport
// metadata when the credential is not payload-owned.
func (s *SessionTransportExpr) TransportAttributeName() string {
	if s == nil {
		return ""
	}
	if s.FieldName != "" {
		return s.FieldName
	}
	if s.Scheme != nil && s.Scheme.SchemeName != "" {
		return s.Scheme.SchemeName
	}
	return s.Kind.String()
}

// Validate validates the session transport.
func (s *SessionTransportExpr) Validate() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if s.Scheme == nil {
		verr.Add(s, "%s transport must reference a security scheme", s.Kind)
		return verr
	}
	switch s.Kind {
	case SessionBearerTransportKind:
		if s.FieldName == "" {
			verr.Add(s, "bearer transport must define a payload field name")
		}
		if s.Scheme.Kind != JWTKind && s.Scheme.Kind != OAuth2Kind {
			verr.Add(s, "bearer transport must use a JWT or OAuth2 security scheme")
		}
	case SessionCookieTransportKind:
		if s.Scheme.Kind != APIKeyKind {
			verr.Add(s, "cookie transport must use an API key security scheme")
		}
		if s.FieldName == "" && s.HTTPName == "" {
			verr.Add(s, "transport-only cookie transport must define a cookie name")
		}
	default:
		verr.Add(s, "unknown session transport kind %d", s.Kind)
	}
	return verr
}

// EvalName returns the generic definition name used in error messages.
func (s *SecurityExpr) EvalName() string {
	var suffix string
	if len(s.Schemes) > 0 && len(s.Schemes[0].SchemeName) > 0 {
		suffix = "scheme " + s.Schemes[0].SchemeName
	}
	return "Security" + suffix
}

// DupRequirement creates a copy of the given security requirement.
func DupRequirement(req *SecurityExpr) *SecurityExpr {
	dup := &SecurityExpr{
		Scopes:  req.Scopes,
		Schemes: make([]*SchemeExpr, 0, len(req.Schemes)),
	}
	for _, s := range req.Schemes {
		dup.Schemes = append(dup.Schemes, DupScheme(s))
	}
	return dup
}

// DupScheme creates a copy of the given scheme expression.
func DupScheme(sch *SchemeExpr) *SchemeExpr {
	dup := SchemeExpr{
		Kind:        sch.Kind,
		SchemeName:  sch.SchemeName,
		Description: sch.Description,
		In:          sch.In,
		Scopes:      sch.Scopes,
		Flows:       sch.Flows,
		Meta:        sch.Meta,
	}
	return &dup
}

// Type returns the type of the scheme.
func (s *SchemeExpr) Type() string {
	switch s.Kind {
	case OAuth2Kind:
		return "OAuth2"
	case BasicAuthKind:
		return "BasicAuth"
	case APIKeyKind:
		return "APIKey"
	case JWTKind:
		return "JWT"
	default:
		panic(fmt.Sprintf("unknown scheme kind: %#v", s.Kind)) // bug
	}
}

// EvalName returns the generic definition name used in error messages.
func (s *SchemeExpr) EvalName() string {
	return s.Type() + "Security"
}

// Hash returns a unique hash value for s.
func (s *SchemeExpr) Hash() string {
	if s.SchemeName != "" {
		return s.SchemeName
	}
	return fmt.Sprintf("%s_%s_%s", s.Type(), s.In, s.Name)
}

// Validate ensures that the method payload contains attributes required
// by the scheme.
func (s *SchemeExpr) Validate() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	for _, f := range s.Flows {
		if err := f.Validate(); err != nil {
			verr.Merge(err)
		}
	}
	return verr
}

// EvalName returns the name of the expression used in error messages.
func (f *FlowExpr) EvalName() string {
	return "flow " + f.Type()
}

// Validate ensures that TokenURL and AuthorizationURL are valid URLs.
func (f *FlowExpr) Validate() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if _, err := url.Parse(f.TokenURL); err != nil {
		verr.Add(f, "invalid token URL %q: %s", f.TokenURL, err)
	}
	if _, err := url.Parse(f.AuthorizationURL); err != nil {
		verr.Add(f, "invalid authorization URL %q: %s", f.AuthorizationURL, err)
	}
	if _, err := url.Parse(f.RefreshURL); err != nil {
		verr.Add(f, "invalid refresh URL %q: %s", f.RefreshURL, err)
	}
	if _, err := url.Parse(f.DeviceAuthorizationURL); err != nil {
		verr.Add(f, "invalid device authorization URL %q: %s", f.DeviceAuthorizationURL, err)
	}
	return verr
}

// Type returns the grant type of the OAuth2 grant.
func (f *FlowExpr) Type() string {
	switch f.Kind {
	case AuthorizationCodeFlowKind:
		return "authorization_code"
	case ImplicitFlowKind:
		return "implicit"
	case PasswordFlowKind:
		return "password"
	case ClientCredentialsFlowKind:
		return "client_credentials"
	case DeviceAuthorizationFlowKind:
		return "device_authorization"
	default:
		panic(fmt.Sprintf("unknown flow kind: %#v", f.Kind)) // bug
	}
}

func (k SchemeKind) String() string {
	switch k {
	case BasicAuthKind:
		return "Basic"
	case APIKeyKind:
		return "APIKey"
	case JWTKind:
		return "JWT"
	case OAuth2Kind:
		return "OAuth2"
	case NoKind:
		return "None"
	default:
		panic("unknown kind") // bug
	}
}

func (k SessionTransportKind) String() string {
	switch k {
	case SessionBearerTransportKind:
		return "bearer"
	case SessionCookieTransportKind:
		return "cookie"
	default:
		panic("unknown session transport kind") // bug
	}
}
