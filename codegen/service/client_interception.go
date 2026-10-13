package service

import (
	"slices"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type (
	// clientInterception retains the distinct endpoint roles of a mixed method.
	// Identical opening-only chains share one plan and one emitted wrapper.
	clientInterception struct {
		primary, stream *clientEndpointInterception
	}

	clientEndpointInterception struct {
		wrapper  string
		bindings []*clientInterceptorBinding
	}

	// clientInterceptorBinding selects operations without duplicating the
	// method/interceptor's accessor or canonical stream metadata.
	clientInterceptorBinding struct {
		name, wrapper string
		opening       bool
		messages      bool
	}
)

// ClientInterceptorWrapper returns the allocated wrapper for the selected client
// endpoint. A true stream argument selects a mixed method's secondary endpoint;
// false selects its primary endpoint, including nonmixed streaming methods.
// An empty name means no interception applies to that endpoint.
func (m *MethodData) ClientInterceptorWrapper(stream bool) string {
	if m.clientInterception == nil {
		return ""
	}
	plan := m.clientInterception.primary
	if stream {
		plan = m.clientInterception.stream
	}
	if plan == nil {
		return ""
	}
	return plan.wrapper
}

func newClientInterception(m *expr.MethodExpr) *clientInterception {
	plan := &clientInterception{primary: &clientEndpointInterception{}}
	if m.HasMixedResults() {
		plan.stream = &clientEndpointInterception{}
	}
	interceptors := slices.Clone(m.ClientInterceptors)
	slices.SortFunc(interceptors, func(a, b *expr.InterceptorExpr) int {
		return strings.Compare(a.Name, b.Name)
	})
	interceptors = slices.CompactFunc(interceptors, func(a, b *expr.InterceptorExpr) bool {
		return a.Name == b.Name
	})
	for _, in := range interceptors {
		messages := hasInterceptorAttributes(in.ReadStreamingPayload) ||
			hasInterceptorAttributes(in.WriteStreamingPayload) ||
			hasInterceptorAttributes(in.ReadStreamingResult) ||
			hasInterceptorAttributes(in.WriteStreamingResult)
		opening := !messages ||
			hasInterceptorAttributes(in.ReadPayload) || hasInterceptorAttributes(in.WritePayload)
		binding := &clientInterceptorBinding{
			name:     codegen.Goify(in.Name, true),
			opening:  opening,
			messages: messages && !m.HasMixedResults(),
		}
		if binding.opening || binding.messages {
			plan.primary.bindings = append(plan.primary.bindings, binding)
		}
		if plan.stream != nil {
			if messages {
				binding = &clientInterceptorBinding{name: binding.name, opening: opening, messages: true}
			}
			plan.stream.bindings = append(plan.stream.bindings, binding)
		}
	}
	if plan.stream != nil && slices.Equal(plan.primary.bindings, plan.stream.bindings) {
		plan.stream = plan.primary
	}
	return plan
}

func hasInterceptorAttributes(att *expr.AttributeExpr) bool {
	if att == nil {
		return false
	}
	obj := expr.AsObject(att.Type)
	return obj != nil && len(*obj) > 0
}

func (p *clientInterception) endpoints() []*clientEndpointInterception {
	plans := []*clientEndpointInterception{p.primary}
	if p.stream != nil && p.stream != p.primary {
		plans = append(plans, p.stream)
	}
	return plans
}

func (p *clientInterception) allocateEndpoints(scope *codegen.NameScope, method string) {
	for _, endpoint := range p.endpoints() {
		if len(endpoint.bindings) == 0 {
			continue
		}
		name := method
		if endpoint != p.primary {
			name += "Stream"
		}
		endpoint.wrapper = scope.Unique("Wrap" + name + "ClientEndpoint")
	}
}

func (p *clientInterception) bindings(name string) []*clientInterceptorBinding {
	var bindings []*clientInterceptorBinding
	for _, endpoint := range p.endpoints() {
		for _, binding := range endpoint.bindings {
			if binding.name == name && !slices.Contains(bindings, binding) {
				bindings = append(bindings, binding)
			}
		}
	}
	return bindings
}

func (p *clientInterception) allocateBindings(scope *codegen.NameScope, method, name string) {
	for _, binding := range p.bindings(name) {
		base := method
		if p.stream != nil && p.stream != p.primary && binding.messages {
			base += "Stream"
		}
		binding.wrapper = scope.Unique("wrapClient" + base + name)
	}
}
