//go:build !race

// Real timers (realPaths), so built without -race like background_real_test.go.

package main

import (
	"strings"
	"testing"
	"time"
)

// The counts follow the marks, which wait for typing to pause: never per
// key.
func TestProductionFollowsAfterPause(t *testing.T) {
	w, _ := scriptWindow(t, revScript)
	p := prodPane(t, w)
	issueNow(t, w, "White", time.Now())
	before := p.count.Text
	realPaths(t)
	replaceText(t, w, "Mud everywhere.", "Mud everywhere, and a dog.")
	uiTurn.Lock()
	now := p.count.Text
	uiTurn.Unlock()
	if now != before {
		t.Errorf("the count changed at the key: %q", now)
	}
	// a late result's refresh while the pause is pending starts no work
	uiTurn.Lock()
	runs, pages := w.revisionRun, p.pagesRun
	p.refresh()
	moved := w.revisionRun != runs || p.pagesRun != pages
	uiTurn.Unlock()
	if moved {
		t.Error("a refresh while typing started the marks or the page count before the pause")
	}
	waitFor(t, "the count", func() bool { return strings.Contains(p.count.Text, "scene 2") })
	waitFor(t, "the pages", func() bool { return strings.Contains(p.count.Text, "1 page changed") })
}
