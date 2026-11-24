package main

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/lexington/rules"
)

// Starting from a structure (Joop 2026-11-24: "We could provide a
// workflows support thing, where we provide a beats workflow and the
// several common strategies as an offered workflow template"): a few
// common ways to lay out a story, each a starting text made of the
// script's own tools and nothing more: sections (# acts, ## parts and
// stages), synopses (= what happens there) and a note on how to use it.
// No mode and no view of their own: the outline, the cards, folding and
// the structural moves work on them as on any script, and each line is
// written over, moved, grown into scenes or deleted. File > New from
// Structure starts a script with one; Insert Structure (:structure) puts
// one at the end of this script's body, one undo step. The descriptions
// are Accolade's own; a method's originator is named as an attribution.
// docs/help/structure.md shows them as worked examples of the tools.

// structureTemplate is a structure to start from.
type structureTemplate struct {
	name  string // as the menus list it
	after string // whose method it follows ("" for none)
	body  string // its sections and synopses, after the note
}

// structureTemplates are the structures, in the menus' order.
var structureTemplates = []structureTemplate{
	{name: "Three Acts", body: `# Act One

= Who, where, and what is wrong in their world.

## Inciting Incident
= Something happens that cannot be ignored.

## First Turning Point
= The hero commits, and there is no going back.

# Act Two

= Trying, failing, and the stakes rising.

## Midpoint
= A win or a loss that changes what the story is about.

## Second Turning Point
= Everything falls apart; the hero has to choose.

# Act Three

= The last confrontation, and what it costs.

## Climax
= The choice made in action.

## Resolution
= The world after, and who the hero has become.
`},
	{name: "Sequences", after: "Frank Daniel", body: `# Act One

= Two sequences: the world, and the problem that comes into it.

## Sequence 1: The World
= The hero's life as it is, and the first hint of trouble.

## Sequence 2: The Push
= The problem arrives; the hero is drawn in.

# Act Two

= Four sequences: the struggle, turning at the midpoint.

## Sequence 3: First Steps
= The first attempt at the goal, in a new situation.

## Sequence 4: Complications
= It is harder than it looked; the stakes rise to the midpoint.

## Sequence 5: New Direction
= After the midpoint, a new plan or a new understanding.

## Sequence 6: The Fall
= The plan fails; the lowest point.

# Act Three

= Two sequences: the last push, and what it leaves.

## Sequence 7: The Final Push
= The hero gathers what is left and acts.

## Sequence 8: Aftermath
= The outcome, and the new balance.
`},
	{name: "Hero's Journey", after: "Joseph Campbell and Christopher Vogler", body: `# Departure

= Leaving the known world.

## The Ordinary World
= The hero at home, and what is missing.

## The Call
= A challenge or a chance appears.

## Refusing the Call
= Fear, doubt or duty holds the hero back.

## Meeting the Mentor
= Someone gives advice, a gift or a push.

## Crossing the Threshold
= The hero leaves the known world.

# Initiation

= Trials in the unknown, and what they change.

## Tests, Allies, Enemies
= Learning the rules of the new world, and who is who.

## Approaching the Cave
= Getting ready for the greatest danger.

## The Ordeal
= Facing death, literal or not.

## The Reward
= What was won by surviving.

# Return

= Coming home, changed.

## The Road Back
= Turning for home, pursued.

## Resurrection
= A last test, where the change is proved.

## Return with the Elixir
= Home again, bringing something that changes it.
`},
	{name: "Story Circle", after: "Dan Harmon", body: `# Order

= The known world: wanting, and setting out.

## 1. You
= Someone in their comfort zone.

## 2. Need
= They want something.

## 3. Go
= They enter an unfamiliar situation.

## 4. Search
= They adapt to it.

# Chaos

= The unknown: getting, paying, coming back changed.

## 5. Find
= They get what they wanted.

## 6. Take
= They pay a heavy price for it.

## 7. Return
= They go back to where they started.

## 8. Change
= Having changed.
`},
	{name: "Kishōtenketsu", body: `# Ki: Introduction

= The characters and their world, as they are.

# Shō: Development

= The same world followed further; nothing breaks yet.

# Ten: Twist

= Something unexpected, perhaps unrelated, comes in.

# Ketsu: Reconciliation

= The parts brought together: the twist makes sense of what came before.
`},
	{name: "Beat Sheet (15 Beats)", after: "Blake Snyder", body: `# Act One

= The world before, and the push out of it.

## Opening Image
= A snapshot of the hero's world before.

## Theme Stated
= Someone says what the story is really about.

## Set-Up
= The hero's life, flaws and what needs fixing.

## Catalyst
= The event that sets the story going.

## Debate
= Should the hero go? Doubt, a last chance to stay.

# Act Two

= The new world, its promise, and its cost.

## Break into Two
= The hero chooses, and enters a new world.

## B Story
= A second story, often a relationship, that carries the theme.

## Fun and Games
= The promise of the premise: what the story is for.

## Midpoint
= A false victory or a false defeat; the stakes go up.

## Bad Guys Close In
= Enemies outside and doubts inside press on.

## All Is Lost
= The lowest moment; something or someone is lost.

## Dark Night of the Soul
= The hero takes in the loss.

# Act Three

= The answer, acted on.

## Break into Three
= An answer found, often through the B story.

## Finale
= The hero acts on what was learned and wins or loses.

## Final Image
= The world after: the opening image, changed.
`},
	{name: "Beats First", body: `# Act One

= What this act does.

= …

= …

= …

# Act Two

= What this act does.

= …

= …

= …

# Act Three

= What this act does.

= …

= …

= …
`},
}

// structureNote is the note a structure starts with: how to use it.
func structureNote(t structureTemplate) string {
	by := ""
	if t.after != "" {
		by = ", after " + t.after
	}
	return "[[" + t.name + by + ". Sections (#) are its parts, synopses (= lines) what happens in each: " +
		"write yours over them and delete what you do not need. Alt+Up and Alt+Down move a part or a synopsis; " +
		"New Scene on a synopsis starts its scene there. More in Help > Structure.]]"
}

// structureText is t as it goes in: its note, then its body.
func structureText(t structureTemplate) string {
	return structureNote(t) + "\n\n" + t.body
}

// titlePageIn is a new script's title page in l's words, and the blank
// line after it.
func titlePageIn(l rules.Language) string {
	key := func(std string) string { return titleKeyIn(l, std) }
	return strings.Join([]string{
		key("Title") + ": UNTITLED SCREENPLAY",
		key("Credit") + ": Written by",
		key("Author") + ": Your Name",
		key("Draft date") + ": " + time.Now().Format("January 2, 2006"),
		"",
		"",
	}, "\n")
}

// newFromStructure starts a new script with structure t (asking first if
// the script has unsaved changes), as New does.
func (w *MainWindow) newFromStructure(t structureTemplate) {
	start := func() {
		w.createNewFile(true)
		w.textEditor.SetText(titlePageIn(w.lex().Language(w.language)) + structureText(t))
		w.textEditor.SetCursorOffset(0)
		w.hasChanges = false
		w.updateTitle()
		w.setStatus("A new script from " + t.name + ": write over its synopses (Help > Structure)")
	}
	if !w.hasChanges {
		start()
		return
	}
	dialog.ShowConfirm("Unsaved Changes", "You have unsaved changes. Create a new file anyway?", func(ok bool) {
		if ok {
			start()
		}
	}, w.fyneWindow)
}

// insertStructure puts structure t at the end of the script's body
// (before the blocks at its end), one undo step, the cursor at its note.
func (w *MainWindow) insertStructure(t structureTemplate) {
	if w.markdown || w.textEditor == nil || w.scriptAway("Insert Structure") {
		return
	}
	text := w.textEditor.Text()
	at := bodyEnd(text)
	before := string([]rune(text)[:at])
	pre := ""
	switch {
	case strings.TrimSpace(before) == "":
	case strings.HasSuffix(before, "\n\n"):
	case strings.HasSuffix(before, "\n"):
		pre = "\n"
	default:
		pre = "\n\n"
	}
	ins := pre + structureText(t)
	switch after := string([]rune(text)[at:]); {
	case strings.HasPrefix(after, "\n"): // its blank lines are there
		ins = strings.TrimSuffix(ins, "\n")
	case strings.TrimSpace(after) != "":
		ins += "\n"
	}
	w.textEditor.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			b.Replace(at, at, ins)
			b.SetCursor(at+len([]rune(pre)), false)
		})
	})
	w.fyneWindow.Canvas().Focus(w.textEditor)
	w.setStatus(fmt.Sprintf("%s put in at the end: write over its synopses", t.name))
}

// templateMenu lists the structures, each doing do with it.
func templateMenu(label string, do func(t structureTemplate)) *fyne.Menu {
	var items []*fyne.MenuItem
	for _, t := range structureTemplates {
		items = append(items, fyne.NewMenuItem(t.name, func() { do(t) }))
	}
	return fyne.NewMenu(label, items...)
}

// templateCommands are New from Structure and Insert Structure.
func templateCommands() []*command {
	return []*command{
		{id: "file.newStructure", label: "New from Structure", icon: theme.DocumentCreateIcon,
			tip: "A new script laid out as a structure: three acts, sequences, a beat sheet…, its parts as sections and synopses",
			submenu: func(w *MainWindow) *fyne.Menu { return templateMenu("New from Structure", w.newFromStructure) }},
		{id: "write.insertStructure", label: "Insert Structure", ex: "structure", icon: theme.ListIcon, when: screenplayOnly,
			tip: "A structure's sections and synopses put in at the end of this script, one undo step",
			submenu: func(w *MainWindow) *fyne.Menu { return templateMenu("Insert Structure", w.insertStructure) }},
	}
}
