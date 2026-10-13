package openapiref

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestName(t *testing.T) {
	for _, tc := range []struct {
		name, ref, want, errorText string
	}{
		{name: "plain", ref: "#/components/schemas/ItemName", want: "ItemName"},
		{name: "encoded letter", ref: "#/components/schemas/%49temName", want: "ItemName"},
		{name: "encoded underscore", ref: "#/components/schemas/Item%5fName", want: "Item_Name"},
		{name: "encoded path", ref: "#%2fcomponents%2fschemas%2fItemName", want: "ItemName"},
		{name: "URI decoded once", ref: "#/components/schemas/%2549temName", want: "%49temName"},
		{name: "pointer decoded once", ref: "#/components/schemas/Item%7E01Name", want: "Item~1Name"},
		{name: "pointer slash", ref: "#/components/schemas/Item%7E1Name", want: "Item/Name"},
		{name: "plus preserved", ref: "#/components/schemas/Item+Name", want: "Item+Name"},
		{name: "empty", ref: "", errorText: "wrong kind"},
		{name: "empty token", ref: "#/components/schemas/", errorText: "wrong kind"},
		{name: "wrong category", ref: "#/components/parameters/ItemName", errorText: "wrong kind"},
		{name: "document", ref: "other.yaml#/components/schemas/ItemName", errorText: "wrong kind"},
		{name: "external", ref: "https://example.com/spec#/components/schemas/ItemName", errorText: "wrong kind"},
		{name: "extra token", ref: "#/components/schemas/ItemName/properties/value", errorText: "wrong kind"},
		{name: "encoded extra token", ref: "#/components/schemas/ItemName%2fvalue", errorText: "wrong kind"},
		{name: "extra fragment", ref: "#/components/schemas/Item#Name", errorText: "wrong kind"},
		{name: "bad URI escape", ref: "#/components/schemas/Item%GG", errorText: "URI fragment escaping"},
		{name: "truncated URI escape", ref: "#/components/schemas/Item%", errorText: "URI fragment escaping"},
		{name: "bad pointer escape", ref: "#/components/schemas/Item~2Name", errorText: "JSON Pointer escaping"},
		{name: "encoded bad pointer escape", ref: "#/components/schemas/Item%7e2Name", errorText: "JSON Pointer escaping"},
		{name: "invalid UTF8", ref: "#/components/schemas/Item%ff", errorText: "JSON Pointer escaping"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name, err := Name(tc.ref, "#/components/schemas/")
			if tc.errorText != "" {
				require.ErrorContains(t, err, tc.errorText)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, name)
		})
	}
}
