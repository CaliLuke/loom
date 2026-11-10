package expr

import (
	"github.com/CaliLuke/loom/eval"
)

// StatusSelected reports whether success branches select their status from a
// shared native StatusField rather than a singleton literal or authored Tags.
func (e *HTTPEndpointExpr) StatusSelected() bool {
	if e.IsJSONRPC() || e.NoSuccess {
		return false
	}
	for _, r := range e.Responses {
		if r.Tag[0] != "" {
			return false
		}
	}
	if len(e.Responses) > 1 {
		return true
	}
	return len(e.Responses) == 1 && !e.Responses[0].Status.IsExact()
}

// InformationalStatusKeys returns a fresh slice of every informational selector
// in declaration order, independently from observer registration. These keys
// select observations; an informational emission domain excludes 100 and 101.
func (e *HTTPEndpointExpr) InformationalStatusKeys() []HTTPStatusSelector {
	keys := make([]HTTPStatusSelector, 0, len(e.Informational))
	for _, info := range e.Informational {
		keys = append(keys, info.Status)
	}
	return keys
}

// FinalStatusKeys returns a fresh slice of all declared final HTTP selectors in
// success-then-error order, including repeated declarations. Named JSON-RPC error
// codes and the transport's synthetic WebSocket upgrade are not HTTP declarations.
func (e *HTTPEndpointExpr) FinalStatusKeys() []HTTPStatusSelector {
	keys := make([]HTTPStatusSelector, 0, len(e.Responses)+len(e.HTTPErrors))
	for _, r := range e.Responses {
		keys = append(keys, r.Status)
	}
	for _, er := range e.HTTPErrors {
		if er.Response.JSONRPCError == nil {
			keys = append(keys, er.Response.Status)
		}
	}
	return keys
}

func (e *HTTPEndpointExpr) validateResponseSelection(verr *eval.ValidationErrors) {
	e.validateNoSuccess(verr)
	success := make(map[HTTPStatusSelector]bool)
	field := ""
	var handshake DataType
	for _, r := range e.Responses {
		r.captureStatusDomain(e, r.responseSource(e.MethodExpr.Result))
		if success[r.Status] {
			verr.Add(r, "Multiple response definitions with status code %s", r.Status.OpenAPIKey())
		}
		success[r.Status] = true
		if e.StatusSelected() {
			if field != "" && field != r.StatusField {
				verr.Add(r, "status-selected success responses must bind the same StatusField")
			}
			field = r.StatusField
		}
		if r.Handshake != nil {
			if handshake != nil && !sameNativeType(handshake, r.Handshake.Type) {
				verr.Add(r, "SSE success responses must use the same native handshake type")
			}
			handshake = r.Handshake.Type
		}
		e.validateSuccessMode(r, verr)
		r.validateStatusSource(e, r.responseSource(e.MethodExpr.Result), e.StatusSelected(), verr)
	}
	for _, er := range e.HTTPErrors {
		r := er.Response
		if success[r.Status] && r.JSONRPCError == nil {
			verr.Add(r, "response key %s cannot belong to both success and error responses", r.Status.OpenAPIKey())
		}
		if source := er.designError(); source != nil {
			r.captureStatusDomain(e, source.AttributeExpr)
			if !e.IsJSONRPC() {
				r.validateStatusSource(e, source.AttributeExpr, false, verr)
			}
		}
	}
	e.validateInformational(verr)
}

func (e *HTTPEndpointExpr) validateSuccessMode(r *HTTPResponseExpr, verr *eval.ValidationErrors) {
	if r.Handshake != nil && (e.SSE == nil || e.IsJSONRPC()) {
		verr.Add(r, "typed handshake is only valid for HTTP SSE responses")
	}
	if e.SSE != nil && r.Tag[0] != "" {
		verr.Add(r, "SSE responses cannot use Tag")
	}
	if r.Tag[0] != "" {
		for _, other := range e.Responses {
			if !other.Status.IsExact() {
				verr.Add(r, "Tag cannot be combined with non-exact success selectors")
				break
			}
		}
	}

	if e.SSE != nil && !e.IsJSONRPC() && r.Handshake == nil && (r.StatusField != "" || !r.Headers.IsEmpty() || len(r.Cookies) > 0) {
		verr.Add(r, "SSE status, headers and cookies require a separate typed handshake value")
	}
	if r.Handshake != nil && r.Body != nil && r.Body.Type != Empty {
		verr.Add(r, "SSE handshake cannot define a response body")
	}
}

func (e *HTTPEndpointExpr) validateNoSuccess(verr *eval.ValidationErrors) {
	if e.NoSuccess {
		if e.IsJSONRPC() {
			verr.Add(e, "NoSuccessResponse is not valid for JSON-RPC")
		}
		if e.MethodExpr.Result.Type != Empty || e.MethodExpr.IsStreaming() {
			verr.Add(e, "NoSuccessResponse requires an empty result and a non-streaming method")
		}
		if len(e.Responses) != 0 || len(e.HTTPErrors) == 0 {
			verr.Add(e, "NoSuccessResponse requires errors and no success response")
		}
	}
}

func (e *HTTPEndpointExpr) validateInformational(verr *eval.ValidationErrors) {
	seen := make(map[string]bool)
	keys := make(map[HTTPStatusSelector]bool)
	for _, info := range e.Informational {
		info.captureStatusDomain(e, info.Value)
		if seen[info.Name] || info.Name == "" {
			verr.Add(info, "Informational name %q must be nonempty and unique", info.Name)
		}
		seen[info.Name] = true
		if keys[info.Status] {
			verr.Add(info, "Informational key %s is duplicated", info.Status.OpenAPIKey())
		}
		keys[info.Status] = true
		if e.IsJSONRPC() || info.Status.Phase() != PhaseInformational {
			verr.Add(info, "Informational requires HTTP 100, 102..199 or 1XX; 101 is a protocol upgrade")
		}
		validateNullableHTTPResponse(info.HTTPResponseExpr, verr)
		info.validateWireBindings(verr)
		info.validateHeaders(e, newHTTPResponseResultType(info.Value), verr)
		info.validateStatusSource(e, info.Value, false, verr)
		validateReconstruction(info.Value, nil, responseReconstructionBindings(info.HTTPResponseExpr), "informational value", info, verr)
	}
}

func (r *HTTPResponseExpr) validateStatusSource(e *HTTPEndpointExpr, source *AttributeExpr, statusMode bool, verr *eval.ValidationErrors) {
	if e.IsJSONRPC() {
		if r.StatusField != "" || !r.Status.IsExact() && r.JSONRPCError == nil {
			verr.Add(r, "StatusField and non-exact HTTP selectors are not valid for JSON-RPC")
		}
		return
	}
	if !r.Status.IsExact() && (e.FileResponse || e.MethodExpr.IsStreaming() && e.SSE == nil) {
		verr.Add(r, "non-exact response selectors are not valid for FileResponse or WebSocket endpoints")
	}
	if r.StatusField == "" {
		if !r.Status.IsExact() || statusMode {
			verr.Add(r, "non-exact or status-selected responses require StatusField")
		}
		return
	}
	key, field := source.FindAttribute(r.StatusField)
	if field == nil || unalias(field.Type) != Int || !source.IsRequired(key) || AllowsNull(field) || field.EffectiveDefault() != nil || IsDefaultErrorResult(source.Type) {
		verr.Add(r, "StatusField %q must be a required non-nullable native Int without a default", r.StatusField)
		return
	}
	r.validateStatusMappings(source, key, verr)
	r.validateStatusViews(source, key, verr)
	domain := r.StatusDomain()
	if r.Status.Phase() == PhaseInformational {
		domain = informationalObservationDomain(e.InformationalStatusKeys(), r.Status)
	}
	r.validateStatusDomain(key, field, domain, verr)
}

func (r *HTTPResponseExpr) validateStatusMappings(source *AttributeExpr, key string, verr *eval.ValidationErrors) {
	if r.bindings != nil {
		for _, binding := range r.bindings.bindings {
			if binding.location != "status" {
				for _, name := range binding.sources {
					if AttributeName(name) == AttributeName(key) {
						verr.Add(r, "StatusField %q competes with an explicit %s binding", key, binding.location)
					}
				}
			}
		}
	}
	// A typed handshake owns metadata separately from OpenAPIBody's event
	// documentation. Only ordinary response documentation shares this source.
	if body := r.OpenAPIBody; r.Handshake == nil && body != nil {
		selected, _ := body.Meta.Last("origin:attribute")
		// Template references supply defaults for selected fields; they do not
		// publish every source member into the documentation body.
		var member *AttributeExpr
		if object := AsObject(body.Type); object != nil {
			_, member = objectAttribute(object, key)
		}
		if AttributeName(selected) == AttributeName(key) || selected == "" && member != nil {
			verr.Add(r, "StatusField %q competes with an explicit body binding", key)
		}
	}
	if r.ErrorNameFromStatus && AttributeName(TaggedAttribute(source, "struct:error:name")) == AttributeName(key) {
		verr.Add(r, "StatusField cannot also be ErrorNameFromStatus")
	}
}

func (r *HTTPResponseExpr) validateStatusViews(source *AttributeExpr, key string, verr *eval.ValidationErrors) {
	if result, ok := source.Type.(*ResultTypeExpr); ok {
		for _, view := range result.Views {
			if selected, fixed := source.Meta.Last(ViewMetaKey); fixed && selected != view.Name {
				continue
			}
			if !result.ViewHasAttribute(view.Name, key) {
				verr.Add(r, "StatusField %q must survive view %q", key, view.Name)
			}
		}
	}
}

func (r *HTTPResponseExpr) validateStatusDomain(key string, field *AttributeExpr, domain []int, verr *eval.ValidationErrors) {
	constraints, err := EffectiveConstraintsFor(field)
	if err != nil {
		verr.Add(r, "StatusField %q: %s", key, err)
		return
	}
	normalized := *field
	normalized.Validation = constraints.Validation().Lowered()
	for _, code := range domain {
		if exampleMatchesValidation(&normalized, code) {
			return
		}
	}
	verr.Add(r, "StatusField %q constraints exclude every owned status for %s", key, r.Status.OpenAPIKey())
}

func (r *HTTPResponseExpr) validateFinalPhase(verr *eval.ValidationErrors) {
	switch r.Status.Phase() {
	case PhaseInformational:
		verr.Add(r, "informational status %s requires Informational, not Response", r.Status.OpenAPIKey())
	case PhaseUpgrade:
		verr.Add(r, "101 upgrade is not an authored Response; use Informational for interim values")
	case 0:
		verr.Add(r, "HTTP response requires an exact status code between 100 and 599 or a valid final class/fallback")
	}
}

// captureStatusDomain freezes validation-time ownership before response storage
// or body lowering changes the physical attribute graph.
func (r *HTTPResponseExpr) captureStatusDomain(e *HTTPEndpointExpr, source *AttributeExpr) {
	if r.bindings == nil {
		r.captureBindings(source)
	}
	if r.bindings.domainCaptured {
		return
	}
	r.bindings.domainCaptured = true
	if r.JSONRPCError != nil {
		return
	}
	switch r.Status.Phase() {
	case PhaseInformational:
		r.bindings.domain = InformationalEmissionDomain(e.InformationalStatusKeys(), r.Status)
	case PhaseFinal:
		r.bindings.domain = FinalRuntimeDomain(e.FinalStatusKeys(), r.Status, r.hasWireBody(e, source))
	}
}
