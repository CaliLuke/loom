package designfingerprint

import (
	"fmt"
	"reflect"

	"golang.org/x/mod/module"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

type (
	// Design identifies the complete ordered input of one evaluated DSL session.
	// Roots must be captured before generator preparation mutates their state.
	Design struct {
		// Package is the canonical import path of the originating design package.
		Package string
		// Roots contains every evaluated root in the order supplied to generators.
		Roots []eval.Root
	}

	// Manifest records the generator dependency and evaluated-design provenance.
	// Generation, staged installation and vet use this same persisted contract.
	Manifest struct {
		// LoomVersion identifies the dependency that generated the tree.
		LoomVersion string `json:"loom_version"`
		// DesignPackage identifies the design package that owns this tree.
		DesignPackage string `json:"design_package"`
		// DesignDigest fingerprints the evaluated input and generation context.
		DesignDigest string `json:"design_digest"`
	}
)

// Validate checks explicit origin and root identities and returns the single
// core root used by Loom's semantic diagnostics. It never consults global state.
func (d Design) Validate() (*expr.RootExpr, error) {
	if err := module.CheckImportPath(d.Package); err != nil {
		return nil, fmt.Errorf("invalid design package: %w", err)
	}
	var core *expr.RootExpr
	names := make(map[string]bool, len(d.Roots))
	for i, root := range d.Roots {
		if root == nil || isNilRoot(root) {
			return nil, fmt.Errorf("evaluated root %d is nil", i)
		}
		name := root.EvalName()
		if name == "" {
			return nil, fmt.Errorf("evaluated root %d has no name", i)
		}
		if names[name] {
			return nil, fmt.Errorf("duplicate evaluated root %q", name)
		}
		names[name] = true
		if r, ok := root.(*expr.RootExpr); ok {
			core = r
		}
	}
	if core == nil {
		return nil, fmt.Errorf("evaluated design has no core root")
	}
	return core, nil
}

// ValidateProvenance rejects unattributed or incomplete generation manifests.
// Version validity remains with the consumer's version-skew or staging policy.
func (m Manifest) ValidateProvenance() error {
	if err := module.CheckImportPath(m.DesignPackage); err != nil {
		return fmt.Errorf("invalid design package provenance: %w", err)
	}
	if m.DesignDigest == "" {
		return fmt.Errorf("no design digest")
	}
	return nil
}

func isNilRoot(root eval.Root) bool {
	value := reflect.ValueOf(root)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
