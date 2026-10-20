package codegen

import (
	"slices"
	"sort"

	"github.com/CaliLuke/loom/codegen/uniongen"
	"github.com/CaliLuke/loom/expr"
)

// httpBodyRoot records the exact prepared graph used to declare a transport body.
// Membership belongs to a package; allocated union identities belong to the scope.
type httpBodyRoot struct {
	attribute    *expr.AttributeExpr
	endpoint     string
	request      bool
	server       bool
	jsonPresence bool
}

func (sd *ServiceData) registerBodyRoot(attribute *expr.AttributeExpr, endpoint string, request, server, jsonPresence bool) {
	if attribute == nil || attribute.Type == expr.Empty {
		return
	}
	sd.bodyRoots = append(sd.bodyRoots, httpBodyRoot{attribute, endpoint, request, server, jsonPresence})
	if server && (request || containsUntaggedUnion(attribute)) {
		recordServerRequestValidationTypes(sd, attribute)
	}
}

// orderedBodyRoots retains the existing endpoint/request/stream/response/error
// allocation order even when streaming bodies are prepared after responses.
func (sd *ServiceData) orderedBodyRoots() []httpBodyRoot {
	order := make(map[string]int, len(sd.Endpoints))
	for i, endpoint := range sd.Endpoints {
		order[endpoint.Method.Name] = i
	}
	roots := slices.Clone(sd.bodyRoots)
	sort.SliceStable(roots, func(i, j int) bool {
		if order[roots[i].endpoint] != order[roots[j].endpoint] {
			return order[roots[i].endpoint] < order[roots[j].endpoint]
		}
		return roots[i].request && !roots[j].request
	})
	return roots
}

func (sd *ServiceData) collectBodyUnionTypes(closedObjects bool) {
	allocated := make(map[string]*uniongen.Type)
	roots := sd.orderedBodyRoots()
	// Allocate client declarations first, preserving the historical order. A
	// server-only view may add declarations, but never reallocates a shared one.
	for _, server := range []bool{false, true} {
		members := make(map[string]*uniongen.Type)
		seen := make(map[string]struct{})
		for _, root := range roots {
			if root.server == server {
				collectHTTPUnionTypes(root.attribute, sd.Scope, members, seen, closedObjects, allocated)
			}
		}
		declarations := make([]*uniongen.Type, 0, len(members))
		for _, declaration := range members {
			declarations = append(declarations, declaration)
		}
		sort.Slice(declarations, func(i, j int) bool {
			return declarations[i].Name < declarations[j].Name
		})
		if server {
			sd.ServerUnionTypes = declarations
		} else {
			sd.ClientUnionTypes = declarations
		}
	}
}
