package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

const protoValidationFieldMetadataHarness = `package roundtrip

import (
	"testing"

	"github.com/stretchr/testify/require"

	probe "%[1]s/gen/probe"
	"%[1]s/gen/grpc/probe/client"
	"%[1]s/gen/grpc/probe/server"
)

func TestValidationUsesProtobufField(t *testing.T) {
	for _, tc := range []struct {
		name string
		valid bool
	}{{"", false}, {"x", false}, {"valid", true}} {
		t.Run(tc.name, func(t *testing.T) {
			value := &probe.Value{Custom: tc.name}
			request := client.NewProtoEchoRequest(value)
			response := server.NewProtoEchoResponse(value)
			require.Equal(t, tc.name, request.Name)
			require.Equal(t, tc.name, response.Name)
			require.Equal(t, value, server.NewEchoPayload(request))
			require.Equal(t, value, client.NewEchoResult(response))
			for _, err := range []error{server.ValidateEchoRequest(request), client.ValidateEchoResponse(response)} {
				if tc.valid {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
				}
			}
		})
	}
}
`

func TestProtoValidationIgnoresServiceFieldNames(t *testing.T) {
	for _, tc := range []struct {
		name, field, upper, lower string
	}{
		{"plain", "label", "Label", "label"},
		{"snake case", "user_name", "UserName", "userName"},
		{"camel case", "fooBar", "FooBar", "fooBar"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			minimum := 2
			field := &expr.AttributeExpr{
				Type:       expr.String,
				Meta:       expr.MetaExpr{"struct:field:name": {"Custom"}},
				Validation: &expr.ValidationExpr{MinLength: &minimum},
			}
			object := &expr.AttributeExpr{
				Type:       &expr.Object{{Name: tc.field, Attribute: field}},
				Validation: &expr.ValidationExpr{Required: []string{tc.field}},
			}
			message := &expr.UserTypeExpr{TypeName: "Payload", AttributeExpr: object}
			context := protoBufTypeContext("pb", codegen.NewNameScope(), false)
			require.Equal(t, tc.upper, context.Scope.Field(field, tc.field, true))
			require.Equal(t, tc.lower, context.Scope.Field(field, tc.field, false))
			validation := codegen.ValidationCode(object, message, context, true, false, false, "message")
			require.Contains(t, validation, "message."+tc.upper)
			require.NotContains(t, validation, "message.Custom")
			require.Equal(t, []string{"Custom"}, field.Meta["struct:field:name"], "protobuf naming must not mutate the service field contract")
		})
	}
}

func TestProtoValidationFieldMetadataGenerated(t *testing.T) {
	runGeneratedRoundTrip(t, "example.com/fieldmetadata", protoValidationFieldMetadataDSL, protoValidationFieldMetadataHarness)
}

func protoValidationFieldMetadataDSL() {
	value := Type("Value", func() {
		Field(1, "name", String, func() {
			Meta("struct:field:name", "Custom")
			MinLength(2)
		})
		Required("name")
	})
	Service("probe", func() {
		Method("echo", func() {
			Payload(value)
			Result(value)
			GRPC(func() {})
		})
	})
}
