package testingx

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModuleBatchContracts(t *testing.T) {
	for _, export := range []bool{false, true} {
		t.Run(fmt.Sprintf("export=%t", export), func(t *testing.T) {
			output := ""
			if export {
				output = t.TempDir()
			}
			batch := NewModuleBatch(t, RepoRoot(), output)
			var cases []*ModuleCase
			for _, value := range []int{1, 2} {
				t.Run(fmt.Sprint(value), func(t *testing.T) {
					c := batch.Case(t, fmt.Sprintf("case-%d", value), "example.com/original")
					cases = append(cases, c)
					c.WriteFile(t, "value/value.go", fmt.Sprintf("package value\nfunc Value() int {\n return %d\n}\n", value))
					c.WriteFile(t, "contract_test.go", fmt.Sprintf(`package contract_test
import (
 "os"
 "testing"
 "example.com/original/value"
)
func TestContract(t *testing.T) {
 if value.Value() != %d {
  t.Fatal("wrong case namespace")
 }
 f, err := os.OpenFile("ran", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
 if err != nil {
  t.Fatal(err)
 }
 if err := f.Close(); err != nil {
  t.Fatal(err)
 }
}
`, value))
					c.WriteFile(t, "no_race_test.go", `//go:build !race

package contract_test
import "testing"
func TestRequiresRace(t *testing.T) {
 t.Fatal("missing race instrumentation")
}
`)
				})
			}
			// Child subtest cleanup must not remove cases before the shared commands.
			batch.Check(t)
			for _, c := range cases {
				require.FileExists(t, filepath.Join(c.Dir, "ran"))
				_, err := os.Stat(filepath.Join(c.Dir, "go.mod"))
				if export {
					require.NoError(t, err)
					require.Equal(t, "example.com/original", c.ModulePath)
				} else {
					require.ErrorIs(t, err, os.ErrNotExist)
				}
			}
		})
	}
}
