// Command loomsource manages the worktree-local Loom source used by temporary
// integration modules.
package main

import (
	"fmt"
	"os"

	"github.com/CaliLuke/loom/internal/loomsource"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: loomsource <local|remote|status>")
	}
	repoRoot, err := loomsource.RepositoryRoot(".")
	if err != nil {
		return err
	}

	switch args[0] {
	case string(loomsource.ModeLocal):
		localDir := os.Getenv("LOOM_DIR")
		if err := loomsource.SetMode(repoRoot, loomsource.ModeLocal, localDir); err != nil {
			return err
		}
		if localDir == "" {
			localDir = repoRoot
		}
		fmt.Printf("loom source mode: local (%s)\n", localDir)
		return nil
	case string(loomsource.ModeRemote):
		if err := loomsource.SetMode(repoRoot, loomsource.ModeRemote, ""); err != nil {
			return err
		}
		fmt.Println("loom source mode: remote")
		return nil
	case "status":
		config, err := loomsource.ReadMode(repoRoot)
		if err != nil {
			return err
		}
		if config.Mode == loomsource.ModeLocal {
			localDir := config.LocalDir
			if localDir == "" {
				localDir = repoRoot
			}
			fmt.Printf("loom source mode: local (%s)\n", localDir)
			return nil
		}
		fmt.Println("loom source mode: remote")
		return nil
	default:
		return fmt.Errorf("usage: loomsource <local|remote|status>")
	}
}
