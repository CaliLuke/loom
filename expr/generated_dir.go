package expr

import (
	"slices"
	"strings"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/internal/naming"
)

type (
	// generatedDirOwner is a design element that owns a generated directory.
	generatedDirOwner struct {
		// name is the design name of the element.
		name string
		// dir is the generated directory name of the element.
		dir string
		// exp is the element expression that validation errors refer to.
		exp eval.Expression
	}

	// cliTransport is a transport whose generated directory holds a client
	// CLI directory, gen/<dir>/cli.
	cliTransport struct {
		// dir is the generated directory of the transport, such as http.
		dir string
		// display names the transport in validation errors.
		display string
		// exposes reports whether the transport exposes the named service.
		exposes func(name string) bool
	}
)

// validateDirOwners adds an error to verr for each owner whose directory is
// the directory of an earlier owner, ignoring case. kind names the owners in
// the error and parent is the directory that contains the directories. The
// directory names are ASCII, so lower casing them folds their case.
func validateDirOwners(verr *eval.ValidationErrors, kind, parent string, owners []generatedDirOwner) {
	seen := make(map[string]generatedDirOwner, len(owners))
	for _, o := range owners {
		key := strings.ToLower(o.dir)
		first, ok := seen[key]
		if !ok {
			seen[key] = o
			continue
		}
		if first.dir == o.dir {
			verr.Add(o.exp, "%s %q and %q both use the generated directory name %q (as in %s/%s); rename one of them",
				kind, first.name, o.name, o.dir, parent, o.dir)
			continue
		}
		verr.Add(o.exp, "%s %q and %q use the generated directory names %q and %q (as in %s/%s and %s/%s), which differ only in case; "+
			"case-insensitive file systems merge them and the Go toolchain rejects import paths that differ only in case, so rename one of them",
			kind, first.name, o.name, first.dir, o.dir, parent, first.dir, parent, o.dir)
	}
}

// validateGeneratedDirs rejects designs in which two servers, or two
// services, generate the same directory. Code generation derives these
// directories from the design names with the naming package, so distinct
// names such as "calc server" and "calc_server" can share one directory, and
// the files of one element would overwrite or merge with the files of the
// other. Directories that differ only in case are also rejected:
// case-insensitive file systems merge them, and the Go toolchain rejects
// import paths that differ only in case.
func (r *RootExpr) validateGeneratedDirs() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if r.API != nil {
		servers := make([]generatedDirOwner, len(r.API.Servers))
		for i, s := range r.API.Servers {
			servers[i] = generatedDirOwner{name: s.Name, dir: naming.ServerDir(s.Name), exp: s}
		}
		validateDirOwners(verr, "servers", "cmd", servers)
	}
	services := make([]generatedDirOwner, len(r.Services))
	for i, s := range r.Services {
		services[i] = generatedDirOwner{name: s.Name, dir: naming.ServiceDir(s.Name), exp: s}
	}
	validateDirOwners(verr, "services", "gen", services)
	if r.API != nil {
		verr.Merge(r.validateCLIDirs())
	}
	return verr
}

// validateCLIDirs rejects designs in which a service whose directory is
// naming.CLIDir shares a package directory with the client CLI tree of a
// transport that exposes the service. The generators put the transport
// packages of such a service, naming.TransportServiceDirs, in
// gen/<transport>/cli, where the client CLI package of each server that hosts
// a service of the transport is gen/<transport>/cli/<server>. A server whose
// directory is one of those packages mixes two packages in one directory.
// HTTP server-only generation, Meta("http:generate", "server"), removes
// gen/http/cli as stale client output, which deletes the HTTP server package
// of the service. The API servers are the default server when the design
// declares none, as RootExpr.Finalize creates it after validation.
func (r *RootExpr) validateCLIDirs() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	servers := r.API.Servers
	if len(servers) == 0 {
		servers = []*ServerExpr{r.API.DefaultServer()}
	}
	mode, _ := r.API.Meta.Last("http:generate")
	for _, svc := range r.Services {
		if !strings.EqualFold(naming.ServiceDir(svc.Name), naming.CLIDir) {
			continue
		}
		for _, t := range r.cliTransports() {
			if !t.exposes(svc.Name) {
				continue
			}
			if t.dir == "http" && mode == "server" {
				verr.Add(svc, "service %q uses the generated directory gen/http/%s, which Meta(\"http:generate\", \"server\") removes as stale HTTP client CLI output; rename the service",
					svc.Name, naming.CLIDir)
				continue
			}
			for _, s := range servers {
				if !slices.ContainsFunc(s.Services, t.exposes) {
					continue
				}
				dir := naming.ServerDir(s.Name)
				for _, pkg := range naming.TransportServiceDirs(t.dir) {
					if strings.EqualFold(dir, pkg) {
						verr.Add(svc, "service %q and server %q both use the generated directory gen/%s/%s/%s: the %s %s package of the service and the %s client CLI package of the server; rename the service or the server",
							svc.Name, s.Name, t.dir, naming.CLIDir, dir, t.display, pkg, t.display)
					}
				}
			}
		}
	}
	return verr
}

// cliTransports returns the transports that generate a client CLI tree.
func (r *RootExpr) cliTransports() []cliTransport {
	return []cliTransport{
		{dir: "http", display: "HTTP", exposes: func(name string) bool {
			return r.API.HTTP != nil && r.API.HTTP.Service(name) != nil
		}},
		{dir: "grpc", display: "gRPC", exposes: func(name string) bool {
			return r.API.GRPC != nil && r.API.GRPC.Service(name) != nil
		}},
		{dir: "jsonrpc", display: "JSON-RPC", exposes: func(name string) bool {
			return r.API.JSONRPC != nil && r.API.JSONRPC.Service(name) != nil
		}},
	}
}
