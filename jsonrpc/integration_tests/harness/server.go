package harness

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Server manages a test server process. The process is the compiled server
// binary itself, not a go run wrapper, so stopping the Server stops the
// process that holds the ports.
type Server struct {
	cmd      *exec.Cmd
	port     int
	grpcPort int
	logFile  *os.File
	binDir   string
	// exited is closed once the process has exited; waitErr is its exit
	// status.
	exited  chan struct{}
	waitErr error
}

// StartServer builds the server of the generated module in workDir, starts it
// on port, or on a free loopback port when port is 0, and waits until it
// accepts connections. It fails when the process exits first, for example
// because its port is taken.
func StartServer(ctx context.Context, workDir string, port int) (*Server, error) {
	serverDir, err := findServerDir(workDir)
	if err != nil {
		return nil, err
	}

	// Ensure the fixture module is consistent before building.
	if output, err := goCommand(workDir, "mod", "tidy").CombinedOutput(); err != nil {
		return nil, fmt.Errorf("go mod tidy failed: %w\nOutput: %s", err, output)
	}

	binDir, err := os.MkdirTemp("", "loom-jsonrpc-server-")
	if err != nil {
		return nil, fmt.Errorf("create server build dir: %w", err)
	}
	server := &Server{binDir: binDir, exited: make(chan struct{})}
	started := false
	defer func() {
		if !started {
			if err := os.RemoveAll(binDir); err != nil {
				fmt.Fprintf(os.Stderr, "remove server build dir: %v\n", err)
			}
		}
	}()

	// Build all files in the package so helpers generated alongside main.go
	// are included.
	binPath := filepath.Join(binDir, "server")
	if output, err := goCommand(serverDir, "build", "-o", binPath, ".").CombinedOutput(); err != nil {
		return nil, fmt.Errorf("build server: %w\nOutput: %s", err, output)
	}

	// Pick the ports only now that the binary is built, on the loopback
	// address the generated server binds, so the kernel does not hand out
	// a port another process holds there.
	if port == 0 {
		if port, err = freeLoopbackPort(); err != nil {
			return nil, fmt.Errorf("find free port: %w", err)
		}
	}
	server.port = port
	args := []string{"--http-port", fmt.Sprintf("%d", port)}
	if _, err := os.Stat(filepath.Join(workDir, "gen", "grpc")); err == nil {
		if server.grpcPort, err = freeLoopbackPort(); err != nil {
			return nil, fmt.Errorf("find free gRPC port: %w", err)
		}
		args = append(args, "--grpc-port", fmt.Sprintf("%d", server.grpcPort))
	}

	logPath := filepath.Join(workDir, fmt.Sprintf("server-%d.log", port))
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}
	server.logFile = logFile

	cmd := exec.CommandContext(ctx, binPath, args...)
	cmd.Dir = serverDir
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		if closeErr := logFile.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close log file: %w", closeErr))
		}
		return nil, fmt.Errorf("failed to start server: %w", err)
	}
	started = true
	server.cmd = cmd
	go func() {
		server.waitErr = cmd.Wait()
		close(server.exited)
	}()

	if err := server.waitForReady(ctx); err != nil {
		logContent := readLog(logPath)
		if stopErr := server.Stop(); stopErr != nil {
			err = errors.Join(err, fmt.Errorf("stop server: %w", stopErr))
		}
		return nil, fmt.Errorf("%w\nServer log:\n%s", err, logContent)
	}

	return server, nil
}

// URL returns the server's base URL
func (s *Server) URL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", s.port)
}

// Stop kills the server process, waits for it to exit, and removes its
// binary. It may be called more than once.
func (s *Server) Stop() error {
	var errs []error
	if s.cmd != nil && s.cmd.Process != nil {
		if err := s.cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			errs = append(errs, fmt.Errorf("kill server process: %w", err))
		}
		<-s.exited
	}
	if s.logFile != nil {
		if err := s.logFile.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			errs = append(errs, fmt.Errorf("close log file: %w", err))
		}
	}
	if err := os.RemoveAll(s.binDir); err != nil {
		errs = append(errs, fmt.Errorf("remove server build dir: %w", err))
	}
	return errors.Join(errs...)
}

// findServerDir returns the directory of the generated server main package.
// It tries the historical generated locations first, then falls back to any
// fixture app under cmd/*/main.go, excluding CLI entrypoints.
func findServerDir(workDir string) (string, error) {
	candidates := []string{
		filepath.Join(workDir, "cmd", "test_api", "main.go"),
		filepath.Join(workDir, "cmd", "test", "main.go"),
		filepath.Join(workDir, "cmd", "api", "main.go"),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return filepath.Dir(path), nil
		}
	}
	matches, err := filepath.Glob(filepath.Join(workDir, "cmd", "*", "main.go"))
	if err != nil {
		return "", fmt.Errorf("failed to scan cmd directories: %w", err)
	}
	for _, match := range matches {
		if !strings.Contains(match, "-cli") {
			return filepath.Dir(match), nil
		}
	}
	return "", fmt.Errorf("server main.go not found in any expected location")
}

// goCommand returns a go command run in dir in module mode, outside any
// workspace.
func goCommand(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GO111MODULE=on", "GOWORK=off")
	return cmd
}

// freeLoopbackPort returns a TCP port that is free on the loopback address
// the generated servers bind: "localhost" resolves to 127.0.0.1 for them.
func freeLoopbackPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		return 0, fmt.Errorf("close port probe listener: %w", err)
	}
	return port, nil
}

// readLog returns the content of the server log for diagnostics.
func readLog(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("read server log: %v", err)
	}
	return string(data)
}

// waitForReady waits until the server accepts connections on its HTTP port
// and, when it serves gRPC, on its gRPC port. It fails when the process exits
// first, as it does when it cannot bind a port. The generated servers log
// that they listen before they bind, so only a successful connection proves
// readiness.
func (s *Server) waitForReady(ctx context.Context) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(60 * time.Second)
	for {
		select {
		case <-s.exited:
			if s.waitErr == nil {
				return errors.New("server exited before accepting connections")
			}
			return fmt.Errorf("server exited before accepting connections: %w", s.waitErr)
		case <-timeout:
			return fmt.Errorf("server failed to start within 60 seconds")
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if s.accepts(s.port) && (s.grpcPort == 0 || s.accepts(s.grpcPort)) {
				return nil
			}
		}
	}
}

// accepts reports whether a TCP connection to port on 127.0.0.1 succeeds.
func (s *Server) accepts(port int) bool {
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	return conn.Close() == nil
}
