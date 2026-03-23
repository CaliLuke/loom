package expr

import (
	"github.com/CaliLuke/loom/eval"
)

type (
	// InterceptorExpr describes an interceptor definition in the design.
	// Interceptors are used to inject user code into the request/response processing pipeline.
	// There are four kinds of interceptors, in order of execution:
	//   * client-side payload: executes after the payload is encoded and before the request is sent to the server
	//   * server-side request: executes after the request is decoded and before the payload is sent to the service
	//   * server-side result: executes after the service returns a result and before the response is encoded
	//   * client-side response: executes after the response is decoded and before the result is sent to the client
	InterceptorExpr struct {
		// Name is the name of the interceptor
		Name string
		// Description is the optional description of the interceptor
		Description string
		// ReadPayload lists the payload attribute names read by the interceptor
		ReadPayload *AttributeExpr
		// WritePayload lists the payload attribute names written by the interceptor
		WritePayload *AttributeExpr
		// ReadResult lists the result attribute names read by the interceptor
		ReadResult *AttributeExpr
		// WriteResult lists the result attribute names written by the interceptor
		WriteResult *AttributeExpr
		// ReadStreamingPayload lists the streaming payload attribute names read by the interceptor
		ReadStreamingPayload *AttributeExpr
		// WriteStreamingPayload lists the streaming payload attribute names written by the interceptor
		WriteStreamingPayload *AttributeExpr
		// ReadStreamingResult lists the streaming result attribute names read by the interceptor
		ReadStreamingResult *AttributeExpr
		// WriteStreamingResult lists the streaming result attribute names written by the interceptor
		WriteStreamingResult *AttributeExpr
	}
)

// EvalName returns the generic expression name used in error messages.
func (i *InterceptorExpr) EvalName() string {
	return "interceptor " + i.Name
}

// validate validates the interceptor.
func (i *InterceptorExpr) validate(m *MethodExpr) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	i.validatePayloadAccess(m, verr)
	i.validateResultAccess(m, verr)
	i.validateStreamingPayloadAccess(m, verr)
	i.validateStreamingResultAccess(m, verr)
	return verr
}

func (i *InterceptorExpr) validatePayloadAccess(m *MethodExpr, verr *eval.ValidationErrors) {
	if i.ReadPayload == nil && i.WritePayload == nil {
		return
	}
	payload := i.objectAccessTarget(m, verr, m.Payload, "payload is not an object")
	if payload == nil {
		return
	}
	i.validateAccessPair(m, verr, payload, "payload", i.ReadPayload, i.WritePayload)
}

func (i *InterceptorExpr) validateResultAccess(m *MethodExpr, verr *eval.ValidationErrors) {
	if i.ReadResult == nil && i.WriteResult == nil {
		return
	}
	if m.IsResultStreaming() {
		verr.Add(m, "interceptor %q cannot be applied because the method result is streaming", i.Name)
	}
	result := i.objectAccessTarget(m, verr, m.Result, "result is not an object")
	if result == nil {
		return
	}
	i.validateAccessPair(m, verr, result, "result", i.ReadResult, i.WriteResult)
}

func (i *InterceptorExpr) validateStreamingPayloadAccess(m *MethodExpr, verr *eval.ValidationErrors) {
	if i.ReadStreamingPayload == nil && i.WriteStreamingPayload == nil {
		return
	}
	if !m.IsPayloadStreaming() || m.StreamingPayload == nil {
		verr.Add(m, "interceptor %q cannot be applied because the method payload is not streaming", i.Name)
		return
	}
	payload := i.objectAccessTarget(m, verr, m.StreamingPayload, "payload is not an object")
	if payload == nil {
		return
	}
	i.validateAccessPair(m, verr, payload, "streaming payload", i.ReadStreamingPayload, i.WriteStreamingPayload)
}

func (i *InterceptorExpr) validateStreamingResultAccess(m *MethodExpr, verr *eval.ValidationErrors) {
	if i.ReadStreamingResult == nil && i.WriteStreamingResult == nil {
		return
	}
	if !m.IsResultStreaming() {
		verr.Add(m, "interceptor %q cannot be applied because the method result is not streaming", i.Name)
		return
	}
	result := i.objectAccessTarget(m, verr, m.Result, "result is not an object")
	if result == nil {
		return
	}
	i.validateAccessPair(m, verr, result, "streaming result", i.ReadStreamingResult, i.WriteStreamingResult)
}

func (i *InterceptorExpr) objectAccessTarget(m *MethodExpr, verr *eval.ValidationErrors, att *AttributeExpr, objectErr string) *AttributeExpr {
	if !IsObject(att.Type) {
		verr.Add(m, "interceptor %q cannot be applied because the method %s", i.Name, objectErr)
		return nil
	}
	return mergeAttributeBases(att)
}

func mergeAttributeBases(att *AttributeExpr) *AttributeExpr {
	merged := DupAtt(att)
	for _, base := range att.Bases {
		if ut, ok := base.(UserType); ok {
			merged.Merge(ut.Attribute())
		}
	}
	return merged
}

func (i *InterceptorExpr) validateAccessPair(m *MethodExpr, verr *eval.ValidationErrors, target *AttributeExpr, suffix string, readAttr, writeAttr *AttributeExpr) {
	if readAttr != nil {
		i.validateAttributeAccess(m, "read "+suffix, verr, target, readAttr)
	}
	if writeAttr != nil {
		i.validateAttributeAccess(m, "write "+suffix, verr, target, writeAttr)
	}
}

// validateAttributeAccess validates that all attributes in attr exist in obj
func (i *InterceptorExpr) validateAttributeAccess(m *MethodExpr, source string, verr *eval.ValidationErrors, target *AttributeExpr, attr *AttributeExpr) {
	attrObj := AsObject(attr.Type)
	if attrObj == nil {
		verr.Add(m, "interceptor %q %s attribute is not an object", i.Name, source)
		return
	}
	for _, att := range *attrObj {
		if target.Find(att.Name) == nil {
			verr.Add(m, "interceptor %q cannot %s attribute %q: attribute does not exist", i.Name, source, att.Name)
		}
	}
}
