package cli

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/require"
)

type testCommandLine struct {
	Users struct {
		Create struct {
			Name string `name:"name" required:""`
		} `cmd:"" name:"create"`
	} `cmd:"" name:"users"`
}

func TestParseReturnsSelectedCommand(t *testing.T) {
	var command testCommandLine

	path, err := Parse(&command, "loom-test", []string{"users", "create", "--name", "Loom"})

	require.NoError(t, err)
	require.Equal(t, "users create", path)
	require.Equal(t, "Loom", command.Users.Create.Name)
}

func TestParseMapsHelpToFlagErrHelp(t *testing.T) {
	var command testCommandLine

	_, err := Parse(&command, "loom-test", []string{"users", "create", "--help"})

	require.ErrorIs(t, err, flag.ErrHelp)
}

func TestHasHelpArg(t *testing.T) {
	require.True(t, HasHelpArg([]string{"users", "-h"}))
	require.True(t, HasHelpArg([]string{"users", "--help"}))
	require.False(t, HasHelpArg([]string{"users", "create"}))
}
