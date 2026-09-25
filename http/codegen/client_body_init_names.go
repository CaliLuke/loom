package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// clientBodyInitKey identifies a client body init function by the Go type of
// the body that the function builds and by the source of the body: the Go
// type of the function argument and the expression that the function reads
// from the argument.
type clientBodyInitKey struct {
	body, argument, source string
}

// Hash returns the key.
func (k clientBodyInitKey) Hash() string {
	return k.body + "\x00" + k.argument + "\x00" + k.source
}

// clientBodyInitName returns the name of the client function that builds the
// request body or the WebSocket streaming body body of a service method from
// an argument of the Go type argument. source is the expression that the
// function reads from the argument, such as p or p.A for a payload attribute
// selected with Body. The name derives from the Go type name of the body, as
// in NewItemRequestBody. The bodies of one Go type built from one source
// share the function. Another function whose name would be the same, such as
// that of a nested collection of the element type of a flat collection, or
// that of the same body built from another named collection type or from
// another payload attribute, gets a name of its own from the client body init
// scope of the service.
func clientBodyInitName(sd *ServiceData, body *expr.AttributeExpr, argument, source string) string {
	name := fmt.Sprintf("New%s", codegen.Goify(sd.Scope.GoTypeName(body), true))
	key := clientBodyInitKey{body: sd.Scope.GoTypeRef(body), argument: argument, source: source}
	return sd.clientBodyInits.HashedUnique(key, name)
}
