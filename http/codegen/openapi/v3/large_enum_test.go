package openapiv3_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestRenderedLargeNestedEnum(t *testing.T) {
	artifacts := renderOpenAPIArtifacts(t, testdata.LargeEnumResponseDSL)
	spec := decodeOpenAPIJSON(t, artifacts.JSON)
	values := make([]any, 433)
	for index := range values {
		values[index] = fmt.Sprintf("category-%04d", index)
	}
	var enums int
	var visit func(any)
	visit = func(value any) {
		switch value := value.(type) {
		case map[string]any:
			if enumeration, ok := value["enum"]; ok {
				require.Equal(t, values, enumeration)
				enums++
			}
			for _, child := range value {
				visit(child)
			}
		case []any:
			for _, child := range value {
				visit(child)
			}
		}
	}
	visit(spec)
	require.Positive(t, enums, "the reachable component graph must retain the enum")
	for index := range 6 {
		operation := requireOperation(t, spec, fmt.Sprintf("/catalog/%d", index), "get")
		require.NotEmpty(t, requireResponseMediaType(t, operation, "application/json")["schema"])
	}
}
