package generator

import (
	"testing"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

const bytesCLIHarness = `package bytescli

import (
	"testing"

	"github.com/stretchr/testify/require"
	client "example.com/bytescli/gen/http/service_query_bytes_validate/client"
)

func TestValidatedBytes(t *testing.T) {
	for _, tc := range []struct {
		name string
		value string
		valid bool
	}{
		{"valid", "hello", true},
		{"empty", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := client.BuildMethodQueryBytesValidatePayload(tc.value)
			if tc.valid {
				require.NoError(t, err)
				require.Equal(t, []byte(tc.value), payload.Q)
			} else {
				require.ErrorContains(t, err, "length")
				require.Nil(t, payload)
			}
		})
	}
}
`

func TestCLIValidatedBytesCompile(t *testing.T) {
	runDesignHarness(t, "example.com/bytescli", testdata.PayloadQueryBytesValidateDSL, bytesCLIHarness)
}
