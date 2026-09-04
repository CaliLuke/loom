package expr

import (
	"testing"

	"github.com/CaliLuke/loom/eval"
	"github.com/stretchr/testify/require"
)

func TestHTTPTransportRejectsNullableOutsideJSONBodies(t *testing.T) {
	nullable := &AttributeExpr{Type: String, Nullable: true}
	params := NewMappedAttributeExpr(&AttributeExpr{
		Type: &Object{{Name: "filter", Attribute: nullable}},
	})
	endpoint := &HTTPEndpointExpr{
		MethodExpr: &MethodExpr{Payload: &AttributeExpr{Type: &Object{}}},
		Params:     params,
		Headers:    NewEmptyMappedAttributeExpr(),
		Cookies:    NewEmptyMappedAttributeExpr(),
	}

	verr := newValidationErrors()
	endpoint.validateNullableTransportLocations(verr)
	require.ErrorContains(t, verr, "query and path parameters do not support nullable attributes")
}

func TestHTTPTransportRejectsNullableFormBody(t *testing.T) {
	body := &AttributeExpr{Type: &Object{{
		Name:      "clear",
		Attribute: &AttributeExpr{Type: String, Nullable: true},
	}}}
	endpoint := &HTTPEndpointExpr{
		MethodExpr:  &MethodExpr{Payload: body},
		Body:        body,
		FormRequest: true,
		Params:      NewEmptyMappedAttributeExpr(),
		Headers:     NewEmptyMappedAttributeExpr(),
		Cookies:     NewEmptyMappedAttributeExpr(),
	}

	verr := newValidationErrors()
	endpoint.validateNullableTransportLocations(verr)
	require.ErrorContains(t, verr, "form and multipart bodies do not support nullable attributes")
}

func TestHTTPTransportRejectsNullableErrorMetadata(t *testing.T) {
	nullable := &AttributeExpr{Type: String, Nullable: true}
	response := &HTTPResponseExpr{
		Headers: NewMappedAttributeExpr(&AttributeExpr{Type: &Object{{Name: "retry", Attribute: nullable}}}),
	}
	endpoint := &HTTPEndpointExpr{HTTPErrors: []*HTTPErrorExpr{{Response: response}}}

	verr := newValidationErrors()
	endpoint.validateNullableTransportLocations(verr)
	require.ErrorContains(t, verr, "response headers do not support nullable attributes")
}

func TestHTTPTransportRejectsNullableMapQueryValues(t *testing.T) {
	mapName := "filters"
	filters := &AttributeExpr{Type: &Map{
		KeyType:  &AttributeExpr{Type: String},
		ElemType: &AttributeExpr{Type: String, Nullable: true},
	}}
	payload := &AttributeExpr{Type: &Object{{Name: mapName, Attribute: filters}}}
	endpoint := &HTTPEndpointExpr{
		MethodExpr:     &MethodExpr{Payload: payload},
		MapQueryParams: &mapName,
	}

	verr := newValidationErrors()
	endpoint.validateNullableTransportLocations(verr)
	require.ErrorContains(t, verr, "map query parameters do not support nullable attributes")
}

func TestGRPCTransportRejectsNullableButAllowsAny(t *testing.T) {
	nullableRequest := &AttributeExpr{Type: &Object{{
		Name:      "clear",
		Attribute: &AttributeExpr{Type: String, Nullable: true},
	}}}
	endpoint := &GRPCEndpointExpr{
		MethodExpr:       &MethodExpr{Name: "Update"},
		Request:          nullableRequest,
		StreamingRequest: &AttributeExpr{Type: Empty},
		Metadata:         NewEmptyMappedAttributeExpr(),
		Response:         ensureGRPCResponse(nil),
	}
	endpoint.Response.Prepare()

	verr := newValidationErrors()
	endpoint.validateNullableTransport(verr)
	require.ErrorContains(t, verr, "gRPC request messages do not support nullable attributes")

	endpoint.Request = &AttributeExpr{Type: &Object{{
		Name:      "anything",
		Attribute: &AttributeExpr{Type: Any, Nullable: true},
	}}}
	verr = newValidationErrors()
	endpoint.validateNullableTransport(verr)
	require.Empty(t, verr.Errors)
}

func TestGRPCTransportRejectsExplicitNullableAnyOutsideObjectFields(t *testing.T) {
	tests := []struct {
		name    string
		request *AttributeExpr
	}{
		{
			name:    "root",
			request: &AttributeExpr{Type: Any, Nullable: true},
		},
		{
			name: "array element",
			request: &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{
				Type:     Any,
				Nullable: true,
			}}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			endpoint := &GRPCEndpointExpr{
				MethodExpr:       &MethodExpr{Name: "Update"},
				Request:          test.request,
				StreamingRequest: &AttributeExpr{Type: Empty},
				Metadata:         NewEmptyMappedAttributeExpr(),
				Response:         ensureGRPCResponse(nil),
			}
			endpoint.Response.Prepare()

			verr := newValidationErrors()
			endpoint.validateNullableTransport(verr)
			require.ErrorContains(t, verr, "gRPC request messages do not support nullable attributes")
		})
	}
}

func TestAllowsNullResolvesNamedAny(t *testing.T) {
	anything := &UserTypeExpr{
		TypeName:      "Anything",
		AttributeExpr: &AttributeExpr{Type: Any},
	}
	require.True(t, AllowsNull(&AttributeExpr{Type: anything}))
}

func TestArrayElementsAllowNullHonorsExplicitRequiredElements(t *testing.T) {
	tests := []struct {
		name  string
		array *Array
		want  bool
	}{
		{name: "string defaults non-null", array: &Array{ElemType: &AttributeExpr{Type: String}}, want: false},
		{name: "nullable string", array: &Array{ElemType: &AttributeExpr{Type: String, Nullable: true}}, want: true},
		{name: "any defaults nullable", array: &Array{ElemType: &AttributeExpr{Type: Any}}, want: true},
		{name: "required any", array: &Array{ElemType: &AttributeExpr{Type: Any}, NonNullableElems: true}, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, ArrayElementsAllowNull(test.array))
		})
	}
	require.True(t, ContainsNonNullableCollectionElement(&AttributeExpr{Type: tests[0].array}))
	require.False(t, ContainsNonNullableCollectionElement(&AttributeExpr{Type: tests[1].array}))
	require.True(t, ContainsNonNullableCollectionElement(&AttributeExpr{Type: &Map{
		KeyType:  &AttributeExpr{Type: String},
		ElemType: &AttributeExpr{Type: tests[3].array},
	}}))
}

func TestMapValuesAllowNull(t *testing.T) {
	nonNullable := &Map{
		KeyType:  &AttributeExpr{Type: String},
		ElemType: &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String}}},
	}
	nullable := &Map{
		KeyType: &AttributeExpr{Type: String},
		ElemType: &AttributeExpr{
			Type:     &Array{ElemType: &AttributeExpr{Type: String}},
			Nullable: true,
		},
	}

	require.False(t, MapValuesAllowNull(nonNullable))
	require.True(t, MapValuesAllowNull(nullable))
	require.True(t, ContainsNonNullableCollectionElement(&AttributeExpr{Type: nonNullable}))
	require.True(t, ContainsNonNullableCollectionElement(&AttributeExpr{Type: nullable}))
}

func TestPresenceValidationRejectsConflictingMetadata(t *testing.T) {
	nullableType := &UserTypeExpr{
		TypeName:      "NullableText",
		AttributeExpr: &AttributeExpr{Type: String, Nullable: true},
	}
	tests := []struct {
		name      string
		attribute *AttributeExpr
		message   string
	}{
		{
			name: "inherited nullable custom type",
			attribute: &AttributeExpr{
				Type: nullableType,
				Meta: MetaExpr{"struct:field:type": []string{"string"}},
			},
			message: "conflict with custom field type metadata",
		},
		{
			name: "any omitted from JSON",
			attribute: &AttributeExpr{
				Type: Any,
				Meta: MetaExpr{"struct:tag:json": []string{"-"}},
			},
			message: "conflict with a JSON tag that omits the field",
		},
		{
			name: "nullable string option",
			attribute: &AttributeExpr{
				Type:     Int,
				Nullable: true,
				Meta:     MetaExpr{"struct:tag:json": []string{"value,string"}},
			},
			message: "JSON ,string is not supported",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			verr := test.attribute.validatePresence("", test.attribute)
			require.ErrorContains(t, verr, test.message)
		})
	}
}

func TestAttributeRejectsJSONStringOption(t *testing.T) {
	field := &AttributeExpr{
		Type: Int,
		Meta: MetaExpr{"struct:tag:json": []string{"value,string"}},
	}

	verr := field.validatePresence("", field)
	require.ErrorContains(t, verr, "JSON ,string is not supported")
}

func TestObjectValidationRejectsDesignWireNameCollisions(t *testing.T) {
	object := &Object{
		{Name: "foo", Attribute: &AttributeExpr{Type: String, Meta: MetaExpr{"struct:tag:json": []string{"bar"}}}},
		{Name: "bar", Attribute: &AttributeExpr{Type: String, Meta: MetaExpr{"struct:tag:json": []string{"baz"}}}},
	}
	attribute := &AttributeExpr{Type: object}

	verr := attribute.validateObjectChildren("", attribute, object)
	require.ErrorContains(t, verr, "conflicts with another design field name")
}

func TestExamplesAllowNullInNullableCollections(t *testing.T) {
	tests := []struct {
		name      string
		attribute *AttributeExpr
	}{
		{
			name: "array",
			attribute: &AttributeExpr{
				Type:         &Array{ElemType: &AttributeExpr{Type: String, Nullable: true}},
				UserExamples: []*ExampleExpr{{Value: []any{"value", nil}}},
			},
		},
		{
			name: "map",
			attribute: &AttributeExpr{
				Type: &Map{
					KeyType:  &AttributeExpr{Type: String},
					ElemType: &AttributeExpr{Type: Int, Nullable: true},
				},
				UserExamples: []*ExampleExpr{{Value: map[string]any{"value": nil}}},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			verr := test.attribute.validateExamples("", test.attribute)
			require.Empty(t, verr.Errors)
		})
	}
}

func newValidationErrors() *eval.ValidationErrors {
	return new(eval.ValidationErrors)
}
