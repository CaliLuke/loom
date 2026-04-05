package codegen

import (
	"fmt"
	"strings"
)

type sourceBuilder struct {
	parts []string
}

func (b *sourceBuilder) Add(s string) {
	b.parts = append(b.parts, s)
}

func (b *sourceBuilder) String() string {
	return strings.Join(b.parts, "")
}

func (b *sourceBuilder) Write(p []byte) (int, error) {
	b.parts = append(b.parts, string(p))
	return len(p), nil
}

func (b *sourceBuilder) WriteString(s string) (int, error) {
	b.parts = append(b.parts, s)
	return len(s), nil
}

func (b *sourceBuilder) Addf(format string, args ...any) {
	b.parts = append(b.parts, fmt.Sprintf(format, args...))
}
