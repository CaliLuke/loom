package openapiv3

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestResponseRefDescriptionRoundTrip(t *testing.T) {
	description, empty := "first: First failure.\n\nsecond: Second failure.", ""
	for _, tc := range []struct {
		name  string
		value *ResponseRef
	}{
		{"override", &ResponseRef{Ref: "#/components/responses/Error", Description: &description}},
		{"empty-override", &ResponseRef{Ref: "#/components/responses/Error", Description: &empty}},
		{"inherited", &ResponseRef{Ref: "#/components/responses/Error"}},
		{"inline", &ResponseRef{Value: &Response{Description: &description}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, format := range []string{"json", "yaml"} {
				t.Run(format, func(t *testing.T) {
					var raw []byte
					var err error
					got := ResponseRef{Ref: "stale", Description: &empty, Value: &Response{}}
					if format == "json" {
						raw, err = json.Marshal(tc.value)
						require.NoError(t, err)
						err = json.Unmarshal(raw, &got)
					} else {
						raw, err = yaml.Marshal(tc.value)
						require.NoError(t, err)
						err = yaml.Unmarshal(raw, &got)
					}
					require.NoError(t, err)
					require.Equal(t, tc.value, &got)
				})
			}
		})
	}
}
