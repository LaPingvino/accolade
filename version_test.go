// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"bytes"
	"runtime/debug"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
)

// accolade -version says the version it was given, from the packaged
// metadata or the build stamp, and never "dev" then.
func TestVersionFlagNeverDevWhenGiven(t *testing.T) {
	keepMeta, keepVersion := appMetadata, version
	t.Cleanup(func() { appMetadata, version = keepMeta, keepVersion })

	packaged := fyne.AppMetadata{ID: AppID, Name: "Accolade", Version: "0.14.1", Release: true}
	for _, c := range []struct {
		name    string
		meta    fyne.AppMetadata
		stamped string
		want    string
	}{
		{"packaged", packaged, "", "0.14.1"},
		{"packaged and stamped", packaged, "0.14.1", "0.14.1"},
		{"stamped only", fyne.AppMetadata{Version: "0.0.1"}, "0.14.1", "0.14.1"},
		{"checkout metadata, stamped", fyne.AppMetadata{ID: AppID, Version: "0.13.0"}, "0.14.1-3-gabcdef0", "0.14.1-3-gabcdef0"},
	} {
		appMetadata = func() fyne.AppMetadata { return c.meta }
		version = c.stamped
		for _, flag := range []string{"-version", "--version"} {
			var out bytes.Buffer
			if code := runCLI([]string{flag}, &out); code != 0 {
				t.Fatalf("%s %s: exit %d", c.name, flag, code)
			}
			first, _, _ := strings.Cut(out.String(), "\n")
			if !strings.HasPrefix(first, "Accolade "+c.want) || strings.Contains(strings.ToLower(out.String()), "dev") {
				t.Errorf("%s %s: %q, want Accolade %s and no dev", c.name, flag, out.String(), c.want)
			}
			if !strings.Contains(out.String(), "\nLexington ") {
				t.Errorf("%s %s: %q names no Lexington", c.name, flag, out.String())
			}
		}
	}
}

// go install …@vX: the module version, when nothing else is given.
func TestResolveVersionFallbacks(t *testing.T) {
	none := fyne.AppMetadata{Version: "0.0.1"}
	if v := resolveVersion(none, "", &debug.BuildInfo{Main: debug.Module{Version: "v0.14.1"}}); v != "v0.14.1" {
		t.Errorf("go install: %q", v)
	}
	bi := &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}, Settings: []debug.BuildSetting{
		{Key: "vcs.revision", Value: "0123456789abcdef"}, {Key: "vcs.time", Value: "2026-11-27T10:00:00Z"}}}
	if v := resolveVersion(none, "", bi); v != "dev 0123456 (2026-11-27)" {
		t.Errorf("checkout: %q", v)
	}
	if v := resolveVersion(none, "", nil); v != "dev" {
		t.Errorf("nothing: %q", v)
	}
}
