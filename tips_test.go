// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// A tip comes only after a pause in typing, and not again before tipGap;
// an event's (the third dd) soon after it.
func TestTipRateLimit(t *testing.T) {
	t0 := time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)
	tp := newTipper(t0)
	if tp.due(t0.Add(time.Minute)) {
		t.Error("a tip a minute after the start")
	}
	if !tp.due(t0.Add(tipGap)) {
		t.Error("no tip after the gap")
	}
	tp.typed(t0.Add(tipGap - time.Second))
	if tp.due(t0.Add(tipGap)) {
		t.Error("a tip while typing")
	}
	now := t0.Add(tipGap + tipIdle)
	if !tp.due(now) {
		t.Fatal("no tip after the pause")
	}
	if tp.choose(tipContext{keys: "standard"}, []string{"one"}, now) != "one" {
		t.Error("no random tip")
	}
	if tp.due(now.Add(time.Minute)) {
		t.Error("a second tip a minute after the first")
	}
	// three dd in one scene: das, soon after the last
	for range 3 {
		tp.noticed("dd", 4)
	}
	tp.typed(now.Add(2 * time.Minute))
	if tp.due(now.Add(2*time.Minute + time.Second)) {
		t.Error("the dd tip at once, while deleting")
	}
	if !tp.due(now.Add(2*time.Minute + tipEventIdle)) {
		t.Fatal("no dd tip after the third dd")
	}
	if got := tp.choose(tipContext{keys: "vim"}, []string{"one"}, now); !strings.HasPrefix(got, "das ") {
		t.Errorf("the third dd's tip: %q", got)
	}
	// dd in another scene each time: none
	tp = newTipper(t0)
	tp.noticed("dd", 4)
	tp.noticed("dd", 9)
	tp.noticed("dd", 9)
	if tp.pending != nil {
		t.Error("a dd tip for dd in two scenes")
	}
	tp.noticed("dd", 9)
	if tp.pending == nil {
		t.Error("no dd tip for three in one")
	}
	// das used: never tipped
	tp = newTipper(t0)
	tp.noticed("obj:as", 4)
	for range 3 {
		tp.noticed("dd", 4)
	}
	if tp.pending != nil {
		t.Error("the das tip after das was used")
	}
}

// The tip for the place, the most particular first; a key used or a tip
// said tipRepeat times gives way; the mode's own.
func TestTipContextChoice(t *testing.T) {
	now := time.Now()
	random := []string{"r1", "r2"}
	cue := tipContext{keys: "vim", onCue: true, inSpeech: true, inScene: true}
	tp := newTipper(now)
	if got := tp.choose(cue, random, now); !strings.HasPrefix(got, "dad ") {
		t.Errorf("vim on a cue: %q", got)
	}
	if got := tp.choose(tipContext{keys: "vim", inSpeech: true, inScene: true}, random, now); !strings.HasPrefix(got, "]d ") {
		t.Errorf("vim in a speech: %q", got)
	}
	if got := tp.choose(tipContext{keys: "vim", inBeat: true, inScene: true}, random, now); !strings.HasPrefix(got, "]B ") {
		t.Errorf("vim in a beat: %q", got)
	}
	tp.noticed("obj:ad", 0) // dad used: the next for the place
	if got := tp.choose(cue, random, now); !strings.HasPrefix(got, "]] ") {
		t.Errorf("vim on a cue, dad used: %q", got)
	}
	tp.choose(cue, random, now) // ]] said twice: the random ones now
	if got := tp.choose(cue, random, now); !slices.Contains(random, got) {
		t.Errorf("all said: %q", got)
	}
	// the random ones in turn: the least said first
	tp = newTipper(now)
	a, b := tp.choose(tipContext{keys: "vim"}, random, now), tp.choose(tipContext{keys: "vim"}, random, now)
	if a == b {
		t.Errorf("random tips not in turn: %q %q", a, b)
	}
	if got := newTipper(now).choose(tipContext{keys: "helix", onCue: true, inSpeech: true}, random, now); !strings.HasPrefix(got, "M ") {
		t.Errorf("helix on a cue: %q", got)
	}
	if got := newTipper(now).choose(tipContext{keys: "standard", onHeading: true, inScene: true}, random, now); !strings.HasPrefix(got, "Alt+Up moves this scene") {
		t.Errorf("standard keys on a heading: %q", got)
	}
	if got := newTipper(now).choose(tipContext{keys: "vim", onCue: true, markdown: true}, random, now); !slices.Contains(random, got) {
		t.Errorf("a Markdown document: %q", got)
	}
}

// The context tips teach keys the modes' table has and commands the
// registry has; the random ones come from Fun Things to Try and the table.
func TestTipsFromTheRegistry(t *testing.T) {
	for _, ct := range append(slices.Clone(contextTips), ddTip) {
		for _, u := range ct.uses {
			if ct.mode == "standard" {
				if _, ok := commands().byID[u]; !ok {
					t.Errorf("tip %q: no command %s", ct.text, u)
				}
				continue
			}
			if !slices.Contains(documented(ct.mode), u) {
				t.Errorf("%s tip %q: %q is not in the modes' table", ct.mode, ct.text, u)
			}
		}
		if s := ct.textOf(); s == "" || strings.Contains(s, "%") {
			t.Errorf("tip %q reads %q", ct.text, s)
		}
	}
	r := randomTips("vim", false)
	if !slices.ContainsFunc(r, func(s string) bool { return strings.HasPrefix(s, "Vim keys: ]d [d") }) {
		t.Error("no random tip from the keys' table")
	}
	if !slices.ContainsFunc(r, func(s string) bool { return strings.HasPrefix(s, funLines["Writing"][1].line) }) {
		t.Error("no random tip from Fun Things to Try")
	}
	if slices.ContainsFunc(randomTips("standard", false), func(s string) bool { return strings.HasPrefix(s, "Vim keys") }) {
		t.Error("vim's keys tipped with the standard ones")
	}
}

// A tip stands beside no fresh message and no warning.
func TestTipStatusAllows(t *testing.T) {
	for _, c := range []struct {
		text string
		age  time.Duration
		ok   bool
	}{
		{"", 0, true},
		{"-- NORMAL --", 0, true},
		{"Auto-saved 12:00", 0, true},
		{"Exported to film.pdf", time.Second, false},
		{"Exported to film.pdf", 2 * tipStale, true},
		{"Auto-save failed: disk full", 2 * tipStale, false},
		{"Could not read the PDF", 2 * tipStale, false},
		{"Tab: the next line", 2 * tipStale, false},
	} {
		if got := statusAllows(c.text, c.age); got != c.ok {
			t.Errorf("%q after %v: %v", c.text, c.age, got)
		}
	}
}

// In a window: a tip for the cursor's place when due, gone with a
// message, none with the setting off; View > Show a Tip at once.
func TestTipsInWindow(t *testing.T) {
	w := newTestWindow(t, vimObjScript)
	w.settings.SetString("editor-keys", "vim")
	t.Cleanup(func() { w.settings.SetString("editor-keys", "standard") })
	w.vim.reset(true)
	cursorTo(t, w.textEditor, vimObjScript, "ALICE")
	w.setStatus("")
	later := time.Now().Add(tipGap + tipIdle)
	w.tipTick(later)
	if got := w.tipText(); !strings.HasPrefix(got, "dad ") {
		t.Fatalf("the tip on a cue: %q", got)
	}
	w.setStatus("Exported to film.pdf")
	if w.tipText() != "" {
		t.Error("the tip over a message")
	}
	w.tips = newTipper(time.Now().Add(-2 * tipGap)) // a tip due, but for the message
	w.tipTick(time.Now().Add(tipIdle))
	if w.tipText() != "" {
		t.Error("a tip beside a fresh message")
	}
	w.setStatus("")
	w.tipInput()
	w.tipTick(time.Now().Add(time.Second)) // typing
	if w.tipText() != "" {
		t.Error("a tip while typing")
	}
	w.settings.SetBoolean(tipsPref, false)
	t.Cleanup(func() { w.settings.SetBoolean(tipsPref, true) })
	w.tipTick(later.Add(3 * tipGap))
	if w.tipText() != "" {
		t.Error("a tip with tips off")
	}
	w.settings.SetBoolean(tipsPref, true)
	cmd("view.tip").run(w)
	if w.tipText() == "" {
		t.Error("Show a Tip showed none")
	}
	// the third dd in a scene: das
	w = newTestWindow(t, "INT. A - DAY\n\nOne.\nTwo.\nThree.\nFour.\n")
	w.settings.SetString("editor-keys", "vim")
	t.Cleanup(func() { w.settings.SetString("editor-keys", "standard") })
	w.vim.reset(true)
	cursorTo(t, w.textEditor, w.textEditor.Text(), "One")
	vimKeys(w.textEditor, "dddddd")
	w.tipTick(time.Now().Add(tipEventIdle + time.Second))
	if got := w.tipText(); !strings.HasPrefix(got, "das ") {
		t.Errorf("after three dd: %q", got)
	}
}
