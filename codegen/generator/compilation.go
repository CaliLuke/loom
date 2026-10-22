package generator

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
)

// generationFiles is invoked after all plugin preparation has completed.
// Owners are local to this invocation; no captured semantics survive a run.
func generationFiles(genpkg string, roots []eval.Root) ([]*codegen.File, error) {
	type design struct {
		root     *expr.RootExpr
		semantic *service.SemanticData
	}
	var designs []design
	for _, root := range roots {
		if r, ok := root.(*expr.RootExpr); ok {
			designs = append(designs, design{r, service.NewSemanticData(r)})
		}
	}
	var files []*codegen.File
	userTypePkgs := make(map[string][]string)
	for _, d := range designs {
		generated, err := serviceFiles(genpkg, d.root, d.semantic.Services(), userTypePkgs)
		if err != nil {
			return nil, err
		}
		files = append(files, generated...)
	}
	for _, d := range designs {
		generated, err := transportFiles(genpkg, d.root, d.semantic.Services())
		if err != nil {
			return nil, err
		}
		files = append(files, generated...)
	}
	if len(designs) > 0 {
		d := designs[0]
		generated, err := httpcodegen.OpenAPIFiles(d.root, openapiv3.WithServiceSemantics(d.semantic))
		if err != nil {
			return nil, err
		}
		files = append(files, generated...)
	}
	return files, nil
}
