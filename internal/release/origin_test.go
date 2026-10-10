package release

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPublicationValidatesEffectivePushDestinations(t *testing.T) {
	for _, c := range []struct {
		name     string
		fetch    string
		config   [][2]string
		rejected bool
	}{
		{name: "default HTTPS", fetch: "https://github.com/CaliLuke/loom.git"},
		{name: "SSH", fetch: "git@github.com:CaliLuke/loom.git"},
		{name: "HTTPS without suffix", fetch: "https://github.com/CaliLuke/loom"},
		{name: "noncanonical fetch", fetch: "https://example.invalid/loom.git", rejected: true},
		{name: "separate destination", config: [][2]string{{"remote.origin.pushurl", "../other.git"}}, rejected: true},
		{name: "all canonical", config: [][2]string{{"remote.origin.pushurl", "https://github.com/CaliLuke/loom.git"}, {"remote.origin.pushurl", "git@github.com:CaliLuke/loom.git"}}},
		{name: "second destination", config: [][2]string{{"remote.origin.pushurl", "https://github.com/CaliLuke/loom.git"}, {"remote.origin.pushurl", "../other.git"}}, rejected: true},
		{name: "push rewrite", config: [][2]string{{"url.https://example.invalid/.pushInsteadOf", "https://github.com/"}}, rejected: true},
		{name: "explicit push rewrite", config: [][2]string{{"remote.origin.pushurl", "push:loom.git"}, {"url.https://example.invalid/.insteadOf", "push:"}}, rejected: true},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
			t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(dir, "global.gitconfig"))
			git := func(args ...string) []byte {
				cmd := exec.CommandContext(t.Context(), "git", args...)
				cmd.Dir = dir
				out, err := cmd.CombinedOutput()
				require.NoError(t, err, "%s", out)
				return out
			}
			git("init", "--quiet")
			fetch := c.fetch
			if fetch == "" {
				fetch = "https://github.com/CaliLuke/loom.git"
			}
			git("remote", "add", "origin", fetch)
			for _, setting := range c.config {
				git("config", "--add", setting[0], setting[1])
			}
			reachedFetch := errors.New("validated origin; stop before network")
			var commands []string
			p := publisher{config: Config{Mode: "daily"}, run: func(_ context.Context, name string, args ...string) ([]byte, error) {
				commands = append(commands, name+" "+strings.Join(args, " "))
				require.Equal(t, "git", name)
				if args[0] == "fetch" {
					return nil, reachedFetch
				}
				require.Equal(t, "remote", args[0], "no publication mutations before validation")
				return git(args...), nil
			}}
			err := p.publish(t.Context())
			if c.rejected {
				require.ErrorContains(t, err, "not canonical Loom")
				require.NotContains(t, commands, "git fetch origin main --tags")
			} else {
				require.ErrorIs(t, err, reachedFetch)
				require.Contains(t, commands, "git remote get-url --push --all origin")
			}
		})
	}
}
