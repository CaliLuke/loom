// Command gosources filters generated Go files from a list of source paths.
package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
)

func main() {
	log.SetFlags(0)
	if err := filterSources(os.Stdin, os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func filterSources(input io.Reader, output io.Writer) error {
	paths := bufio.NewScanner(input)
	for paths.Scan() {
		path := paths.Text()
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly|parser.ParseComments)
		if err != nil {
			return fmt.Errorf("read Go source %s: %w", path, err)
		}
		if ast.IsGenerated(file) {
			continue
		}
		if _, err := fmt.Fprintln(output, path); err != nil {
			return fmt.Errorf("write Go source path: %w", err)
		}
	}
	if err := paths.Err(); err != nil {
		return fmt.Errorf("read Go source paths: %w", err)
	}
	return nil
}
