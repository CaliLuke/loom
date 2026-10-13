package generator

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestInterceptorPublicNameReadinessCompile(t *testing.T) {
	for _, mode := range []string{"method", "info", "ordinary"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			renderDesignModule(t, dir, "example.com/publicnames", func() {
				testdata.InterceptorPublicNamesDSL(mode)
			})
			for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}} {
				output, err := testingx.RunCmd(dir, "go", args...)
				require.NoError(t, err, output)
			}
		})
	}
}

func TestGeneratedInterceptorPublicRoles(t *testing.T) {
	for _, streaming := range []bool{false, true} {
		name := "unary"
		if streaming {
			name = "stream"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			renderDesignModule(t, dir, "example.com/roles", func() {
				testdata.InterceptorPublicRolesDSL(streaming, streaming, "both", []string{"AuditInfo", "AuditPayload", "AuditResult", "AuditStreamingPayload", "AuditStreamingPayload2", "AuditStreamingResult"})
			})
			for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}} {
				output, err := testingx.RunCmd(dir, "go", args...)
				require.NoError(t, err, output)
			}
		})
	}
}
