package selvedge_test

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"os"
	"path/filepath"
	"testing"

	sv "github.com/CaliLuke/selvedge"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/execution"
	adapter "github.com/CaliLuke/loom/execution/selvedge"
)

func workflowPostgresDatabase(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("LOOM_WORKFLOW_POSTGRES_DSN")
	if dsn == "" {
		if os.Getenv("LOOM_WORKFLOW_REQUIRE_POSTGRES") == "1" {
			t.Fatal("LOOM_WORKFLOW_POSTGRES_DSN is required")
		}
		t.Skip("set LOOM_WORKFLOW_POSTGRES_DSN for the explicit PostgreSQL acceptance suite")
	}
	return dsn
}

func workflowStoredIdentity(t *testing.T, database string) {
	t.Helper()
	ctx := context.Background()
	p, err := adapter.NewProducer(sv.WithDatabase(database))
	require.NoError(t, err)
	w, err := adapter.NewWorker(sv.WithDatabase(database))
	require.NoError(t, err)
	wf := execution.Workflow[jsontext.Value, string]{Name: t.Name() + ".bytes"}
	nils := execution.Workflow[[]string, string]{Name: t.Name() + ".nil"}
	for _, role := range []adapter.Role{p, w} {
		require.NoError(t, adapter.Declare(role, wf))
		require.NoError(t, adapter.Declare(role, nils))
	}
	require.NoError(t, p.Start(ctx))
	t.Cleanup(func() {
		require.NoError(t, p.Shutdown(ctx))
	})
	require.NoError(t, w.Start(ctx))
	t.Cleanup(func() {
		require.NoError(t, w.Shutdown(ctx))
	})
	observer, err := sv.OpenProducer(ctx, sv.WithDatabase(database))
	require.NoError(t, err)
	t.Cleanup(func() {
		finishNativeCleanup(t, observer.Shutdown)
	})
	input := jsontext.Value(`{ "z": "<>&", "a": 1.00 }`)
	expected := []byte(`{"z":"\u003c\u003e\u0026","a":1.00}`)
	for index, role := range []adapter.Role{p, w} {
		id := t.Name() + []string{".producer", ".worker"}[index]
		first, err := adapter.Submit(ctx, role, wf, id, input)
		require.NoError(t, err)
		require.True(t, first.Created)
		desc, err := observer.Describe(ctx, id)
		require.NoError(t, err)
		require.True(t, bytes.Equal(expected, desc.InputData), "stored %q", desc.InputData)
		again, err := adapter.Submit(ctx, role, wf, id, input)
		require.NoError(t, err)
		require.False(t, again.Created)
		_, err = adapter.Submit(ctx, role, wf, id, jsontext.Value(`{"a":2}`))
		var conflict *adapter.AdmissionConflictError
		require.ErrorAs(t, err, &conflict)
		require.Equal(t, expected, conflict.ExistingInput)
		_, err = adapter.Submit(ctx, role, nils, id+".nil", nil)
		require.NoError(t, err)
		desc, err = observer.Describe(ctx, id+".nil")
		require.NoError(t, err)
		require.Equal(t, "null", string(desc.InputData))
	}
}

func TestWorkflowSQLiteAdmission(t *testing.T) {
	workflowStoredIdentity(t, filepath.Join(t.TempDir(), "identity.db"))
}

func TestWorkflowPostgresAdmission(t *testing.T) {
	workflowStoredIdentity(t, workflowPostgresDatabase(t))
}

func TestWorkflowPostgresTwoProcess(t *testing.T) {
	workflowTwoProcess(t, workflowPostgresDatabase(t))
}
