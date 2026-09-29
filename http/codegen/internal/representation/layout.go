package representation

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

// Layout records the physical presence policy allocated to Go declarations.
// It is generation-local mutable analysis state, not semantic occurrence state.
type Layout struct {
	// JSONPresence selects explicit JSON field presence.
	JSONPresence map[string]bool
	// Pointer selects pointer storage.
	Pointer map[string]bool
	// UseDefault selects defaulted value storage.
	UseDefault map[string]bool
}

// NewLayout creates an empty physical declaration registry.
func NewLayout() Layout {
	return Layout{make(map[string]bool), make(map[string]bool), make(map[string]bool)}
}

// ServiceLayouts allocates the same root-before-nested layout precedence used
// by emitted server and client body declarations.
func ServiceLayouts(endpoints []*transportir.Endpoint) (Layout, Layout) {
	server, client := NewLayout(), NewLayout()
	for _, endpoint := range endpoints {
		presence := !endpoint.Request.FormEncoded && !endpoint.Request.Multipart
		server.RecordRoot(endpoint.Request.Body, presence, true, false)
		client.RecordRoot(endpoint.Request.Body, false, false, true)
		server.RecordRoot(endpoint.Request.StreamingBody, true, true, false)
		client.RecordRoot(endpoint.Request.StreamingBody, false, false, true)
		for _, response := range serviceResponses(endpoint) {
			server.RecordRoot(response.Body, false, false, true)
			client.RecordRoot(response.Body, true, true, false)
		}
	}
	for _, endpoint := range endpoints {
		for _, response := range serviceResponses(endpoint) {
			server.RecordNested(response.Body, false, false, true)
		}
	}
	for _, endpoint := range endpoints {
		server.RecordNested(endpoint.Request.Body, !endpoint.Request.FormEncoded && !endpoint.Request.Multipart, true, false)
		server.RecordNested(endpoint.Request.StreamingBody, true, true, false)
	}
	for _, endpoint := range endpoints {
		client.RecordNested(endpoint.Request.Body, false, false, true)
		client.RecordNested(endpoint.Request.StreamingBody, false, false, true)
		for _, response := range serviceResponses(endpoint) {
			client.RecordNested(response.Body, true, true, false)
		}
	}
	return server, client
}

// Bind exposes this registry through the shared attribute context.
func (l Layout) Bind(context *codegen.AttributeContext) {
	context.JSONPresenceTypes = l.JSONPresence
	context.PresencePointerTypes = l.Pointer
	context.PresenceUseDefaultTypes = l.UseDefault
}

// Record preserves the highest-priority physical layout for a declaration.
func (l Layout) Record(userType expr.UserType, presence, pointer, defaults bool) {
	key := userType.Hash()
	if old, found := l.JSONPresence[key]; found && layoutPriority(old, l.Pointer[key]) >= layoutPriority(presence, pointer) {
		return
	}
	l.JSONPresence[key], l.Pointer[key], l.UseDefault[key] = presence, pointer, defaults
}

// RecordRoot records only a named root declaration.
func (l Layout) RecordRoot(attribute *expr.AttributeExpr, presence, pointer, defaults bool) {
	if attribute != nil {
		if typ, ok := attribute.Type.(expr.UserType); ok {
			l.Record(typ, presence, pointer, defaults)
		}
	}
}

// RecordAll records every declaration reachable from an attribute.
func (l Layout) RecordAll(attribute *expr.AttributeExpr, presence, pointer, defaults bool) {
	if attribute != nil {
		WalkUserTypes(attribute.Type, func(typ expr.UserType) {
			l.Record(typ, presence, pointer, defaults)
		})
	}
}

// RecordNested records descendants without replacing the separately allocated root.
func (l Layout) RecordNested(attribute *expr.AttributeExpr, presence, pointer, defaults bool) {
	if attribute == nil {
		return
	}
	root := ""
	if typ, ok := attribute.Type.(expr.UserType); ok {
		root = typ.Hash()
	}
	WalkUserTypes(attribute.Type, func(typ expr.UserType) {
		if typ.Hash() != root {
			l.Record(typ, presence, pointer, defaults)
		}
	})
}

func serviceResponses(endpoint *transportir.Endpoint) []*transportir.ResponseStatus {
	return append(append([]*transportir.ResponseStatus(nil), endpoint.Response.Responses...), endpoint.Response.ErrorResponses...)
}

func layoutPriority(presence, pointer bool) uint8 {
	if presence {
		return 2
	}
	if pointer {
		return 1
	}
	return 0
}

// ApplyLayout applies the already allocated physical declaration layout. The
// maps belong to the generation context and use generated Go identity, never
// schema identity or an occurrence's semantic cache key.
func ApplyLayout(context *codegen.AttributeContext, attribute *expr.AttributeExpr) {
	if attribute == nil {
		return
	}
	userType, ok := attribute.Type.(expr.UserType)
	if !ok {
		return
	}
	key := userType.Hash()
	presence, recorded := context.JSONPresenceTypes[key]
	if !recorded {
		return
	}
	context.JSONPresence = presence
	context.CollectionElementPresence = presence
	context.Pointer = context.PresencePointerTypes[key]
	context.UseDefault = context.PresenceUseDefaultTypes[key]
}
