package valuecontract

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/testprocess"
)

type (
	// referenceRequest is the versioned transport envelope, not a semantic model.
	referenceRequest struct {
		Version int `json:"version"`
		Command any `json:"command"`
	}
	// referenceResponse retains exact JSON numbers until a typed assertion reads them.
	referenceResponse struct {
		Version int            `json:"version"`
		Result  jsontext.Value `json:"result"`
	}
	// referenceIdentity maps an opaque production handle into one corpus-local space.
	referenceIdentity struct {
		Occurrence  uint64 `json:"occurrence"`
		Declaration uint64 `json:"declaration"`
	}
	// referenceDecimal is the exact semantic decimal used by the Lean model.
	referenceDecimal struct {
		Coefficient jsontext.Value `json:"coefficient"`
		Exponent    int64          `json:"exponent"`
	}
)

func TestReferenceAdapterPreservesEntriesAndPresence(t *testing.T) {
	input := referenceConstructor("map", map[string]any{
		"entries": []any{
			[]any{map[string]any{"value": referenceConstructor("integer", map[string]any{"value": int64(9007199254740993)}), "primitive": "builtinInt64"}, "absent"},
			[]any{map[string]any{"value": referenceConstructor("string", map[string]any{"value": "9007199254740993"}), "primitive": "builtinString"}, "null"},
		},
	})
	encoded, err := json.Marshal(input, json.Deterministic(true))
	require.NoError(t, err)
	require.Equal(t, `{"map":{"entries":[[{"primitive":"builtinInt64","value":{"integer":{"value":9007199254740993}}},"absent"],[{"primitive":"builtinString","value":{"string":{"value":"9007199254740993"}}},"null"]]}}`, string(encoded))
	identity, err := json.Marshal(referenceIdentity{Occurrence: 11, Declaration: 23})
	require.NoError(t, err)
	require.Equal(t, `{"occurrence":11,"declaration":23}`, string(identity))
}

func TestReferenceAdapterPreservesExactBinaryCoefficient(t *testing.T) {
	value := referenceBinaryDecimal(float64(float32(0.1)))
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	require.JSONEq(t, `{"coefficient":100000001490116119384765625,"exponent":-27}`, string(encoded))
	decoded := referenceDecode[referenceDecimal](t, encoded)
	require.Equal(t, value, decoded)
}

// referenceBinaryDecimal translates a finite IEEE value into an exact decimal.
// The coefficient is a JSON integer of arbitrary size, never a float64 or int64.
func referenceBinaryDecimal(value float64) referenceDecimal {
	rational := new(big.Rat).SetFloat64(value)
	if rational == nil {
		panic("referenceBinaryDecimal requires a finite number")
	}
	power := int64(rational.Denom().BitLen() - 1)
	coefficient := new(big.Int).Exp(big.NewInt(5), big.NewInt(power), nil)
	coefficient.Mul(coefficient, rational.Num())
	return referenceDecimal{Coefficient: jsontext.Value(coefficient.String()), Exponent: -power}
}

func TestReferenceAdapterKeepsNullAndMissingDistinct(t *testing.T) {
	for _, presence := range []string{"absent", "null", "nilBytes", "nilArray", "nilMap"} {
		t.Run(presence, func(t *testing.T) {
			encoded, err := json.Marshal(presence)
			require.NoError(t, err)
			require.Equal(t, `"`+presence+`"`, string(encoded))
		})
	}
}

func referenceConstructor(name string, fields map[string]any) map[string]any {
	return map[string]any{name: fields}
}

func referenceExecutable(t *testing.T) string {
	t.Helper()
	path := os.Getenv("LOOM_LEAN_REFERENCE")
	require.True(t, filepath.IsAbs(path), "LOOM_LEAN_REFERENCE must name an absolute executable path")
	info, err := os.Stat(path)
	require.NoError(t, err, "enabled conformance requires the built Lean reference")
	require.False(t, info.IsDir())
	require.NotZero(t, info.Mode()&0111, "reference must be executable")
	return path
}

func runReference(t *testing.T, executable string, commands []any) []jsontext.Value {
	t.Helper()
	var input bytes.Buffer
	for _, command := range commands {
		request, err := json.Marshal(referenceRequest{Version: 1, Command: command}, json.Deterministic(true))
		require.NoError(t, err)
		input.Write(request)
		input.WriteByte('\n')
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	cmd := testprocess.CommandContext(ctx, executable)
	cmd.Stdin = &input
	var output, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &stderr
	require.NoError(t, cmd.Run(), "reference execution failed: %s", stderr.String())
	require.Empty(t, stderr.String(), "reference diagnostics cannot be silently discarded")
	scanner := bufio.NewScanner(&output)
	scanner.Buffer(make([]byte, 4096), 16<<20)
	var results []jsontext.Value
	for scanner.Scan() {
		var response referenceResponse
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &response, json.RejectUnknownMembers(true)))
		require.Equal(t, 1, response.Version)
		require.NotEmpty(t, response.Result)
		results = append(results, response.Result.Clone())
	}
	require.NoError(t, scanner.Err())
	require.Len(t, results, len(commands), "each command needs exactly one response")
	return results
}

func referenceDecode[T any](t *testing.T, value jsontext.Value) T {
	t.Helper()
	var result T
	require.NoError(t, json.Unmarshal(value, &result), fmt.Sprintf("reference value: %s", value))
	return result
}
