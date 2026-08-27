// Package examplegen derives deterministic generators for stable codegen scopes.
package examplegen

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

const scopeVersion = "loom-codegen-example-scope-v1"

// ForScope derives an independent generator for parts when generator uses
// Loom's seeded faker. Custom generators retain their caller-defined behavior.
func ForScope(generator *expr.ExampleGenerator, parts ...string) *expr.ExampleGenerator {
	if generator == nil || generator.Randomizer == nil {
		return generator
	}
	faker, ok := generator.Randomizer.(*expr.FakerRandomizer)
	if !ok {
		return generator
	}
	sum := sha256.Sum256([]byte(strings.Join(append([]string{scopeVersion}, parts...), "\x00")))
	return expr.NewRandom(faker.Seed + ":" + hex.EncodeToString(sum[:]))
}
