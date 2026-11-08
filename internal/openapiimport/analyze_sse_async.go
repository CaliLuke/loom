package openapiimport

import (
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
)

// analyzeSSEAsync normalizes the reserved typed payload through the same schema
// adapter and index as ordinary occurrences. Recognition is separate from the
// complete contract check; the extension never selects streaming behavior.
func (a *analyzer) analyzeSSEAsync(operation *Operation, source *v3.Operation, components Components) {
	node := source.Extensions.GetOrZero("x-loom-async")
	if node == nil {
		return
	}
	indexes := indexReusableComponents(components)
	guard := false
	for _, response := range operation.Responses {
		resolved, _, err := indexes.responses.resolve(response.Response)
		if err == nil && resolved.stream != nil {
			guard = true
		}
	}
	if !guard {
		return
	}
	roots, err := reservedSSESchemaRoots(source.GoLow().RootNode, operation.sourcePath)
	if err != nil {
		a.unsupported("sse-async", operation.sourcePath+"/x-loom-async", err.Error())
		return
	}
	for _, root := range roots {
		proxy := &lowbase.SchemaProxy{}
		if err := proxy.Build(source.GoLow().GetContext(), nil, root.node, source.GoLow().GetIndex()); err != nil {
			a.unsupported("sse-async", root.path, err.Error())
			continue
		}
		value := a.schema(base.NewSchemaProxy(&low.NodeReference[*lowbase.SchemaProxy]{Value: proxy, ValueNode: root.node}), root.path)
		operation.asyncSchemas = append(operation.asyncSchemas, sseAsyncSchema{value, root.path})
	}
}
