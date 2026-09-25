package service

import (
	"slices"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/naming"
)

// PackageBaseName returns the name of the generated package of the service
// or API with the given design name before it is made unique in its name
// scope. It escapes non-ASCII runes so that the package name matches the
// ASCII directory of a service package and uses the same scheme for the API
// package of the example files.
func PackageBaseName(name string) string {
	return naming.EscapeNonASCII(strings.ToLower(codegen.Goify(name, false)))
}

// PackageName returns the name of the generated package of the service of
// root, the value of Data.PkgName. It is PackageBaseName of the service name
// with an "svc" suffix when that name is reserved, or when the service has a
// transport and the name also names a struct:pkg:path package of the service
// types: the transport files import both packages. Without a transport no
// generated file imports both under the same name (the example service stub
// aliases the type package), so the name is kept.
func PackageName(root *expr.RootExpr, service *expr.ServiceExpr) string {
	return servicePackageName(newServiceNameScope(nil), root, service)
}

// newServiceNameScope returns the name scope of the service package with the
// names that generated service code reserves. The scope names the
// struct:pkg:path packages as names does, see
// codegen.NewNameScopeWithPackageNames.
func newServiceNameScope(names map[string]string) *codegen.NameScope {
	scope := codegen.NewNameScopeWithPackageNames(names)
	scope.Unique("Use")
	scope.Unique("websocket")
	return scope
}

// servicePackageName allocates the package name of the service of root in
// scope, see PackageName.
func servicePackageName(scope *codegen.NameScope, root *expr.RootExpr, service *expr.ServiceExpr) string {
	base := PackageBaseName(service.Name)
	if hasTransport(root, service) && usesUserTypePackage(root, service, base) {
		scope.Unique(base)
	}
	return scope.HashedUnique(service, base, "svc")
}

// hasTransport reports whether root exposes service over HTTP, gRPC or
// JSON-RPC. A nil root reports true so that the name stays collision-free.
func hasTransport(root *expr.RootExpr, service *expr.ServiceExpr) bool {
	if root == nil || root.API == nil {
		return true
	}
	api := root.API
	return api.HTTP != nil && api.HTTP.Service(service.Name) != nil ||
		api.GRPC != nil && api.GRPC.Service(service.Name) != nil ||
		api.JSONRPC != nil && api.JSONRPC.HTTPExpr.Service(service.Name) != nil
}

// usesUserTypePackage reports whether a user type reachable from the methods
// or errors of service, or a type of root generated for it with
// "type:generate:force" metadata, is generated in a struct:pkg:path package
// named name other than the service package itself.
func usesUserTypePackage(root *expr.RootExpr, service *expr.ServiceExpr, name string) bool {
	dir := naming.ServiceDir(service.Name)
	atts := append(serviceAttributes(service), forcedTypeAttributes(root, service)...)
	return findUserType(atts, func(loc *codegen.Location) bool {
		return loc.PackageName() == name && loc.RelImportPath != dir
	}) != nil
}

// ownPackageUserType returns a user type reachable from the methods or
// errors of service that struct:pkg:path places in the package generated for
// the service, nil if there is none. The service files would import their
// own package to reference it.
func ownPackageUserType(service *expr.ServiceExpr) expr.UserType {
	dir := naming.ServiceDir(service.Name)
	return findUserType(serviceAttributes(service), func(loc *codegen.Location) bool {
		return loc.RelImportPath == dir
	})
}

// serviceAttributes returns the attributes of the payloads, results and
// errors of the methods of service and of its errors.
func serviceAttributes(service *expr.ServiceExpr) []*expr.AttributeExpr {
	atts := make([]*expr.AttributeExpr, 0, 4*len(service.Methods)+len(service.Errors))
	for _, e := range service.Errors {
		atts = append(atts, e.AttributeExpr)
	}
	for _, m := range service.Methods {
		atts = append(atts, m.Payload, m.StreamingPayload, m.Result, m.StreamingResult)
		for _, e := range m.Errors {
			atts = append(atts, e.AttributeExpr)
		}
	}
	return atts
}

// forcedTypeAttributes returns attributes of the types of root that
// "type:generate:force" metadata generates for service.
func forcedTypeAttributes(root *expr.RootExpr, service *expr.ServiceExpr) []*expr.AttributeExpr {
	if root == nil {
		return nil
	}
	var atts []*expr.AttributeExpr
	for _, t := range root.Types {
		svcs, ok := t.Attribute().Meta["type:generate:force"]
		if ok && (len(svcs) == 0 || slices.Contains(svcs, service.Name)) {
			atts = append(atts, &expr.AttributeExpr{Type: t})
		}
	}
	return atts
}

// findUserType returns the first user type reachable from atts, in order,
// whose struct:pkg:path location satisfies match, nil if there is none.
func findUserType(atts []*expr.AttributeExpr, match func(*codegen.Location) bool) expr.UserType {
	var found expr.UserType
	for _, att := range atts {
		if att == nil || found != nil {
			continue
		}
		err := codegen.Walk(att, func(a *expr.AttributeExpr) error {
			if loc := codegen.UserTypeLocation(a.Type); found == nil && loc != nil && match(loc) {
				found = a.Type.(expr.UserType)
			}
			return nil
		})
		if err != nil {
			panic(codegen.NewError(nil, att, err))
		}
	}
	return found
}
