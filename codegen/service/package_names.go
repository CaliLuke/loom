package service

import (
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

// PackageBaseName returns the name of the generated package of the service
// or API with the given design name before it is made unique in its name
// scope. It escapes non-ASCII runes so that the package name matches the
// ASCII directory of a service package and uses the same scheme for the API
// package of the example files.
func PackageBaseName(name string) string {
	return codegen.EscapeNonASCII(strings.ToLower(codegen.Goify(name, false)))
}

// DirName returns the directory name of the generated packages of
// the service with the given design name, such as gen/<name> and
// gen/http/<name>/server. Go import paths are ASCII only, so DirName
// escapes the non-ASCII runes of the snake_case name with
// codegen.EscapeNonASCII: the packages of a "Café" service are under
// gen/cafu00e9.
func DirName(name string) string {
	return codegen.EscapeNonASCII(codegen.SnakeCase(codegen.Goify(name, false)))
}
