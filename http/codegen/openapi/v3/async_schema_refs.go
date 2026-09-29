package openapiv3

import "github.com/CaliLuke/loom/http/codegen/openapi"

// visitAsyncMessageSchemas exposes only the framework-owned schema roots in
// the async extension. Other extension fields and example/default values are
// data, even when they contain keys named schema or $ref. Alias rewriting and
// component reachability must traverse the same roots before deleting schemas.
func visitAsyncMessageSchemas(op *Operation, visit func(*openapi.Schema)) {
	contract, ok := op.Extensions[openAPIAsyncExtension].(map[string]any)
	if !ok {
		return
	}
	messages, ok := contract["messages"].(map[string]any)
	if !ok {
		return
	}
	for _, direction := range []string{"inbound", "outbound"} {
		message, ok := messages[direction].(map[string]any)
		if !ok {
			continue
		}
		if schema, ok := message["schema"].(*openapi.Schema); ok {
			visit(schema)
		}
	}
}
