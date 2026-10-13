package codegen

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/testingx"
)

func TestTransportImportAliasReservationsCoverGeneratedImports(t *testing.T) {
	for _, c := range localNameDesigns {
		t.Run(c.Name, func(t *testing.T) {
			const modulePath = "example.com/importnames"
			dir := renderLocalNameModule(t, modulePath, c.Design, localNameMarker+"svc")
			// These controlled designs contain no authored external custom types.
			// Their external imports therefore describe framework alias ownership.
			names := testingx.ServicePackageImportNames(t, dir, modulePath+"/gen")
			require.NotEmpty(t, names)
			require.Empty(t, missingImportReservations(names, transportGeneratedImportNames))
		})
	}
}

func TestMissingImportReservations(t *testing.T) {
	for _, tc := range []struct {
		name     string
		observed []string
		want     []string
	}{
		{"all reserved", []string{"fmt"}, nil},
		{"unreserved co-located alias", []string{"fmt", "loomdiagnostics"}, []string{"loomdiagnostics"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, missingImportReservations(tc.observed, []string{"fmt"}))
		})
	}
}

func missingImportReservations(observed, reserved []string) []string {
	var missing []string
	for _, name := range observed {
		if !slices.Contains(reserved, name) {
			missing = append(missing, name)
		}
	}
	return missing
}
