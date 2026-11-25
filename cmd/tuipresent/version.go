package main

import (
	"fmt"
	"io"
	"runtime"
	"runtime/debug"
	"strings"
)

// Release builds set these with
//
//	-ldflags "-X main.version=v1.0.0-rc1 -X main.commit=abc1234 -X main.date=2026-01-02T03:04:05Z"
//
// Left empty, they come from the build info Go embeds (vcs.revision and
// vcs.time with go build in a git checkout, or the module version with
// go install pkg@version).
var (
	version string
	commit  string
	date    string
)

// versionString is the one-line output of -version.
func versionString() string {
	return formatVersion(version, commit, date, runtime.Version(), debug.ReadBuildInfo)
}

func formatVersion(ver, rev, built, goVer string, read func() (*debug.BuildInfo, bool)) string {
	dirty := false
	injected := rev != ""
	if bi, ok := read(); ok {
		if ver == "" && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			ver = bi.Main.Version
		}
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				if rev == "" {
					rev = s.Value
				}
			case "vcs.time":
				if built == "" {
					built = s.Value
				}
			case "vcs.modified":
				dirty = s.Value == "true" && !injected
			}
		}
	}
	if ver == "" {
		ver = "dev"
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}
	if rev != "" && dirty {
		rev += "-dirty"
	}
	var parts []string
	if rev != "" {
		parts = append(parts, "commit "+rev)
	}
	if built != "" {
		parts = append(parts, "built "+built)
	}
	parts = append(parts, goVer)
	return "tuipresent " + ver + " (" + strings.Join(parts, ", ") + ")"
}

func printVersion(w io.Writer) {
	fmt.Fprintln(w, versionString())
}
