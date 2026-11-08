package openapiimport

import (
	"fmt"
	"strings"
)

func (a *designNameAllocator) allocateHandshakes(operations []Operation) {
	for _, operation := range operations {
		plan, _ := classifyOperationResponses(&operation, operation.sourcePath, a.document.Components)
		if !plan.handshake {
			continue
		}
		name := a.scope.Unique(operation.GoName + "Handshake")
		for a.types[strings.ToLower(name)] > 0 {
			name = a.scope.Unique(operation.GoName + "Handshake")
		}
		a.types[strings.ToLower(name)] = 1
		variable := uniqueName("Imported"+name, a.variables)
		a.document.names.handshakes[cloneIdentity{operation.Method, operation.Path, ""}] = cloneNames{name, variable}
	}
}

func (r *renderer) handshakeTypes() error {
	for i, plan := range r.operations {
		if plan.mode != responseBodySSE {
			continue
		}
		operation := r.document.Operations[i]
		name := r.document.names.handshakes[cloneIdentity{operation.Method, operation.Path, ""}]
		if name.variable == "" {
			continue
		}
		r.open("var %s = Type(%q, func()", name.variable, name.typeName)
		var required []string
		if plan.successes[0].statusField != "" {
			r.line("Attribute(%q, Int)", "http_status")
			required = append(required, "http_status")
		}
		for _, response := range plan.successes {
			path := fmt.Sprintf("#/operations/%d/responses/%s/headers", i, response.status)
			if err := r.responseHeaderAttributes(response.headers, path); err != nil {
				return err
			}
			for _, header := range response.headers {
				if header.header.Required {
					required = append(required, header.field)
				}
			}
		}
		if len(required) > 0 {
			r.quotedCall("Required", required)
		}
		r.close()
		r.line("")
	}
	return nil
}
