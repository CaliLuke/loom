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

// validateErrors validates the method errors.
func (m *MethodExpr) validateErrors() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	for _, e := range m.Errors {
		if err := e.Validate(); err != nil {
			var verrs *eval.ValidationErrors
			if errors.As(err, &verrs) {
				verr.Merge(verrs)
			}
		}
	}
	verr.Merge(validateErrorTypeDiscriminators(m.effectiveErrors()))
	return verr
}

func validateErrorTypeDiscriminators(errors []*ErrorExpr) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	reported := make([]bool, len(errors))
	for i, e := range errors {
		if reported[i] || !IsObject(e.Type) {
			continue
		}
		for j := i + 1; j < len(errors); j++ {
			if !sameErrorType(e.Type, errors[j].Type) {
				continue
			}
			for k := j; k < len(errors); k++ {
				if sameErrorType(e.Type, errors[k].Type) {
					reported[k] = true
				}
			}
			if !hasErrorNameAttribute(e.AttributeExpr) {
				verr.Add(e, "type %q is used to define multiple errors and must identify the attribute containing the error name with ErrorName", e.Type.Name())
			}
			break
		}
	}
	return verr
}

func sameErrorType(left, right DataType) bool {
	leftUserType, leftIsUserType := left.(UserType)
	rightUserType, rightIsUserType := right.(UserType)
	if leftIsUserType || rightIsUserType {
		return leftIsUserType && rightIsUserType && leftUserType.ID() == rightUserType.ID()
	}
	return left == right
}

func hasErrorNameAttribute(att *AttributeExpr) bool {
	var found bool
	walkAttribute(att, func(_ string, nested *AttributeExpr) error { // nolint: errcheck
		if _, ok := nested.Meta["struct:error:name"]; ok {
			found = true
			return errors.New("struct:error:name found: stop iteration")
		}
		return nil
	})
	return found
}

// validateInterceptors validates the method interceptors.
func (m *MethodExpr) validateInterceptors() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	clientInterceptors := mergeInterceptors(m.ClientInterceptors, m.Service.ClientInterceptors, Root.API.ClientInterceptors)
	for _, i := range clientInterceptors {
		verr.Merge(i.validate(m))
	}
	serverInterceptors := mergeInterceptors(m.ServerInterceptors, m.Service.ServerInterceptors, Root.API.ServerInterceptors)
	for _, i := range serverInterceptors {
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
	m.Payload = finalizeMethodAttribute(m.Payload)
	m.StreamingPayload = finalizeMethodAttribute(m.StreamingPayload)
	m.finalizeMethodResult(&m.StreamingResult)
	m.Result = finalizeMethodResultAttr(m.Result)
	m.finalizeInterceptors()
	m.Errors = m.effectiveErrors()
	m.finalizeErrors()
	if m.hasNoSecurityRequirement() {
		m.Requirements = nil
		m.SessionAuths = nil
		return
	}
	m.inheritSecurityRequirements()
	m.Requirements = mergeRequirements(m.Requirements, sessionRequirements(m.SessionAuths))
}

func (m *MethodExpr) finalizeInterceptors() {
	if m.Service == nil {
		return
	}
	m.ClientInterceptors = mergeInterceptors(m.ClientInterceptors, m.Service.ClientInterceptors, Root.API.ClientInterceptors)
	m.ServerInterceptors = mergeInterceptors(m.ServerInterceptors, m.Service.ServerInterceptors, Root.API.ServerInterceptors)
}

func finalizeMethodAttribute(att *AttributeExpr) *AttributeExpr {
	if att == nil {
		return &AttributeExpr{Type: Empty}
	}
	att.Finalize()
	return att
}

func (m *MethodExpr) finalizeMethodResult(result **AttributeExpr) {
	if *result == nil {
		return
	}
	(*result).Finalize()
	if rt, ok := (*result).Type.(*ResultTypeExpr); ok {
		rt.Finalize()
	}
}

func finalizeMethodResultAttr(att *AttributeExpr) *AttributeExpr {
	if att == nil {
		return &AttributeExpr{Type: Empty}
	}
	att.Finalize()
	if rt, ok := att.Type.(*ResultTypeExpr); ok {
		rt.Finalize()
	}
	return att
}

func (m *MethodExpr) effectiveErrors() []*ErrorExpr {
	errors := append([]*ErrorExpr(nil), m.Errors...)
	if m.Service == nil {
		return errors
	}
	for _, serviceErr := range m.Service.Errors {
		if !methodHasError(errors, serviceErr.Name) {
			errors = append(errors, serviceErr)
		}
	}
	return errors
}

func methodHasError(errors []*ErrorExpr, name string) bool {
	for _, err := range errors {
		if err.Name == name {
			return true
		}
	}
	return false
}

func (m *MethodExpr) finalizeErrors() {
	for _, err := range m.Errors {
		err.Finalize()
	}
}

func (m *MethodExpr) inheritSecurityRequirements() {
	if len(m.Requirements) == 0 {
		m.Requirements = m.inheritedRequirements()
	}
	if len(m.SessionAuths) == 0 {
		m.SessionAuths = m.inheritedSessionAuths()
	}
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
