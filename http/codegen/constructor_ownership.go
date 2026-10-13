package codegen

import (
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

// finalizeConstructors runs after body/union collection and before any derived
// call expressions or CLI inputs escape analysis. HTTP and JSON-RPC share this
// owner, including analysis repeated for a file's import aliases.
func (sd *ServiceData) finalizeConstructors() {
	for _, client := range []bool{false, true} {
		scope := codegen.NewNameScope()
		for name := range sd.constructorReservations(client) {
			scope.Unique(name)
		}
		sd.walkConstructors(client, func(init *InitData, body bool) {
			plan := newConstructorPlan(init, client, body, sd.Service.Scope)
			previous := init.Name
			init.Name = scope.HashedUnique(plan, previous)
			if suffix, ok := strings.CutPrefix(init.Description, previous); ok {
				init.Description = init.Name + suffix
			}
			init.constructor = plan
		})
	}
	// Response plans may retain complete constructor call expressions. Rebuild
	// them from the allocated declarations, never by rewriting captured text.
	for _, ep := range sd.Endpoints {
		walkResponses(ep, func(response *ResponseData) {
			response.EncodePlan = newResponseEncodePlan(response)
		})
	}
}

func (sd *ServiceData) walkConstructors(client bool, visit func(*InitData, bool)) {
	seen := make(map[*InitData]bool)
	add := func(init *InitData, body bool) {
		if init != nil && !seen[init] {
			seen[init] = true
			visit(init, body)
		}
	}
	for _, ep := range sd.Endpoints {
		if client {
			if body := ep.Payload.Request.ClientBody; body != nil {
				add(body.Init, true)
			}
			if ws := ep.ClientWebSocket; ws != nil && ws.Payload != nil {
				add(ws.Payload.Init, true)
			}
		} else {
			add(ep.Payload.Request.PayloadInit, false)
			if ws := ep.ServerWebSocket; ws != nil && ws.Payload != nil {
				add(ws.Payload.Init, false)
			}
		}
		walkResponses(ep, func(response *ResponseData) {
			if client {
				add(response.ResultInit, false)
			} else {
				for _, body := range response.ServerBody {
					add(body.Init, true)
				}
			}
		})
	}
}

func walkResponses(ep *EndpointData, visit func(*ResponseData)) {
	for _, response := range ep.Result.Responses {
		visit(response)
	}
	for _, group := range ep.Errors {
		for _, err := range group.Errors {
			visit(err.Response)
		}
	}
}
