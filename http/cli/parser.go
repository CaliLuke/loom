// Package cli contains runtime helpers for generated HTTP command-line clients.
package cli

import (
	"flag"

	"github.com/alecthomas/kong"
)

// Parse parses args into commandLine and returns the selected command path.
func Parse(commandLine any, program string, args []string) (string, error) {
	parser, err := kong.New(commandLine, kong.Name(program), kong.Exit(func(int) {}))
	if err != nil {
		return "", err
	}
	ctx, err := parser.Parse(args)
	if err != nil {
		if HasHelpArg(args) {
			return "", flag.ErrHelp
		}
		return "", err
	}
	return ctx.Command(), nil
}

// HasHelpArg reports whether args request Kong context-sensitive help.
func HasHelpArg(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}
