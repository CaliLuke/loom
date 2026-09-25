package expr

import (
	"strings"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/internal/naming"
)

type (
	// generatedDirOwner is a design element that owns a generated directory.
	generatedDirOwner struct {
		// name is the design name of the element.
		name string
		// dir is the generated directory name of the element.
		dir string
		// exp is the element expression that validation errors refer to.
		exp eval.Expression
	}
)

// validateDirOwners adds an error to verr for each owner whose directory is
// the directory of an earlier owner, ignoring case. kind names the owners in
// the error and parent is the directory that contains the directories. The
// directory names are ASCII, so lower casing them folds their case.
func validateDirOwners(verr *eval.ValidationErrors, kind, parent string, owners []generatedDirOwner) {
	seen := make(map[string]generatedDirOwner, len(owners))
	for _, o := range owners {
		key := strings.ToLower(o.dir)
		first, ok := seen[key]
		if !ok {
			seen[key] = o
			continue
		}
		if first.dir == o.dir {
			verr.Add(o.exp, "%s %q and %q both use the generated directory name %q (as in %s/%s); rename one of them",
				kind, first.name, o.name, o.dir, parent, o.dir)
			continue
		}
		verr.Add(o.exp, "%s %q and %q use the generated directory names %q and %q (as in %s/%s and %s/%s), which differ only in case; "+
			"case-insensitive file systems merge them and the Go toolchain rejects import paths that differ only in case, so rename one of them",
			kind, first.name, o.name, first.dir, o.dir, parent, first.dir, parent, o.dir)
	}
}

// validateGeneratedDirs rejects designs in which two servers, or two
// services, generate the same directory. Code generation derives these
// directories from the design names with the naming package, so distinct
// names such as "calc server" and "calc_server" can share one directory, and
// the files of one element would overwrite or merge with the files of the
// other. Directories that differ only in case are also rejected:
// case-insensitive file systems merge them, and the Go toolchain rejects
// import paths that differ only in case.
func (r *RootExpr) validateGeneratedDirs() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if r.API != nil {
		servers := make([]generatedDirOwner, len(r.API.Servers))
		for i, s := range r.API.Servers {
			servers[i] = generatedDirOwner{name: s.Name, dir: naming.ServerDir(s.Name), exp: s}
		}
		validateDirOwners(verr, "servers", "cmd", servers)
	}
	services := make([]generatedDirOwner, len(r.Services))
	for i, s := range r.Services {
		services[i] = generatedDirOwner{name: s.Name, dir: naming.ServiceDir(s.Name), exp: s}
	}
	validateDirOwners(verr, "services", "gen", services)
	return verr
}
