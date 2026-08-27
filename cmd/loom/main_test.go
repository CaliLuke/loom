package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunPackageCommandHelp(t *testing.T) {
	original := gen
	t.Cleanup(func() { gen = original })
	gen = func(_, _, _ string, _ bool) error {
		t.Error("generator must not run for help")
		return nil
	}

	for _, command := range []string{"gen", "example", "test-scaffold"} {
		for _, args := range [][]string{
			{"-h"},
			{"--help"},
			{"example.com/service/design", "-h"},
			{"example.com/service/design", "--help"},
		} {
			name := command + " " + strings.Join(args, " ")
			t.Run(name, func(t *testing.T) {
				var stderr bytes.Buffer

				exitCode := runPackageCommand(command, args, &stderr)

				require.Zero(t, exitCode)
				require.Contains(t, stderr.String(), "Usage:\n  loom "+command+" PACKAGE")
			})
		}
	}
}

func TestRunPackageCommandRequiresPackage(t *testing.T) {
	for _, command := range []string{"gen", "example", "test-scaffold"} {
		t.Run(command, func(t *testing.T) {
			var stderr bytes.Buffer

			exitCode := runPackageCommand(command, nil, &stderr)

			require.Equal(t, 1, exitCode)
			require.Contains(t, stderr.String(), "usage: loom "+command+" PACKAGE")
		})
	}
}

func TestCmdLine(t *testing.T) {
	const (
		testPkg    = "/test"
		testOutput = "testOutput"
	)
	var (
		usageCalled  bool
		cmd          string
		path, output string
		debug        bool
	)

	usage = func() { usageCalled = true }
	gen = func(c string, p, o string, d bool) error { cmd, path, output, debug = c, p, o, d; return nil }
	defer func() {
		usage = help
		gen = generate
	}()

	cases := map[string]struct {
		CmdLine         string
		ExpectedUsage   bool
		ExpectedCommand string
		ExpectedPath    string
		ExpectedOutput  string
		ExpectedDebug   bool
	}{
		"gen":           {"gen " + testPkg, false, "gen", testPkg, ".", false},
		"test scaffold": {"test-scaffold " + testPkg, false, "test-scaffold", testPkg, ".", false},

		"invalid":     {"invalid " + testPkg, true, "", "", "", false},
		"empty":       {"", true, "", "", "", false},
		"invalid gen": {"invalid gen" + testPkg, true, "", "", "", false},

		"output":       {"gen " + testPkg + " -output " + testOutput, false, "gen", testPkg, testOutput, false},
		"output short": {"gen " + testPkg + " -o " + testOutput, false, "gen", testPkg, testOutput, false},

		"debug": {"gen " + testPkg + " -debug", false, "gen", testPkg, ".", true},
	}

	for k, c := range cases {
		{
			args := strings.Split(c.CmdLine, " ")
			os.Args = append([]string{"loom"}, args...)
			usageCalled = false
			cmd = ""
			path = ""
			output = ""
			debug = false
		}

		main()

		if usageCalled != c.ExpectedUsage {
			t.Errorf("%s: Expected usage to be %v but got %v", k, c.ExpectedUsage, usageCalled)
		}
		if cmd != c.ExpectedCommand {
			t.Errorf("%s: Expected command to be %s but got %s", k, c.ExpectedCommand, cmd)
		}
		if path != c.ExpectedPath {
			t.Errorf("%s: Expected path to be %s but got %s", k, c.ExpectedPath, path)
		}
		if output != c.ExpectedOutput {
			t.Errorf("%s: Expected output to be %s but got %s", k, c.ExpectedOutput, output)
		}
		if debug != c.ExpectedDebug {
			t.Errorf("%s: Expected debug to be %v but got %v", k, c.ExpectedDebug, debug)
		}
	}
}
