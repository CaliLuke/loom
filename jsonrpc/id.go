package jsonrpc

import (
	"crypto/rand"
	"fmt"
	"io"

	"github.com/google/uuid"
)

// NewRequestID returns a random RFC 4122 UUID for a JSON-RPC request.
func NewRequestID() (string, error) {
	return newRequestID(rand.Reader)
}

func newRequestID(reader io.Reader) (string, error) {
	id, err := uuid.NewRandomFromReader(reader)
	if err != nil {
		return "", fmt.Errorf("generate JSON-RPC request ID: %w", err)
	}
	return id.String(), nil
}
