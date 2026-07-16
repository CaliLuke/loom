// Command release publishes a verified Loom release.
package main

import (
	"context"
	"flag"
	"log"

	loomrelease "github.com/CaliLuke/loom/internal/release"
)

func main() {
	version := flag.String("version", "", "stable release version in vX.Y.Z form")
	flag.Parse()
	if err := loomrelease.Run(context.Background(), loomrelease.Config{Version: *version}); err != nil {
		log.Fatalf("release failed: %v", err)
	}
	log.Printf("Release %s is published and verified", *version)
}
