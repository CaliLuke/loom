package http

import (
	"io"
	"net/http"
	"time"
)

// FileResponse describes seekable file or media content returned by a service
// method. Name is used to infer Content-Type, and a non-zero ModTime enables
// Last-Modified and conditional request handling. Content must support seeking
// so the standard library can serve byte ranges.
type FileResponse struct {
	// Name is the content name used to infer its media type.
	Name string
	// ModTime is the content modification time. A zero value omits Last-Modified.
	ModTime time.Time
	// Content is the seekable content served in the response.
	Content io.ReadSeeker
}

// ServeHTTP serves the file response using net/http ServeContent semantics.
// Callers must set application response headers before calling ServeHTTP.
func (f *FileResponse) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.ServeContent(w, r, f.Name, f.ModTime, f.Content)
}
