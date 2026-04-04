package expr

import (
	"errors"
	"fmt"

	"github.com/CaliLuke/loom/eval"
)

type (
	// StreamKind is a type denoting the kind of stream.
	StreamKind int

	// MethodExpr defines a single method.
	MethodExpr struct {
		// DSLFunc contains the DSL used to initialize the expression.
		eval.DSLFunc
		// Name of method.
		Name string
		// Description of method for consumption by humans.
		Description string
		// Docs points to the method external documentation if any.
		Docs *DocsExpr
		// Payload attribute
		Payload *AttributeExpr
		// Result attribute
		Result *AttributeExpr
		// Errors lists the error responses.
		Errors []*ErrorExpr
		// Requirements contains the security requirements for the
		// method. One requirement is composed of potentially multiple
		// schemes. Incoming requests must validate at least one
		// requirement to be authorized.
		Requirements []*SecurityExpr
		// SessionAuths contains the multi-transport session auth contracts that
		// apply to the method.
		SessionAuths []*SessionAuthExpr
		// ClientInterceptors is the list of client interceptors.
		ClientInterceptors []*InterceptorExpr
		// ServerInterceptors is the list of server interceptors.
		ServerInterceptors []*InterceptorExpr
		// Service that owns method.
		Service *ServiceExpr
		// Meta is an arbitrary set of key/value pairs, see dsl.Meta
		Meta MetaExpr
		// Stream is the kind of stream (none, payload, result, or both)
		// the method defines.
		Stream StreamKind
		// StreamingPayload is the payload sent across the stream.
		StreamingPayload *AttributeExpr
		// StreamingResult is the result sent across the stream when using SSE.
		// When both Result and StreamingResult are defined with different types,
		// the method supports content negotiation between standard HTTP responses
		// (using Result) and SSE streams (using StreamingResult).
		StreamingResult *AttributeExpr
	}
)

const (
	// NoStreamKind represents no payload or result stream in method.
	NoStreamKind StreamKind = iota + 1
	// ClientStreamKind represents client sends a streaming payload to
	// method.
	ClientStreamKind
	// ServerStreamKind represents server sends a streaming result from
	// method.
	ServerStreamKind
	// BidirectionalStreamKind represents client and server sending payload
	// and result respectively via a stream.
	BidirectionalStreamKind
)

// Error returns the error with the given name. It looks up recursively in the
// endpoint then the service and finally the root expression.
func (m *MethodExpr) Error(name string) *ErrorExpr {
	for _, err := range m.Errors {
		if err.Name == name {
			return err
		}
	}
	return m.Service.Error(name)
}

// EvalName returns the generic expression name used in error messages.
func (m *MethodExpr) EvalName() string {
	var prefix, suffix string
	if m.Name != "" {
		suffix = fmt.Sprintf("method %#v", m.Name)
	} else {
		suffix = "unnamed method"
	}
	if m.Service != nil {
		prefix = m.Service.EvalName() + " "
	}
	return prefix + suffix
}

// Prepare makes sure the payload and result types are initialized (to the Empty
// type if nil) and merges the method interceptors with the API and service level
// interceptors.
func (m *MethodExpr) Prepare() {
	if m.Stream == 0 {
		m.Stream = NoStreamKind
	}
	if m.Payload == nil {
		m.Payload = &AttributeExpr{Type: Empty}
	}
	if m.StreamingPayload == nil {
		m.StreamingPayload = &AttributeExpr{Type: Empty}
	}

	// Backward compatibility: if StreamingResult is set but Result is not,
	// copy StreamingResult to Result so existing code generation continues to work
	if m.StreamingResult != nil && m.Result == nil {
		m.Result = m.StreamingResult
	}

	// Initialize Result to Empty if still nil
	if m.Result == nil {
		m.Result = &AttributeExpr{Type: Empty}
	}

	// If this is a streaming method without explicit StreamingResult,
	// use Result for backward compatibility
	if m.StreamingResult == nil && m.Stream != NoStreamKind {
		m.StreamingResult = m.Result
	}
}

// Validate validates the method payloads, results, errors, security
// requirements, and interceptors.
func (m *MethodExpr) Validate() error {
	verr := new(eval.ValidationErrors)
	injectErr := m.injectSessionAuthPayloadFields()
	verr.Merge(injectErr)
	verr.Merge(m.Payload.Validate("payload", m))
	verr.Merge(m.StreamingPayload.Validate("streaming_payload", m))
	verr.Merge(m.Result.Validate("result", m))
	if m.StreamingResult != nil && m.StreamingResult != m.Result {
		verr.Merge(m.StreamingResult.Validate("streaming_result", m))
	}
	if injectErr == nil || len(injectErr.Errors) == 0 {
		verr.Merge(m.validateRequirements())
	}
	verr.Merge(m.validateErrors())
	verr.Merge(m.validateInterceptors())
	return verr
}

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

// validateErrors validates the method errors.
func (m *MethodExpr) validateErrors() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	for i, e := range m.Errors {
		if err := e.Validate(); err != nil {
			var verrs *eval.ValidationErrors
			if errors.As(err, &verrs) {
				verr.Merge(verrs)
			}
		}
		for j, e2 := range m.Errors {
			// If an object type is used to define more than one errors validate the
			// presence of struct:error:name meta in the object type.
			if i != j && e.Type == e2.Type && IsObject(e.Type) {
				var found bool
				walkAttribute(e.AttributeExpr, func(_ string, att *AttributeExpr) error { // nolint: errcheck
					if _, ok := att.Meta["struct:error:name"]; ok {
						found = true
						return fmt.Errorf("struct:error:name found: stop iteration")
					}
					return nil
				})
				if !found {
					verr.Add(e, "type %q is used to define multiple errors and must identify the attribute containing the error name with ErrorName", e.Type.Name())
					break
				}
			}
		}
	}
	return verr
}

// validateInterceptors validates the method interceptors.
func (m *MethodExpr) validateInterceptors() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	m.ClientInterceptors = mergeInterceptors(m.ClientInterceptors, m.Service.ClientInterceptors, Root.API.ClientInterceptors)
	for _, i := range m.ClientInterceptors {
		verr.Merge(i.validate(m))
	}
	m.ServerInterceptors = mergeInterceptors(m.ServerInterceptors, m.Service.ServerInterceptors, Root.API.ServerInterceptors)
	for _, i := range m.ServerInterceptors {
		verr.Merge(i.validate(m))
	}
	return verr
}

// mergeInterceptors merges interceptors from different levels (method, service, API)
// while avoiding duplicates. The order of precedence is: method > service > API.
func mergeInterceptors(methodLevel, serviceLevel, apiLevel []*InterceptorExpr) []*InterceptorExpr {
	existing := make(map[string]struct{})
	result := make([]*InterceptorExpr, 0, len(methodLevel)+len(serviceLevel)+len(apiLevel))

	for _, i := range methodLevel {
		existing[i.Name] = struct{}{}
		result = append(result, i)
	}
	for _, i := range serviceLevel {
		if _, ok := existing[i.Name]; !ok {
			result = append(result, i)
			existing[i.Name] = struct{}{}
		}
	}
	for _, i := range apiLevel {
		if _, ok := existing[i.Name]; !ok {
			result = append(result, i)
		}
	}
	return result
}

// hasTag is a helper function that traverses the given attribute and all its
// bases recursively looking for an attribute with the given tag meta. This
// recursion is only needed for attributes that have not been finalized yet.
func hasTag(p *AttributeExpr, tag string) bool {
	if p.HasTag(tag) {
		return true
	}
	for _, base := range p.Bases {
		ut, ok := base.(UserType)
		if !ok {
			continue
		}
		if hasTag(ut.Attribute(), tag) {
			return true
		}
	}
	if ut, ok := p.Type.(UserType); ok {
		return hasTag(ut.Attribute(), tag)
	}
	return false
}

// hasTag is a helper function that traverses the given attribute and all its
// bases recursively looking for an attribute with the given tag meta prefix. This
// recursion is only needed for attributes that have not been finalized yet.
func hasTagPrefix(p *AttributeExpr, prefix string) bool {
	if p.HasTagPrefix(prefix) {
		return true
	}
	for _, base := range p.Bases {
		ut, ok := base.(UserType)
		if !ok {
			continue
		}
		if hasTagPrefix(ut.Attribute(), prefix) {
			return true
		}
	}
	if ut, ok := p.Type.(UserType); ok {
		return hasTagPrefix(ut.Attribute(), prefix)
	}
	return false
}

// Finalize makes sure the method payload and result types are set. It also
// projects the result if it is a result type and a view is explicitly set in
// the design or a result type having at most one view.
func (m *MethodExpr) Finalize() {
	if m.Payload == nil {
		m.Payload = &AttributeExpr{Type: Empty}
	} else {
		m.Payload.Finalize()
	}
	if m.StreamingPayload == nil {
		m.StreamingPayload = &AttributeExpr{Type: Empty}
	} else {
		m.StreamingPayload.Finalize()
	}

	// Handle StreamingResult finalization
	if m.StreamingResult != nil {
		m.StreamingResult.Finalize()
		if rt, ok := m.StreamingResult.Type.(*ResultTypeExpr); ok {
			rt.Finalize()
		}
	}

	// Handle Result finalization (may be same as StreamingResult for backward compat)
	if m.Result == nil {
		m.Result = &AttributeExpr{Type: Empty}
	} else {
		m.Result.Finalize()
		if rt, ok := m.Result.Type.(*ResultTypeExpr); ok {
			rt.Finalize()
		}
	}
	if m.Service != nil {
		for _, e := range m.Service.Errors {
			found := false
			for _, f := range m.Errors {
				if e.Name == f.Name {
					found = true
					break
				}
			}
			if !found {
				m.Errors = append(m.Errors, e)
			}
		}
	}
	for _, e := range m.Errors {
		e.Finalize()
	}

	// Inherit security requirements
	noreq := false
loop:
	for _, r := range m.Requirements {
		// Handle special case of no security
		for _, s := range r.Schemes {
			if s.Kind == NoKind {
				noreq = true
				break loop
			}
		}
	}
	if noreq {
		m.Requirements = nil
		m.SessionAuths = nil
		return
	}
	if len(m.Requirements) == 0 {
		m.Requirements = m.inheritedRequirements()
	}
	if len(m.SessionAuths) == 0 {
		m.SessionAuths = m.inheritedSessionAuths()
	}
	m.Requirements = mergeRequirements(m.Requirements, sessionRequirements(m.SessionAuths))
}

// IsStreaming determines whether the method streams payload or result.
func (m *MethodExpr) IsStreaming() bool {
	return m.IsPayloadStreaming() || m.IsResultStreaming()
}

// IsPayloadStreaming determines whether the method streams payload.
func (m *MethodExpr) IsPayloadStreaming() bool {
	return m.Stream == ClientStreamKind || m.Stream == BidirectionalStreamKind
}

// IsResultStreaming determines whether the method streams payload.
func (m *MethodExpr) IsResultStreaming() bool {
	return m.Stream == ServerStreamKind || m.Stream == BidirectionalStreamKind
}

// HasMixedResults returns true if the method has both Result and StreamingResult
// defined with different types, indicating support for content negotiation.
func (m *MethodExpr) HasMixedResults() bool {
	return m.Result != nil && m.StreamingResult != nil && m.Result != m.StreamingResult
}

// helper function that duplicates just enough of a security expression so that
// its scheme names can be overridden without affecting the original.
func copyReqs(reqs []*SecurityExpr) []*SecurityExpr {
	reqs2 := make([]*SecurityExpr, len(reqs))
	for i, req := range reqs {
		req2 := &SecurityExpr{Scopes: req.Scopes}
		schs := make([]*SchemeExpr, len(req.Schemes))
		for j, sch := range req.Schemes {
			schs[j] = &SchemeExpr{
				Kind:        sch.Kind,
				SchemeName:  sch.SchemeName,
				Description: sch.Description,
				In:          sch.In,
				Name:        sch.Name,
				Scopes:      sch.Scopes,
				Flows:       sch.Flows,
				Meta:        sch.Meta,
			}
		}
		req2.Schemes = schs
		reqs2[i] = req2
	}
	return reqs2
}

func copySessionAuths(sessionAuths []*SessionAuthExpr) []*SessionAuthExpr {
	if len(sessionAuths) == 0 {
		return nil
	}
	dups := make([]*SessionAuthExpr, len(sessionAuths))
	copy(dups, sessionAuths)
	return dups
}

// EffectiveRequirements returns the method security requirements after applying
// no-security overrides, inherited service/API requirements, and derived
// session-auth transports.
func (m *MethodExpr) EffectiveRequirements() []*SecurityExpr {
	return m.validationRequirements()
}

func (m *MethodExpr) validationRequirements() []*SecurityExpr {
	if m.hasNoSecurityRequirement() {
		return nil
	}
	requirements := copyReqs(m.Requirements)
	if len(requirements) == 0 {
		requirements = m.inheritedRequirements()
	}
	sessionAuths := m.validationSessionAuths()
	return mergeRequirements(requirements, sessionRequirements(sessionAuths))
}

// EffectiveSessionAuths returns the session auth contracts that apply to the
// method after honoring no-security overrides and inherited service/API
// session-auth declarations.
func (m *MethodExpr) EffectiveSessionAuths() []*SessionAuthExpr {
	return m.validationSessionAuths()
}

func (m *MethodExpr) validationSessionAuths() []*SessionAuthExpr {
	if m.hasNoSecurityRequirement() {
		return nil
	}
	sessionAuths := copySessionAuths(m.SessionAuths)
	if len(sessionAuths) == 0 {
		sessionAuths = m.inheritedSessionAuths()
	}
	return sessionAuths
}

func (m *MethodExpr) inheritedRequirements() []*SecurityExpr {
	switch {
	case m.Service != nil && len(m.Service.Requirements) > 0:
		return copyReqs(m.Service.Requirements)
	case Root.API != nil && len(Root.API.Requirements) > 0:
		return copyReqs(Root.API.Requirements)
	default:
		return nil
	}
}

func (m *MethodExpr) inheritedSessionAuths() []*SessionAuthExpr {
	switch {
	case m.Service != nil && len(m.Service.SessionAuths) > 0:
		return copySessionAuths(m.Service.SessionAuths)
	case Root.API != nil && len(Root.API.SessionAuths) > 0:
		return copySessionAuths(Root.API.SessionAuths)
	default:
		return nil
	}
}

func sessionRequirements(sessionAuths []*SessionAuthExpr) []*SecurityExpr {
	if len(sessionAuths) == 0 {
		return nil
	}
	reqs := make([]*SecurityExpr, 0, len(sessionAuths))
	for _, sessionAuth := range sessionAuths {
		for _, transport := range sessionAuth.Transports {
			if transport == nil || transport.Scheme == nil {
				continue
			}
			reqs = append(reqs, &SecurityExpr{
				Schemes: []*SchemeExpr{DupScheme(transport.Scheme)},
			})
		}
	}
	return reqs
}

func mergeRequirements(existing []*SecurityExpr, derived []*SecurityExpr) []*SecurityExpr {
	if len(derived) == 0 {
		return existing
	}
	merged := make([]*SecurityExpr, 0, len(existing)+len(derived))
	merged = append(merged, existing...)
	for _, req := range derived {
		if !containsRequirement(merged, req) {
			merged = append(merged, req)
		}
	}
	return merged
}

func containsRequirement(requirements []*SecurityExpr, candidate *SecurityExpr) bool {
	for _, req := range requirements {
		if requirementEqual(req, candidate) {
			return true
		}
	}
	return false
}

func requirementEqual(left *SecurityExpr, right *SecurityExpr) bool {
	if len(left.Scopes) != len(right.Scopes) || len(left.Schemes) != len(right.Schemes) {
		return false
	}
	for i, scope := range left.Scopes {
		if right.Scopes[i] != scope {
			return false
		}
	}
	for i, scheme := range left.Schemes {
		other := right.Schemes[i]
		if scheme.Kind != other.Kind || scheme.SchemeName != other.SchemeName || scheme.In != other.In || scheme.Name != other.Name {
			return false
		}
	}
	return true
}

func (m *MethodExpr) hasNoSecurityRequirement() bool {
	if m != nil && m.Meta != nil {
		if _, ok := m.Meta["security:no"]; ok {
			return true
		}
	}
	for _, req := range m.Requirements {
		for _, scheme := range req.Schemes {
			if scheme.Kind == NoKind {
				return true
			}
		}
	}
	return false
}

func (m *MethodExpr) injectSessionAuthPayloadFields() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	sessionAuths := m.validationSessionAuths()
	if len(sessionAuths) == 0 {
		return verr
	}
	if m.Payload == nil || m.Payload.Type == Empty {
		m.Payload = &AttributeExpr{Type: &Object{}}
	}
	obj := AsObject(m.Payload.Type)
	if obj == nil {
		verr.Add(m, "payload of method %q of service %q must be an object to inject session auth fields", m.Name, m.Service.Name)
		return verr
	}
	if ut, ok := m.Payload.Type.(UserType); ok {
		dupped := Dup(ut).(UserType)
		if renamer, ok := dupped.(interface{ Rename(string) }); ok {
			renamer.Rename(ut.Name() + "_" + m.Name + "_SessionPayload")
		}
		m.Payload.Type = dupped
		obj = AsObject(m.Payload.Type)
	}
	for _, sessionAuth := range sessionAuths {
		for _, transport := range sessionAuth.Transports {
			verr.Merge(m.injectSessionTransportField(sessionAuth, transport, obj))
		}
	}
	return verr
}

func (m *MethodExpr) injectSessionTransportField(sessionAuth *SessionAuthExpr, transport *SessionTransportExpr, obj *Object) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if transport == nil || transport.Scheme == nil {
		return verr
	}
	if !transport.PayloadOwned() {
		return verr
	}
	if terr := transport.Validate(); terr != nil && len(terr.Errors) > 0 {
		return verr
	}
	if existing := obj.Attribute(transport.FieldName); existing != nil {
		if sessionTransportFieldCompatible(existing, transport) {
			return verr
		}
		verr.Add(
			m,
			"payload of method %q of service %q defines field %q which conflicts with session auth %q %s transport",
			m.Name,
			m.Service.Name,
			transport.FieldName,
			sessionAuth.Name,
			transport.Kind,
		)
		return verr
	}
	att := &AttributeExpr{Type: String}
	switch transport.Kind {
	case SessionBearerTransportKind:
		if transport.Scheme.Kind == OAuth2Kind {
			att.AddMeta("security:accesstoken")
		} else {
			att.AddMeta("security:token")
		}
	case SessionCookieTransportKind:
		att.AddMeta("security:apikey:"+transport.Scheme.SchemeName, transport.Scheme.SchemeName)
	}
	obj.Set(transport.FieldName, att)
	return verr
}

func sessionTransportFieldCompatible(att *AttributeExpr, transport *SessionTransportExpr) bool {
	if att == nil || att.Type != String {
		return false
	}
	switch transport.Kind {
	case SessionBearerTransportKind:
		if transport.Scheme.Kind == OAuth2Kind {
			_, ok := att.Meta["security:accesstoken"]
			return ok
		}
		_, ok := att.Meta["security:token"]
		return ok
	case SessionCookieTransportKind:
		_, ok := att.Meta["security:apikey:"+transport.Scheme.SchemeName]
		return ok
	default:
		return false
	}
}
