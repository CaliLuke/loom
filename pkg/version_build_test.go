package loom

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildVersion(t *testing.T) {
	for _, tc := range []struct {
		name string
		info *debug.BuildInfo
		want string
	}{
		{"installed alpha", &debug.BuildInfo{Main: debug.Module{Path: "github.com/CaliLuke/loom", Version: "v1.10.0-alpha.6"}}, "v1.10.0-alpha.6"},
		{"promoted installation", &debug.BuildInfo{Main: debug.Module{Path: "github.com/CaliLuke/loom", Version: "v1.10.0"}}, "v1.10.0"},
		{"dependency", &debug.BuildInfo{Deps: []*debug.Module{{Path: "github.com/CaliLuke/loom", Version: "v1.10.0"}}}, "v1.10.0"},
		{"local replacement", &debug.BuildInfo{Deps: []*debug.Module{{Path: "github.com/CaliLuke/loom", Version: "v1.10.0", Replace: &debug.Module{Path: "../loom"}}}}, "(devel)"},
		{"versioned replacement", &debug.BuildInfo{Deps: []*debug.Module{{Path: "github.com/CaliLuke/loom", Version: "v1.10.0", Replace: &debug.Module{Path: "example.com/fork", Version: "v1.11.0"}}}}, "v1.11.0"},
		{"local checkout", &debug.BuildInfo{Main: debug.Module{Path: "github.com/CaliLuke/loom", Version: "(devel)"}}, "(devel)"},
		{"no metadata", nil, "(devel)"},
	} {
		t.Run(tc.name, func(t *testing.T) { require.Equal(t, tc.want, buildVersion(tc.info)) })
	}
}
