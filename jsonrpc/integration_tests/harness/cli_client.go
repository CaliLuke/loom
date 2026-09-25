package harness

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CLIClient runs the generated CLI of a module against a server. The CLI is
// built once, and each call runs the binary itself, not a go run wrapper, so
// ending a call's context kills the process that talks to the server.
type CLIClient struct {
	cliPath   string
	binDir    string
	binPath   string
	serverURL string
}

// ErrCLINotFound reports that a generated module has no CLI main package.
var ErrCLINotFound = errors.New("CLI source not found")

// NewCLIClient builds the generated CLI of the module in workDir and returns
// a client that runs it against serverURL. It returns an error wrapping
// ErrCLINotFound when the module has no CLI. Close removes the binary.
func NewCLIClient(workDir, serverURL string) (*CLIClient, error) {
	cliPath, err := findCLIDir(workDir)
	if err != nil {
		return nil, err
	}

	binDir, err := os.MkdirTemp("", "loom-jsonrpc-cli-")
	if err != nil {
		return nil, fmt.Errorf("create CLI build dir: %w", err)
	}
	// Build all files in the package so helpers generated alongside main.go
	// are included.
	binPath := filepath.Join(binDir, "cli")
	if output, err := goCommand(cliPath, "build", "-o", binPath, ".").CombinedOutput(); err != nil {
		err = fmt.Errorf("build CLI: %w\nOutput: %s", err, output)
		if removeErr := os.RemoveAll(binDir); removeErr != nil {
			err = errors.Join(err, fmt.Errorf("remove CLI build dir: %w", removeErr))
		}
		return nil, err
	}

	return &CLIClient{
		cliPath:   cliPath,
		binDir:    binDir,
		binPath:   binPath,
		serverURL: serverURL,
	}, nil
}

// Close removes the CLI binary. Calls must not run concurrently with or after
// Close. It may be called more than once.
func (c *CLIClient) Close() error {
	if err := os.RemoveAll(c.binDir); err != nil {
		return fmt.Errorf("remove CLI build dir: %w", err)
	}
	return nil
}

// CallMethod invokes a service method via the CLI. Ending ctx kills the CLI
// process.
func (c *CLIClient) CallMethod(ctx context.Context, service, method string, payload any) (jsontext.Value, error) {
	// Convert method name from snake_case to kebab-case for CLI
	cliMethod := strings.ReplaceAll(method, "_", "-")

	// URL must come before service and method for proper flag parsing
	args := []string{
		"-url", c.serverURL,
		"-verbose",
		service,
		cliMethod,
	}

	// Add payload if provided
	// Note: Generate methods don't take payload but the scenario might still
	// have params: {} which results in an empty map
	if payload != nil {
		// Skip empty maps for methods without payloads (like generate_*)
		if m, ok := payload.(map[string]any); ok && len(m) == 0 && strings.Contains(method, "generate_") {
			// Don't add any payload for empty maps on generate methods
		} else if str, ok := payload.(string); ok {
			// Check if payload is a simple string - use -p flag
			args = append(args, "-p", str)
		} else {
			// For any other payload, use --body flag
			payloadJSON, err := json.Marshal(payload)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal payload: %w", err)
			}

			args = append(args, "--body", string(payloadJSON))
		}
	}
	// If payload is nil, don't add any body argument - let the CLI handle it

	cmd := exec.CommandContext(ctx, c.binPath, args...)
	cmd.Dir = c.cliPath

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run command
	err := cmd.Run()

	// Check for errors
	if err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		// Include both stdout and stderr for debugging
		return nil, fmt.Errorf("CLI command failed: %s\nStderr: %s\nStdout: %s", errMsg, stderr.String(), stdout.String())
	}

	// Parse verbose output from stderr to get the raw JSON-RPC response
	verboseOutput := stderr.String()
	lines := strings.Split(verboseOutput, "\n")

	// Find the JSON-RPC response - it's the last line starting with {
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "{") {
			var resp struct {
				Result jsontext.Value `json:"result"`
				Error  *struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}

			if err := json.Unmarshal([]byte(line), &resp); err == nil && resp.Result != nil {
				return resp.Result, nil
			}
		}
	}

	// Fallback to stdout
	output := stdout.Bytes()
	if len(output) == 0 {
		return nil, nil
	}
	return jsontext.Value(output), nil
}

// CallJSONRPC makes a raw JSON-RPC call via the CLI
func (c *CLIClient) CallJSONRPC(ctx context.Context, request map[string]any) (jsontext.Value, error) {
	// For JSON-RPC, we need to use the jsonrpc command if available
	// Otherwise fall back to method call

	method, ok := request["method"].(string)
	if !ok {
		return nil, fmt.Errorf("no method in request")
	}

	// Extract service and method from JSON-RPC method name
	// Assuming format: service.method or just method
	parts := strings.Split(method, ".")
	service := "test" // default service
	methodName := method

	if len(parts) == 2 {
		service = parts[0]
		methodName = parts[1]
	}

	// Use the CLI to call the method
	return c.CallMethod(ctx, service, methodName, request["params"])
}

// CanHandle returns true if the CLI can handle this method
func (c *CLIClient) CanHandle(method string, params any) bool {
	// CLI can handle HTTP methods but not streaming
	// Check if it's a streaming method by looking for WebSocket or SSE in the method name
	if strings.Contains(method, "_ws") || strings.Contains(method, "_sse") {
		return false
	}

	// CLI doesn't handle notification methods (no response expected)
	if strings.Contains(method, "_notify") {
		return false
	}

	// CLI can handle methods with payloads
	return true
}

// findCLIDir returns the directory of the generated CLI main package.
func findCLIDir(workDir string) (string, error) {
	candidates := []string{
		filepath.Join(workDir, "cmd", "test_api-cli"),
		filepath.Join(workDir, "cmd", "test-cli"),
		filepath.Join(workDir, "cmd", "api-cli"),
	}
	for _, path := range candidates {
		if _, err := os.Stat(filepath.Join(path, "main.go")); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("%w in %s", ErrCLINotFound, workDir)
}
