// Package naming derives the Go identifiers, file names and directory names
// that Loom generates from design names. It depends on no other Loom package,
// so the expr package can validate designs with the same names that code
// generation uses.
package naming

// CLIDir is the name of the directory of gen/<transport> that holds the
// client CLI support package of each server, gen/<transport>/cli/<ServerDir>.
// The transport packages of a service whose ServiceDir is CLIDir are in the
// same directory.
const CLIDir = "cli"

// ServerDir returns the name of the directories and packages generated for
// the server with the given design name: the example commands under cmd and
// the client CLI support packages under gen/<transport>/cli. Go import paths
// are ASCII only, so ServerDir escapes the non-ASCII runes of the snake_case
// name with EscapeNonASCII.
func ServerDir(name string) string {
	return EscapeNonASCII(SnakeCase(Goify(name, true)))
}

// ServiceDir returns the directory name of the generated packages of the
// service with the given design name, such as gen/<name> and
// gen/http/<name>/server. Go import paths are ASCII only, so ServiceDir
// escapes the non-ASCII runes of the snake_case name with EscapeNonASCII:
// the packages of a "Café" service are under gen/cafu00e9.
func ServiceDir(name string) string {
	return EscapeNonASCII(SnakeCase(Goify(name, false)))
}

// TransportServiceDirs returns the names of the directories of the packages
// that the generators create in gen/<transport>/<ServiceDir> for a service
// exposed over transport, one of "http", "grpc" and "jsonrpc".
func TransportServiceDirs(transport string) []string {
	if transport == "grpc" {
		return []string{"server", "client", "pb"}
	}
	return []string{"server", "client"}
}
