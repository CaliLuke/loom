package http

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
)

func TestProblemResponsePreservesErrorProperties(t *testing.T) {
	for bits := range 8 {
		for _, status := range []int{409, 500, 503, 504, -32002} {
			t.Run(fmt.Sprintf("properties%d/status%d", bits, status), func(t *testing.T) {
				source := loom.NewServiceError(errors.New("safe detail"), "designed", bits&1 != 0, bits&2 != 0, bits&4 != 0)
				source.ID = "instance-id"
				source = loom.WithErrorRemedy(source, &loom.ErrorRemedy{SafeMessage: "public detail", RetryHint: "retry later"})
				body := NewProblemResponse(context.Background(), source, status, "", "")
				encoded, err := json.Marshal(body)
				require.NoError(t, err)
				var document map[string]jsontext.Value
				require.NoError(t, json.Unmarshal(encoded, &document))
				for name, expected := range map[string]bool{"timeout": source.Timeout, "temporary": source.Temporary, "fault": source.Fault} {
					t.Run(name, func(t *testing.T) {
						value, exists := document[name]
						require.True(t, exists, "full problem response omits %s", name)
						var actual bool
						require.NoError(t, json.Unmarshal(value, &actual))
						require.Equal(t, expected, actual)
					})
				}
				var decoded ProblemResponse
				require.NoError(t, json.Unmarshal(encoded, &decoded))
				result := ProblemErrorFromBody(decoded)
				require.Equal(t, source.Name, result.Name)
				require.Equal(t, source.ID, result.ID)
				require.Equal(t, "public detail", result.Message)
				require.Equal(t, "retry later", loom.ErrorRetryHint(result))
				require.Equal(t, source.Timeout, result.Timeout)
				require.Equal(t, source.Temporary, result.Temporary)
				require.Equal(t, source.Fault, result.Fault)
			})
		}
	}
}

func TestProblemBodyOmissionDoesNotInferProperties(t *testing.T) {
	for _, status := range []int{408, 429, 500, 503, 504, -32002} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			result := ProblemErrorFromBody(ProblemResponse{Status: status, Detail: "selected detail"})
			require.False(t, result.Timeout)
			require.False(t, result.Temporary)
			require.False(t, result.Fault)
		})
	}
}
