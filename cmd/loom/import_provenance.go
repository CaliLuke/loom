package main

import (
	"fmt"
	"io"

	"github.com/CaliLuke/loom/internal/openapiimport"
)

func writeProvenance(writer io.Writer, document *openapiimport.Document) error {
	entries := document.Provenance()
	if len(entries) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(writer, "provenance:"); err != nil {
		return err
	}
	for _, entry := range entries {
		if _, err := fmt.Fprintf(writer, "  %s%s -> %s\n", entry.Origin.Document, entry.Origin.Pointer, entry.Path); err != nil {
			return err
		}
		for _, reference := range entry.References {
			if _, err := fmt.Fprintf(writer, "    referenced by %s (%s%s)\n", reference.Path, reference.Origin.Document, reference.Origin.Pointer); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeDiagnosticLocation(writer io.Writer, diagnostic openapiimport.Diagnostic) error {
	if diagnostic.Origin != nil {
		origin := diagnostic.Origin
		if _, err := fmt.Fprintf(writer, "    origin: %s%s\n", origin.Document, origin.Pointer); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(writer, "    path: %s\n", diagnostic.Path)
	return err
}
