package quality_test

import (
	"encoding/json/jsontext"
	"testing"

	client "example.com/http-quality/gen/http/accounts/client"
	"github.com/stretchr/testify/require"
)

func TestCLIJSONDecoderCause(t *testing.T) {
	for _, input := range []string{"{", `{"email":]}`} {
		t.Run(input, func(t *testing.T) {
			payload, err := client.BuildCreatePayload(input, "", "")
			require.Nil(t, payload)
			require.ErrorContains(t, err, "invalid JSON for body")
			var syntaxError *jsontext.SyntacticError
			require.ErrorAs(t, err, &syntaxError)
		})
	}
	payload, err := client.BuildCreatePayload(`{"email":"ada@example.com","display_name":"Ada"}`, "", "")
	require.NoError(t, err)
	require.Equal(t, "Ada", payload.DisplayName)
}
