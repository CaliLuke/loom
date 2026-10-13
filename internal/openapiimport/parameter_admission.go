package openapiimport

import (
	"fmt"

	"github.com/CaliLuke/loom/internal/httpencoding"
)

// admitParameterShape checks the physical binding before render planning.
// Query serialization is checked separately only after its carrier is admitted.
// Request and response headers share this same location contract.
func (p *documentPlanner) admitParameterShape(schema *Schema, location, path string) bool {
	shape := p.urlShape(schema)
	primitive := p.schemaIsPrimitive(p.urlSchema(schema), make(map[string]struct{}))
	var supported bool
	var expected string
	switch location {
	case "query":
		supported = httpencoding.Select(httpencoding.NamedQuery, shape) != httpencoding.Unsupported
		expected = "primitive, primitive array, or JSON-encoded map"
	case "path", "header":
		supported = primitive || shape == httpencoding.Sequence
		expected = "primitive or primitive array"
	case "cookie":
		supported = primitive
		expected = "primitive"
	default:
		expected = "a supported HTTP parameter location"
	}
	if !supported {
		p.unsupported("parameter-shape", path, fmt.Sprintf("%s binding requires %s", location, expected))
	}
	return supported
}
