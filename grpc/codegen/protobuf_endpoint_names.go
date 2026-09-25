package codegen

import (
	"strconv"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/grpc/codegen/internal/transportir"
)

type (
	// messagePosition is the position of an attribute in the messages of a
	// service, which decides whether a user type there is a message of its
	// own.
	messagePosition int

	// designMessageWalker records the shapes of the messages of the design
	// user types reachable from the messages of a service.
	designMessageWalker struct {
		shapes map[string]string
		seen   map[string]struct{}
	}
)

const (
	// topMessage is the payload, streaming payload, result or error of a
	// method.
	topMessage messagePosition = iota
	// fieldMessage is a field of a message.
	fieldMessage
	// branchMessage is a branch of a union.
	branchMessage
	// elemMessage is an array element, a map key or a map value.
	elemMessage
)

// conflictingDesignMessage is the shape recorded for a name that design types
// with different messages share. No generated message has that shape.
const conflictingDesignMessage = "\x00"

// designMessageShapes returns the shapes of the protocol buffer messages of
// the design user types that the messages of the endpoints declare, by user
// type name. Messages are identified by name, so a message generated for a
// method, such as its request message, must not take the name of one of these
// types unless it has the same shape, see endpointMessageName.
func designMessageShapes(endpoints []*transportir.Endpoint) map[string]string {
	w := &designMessageWalker{shapes: make(map[string]string), seen: make(map[string]struct{})}
	for _, endpoint := range endpoints {
		w.walk(endpoint.Request.Message, topMessage)
		if endpoint.Request.StreamingPayload.Type != expr.Empty {
			w.walk(endpoint.Request.StreamingMessage, topMessage)
		}
		w.walk(endpoint.Response.Message, topMessage)
		for _, grpcErr := range endpoint.Errors {
			if grpcErr.Type == expr.ErrorResult || !expr.IsObject(grpcErr.Attribute.Type) {
				continue
			}
			w.walk(grpcErr.Response.Message, topMessage)
		}
	}
	return w.shapes
}

// generatedMessageAttribute returns the attribute of the message that
// makeProtoBufMessage generates for the top-level attribute att that is not a
// user type: an empty object for Empty, att for an object and the message
// that wraps any other value.
func generatedMessageAttribute(att *expr.AttributeExpr) *expr.AttributeExpr {
	switch {
	case att.Type == expr.Empty:
		return &expr.AttributeExpr{Type: &expr.Object{}}
	case expr.IsObject(att.Type):
		return att
	case expr.IsArray(att.Type) || expr.IsMap(att.Type):
		return wrapperAttribute(att, false)
	default:
		return wrapperAttribute(att, true)
	}
}

// designMessageAttribute returns the attribute of the message of the design
// user type ut: the message that wraps a named primitive, union, array or
// map, or the attribute of an object.
func designMessageAttribute(ut expr.UserType) *expr.AttributeExpr {
	switch {
	case expr.IsArray(ut) || expr.IsMap(ut):
		return collectionMessageAttribute(ut)
	case expr.IsPrimitive(ut) || expr.IsUnion(ut):
		return wrapperAttribute(ut.Attribute(), true)
	default:
		return ut.Attribute()
	}
}

// isDesignMessage reports whether the user type ut at position pos is a
// protocol buffer message of its own: an object or a named array or map
// anywhere, a named union except as a field, whose branches are then oneof
// fields of the holder, and a named primitive only at the top level.
func isDesignMessage(ut expr.UserType, pos messagePosition) bool {
	switch {
	case expr.IsPrimitive(ut):
		return pos == topMessage
	case expr.IsUnion(ut):
		return pos != fieldMessage
	default:
		return true
	}
}

// walk records the user types reachable from att at position pos that are
// protocol buffer messages.
func (w *designMessageWalker) walk(att *expr.AttributeExpr, pos messagePosition) {
	switch dt := att.Type.(type) {
	case expr.UserType:
		if dt == expr.Empty {
			return
		}
		if isDesignMessage(dt, pos) {
			w.record(dt)
		}
		if _, ok := w.seen[dt.ID()]; ok {
			return
		}
		w.seen[dt.ID()] = struct{}{}
		w.walk(dt.Attribute(), pos)
	case *expr.Object:
		for _, nat := range *dt {
			w.walk(nat.Attribute, fieldMessage)
		}
	case *expr.Array:
		w.walk(dt.ElemType, elemMessage)
	case *expr.Map:
		w.walk(dt.KeyType, elemMessage)
		w.walk(dt.ElemType, elemMessage)
	case *expr.Union:
		for _, nat := range dt.Values {
			w.walk(nat.Attribute, branchMessage)
		}
	}
}

// record records the shape of the message of ut under the name of ut.
func (w *designMessageWalker) record(ut expr.UserType) {
	shape := protoAttributeShape(designMessageAttribute(ut))
	if other, ok := w.shapes[ut.Name()]; ok && other != shape {
		shape = conflictingDesignMessage
	}
	w.shapes[ut.Name()] = shape
}

// endpointMessageName returns the name of the message that makeProtoBufMessage
// generates for the top-level attribute message of a method: candidate,
// unless a design user type that a message of the service declares has that
// name and a different message. It then returns candidate followed by the
// first number from 2 that no design type and no other generated message
// uses. A user type message keeps its own name, so candidate is not used.
// message is nil for a message that has no design counterpart, such as a
// stream envelope.
func (sd *ServiceData) endpointMessageName(candidate string, message *expr.AttributeExpr) string {
	shape, ok := sd.designMessages[candidate]
	if !ok {
		return candidate
	}
	if message != nil {
		if _, ok := message.Type.(expr.UserType); ok && message.Type != expr.Empty {
			return candidate
		}
		if shape == protoAttributeShape(generatedMessageAttribute(message)) {
			return candidate
		}
	}
	for i := 2; ; i++ {
		name := candidate + strconv.Itoa(i)
		if _, ok := sd.designMessages[name]; ok || isDesignUserTypeName(name) {
			continue
		}
		if _, ok := sd.anonymousMessages[name]; ok {
			continue
		}
		return name
	}
}
