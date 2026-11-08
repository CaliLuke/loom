package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
)

type (
	nativeHeaderDeclaration struct {
		typ     expr.UserType
		context *codegen.AttributeContext
	}
	nativeHeaderIdentity struct {
		declaration, ref, pkg               string
		pointer, ignoreRequired, useDefault bool
	}
)

// finalizeOutgoingHeaderValidation inventories native declarations once, before
// constructor allocation and rendering. Both Send and handshake encoders use
// this same native context, distinct from transport-local body declarations.
func (data *ServiceData) finalizeOutgoingHeaderValidation() {
	var declarations []nativeHeaderDeclaration
	seen := make(map[nativeHeaderIdentity]bool)
	for _, endpoint := range data.Endpoints {
		for _, info := range endpoint.Informational {
			declarations = append(declarations, nativeHeaderDeclarations(info.Response, outgoingHeaderContext(data, info.Response.Carrier), seen)...)
		}
		collect := func(response *ResponseData) {
			if response.Handshake || response.HasStatusBinding() {
				context := response.serverContext.Dup()
				declarations = append(declarations, nativeHeaderDeclarations(response, context, seen)...)
			}
		}
		for _, response := range endpoint.Result.Responses {
			collect(response)
		}
		for _, group := range endpoint.Errors {
			for _, failure := range group.Errors {
				collect(failure.Response)
			}
		}
	}
	if len(declarations) == 0 {
		return
	}
	data.outgoingHeaderValidationPrefix = reserveNativeHeaderValidationPrefix(data, declarations)
	data.outgoingHeaderValidationTypes = nativeHeaderValidationTypes(declarations, data.outgoingHeaderValidationPrefix)
}

// nativeHeaderDeclarations retains declaration and physical context identity
// independently of the HTTP caller that will validate the reconstructed value.
func nativeHeaderDeclarations(response *ResponseData, context *codegen.AttributeContext, seen map[nativeHeaderIdentity]bool) []nativeHeaderDeclaration {
	var declarations []nativeHeaderDeclaration
	for _, header := range response.Headers {
		_, native := response.Carrier.FindAttribute(header.Name)
		if native == nil {
			native = response.Carrier
		}
		representation.WalkUserTypes(native, func(typ expr.UserType) {
			if expr.IsAlias(typ) {
				return
			}
			attribute := &expr.AttributeExpr{Type: typ}
			identity := nativeHeaderIdentity{typ.Hash(), context.Scope.Ref(attribute, context.Pkg(attribute)), context.DefaultPkg, context.Pointer, context.IgnoreRequired, context.UseDefault}
			if seen[identity] {
				return
			}
			seen[identity] = true
			declarations = append(declarations, nativeHeaderDeclaration{typ, context})
		})
	}
	return declarations
}

func reserveNativeHeaderValidationPrefix(data *ServiceData, declarations []nativeHeaderDeclaration) string {
	reserved := data.constructorReservations(false)
	for {
		prefix := data.Scope.Unique("ValidateHTTPHeader")
		available := true
		for _, declaration := range declarations {
			declaration.context.ValidationPrefix = prefix
			function := codegen.NamedValidationFunction(&expr.AttributeExpr{Type: declaration.typ}, declaration.context, "body")
			if reserved[function.Name] || data.Scope.PeekUnique(function.Name) != function.Name {
				available = false
				break
			}
		}
		if available {
			for _, declaration := range declarations {
				declaration.context.ValidationPrefix = prefix
				function := codegen.NamedValidationFunction(&expr.AttributeExpr{Type: declaration.typ}, declaration.context, "body")
				data.Scope.Unique(function.Name)
			}
			return prefix
		}
	}
}

func nativeHeaderValidationTypes(declarations []nativeHeaderDeclaration, prefix string) []*TypeData {
	var types []*TypeData
	for _, declaration := range declarations {
		attribute := &expr.AttributeExpr{Type: declaration.typ}
		context := declaration.context
		context.ValidationPrefix = prefix
		function := codegen.NamedValidationFunction(attribute, context, "body")
		validation := codegen.ValidationCode(declaration.typ.Attribute(), declaration.typ, function.Context(context), true, false, false, "body")
		if validation == "" {
			continue
		}
		types = append(types, &TypeData{
			attribute: attribute, context: context, Name: declaration.typ.Name(),
			VarName: context.Scope.Name(attribute, "", context), Ref: function.Ref,
			validationFunction: function, ValidateDef: validation,
		})
	}
	return types
}

func outgoingHeaderContext(data *ServiceData, carrier *expr.AttributeExpr) *codegen.AttributeContext {
	pkg := data.Service.LocationPackageName(codegen.UserTypeLocation(carrier.Type))
	context := serviceContext(pkg, data.Service.Scope)
	context.ValidationPrefix = data.outgoingHeaderValidationPrefix
	return context
}
