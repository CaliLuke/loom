package openapiv3

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestReusableObjectReferenceAnnotations(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value any
		fresh func() any
	}{
		{"body", &RequestBodyRef{Ref: "#/components/requestBodies/Input", Description: new(""), Summary: new("hint")}, func() any {
			return &RequestBodyRef{}
		}},
		{"header", &HeaderRef{Ref: "#/components/headers/Trace", Description: new("local"), Summary: new("")}, func() any {
			return &HeaderRef{}
		}},
		{"response", &ResponseRef{Ref: "#/components/responses/Result", Description: new(""), Summary: new("")}, func() any {
			return &ResponseRef{}
		}},
		{"inherited-body", &RequestBodyRef{Ref: "#/components/requestBodies/Input"}, func() any {
			return &RequestBodyRef{}
		}},
		{"inherited-header", &HeaderRef{Ref: "#/components/headers/Trace"}, func() any {
			return &HeaderRef{}
		}},
		{"inline-body", &RequestBodyRef{Value: &RequestBody{Description: "body"}}, func() any {
			return &RequestBodyRef{Ref: "stale", Description: new("stale"), Summary: new("stale")}
		}},
		{"inline-header", &HeaderRef{Value: &Header{Description: "header"}}, func() any {
			return &HeaderRef{Ref: "stale", Description: new("stale"), Summary: new("stale")}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, format := range []string{"json", "yaml"} {
				t.Run(format, func(t *testing.T) {
					got := tc.fresh()
					var wire []byte
					var err error
					if format == "json" {
						wire, err = json.Marshal(tc.value, json.Deterministic(true))
						require.NoError(t, err)
						require.NoError(t, json.Unmarshal(wire, got))
					} else {
						wire, err = yaml.Marshal(tc.value)
						require.NoError(t, err)
						require.NoError(t, yaml.Unmarshal(wire, got))
					}
					require.Equal(t, tc.value, got)
				})
			}
		})
	}
}
