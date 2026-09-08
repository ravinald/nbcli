// Package version exposes build metadata baked in via -ldflags at link time.
//
// The Go linker overrides these vars when the binary is built with:
//
//	-ldflags "-X github.com/ravinald/nbcli/internal/version.Version=v0.1.0 ..."
//
// At `go run` time the defaults below are used so the binary still reports
// something coherent.
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

// Build metadata, overridden via -ldflags. Vars (not consts) so the linker
// can patch them. Keep names stable — the Makefile references them.
var (
	// Version is the semver tag (e.g. "v0.1.0") or "dev" when unset.
	Version = "dev"

	// Commit is the short git SHA the binary was built from.
	Commit = "none"

	// Date is the ISO-8601 build timestamp.
	Date = "unknown"
)

// Info is the snapshot of build metadata for display or JSON output.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// Get returns the current build info. When -ldflags weren't set (e.g. `go run`)
// it falls back to runtime/debug so `nbcli version` still says something useful.
func Get() Info {
	info := Info{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
	if info.Commit != "none" {
		return info
	}
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return info
	}
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) >= 7 {
				info.Commit = s.Value[:7]
			}
		case "vcs.time":
			info.Date = s.Value
		}
	}
	return info
}

// String returns a single-line human summary suitable for `--version`.
func (i Info) String() string {
	return fmt.Sprintf("nbcli %s (commit %s, built %s, %s %s/%s)",
		i.Version, i.Commit, i.Date, i.GoVersion, i.OS, i.Arch)
}
