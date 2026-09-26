package http

import (
	"fmt"
	"net/url"
	"strings"
)

// EscapePathSegment returns value escaped as one URL path segment, so that a
// value containing "/" or other reserved characters cannot change the route.
// It escapes like url.PathEscape and also escapes the dots of the "." and
// ".." segments, which path normalization would otherwise remove. The muxer
// returned by NewMuxer decodes the segment back to value. Generated path
// builders call it for every non-numeric path parameter.
func EscapePathSegment(value string) string {
	switch value {
	case ".":
		return "%2E"
	case "..":
		return "%2E%2E"
	}
	return url.PathEscape(value)
}

// EscapePathRemainder returns value escaped as the remainder of a URL path
// matched by a trailing "{*name}" catch-all wildcard. It keeps the "/"
// separators of value and escapes each segment with EscapePathSegment, so the
// muxer returned by NewMuxer decodes the remainder back to value.
func EscapePathRemainder(value string) string {
	segments := strings.Split(value, "/")
	for i, segment := range segments {
		segments[i] = EscapePathSegment(segment)
	}
	return strings.Join(segments, "/")
}

// RequestURL returns the URL of a request sent to host with scheme, whose
// escaped path is path, as returned by a generated path builder. The URL keeps
// the escaping of path, so an escaped "/" stays inside its segment. When path
// is not a valid escaped path, RequestURL returns the URL with path as its
// unescaped path and the error of unescaping it.
func RequestURL(scheme, host, path string) (*url.URL, error) {
	u := &url.URL{Scheme: scheme, Host: host}
	unescaped, err := url.PathUnescape(path)
	if err != nil {
		u.Path = path
		return u, fmt.Errorf("escaped path %q: %w", path, err)
	}
	u.Path = unescaped
	u.RawPath = path
	return u, nil
}
