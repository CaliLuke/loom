package expr

import "testing"

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
