package service

import (
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

// PackageBaseName returns the name of the generated package of the service
// with the given design name before it is made unique in the service name
// scope. It escapes non-ASCII runes so that the package name matches the
// ASCII directory of the package.
func PackageBaseName(name string) string {
	return codegen.EscapeNonASCII(strings.ToLower(codegen.Goify(name, false)))
}

// servicePathName returns the directory name of the generated packages of
// the service with the given design name. Go import paths are ASCII only, so
// servicePathName escapes the non-ASCII runes of the snake_case name.
func servicePathName(name string) string {
	return codegen.EscapeNonASCII(codegen.SnakeCase(codegen.Goify(name, false)))
}
