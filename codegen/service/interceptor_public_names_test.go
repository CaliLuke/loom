package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testdata"
)

func TestInterceptorPublicNameReadiness(t *testing.T) {
	for _, mode := range []string{"method", "info", "ordinary"} {
		t.Run(mode, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				testdata.InterceptorPublicNamesDSL(mode)
			})
			data := NewServicesData(root).Get("svc")
			occupied := map[string]bool{"AuditInfo": true, "AuditPayload": true, "AuditResult": true}
			for _, typ := range data.userTypes {
				require.False(t, occupied[typ.VarName], "authored %s emitted as %s collides with public interceptor declaration", typ.Name, typ.VarName)
			}
		})
	}
}
