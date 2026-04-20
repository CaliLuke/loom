package service

import (
	"go/build"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
)

func commonPath(sep byte, paths ...string) string {
	switch len(paths) {
	case 0:
		return ""
	case 1:
		return path.Clean(paths[0])
	}

	c := []byte(path.Clean(paths[0]))
	c = append(c, sep)

	for _, currentPath := range paths[1:] {
		currentPath = path.Clean(currentPath) + string(sep)
		if len(currentPath) < len(c) {
			c = c[:len(currentPath)]
		}
		for i := 0; i < len(c); i++ {
			if currentPath[i] != c[i] {
				c = c[:i]
				break
			}
		}
	}

	for i := len(c) - 1; i >= 0; i-- {
		if c[i] == sep {
			c = c[:i]
			break
		}
	}

	return string(c)
}

// getPkgImport returns the correct import path of a package.
// It's needed because the "reflect" package provides the binary import path
// ("github.com/CaliLuke/loom/vendor/some/package") for vendored packages
// instead the source import path ("some/package")
func getPkgImport(pkg, cwd string) string {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	gosrc := path.Join(filepath.ToSlash(gopath), "src")
	cwd = filepath.ToSlash(cwd)

	if !strings.HasPrefix(cwd, gosrc) {
		return pkg
	}

	pkgpath := path.Join(gosrc, pkg)
	parentpath := commonPath(os.PathSeparator, cwd, pkgpath)
	if parentpath == gosrc {
		return pkg
	}

	rootpkg := parentpath[len(gosrc)+1:]
	vendorPrefix := path.Join(rootpkg, "vendor")
	if strings.HasPrefix(pkg, vendorPrefix) {
		return pkg[len(vendorPrefix)+1:]
	}

	return pkg
}

func getExternalTypeInfo(external any) (string, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}

	pkg := reflect.TypeOf(external)
	pkgImport := getPkgImport(pkg.PkgPath(), cwd)
	alias := strings.Split(pkg.String(), ".")[0]
	return pkgImport, alias, nil
}
