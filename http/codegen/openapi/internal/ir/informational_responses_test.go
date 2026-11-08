package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
)

func TestInformationalResponsesAreDocumented(t *testing.T) {
	root := codegen.RunDSL(t, informationalResponsesDSL)
	doc, err := BuildDocument(root.API, root.Types, root.ResultTypes)
	require.NoError(t, err)
	operation := doc.Paths["/widgets"].Operations["GET"]
	require.ElementsMatch(t, []string{"100", "103", "1XX", "200"}, mapKeys(operation.Responses))

	continued := resolveSpecimenResponse(doc, operation, "100")
	require.Equal(t, "Automatic continuation.", continued.Description)
	require.Empty(t, continued.Content)
	require.Contains(t, continued.Headers, "Link")

	hints := resolveSpecimenResponse(doc, operation, "103")
	require.Equal(t, "Early Hints response.", hints.Description)
	require.Empty(t, hints.Content, "informational responses carry no content")
	require.Contains(t, hints.Headers, "Link")

	progress := resolveSpecimenResponse(doc, operation, "1XX")
	require.Equal(t, "Interim progress.", progress.Description)
	require.Empty(t, progress.Content)
	require.Contains(t, progress.Headers, "X-Progress")
	require.NotContains(t, progress.Headers, "code", "the StatusField is the status line, not a header")
	require.Len(t, progress.Headers, 1)

	final := resolveSpecimenResponse(doc, operation, "200")
	require.NotNil(t, final.Content["application/json"])
}

func informationalResponsesDSL() {
	hints := dsl.Type("Hints", func() {
		dsl.Attribute("link", dsl.String)
		dsl.Required("link")
	})
	progress := dsl.Type("Progress", func() {
		dsl.Attribute("code", dsl.Int)
		dsl.Attribute("percent", dsl.Int)
		dsl.Required("code")
	})
	dsl.Service("widgets", func() {
		dsl.Method("list", func() {
			dsl.Result(dsl.String)
			dsl.HTTP(func() {
				dsl.GET("/widgets")
				dsl.Informational("continue", dsl.StatusContinue, hints, func() {
					dsl.Description("Automatic continuation.")
					dsl.Header("link:Link")
				})
				dsl.Informational("hints", dsl.StatusEarlyHints, hints, func() {
					dsl.Header("link:Link")
				})
				dsl.Informational("progress", dsl.Status1XX, progress, func() {
					dsl.Description("Interim progress.")
					dsl.StatusField("code")
					dsl.Header("percent:X-Progress")
				})
				dsl.Response(dsl.StatusOK)
			})
		})
	})
}

func mapKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
