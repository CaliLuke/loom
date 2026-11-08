package openapiimport

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSourceDisplayProjection(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"https://review-user:review-secret@auth.test/path", "https://[redacted]@auth.test/path"},
		{"https://review-user:review-secret@auth.test/%ZZ", "https://[redacted]@auth.test/%ZZ"},
		{"#/security/0/https:~1~1review-user:review-secret@auth.test~1path", "#/security/0/https:~1~1[redacted]@auth.test~1path"},
		{`"https:\/\/review-user:review-secret@auth.test\/path"`, `"https://[redacted]@auth.test/path"`},
		{"//review-user:review-secret@host/path", "//[redacted]@host/path"},
		{"https://review-user@host/path", "https://[redacted]@host/path"},
		{"https://host/path@literal", "https://host/path@literal"},
		{"https://host/path//literal@name", "https://host/path//literal@name"},
		{"#/security/0/https:~1~1host~1path~1~1literal@name", "#/security/0/https:~1~1host~1path~1~1literal@name"},
		{"#/components/schemas/a~1b", "#/components/schemas/a~1b"},
		{"invalid URI escape", "invalid URI escape"},
	} {
		require.Equal(t, tc.want, sourceDisplayText(tc.input))
	}
	original := Diagnostics{{Code: "external-reference", Path: "#/security/0/https:~1~1review-user:review-secret@host", Origin: &SourceLocation{Document: "entry", Pointer: "#/security/0/https:~1~1review-user:review-secret@host"}, Message: "https://review-user:review-secret@host"}}
	projected := publicDiagnostics(original, nil)
	requirePrivateOutput(t, projected)
	require.Contains(t, original[0].Path, "review-secret")
	require.Contains(t, original[0].Origin.Pointer, "review-secret")
	sentinel := errors.New("source rejected")
	raw := fmt.Errorf("https://review-user:review-secret@host: %w", sentinel)
	safe := RedactSourceError(raw)
	require.ErrorIs(t, safe, sentinel)
	require.Nil(t, errors.Unwrap(safe))
	requirePrivateOutput(t, safe.Error())
	require.Same(t, sentinel, RedactSourceError(sentinel))
}

func TestPublicDisplayBoundaries(t *testing.T) {
	const pointer = "#/paths/~1https:~1~1host.example/get/parameters/0/schema/properties/contact@email/madeUp"
	const prose = "reference https://host.example is unavailable for contact user@example.test"
	diagnostics := publicDiagnostics(Diagnostics{{Path: pointer, Message: prose, Origin: &SourceLocation{Document: "entry", Pointer: pointer, Authored: "https://host.example/path@literal"}}}, nil)
	require.Equal(t, pointer, diagnostics[0].Path)
	require.Equal(t, pointer, diagnostics[0].Origin.Pointer)
	require.Equal(t, prose, diagnostics[0].Message)
	require.Equal(t, "https://host.example/path@literal", diagnostics[0].Origin.Authored)
	require.Equal(t, prose, sourceDisplayText(prose))
	require.Equal(t, `reference "https://host.example" contact user@example.test`, sourceDisplayText(`reference "https://host.example" contact user@example.test`))
}

func TestPublicCredentialFieldContexts(t *testing.T) {
	for _, uri := range []string{"https://review-user:review-secret@auth.test/path", "https://review-user:review-secret@auth.test/%ZZ", "https://review-user:review-secret%ZZ@auth.test/path", "https://review-user:review-secret with-space@auth.test/path"} {
		t.Run(uri, func(t *testing.T) {
			pointer := "#/security/0/" + escapeJSONPointer(uri) + "/contact@email"
			diagnostics := Diagnostics{{Path: pointer, Origin: &SourceLocation{Document: "entry", Pointer: pointer, Authored: uri}, Message: fmt.Sprintf("rejected %q; contact user@example.test", sourceDisplayURI(uri))}}
			projected := publicDiagnostics(diagnostics, nil)
			requirePrivateOutput(t, projected)
			requirePrivateOutput(t, diagnostics.Error())
			require.Contains(t, projected[0].Path, "/contact@email")
			require.Contains(t, projected[0].Message, "contact user@example.test")
			require.Contains(t, diagnostics[0].Path, "review-user")
		})
	}
	const prose = `"https://host.example" is unavailable; contact user@example.test`
	require.Equal(t, prose, sourceDisplayText(prose))
	const quotedProse = `"reference https://host.example is unavailable; contact user@example.test"`
	require.Equal(t, quotedProse, sourceDisplayText(quotedProse))
}

func TestQuotedSourceProseBoundary(t *testing.T) {
	const prose = `"https://host.example is unavailable; contact user@example.test"`
	require.Equal(t, prose, sourceDisplayText(prose))
	require.Equal(t, prose, RedactSourceError(errors.New(prose)).Error())
	const key = "https://host.example is unavailable; contact user@example.test"
	pointer := "#/components/schemas/Item/properties/" + escapeJSONPointer(key)
	require.Equal(t, pointer, sourceDisplayPointer(pointer))
}
