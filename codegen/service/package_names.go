package service

import (
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/internal/naming"
)

// PackageBaseName returns the name of the generated package of the service
// or API with the given design name before it is made unique in its name
// scope. It escapes non-ASCII runes so that the package name matches the
// ASCII directory of a service package and uses the same scheme for the API
// package of the example files.
func PackageBaseName(name string) string {
	return naming.EscapeNonASCII(strings.ToLower(codegen.Goify(name, false)))
}
