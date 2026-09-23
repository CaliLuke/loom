package codegen

import (
	"path"
	"regexp"

	"github.com/CaliLuke/loom/codegen"
)

// transportVarScope allocates the variables that the generated functions of
// one request, path builder, or response declare from design attributes,
// together with the locals those functions derive from each variable.
type transportVarScope struct {
	names  *codegen.NameScope
	derive func(names *codegen.NameScope, varName string) DerivedVarNames
}

// transportFunctionImportNames lists the local names of the fixed imports of
// the generated files that hold the transport functions using attribute
// variables: the server and client encode_decode.go and types.go files, the
// client CLI support file, and the path builder files. Every variable scope
// reserves them so that no variable shadows a package that the function may
// reference. TestTransportFunctionImportNamesCoverFileImports keeps the list
// aligned with the import lists of those files.
var transportFunctionImportNames = []string{
	"bytes", "context", "errors", "fmt", "http", "io", "json", "jsontext", "loom", "loomhttp",
	"multipart", "os", "strconv", "strings", "url", "utf8",
}

// requestLocalNames lists the identifiers that the generated server request
// decoder, server payload initializer, and CLI payload builder of an endpoint
// declare or receive in the scopes where they also use the variables
// named after the request query params, headers, and cookies.
var requestLocalNames = []string{
	"body", "closeIdx", "decoder", "err", "err2", "i", "keyRaw", "mr", "multipartErr",
	"multipartForm", "mux", "ok", "openIdx", "params", "pass", "payload", "present", "pv", "r", "raw",
	"rawValues", "req", "res", "rv", "user", "v", "val", "valRaw",
}

// cookieLocalNames lists the identifiers that the generated server request
// decoder declares in the block that decodes each request cookie.
var cookieLocalNames = []string{"c", "cookieErr"}

// mapQueryLocalPattern matches the key and value locals of the map query
// decoding blocks, one set per map nesting level.
var mapQueryLocalPattern = regexp.MustCompile(`^(key[a-z]?(Err|Raw)?|val[a-z]?(Raw)?)$`)

// pathLocalNames lists the identifiers that a generated path builder and the
// client request builder that calls it declare or receive in the scopes where
// they also use the variables named after the path params.
var pathLocalNames = []string{
	"body", "c", "ctx", "err", "ok", "p", "req", "u", "v",
}

// responseLocalNames lists the identifiers that the generated server response
// or error encoder, client response decoder, and client result initializer of
// one response declare or receive in the scopes where they also use
// the variables named after the response headers and cookies.
var responseLocalNames = []string{
	"body", "cookies", "ctx", "decoder", "en", "enc", "encodeError", "encoder", "err", "err2", "formatter",
	"i", "p", "problem", "pv", "res", "resp", "restoreBody", "rv", "v", "view", "vres", "w",
}

// newRequestVarScope returns the variable scope of the server request
// decoder, server payload initializer, and CLI payload builder of an
// endpoint.
func newRequestVarScope(sd *ServiceData) *transportVarScope {
	return newTransportVarScope(sd, requestLocalNames, func(names *codegen.NameScope, varName string) DerivedVarNames {
		return DerivedVarNames{
			Raw:       names.Unique(varName + "Raw"),
			RawSlice:  names.Unique(varName + "RawSlice"),
			HasValues: names.Unique(varName + "HasValues"),
			Val:       names.Unique(varName + "Val"),
		}
	})
}

// newPathVarScope returns the variable scope of a path builder and of the
// client request builder that calls it.
func newPathVarScope(sd *ServiceData) *transportVarScope {
	return newTransportVarScope(sd, pathLocalNames, func(names *codegen.NameScope, varName string) DerivedVarNames {
		return DerivedVarNames{Slice: names.Unique(varName + "Slice")}
	})
}

// newResponseVarScope returns the variable scope of the server encoder,
// client decoder, and client initializer of one response or error response.
func newResponseVarScope(sd *ServiceData) *transportVarScope {
	return newTransportVarScope(sd, responseLocalNames, func(names *codegen.NameScope, varName string) DerivedVarNames {
		encoded := names.Unique(varName + "s")
		return DerivedVarNames{
			Raw:          names.Unique(varName + "Raw"),
			Val:          names.Unique(varName + "Val"),
			Slice:        names.Unique(varName + "Slice"),
			Encoded:      encoded,
			EncodedSlice: names.Unique(encoded + "Slice"),
			EncodedRaw:   names.Unique(varName + "raw"),
		}
	})
}

// newTransportVarScope returns a scope that reserves the fixed import names of
// the generated transport files, the aliases of the service, views, and user
// type packages of sd, and the given locals. It derives the locals of each
// allocated variable with derive.
func newTransportVarScope(sd *ServiceData, locals []string, derive func(*codegen.NameScope, string) DerivedVarNames) *transportVarScope {
	s := &transportVarScope{names: codegen.NewNameScope(), derive: derive}
	s.reserve(transportFunctionImportNames...)
	s.reserve(sd.Service.PkgName, sd.Service.ViewsPkg)
	for _, spec := range sd.Service.UserTypeImports {
		s.reserve(importName(spec))
	}
	s.reserve(locals...)
	return s
}

// importName returns the local name that the generated file uses for spec.
func importName(spec *codegen.ImportSpec) string {
	if spec.Name != "" {
		return spec.Name
	}
	return path.Base(spec.Path)
}

// reserve prevents later variables from using the given names.
func (s *transportVarScope) reserve(names ...string) {
	for _, name := range names {
		s.names.Unique(name)
	}
}

// allocate reserves a variable named after base, or after base with a numeric
// suffix when base is taken or avoid reports the name, and reserves the locals
// derived from it. avoid may be nil.
func (s *transportVarScope) allocate(base string, avoid func(string) bool) (string, DerivedVarNames) {
	varName := s.names.Unique(base)
	for avoid != nil && avoid(varName) {
		varName = s.names.Unique(base)
	}
	return varName, s.derive(s.names, varName)
}

// isMapQueryLocal reports whether name is a local that the generated server
// request decoder declares while it decodes the keys and values of a map
// query param into the param variable.
func isMapQueryLocal(name string) bool {
	return mapQueryLocalPattern.MatchString(name)
}
