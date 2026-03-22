package generator

import (
	"github.com/CaliLuke/loom/v3/codegen"
	"github.com/CaliLuke/loom/v3/codegen/service"
	"github.com/CaliLuke/loom/v3/eval"
	"github.com/CaliLuke/loom/v3/expr"
)

// Service iterates through the roots and returns the files needed to render
// the service code. It returns an error if the roots slice does not include
// a goa design.
func Service(genpkg string, roots []eval.Root) ([]*codegen.File, error) {
	var files []*codegen.File
	var userTypePkgs = make(map[string][]string)
	for _, root := range roots {
		r, ok := root.(*expr.RootExpr)
		if !ok {
			continue
		}
		// Create service data
		services := service.NewServicesData(r)

		for _, s := range r.Services {
			// Make sure service is first so name scope is
			// properly initialized.
			files = append(files, service.Files(genpkg, s, services, userTypePkgs)...)
			files = append(files, service.EndpointFile(genpkg, s, services), service.ClientFile(genpkg, s, services))
			if f := service.ViewsFile(genpkg, s, services); f != nil {
				files = append(files, f)
			}
			for _, f := range files {
				if header := f.HeaderTemplate(); header != nil {
					d := services.Get(s.Name)
					service.AddServiceDataMetaTypeImports(header, s, d)
					service.AddUserTypeImports(genpkg, header, d)
				}
			}
			convFiles, err := service.ConvertFiles(r, s, services)
			if err != nil {
				return nil, err
			}
			files = append(files, convFiles...)
		}
	}
	return files, nil
}
