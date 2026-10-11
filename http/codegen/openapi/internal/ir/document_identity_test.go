package ir

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateOperationIdentities(t *testing.T) {
	tests := []struct {
		name   string
		claims []OperationIdentity
		want   string
	}{
		{
			name: "same path supports different methods",
			claims: []OperationIdentity{
				{Path: "/widgets/{id}", Method: "GET", OperationID: "widgets.get"},
				{Path: "/widgets/{id}", Method: "POST", OperationID: "widgets.create"},
			},
		},
		{
			name: "equivalent templates conflict across methods",
			claims: []OperationIdentity{
				{Path: "/widgets/{id}", Method: "GET", OperationID: "widgets.get"},
				{Path: "/widgets/{slug}", Method: "POST", OperationID: "widgets.create"},
			},
			want: "path templates",
		},
		{
			name: "duplicate route slot conflicts",
			claims: []OperationIdentity{
				{Path: "/widgets", Method: "GET", OperationID: "widgets.get", Source: "widgets.get"},
				{Path: "/widgets", Method: "GET", OperationID: "widgets.list", Source: "widgets.list"},
			},
			want: "GET /widgets",
		},
		{
			name: "operation IDs are document-wide",
			claims: []OperationIdentity{
				{Path: "/widgets", Method: "GET", OperationID: "same"},
				{Path: "/gadgets", Method: "GET", OperationID: "same"},
			},
			want: `operationId "same"`,
		},
		{
			name: "escaped brace literal is distinct",
			claims: []OperationIdentity{
				{Path: "/widgets/{id}", Method: "GET", OperationID: "widgets.get"},
				{Path: "/widgets/%7Bid%7D", Method: "POST", OperationID: "widgets.literal"},
			},
		},
		{
			name: "different static suffixes are distinct",
			claims: []OperationIdentity{
				{Path: "/items/{id}.json", Method: "GET", OperationID: "items.json"},
				{Path: "/items/{slug}.xml", Method: "POST", OperationID: "items.xml"},
			},
		},
		{
			name: "parameter names inside a static suffix do not change identity",
			claims: []OperationIdentity{
				{Path: "/items/{id}.json", Method: "GET", OperationID: "items.get"},
				{Path: "/items/{slug}.json", Method: "POST", OperationID: "items.post"},
			},
			want: "path templates",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateOperationIdentities(test.claims)
			if test.want == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestValidateComponentIdentities(t *testing.T) {
	require.NoError(t, ValidateComponentIdentities([]ComponentIdentity{
		{Kind: "mediaTypes", Name: "Shared", Value: `{"schema":{"type":"string"}}`},
		{Kind: "mediaTypes", Name: "Shared", Value: `{"schema":{"type":"string"}}`},
	}))

	forward := []ComponentIdentity{
		{Kind: "mediaTypes", Name: "Shared", Value: `{"schema":{"type":"string"}}`, Source: "first"},
		{Kind: "mediaTypes", Name: "Shared", Value: `{"schema":{"type":"integer"}}`, Source: "second"},
	}
	backward := []ComponentIdentity{forward[1], forward[0]}
	require.EqualError(t, ValidateComponentIdentities(forward), ValidateComponentIdentities(backward).Error())
	require.ErrorContains(t, ValidateComponentIdentities(forward), `mediaTypes component "Shared"`)
}
