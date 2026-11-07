package expr

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/httpencoding"
)

func TestHTTPFormBodyEncodingContext(t *testing.T) {
	value := &AttributeExpr{Type: &Union{Untagged: true, Values: []*NamedAttributeExpr{
		{Name: "Text", Attribute: &AttributeExpr{Type: String}},
		{Name: "Number", Attribute: &AttributeExpr{Type: Int}},
	}}}
	declaration := &AttributeExpr{Type: &Object{{Name: "value", Attribute: value}}, Meta: MetaExpr{
		"openapi:encoding:value:contentType": {"text/plain"},
	}}
	named := &UserTypeExpr{TypeName: "Record", AttributeExpr: declaration}
	branch := &AttributeExpr{Type: named}
	inner := &AttributeExpr{Type: &Union{Untagged: true, Values: []*NamedAttributeExpr{
		{Name: "Record", Attribute: branch},
		{Name: "Empty", Attribute: &AttributeExpr{Type: &Object{}}},
	}}}
	body := &AttributeExpr{Type: &Union{Untagged: true, Values: []*NamedAttributeExpr{
		{Name: "Nested", Attribute: inner},
		{Name: "Other", Attribute: &AttributeExpr{Type: &Object{{Name: "other", Attribute: &AttributeExpr{Type: String}}}}},
	}}, Meta: MetaExpr{"openapi:encoding:value:contentType": {"application/json"}}}
	contract, err := HTTPFormBodyEncoding(body)
	require.NoError(t, err)
	require.Len(t, contract.Branches, 2)
	require.Equal(t, httpencoding.JSON, contract.Branches[0].Branches[0].Fields["value"].Style)
	require.Empty(t, contract.Branches[0].Branches[1].Fields)
	require.Equal(t, httpencoding.Form, contract.Branches[1].Fields["other"].Style)
	standalone, err := HTTPFormBodyEncoding(branch)
	require.NoError(t, err)
	require.Equal(t, httpencoding.Form, standalone.Fields["value"].Style)
	require.Equal(t, MetaExpr{"openapi:encoding:value:contentType": {"text/plain"}}, declaration.Meta)
	require.Nil(t, branch.Meta)
	contract.Branches[0].Branches[0].Metadata["openapi:encoding:value:contentType"][0] = "changed"
	require.Equal(t, []string{"application/json"}, body.Meta["openapi:encoding:value:contentType"])
	require.Equal(t, []string{"application/json"}, contract.Metadata["openapi:encoding:value:contentType"])
}

func TestHTTPFormBodyEncodingGrammar(t *testing.T) {
	for _, tc := range []struct {
		name string
		body *AttributeExpr
		want string
	}{
		{"empty object", &AttributeExpr{Type: &Object{}}, ""},
		{"map", &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: String}, ElemType: &AttributeExpr{Type: Int}}}, ""},
		{"nullable root", &AttributeExpr{Type: &Object{}, Nullable: true}, "non-nullable"},
		{"nullable branch", &AttributeExpr{Type: &Union{Untagged: true, Values: []*NamedAttributeExpr{
			{Name: "Nullable", Attribute: &AttributeExpr{Type: &Object{}, Nullable: true}},
		}}}, "Nullable"},
		{"scalar", &AttributeExpr{Type: String}, "object or map"},
		{"array", &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String}}}, "object or map"},
		{"nullable map key", &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: String, Nullable: true}, ElemType: &AttributeExpr{Type: String}}}, "map keys"},
		{"typed extras", &AttributeExpr{Type: &Object{}, AdditionalAttributes: &AttributeExpr{Type: Int}}, "AdditionalAttributes"},
		{"unknown encoding", &AttributeExpr{Type: &Object{}, Meta: MetaExpr{"openapi:encoding:missing:contentType": {"application/json"}}}, "missing"},
		{"invalid style", &AttributeExpr{Type: &Object{{Name: "value", Attribute: &AttributeExpr{Type: String}}}, Meta: MetaExpr{"openapi:encoding:value:style": {""}}}, "nonempty"},
		{"invalid boolean", &AttributeExpr{Type: &Object{{Name: "value", Attribute: &AttributeExpr{Type: String}}}, Meta: MetaExpr{"openapi:encoding:value:explode": {"yes"}}}, "boolean"},
		{"invalid source type", &AttributeExpr{Type: &Object{}, Meta: MetaExpr{"http:encoding:sourceType": {""}}}, "sourceType"},

		{"tagged root", &AttributeExpr{Type: &Union{Values: []*NamedAttributeExpr{
			{Name: "Text", Attribute: &AttributeExpr{Type: String}},
			{Name: "Number", Attribute: &AttributeExpr{Type: Int}},
		}}}, ""},
		{"tagged alternative", &AttributeExpr{Type: &Union{Untagged: true, Values: []*NamedAttributeExpr{
			{Name: "Tagged", Attribute: &AttributeExpr{Type: &Union{Values: []*NamedAttributeExpr{
				{Name: "Text", Attribute: &AttributeExpr{Type: String}},
			}}}},
		}}}, "not tagged unions"},
		{"empty union", &AttributeExpr{Type: &Union{Untagged: true}}, "at least one alternative"},
		{"bad branch", &AttributeExpr{Type: &Union{Untagged: true, Values: []*NamedAttributeExpr{
			{Name: "Record", Attribute: &AttributeExpr{Type: &Object{}}},
			{Name: "Text", Attribute: &AttributeExpr{Type: String}},
		}}}, "Text"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			contract, err := HTTPFormBodyEncoding(tc.body)
			if tc.want != "" {
				require.ErrorContains(t, err, tc.want)
				return
			}
			require.NoError(t, err)
			require.Same(t, tc.body, contract.Body)
			if IsMap(tc.body.Type) {
				require.NotNil(t, contract.Element)
				require.Equal(t, httpencoding.Form, contract.Element.Style)
			}
		})
	}
}

func TestHTTPFormBodyEncodingCycles(t *testing.T) {
	union := &Union{Untagged: true}
	root := &AttributeExpr{Type: union}
	union.Values = []*NamedAttributeExpr{{Name: "Loop", Attribute: root}}
	_, err := HTTPFormBodyEncoding(root)
	require.ErrorContains(t, err, "recursive root")
}

func TestHTTPFormBodyEncodingOpenness(t *testing.T) {
	for _, tc := range []struct {
		name  string
		base  ObjectOpenness
		local ObjectOpenness
		want  ObjectOpenness
	}{
		{name: "implicit"},
		{name: "explicit declaration", base: ObjectOpen, want: ObjectOpen},
		{name: "closed declaration", base: ObjectClosed, want: ObjectClosed},
		{name: "closed occurrence", base: ObjectOpen, local: ObjectClosed, want: ObjectClosed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			declaration := &AttributeExpr{Type: &Object{}}
			if tc.base == ObjectOpen {
				declaration.Meta = MetaExpr{objectOpennessMeta: {"true"}}
			} else if tc.base == ObjectClosed {
				declaration.CloseObject()
			}
			body := &AttributeExpr{Type: &UserTypeExpr{TypeName: "Record", AttributeExpr: declaration}}
			if tc.local == ObjectClosed {
				body.CloseObject()
			}
			contract, err := HTTPFormBodyEncoding(body)
			require.NoError(t, err)
			require.Equal(t, tc.want, contract.ObjectOpenness)
		})
	}
}

func TestHTTPFormBodyEncodingContextSlots(t *testing.T) {
	field := &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String}}}
	declaration := &AttributeExpr{Type: &Object{{Name: "value", Attribute: field}}, Meta: MetaExpr{
		"openapi:encoding:value:style":   {"form"},
		"openapi:encoding:value:explode": {"true"},
	}}
	branch := &AttributeExpr{Type: &UserTypeExpr{TypeName: "Record", AttributeExpr: declaration}}
	body := &AttributeExpr{Type: &Union{Untagged: true, Values: []*NamedAttributeExpr{{Name: "Record", Attribute: branch}}}, Meta: MetaExpr{
		"openapi:encoding:value:explode": {"false"},
		"struct:tag:form":                {"rootTag"},
	}}
	body.CloseObject()
	contract, err := HTTPFormBodyEncoding(body)
	require.NoError(t, err)
	child := contract.Branches[0]
	require.Equal(t, httpencoding.Form, child.Fields["value"].Style)
	require.False(t, child.Fields["value"].Explode)
	require.NotContains(t, child.Metadata, "struct:tag:form")
	require.Equal(t, ObjectOpennessUnset, child.ObjectOpenness)
	require.Equal(t, ObjectClosed, contract.ObjectOpenness)
	standalone, err := HTTPFormBodyEncoding(branch)
	require.NoError(t, err)
	require.True(t, standalone.Fields["value"].Explode)
	// A branch can inherit a root slot for a sibling's property, but cannot
	// author an encoding for a property it does not itself declare.
	branch.Meta = MetaExpr{"openapi:encoding:sibling:style": {"form"}}
	_, err = HTTPFormBodyEncoding(body)
	require.ErrorContains(t, err, "sibling")
}

func TestHTTPFormBodyEncodingKeepsObjectIdentity(t *testing.T) {
	union := &AttributeExpr{Type: &Union{Untagged: true, Values: []*NamedAttributeExpr{
		{Name: "Text", Attribute: &AttributeExpr{Type: String}},
		{Name: "Number", Attribute: &AttributeExpr{Type: Int}},
	}}}
	body := &AttributeExpr{Type: &Object{{Name: "choice", Attribute: union}}, Meta: MetaExpr{
		"openapi:encoding:choice:contentType": {"application/json"},
	}}
	contract, err := HTTPFormBodyEncoding(body)
	require.NoError(t, err)
	require.Empty(t, contract.Branches)
	require.Equal(t, httpencoding.JSON, contract.Fields["choice"].Style)
}

func TestHTTPOptionalSelectedUnionRequired(t *testing.T) {
	union := &AttributeExpr{Type: &Union{Untagged: true}, Meta: MetaExpr{"origin:attribute": {"choice"}}}
	payload := &AttributeExpr{Type: &Object{{Name: "choice", Attribute: union}}, Validation: &ValidationExpr{Required: []string{"choice"}}}
	endpoint := &HTTPEndpointExpr{MethodExpr: &MethodExpr{Payload: payload}, Body: union, OptionalRequestBody: true}
	verr := newValidationErrors()
	endpoint.validateObjectPayloadTransport(verr)
	require.ErrorContains(t, verr, "payload attribute mapped to the request body to be optional")
}

func TestHTTPFormBodyEncodingBranchSourceTypes(t *testing.T) {
	value := &AttributeExpr{Type: &Union{Untagged: true, Values: []*NamedAttributeExpr{
		{Name: "Texts", Attribute: &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String}}}},
		{Name: "Numbers", Attribute: &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: Int}}}},
	}}}
	value.Meta = MetaExpr{"http:encoding:sourceType": {"array"}}
	declaration := &AttributeExpr{Type: &Object{{Name: "value", Attribute: value}}}
	branch := &AttributeExpr{Type: &UserTypeExpr{TypeName: "Arrays", AttributeExpr: declaration}}
	binary := &AttributeExpr{Type: &Object{{Name: "value", Attribute: &AttributeExpr{Type: Bytes}}}}
	body := &AttributeExpr{Type: &Union{Untagged: true, Values: []*NamedAttributeExpr{
		{Name: "Arrays", Attribute: branch},
		{Name: "Binary", Attribute: binary},
	}}}
	standalone, err := HTTPFormBodyEncoding(branch)
	require.NoError(t, err)
	require.Equal(t, httpencoding.Content, standalone.Fields["value"].Style)
	require.Equal(t, httpencoding.Items, standalone.Fields["value"].Application)
	for _, codec := range standalone.Fields["value"].Branches {
		require.Equal(t, httpencoding.Form, codec.Codec.Style)
		require.Equal(t, httpencoding.Items, codec.Codec.Application)
	}
	// Authored root flags select whole-value style grammar. Source type
	// facts stay on the field and never become authored Encoding Object slots.
	body.Meta = MetaExpr{
		"openapi:encoding:value:style":   {"form"},
		"openapi:encoding:value:explode": {"true"},
	}
	contract, err := HTTPFormBodyEncoding(body)
	require.NoError(t, err)
	for _, child := range contract.Branches {
		require.Equal(t, httpencoding.Form, child.Fields["value"].Style)
		require.True(t, child.Fields["value"].Explode)
		require.NotContains(t, child.Metadata, "openapi:encoding:value:contentType")
	}
	require.NotContains(t, contract.Branches[1].Metadata, "http:encoding:sourceType")
	require.Equal(t, []string{"array"}, value.Meta["http:encoding:sourceType"])
	// Even a source type fact on the root occurrence is not inherited into its
	// alternatives; only authored Encoding Object slots are inherited.
	body.Meta["http:encoding:sourceType"] = []string{"absent"}
	contract, err = HTTPFormBodyEncoding(body)
	require.NoError(t, err)
	require.NotContains(t, contract.Branches[0].Metadata, "http:encoding:sourceType")
	require.Equal(t, []string{"array"}, value.Meta["http:encoding:sourceType"])
	require.NotContains(t, contract.Branches[1].Metadata, "http:encoding:sourceType")
}
