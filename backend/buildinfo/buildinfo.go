// Package buildinfo reports which build of the backend is running.
//
// Release builds set the values with -ldflags, e.g.
//
//	go build -ldflags "-X github.com/imadMansour85/Jetyard/backend/buildinfo.Version=v0.1.0 \
//	  -X github.com/imadMansour85/Jetyard/backend/buildinfo.Commit=$(git rev-parse HEAD) \
//	  -X github.com/imadMansour85/Jetyard/backend/buildinfo.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
//
// A plain `go build` inside the git checkout still reports the commit and
// commit time, from the VCS info the Go toolchain embeds.
package buildinfo

import (
	"runtime"
	"runtime/debug"
)

// Set with -ldflags -X at build time.
var (
	Version   = "dev"
	Commit    = ""
	BuildTime = ""
)

// Info is what GET /version returns.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit,omitempty"`
	BuildTime string `json:"build_time,omitempty"`
	Modified  bool   `json:"modified,omitempty"` // built from a dirty working tree
	GoVersion string `json:"go_version"`
}

// Get returns the build info, filling gaps from the embedded VCS info.
func Get() Info {
	info := Info{
		Version:   Version,
		Commit:    Commit,
		BuildTime: BuildTime,
		GoVersion: runtime.Version(),
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				if info.Commit == "" {
					info.Commit = s.Value
				}
			case "vcs.time":
				if info.BuildTime == "" {
					info.BuildTime = s.Value
				}
			case "vcs.modified":
				info.Modified = s.Value == "true"
			}
		}
	}
	return info
}
