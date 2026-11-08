package expr

import (
	"slices"

	"github.com/CaliLuke/loom/eval"
)

// httpResponseBindingPlan retains one response occurrence's native source and
// transport operations. It is captured during validation, before any finalizer
// changes physical storage. Request plans never contain these operations.
type httpResponseBindingPlan struct {
	httpWireBindingPlan
	source         *AttributeExpr
	required       map[string][]string
	members        map[string]bool
	branchOnly     map[string]bool
	domain         []int
	domainCaptured bool
	authoredBody   *AttributeExpr
}

// RequiredFields returns a detached snapshot of the members required by this
// response branch in view. An empty view selects the ordinary unviewed source.
func (r *HTTPResponseExpr) RequiredFields(view string) []string {
	if r.bindings == nil {
		return nil
	}
	return slices.Clone(r.bindings.required[view])
}

// StatusDomain returns a detached ascending snapshot of the response's captured
// status ownership. Final responses use their runtime domain; informational
// responses use their emission domain, excluding automatic 100 and upgrade 101.
// It is nil before validation and for named JSON-RPC errors. Invalid designs do
// not provide a usable analysis contract. Field constraints remain separate.
func (r *HTTPResponseExpr) StatusDomain() []int {
	if r.bindings == nil {
		return nil
	}
	return slices.Clone(r.bindings.domain)
}

// AuthoredBody returns the original explicit Body occurrence retained before
// finalization, or nil before capture or when no Body was authored. Body(Empty)
// remains a nonnil occurrence. The returned provenance is shared and read-only;
// callers must not mutate it or treat it as a new example authority.
func (r *HTTPResponseExpr) AuthoredBody() *AttributeExpr {
	if r.bindings == nil {
		return nil
	}
	return r.bindings.authoredBody
}

func (e *HTTPEndpointExpr) captureResponseBindings() {
	for _, r := range e.Responses {
		r.captureBindings(r.responseSource(e.MethodExpr.Result))
	}
	for _, er := range e.HTTPErrors {
		if source := er.designError(); source != nil {
			er.Response.prepareErrorWireBindings(source.AttributeExpr)
			er.Response.captureBindings(source.AttributeExpr)
		}
	}
	for _, info := range e.Informational {
		info.captureBindings(info.Value)
	}
	if !e.StatusSelected() || len(e.Responses) < 2 {
		return
	}
	counts := make(map[string]int)
	for _, r := range e.Responses {
		for name := range r.bindings.members {
			counts[name]++
		}
	}
	for _, r := range e.Responses {
		for name, count := range counts {
			if count < len(e.Responses) {
				r.bindings.branchOnly[name] = true
			}
		}
	}
}

func (r *HTTPResponseExpr) responseSource(fallback *AttributeExpr) *AttributeExpr {
	if r.bindings != nil && r.bindings.source != nil {
		return r.bindings.source
	}
	if r.Handshake != nil {
		return r.Handshake
	}
	return fallback
}

func (r *HTTPResponseExpr) captureBindings(source *AttributeExpr) {
	if r.bindings != nil && r.bindings.source != nil {
		return
	}
	plan := &httpResponseBindingPlan{
		httpWireBindingPlan: httpWireBindingPlan{bodyExclusions: make(map[string]bool)},
		authoredBody:        r.Body,
		source:              source, required: make(map[string][]string), members: make(map[string]bool), branchOnly: make(map[string]bool),
	}
	r.captureMappedSources(plan, source)
	r.bindings = plan
	r.captureBodyRequirements(plan, source)
}

func (r *HTTPResponseExpr) captureMappedSources(plan *httpResponseBindingPlan, source *AttributeExpr) {
	if r.Headers != nil {
		for _, field := range *AsObject(r.Headers.Type) {
			plan.add(&httpWireBinding{location: "header", name: r.Headers.ElemName(field.Name), sourceKind: "result", sources: []string{field.Name}, codec: "raw", ordinary: true})
			plan.bodyExclusions[AttributeName(field.Name)] = true
			plan.members[AttributeName(field.Name)] = true
		}
	}
	for _, cookie := range r.Cookies {
		if cookie != nil && !cookie.IsEmpty() {
			name := cookie.AttributeName()
			plan.add(&httpWireBinding{location: "cookie", name: cookie.HTTPName(), sourceKind: "result", sources: []string{name}, codec: "raw", ordinary: true})
			plan.bodyExclusions[name] = true
			plan.members[name] = true
		}
	}
	if r.StatusField != "" {
		name := AttributeName(r.StatusField)
		plan.add(&httpWireBinding{location: "status", name: "status", sourceKind: "result", sources: []string{name}, codec: "raw", ordinary: true})
		plan.bodyExclusions[name] = true
		plan.members[name] = true
	}
	if r.ErrorNameFromStatus && source != nil {
		name := TaggedAttribute(source, "struct:error:name")
		plan.add(&httpWireBinding{location: "error-name", sourceKind: "result", sources: []string{name}, codec: "identity", ordinary: true})
		plan.bodyExclusions[name] = true
		plan.members[name] = true
	}
	if r.Tag[0] != "" {
		plan.members[AttributeName(r.Tag[0])] = true
	}
}

func (r *HTTPResponseExpr) captureBodyRequirements(plan *httpResponseBindingPlan, source *AttributeExpr) {
	body, _ := responseBodySource(source, r)
	if body != nil {
		if selected, ok := body.Meta.Last("origin:attribute"); ok {
			plan.members[AttributeName(selected)] = true
			plan.add(&httpWireBinding{location: "body", sourceKind: "result", sources: []string{AttributeName(selected)}, codec: "body", ordinary: true})
		} else if obj := AsObject(body.Type); obj != nil {
			for _, field := range *obj {
				plan.members[AttributeName(field.Name)] = true
				plan.add(&httpWireBinding{location: "body", name: ElementName(field.Name), sourceKind: "result", sources: []string{AttributeName(field.Name)}, codec: "body", ordinary: true})
			}
		}
	}
	if source == nil {
		return
	}
	requirements := r.wireRequired(body)
	plan.captureRequired("", source, requirements)
	if result, ok := source.Type.(*ResultTypeExpr); ok {
		for _, view := range result.Views {
			if selected, fixed := source.Meta.Last(ViewMetaKey); fixed && selected != view.Name {
				continue
			}
			if projected, err := Project(result, view.Name); err == nil {
				plan.captureRequired(view.Name, projected.AttributeExpr, requirements)
			}
		}
	}
}

// wireRequired snapshots local transport requirements in native field identity.
// A selected body's nested requirements belong to its body value, not the
// enclosing carrier, and remain on that body's attribute tree.
func (r *HTTPResponseExpr) wireRequired(body *AttributeExpr) []string {
	var required []string
	if r.Headers != nil {
		required = append(required, r.Headers.AllRequired()...)
	}
	for _, cookie := range r.Cookies {
		if !cookie.IsEmpty() {
			required = append(required, cookie.AllRequired()...)
		}
	}
	if body != nil {
		if _, selected := body.Meta.Last("origin:attribute"); !selected {
			required = append(required, body.AllRequired()...)
		}
	}
	return required
}

func (p *httpResponseBindingPlan) captureRequired(view string, source *AttributeExpr, wireRequired []string) {
	for _, keys := range [][]string{source.AllRequired(), wireRequired} {
		for _, key := range keys {
			name := AttributeName(key)
			_, field := source.FindAttribute(name)
			if p.members[name] && field != nil && !slices.Contains(p.required[view], name) {
				p.required[view] = append(p.required[view], name)
			}
		}
	}
}

func (r *HTTPResponseExpr) validateWireBindings(errors *eval.ValidationErrors) {
	if r.bindings == nil {
		r.captureBindings(nil)
	}
	// Cookie source/wire duplicate diagnostics remain in validateCookies.
	// This transient projection validates headers from the same captured plan.
	var headers []*httpWireBinding
	for _, binding := range r.bindings.bindings {
		if binding.location == "header" {
			headers = append(headers, binding)
		}
	}
	checks := httpWireBindingPlan{bindings: headers}
	checks.validate(r, errors)
}

func (r *HTTPResponseExpr) prepareErrorWireBindings(source *AttributeExpr) {
	if r.bindings != nil || source.Type != ErrorResult || r.Body == nil {
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

func (r *HTTPResponseExpr) excludeBodyFields(body *MappedAttributeExpr) {
	if r.bindings == nil {
		r.captureBindings(nil)
	}
	for _, binding := range r.bindings.bindings {
		if binding.location == "body" {
			continue
		}
		for _, name := range binding.sources {
			if r.bindings.bodyExclusions[AttributeName(name)] {
				removeAttribute(body, name)
			}
		}
	}
}

func (r *HTTPResponseExpr) validateResponseReconstruction(source, body *AttributeExpr, bindings []string, label string, errors *eval.ValidationErrors) {
	for _, name := range missingReconstructionFields(source, body, bindings) {
		if r.bindings != nil && r.bindings.branchOnly[AttributeName(name)] && !r.bindings.members[AttributeName(name)] {
			continue
		}
		errors.Add(r, "required %s attribute %q has no reconstruction source in the runtime body or transport bindings", label, name)
	}
}

func (e *HTTPEndpointExpr) finalizeResponseStorage() {
	if !e.StatusSelected() || len(e.Responses) < 2 {
		return
	}
	optional := make(map[string]bool)
	for _, r := range e.Responses {
		if r.bindings != nil {
			for name := range r.bindings.branchOnly {
				optional[name] = true
			}
		}
	}
	if len(optional) == 0 {
		return
	}
	if e.SSE == nil {
		e.MethodExpr.Result = responseStorageOccurrence(e.MethodExpr.Result, optional)
		return
	}
	for _, r := range e.Responses {
		if r.Handshake != nil {
			r.Handshake = responseStorageOccurrence(r.Handshake, optional)
		}
	}
}

// responseStorageOccurrence detaches the root declaration/constraint ancestry
// before relaxing physical branch storage. Child contracts remain unchanged;
// the captured branch plans retain the original semantic requiredness.
func responseStorageOccurrence(source *AttributeExpr, optional map[string]bool) *AttributeExpr {
	copies := make(map[*AttributeExpr]*AttributeExpr)
	var detach func(*AttributeExpr) *AttributeExpr
	detach = func(attribute *AttributeExpr) *AttributeExpr {
		if attribute == nil {
			return nil
		}
		if copy := copies[attribute]; copy != nil {
			return copy
		}
		copy := DupAtt(attribute)
		copies[attribute] = copy
		if copy.Validation != nil {
			copy.Validation.Required = slices.DeleteFunc(copy.Validation.Required, func(name string) bool {
				return optional[AttributeName(name)]
			})
		}
		if original, ok := attribute.Type.(UserType); ok {
			owned := copy.Type.(UserType)
			*owned.Attribute() = *detach(original.Attribute())
		}
		copy.constraintLocal = detach(attribute.constraintLocal)
		copy.constraintParents = make([]*AttributeExpr, len(attribute.constraintParents))
		for i, parent := range attribute.constraintParents {
			copy.constraintParents[i] = detach(parent)
		}
		copy.References = make([]DataType, len(attribute.References))
		for i, reference := range attribute.References {
			copy.References[i] = detach(&AttributeExpr{Type: reference}).Type
		}
		copy.Bases = make([]DataType, len(attribute.Bases))
		for i, base := range attribute.Bases {
			copy.Bases[i] = detach(&AttributeExpr{Type: base}).Type
		}
		return copy
	}
	return detach(source)
}
