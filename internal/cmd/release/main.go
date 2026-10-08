// Command release dispatches publication, or runs the publisher inside GitHub Actions.
package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/CaliLuke/loom/internal/release"
)

func main() {
	var config release.Config
	flag.StringVar(&config.Mode, "mode", "alpha", "alpha, daily, or promote")
	flag.StringVar(&config.Source, "source", "", "exact source commit SHA")
	flag.StringVar(&config.Alpha, "alpha", "", "published alpha to promote")
	flag.StringVar(&config.Version, "version", "", "target version (required for promotion)")
	publish := flag.Bool("publish", false, "publish inside the trusted GitHub workflow")
	flag.Parse()
	ctx := context.Background()
	var err error
	if *publish {
		if os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("GITHUB_REPOSITORY") != "CaliLuke/loom" || os.Getenv("GITHUB_REF") != "refs/heads/main" {
			log.Fatal("publication requires the main-branch GitHub workflow")
		}
		err = release.Run(ctx, config)
	} else {
		err = release.Dispatch(ctx, config)
	}
	if err != nil {
		log.Fatal(err)
	}
}
