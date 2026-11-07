package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPFormFieldTags(t *testing.T) {
	for _, tc := range []struct {
		name    string
		meta    MetaExpr
		wire    string
		omitted bool
	}{
		{"default", nil, "mapped", false},
		{"renamed", MetaExpr{"struct:tag:form": {"wire"}}, "wire", false},
		{"options", MetaExpr{"struct:tag:form": {"wire", "omitempty"}}, "wire", false},
		{"empty", MetaExpr{"struct:tag:form": {""}}, "", true},
		{"skip", MetaExpr{"struct:tag:form": {"-"}}, "-", true},
		{"full JSON", MetaExpr{"struct:tag:json": {"jsonName"}}, "", true},
		{"JSON name", MetaExpr{"struct:tag:json:name": {"jsonName"}}, "mapped", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			field := &AttributeExpr{Type: String, Meta: tc.meta}
			body := &AttributeExpr{Type: &Object{{Name: "value:mapped", Attribute: field}}}
			alias := &AttributeExpr{Type: &UserTypeExpr{TypeName: "Body", AttributeExpr: body}}
			contract, err := HTTPFormBodyEncoding(alias)
			require.NoError(t, err)
			require.Equal(t, tc.wire, contract.Fields["value:mapped"].Name)
			require.Equal(t, tc.omitted, contract.Fields["value:mapped"].Omitted)
		})
	}
}

func TestHTTPFormFieldTagNamespaces(t *testing.T) {
	body := &AttributeExpr{Type: &Object{
		{Name: "value", Attribute: &AttributeExpr{Type: String, Meta: MetaExpr{"struct:tag:form": {"wire"}}}},
		{Name: "wire", Attribute: &AttributeExpr{Type: String}},
	}}
	_, err := HTTPFormBodyEncoding(body)
	require.ErrorContains(t, err, "overlap")
}

func TestHTTPFormTagCodecNamespaceMatrix(t *testing.T) {
	union := &Union{Untagged: true, Values: []*NamedAttributeExpr{
		{Name: "Text", Attribute: &AttributeExpr{Type: String}},
		{Name: "List", Attribute: &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String}}}},
	}}
	for _, tc := range []struct {
		name     string
		typ      DataType
		metadata MetaExpr
		peer     string
		omitPeer bool
		rejected bool
	}{
		{"JSON union", union, MetaExpr{"openapi:encoding:value:contentType": {"application/json"}}, "wire", false, true},
		{"styled union", union, MetaExpr{"openapi:encoding:value:style": {"form"}, "openapi:encoding:value:explode": {"false"}}, "wire", false, true},
		{"styled ordinary", &Array{ElemType: &AttributeExpr{Type: Int}}, MetaExpr{"openapi:encoding:value:style": {"pipeDelimited"}, "openapi:encoding:value:explode": {"false"}}, "wire", false, true},
		{"deep prefix", &Map{KeyType: &AttributeExpr{Type: String}, ElemType: &AttributeExpr{Type: Int}}, MetaExpr{"openapi:encoding:value:style": {"deepObject"}}, "wire[key]", false, true},
		{"omitted sibling", union, MetaExpr{"openapi:encoding:value:contentType": {"application/json"}}, "wire", true, false},
		{"disjoint", union, MetaExpr{"openapi:encoding:value:contentType": {"application/json"}}, "other", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			peer := &AttributeExpr{Type: String, Meta: MetaExpr{"struct:tag:form": {tc.peer}}}
			if tc.omitPeer {
				peer.Meta["struct:tag:form"] = []string{"-"}
			}
			body := &AttributeExpr{Type: &Object{
				{Name: "value", Attribute: &AttributeExpr{Type: tc.typ, Meta: MetaExpr{"struct:tag:form": {"wire"}}}},
				{Name: "peer", Attribute: peer},
			}, Meta: tc.metadata}
			_, err := HTTPFormBodyEncoding(body)
			if tc.rejected {
				require.ErrorContains(t, err, "overlap")
			} else {
				require.NoError(t, err)
			}
		})
	}
}
