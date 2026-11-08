package expr

import (
	"testing"

	"github.com/CaliLuke/loom/internal/httpencoding"
	"github.com/stretchr/testify/require"
)

func TestHTTPFormTaggedBranchContracts(t *testing.T) {
	child := &AttributeExpr{Type: &Object{{Name: "code", Attribute: &AttributeExpr{Type: String}}}}
	object := &AttributeExpr{Type: &Object{
		{Name: "details", Attribute: child},
		{Name: "renamed", Attribute: &AttributeExpr{Type: String, Meta: MetaExpr{"struct:tag:form": {"wire"}}}},
		{Name: "hidden", Attribute: &AttributeExpr{Type: String, Meta: MetaExpr{"struct:tag:form": {"-"}}}},
	}}
	array := &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String}}}
	mapping := &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: String}, ElemType: &AttributeExpr{Type: String}}}
	union := &Union{TypeName: "Choice", TypeKey: "kind", ValueKey: "data", Values: []*NamedAttributeExpr{
		{Name: "Record", Attribute: object}, {Name: "Array", Attribute: array}, {Name: "Map", Attribute: mapping},
	}}
	body := &AttributeExpr{Type: union}
	contract, err := HTTPFormBodyEncoding(body)
	require.NoError(t, err)
	require.Empty(t, contract.Branches, "tagged dispatch never becomes exact-one matching")
	require.Len(t, contract.TaggedBranches, 3)
	require.Equal(t, "Record", contract.TaggedBranches[0].Tag)
	fields := contract.TaggedBranches[0].Object.Fields
	require.Equal(t, httpencoding.JSON, fields["details"].Style)
	require.Equal(t, "wire", fields["renamed"].Name)
	require.True(t, fields["hidden"].Omitted)
	require.Equal(t, httpencoding.Form, contract.TaggedBranches[1].Value.Style)
	require.Equal(t, httpencoding.JSON, contract.TaggedBranches[2].Value.Style)
	require.Equal(t, "data", contract.TaggedBranches[2].Value.Name)
	require.Same(t, child, AsObject(object.Type).Attribute("details"))
	require.Nil(t, child.Meta, "capture does not rename nested JSON source members")
}

func TestHTTPFormTaggedBranchRejectedContracts(t *testing.T) {
	for _, tc := range []struct {
		name    string
		branch  *AttributeExpr
		message string
	}{
		{"tag collision", &AttributeExpr{Type: &Object{{Name: "kind", Attribute: &AttributeExpr{Type: String}}}}, "conflicts with discriminator"},
		{"renamed tag collision", &AttributeExpr{Type: &Object{{Name: "other", Attribute: &AttributeExpr{Type: String, Meta: MetaExpr{"struct:tag:form": {"kind"}}}}}}, "conflicts with discriminator"},
		{"nullable flattened object", &AttributeExpr{Type: &Object{}, Nullable: true}, "non-nullable"},
		{"nullable repeated array", &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String}}, Nullable: true}, "nullable array fields"},
		{"source type absent map", &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: String}, ElemType: &AttributeExpr{Type: String}}, Meta: MetaExpr{"http:encoding:sourceType": {"absent"}}}, "no equivalent codec"},
		{"typed additional members", &AttributeExpr{Type: &Object{}, AdditionalAttributes: &AttributeExpr{Type: String}}, "AdditionalAttributes"},
		{"ignored style", &AttributeExpr{Type: &Object{{Name: "field", Attribute: &AttributeExpr{Type: String}}}, Meta: MetaExpr{"openapi:encoding:field:style": {"form"}}}, "does not support Encoding Object overrides"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := HTTPFormBodyEncoding(&AttributeExpr{Type: &Union{TypeKey: "kind", ValueKey: "data", Values: []*NamedAttributeExpr{{Name: "Branch", Attribute: tc.branch}}}})
			require.ErrorContains(t, err, tc.message)
		})
	}
}

func TestHTTPFormBranchEmptyPresence(t *testing.T) {
	for _, tc := range []struct {
		name  string
		body  *AttributeExpr
		empty bool
	}{
		{"repeated scalar array", &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String}}}, true},
		{"nullable array", &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String}}, Nullable: true}, false},
		{"JSON object array", &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: &Object{}}}}, true},
		{"JSON map", &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: String}, ElemType: &AttributeExpr{Type: String}}}, false},
		{"optional object", &AttributeExpr{Type: &Object{}}, true},
		{"required object", &AttributeExpr{Type: &Object{{Name: "id", Attribute: &AttributeExpr{Type: String}}}, Validation: &ValidationExpr{Required: []string{"id"}}}, false},
		{"text", &AttributeExpr{Type: String}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			captured, err := HTTPFormTaggedBranchEncoding(tc.body, "value")
			if IsNullable(tc.body) {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.empty, captured.AllowsEmpty)
		})
	}
}

func TestTaggedFormPreservesNestedUntaggedRejection(t *testing.T) {
	inner := &AttributeExpr{Type: &Union{Untagged: true, Values: []*NamedAttributeExpr{
		{Name: "Text", Attribute: &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String}}}},
		{Name: "Number", Attribute: &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: Int}}}},
	}}}
	for _, member := range []bool{false, true} {
		branch := inner
		if member {
			branch = &AttributeExpr{Type: &Object{{Name: "value", Attribute: inner}}}
		}
		body := &AttributeExpr{Type: &Union{Values: []*NamedAttributeExpr{{Name: "Selected", Attribute: branch}}}}
		_, err := HTTPFormBodyEncoding(body)
		require.ErrorContains(t, err, "tagged form bodies do not support nested untagged OneOf values")
	}
}
