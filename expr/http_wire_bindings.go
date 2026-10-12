package expr

import (
	"fmt"
	"slices"
	"strings"

	"github.com/CaliLuke/loom/eval"
)

type (
	// httpWireBindingPlan resolves request sources once, before validation and
	// body inference. Finalization only initializes its projected attributes.
	httpWireBindingPlan struct {
		bindings       []*httpWireBinding
		bodyExclusions map[string]bool
	}

	// httpWireBinding is one physical operation. Basic uses an ordered pair of
	// sources; a scheme sharing the same sources and codec reuses the operation.
	httpWireBinding struct {
		location     string
		name         string
		sourceKind   string
		sources      []string
		codec        string
		schemes      []string
		requirements []int
		ordinary     bool
	}
)

func (e *HTTPEndpointExpr) ensureWireBindings() *httpWireBindingPlan {
	if e.wireBindings != nil {
		return e.wireBindings
	}
	// Scope composition already happened in Prepare. Work on detached mapping
	// occurrences so inference cannot mutate a service or canonical endpoint.
	e.Headers = copyHTTPMapping(e.Headers)
	e.Cookies = copyHTTPMapping(e.Cookies)
	e.Params = copyHTTPMapping(e.Params)
	e.Requirements = e.MethodExpr.EffectiveRequirements()
	e.resolveSessionWireMappings(e.MethodExpr.EffectiveSessionAuths())
	plan := &httpWireBindingPlan{bodyExclusions: make(map[string]bool)}
	classified := make(map[string]bool)
	for index, requirement := range e.Requirements {
		for _, scheme := range requirement.Schemes {
			binding := e.resolveCredentialWireBinding(scheme)
			if binding == nil {
				continue
			}
			binding.schemes = []string{scheme.SchemeName}
			binding.requirements = []int{index}
			plan.add(binding)
			for _, source := range binding.sources {
				if source == "" {
					continue
				}
				if binding.codec != "basic" && binding.sourceKind != "transport session" {
					classified[binding.location+":"+source] = true
				}
				if binding.location != "body" && binding.sourceKind == "payload" {
					plan.bodyExclusions[source] = true
				}
			}
		}
	}
	pathNames := make(map[string]bool)
	for _, route := range e.Routes {
		for _, name := range route.Params() {
			pathNames[name] = true
		}
	}
	for _, mapping := range []struct {
		attributes *MappedAttributeExpr
		location   string
	}{
		{e.Params, "query"}, {e.Headers, "header"}, {e.Cookies, "cookie"},
	} {
		for _, field := range *AsObject(mapping.attributes.Type) {
			name := mapping.attributes.ElemName(field.Name)
			location := mapping.location
			if location == "query" && pathNames[name] {
				location = "path"
			}
			if classified[location+":"+field.Name] || field.Attribute.Meta["loom:transport-only-session-cookie"] != nil {
				continue
			}
			plan.add(&httpWireBinding{location: location, name: name, sourceKind: "payload", sources: []string{field.Name}, codec: "raw", ordinary: true})
		}
	}
	e.wireBindings = plan
	return plan
}

func copyHTTPMapping(mapping *MappedAttributeExpr) *MappedAttributeExpr {
	if mapping == nil {
		return NewEmptyMappedAttributeExpr()
	}
	return DupMappedAtt(mapping)
}

func (e *HTTPEndpointExpr) resolveCredentialWireBinding(scheme *SchemeExpr) *httpWireBinding {
	payload := e.MethodExpr.Payload
	if scheme.Kind == NoKind || payload == nil {
		return nil
	}
	if scheme.Kind == BasicAuthKind {
		scheme.In, scheme.Name = "header", "Authorization"
		return &httpWireBinding{
			location: "header", name: "Authorization", codec: "basic", sourceKind: "payload",
			sources: []string{TaggedAttribute(payload, "security:username"), TaggedAttribute(payload, "security:password")},
		}
	}
	var field string
	sourceKind := "payload"
	switch scheme.Kind {
	case APIKeyKind:
		field = TaggedAttribute(payload, "security:apikey:"+scheme.SchemeName)
	case JWTKind:
		field = TaggedAttribute(payload, "security:token")
	case OAuth2Kind:
		field = TaggedAttribute(payload, "security:accesstoken")
	default:
		return nil
	}
	if field == "" {
		if scheme.Kind == APIKeyKind {
			for _, session := range e.MethodExpr.EffectiveSessionAuths() {
				for _, transport := range session.Transports {
					if transport == nil || transport.Scheme == nil || transport.PayloadOwned() || transport.Kind != SessionCookieTransportKind || transport.Scheme.SchemeName != scheme.SchemeName {
						continue
					}
					field = transport.TransportAttributeName()
					sourceKind = "transport session"
				}
			}
		}
		if field == "" {
			// Method validation owns missing or malformed credential declarations.
			return nil
		}
	}
	scheme.Name, scheme.In = findKey(e, field)
	if scheme.Name == "" {
		attribute := payload.Find(field)
		if attribute == nil {
			return nil
		}
		scheme.Name, scheme.In = "Authorization", "header"
		addHTTPWireMapping(e.Headers, field, scheme.Name, attribute, payload.IsRequired(field))
	}
	codec := "raw"
	if (scheme.Kind == JWTKind || scheme.Kind == OAuth2Kind) && scheme.In == "header" && strings.EqualFold(scheme.Name, "Authorization") {
		codec = "bearer"
	}
	return &httpWireBinding{location: scheme.In, name: scheme.Name, sourceKind: sourceKind, sources: []string{field}, codec: codec}
}

func addHTTPWireMapping(mapping *MappedAttributeExpr, field, name string, attribute *AttributeExpr, required bool) {
	AsObject(mapping.Type).Set(field, DupAtt(attribute))
	mapping.Map(name, field)
	if required {
		if mapping.Validation == nil {
			mapping.Validation = &ValidationExpr{}
		}
		mapping.Validation.AddRequired(field)
	}
}

func (e *HTTPEndpointExpr) resolveSessionWireMappings(sessions []*SessionAuthExpr) {
	for _, session := range sessions {
		for _, transport := range session.Transports {
			if transport == nil || transport.Scheme == nil || transport.Kind != SessionCookieTransportKind {
				continue
			}
			field := transport.TransportAttributeName()
			if name, _ := findKey(e, field); name != "" {
				continue
			}
			attribute := e.MethodExpr.Payload.Find(field)
			if attribute == nil {
				if transport.PayloadOwned() {
					continue
				}
				attribute = &AttributeExpr{Type: String, Meta: MetaExpr{"loom:transport-only-session-cookie": {"true"}}}
			}
			name := transport.HTTPName
			if name == "" {
				name = field
			}
			addHTTPWireMapping(e.Cookies, field, name, attribute, transport.PayloadOwned() && e.MethodExpr.Payload.IsRequired(field))
		}
	}
}

func (p *httpWireBindingPlan) add(binding *httpWireBinding) {
	for _, existing := range p.bindings {
		if existing.sameSlot(binding) && existing.codec == binding.codec && existing.sourceKind == binding.sourceKind && slices.Equal(existing.sources, binding.sources) {
			existing.ordinary = existing.ordinary || binding.ordinary
			for _, index := range binding.requirements {
				if !slices.Contains(existing.requirements, index) {
					existing.requirements = append(existing.requirements, index)
				}
			}
			for _, scheme := range binding.schemes {
				if !slices.Contains(existing.schemes, scheme) {
					existing.schemes = append(existing.schemes, scheme)
				}
			}
			return
		}
	}
	p.bindings = append(p.bindings, binding)
}

func (p *httpWireBindingPlan) validate(owner eval.Expression, errors *eval.ValidationErrors) {
	for index, first := range p.bindings {
		for _, second := range p.bindings[index+1:] {
			if !first.sameSlot(second) || first.disjointAlternative(second) {
				continue
			}
			errors.Add(owner, "HTTP %s %q has conflicting bindings for %s and %s; map one attribute or credential to a different wire name or location", first.location, first.name, first.description(), second.description())
		}
	}
}

func (b *httpWireBinding) sameSlot(other *httpWireBinding) bool {
	if b.location != other.location {
		return false
	}
	if b.location == "header" {
		return strings.EqualFold(b.name, other.name)
	}
	return b.name == other.name
}

func (b *httpWireBinding) disjointAlternative(other *httpWireBinding) bool {
	if b.ordinary || other.ordinary || b.codec == other.codec || b.codec == "raw" || other.codec == "raw" {
		return false
	}
	for _, index := range b.requirements {
		if slices.Contains(other.requirements, index) {
			return false
		}
	}
	for _, source := range b.sources {
		if b.sourceKind == other.sourceKind && source != "" && slices.Contains(other.sources, source) {
			return false
		}
	}
	return true
}

func (b *httpWireBinding) description() string {
	description := fmt.Sprintf("%s attributes %q", b.sourceKind, strings.Join(b.sources, ", "))
	if len(b.schemes) > 0 {
		description += fmt.Sprintf(" (security %s, %s encoding)", strings.Join(b.schemes, ", "), b.codec)
	}
	return description
}

func (r *HTTPResponseExpr) validateWireBindings(errors *eval.ValidationErrors) {
	if r.wireBindings != nil {
		r.wireBindings.validate(r, errors)
		return
	}
	plan := &httpWireBindingPlan{}
	if r.Headers != nil {
		for _, field := range *AsObject(r.Headers.Type) {
			plan.add(&httpWireBinding{location: "header", name: r.Headers.ElemName(field.Name), sourceKind: "result", sources: []string{field.Name}, codec: "raw", ordinary: true})
		}
	}
	// Response cookies already validate both duplicate source and wire names;
	// retain those more specific diagnostics in validateCookies.
	r.wireBindings = plan
	plan.validate(r, errors)
}

func (r *HTTPResponseExpr) prepareErrorWireBindings(source *AttributeExpr) {
	if r.wireBindings != nil || source.Type != ErrorResult || r.Body == nil {
		return
	}
	origin, selected := r.Body.Meta["origin:attribute"]
	if r.Body.Type != Empty && !selected {
		return
	}
	r.Headers = copyHTTPMapping(r.Headers)
	for _, field := range *AsObject(source.Type) {
		if selected && slices.Contains(origin, field.Name) {
			continue
		}
		if _, exists := r.Headers.FindKey(field.Name); exists {
			continue
		}
		addHTTPWireMapping(r.Headers, field.Name, "loom-attribute-"+field.Name, field.Attribute, source.IsRequired(field.Name))
	}
}
