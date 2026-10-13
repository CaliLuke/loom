package generator

import "testing"

func TestUnionNullableWireAdmission(t *testing.T) {
	runDesignHarness(t, "example.com/branchcarrier", unionBranchCarrierDSL, unionNullableWireHarness)
}

const unionNullableWireHarness = `package branchcarrier

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"

	service "example.com/branchcarrier/gen/carrier"
	client "example.com/branchcarrier/gen/http/carrier/client"
	server "example.com/branchcarrier/gen/http/carrier/server"
)

func TestTaggedBranchAdmission(t *testing.T) {
	for _, tc := range []struct {
		name, branch string
		valid bool
	}{
		{"nullable null", "{\"type\":\"nullable\",\"value\":null}", true},
		{"nullable whitespace null", "{\"type\":\"nullable\",\"value\": \n null \t}", true},
		{"nullable value", "{\"type\":\"nullable\",\"value\":{\"type\":\"Leaf\",\"value\":{\"name\":\"ok\"}}}", true},
		{"nullable missing", "{\"type\":\"nullable\"}", false},
		{"nested null", "{\"type\":\"nested\",\"value\":null}", false},
		{"object null", "{\"type\":\"object\",\"value\":null}", false},
		{"text null", "{\"type\":\"text\",\"value\":null}", false},
		{"unknown kind", "{\"type\":\"unknown\",\"value\":null}", false},
		{"missing kind", "{\"value\":null}", false},
		{"malformed", "{\"type\":\"nullable\",\"value\":", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, target := range []struct {
				name string
				value any
			}{
				{"service", &service.Holder{}},
				{"HTTP client", &client.ShowResponseBody{}},
				{"HTTP server", &server.ShowRequestBody{}},
			} {
				t.Run(target.name, func(t *testing.T) {
					wire := []byte("{\"branch\":" + tc.branch + "}")
					err := json.Unmarshal(wire, target.value)
					if !tc.valid {
						require.Error(t, err)
						return
					}
					require.NoError(t, err)
					encoded, err := json.Marshal(target.value)
					require.NoError(t, err)
					require.JSONEq(t, string(wire), string(encoded))
				})
			}
		})
	}
}
`
