package expr_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/expr/testdata"
)

func TestGRPCEndpointValidation(t *testing.T) {
	cases := map[string]struct {
		DSL    func()
		Errors []string
	}{
		"endpoint-with-any-type": {
			DSL:    testdata.GRPCEndpointWithAnyType,
			Errors: []string{}, // Any type is now supported in gRPC
		},
		"endpoint-with-untagged-fields": {
			DSL: testdata.GRPCEndpointWithUntaggedFields,
			Errors: []string{`service "Service" gRPC endpoint "Method": attribute "req_not_field" does not have "rpc:tag" defined in the meta, use "Field" to define the attribute of a type used in a gRPC method
service "Service" gRPC endpoint "Method": attribute "resp_not_field" does not have "rpc:tag" defined in the meta, use "Field" to define the attribute of a type used in a gRPC method`,
			},
		},
		"endpoint-with-repeated-field-tags": {
			DSL: testdata.GRPCEndpointWithRepeatedFieldTags,
			Errors: []string{`service "Service" gRPC endpoint "Method": field number 1 in attribute "key_dup_id" already exists for attribute "key"
service "Service" gRPC endpoint "Method": field number 2 in attribute "key_dup_id" already exists for attribute "key"`,
			},
		},
		"endpoint-with-reference-types-field-inheritance": {
			DSL:    testdata.GRPCEndpointWithReferenceTypes,
			Errors: []string{},
		},
		"endpoint-with-extended-types": {
			DSL:    testdata.GRPCEndpointWithExtendedTypes,
			Errors: []string{},
		},
		"endpoint-with-inherit-error": {
			DSL:    testdata.GRPCEndpointWithInheritErrorDSL,
			Errors: []string{},
		},
		"endpoint-with-constructor-union-field": {
			DSL:    testdata.GRPCEndpointWithConstructorUnionField,
			Errors: []string{},
		},
		"endpoint-with-constructor-union-field-collision": {
			DSL: testdata.GRPCEndpointWithConstructorUnionFieldCollision,
			Errors: []string{
				`service "Service" gRPC endpoint "Method": field number 3 in attribute "next" already exists for attribute "pick.Other"; a OneOf passed to Field numbers its branches consecutively from the field number`,
			},
		},
		"endpoint-with-untagged-union-branches": {
			DSL: testdata.GRPCEndpointWithUntaggedUnionBranches,
			Errors: []string{
				`service "Service" gRPC endpoint "Method": union branch "Leaf" of attribute "pick" does not have "rpc:tag" defined in the meta, use "Field" to define each branch of a OneOf block or pass the OneOf to "Field" to number its branches
service "Service" gRPC endpoint "Method": union branch "Other" of attribute "pick" does not have "rpc:tag" defined in the meta, use "Field" to define each branch of a OneOf block or pass the OneOf to "Field" to number its branches
service "Service" gRPC endpoint "Method": union branch "fast" of attribute "mode" does not have "rpc:tag" defined in the meta, use "Field" to define each branch of a OneOf block or pass the OneOf to "Field" to number its branches`,
			},
		},
		"request-message-missing-second-attribute": {
			DSL: testdata.GRPCRequestMessageWithMissingSecondAttribute,
			Errors: []string{
				`service "Service" gRPC endpoint "Method": Request message attribute "missing" is not found in Payload`,
			},
		},
		"request-message-untagged-second-attribute": {
			DSL: testdata.GRPCRequestMessageWithUntaggedSecondAttribute,
			Errors: []string{
				`service "Service" gRPC endpoint "Method": attribute "second" does not have "rpc:tag" defined in the meta, use "Field" to define the attribute of a type used in a gRPC method`,
			},
		},
		"request-message-duplicate-second-tag": {
			DSL: testdata.GRPCRequestMessageWithDuplicateSecondTag,
			Errors: []string{
				`service "Service" gRPC endpoint "Method": field number 1 in attribute "second" already exists for attribute "first"`,
			},
		},
		"request-message-multiple-attributes": {
			DSL:    testdata.GRPCRequestMessageWithMultipleAttributes,
			Errors: []string{},
		},
		"response-message-missing-second-attribute": {
			DSL: testdata.GRPCResponseMessageWithMissingSecondAttribute,
			Errors: []string{
				`service "Service" gRPC endpoint "Method": Response message attribute "missing" is not found in Result`,
			},
		},
		"response-message-untagged-second-attribute": {
			DSL: testdata.GRPCResponseMessageWithUntaggedSecondAttribute,
			Errors: []string{
				`service "Service" gRPC endpoint "Method": attribute "second" does not have "rpc:tag" defined in the meta, use "Field" to define the attribute of a type used in a gRPC method`,
			},
		},
		"response-message-duplicate-second-tag": {
			DSL: testdata.GRPCResponseMessageWithDuplicateSecondTag,
			Errors: []string{
				`service "Service" gRPC endpoint "Method": field number 1 in attribute "second" already exists for attribute "first"`,
			},
		},
		"response-message-multiple-attributes": {
			DSL:    testdata.GRPCResponseMessageWithMultipleAttributes,
			Errors: []string{},
		},
		"endpoint-with-mixed-results": {
			DSL: testdata.GRPCEndpointWithMixedResults,
			Errors: []string{
				`service "Service" gRPC endpoint "Method": gRPC methods cannot define both Result and StreamingResult with different types because a gRPC server stream sends only the streaming result`,
			},
		},
		"endpoint-with-streaming-result-only": {
			DSL:    testdata.GRPCEndpointWithStreamingResultOnly,
			Errors: []string{},
		},
		"endpoint-with-named-union-field": {
			DSL:    testdata.GRPCEndpointWithNamedUnionField,
			Errors: []string{},
		},
		"endpoint-with-union-collections": {
			DSL: testdata.GRPCEndpointWithUnionCollections,
			Errors: []string{
				`service "Service" method "Method": union type Choice is an array element, not supported by gRPC; wrap the union in a Type with one Field and use that type as the element
service "Service" method "Method": union type LeafOrOther is an array element, not supported by gRPC; wrap the union in a Type with one Field and use that type as the element
service "Service" method "Method": union type Choice is a map value, not supported by gRPC; wrap the union in a Type with one Field and use that type as the value
service "Service" method "Method": union type Choice is an array element, not supported by gRPC; wrap the union in a Type with one Field and use that type as the element`,
			},
		},
		"endpoint-union-containing-any": {
			DSL: testdata.GRPCEndpointWithUnionContainingAny,
			Errors: []string{
				`service "Service" method "MethodUnion": union type choice has map elements, not supported by gRPC; wrap the map in a Type with one Field and use that type as the branch`,
			},
		},
		"endpoint-union-collection-branches": {
			DSL: testdata.GRPCEndpointWithUnionCollectionBranches,
			Errors: []string{
				`service "Service" method "Method": union type IndexOrLeaf has map elements, not supported by gRPC; wrap the map in a Type with one Field and use that type as the branch
service "Service" method "Method": union type inline has map elements, not supported by gRPC; wrap the map in a Type with one Field and use that type as the branch`,
			},
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if len(c.Errors) == 0 {
				expr.RunDSL(t, c.DSL)
			} else {
				var errs []error

				err := expr.RunInvalidDSL(t, c.DSL)
				if err != nil {
					var merr eval.MultiError
					if errors.As(err, &merr) {
						for _, e := range merr {
							errs = append(errs, e.GoError)
						}
					} else {
						errs = append(errs, err)
					}
				}

				if len(c.Errors) != len(errs) {
					t.Errorf("%s: got %d, expected the number of error values to match %d", name, len(errs), len(c.Errors))
				} else {
					for i, err := range errs {
						got := stripValidationLocations(err.Error())
						if got != c.Errors[i] {
							t.Errorf("%s:\ngot \t%q,\nexpected\t%q at index %d", name, got, c.Errors[i], i)
						}
					}
				}
			}
		})
	}
}

// TestGRPCEndpointStreamingPayloadKeepsInitialRequest regresses Finalize so
// that methods declaring both Payload and StreamingPayload keep the ordinary
// payload fields on the request message rather than rewriting them into
// gRPC metadata.
func TestGRPCEndpointStreamingPayloadKeepsInitialRequest(t *testing.T) {
	root := expr.RunDSL(t, testdata.GRPCEndpointWithStreamingPayloadInitialRequest)
	grpcSvc := root.API.GRPC.Service("Service")
	require.NotNil(t, grpcSvc)
	require.Len(t, grpcSvc.GRPCEndpoints, 1)

	endpoint := grpcSvc.GRPCEndpoints[0]
	req := expr.AsObject(endpoint.Request.Type)
	require.NotNil(t, req)
	require.NotNil(t, req.Attribute("repository_id"))
	require.NotNil(t, req.Attribute("version_ref"))
	require.True(t, endpoint.Metadata.IsEmpty())
}
