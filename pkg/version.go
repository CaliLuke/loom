package loom

import (
	"fmt"
	"regexp"
	"runtime/debug"
	"strconv"
)

// Major is the Go module's major API namespace. It is independent of release tags.
const Major = 1

var (
	// Version format
	versionFormat = regexp.MustCompile(`v(\d+?)\.(\d+?)\.(\d+?)(?:-.+)?`)
)

// Version returns the Loom module version recorded by the Go toolchain.
// Local checkouts and unversioned replacements report "(devel)".
func Version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "(devel)"
	}
	return buildVersion(info)
}

func buildVersion(info *debug.BuildInfo) string {
	if info != nil {
		modules := append([]*debug.Module{&info.Main}, info.Deps...)
		for _, module := range modules {
			if module.Path != "github.com/CaliLuke/loom" {
				continue
			}
			if module.Replace != nil {
				module = module.Replace
			}
			if module.Version != "" {
				return module.Version
			}
			return "(devel)"
		}
	}
	return "(devel)"
}

// Compatible returns true if Major matches the major version of the given version string.
// It returns an error if the given string is not a valid version string.
func Compatible(v string) (bool, error) {
	matches := versionFormat.FindStringSubmatch(v)
	if len(matches) != 4 {
		return false, fmt.Errorf("invalid version string format %#v, %+v", v, matches)
	}
	mj, err := strconv.Atoi(matches[1])
	if err != nil {
		return false, fmt.Errorf("invalid major version number %#v, must be number, %w", matches[1], err)
	}
	return mj == Major, nil
}
