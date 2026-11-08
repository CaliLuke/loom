package openapiv3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/CaliLuke/loom/internal/openapiimport"
)

func TestInformationalContinueArtifacts(t *testing.T) {
	artifacts := renderOpenAPIArtifacts(t, testdata.InformationalServerDSL)
	for _, raw := range [][]byte{artifacts.JSON, artifacts.YAML} {
		document, diagnostics, err := openapiimport.Analyze(raw)
		require.NoError(t, err)
		require.Empty(t, diagnostics)
		rendered, err := openapiimport.Render(document, openapiimport.Options{})
		require.NoError(t, err)
		require.Contains(t, string(rendered), `Informational("SelectionReadStatus100", 100`)
		require.Contains(t, string(rendered), `Informational("SelectionBodyStatus100", 100`)
	}
	contract := decodeOpenAPIJSON(t, artifacts.JSON)
	operation := requireOperation(t, contract, "/read", "get")
	responses := requireMap(t, operation["responses"], "response keys")
	require.Contains(t, responses, "100")
	require.Contains(t, responses, "1XX")
}
