package codegen

// transportGeneratedImportNames is the union of local names from literal
// ImportSpec and Loom import constructors in generated files that also emit a
// dynamic transport import. TestTransportImportAliasReservationsCoverGeneratedImports
// keeps it aligned with those constructors. Service, view, and user-type
// imports are dynamic design data and therefore allocate their own aliases.
var transportGeneratedImportNames = []string{
	"atomic",
	"bufio",
	"bytes",
	"context",
	"debug",
	"errors",
	"flag",
	"fmt",
	"http",
	"io",
	"json",
	"jsontext",
	"jsonrpc",
	"log",
	"loom",
	"loomhttp",
	"loomhttpcli",
	"loomtransport",
	"middleware",
	"multipart",
	"os",
	"path",
	"strconv",
	"strings",
	"sync",
	"testing",
	"time",
	"url",
	"utf8",
	"websocket",
}

// transportGeneratedLocalNames lists the names that the generated HTTP and
// JSON-RPC transport, example server and CLI files declare where a service
// package imported under the same name would not compile: the receivers,
// parameters and locals in scope where they use a service package, such as
// the server receiver s, the request r and the response writer w, the
// package-level names of their packages, and the locals that are the
// operand of a selector in a file that does not use the service package,
// which would keep its import unused. A service package named like one of
// them is imported under an alias, see newServiceImportAliases.
// TestTransportLocalNameReservationsCoverGeneratedLocals and, for JSON-RPC,
// TestServicePackageNamedLikeLocalGeneratedModuleBuilds collect these names
// from the generated code (testingx.ServicePackageShadowNames) and fail
// when the list misses one.
var transportGeneratedLocalNames = []string{
	"body", "c", "cancel", "configurer", "conn", "ctx", "data", "decoder",
	"encoder", "endpoint", "err", "errhandler", "event", "f", "formatter",
	"fpath", "id", "jresp", "lifecycle", "mux", "mw", "p", "params", "parsed",
	"payload", "r", "req", "res", "resp", "response", "rv", "s", "stream", "strm",
	"u", "upgrader", "v", "view", "vres", "w", "ws", "wsconn",
}
