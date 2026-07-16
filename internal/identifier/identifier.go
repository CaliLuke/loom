// Package identifier generates opaque identifiers from checked entropy.
package identifier

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

// ErrEntropy identifies failures to obtain all bytes required for an identifier.
var ErrEntropy = errors.New("loom identifier entropy failure")

// Base64 reads size bytes from reader and returns their unpadded URL-safe base64 encoding.
func Base64(reader io.Reader, size int) (string, error) {
	b, err := read(reader, size)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Hex reads size bytes from reader and returns their lowercase hexadecimal encoding.
func Hex(reader io.Reader, size int) (string, error) {
	b, err := read(reader, size)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// MustBase64 returns an identifier made from operating-system entropy.
// It panics if the operating system cannot provide all required bytes.
func MustBase64(size int) string {
	return must(rand.Reader, size, Base64)
}

// MustHex returns an identifier made from operating-system entropy.
// It panics if the operating system cannot provide all required bytes.
func MustHex(size int) string {
	return must(rand.Reader, size, Hex)
}

func read(reader io.Reader, size int) ([]byte, error) {
	b := make([]byte, size)
	if _, err := io.ReadFull(reader, b); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEntropy, err)
	}
	return b, nil
}

func must(reader io.Reader, size int, generate func(io.Reader, int) (string, error)) string {
	id, err := generate(reader, size)
	if err != nil {
		panic(fmt.Errorf("generate identifier: %w", err))
	}
	return id
}
