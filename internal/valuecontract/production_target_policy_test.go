package valuecontract

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/testprocess"
)

func checkProductionTargetPolicyRejection(t *testing.T, executable string) {
	t.Helper()
	t.Run("decoded map key requires encoding closure", func(t *testing.T) {
		tc := productionTargetNumericCase(t, "0.1", []productionNumericPolicy{productionNumericPolicies()[4]}, true)
		request := tc.command.(map[string]any)["decode"].(map[string]any)["request"].(map[string]any)
		codecs := request["codecs"].(map[string]any)
		require.NotEmpty(t, codecs["decimalReadings"])
		codecs["decimalSpellings"] = []any{}
		actual := referenceDecode[map[string]jsontext.Value](t, runReference(t, executable, []any{tc.command})[0])
		require.Contains(t, actual, "decimalRequests")
		require.NotContains(t, actual, "decoded")
		require.NotContains(t, actual, "schema")
	})
	for _, decimal := range []bool{false, true} {
		name := "integer"
		policy := productionNumericPolicies()[0]
		if decimal {
			name, policy = "decimal", productionNumericPolicies()[4]
		}
		t.Run(name, func(t *testing.T) {
			makeCase := func() (productionTargetCase, map[string]any) {
				tc := productionTargetNumericCase(t, "2147483648", []productionNumericPolicy{policy}, false)
				request := tc.command.(map[string]any)["decode"].(map[string]any)["request"].(map[string]any)
				return tc, request
			}
			field := name + "Readings"
			t.Run("missing", func(t *testing.T) {
				tc, request := makeCase()
				request["codecs"].(map[string]any)[field] = []any{}
				actual := referenceDecode[map[string]jsontext.Value](t, runReference(t, executable, []any{tc.command})[0])
				require.Contains(t, actual, name+"ReadingRequests")
				require.NotContains(t, actual, "decoded")
				require.NotContains(t, actual, "schema")
			})
			t.Run("foreign policy cannot satisfy row", func(t *testing.T) {
				tc, request := makeCase()
				rows := request["codecs"].(map[string]any)[field].([]any)
				if decimal {
					rows[0].(map[string]any)["format"] = "binary64"
				} else {
					rows[0].(map[string]any)["format"] = referenceConstructor("signed", map[string]any{"bits": 64})
				}
				actual := referenceDecode[map[string]jsontext.Value](t, runReference(t, executable, []any{tc.command})[0])
				require.Contains(t, actual, name+"ReadingRequests")
				require.NotContains(t, actual, "decoded")
			})
			t.Run("conflicting duplicate", func(t *testing.T) {
				tc, request := makeCase()
				codecs := request["codecs"].(map[string]any)
				rows := codecs[field].([]any)
				prior := rows[0].(map[string]any)
				conflict := map[string]any{"format": prior["format"], "text": prior["text"], "scalar": nil, "key": nil}
				codecs[field] = append(rows, conflict)
				productionReferenceRejects(t, executable, tc.command, "duplicate "+name+" codec reading")
			})
		})
	}
	t.Run("unsupported integer width", func(t *testing.T) {
		tc := productionTargetNumericCase(t, "1", []productionNumericPolicy{productionNumericPolicies()[0]}, false)
		request := tc.command.(map[string]any)["decode"].(map[string]any)["request"].(map[string]any)
		codecs := request["codecs"].(map[string]any)
		codecs["integerReadings"].([]any)[0].(map[string]any)["format"] = referenceConstructor("signed", map[string]any{"bits": 16})
		productionReferenceRejects(t, executable, tc.command, "unsupported runtime integer width")
	})
	t.Run("key and scalar routing corruption detected", func(t *testing.T) {
		policy := productionNumericPolicies()[0]
		for _, mapping := range []bool{false, true} {
			tc := productionTargetNumericCase(t, " 1", []productionNumericPolicy{policy}, mapping)
			request := tc.command.(map[string]any)["decode"].(map[string]any)["request"].(map[string]any)
			rows := request["codecs"].(map[string]any)["integerReadings"].([]any)
			row := rows[0].(map[string]any)
			row["scalar"], row["key"] = row["key"], row["scalar"]
			actual := referenceDecode[map[string]jsontext.Value](t, runReference(t, executable, []any{tc.command})[0])
			expected := productionCanonical(t, jsontext.Value(productionJSON(t, map[string]any{"ok": tc.decoded})))
			require.NotEqual(t, expected, productionCanonical(t, actual["decoded"]), "the exact-output comparison must detect key/scalar routing corruption")
		}
	})
}

func productionReferenceRejects(t *testing.T, executable string, command any, message string) {
	t.Helper()
	request, err := json.Marshal(referenceRequest{Version: 1, Command: command}, json.Deterministic(true))
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	cmd := testprocess.CommandContext(ctx, executable)
	cmd.Stdin = bytes.NewReader(append(request, '\n'))
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	require.Error(t, cmd.Run(), "invalid policy cannot be a semantic success")
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), message)
}
