// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2/theme"

	"github.com/LaPingvino/accolade/internal/structure"
)

// Tips in the status bar (Joop 2026-11-28): now and then, while nothing
// is typed, a quiet line beside the status says something worth trying.
// Context first: with vim keys and the cursor on a cue, "dad deletes this
// speech"; in a beat, "]B jumps to the next beat"; after the third dd in
// one scene, "das deletes a whole scene". Else one at random, from Fun
// Things to Try (fun.go) and the keys' table (vim_keys.go), so they stay
// what the app does. A key already used is not tipped again.
//
// At most one, in a label of its own after the status: never over a
// message or a warning (beside one only once it has stood a while and is
// no warning), never while typing (tipIdle), and not often (tipGap).
// View > Show's More has Show a Tip, one now, and Tips in the Status Bar,
// the setting (on by default).

const (
	tipsPref     = "status-tips"
	tipIdle      = 8 * time.Second  // no key for so long first
	tipEventIdle = 2 * time.Second  // an event's tip (the third dd): soon after
	tipGap       = 4 * time.Minute  // between two tips
	tipTick      = 3 * time.Second  // how often the window looks
	tipStale     = 90 * time.Second // a message older is not in the way (no warning)
	tipRepeat    = 2                // a context tip is said so many times at most
)

// tipContext is where the cursor is, as the tips read it.
type tipContext struct {
	keys                                      string // standard, vim, helix
	onCue, inSpeech, inBeat, inNote, inAction bool
	inScene, onHeading                        bool
	selection                                 bool // something selected
	markdown                                  bool
}

// contextTip is a tip for a place: shown when when holds, in mode
// ("standard", "vim", "helix"); uses are the keys it teaches (a key
// sequence of the modes' table, vim_keys_test checks; a command's id in
// standard mode), not tipped once used.
type contextTip struct {
	mode string
	when func(tipContext) bool
	uses []string
	text string
}

func tipOnCue(c tipContext) bool    { return c.onCue }
func tipInSpeech(c tipContext) bool { return c.inSpeech && !c.onCue }
func tipInBeat(c tipContext) bool   { return c.inBeat }
func tipInNote(c tipContext) bool   { return c.inNote }
func tipInAction(c tipContext) bool { return c.inAction }
func tipInScene(c tipContext) bool  { return c.inScene }
func tipOnHead(c tipContext) bool   { return c.onHeading }
func tipSelected(c tipContext) bool { return c.selection }

// contextTips are the tips for places, the most particular first.
var contextTips = []contextTip{
	{"helix", tipSelected, []string{"s"}, "s @dialogue selects the dialogue in the selection; @cue @paren @note @beat and more (S splits at them)"},
	{"helix", tipSelected, []string{"structure.expand"}, "%s grows the selection to the speech, the scene, the section"},
	{"vim", tipSelected, []string{"structure.expand"}, "%s grows the selection to the speech, the scene, the section"},
	{"standard", tipSelected, []string{"structure.expand"}, "%s grows the selection to the speech, the scene, the section"},
	{"standard", tipSelected, []string{"select.cue"}, "%s, and the other Select All commands, pick out every element of a kind in the selection"},
	{"vim", tipInNote, []string{"dan", "cin"}, "dan deletes this note, cin rewrites what it says"},
	{"vim", tipOnCue, []string{"dad"}, "dad deletes this speech; cic renames the cue, keeping its (V.O.)"},
	{"vim", tipInSpeech, []string{"]d", "cid"}, "]d jumps to the next speech; cid says this one again"},
	{"vim", tipInBeat, []string{"]B"}, "]B jumps to the next beat, daB takes this one out"},
	{"vim", tipInAction, []string{"daa"}, "daa deletes this action paragraph, yaa copies it"},
	{"vim", tipOnHead, []string{"das"}, "das deletes this scene; p puts it back after the scene the cursor is in"},
	{"vim", tipInScene, []string{"]]"}, "]] and [[ go from scene to scene"},
	{"helix", tipInNote, []string{"man"}, "man selects this note"},
	{"helix", tipOnCue, []string{"M"}, "M selects every speech of this character"},
	{"helix", tipInSpeech, []string{"mad"}, "mad selects this speech, ]d the next"},
	{"helix", tipInBeat, []string{"]B"}, "]B selects the next beat"},
	{"helix", tipInAction, []string{"maa"}, "maa selects this action paragraph"},
	{"helix", tipOnHead, []string{"mas"}, "mas selects this scene; : then moves it"},
	{"standard", tipOnCue, []string{"view.characters"}, "%s shows everyone who speaks; rename one there, everywhere at once"},
	{"standard", tipOnHead, []string{"structure.up"}, "%s moves this scene up past the one before, its links and numbers with it"},
	{"standard", tipInSpeech, []string{"structure.transpose"}, "%s gives this speech's lines to the speaker before"},
	{"standard", tipInNote, []string{"view.notes"}, "%s gathers every note in the Notes pane"},
}

// ddTip is the third dd's in one scene.
var ddTip = contextTip{"vim", nil, []string{"das"}, "das deletes a whole scene, 3dd at a time no longer"}

// textOf is t's line: a standard tip's %s is its command's key, or its
// name where it has none.
func (t contextTip) textOf() string {
	if !strings.Contains(t.uses[0], ".") { // a key of the modes' table
		return t.text
	}
	c, ok := commands().byID[t.uses[0]]
	if !ok {
		return ""
	}
	k := c.label
	if len(c.keys) > 0 {
		k = keyName(c.keys[0])
	}
	return fmt.Sprintf(t.text, k)
}

// tipper decides when a tip comes and which.
type tipper struct {
	last    time.Time // the last tip
	input   time.Time // the last key
	said    map[string]int
	used    map[string]bool // keys used: no tip for them
	pending *contextTip     // an event's tip, for the next chance
	dd      struct{ scene, n int }
	rnd     func(n int) int
}

func newTipper(now time.Time) *tipper {
	return &tipper{last: now, input: now, said: map[string]int{}, used: map[string]bool{}, rnd: rand.IntN}
}

// typed is a key at now: no tip for a while.
func (t *tipper) typed(now time.Time) { t.input = now }

// due tells whether a tip may come at now.
func (t *tipper) due(now time.Time) bool {
	if t.pending != nil {
		return now.Sub(t.input) >= tipEventIdle
	}
	return now.Sub(t.input) >= tipIdle && now.Sub(t.last) >= tipGap
}

// noticed is what the keys did: an object's or motion's keys used, a dd
// at a line of the scene with heading scene (-1 none).
func (t *tipper) noticed(what string, scene int) {
	switch {
	case what == "dd":
		if t.dd.n > 0 && t.dd.scene == scene && scene >= 0 {
			t.dd.n++
		} else {
			t.dd.scene, t.dd.n = scene, 1
		}
		if t.dd.n == 3 && !t.usedAny(ddTip.uses) && t.said[ddTip.text] < tipRepeat {
			tp := ddTip
			t.pending = &tp
		}
		return
	case strings.HasPrefix(what, "obj:"):
		obj := strings.TrimPrefix(what, "obj:")
		for _, op := range "dcyvm" {
			t.used[string(op)+obj] = true
		}
	case strings.HasPrefix(what, "motion:"):
		k := strings.TrimPrefix(what, "motion:")
		t.used["]"+k], t.used["["+k] = true, true
	default:
		t.used[what] = true
	}
	t.dd.n = 0
}

func (t *tipper) usedAny(keys []string) bool {
	return slices.ContainsFunc(keys, func(k string) bool { return t.used[k] })
}

// choose is the tip for c (random: the context-free ones), "" none,
// and marks it said at now.
func (t *tipper) choose(c tipContext, random []string, now time.Time) string {
	say := func(s string) string {
		t.said[s]++
		t.last = now
		return s
	}
	if p := t.pending; p != nil {
		t.pending = nil
		if c.keys == p.mode {
			return say(p.text)
		}
	}
	for _, ct := range contextTips {
		if ct.mode != c.keys || c.markdown || !ct.when(c) || t.usedAny(ct.uses) {
			continue
		}
		if s := ct.textOf(); s != "" && t.said[s] < tipRepeat {
			return say(s)
		}
	}
	if len(random) == 0 {
		return ""
	}
	least := -1 // the ones said least, one of them at random
	var pick []string
	for _, s := range random {
		switch n := t.said[s]; {
		case least < 0 || n < least:
			least, pick = n, []string{s}
		case n == least:
			pick = append(pick, s)
		}
	}
	return say(pick[t.rnd(len(pick))])
}

// statusAllows tells whether a tip may stand beside the status text,
// shown for age: none, a mode's line, Saved; another message once it is
// old, and never a warning.
func statusAllows(text string, age time.Duration) bool {
	if text == "" || strings.HasPrefix(text, "-- ") || strings.HasPrefix(text, "Auto-saved ") || text == "Saved" {
		return true
	}
	if strings.HasPrefix(text, hintPrefix) || age < tipStale {
		return false
	}
	l := strings.ToLower(text)
	for _, w := range []string{"fail", "error", "could not", "couldn't", "cannot", "can't", "warning", "not saved", "!"} {
		if strings.Contains(l, w) {
			return false
		}
	}
	return true
}

// randomTips are the context-free tips for keys: Fun Things to Try's
// lines with their command's key, and the keys' table's lines for vim's
// or helix's.
func randomTips(keys string, markdown bool) []string {
	var out []string
	for _, m := range funMoments {
		for _, l := range funLines[m] {
			c, ok := commands().byID[l.run]
			if !ok || markdown && c.when != nil {
				continue
			}
			s := l.line
			if len(c.keys) > 0 {
				s += " (" + keyName(c.keys[0]) + ")"
			}
			out = append(out, s)
		}
	}
	if keys == "vim" || keys == "helix" {
		for _, k := range keysOfMode(keys) {
			if k.group == "Modes" || k.group == "Moving" && k.keys != "1 … 9" {
				continue
			}
			out = append(out, modeTitle(keys)+" keys: "+k.keys+" — "+k.desc)
		}
	}
	return out
}

// startTips arms the window's tips: it looks every tipTick.
func (w *MainWindow) startTips() {
	w.tips = newTipper(time.Now())
	if debounceAtOnce { // the tests: they tick themselves (a timer's function would run beside them)
		return
	}
	var tick func()
	tick = func() {
		w.tipTick(time.Now())
		w.after(tipTick, tick)
	}
	w.after(tipTick, tick)
}

// tipsOn tells whether the setting has tips.
func (w *MainWindow) tipsOn() bool { return w.settings != nil && w.settings.GetBoolean(tipsPref) }

// tipInput is a key typed in the window's editors: the tip goes.
func (w *MainWindow) tipInput() {
	if w.tips != nil {
		w.tips.typed(time.Now())
	}
	w.hideTip()
}

// tipNoticed is what vim's or helix's keys did (vim.noticed).
func (w *MainWindow) tipNoticed(what string, line int) {
	if w.tips == nil {
		return
	}
	scene := -1
	if what == "dd" {
		if tr, _ := w.structTree(w.textEditor.Text()); tr != nil {
			if f, _, ok := tr.Span(line, structure.Scene); ok {
				scene = f
			}
		}
	}
	w.tips.noticed(what, scene)
}

// tipTick shows a tip if one is due at now and the status lets it.
func (w *MainWindow) tipTick(now time.Time) {
	if w.tips == nil || !w.tipsOn() || w.tipLabel == nil || w.tipLabel.Visible() || !w.tips.due(now) {
		return
	}
	if !statusAllows(w.statusText(), now.Sub(w.statusAt)) {
		return
	}
	w.showTip(now)
}

// showTip shows the tip for here (View > Show > Tips: at once).
func (w *MainWindow) showTip(now time.Time) {
	if w.tips == nil {
		w.tips = newTipper(now)
	}
	c := w.tipContextHere()
	s := w.tips.choose(c, randomTips(c.keys, c.markdown), now)
	if s == "" || w.tipLabel == nil {
		return
	}
	w.tipLabel.SetText("Tip: " + s)
	w.tipLabel.Show()
	if w.statusBar != nil {
		w.statusBar.Refresh()
	}
}

// hideTip takes the tip away.
func (w *MainWindow) hideTip() {
	if w.tipLabel != nil && w.tipLabel.Visible() {
		w.tipLabel.Hide()
		w.tipLabel.SetText("")
	}
}

// tipText is the tip shown, "" none.
func (w *MainWindow) tipText() string {
	if w.tipLabel == nil || !w.tipLabel.Visible() {
		return ""
	}
	return strings.TrimPrefix(w.tipLabel.Text, "Tip: ")
}

// tipContextHere is where the main editor's cursor is, read as vim's
// objects read it.
func (w *MainWindow) tipContextHere() tipContext {
	c := tipContext{keys: w.editorKeys(), markdown: w.markdown}
	v := w.vim
	if v == nil || w.textEditor == nil || w.markdown || !w.scriptInMain() {
		return c
	}
	tr := v.unitTree()
	line := v.curLine()
	if sp, ok := tr.SpeechAt(line); ok {
		c.inSpeech, c.onCue = true, line == sp.Cue
	}
	c.selection = w.textEditor.Buffer().HasSelection()
	_, why := v.objectSpan('a', 'B', 1)
	c.inBeat = why == ""
	_, why = v.objectSpan('a', 'n', 1)
	c.inNote = why == ""
	_, _, c.inAction = tr.Paragraph(line)
	if f, _, ok := tr.Span(line, structure.Scene); ok {
		c.inScene, c.onHeading = true, f == line
	}
	return c
}

// tipCommands are View > Show's: a tip now, and the setting.
func tipCommands() []*command {
	return []*command{
		{id: "view.tip", label: "Show a Tip", ribbonLabel: "Tips", ex: "tip", icon: theme.QuestionIcon,
			tip: "A tip for where the cursor is, in the status bar",
			run: func(w *MainWindow) { w.hideTip(); w.showTip(time.Now()) }},
		toggleSetting("set.tips", "Tips in the Status Bar", tipsPref, theme.InfoIcon(), func(w *MainWindow) {
			if !w.tipsOn() {
				w.hideTip()
			}
		}),
	}
}
