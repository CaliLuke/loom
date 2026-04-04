package harness

import (
	"bufio"
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

type (
	// Server manages a test server process.
	Server struct {
		cmd     *exec.Cmd
		port    int
		logFile *os.File
	}
)

// StartServer starts a fixture server from the given working directory.
func StartServer(ctx context.Context, workDir string, port int) (*Server, error) {
	if port == 0 {
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			return nil, fmt.Errorf("find free port: %w", err)
		}
		port = listener.Addr().(*net.TCPAddr).Port
		if err := listener.Close(); err != nil {
			return nil, fmt.Errorf("close free-port listener: %w", err)
		}
	}

	logPath := filepath.Join(workDir, fmt.Sprintf("server-%d.log", port))
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("create log file: %w", err)
	}

	serverPath := filepath.Join(workDir, "cmd", "ticktock", "main.go")
	if _, err := os.Stat(serverPath); err != nil {
		if closeErr := logFile.Close(); closeErr != nil {
			return nil, fmt.Errorf("close log file after missing server: %w", closeErr)
		}
		return nil, fmt.Errorf("server main.go not found at %s: %w", serverPath, err)
	}

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = workDir
	tidyCmd.Env = append(os.Environ(), "GO111MODULE=on", "GOWORK=off")
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		if closeErr := logFile.Close(); closeErr != nil {
			return nil, fmt.Errorf("close log file after tidy failure: %w", closeErr)
		}
		return nil, fmt.Errorf("go mod tidy failed: %w\n%s", err, output)
	}

	cmd := exec.CommandContext(ctx, "go", "run", ".", "--http-port", fmt.Sprintf("%d", port))
	cmd.Dir = filepath.Dir(serverPath)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Env = append(os.Environ(), "GO111MODULE=on", "GOWORK=off")
	if err := cmd.Start(); err != nil {
		if closeErr := logFile.Close(); closeErr != nil {
			return nil, fmt.Errorf("close log file after start failure: %w", closeErr)
		}
		return nil, fmt.Errorf("start server: %w", err)
	}

	server := &Server{
		cmd:     cmd,
		port:    port,
		logFile: logFile,
	}
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
	return fmt.Sprintf("http://localhost:%d", s.port)
}

// Stop terminates the server process.
func (s *Server) Stop() error {
	var errs []error
	if s.cmd != nil && s.cmd.Process != nil {
		if err := s.cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			errs = append(errs, fmt.Errorf("kill server process: %w", err))
		}
		if err := s.cmd.Wait(); !isExpectedStopError(err) {
			errs = append(errs, fmt.Errorf("wait for server process: %w", err))
		}
	}
	if s.logFile != nil {
		if err := s.logFile.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close log file: %w", err))
		}
	}
	return errors.Join(errs...)
}

func (s *Server) waitForReady(ctx context.Context) error {
	logScanner := bufio.NewScanner(s.logFile)
	ready := make(chan struct{}, 1)
	errc := make(chan error, 1)

	go func() {
		for logScanner.Scan() {
			line := logScanner.Text()
			if strings.Contains(line, "HTTP server listening") || strings.Contains(line, fmt.Sprintf(":%d", s.port)) {
				ready <- struct{}{}
				return
			}
			if strings.Contains(line, "error") || strings.Contains(line, "failed") {
				errc <- fmt.Errorf("server error: %s", line)
				return
			}
		}
		if err := logScanner.Err(); err != nil {
			errc <- fmt.Errorf("scan log: %w", err)
		}
	}()

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", s.port))
				if err != nil {
					continue
				}
				if err := conn.Close(); err != nil {
					errc <- fmt.Errorf("close readiness probe connection: %w", err)
					return
				}
				ready <- struct{}{}
				return
			}
		}
	}()

	select {
	case <-ready:
		return nil
	case err := <-errc:
		return err
	case <-time.After(60 * time.Second):
		return fmt.Errorf("server failed to start within 60 seconds")
	case <-ctx.Done():
		return ctx.Err()
	}
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
