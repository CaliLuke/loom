package harness

import (
	"bufio"
	"context"
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

	downloadCmd := exec.Command("go", "mod", "download")
	downloadCmd.Dir = workDir
	downloadCmd.Env = append(os.Environ(), "GO111MODULE=on", "GOWORK=off")
	if output, err := downloadCmd.CombinedOutput(); err != nil {
		if closeErr := logFile.Close(); closeErr != nil {
			return nil, fmt.Errorf("close log file after download failure: %w", closeErr)
		}
		return nil, fmt.Errorf("go mod download failed: %w\n%s", err, output)
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
		_ = server.Stop()
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
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_, _ = s.cmd.Process.Wait()
	}
	if s.logFile != nil {
		return s.logFile.Close()
	}
	return nil
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
				_ = conn.Close()
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
