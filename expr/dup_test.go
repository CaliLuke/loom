package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDupArrayPreservesNonNullableElems(t *testing.T) {
	original := &Array{
		ElemType:         &AttributeExpr{Type: String},
		NonNullableElems: true,
	}

	duplicated, ok := Dup(original).(*Array)
	if !ok {
		t.Fatalf("expected duplicated array, got %T", duplicated)
	}
	if !duplicated.NonNullableElems {
		t.Fatalf("duplicated array did not preserve non-nullable elements")
	}
	if duplicated.ElemType == original.ElemType {
		t.Fatalf("duplicated array reused original element attribute")
	}
}

func TestDupAttPreservesTitleNullableAndExplicitNullExample(t *testing.T) {
	attribute := &AttributeExpr{
		Type:     String,
		Title:    "Display Name",
		Nullable: true,
		UserExamples: []*ExampleExpr{{
			Summary:      "null",
			ExplicitNull: true,
		}},
	}

	duplicated := DupAtt(attribute)
	if duplicated.Title != "Display Name" {
		t.Errorf("duplicated attribute title is %q, expected %q", duplicated.Title, "Display Name")
	}
	if !duplicated.Nullable {
		t.Error("duplicated attribute did not preserve nullability")
	}
	if len(duplicated.UserExamples) != 1 || !duplicated.UserExamples[0].ExplicitNull {
		t.Error("duplicated attribute did not preserve explicit null example")
	}
}

func TestPresenceValidation(t *testing.T) {
	tests := []struct {
		name      string
		attribute *AttributeExpr
		message   string
	}{
		{
			name: "nullable map key",
			attribute: &AttributeExpr{Type: &Map{
				KeyType:  &AttributeExpr{Type: String, Nullable: true},
				ElemType: &AttributeExpr{Type: String},
			}},
			message: "map keys cannot be nullable",
		},
		{
			name: "nullable required array element",
			attribute: &AttributeExpr{Type: &Array{
				ElemType:         &AttributeExpr{Type: String, Nullable: true},
				NonNullableElems: true,
			}},
			message: "array elements cannot be both nullable and required",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validated = make(map[*AttributeExpr]bool)
			errors := test.attribute.Validate("attribute", test.attribute)
			require.ErrorContains(t, errors, test.message)
		})
	}
}

func TestPrepareNormalizesLegacyNullableMetadata(t *testing.T) {
	tests := []struct {
		name     string
		typeName DataType
		meta     MetaExpr
	}{
		{
			name:     "nullable pair",
			typeName: String,
			meta: MetaExpr{
				"openapi:nullable":  []string{"true"},
				"struct:field:type": []string{"loom.Nullable[string]", "github.com/CaliLuke/loom/pkg", "loom"},
			},
		},
		{
			name:     "unconstrained wrapper",
			typeName: Any,
			meta: MetaExpr{
				"struct:field:type": []string{"loom.Nullable[any]", "github.com/CaliLuke/loom/pkg", "loom"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			attribute := &AttributeExpr{Type: test.typeName, Meta: test.meta}
			attribute.Prepare()
			require.True(t, attribute.Nullable)
			require.NotContains(t, attribute.Meta, "openapi:nullable")
			require.NotContains(t, attribute.Meta, "struct:field:type")
		})
	}
}
