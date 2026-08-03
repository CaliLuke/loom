package jsonrpc

import (
	"crypto/rand"
	"fmt"
	"io"
	"uuid"
)

// NewRequestID returns a random RFC 9562 version 4 UUID for a JSON-RPC request.
func NewRequestID() (string, error) {
	return newRequestID(rand.Reader)
}

func newRequestID(reader io.Reader) (string, error) {
	var id uuid.UUID
	if _, err := io.ReadFull(reader, id[:]); err != nil {
		return "", fmt.Errorf("generate JSON-RPC request ID: %w", err)
	}
	id[6] = id[6]&0x0f | 0x40
	id[8] = id[8]&0x3f | 0x80
	return id.String(), nil
}
