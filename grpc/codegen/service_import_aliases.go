package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/naming"
)

// transportImportAliases qualifies service and protocol buffer types without
// changing either generated package declaration.
type transportImportAliases struct {
	service  string
	protobuf string
}

const grpcClientReceiverName = "c"

// transportGeneratedImportNames reserves the fixed imports of generated gRPC
// transport and example files before service aliases are allocated. Generated
// import coverage keeps this list aligned with the rendered files.
var transportGeneratedImportNames = []string{
	"codes", "context", "debug", "errors", "flag", "fmt", "grpc", "insecure",
	"io", "json", "log", "loom", "loomgrpc", "loompb", "metadata", "net", "os",
	"protojson", "reflection", "strconv", "strings", "structpb", "sync", "testing",
	"time", "url", "utf8",
}

func newImportAliases(root *expr.RootExpr) map[string]transportImportAliases {
	scope := codegen.NewNameScope()
	for _, name := range transportGeneratedImportNames {
		scope.Unique(name)
	}
	// Client methods use c as their receiver. Reserve it because selectors on
	// that receiver are otherwise indistinguishable from import qualifiers.
	scope.Unique(grpcClientReceiverName)
	aliases := make(map[string]transportImportAliases, len(root.API.GRPC.Services))
	// Reserve protocol buffer imports before service imports: every aggregate
	// file can import both sets, regardless of the order of its services.
	for _, grpcService := range root.API.GRPC.Services {
		aliases[grpcService.Name()] = transportImportAliases{
			protobuf: scope.Unique(naming.ServiceDir(grpcService.Name()) + pbPkgName),
		}
	}
	for _, grpcService := range root.API.GRPC.Services {
		entry := aliases[grpcService.Name()]
		entry.service = scope.Unique(service.PackageName(root, grpcService.ServiceExpr), "svc")
		aliases[grpcService.Name()] = entry
	}
	return aliases
}

// metadataVarScope keeps locals separate from the imports used to qualify
// transport and custom types. It must not mutate the neutral service scope:
// doing so would also change names in the generated service declarations.
func metadataVarScope(sd *ServiceData, att *expr.AttributeExpr) *codegen.NameScope {
	scope := codegen.NewNameScope()
	reserve := func(name string) {
		if name != "" && scope.PeekUnique(name) == name {
			scope.Unique(name)
		}
	}
	for _, name := range transportGeneratedImportNames {
		reserve(name)
	}
	reserve(sd.Service.PkgName)
	reserve(sd.Service.ViewsPkg)
	reserve(sd.PkgName)
	if err := codegen.Walk(att, func(a *expr.AttributeExpr) error {
		reserve(sd.Service.Scope.PackageName(codegen.UserTypeLocation(a.Type)))
		return nil
	}); err != nil {
		panic(err)
	}
	return scope
}
