package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// clientBodyInitKey identifies a client body init function by the Go type of
// the body that the function builds.
type clientBodyInitKey string

// Hash returns the key.
func (k clientBodyInitKey) Hash() string {
	return string(k)
}

// clientBodyInitName returns the name of the client function that builds the
// request body or the WebSocket streaming body body of a service method. The
// name derives from the Go type name of the body, as in NewItemRequestBody.
// The bodies of one Go type share the function. The function of a body whose
// Go type differs from that of a body with the same type name, such as a
// nested collection of the element type of a flat collection, gets a name of
// its own from the client body init scope of the service.
func clientBodyInitName(sd *ServiceData, body *expr.AttributeExpr) string {
	name := fmt.Sprintf("New%s", codegen.Goify(sd.Scope.GoTypeName(body), true))
	return sd.clientBodyInits.HashedUnique(clientBodyInitKey(sd.Scope.GoTypeRef(body)), name)
}
