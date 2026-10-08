package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestViewedResultConstructorRejectsMissingObject(t *testing.T) {
	root := codegen.RunDSL(t, testdata.WithExplicitAndDefaultViewsDSL)
	services := NewServicesData(root)
	code := generatedFunction(t, renderServiceFile(t, root, services), "NewViewedMultipleViews")
	require.Contains(t, code, "if res == nil {")
	require.Contains(t, code, `loom.Fault("missing result")`)
	require.Less(t, strings.Index(code, "if res == nil"), strings.Index(code, "switch view"))
}

func TestViewedResultConstructorAcceptsNilCollection(t *testing.T) {
	root := codegen.RunDSL(t, testdata.ResultCollectionMultipleViewsMethodDSL)
	services := NewServicesData(root)
	code := generatedFunction(t, renderServiceFile(t, root, services), "NewViewedMultipleViewsCollection")
	require.NotContains(t, code, "if res == nil")
}

func TestViewedResultConstructorValidatesOutput(t *testing.T) {
	root := codegen.RunDSL(t, testdata.WithExplicitAndDefaultViewsDSL)
	services := NewServicesData(root)
	code := generatedFunction(t, renderServiceFile(t, root, services), "NewViewedMultipleViews")
	require.Contains(t, code, ".ValidateMultipleViews(vres)")
	require.Contains(t, code, `loom.NewServiceError(err, "fault", false, false, true)`)

	client := generatedFunction(t, renderServiceFile(t, root, services), "NewMultipleViews")
	require.NotContains(t, client, `"fault"`, "client conversion must retain client error semantics")
}

func TestResultProjectionPreservesNilObject(t *testing.T) {
	root := codegen.RunDSL(t, testdata.WithExplicitAndDefaultViewsDSL)
	services := NewServicesData(root)
	code := generatedFunction(t, renderServiceFile(t, root, services), "ProjectMultipleViews")
	require.Contains(t, code, "if res == nil {\n\t\treturn nil\n\t}")
}
