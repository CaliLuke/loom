package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestSelectedProblemBodyConstruction(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		mapCode  bool
		custom   bool
		empty    bool
		shaped   bool
		cookie   bool
		override bool
	}{
		{name: "full problem"},
		{name: "type", body: "type"},
		{name: "title", body: "title"},
		{name: "status", body: "status"},
		{name: "detail", body: "detail"},
		{name: "instance", body: "instance"},
		{name: "code", body: "code"},
		{name: "retry hint", body: "retry_hint"},
		{name: "code header", mapCode: true},
		{name: "custom detail", body: "detail", custom: true},
		{name: "empty", empty: true},
		{name: "explicit detail object", shaped: true},
		{name: "code cookie", cookie: true},
		{name: "full overrides", override: true},
		{name: "type override", body: "type", override: true},
		{name: "title override", body: "title", override: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				Service("problems", func() {
					Method("selected", func() {
						if tc.custom {
							Error("oops", func() {
								Attribute("detail", String)
								Required("detail")
							})
						} else {
							Error("oops", func() {
								if tc.override {
									Meta("http:problem:type", "https://example.com/problem")
									Meta("http:problem:title", "Explicit title")
								}
							})
						}
						HTTP(func() {
							GET("/selected")
							Response("oops", StatusBadRequest, func() {
								if tc.body != "" {
									Body(tc.body)
								}
								if tc.empty {
									Body(Empty)
								}
								if tc.shaped {
									Body(func() { Attribute("detail") })
								}
								if tc.cookie {
									Cookie("code:error_code")
								}
								if tc.mapCode {
									Header("code:X-Code")
								}
							})
						})
					})
				})
			})
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/problems", root)
			runGoCommand(t, dir, "mod", "edit", "-require=github.com/CaliLuke/loom@v1.10.0-alpha.9.0.20261012201303-8c7d8d711b9e")
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "build", "./...")
			if !tc.custom {
				harness := fmt.Sprintf(selectedProblemWireHarness, tc.body, tc.mapCode, tc.cookie, tc.empty, tc.shaped, tc.override)
				require.NoError(t, os.WriteFile(filepath.Join(dir, "problem_test.go"), []byte(harness), 0o600))
				runGoCommand(t, dir, "mod", "tidy")
				runGoCommand(t, dir, "test", "-count=1", "./...")
			}
		})
	}
}
