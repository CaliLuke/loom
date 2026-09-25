package harness

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type (
	// Server manages a test server process. The process is the compiled
	// server binary itself, not a go run wrapper, so stopping the Server
	// stops the process that holds the port.
	Server struct {
		cmd     *exec.Cmd
		port    int
		logFile *os.File
		binDir  string
		// exited is closed once the process has exited; waitErr is its
		// exit status.
		exited  chan struct{}
		waitErr error
	}
)

// StartServer builds the fixture server in workDir, starts it on port, or on
// a free loopback port when port is 0, and waits until it accepts
// connections. It fails when the process exits first, for example because
// its port is taken.
func StartServer(ctx context.Context, workDir string, port int) (*Server, error) {
	serverDir := filepath.Join(workDir, "cmd", "ticktock")
	if _, err := os.Stat(filepath.Join(serverDir, "main.go")); err != nil {
		return nil, fmt.Errorf("server main.go not found in %s: %w", serverDir, err)
	}

	if output, err := goCommand(workDir, "mod", "tidy").CombinedOutput(); err != nil {
		return nil, fmt.Errorf("go mod tidy failed: %w\n%s", err, output)
	}

	binDir, err := os.MkdirTemp("", "loom-http-server-")
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

	binPath := filepath.Join(binDir, "server")
	if output, err := goCommand(serverDir, "build", "-o", binPath, ".").CombinedOutput(); err != nil {
		return nil, fmt.Errorf("build server: %w\n%s", err, output)
	}

	// Pick the port only now that the binary is built, on the loopback
	// address the fixture server binds, so the kernel does not hand out a
	// port another process holds there.
	if port == 0 {
		if port, err = freeLoopbackPort(); err != nil {
			return nil, fmt.Errorf("find free port: %w", err)
		}
	}
	server.port = port

	logPath := filepath.Join(workDir, fmt.Sprintf("server-%d.log", port))
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("create log file: %w", err)
	}
	server.logFile = logFile

	cmd := exec.CommandContext(ctx, binPath, "--http-port", fmt.Sprintf("%d", port))
	cmd.Dir = serverDir
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		if closeErr := logFile.Close(); closeErr != nil {
			return nil, fmt.Errorf("close log file after start failure: %w", closeErr)
		}
		return nil, fmt.Errorf("start server: %w", err)
	}
	started = true
	server.cmd = cmd
	go func() {
		server.waitErr = cmd.Wait()
		close(server.exited)
	}()

	if err := server.waitForReady(ctx); err != nil {
		content := readLogFile(logPath)
		if stopErr := server.Stop(); stopErr != nil {
			return nil, fmt.Errorf("%w\nfailed to stop server after startup failure: %w\nServer log:\n%s", err, stopErr, content)
		}
		return nil, fmt.Errorf("%w\nServer log:\n%s", err, content)
	}
	return server, nil
}

// URL returns the server base URL.
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
		if !isExpectedStopError(s.waitErr) {
			errs = append(errs, fmt.Errorf("wait for server process: %w", s.waitErr))
		}
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

// goCommand returns a go command run in dir in module mode, outside any
// workspace.
func goCommand(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GO111MODULE=on", "GOWORK=off")
	return cmd
}

// freeLoopbackPort returns a TCP port that is free on the loopback address
// the fixture server binds: "localhost" resolves to 127.0.0.1 for it.
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

func readLogFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("read log file: %v", err)
	}
	return string(data)
}

func isExpectedStopError(err error) bool {
	if err == nil {
		return true
	}
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr)
}

// waitForReady waits until the server accepts connections on its port. It
// fails when the process exits first, as it does when it cannot bind the
// port. The fixture server logs that it listens before it binds, so only a
// successful connection proves readiness.
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
			conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", s.port))
			if err != nil {
				continue
			}
			if err := conn.Close(); err != nil {
				return fmt.Errorf("close readiness probe connection: %w", err)
			}
			return nil
		}
	}
}
