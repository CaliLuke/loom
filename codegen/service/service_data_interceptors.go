package service

import (
	"sort"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// collectInterceptors returns the set of interceptors defined on the given
// service including any interceptor defined on specific service methods or API.
func (d *ServicesData) collectInterceptors(svc *expr.ServiceExpr, methods []*MethodData, scope *codegen.NameScope, server bool) []*InterceptorData {
	var ints []*expr.InterceptorExpr
	if server {
		ints = d.Root.API.ServerInterceptors
		ints = append(ints, svc.ServerInterceptors...)
		for _, m := range svc.Methods {
			ints = append(ints, m.ServerInterceptors...)
		}
	} else {
		ints = d.Root.API.ClientInterceptors
		ints = append(ints, svc.ClientInterceptors...)
		for _, m := range svc.Methods {
			ints = append(ints, m.ClientInterceptors...)
		}
	}
	sort.Slice(ints, func(i, j int) bool {
		return ints[i].Name < ints[j].Name
	})
	for i := 1; i < len(ints); i++ {
		if ints[i-1].Name == ints[i].Name {
			ints = append(ints[:i], ints[i+1:]...)
			i--
		}
	}

	res := make([]*InterceptorData, 0, len(ints))
	for _, i := range ints {
		res = append(res, buildInterceptorData(svc, methods, i, scope, server))
	}
	return res
}

// buildInterceptorData creates the data needed to generate interceptor code.
func buildInterceptorData(svc *expr.ServiceExpr, methods []*MethodData, i *expr.InterceptorExpr, scope *codegen.NameScope, server bool) *InterceptorData {
	data := &InterceptorData{
		Name:        codegen.Goify(i.Name, true),
		DesignName:  i.Name,
		Description: i.Description,
	}
	if len(svc.Methods) == 0 {
		return data
	}
	attributesCollected := false
	for _, m := range svc.Methods {
		applies := false
		intExprs := m.ServerInterceptors
		if !server {
			intExprs = m.ClientInterceptors
		}
		for _, in := range intExprs {
			if in.Name == i.Name {
				if !attributesCollected {
					payload, result, streamingPayload := m.Payload, m.Result, m.StreamingPayload
					data.ReadPayload = collectAttributes(i.ReadPayload, payload, scope)
					data.WritePayload = collectAttributes(i.WritePayload, payload, scope)
					data.ReadResult = collectAttributes(i.ReadResult, result, scope)
					data.WriteResult = collectAttributes(i.WriteResult, result, scope)
					data.ReadStreamingPayload = collectAttributes(i.ReadStreamingPayload, streamingPayload, scope)
					data.WriteStreamingPayload = collectAttributes(i.WriteStreamingPayload, streamingPayload, scope)
					data.ReadStreamingResult = collectAttributes(i.ReadStreamingResult, result, scope)
					data.WriteStreamingResult = collectAttributes(i.WriteStreamingResult, result, scope)
					if len(data.ReadPayload) > 0 || len(data.WritePayload) > 0 {
						data.HasPayloadAccess = true
					}
					if len(data.ReadResult) > 0 || len(data.WriteResult) > 0 {
						data.HasResultAccess = true
					}
					if len(data.ReadStreamingPayload) > 0 || len(data.WriteStreamingPayload) > 0 {
						data.HasStreamingPayloadAccess = true
					}
					if len(data.ReadStreamingResult) > 0 || len(data.WriteStreamingResult) > 0 {
						data.HasStreamingResultAccess = true
					}
					attributesCollected = true
				}
				applies = true
				break
			}
		}
		if !applies {
			continue
		}
		var md *MethodData
		for _, mt := range methods {
			if m.Name == mt.Name {
				md = mt
				break
			}
		}
		data.Methods = append(data.Methods, buildInterceptorMethodData(i, md))
		if server {
			md.ServerInterceptors = append(md.ServerInterceptors, i.Name)
		} else {
			md.ClientInterceptors = append(md.ClientInterceptors, i.Name)
		}
	}
	return data
}

// buildInterceptorMethodData creates the data needed to generate interceptor
// method code.
func buildInterceptorMethodData(i *expr.InterceptorExpr, md *MethodData) *MethodInterceptorData {
	var serverStream, clientStream *StreamInterceptorData
	if md.ServerStream != nil {
		serverStream = &StreamInterceptorData{
			Interface:           md.ServerStream.Interface,
			SendName:            md.ServerStream.SendName,
			SendWithContextName: md.ServerStream.SendWithContextName,
			SendTypeRef:         md.ServerStream.SendTypeRef,
			RecvName:            md.ServerStream.RecvName,
			RecvWithContextName: md.ServerStream.RecvWithContextName,
			RecvTypeRef:         md.ServerStream.RecvTypeRef,
			MustClose:           md.ServerStream.MustClose,
			EndpointStruct:      md.ServerStream.EndpointStruct,
		}
	}
	if md.ClientStream != nil {
		clientStream = &StreamInterceptorData{
			Interface:           md.ClientStream.Interface,
			SendName:            md.ClientStream.SendName,
			SendWithContextName: md.ClientStream.SendWithContextName,
			SendTypeRef:         md.ClientStream.SendTypeRef,
			RecvName:            md.ClientStream.RecvName,
			RecvWithContextName: md.ClientStream.RecvWithContextName,
			RecvTypeRef:         md.ClientStream.RecvTypeRef,
			MustClose:           md.ClientStream.MustClose,
		}
	}
	var payloadAccess, resultAccess, streamingPayloadAccess, streamingResultAccess string
	if i.ReadPayload != nil || i.WritePayload != nil {
		payloadAccess = codegen.Goify(i.Name, false) + md.VarName + "Payload"
	}
	if i.ReadResult != nil || i.WriteResult != nil {
		resultAccess = codegen.Goify(i.Name, false) + md.VarName + "Result"
	}
	if i.ReadStreamingPayload != nil || i.WriteStreamingPayload != nil {
		streamingPayloadAccess = codegen.Goify(i.Name, false) + md.VarName + "StreamingPayload"
	}
	if i.ReadStreamingResult != nil || i.WriteStreamingResult != nil {
		streamingResultAccess = codegen.Goify(i.Name, false) + md.VarName + "StreamingResult"
	}
	return &MethodInterceptorData{
		MethodName:             md.VarName,
		PayloadAccess:          payloadAccess,
		ResultAccess:           resultAccess,
		PayloadRef:             md.PayloadRef,
		ResultRef:              md.ResultRef,
		StreamingPayloadAccess: streamingPayloadAccess,
		StreamingPayloadRef:    md.StreamingPayloadRef,
		StreamingResultAccess:  streamingResultAccess,
		StreamingResultRef:     md.ResultRef,
		ClientStream:           clientStream,
		ServerStream:           serverStream,
	}
}

// collectAttributes builds AttributeData from an AttributeExpr
func collectAttributes(attrNames, parent *expr.AttributeExpr, scope *codegen.NameScope) []*AttributeData {
	if attrNames == nil {
		return nil
	}
	obj := expr.AsObject(attrNames.Type)
	if obj == nil {
		return nil
	}
	data := make([]*AttributeData, len(*obj))
	for i, nat := range *obj {
		parentAttr := parent.Find(nat.Name)
		if parentAttr == nil {
			continue
		}
		var pkg string
		if loc := codegen.UserTypeLocation(parentAttr.Type); loc != nil {
			pkg = loc.PackageName()
		}
		data[i] = &AttributeData{
			Name:    codegen.Goify(nat.Name, true),
			TypeRef: scope.GoFullTypeRef(parentAttr, pkg),
			Pointer: parent.IsPrimitivePointer(nat.Name, true),
		}
	}
	return data
}
