// Package schematest runs independent JSON Schema contract checks in integration
// tests. Its pinned JavaScript dependencies are installed only in a test temp dir.
package schematest

import (
	"bytes"
	"context"
	"embed"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/testprocess"
)

type (
	// Validator owns a temporary Ajv installation with native ECMA-262 regexes.
	Validator struct {
		dir string
	}

	// Batch pairs an emitted schema with the JSON instances to validate against it.
	Batch struct {
		// Schema is the complete schema, including any referenced definitions.
		Schema any `json:"schema"`
		// Instances are JSON values; validation never coerces or mutates them.
		Instances []any `json:"instances"`
	}

	// Result reports acceptance and the validator's original failure diagnostics.
	Result struct {
		// Valid indicates that the schema accepted the complete instance.
		Valid bool `json:"valid"`
		// Errors contains independent validator diagnostics, empty on success.
		Errors jsontext.Value `json:"errors"`
	}
)

//go:embed package.json package-lock.json validate.cjs
var files embed.FS

// New installs the locked test-only validator. Missing tools and installation
// failures fail the caller; an enabled contract gate can never silently skip.
func New(t *testing.T) *Validator {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"package.json", "package-lock.json", "validate.cjs"} {
		data, err := files.ReadFile(name)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), data, 0o600))
	}
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	cmd := testprocess.CommandContext(ctx, "npm", "ci", "--ignore-scripts", "--no-audit", "--no-fund")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "install independent schema validator: %s", out)
	return &Validator{dir: dir}
}

// Check validates each batch in order and preserves one result per instance.
// Invalid schemas and protocol/tool failures fail the caller independently of
// ordinary instance rejection.
func (v *Validator) Check(t *testing.T, batches []Batch) [][]Result {
	t.Helper()
	data, err := json.Marshal(batches)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	cmd := testprocess.CommandContext(ctx, "node", filepath.Join(v.dir, "validate.cjs"))
	cmd.Dir = v.dir
	cmd.Stdin = bytes.NewReader(data)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "independent schema validation: %s", out)
	var results [][]Result
	require.NoError(t, json.Unmarshal(out, &results, json.RejectUnknownMembers(true)))
	require.Len(t, results, len(batches))
	for i, batch := range batches {
		require.Len(t, results[i], len(batch.Instances))
	}
	return results
}
