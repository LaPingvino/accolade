package main

import (
	"runtime/debug"
	"strings"

	"fyne.io/fyne/v2"
)

const lexingtonModule = "github.com/LaPingvino/lexington"

// version is set when building (-ldflags "-X main.version=…"): the release
// builds, the Arch package and the web version do. A packaged release's
// Fyne metadata (fyne package --app-version) comes first.
var version string

// appMetadata is the Fyne app's metadata, which fyne package embeds (on
// the command line, main makes an app for -version to read it from).
var appMetadata = func() fyne.AppMetadata { return fyne.CurrentApp().Metadata() }

// appVersion is Accolade's version: a packaged release's, else as stamped
// when building, else the module version go install records, else the
// commit and date Go recorded, else "dev".
func appVersion() string {
	bi, _ := debug.ReadBuildInfo()
	return resolveVersion(appMetadata(), version, bi)
}

func resolveVersion(md fyne.AppMetadata, stamped string, bi *debug.BuildInfo) string {
	// without fyne package --release, the metadata is Fyne's default (0.0.1)
	// or, in a checkout, FyneApp.toml's: the last release, not this build
	if md.Release && md.ID != "" && md.Version != "" {
		return md.Version
	}
	if stamped != "" {
		return stamped
	}
	if bi == nil {
		return "dev"
	}
	if v := bi.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	if rev, at := vcsInfo(bi); len(rev) >= 7 {
		if len(at) >= 10 {
			return "dev " + rev[:7] + " (" + at[:10] + ")"
		}
		return "dev " + rev[:7]
	}
	return "dev"
}

// vcsInfo is the commit and its time Go recorded when building.
func vcsInfo(bi *debug.BuildInfo) (rev, at string) {
	if bi == nil {
		return "", ""
	}
	for _, kv := range bi.Settings {
		switch kv.Key {
		case "vcs.revision":
			rev = kv.Value
		case "vcs.time":
			at = kv.Value
		}
	}
	return rev, at
}

// wantsVersion is whether the options ask for -version.
func wantsVersion(args []string) bool {
	for _, a := range args {
		if a == "-version" || a == "--version" {
			return true
		}
	}
	return false
}

// versionLine is what accolade -version prints.
func versionLine() string {
	bi, _ := debug.ReadBuildInfo()
	v := resolveVersion(appMetadata(), version, bi)
	line := "Accolade " + v
	if rev, at := vcsInfo(bi); len(rev) >= 7 && !strings.HasPrefix(v, "dev") {
		line += " (commit " + rev[:7]
		if len(at) >= 10 {
			line += ", " + at[:10]
		}
		line += ")"
	}
	return line + "\nLexington " + moduleVersion(lexingtonModule) + " (built in)"
}
