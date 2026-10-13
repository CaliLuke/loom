package http

import (
	"fmt"
	"net/http"

	"github.com/CaliLuke/loom/internal/headerlist"
)

// HeaderValues decodes a schema-based simple collection header. Repeated field
// lines and comma-separated items are equivalent. An absent field returns nil;
// a present field with no nonempty items returns a nonnil empty slice. Scalars,
// cookies and header-specific quoted grammars must use their own codecs.
func HeaderValues(header http.Header, name string) []string {
	lines, present := header[http.CanonicalHeaderKey(name)]
	return headerlist.Decode(lines, present)
}

// SetHeaderValues writes a schema-based simple collection header. Nil values
// leave the header unchanged; nonnil empty values write an empty field. It rejects
// unrepresentable items before changing the header, without disclosing their text.
// Set-Cookie is not a comma-list field and must use the cookie APIs instead.
func SetHeaderValues(header http.Header, name string, values []string) error {
	if values == nil {
		return nil
	}
	value, err := headerlist.Encode(values)
	if err != nil {
		return fmt.Errorf("header %q: %w", name, err)
	}
	header.Set(name, value)
	return nil
}
