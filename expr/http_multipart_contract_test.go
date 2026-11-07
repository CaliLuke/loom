package expr

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/httpencoding"
)

func TestHTTPMultipartBodyEncoding(t *testing.T) {
	union := &AttributeExpr{Type: &Union{Untagged: true, Values: []*NamedAttributeExpr{
		{Name: "Text", Attribute: &AttributeExpr{Type: String}},
		{Name: "Number", Attribute: &AttributeExpr{Type: Int}},
	}}}
	binary := &UserTypeExpr{TypeName: "Blob", AttributeExpr: &AttributeExpr{Type: Bytes}}
	body := &AttributeExpr{Type: &Object{
		{Name: "choice:shared", Attribute: &AttributeExpr{Type: union.Type, Meta: MetaExpr{"struct:tag:form": {"choice"}}}},
		{Name: "file:shared", Attribute: &AttributeExpr{Type: binary, Meta: MetaExpr{"struct:tag:form": {"upload"}}}},
		{Name: "ignored", Attribute: &AttributeExpr{Type: String, Meta: MetaExpr{"struct:tag:json": {"ignored"}}}},
	}, Meta: MetaExpr{"openapi:encoding:shared:contentType": {"application/json"}}}
	contract, err := HTTPMultipartBodyEncoding(body)
	require.NoError(t, err)
	require.False(t, contract.Custom)
	require.Equal(t, "choice", contract.Fields["choice:shared"].Name)
	require.Equal(t, httpencoding.JSONPart, contract.Fields["choice:shared"].Codec.Kind)
	require.Equal(t, "upload", contract.Fields["file:shared"].Name)
	require.Equal(t, httpencoding.BinaryPart, contract.Fields["file:shared"].Codec.Kind)
	require.Equal(t, "application/json", contract.Fields["file:shared"].Codec.ContentType)
	require.True(t, contract.Fields["ignored"].Omitted)
	require.Empty(t, contract.Branches)
}

func TestHTTPMultipartBranchContext(t *testing.T) {
	arrays := &AttributeExpr{Type: &Union{Untagged: true, Values: []*NamedAttributeExpr{
		{Name: "Texts", Attribute: &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String}}}},
		{Name: "Numbers", Attribute: &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: Int}}}},
	}}}
	arrays.Meta = MetaExpr{"http:encoding:sourceType": {"array"}}
	declaration := &AttributeExpr{Type: &Object{{Name: "value", Attribute: arrays}}}
	declaration.CloseObject()
	leaf := &AttributeExpr{Type: &UserTypeExpr{TypeName: "Arrays", AttributeExpr: declaration}}
	nested := &AttributeExpr{Type: &Union{Untagged: true, Values: []*NamedAttributeExpr{{Name: "Arrays", Attribute: leaf}, {Name: "Empty", Attribute: &AttributeExpr{Type: &Object{}}}}}}
	root := &AttributeExpr{Type: &Union{Untagged: true, Values: []*NamedAttributeExpr{{Name: "Nested", Attribute: nested}, {Name: "Binary", Attribute: &AttributeExpr{Type: &Object{{Name: "value", Attribute: &AttributeExpr{Type: Bytes}}}}}}}, Meta: MetaExpr{"openapi:encoding:value:style": {"form"}, "openapi:encoding:value:explode": {"true"}}}
	contract, err := HTTPMultipartBodyEncoding(root)
	require.NoError(t, err)
	require.Equal(t, httpencoding.TextPart, contract.Branches[0].Branches[0].Fields["value"].Codec.Kind)
	require.Equal(t, ObjectClosed, contract.Branches[0].Branches[0].ObjectOpenness)
	require.Empty(t, contract.Branches[0].Branches[1].Fields)
	require.Equal(t, httpencoding.BinaryPart, contract.Branches[1].Fields["value"].Codec.Kind)
	require.Equal(t, "application/octet-stream", contract.Branches[1].Fields["value"].Codec.ContentType)
	require.NotContains(t, contract.Branches[1].Metadata, "http:encoding:sourceType")
	standalone, err := HTTPMultipartBodyEncoding(leaf)
	require.NoError(t, err)
	require.Equal(t, httpencoding.BranchPart, standalone.Fields["value"].Codec.Kind)
	for _, codec := range standalone.Fields["value"].Codec.Branches {
		require.Equal(t, httpencoding.TextPart, codec.Kind)
		require.Equal(t, httpencoding.Items, codec.Application)
	}
	require.NotContains(t, declaration.Meta, "openapi:encoding:value:style")
}

func TestHTTPMultipartContractAdmission(t *testing.T) {
	for _, tc := range []struct {
		name   string
		body   *AttributeExpr
		custom bool
		want   string
	}{
		{"object", &AttributeExpr{Type: &Object{}}, false, ""},
		{"ordinary scalar hook", &AttributeExpr{Type: String}, true, ""},
		{"ordinary Any untyped rejection", &AttributeExpr{Type: &Object{{Name: "value", Attribute: &AttributeExpr{Type: Any}}}}, false, "application/octet-stream"},
		{"nullable root", &AttributeExpr{Type: &Object{}, Nullable: true}, false, "non-nullable"},
		{"tagged root", &AttributeExpr{Type: &Union{}}, false, "constructor unions"},
		{"array alternative", &AttributeExpr{Type: &Union{Untagged: true, Values: []*NamedAttributeExpr{{Name: "List", Attribute: &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String}}}}}}}, false, "List"},
		{"nested multipart encoding", &AttributeExpr{Type: &Object{}, Meta: MetaExpr{"openapi:itemEncoding:contentType": {"application/json"}}}, false, "no generated field contract"},
		{"typed object", &AttributeExpr{Type: &Object{}, AdditionalAttributes: &AttributeExpr{Type: Int}}, false, "AdditionalAttributes"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			contract, err := HTTPMultipartBodyEncoding(tc.body)
			if tc.want != "" {
				require.ErrorContains(t, err, tc.want)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.custom, contract.Custom)
		})
	}
}

func TestHTTPMultipartJSONValues(t *testing.T) {
	tree := &UserTypeExpr{TypeName: "Tree", AttributeExpr: &AttributeExpr{Type: &Object{}}}
	*AsObject(tree.Type) = Object{{Name: "next", Attribute: &AttributeExpr{Type: tree}}}
	for _, tc := range []struct {
		name      string
		field     *AttributeExpr
		media     string
		wantError bool
	}{
		{name: "nullable JSON", field: &AttributeExpr{Type: String, Nullable: true}, media: "application/json"},
		{name: "nullable text", field: &AttributeExpr{Type: String, Nullable: true}, media: "text/plain", wantError: true},
		{name: "nullable binary", field: &AttributeExpr{Type: Bytes, Nullable: true}, media: "application/json", wantError: true},
		{name: "Any JSON", field: &AttributeExpr{Type: Any}, media: "application/json"},
		{name: "recursive JSON", field: &AttributeExpr{Type: tree}, media: "application/json"},
		{name: "tagged JSON", field: &AttributeExpr{Type: &Union{Values: []*NamedAttributeExpr{
			{Name: "Text", Attribute: &AttributeExpr{Type: String}},
			{Name: "Number", Attribute: &AttributeExpr{Type: Int}},
		}}}, media: "application/json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := &AttributeExpr{Type: &Object{{Name: "value", Attribute: tc.field}}, Meta: MetaExpr{
				"openapi:encoding:value:contentType": {tc.media},
			}}
			contract, err := HTTPMultipartBodyEncoding(body)
			if tc.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.False(t, contract.Custom)
			require.Equal(t, httpencoding.JSONPart, contract.Fields["value"].Codec.Kind)
		})
	}
}

func TestHTTPMultipartCustomAndNamespaces(t *testing.T) {
	opaque := &AttributeExpr{Type: &Object{{Name: "value", Attribute: &AttributeExpr{Type: String, Meta: MetaExpr{
		"struct:field:type": {"Custom", "example.com/custom"},
	}}}}}
	contract, err := HTTPMultipartBodyEncoding(opaque)
	require.NoError(t, err)
	require.True(t, contract.Custom)
	require.Nil(t, contract.Fields)
	root := &AttributeExpr{Type: &Union{Untagged: true, Values: []*NamedAttributeExpr{{Name: "Custom", Attribute: opaque}}}}
	_, err = HTTPMultipartBodyEncoding(root)
	require.ErrorContains(t, err, "generated codecs")
	collision := &AttributeExpr{Type: &Object{
		{Name: "first:shared", Attribute: &AttributeExpr{Type: String}},
		{Name: "second:shared", Attribute: &AttributeExpr{Type: Int}},
	}}
	_, err = HTTPMultipartBodyEncoding(collision)
	require.ErrorContains(t, err, "overlap")
	cycle := &Union{Untagged: true}
	root = &AttributeExpr{Type: cycle}
	cycle.Values = []*NamedAttributeExpr{{Name: "Loop", Attribute: root}}
	_, err = HTTPMultipartBodyEncoding(root)
	require.ErrorContains(t, err, "recursive root")
}

func TestHTTPMultipartApplication(t *testing.T) {
	recursive := &UserTypeExpr{TypeName: "RecursiveArray", AttributeExpr: &AttributeExpr{}}
	recursive.Type = &Array{ElemType: &AttributeExpr{Type: recursive}}
	arrays := &Union{Untagged: true, Values: []*NamedAttributeExpr{
		{Name: "Texts", Attribute: &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String}}}},
		{Name: "Numbers", Attribute: &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: Int}}}},
	}}
	for _, tc := range []struct {
		name        string
		field       *AttributeExpr
		media       string
		application httpencoding.Application
		kind        httpencoding.PartKind
	}{
		{"native strings", &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String}}}, "", httpencoding.Items, httpencoding.TextPart},
		{"native binary items", &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: Bytes}}}, "", httpencoding.Items, httpencoding.BinaryPart},
		{"native objects", &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: &Object{}}}}, "", httpencoding.Items, httpencoding.JSONPart},
		{"recursive array items", &AttributeExpr{Type: recursive}, "application/json", httpencoding.Items, httpencoding.JSONPart},
		{"nested arrays", &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String}}}}}, "", httpencoding.Items, httpencoding.JSONPart},
		{"nullable items JSON", &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String, Nullable: true}}}, "application/json", httpencoding.Items, httpencoding.JSONPart},
		{"Any items JSON", &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: Any}}}, "application/json", httpencoding.Items, httpencoding.JSONPart},
		{"typeless array union JSON", &AttributeExpr{Type: arrays, Meta: MetaExpr{"http:encoding:sourceType": {"absent"}}}, "application/json", httpencoding.Value, httpencoding.JSONPart},
		{"source array union JSON", &AttributeExpr{Type: arrays, Meta: MetaExpr{"http:encoding:sourceType": {"array"}}}, "application/json", httpencoding.Items, httpencoding.BranchPart},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := &AttributeExpr{Type: &Object{{Name: "value", Attribute: tc.field}}}
			if tc.media != "" {
				body.Meta = MetaExpr{"openapi:encoding:value:contentType": {tc.media}}
			}
			contract, err := HTTPMultipartBodyEncoding(body)
			require.NoError(t, err)
			codec := contract.Fields["value"].Codec
			require.Equal(t, tc.application, codec.Application)
			require.Equal(t, tc.kind, codec.Kind)
		})
	}
}

func TestHTTPMultipartStyleOverridesAuthoredContent(t *testing.T) {
	body := &AttributeExpr{Type: &Object{{Name: "values", Attribute: &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: Int}}}}}, Meta: MetaExpr{
		"openapi:encoding:values:contentType": {"application/json"},
		"openapi:encoding:values:style":       {"form"},
		"openapi:encoding:values:explode":     {"false"},
	}}
	contract, err := HTTPMultipartBodyEncoding(body)
	require.NoError(t, err)
	codec := contract.Fields["values"].Codec
	require.Equal(t, httpencoding.Value, codec.Application)
	require.Equal(t, httpencoding.TextPart, codec.Kind)
	require.NotNil(t, codec.StyleCodec)
	require.Equal(t, httpencoding.Form, codec.StyleCodec.Style)
	require.False(t, codec.StyleCodec.Explode)
}

func TestHTTPMultipartStyledNamespace(t *testing.T) {
	body := &AttributeExpr{Type: &Object{
		{Name: "filter", Attribute: &AttributeExpr{Type: &Object{{Name: "x", Attribute: &AttributeExpr{Type: String}}}}},
		{Name: "x", Attribute: &AttributeExpr{Type: String}},
	}, Meta: MetaExpr{
		"openapi:encoding:filter:style":   {"form"},
		"openapi:encoding:filter:explode": {"true"},
	}}
	_, err := HTTPMultipartBodyEncoding(body)
	require.ErrorContains(t, err, "overlap")
}
