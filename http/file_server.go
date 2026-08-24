package http

import (
	"errors"
	"io/fs"
	"net/http"
	"path"
)

type (
	staticFileServer struct {
		fileSystem       http.FileSystem
		target           string
		directoryHandler http.Handler
	}

	staticPrefixFileSystem struct {
		fileSystem http.FileSystem
		prefix     string
	}

	staticSingleFileSystem struct {
		file http.File
	}
)

// NewStaticFileServer returns a handler that serves target from fileSystem.
// If target is a directory, request paths select files below that directory.
// If target is a file, every request serves that file so wildcard routes can
// implement single-page application fallbacks.
func NewStaticFileServer(fileSystem http.FileSystem, target string) http.Handler {
	if fileSystem == nil {
		fileSystem = http.Dir(".")
	}
	target = path.Join("/", target)
	return &staticFileServer{
		fileSystem: fileSystem,
		target:     target,
		directoryHandler: http.FileServer(staticPrefixFileSystem{
			fileSystem: fileSystem,
			prefix:     target,
		}),
	}
}

func serveStaticFileError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "500 Internal Server Error"
	switch {
	case errors.Is(err, fs.ErrNotExist):
		status = http.StatusNotFound
		message = "404 page not found"
	case errors.Is(err, fs.ErrPermission):
		status = http.StatusForbidden
		message = "403 Forbidden"
	}
	http.Error(w, message, status)
}

func (s *staticFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	file, err := s.fileSystem.Open(s.target)
	if err != nil {
		serveStaticFileError(w, err)
		return
	}
	info, err := file.Stat()
	if err != nil {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
		serveStaticFileError(w, err)
		return
	}
	if info.IsDir() {
		if err := file.Close(); err != nil {
			serveStaticFileError(w, err)
			return
		}
		s.directoryHandler.ServeHTTP(w, r)
		return
	}

	request := r.Clone(r.Context())
	request.URL.Path = "/_loom_static_file"
	request.URL.RawPath = ""
	http.FileServer(staticSingleFileSystem{file: file}).ServeHTTP(w, request)
}

func (s staticPrefixFileSystem) Open(name string) (http.File, error) {
	return s.fileSystem.Open(path.Join(s.prefix, name))
}

func (s staticSingleFileSystem) Open(string) (http.File, error) {
	return s.file, nil
}
