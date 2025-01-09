package health

import (
	"runtime/debug"
)

// VersionInfo contains version information
type VersionInfo struct {
	Environment string
}

// NewVersionInfo creates a new version info instance
func NewVersionInfo(environment string) *VersionInfo {
	return &VersionInfo{
		Environment: environment,
	}
}

// GetVersionString returns a formatted version string
func (v *VersionInfo) GetVersionString() string {
	return getSHA()
}

// getSHA returns the commit SHA from build info
func getSHA() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	for _, s := range bi.Settings {
		if s.Key == "vcs.revision" {
			if s.Value == "" {
				return "unknown"
			}
			if len(s.Value) <= 7 {
				return s.Value
			}
			return s.Value[:7]
		}
	}

	return "unknown"
}

