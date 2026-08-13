package generator

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/eval"
)

type genfunc func(genpkg string, roots []eval.Root) ([]*codegen.File, error)

// generatorLoader is replaceable by package tests that exercise the generation
// pipeline with controlled output.
var generatorLoader = generators

// generators returns the generator functions exposed by the generator package
// for the given command.
func generators(cmd string) ([]genfunc, error) {
	switch cmd {
	case "gen":
		return []genfunc{Service, Transport, OpenAPI}, nil
	case "example":
		return []genfunc{Example}, nil
	case "test-scaffold":
		return []genfunc{TestScaffold}, nil
	default:
		return nil, fmt.Errorf("unknown command %q", cmd)
	}
}
