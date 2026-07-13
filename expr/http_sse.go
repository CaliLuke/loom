package expr

import (
	"fmt"
	"strings"

	"slices"

	"github.com/CaliLuke/loom/eval"
)

type (
	// SSEProjectionExpr maps an SSE event discriminator to a result view.
	SSEProjectionExpr struct {
		// EventType is the value written to the SSE event field.
		EventType string
		// View is the result view used for the JSON event data.
		View string
	}

	// HTTPSSEExpr describes a Server-Sent Events configuration for a HTTP endpoint.
	// It defines how a streaming endpoint should use the Server-Sent Events protocol
	// instead of WebSockets.
	HTTPSSEExpr struct {
		// RequestIDField is the name of the attribute in the Payload type
		// that provides the Last-Event-ID request header value.
		// If empty, no Last-Event-ID request header is included in the request.
		RequestIDField string
		// NotificationMethod is the JSON-RPC method used for intermediate SSE
		// notifications. If empty, code generation uses a namespaced default.
		NotificationMethod string
		// DataField is the name of the attribute in the StreamingResult type
		// that provides the data field for a Server-Sent Event.
		// If empty, the entire StreamingResult is used as the data field.
		DataField string
		// IDField is the name of the attribute in the StreamingResult type
		// that provides the id field for a Server-Sent Event.
		// If empty, no id field is included in the event.
		IDField string
		// EventField is the name of the attribute in the StreamingResult type
		// that provides the event field (event type) for a Server-Sent Event.
		// If empty, no event field is included in the event.
		EventField string
		// RetryField is the name of the attribute in the StreamingResult type
		// that provides the retry field for a Server-Sent Event.
		// If empty, no retry field is included in the event.
		RetryField string
		// Projections map SSE event discriminator values to result views.
		Projections []*SSEProjectionExpr
	}
)

// EvalName returns the generic expression name used in error messages.
func (e *HTTPSSEExpr) EvalName() string {
	return "Server-Sent Events"
}

// Validate validates the Server-Sent Events expression against a specific result type.
func (e *HTTPSSEExpr) Validate(method *MethodExpr) error {
	if method == nil || method.Result == nil {
		return nil
	}

	verr := new(eval.ValidationErrors)
	if err := validateSSEField(method.Payload, e.RequestIDField, "request ID", []DataType{String}); err != nil {
		verr.Add(method, "%s", err.Error())
	}
	if err := validateSSEField(method.Result, e.DataField, "event data", nil); err != nil {
		verr.Add(method, "%s", err.Error())
	}
	if err := validateSSEField(method.Result, e.IDField, "event id", []DataType{String}); err != nil {
		verr.Add(method, "%s", err.Error())
	}
	if err := validateSSEField(method.Result, e.EventField, "event type", []DataType{String}); err != nil {
		verr.Add(method, "%s", err.Error())
	}
	if err := validateSSEField(method.Result, e.RetryField, "event retry", []DataType{Int, Int32, Int64, UInt, UInt32, UInt64}); err != nil {
		verr.Add(method, "%s", err.Error())
	}
	if err := e.validateProjections(method); err != nil {
		verr.Add(method, "%s", err.Error())
	}

	if len(verr.Errors) == 0 {
		return nil
	}
	return verr
}

func (e *HTTPSSEExpr) validateProjections(method *MethodExpr) error {
	if len(e.Projections) == 0 {
		return nil
	}
	if len(e.Projections) < 2 {
		return fmt.Errorf("SSE projections require at least two event-to-view mappings")
	}
	if e.EventField == "" {
		return fmt.Errorf("SSE projections require SSEEventType to select a projection")
	}
	if e.DataField != "" {
		return fmt.Errorf("SSE projections cannot be combined with SSEEventData")
	}
	if !method.Result.IsRequired(e.EventField) {
		return fmt.Errorf("SSE projection discriminator field %q must be required", e.EventField)
	}
	resultType, ok := method.Result.Type.(*ResultTypeExpr)
	if !ok {
		return fmt.Errorf("SSE projections require StreamingResult to use a ResultType")
	}
	events := make(map[string]struct{}, len(e.Projections))
	views := make(map[string]struct{}, len(e.Projections))
	for _, projection := range e.Projections {
		if projection == nil || projection.EventType == "" || projection.View == "" {
			return fmt.Errorf("SSE projection event type and view cannot be empty")
		}
		if _, ok := events[projection.EventType]; ok {
			return fmt.Errorf("SSE projection event type %q is mapped more than once", projection.EventType)
		}
		if _, ok := views[projection.View]; ok {
			return fmt.Errorf("SSE projection view %q is mapped more than once", projection.View)
		}
		if resultType.View(projection.View) == nil {
			return fmt.Errorf("SSE projection references unknown result view %q", projection.View)
		}
		events[projection.EventType] = struct{}{}
		views[projection.View] = struct{}{}
	}
	return nil
}

// validateSSEField validates that the given field exists in the result type and has the expected type.
func validateSSEField(rt *AttributeExpr, field, desc string, expectedTypes []DataType) error {
	if field == "" {
		return nil
	}

	if rt == nil {
		return fmt.Errorf("cannot use %q for SSE %s field: result type is nil", field, desc)
	}

	obj := AsObject(rt.Type)
	if obj == nil {
		return fmt.Errorf("cannot use %q for SSE %s field: result type is not an object", field, desc)
	}

	att := obj.Attribute(field)
	if att == nil {
		return fmt.Errorf("cannot use %q for SSE %s field: attribute not found in result type", field, desc)
	}

	if len(expectedTypes) > 0 && !slices.Contains(expectedTypes, att.Type) {
		typeNames := make([]string, len(expectedTypes))
		for i, t := range expectedTypes {
			typeNames[i] = t.Name()
		}
		return fmt.Errorf("cannot use %q for SSE %s field: attribute type must be one of %s", field, desc, strings.Join(typeNames, ", "))
	}

	return nil
}
