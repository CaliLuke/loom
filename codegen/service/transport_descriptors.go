package service

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type (
	// TypeDescriptor describes a service-level payload or result type together
	// with its resolved package and Go references.
	TypeDescriptor struct {
		// Attribute is the underlying type attribute.
		Attribute *expr.AttributeExpr
		// Package is the resolved Go package name for the type.
		Package string
		// Name is the fully qualified Go type name.
		Name string
		// Ref is the fully qualified Go type reference.
		Ref string
	}

	// ResultDescriptor describes both the declared method result and the
	// effective result type used by transports after view projection.
	ResultDescriptor struct {
		// Declared is the service method result as declared on the method.
		Declared TypeDescriptor
		// Effective is the result shape transports should use. It is the
		// projected result when the method uses viewed results, otherwise it is
		// the same as Declared.
		Effective TypeDescriptor
		// View is the default result view name when available.
		View string
		// ViewedRef is the fully qualified viewed-result wrapper reference when
		// the method uses viewed results.
		ViewedRef string
		// UsesViewedResult reports whether Effective comes from a viewed result.
		UsesViewedResult bool
	}

	// ErrorDescriptor describes a method error type together with its resolved
	// package and Go references.
	ErrorDescriptor struct {
		// Name is the method error name.
		Name string
		// Type is the resolved error type descriptor.
		Type TypeDescriptor
	}

	// StreamDescriptor captures transport-agnostic stream facts derived from a
	// service method.
	StreamDescriptor struct {
		// Kind is the stream kind declared by the service method.
		Kind expr.StreamKind
		// IsStreaming reports whether the method has any stream direction.
		IsStreaming bool
		// HasPayload reports whether the stream carries payload values.
		HasPayload bool
		// HasResult reports whether the stream carries result values.
		HasResult bool
		// IsClient reports whether the client sends stream values.
		IsClient bool
		// IsServer reports whether the server sends stream values.
		IsServer bool
		// IsBidirectional reports whether both sides send stream values.
		IsBidirectional bool
		// Payload is the streaming payload descriptor when the method accepts a
		// payload stream.
		Payload TypeDescriptor
		// Result is the streaming result descriptor when the method produces a
		// result stream.
		Result ResultDescriptor
	}

	// MethodCapabilityDescriptor captures transport-agnostic method capability
	// flags used by HTTP and gRPC analyzers.
	MethodCapabilityDescriptor struct {
		// HasPayload reports whether the method accepts a regular payload.
		HasPayload bool
		// HasResult reports whether the method returns a regular result.
		HasResult bool
		// HasViewedResult reports whether the effective result uses a viewed
		// wrapper type.
		HasViewedResult bool
		// HasStreamingPayload reports whether the method accepts a payload stream.
		HasStreamingPayload bool
		// HasStreamingResult reports whether the method produces a result stream.
		HasStreamingResult bool
		// HasMixedResults reports whether the method has both unary and streaming
		// results.
		HasMixedResults bool
		// HasRequestStruct reports whether transport code uses the generated
		// request wrapper struct.
		HasRequestStruct bool
		// HasResponseStruct reports whether transport code uses the generated
		// response wrapper struct.
		HasResponseStruct bool
		// Stream contains the stream capability descriptor.
		Stream StreamDescriptor
	}
)

// DefaultPackageName returns loc package when present, otherwise def.
func DefaultPackageName(loc *codegen.Location, def string) string {
	if loc == nil {
		return def
	}
	return loc.PackageName()
}

// BuildPayloadDescriptor returns the resolved payload type information for a
// method.
func BuildPayloadDescriptor(svc *Data, method *MethodData, payload *expr.AttributeExpr) TypeDescriptor {
	pkg := DefaultPackageName(method.PayloadLoc, svc.PkgName)
	return buildTypeDescriptor(svc.Scope, payload, pkg)
}

// BuildResultDescriptor returns the declared and effective result type
// information for a method.
func BuildResultDescriptor(svc *Data, method *MethodData, result *expr.AttributeExpr) ResultDescriptor {
	pkg := DefaultPackageName(method.ResultLoc, svc.PkgName)
	return buildResultDescriptorForPackage(svc, method, result, pkg)
}

// BuildErrorDescriptor returns the resolved error type information for a
// method error.
func BuildErrorDescriptor(svc *Data, method *MethodData, name string, errAttr *expr.AttributeExpr) ErrorDescriptor {
	pkg := DefaultPackageName(method.ErrorLocs[name], packageNameForAttribute(errAttr, svc.PkgName))
	return ErrorDescriptor{
		Name: name,
		Type: buildTypeDescriptor(svc.Scope, errAttr, pkg),
	}
}

// BuildStreamDescriptor returns transport-agnostic stream facts together with
// the streaming payload and result type descriptors used by transports.
func BuildStreamDescriptor(svc *Data, method *MethodData, payload, result *expr.AttributeExpr) StreamDescriptor {
	desc := DescribeStream(method)
	if desc.HasPayload {
		desc.Payload = buildTypeDescriptor(svc.Scope, payload, packageNameForAttribute(payload, svc.PkgName))
	}
	if desc.HasResult {
		desc.Result = buildResultDescriptorForPackage(svc, method, result, packageNameForAttribute(result, svc.PkgName))
	}
	return desc
}

// DescribeMethodCapabilities returns shared transport capability flags for the
// given method.
func DescribeMethodCapabilities(method *MethodData) MethodCapabilityDescriptor {
	stream := DescribeStream(method)
	return MethodCapabilityDescriptor{
		HasPayload:          method.PayloadRef != "",
		HasResult:           method.ResultRef != "",
		HasViewedResult:     method.ViewedResult != nil,
		HasStreamingPayload: stream.HasPayload,
		HasStreamingResult:  stream.HasResult,
		HasMixedResults:     method.HasMixedResults,
		HasRequestStruct:    method.SkipRequestBodyEncodeDecode && method.RequestStruct != "",
		HasResponseStruct:   method.SkipResponseBodyEncodeDecode && method.ResponseStruct != "",
		Stream:              stream,
	}
}

func buildResultDescriptorForPackage(svc *Data, method *MethodData, result *expr.AttributeExpr, pkg string) ResultDescriptor {
	declared := buildTypeDescriptor(svc.Scope, result, pkg)
	effective := declared
	view := expr.DefaultView
	if result != nil {
		if v, ok := result.Meta.Last(expr.ViewMetaKey); ok {
			view = v
		}
	}
	desc := ResultDescriptor{
		Declared:  declared,
		Effective: effective,
		View:      view,
	}
	if method.ViewedResult == nil {
		return desc
	}
	desc.UsesViewedResult = true
	desc.ViewedRef = method.ViewedResult.FullRef
	desc.Effective = TypeDescriptor{
		Attribute: projectedViewedResultAttribute(method.ViewedResult),
		Package:   svc.ViewsPkg,
		Name:      method.ViewedResult.FullName,
		Ref:       method.ViewedResult.FullRef,
	}
	return desc
}

// DescribeStream returns transport-agnostic stream facts for the given
// method.
func DescribeStream(method *MethodData) StreamDescriptor {
	desc := StreamDescriptor{
		Kind:        method.StreamKind,
		IsStreaming: method.ServerStream != nil || method.ClientStream != nil,
	}
	switch method.StreamKind {
	case expr.ClientStreamKind:
		desc.IsClient = true
		desc.HasPayload = true
		desc.HasResult = method.ResultRef != ""
	case expr.ServerStreamKind:
		desc.IsServer = true
		desc.HasResult = true
	case expr.BidirectionalStreamKind:
		desc.IsClient = true
		desc.IsServer = true
		desc.IsBidirectional = true
		desc.HasPayload = true
		desc.HasResult = true
	}
	return desc
}

func packageNameForAttribute(att *expr.AttributeExpr, def string) string {
	if att == nil {
		return def
	}
	return DefaultPackageName(codegen.UserTypeLocation(att.Type), def)
}

func buildTypeDescriptor(scope *codegen.NameScope, att *expr.AttributeExpr, pkg string) TypeDescriptor {
	if att == nil {
		return TypeDescriptor{Package: pkg}
	}
	if att.Type == expr.Empty {
		return TypeDescriptor{Attribute: att, Package: pkg}
	}
	return TypeDescriptor{
		Attribute: att,
		Package:   pkg,
		Name:      scope.GoFullTypeName(att, pkg),
		Ref:       scope.GoFullTypeRef(att, pkg),
	}
}

func projectedViewedResultAttribute(viewed *ViewedResultTypeData) *expr.AttributeExpr {
	projected := expr.AsObject(viewed.Type.Attribute().Type)
	if projected == nil {
		return viewed.Type.Attribute()
	}
	return projected.Attribute("projected")
}
