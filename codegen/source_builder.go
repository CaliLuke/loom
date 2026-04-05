package codegen

import "strings"

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
