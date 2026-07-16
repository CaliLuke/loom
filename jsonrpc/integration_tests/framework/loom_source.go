package framework

import (
	"path/filepath"

	"github.com/CaliLuke/loom/internal/loomsource"
)

// repoRootReplace returns the selected Loom checkout for the generated
// temporary module's replace directive.
func (g *Generator) repoRootReplace() (string, error) {
	repoRoot, err := loomsource.RepositoryRoot(".")
	if err != nil {
		return "", err
	}
	return loomsource.Resolve(repoRoot, filepath.Join(g.workDir, ".loom-pinned"))
}
