// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

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
// Structure starts a script with one; Insert Beats / Structure (:structure) puts
// one in after the cursor's section (structureInsertAt), one undo step. The descriptions
// are Accolade's own; a method's originator is named as an attribution.
// docs/help/structure.md shows them as worked examples of the tools.

// structureTemplate is a structure to start from.
type structureTemplate struct {
	name  string // as the menus list it
	after string // whose method it follows ("" for none)
	body  string // its sections and synopses, after the note
	// ranges are where its sections usually start, in % of the script's
	// pages, by sectionKey (beat_place.go): structureRanges
	ranges map[string]beatRange
	// aims are what each section is for and how tense it is meant to
	// feel, by sectionKey: structureAims
	aims map[string]beatAim
}

// beatAim is what a structure's section is for: its purpose in a line,
// and the tension it is meant to carry, 1 (calm) to 5 (the most).
type beatAim struct {
	purpose string
	tension int
}

// tensionWord says a tension level in words, for the tooltips.
func (a beatAim) tensionWord() string {
	return fmt.Sprintf("tension %d of 5", a.tension)
}

// structureRanges are where the structures' sections usually start, in
// % of a feature's pages, by their names: measured on films where a
// published study times that beat (beat_films.go, with its source; the
// beat taken as the one the method itself equates it with), else the
// conventional range the method and the manuals give (a page count on a
// 110-page script, or a share of it), widened a little, and labelled
// so. A section not here gets its page and place without a range.
var structureRanges = map[string]map[string]beatRange{
	"Three Acts": {
		"Inciting Incident": films(filmInciting), "First Turning Point": films(filmActOneEnd), "Act Two": films(filmActOneEnd),
		"Midpoint": films(filmMidpoint), "Second Turning Point": films(filmActTwoEnd), "Act Three": films(filmActTwoEnd),
		"Climax": films(filmClimax), "Resolution": films(filmEpilog),
	},
	"Sequences": { // eight of about an eighth each; the act breaks and the midpoint measured
		"Sequence 2: The Push": conv(9, 16), "Act Two": films(filmActOneEnd), "Sequence 3: First Steps": films(filmActOneEnd),
		"Sequence 4: Complications": conv(33, 42), "Sequence 5: New Direction": films(filmMidpoint), "Sequence 6: The Fall": conv(58, 67),
		"Act Three": films(filmActTwoEnd), "Sequence 7: The Final Push": films(filmActTwoEnd), "Sequence 8: Aftermath": conv(84, 95),
	},
	"Hero's Journey": { // its stages are not timed by any study: convention
		"The Call": conv(5, 14), "Refusing the Call": conv(8, 18), "Meeting the Mentor": conv(12, 22),
		"Crossing the Threshold": conv(20, 30), "Initiation": conv(20, 30), "Tests, Allies, Enemies": conv(22, 40),
		"Approaching the Cave": conv(38, 50), "The Ordeal": conv(45, 60), "The Reward": conv(55, 70),
		"Return": conv(70, 80), "The Road Back": conv(70, 80), "Resurrection": conv(80, 92), "Return with the Elixir": conv(90, 100),
	},
	"Story Circle": { // eight steps, the second half's Chaos from the middle; not timed by any study
		"2. Need": conv(8, 18), "3. Go": conv(20, 30), "4. Search": conv(30, 42),
		"Chaos": conv(45, 55), "5. Find": conv(45, 55), "6. Take": conv(58, 70), "7. Return": conv(70, 82), "8. Change": conv(84, 95),
	},
	"Kishōtenketsu": {"Shō: Development": conv(20, 30), "Ten: Twist": conv(45, 60), "Ketsu: Reconciliation": conv(72, 85)},
	"Beat Sheet (15 Beats)": { // after Blake Snyder's page guide for 110 pages; his act breaks, Catalyst, Midpoint and All Is Lost measured
		"Theme Stated": conv(3, 7), "Set-Up": conv(0, 9), "Catalyst": films(filmInciting), "Debate": conv(10, 20),
		"Act Two": films(filmActOneEnd), "Break into Two": films(filmActOneEnd), "B Story": conv(25, 32), "Fun and Games": conv(25, 32),
		"Midpoint": films(filmMidpoint), "Bad Guys Close In": conv(48, 56), "All Is Lost": films(filmLowPoint),
		"Dark Night of the Soul": conv(66, 77), "Act Three": films(filmActTwoEnd), "Break into Three": films(filmActTwoEnd),
		"Finale": conv(75, 85), "Final Image": conv(95, 100),
	},
	// Freytag's five parts of a five-act drama, each about a fifth; his
	// climax is the turn in the middle (not the films' climax near the
	// end), so all convention
	"Freytag's Pyramid": {
		"Inciting Moment": conv(8, 20), "Rising Action": conv(15, 28), "Climax": conv(42, 56), "Tragic Force": conv(48, 62),
		"Falling Action": conv(55, 68), "Moment of Final Suspense": conv(70, 86), "Catastrophe": conv(78, 95),
	},
	// Truby puts his steps in an order, not on pages: wide ranges where
	// a feature usually has them, convention
	"Seven Key Steps": {
		"Desire": conv(5, 22), "Opponent": conv(5, 30), "Plan": conv(15, 38),
		"Battle": conv(72, 92), "Self-Revelation": conv(82, 97), "New Equilibrium": conv(90, 100),
	},
	// Hauge's turning points at 10, 25, 50, 75 and 90–99%, each stage
	// starting at the one before it: the measured beats where they are
	// the inciting incident, the act breaks, the midpoint and the climax
	"Six Stages": {
		"Opportunity": films(filmInciting), "New Situation": conv(8, 20), "Change of Plans": films(filmActOneEnd),
		"Progress": films(filmActOneEnd), "Point of No Return": films(filmMidpoint), "Complications and Higher Stakes": conv(48, 60),
		"Major Setback": films(filmActTwoEnd), "Final Push": films(filmActTwoEnd), "Climax": films(filmClimax),
		"Aftermath": films(filmEpilog),
	},
}

// structureAims are what each structure's sections are for and the
// tension each is meant to carry (1 calm to 5 the most), by their
// names: Accolade's own reading, a line each, for the Beats tooltip
// and Help > Structure (every section has one: TestStructureAims).
var structureAims = map[string]map[string]beatAim{
	"Three Acts": {
		"Act One":              {"Make us care about someone, and show what is wrong", 1},
		"Inciting Incident":    {"Upset the balance so the story has to start", 2},
		"First Turning Point":  {"Lock the hero into the story", 3},
		"Act Two":              {"Make every attempt cost more than the last", 3},
		"Midpoint":             {"Turn the story: the goal or the stakes change", 4},
		"Second Turning Point": {"Take away the easy ways out", 4},
		"Act Three":            {"Bring every thread to one confrontation", 4},
		"Climax":               {"Answer the story's question in action", 5},
		"Resolution":           {"Let us see what the change means", 1},
	},
	"Sequences": {
		"Act One":                    {"Lay the ground the other sequences build on", 1},
		"Sequence 1: The World":      {"Show the normal, so that its loss is felt", 1},
		"Sequence 2: The Push":       {"Bring the problem in and pull the hero toward it", 2},
		"Act Two":                    {"Carry the struggle, and turn it at the middle", 3},
		"Sequence 3: First Steps":    {"Show the hero trying, and the new situation's rules", 2},
		"Sequence 4: Complications":  {"Make the goal harder, climbing to the midpoint", 3},
		"Sequence 5: New Direction":  {"Commit the hero again, on new terms", 3},
		"Sequence 6: The Fall":       {"Break the plan and reach the lowest point", 4},
		"Act Three":                  {"Spend what is left on the outcome", 4},
		"Sequence 7: The Final Push": {"Stake everything on one last attempt", 5},
		"Sequence 8: Aftermath":      {"Show the new balance and what it cost", 2},
	},
	"Hero's Journey": {
		"Departure":              {"Take the hero out of the familiar", 1},
		"The Ordinary World":     {"Show where the hero starts, and what they lack", 1},
		"The Call":               {"Offer the adventure", 2},
		"Refusing the Call":      {"Show what holding back protects, and what it costs", 2},
		"Meeting the Mentor":     {"Give the hero what they need to dare", 2},
		"Crossing the Threshold": {"Commit the hero to the unknown", 3},
		"Initiation":             {"Change the hero through trial", 3},
		"Tests, Allies, Enemies": {"Teach the new world, and sort friend from foe", 3},
		"Approaching the Cave":   {"Gather the dread before the greatest danger", 3},
		"The Ordeal":             {"Bring the hero face to face with defeat", 5},
		"The Reward":             {"Let the hero hold what surviving bought", 3},
		"Return":                 {"Bring the change back to the world it came from", 3},
		"The Road Back":          {"Make going home a danger of its own", 4},
		"Resurrection":           {"Prove the change in a last test", 5},
		"Return with the Elixir": {"Show the world made better by what came back", 1},
	},
	"Story Circle": {
		"Order":     {"Set out from what is known", 1},
		"1. You":    {"Fix the character in their comfort", 1},
		"2. Need":   {"Open the want that drives the story", 2},
		"3. Go":     {"Take them over the line into the unknown", 2},
		"4. Search": {"Make them adapt to get by there", 3},
		"Chaos":     {"Turn the getting into a price", 3},
		"5. Find":   {"Give them what they wanted", 3},
		"6. Take":   {"Make them pay for it", 5},
		"7. Return": {"Bring them back to where they began", 3},
		"8. Change": {"Show who they have become", 2},
	},
	"Kishōtenketsu": {
		"Ki: Introduction":      {"Introduce, without a conflict", 1},
		"Shō: Development":      {"Deepen what we know, still calm", 2},
		"Ten: Twist":            {"Shift the ground with something unexpected", 4},
		"Ketsu: Reconciliation": {"Let the parts meet and show the whole", 2},
	},
	"Beat Sheet (15 Beats)": {
		"Act One":                {"Set up the world before the change", 1},
		"Opening Image":          {"Give the picture of before", 1},
		"Theme Stated":           {"Plant the question the story will answer", 1},
		"Set-Up":                 {"Show what needs fixing in the hero's life", 1},
		"Catalyst":               {"Knock the hero's life off course", 2},
		"Debate":                 {"Let the hero hesitate, so that the choice weighs", 2},
		"Act Two":                {"Turn the hero's world upside down", 3},
		"Break into Two":         {"Make the hero choose the new world", 3},
		"B Story":                {"Bring in the relationship that carries the theme", 2},
		"Fun and Games":          {"Give the audience the premise they came for", 2},
		"Midpoint":               {"Raise the stakes with a false win or a false loss", 3},
		"Bad Guys Close In":      {"Tighten the pressure from outside and within", 4},
		"All Is Lost":            {"Take away what the hero relied on", 4},
		"Dark Night of the Soul": {"Let the loss sink in before the answer comes", 3},
		"Act Three":              {"Act on what was learned", 4},
		"Break into Three":       {"Find the answer that joins the two stories", 4},
		"Finale":                 {"Prove the change by beating the problem", 5},
		"Final Image":            {"Mirror the opening, changed", 1},
	},
	"Freytag's Pyramid": {
		"Exposition":               {"Ground us in the people and the place before anything moves", 1},
		"Inciting Moment":          {"Set the conflict going", 2},
		"Rising Action":            {"Build the conflict step by step toward its height", 3},
		"Climax":                   {"Turn the hero's fortune at the peak", 5},
		"Tragic Force":             {"Set the fall in motion", 4},
		"Falling Action":           {"Let the consequences close in", 3},
		"Moment of Final Suspense": {"Hold out one last hope that it could still turn", 4},
		"Catastrophe":              {"End the struggle and settle the hero's fate", 4},
	},
	"Seven Key Steps": {
		"Weakness and Need": {"Show the flaw the story will put to the test", 1},
		"Desire":            {"Give the story a line to follow", 2},
		"Opponent":          {"Embody the pressure on the hero's weakness", 3},
		"Plan":              {"Lay out what the hero will try, so it can break", 3},
		"Battle":            {"Force the final test of what the hero believes", 5},
		"Self-Revelation":   {"Let the hero see what they were blind to", 4},
		"New Equilibrium":   {"Show the hero's world after the change", 1},
	},
	"Six Stages": {
		"Setup":                           {"Make us know and like the hero before the change", 1},
		"Opportunity":                     {"Open a door the hero can walk through", 2},
		"New Situation":                   {"Let the hero react and find their footing", 2},
		"Change of Plans":                 {"Turn a vague hope into a goal we can see", 3},
		"Progress":                        {"Let the plan seem to work while the cost grows", 3},
		"Point of No Return":              {"Burn the hero's bridges", 4},
		"Complications and Higher Stakes": {"Make the goal harder as more rides on it", 4},
		"Major Setback":                   {"Make the goal look lost", 4},
		"Final Push":                      {"Make the hero risk everything", 5},
		"Climax":                          {"Settle the outer goal and the inner change", 5},
		"Aftermath":                       {"Show the life the journey earned", 1},
	},
	"Beats First": {
		"Act One":   {"Start the story: who, where, and what is wrong", 1},
		"Act Two":   {"Raise the cost of what the hero wants", 3},
		"Act Three": {"Bring it to its end, and show what changed", 4},
	},
}

func init() {
	for i, t := range structureTemplates {
		r := map[string]beatRange{}
		for name, v := range structureRanges[t.name] {
			r[sectionKey(name)] = v
		}
		structureTemplates[i].ranges = r
		a := map[string]beatAim{}
		for name, v := range structureAims[t.name] {
			a[sectionKey(name)] = v
		}
		structureTemplates[i].aims = a
	}
}

// structureTemplates are the structures, in the menus' order.
var structureTemplates = []structureTemplate{
	{name: "Three Acts", body: `# Act One

= Who, where, and what is wrong in their world.

## Inciting Incident [[beat]]
= Something happens that cannot be ignored.

## First Turning Point [[beat]]
= The hero commits, and there is no going back.

# Act Two

= Trying, failing, and the stakes rising.

## Midpoint [[beat]]
= A win or a loss that changes what the story is about.

## Second Turning Point [[beat]]
= Everything falls apart; the hero has to choose.

# Act Three

= The last confrontation, and what it costs.

## Climax [[beat]]
= The choice made in action.

## Resolution [[beat]]
= The world after, and who the hero has become.
`},
	{name: "Sequences", after: "Frank Daniel", body: `# Act One

= Two sequences: the world, and the problem that comes into it.

## Sequence 1: The World [[beat]]
= The hero's life as it is, and the first hint of trouble.

## Sequence 2: The Push [[beat]]
= The problem arrives; the hero is drawn in.

# Act Two

= Four sequences: the struggle, turning at the midpoint.

## Sequence 3: First Steps [[beat]]
= The first attempt at the goal, in a new situation.

## Sequence 4: Complications [[beat]]
= It is harder than it looked; the stakes rise to the midpoint.

## Sequence 5: New Direction [[beat]]
= After the midpoint, a new plan or a new understanding.

## Sequence 6: The Fall [[beat]]
= The plan fails; the lowest point.

# Act Three

= Two sequences: the last push, and what it leaves.

## Sequence 7: The Final Push [[beat]]
= The hero gathers what is left and acts.

## Sequence 8: Aftermath [[beat]]
= The outcome, and the new balance.
`},
	{name: "Hero's Journey", after: "Joseph Campbell and Christopher Vogler", body: `# Departure

= Leaving the known world.

## The Ordinary World [[beat]]
= The hero at home, and what is missing.

## The Call [[beat]]
= A challenge or a chance appears.

## Refusing the Call [[beat]]
= Fear, doubt or duty holds the hero back.

## Meeting the Mentor [[beat]]
= Someone gives advice, a gift or a push.

## Crossing the Threshold [[beat]]
= The hero leaves the known world.

# Initiation

= Trials in the unknown, and what they change.

## Tests, Allies, Enemies [[beat]]
= Learning the rules of the new world, and who is who.

## Approaching the Cave [[beat]]
= Getting ready for the greatest danger.

## The Ordeal [[beat]]
= Facing death, literal or not.

## The Reward [[beat]]
= What was won by surviving.

# Return

= Coming home, changed.

## The Road Back [[beat]]
= Turning for home, pursued.

## Resurrection [[beat]]
= A last test, where the change is proved.

## Return with the Elixir [[beat]]
= Home again, bringing something that changes it.
`},
	{name: "Story Circle", after: "Dan Harmon", body: `# Order

= The known world: wanting, and setting out.

## 1. You [[beat]]
= Someone at ease in the world they know.

## 2. Need [[beat]]
= Something is missing, and they want it.

## 3. Go [[beat]]
= They cross into a place they do not know.

## 4. Search [[beat]]
= They struggle there, and learn its ways.

# Chaos

= The unknown: getting, paying, coming back changed.

## 5. Find [[beat]]
= They find what they were after.

## 6. Take [[beat]]
= It costs them dearly.

## 7. Return [[beat]]
= They come back to where they began.

## 8. Change [[beat]]
= They are not who they were.
`},
	{name: "Kishōtenketsu", body: `# Ki: Introduction [[beat]]

= The characters and their world, as they are.

# Shō: Development [[beat]]

= The same world followed further; nothing breaks yet.

# Ten: Twist [[beat]]

= Something unexpected, perhaps unrelated, comes in.

# Ketsu: Reconciliation [[beat]]

= The parts brought together: the twist makes sense of what came before.
`},
	{name: "Beat Sheet (15 Beats)", after: "Blake Snyder", body: `# Act One

= The world before, and the push out of it.

## Opening Image [[beat]]
= The first picture: the hero's world as it is.

## Theme Stated [[beat]]
= Someone says what the story is really about.

## Set-Up [[beat]]
= The hero's life, flaws and what needs fixing.

## Catalyst [[beat]]
= The event that sets the story going.

## Debate [[beat]]
= Should the hero go? Doubt, a last chance to stay.

# Act Two

= The new world, its promise, and its cost.

## Break into Two [[beat]]
= The hero chooses, and enters a new world.

## B Story [[beat]]
= A second story, often a relationship, that carries the theme.

## Fun and Games [[beat]]
= The premise played out: the scenes the audience came for.

## Midpoint [[beat]]
= Things look won, or lost, and the stakes go up.

## Bad Guys Close In [[beat]]
= Pressure grows from outside and from within.

## All Is Lost [[beat]]
= The lowest moment; something or someone is lost.

## Dark Night of the Soul [[beat]]
= The hero takes in the loss.

# Act Three

= The answer, acted on.

## Break into Three [[beat]]
= An answer found, often through the B story.

## Finale [[beat]]
= The hero acts on what was learned and wins or loses.

## Final Image [[beat]]
= The world after: the opening image, changed.
`},
	{name: "Freytag's Pyramid", after: "Gustav Freytag (1863)", body: `# Exposition [[beat]]

= The people, the place and the mood, before anything moves.

## Inciting Moment [[beat]]
= Something sets the action going.

# Rising Action [[beat]]

= Complications pile up and the conflict tightens.

# Climax [[beat]]

= The peak, where the hero's fortunes turn.

## Tragic Force [[beat]]
= The event that starts the fall.

# Falling Action [[beat]]

= What the turn set loose gathers against the hero.

## Moment of Final Suspense [[beat]]
= A last hope that it may still come out otherwise.

# Catastrophe [[beat]]

= The struggle ends: ruin, or release.
`},
	{name: "Seven Key Steps", after: "John Truby", body: `# Weakness and Need [[beat]]

= What is wrong inside the hero, and what they must learn.

# Desire [[beat]]

= What the hero sets out to get.

# Opponent [[beat]]

= Who stands in the way, wanting the same thing.

# Plan [[beat]]

= How the hero means to win.

# Battle [[beat]]

= The last clash between hero and opponent.

# Self-Revelation [[beat]]

= The hero sees the truth about themselves.

# New Equilibrium [[beat]]

= Life settles again, higher or lower than before.
`},
	{name: "Six Stages", after: "Michael Hauge", body: `# Setup [[beat]]

= The hero's everyday life, and what is missing from it.

## Opportunity [[beat]]
= Something new comes in and starts things moving.

# New Situation [[beat]]

= The hero is somewhere new, getting the measure of it.

## Change of Plans [[beat]]
= A clear goal takes shape, one we can see.

# Progress [[beat]]

= The plan seems to work.

## Point of No Return [[beat]]
= The hero commits fully; going back is gone.

# Complications and Higher Stakes [[beat]]

= Harder now, with more to lose.

## Major Setback [[beat]]
= Everything goes wrong; the goal seems out of reach.

# Final Push [[beat]]

= With nothing left to lose, the hero risks it all.

## Climax [[beat]]
= The last obstacle faced, the goal won or lost.

# Aftermath [[beat]]

= The new life the journey made.
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
	return "[[" + t.name + by + ". Sections (#) are its parts, synopses (= lines) what happens in each, [[beat]] marks a part as a beat: " +
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

// insertStructure puts structure t in at the cursor's section boundary
// (structureInsertAt), one undo step, the cursor at its note.
func (w *MainWindow) insertStructure(t structureTemplate) {
	w.insertStructureText(structureText(t), t.name)
}

// insertStructureText puts the structure ins (sections, synopses, notes)
// in at the cursor's section boundary, one undo step, the cursor at its
// start; what names it in the status line.
func (w *MainWindow) insertStructureText(ins, what string) {
	if w.markdown || w.textEditor == nil || w.scriptAway("Insert Beats / Structure") {
		return
	}
	text := w.textEditor.Text()
	at, where := structureInsertAt(text, w.textEditor.CursorOffset(), w.sceneStarts())
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
	ins = pre + strings.TrimRight(ins, "\n") + "\n"
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
	w.setStatus(fmt.Sprintf("%s put in %s: write over its synopses", what, where))
}

// structureInsertAt is where a structure goes in, in runes, with the
// cursor at cursor (runes): at the cursor's section boundary, where no
// part of the script comes to be in another section: before the next
// top-level section (# …) after the cursor's line, or the one it is on;
// else at the end of the script's body (before the blocks at its end).
// where says it for the status line.
func structureInsertAt(text string, cursor int, starts []string) (at int, where string) {
	body := bodyEnd(text)
	runes := []rune(text)
	cursor = min(max(cursor, 0), len(runes))
	line := strings.Count(string(runes[:cursor]), "\n")
	top := map[int]string{} // the top-level sections, by their lines
	for _, it := range outlineOf(text, starts) {
		if !it.Scene && it.Level == 1 {
			top[it.Line] = it.Text
		}
	}
	n := 0 // line i's start, in runes
	for i, l := range strings.Split(text, "\n") {
		if n >= body {
			break
		}
		if name, ok := top[i]; ok && i >= line {
			return n, "before " + name
		}
		n += len([]rune(l)) + 1
	}
	return body, "at the end"
}

// templateMenu lists the structures, each doing do with it.
func templateMenu(label string, do func(t structureTemplate)) *fyne.Menu {
	var items []*fyne.MenuItem
	for _, t := range structureTemplates {
		items = append(items, fyne.NewMenuItem(t.name, func() { do(t) }))
	}
	return fyne.NewMenu(label, items...)
}

// templateCommands are New from Structure and Insert Beats / Structure.
func templateCommands() []*command {
	return []*command{
		{id: "file.newStructure", label: "New from Structure", icon: theme.DocumentCreateIcon,
			keywords: []string{"beats", "beat sheet", "structure template", "story structure", "plot outline"},
			tip:      "A new script laid out as a structure: three acts, sequences, a beat sheet…, its parts as sections and synopses",
			submenu:  func(w *MainWindow) *fyne.Menu { return templateMenu("New from Structure", w.newFromStructure) }},
		{id: "write.insertStructure", label: "Insert Beats / Structure", ex: "structure", icon: theme.ListIcon, when: screenplayOnly,
			keywords: []string{"beats", "beat sheet", "structure template", "story structure", "plot outline"},
			tip:      "A structure's beats (sections and synopses) put in after the cursor's section (before the next # section), one undo step",
			submenu:  func(w *MainWindow) *fyne.Menu { return templateMenu("Insert Beats / Structure", w.insertStructure) }},
	}
}
