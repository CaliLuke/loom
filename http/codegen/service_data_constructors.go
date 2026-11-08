package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type (
	// InitData contains the data required to render a constructor.
	InitData struct {
		constructor        *constructorPlan
		nullableProjection *nullableResponseProjection
		errorConstructor   string
		errorOwnerRef      string
		// serverOptionalObjectBody makes whole-object body conversion conditional
		// on presence. Selected bodies retain ReturnIsOptionalBody semantics.
		serverOptionalObjectBody bool
		// serverBodyEnvelope allocates the outer payload for non-body fields or
		// credentials even when the whole object body is absent.
		serverBodyEnvelope bool
		// returnContext retains a response's selected declaration owner through
		// constructor finalization, including projected names and import aliases.
		returnContext *codegen.AttributeContext
		// Name is the constructor function name.
		Name string
		// Description is the function description.
		Description string
		// ServerArgs is the list of constructor arguments for server
		// side code.
		ServerArgs []*InitArgData
		// ClientArgs is the list of constructor arguments for client
		// side code.
		ClientArgs []*InitArgData
		// CLIArgs is the list of arguments that should be initialized
		// from CLI flags. This is used for implicit attributes which
		// as the time of writing is only used for the basic auth
		// username and password.
		CLIArgs []*InitArgData
		// ServerCode is the code that builds the payload from the
		// request on the server when it contains user types.
		ServerCode string
		// ServerErrorAware reports fallible server-side conversion.
		ServerErrorAware bool
		// ClientCode is the code that builds the payload or result type
		// from the request or response state on the client when it
		// contains user types.
		ClientCode string
		// ClientErrorAware reports fallible client-side conversion.
		ClientErrorAware bool
		// ReturnTypePkg is the package where the return type is present.
		ReturnTypePkg string
		// ReturnTypeName is the qualified (including the package name)
		// name of the payload, result or error type.
		ReturnTypeName string
		// ReturnTypeRef is the qualified (including the package name)
		// reference to the payload, result or error type.
		ReturnTypeRef string
		// ReturnTypeAttribute is the name of the attribute initialized by this
		// constructor when it only initializes one attribute (i.e. body was
		// defined with Body("name") syntax).
		ReturnTypeAttribute string
		// ReturnIsStruct is true if the payload, result or error type is a struct.
		ReturnIsStruct bool
		// ReturnIsPrimitivePointer indicates whether the payload, result or error
		// type is a primitive pointer.
		ReturnIsPrimitivePointer bool
		// ReturnIsUnionValue indicates that a detached union transform returns a
		// pointer that must be dereferenced for a required struct field.
		ReturnIsUnionValue bool
		// ReturnIsOptionalBody indicates that the server body argument is nil
		// when the request has no body. The server constructor then builds
		// the ReturnTypeAttribute field only for a non-nil body and leaves it
		// nil otherwise.
		ReturnIsOptionalBody bool
		// SkipFieldInit disables the generic struct-field assignment path when
		// a constructor already builds the full return value itself.
		SkipFieldInit bool
	}

	// AttributeData contains the information needed to generate the code
	// related to a specific payload or result attribute.
	AttributeData struct {
		nativeContext *codegen.AttributeContext
		// fieldAttribute and fieldContext retain the actual physical service
		// occurrence bound to the value and the context that references it.
		fieldAttribute *expr.AttributeExpr
		fieldContext   *codegen.AttributeContext
		// Name is the name of the attribute.
		Name string
		// VarName is the name of the variable that holds the attribute value.
		VarName string
		// Pointer is true if the attribute value is a pointer.
		Pointer bool
		// Required is true if the attribute is required in the parent attribute.
		Required bool
		// Type is the attribute type.
		Type expr.DataType
		// TypeName is the generated attribute type name.
		TypeName string
		// TypeRef is the generated attribute type reference.
		TypeRef string
		// Description is the attribute description as defined in the design.
		Description string
		// FieldName is the name of the data structure field that should
		// be initialized with the value if any.
		FieldName string
		// FieldType is the type of the data structure field that should be
		// initialized with the attribute value or read into the attribute value.
		FieldType expr.DataType
		// FieldPointer if true indicates that the data structure field is a
		// pointer.
		FieldPointer bool
		// DefaultValue is the default value of the attribute if any.
		DefaultValue any
		// Validate contains the validation code for the attribute value if any.
		Validate string
		// IsTextUnmarshaler is true when the field has a custom string-backed
		// type that should be decoded through encoding.TextUnmarshaler.
		IsTextUnmarshaler bool
		// Example is an example attribute value
		Example any
		// IsAliased is true if the field type is a user-defined type (alias).
		IsAliased bool
		// ServiceTypeRef is the service-aware type reference for cross-service resolution.
		ServiceTypeRef string
		// Locals lists the local variables that generated transport code
		// derives from VarName.
		Locals DerivedVarNames
	}

	// InitArgData represents a single constructor argument.
	InitArgData struct {
		// attribute and context retain the actual decoded argument representation.
		attribute *expr.AttributeExpr
		// fieldAttribute retains the selected native destination occurrence.
		fieldAttribute      *expr.AttributeExpr
		context             *codegen.AttributeContext
		parameter           *Element
		parameterValueVar   string
		parameterSegmentVar string
		// responseStatus identifies the captured actual-response status argument.
		responseStatus bool
		*AttributeData
		// Reference to the argument, e.g. "&body".
		Ref string
		// TransformCode assigns a service value to the transport argument when
		// direct assignment or a single type conversion cannot preserve its type.
		TransformCode string
	}
)
