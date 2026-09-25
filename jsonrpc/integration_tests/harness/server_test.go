package harness

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeServerMain is a server that behaves like the generated ones at startup:
// it logs that it listens before it binds its port. It exits instead of
// binding when it is built with exitAtStart set, as a generated server does
// when its port is taken.
const fakeServerMain = `package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
)

const exitAtStart = %t

func main() {
	port := flag.Int("http-port", 0, "HTTP port")
	flag.Parse()
	addr := fmt.Sprintf("localhost:%%d", *port)
	fmt.Printf("HTTP server listening on %%q\n", addr)
	if exitAtStart {
		fmt.Printf("exiting (listen tcp %%s: bind: address already in use)\n", addr)
		os.Exit(1)
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "fake")
	})
	if err := http.ListenAndServe(addr, handler); err != nil {
		fmt.Printf("exiting (%%v)\n", err)
		os.Exit(1)
	}
}
`

func TestStartServerFailsWhenServerExits(t *testing.T) {
	server, err := StartServer(t.Context(), writeFakeServer(t, true), 0)
	if err == nil {
		require.NoError(t, server.Stop())
	}
	require.ErrorContains(t, err, "server exited before accepting connections")
}

func TestStopStopsServerProcess(t *testing.T) {
	server, err := StartServer(context.Background(), writeFakeServer(t, false), 0)
	require.NoError(t, err)

	resp, err := http.Get(server.URL())
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, "fake", string(body))

	require.NoError(t, server.Stop())
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", server.port))
	if err == nil {
		require.NoError(t, conn.Close())
	}
	require.Error(t, err, "the server process still accepts connections after Stop")
	require.NoError(t, server.Stop(), "Stop must be idempotent")
}

// writeFakeServer writes a module with fakeServerMain at the path StartServer
// looks for and returns the module directory.
func writeFakeServer(t *testing.T, exitAtStart bool) string {
	t.Helper()
	workDir := t.TempDir()
	mainDir := filepath.Join(workDir, "cmd", "test_api")
	require.NoError(t, os.MkdirAll(mainDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "go.mod"), []byte("module fakeserver\n\ngo 1.27\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(mainDir, "main.go"), []byte(fmt.Sprintf(fakeServerMain, exitAtStart)), 0o600))
	return workDir
}
