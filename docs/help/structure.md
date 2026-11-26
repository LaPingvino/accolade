# Structure: Sections, Synopses and Notes

A script's structure is written in the script itself, in three kinds of lines that never print:

- **Sections**, lines starting with `#`: acts, sequences, chapters, whatever parts you think in. `##` is a part inside a part.
- **Synopses**, lines starting with `=`: what happens. The first synopsis under a heading (blank lines may come between) is that heading's own. Every other synopsis paragraph stands on its own: a beat, a moment you are planning.
- **Notes**, `[[like this]]`: questions, reminders, anything for you alone; `[[todo: …]]` is one still to do. A longer note goes in a `/* … */` boneyard, one about the whole script in the unsorted notes.

Everything below works on these lines. There is no separate board to keep in step with the script: the outline and the cards are views of the text, and each change you make in them is an edit of the text, undone in one step.

## Seeing it

- **Outline** (Ctrl+Shift+O): sections and scenes, each with its own synopsis under it. A synopsis standing on its own is a line of its own, `= …`, under the section or scene it is in. The filter finds words in synopses too. Tap a row to go there.
- **Cards** (Ctrl+Shift+D): a group for each section, with its synopsis on the header. A section's own synopses come first, as outlined cards marked `=`, then its scenes as index cards, then a card for a new scene (a New scene row while it has no scenes). A synopsis standing on its own inside a scene gets a card after that scene's. Tap a card to go to its line.
- **Folding**: Fold or Unfold Here (Ctrl+Shift+., or Tab at a heading's start) folds the section or scene at the cursor; Shift+Tab goes from the sections alone, to all headings, to everything. `:fold-sections`, `:headings-only` and `:unfold-all` do the same from the `:` line, and in vim mode so do `za`, `zM` and `zR`.

## What is still open: the Notes pane

The script is its own to-do list. **View > Notes** lists, in the script's order and under the scene or section each is in:

- **To do**: every `[[todo: …]]` (or `[[TODO …]]`) note; `[[done: …]]` and `[[todo: [x] …]]` are done ones.
- **Cut**: a `/* … */` boneyard that reads as script (a scene heading, a speech, a transition), named by its first heading. A scene omitted in a locked script is one too.
- **Note**: every other `[[note]]`, and a boneyard of prose. Notes another tool reads (a colour, a link to the source, a storyboard frame, a heading's tags) are left to it.
- **Beats**: a synopsis no scene grew from yet: one in a section before its scenes, one at a scene's end, one before the first heading (Unplaced).
- **Revision**: a finding kept, `[[revision: …]]` (Keep writes it; see below).
- **Checks**: what the checks find, worked out from the text after each pause and never stored: a heading without INT./EXT. or a time of day, a `/*` or `[[` not closed, one character under two cues, a very long scene, and Lexington's linter (a parenthetical with no speech, a cue without a name, a dual dialogue gone wrong). Fix the text and the finding is gone.
- **Unsorted notes**, last: notes about the whole script, kept in the notes block at its end.

Tap one to go there. Below the list you edit it, and Save (Ctrl+Enter, or leaving the field) writes it into the text, one undo step; emptied, or with the bin, it goes. A boneyard has **Restore**: its marks go and its inside is script again (prose comes back as action; a scene omitted in a locked script comes back with its number, as Production > Restore Scene does). A beat has **New Scene**, which grows it into a scene there.

**Add Note** (Ctrl+Shift+K, `:note`) writes `[[ ]]` at the cursor to type in; **Unsorted** (`:unsorted-note`) adds a note about the whole script. The filters at the top show or hide each kind, with how many there are. The status bar's **N open** counts the to-dos and the beats without a scene; tap it for the pane. The outline's and the cards' section headers say **N beats without a scene** when they have some: tap it to step through them. Notes, cuts and to-dos have a quiet mark in the editor's margin.

Nothing of this is kept apart from the text: an item is done when the text no longer has it.

## Going through it: the Revision tab

The ribbon's **Revision** tab steps through the same items as Find steps through matches: **Next** (F8, `:next-item`) and **Previous** (Shift+F8) go from the cursor, round at the ends, and the status bar says what each one is. The item the cursor is in is the current one; the readout beside them says which of how many. **Show** is the pane's filters (one for both, and for the margin's marks): turn Checks off to write without them.

The current item's actions: **Keep** writes a finding into the script as `[[revision: …]]` at its line's end, saved with it and listed as a revision note (the finding it was is then left out); a finding a note on its line would change (two cues for one character, a `/*` or `[[` not closed, a stray `*/` or `]]`) is fixed in the text instead; **Dismiss** puts a finding out of sight, kept so when the file is opened again: beside the document by default (in a bundle; for a loose file a `Name.fountain.dismissed` file next to it; not saved yet, with the files Accolade keeps for it), or, with **All at Once > Dismissed in Document** on, as a paragraph in the document's notes block (switching moves them; deleting a line brings a finding back either way), and takes a `[[note]]`, to-do or revision note out of the text; **Done** turns `[[todo: …]]` into `[[done: …]]` and back; **Restore** brings a cut back; **New Scene** grows a beat into one; **Edit** opens it in the Notes pane. **All at Once**: **Remove Done To-dos** and **Restore Cuts in This Scene**, each asking with the count first and each one undo step.

**Beats: where they land.** For every `#` section: the page it lands on (the next printed line's, in the layout the PDF and the page numbers use, the title page not counted) and its place in the script, as a percentage of the pages. When the sections are a structure's (by their names: Three Acts, Sequences, Hero's Journey, Story Circle, Kishōtenketsu, the 15 beats), each gets that structure's usual range, and **Next Off** / **Previous Off** step to the ones outside it with what to do: "Midpoint comes late: page 72 of 110 (65%), usually 49–53% (films: median 50%, 49–53%, as the midpoint; O'Brien 2026, 10 films): what before it could go or be shorter?". The list under the readout goes to any of them; the Cards' section headers, the timeline's bands and the outline's rows carry the same mark. Each range says whether it is **measured** (where real films put that beat, from a published study, named with its numbers) or the **convention** (the range the method and the manuals give, where no study times that beat). A section counts as off only when it is more than a page outside its range, since its place is counted by the page it lands on. Your own structure gets the pages and percentages without a range.

**Where films put the beats.** The measured ranges come from two published studies, quoted as they give them; Accolade adds no numbers of its own:

- **Thompson 1999, 73 films.** Kristin Thompson, *Storytelling in the New Hollywood* (Harvard University Press, 1999), timed the large parts of 73 Hollywood films; her figures as tabulated by James E. Cutting, "Narrative theory and the dynamics of popular movies", *Psychonomic Bulletin & Review* 23(6), 1713–1743 (2016), doi:10.3758/s13423-016-1051-4. The setup runs 24.9% of the running time (σ 2.8), the complication 26.5% (σ 3.7), the development 25.1% (σ 3.5), the climax 23.5% (σ 3.8); epilogs, in 83% of the films, 1.8% (σ 1.6). So the act-one break falls at 25% (σ 2.8), the act-two break at 76.5% (σ 3.8), the epilog at 98% (σ 1.6). Accolade's range is two σ either side (about 19 films in 20): act one ends 19–31%, act three starts 69–84%.
- **O'Brien 2026, 10 films.** Eoin O'Brien, "Narrative Structure in Highly Rated Hollywood Cinema: An Empirical Study of Three-Act Plot Points", *Proceedings of the RSU International Research Conference 2026* (Rangsit University), 160–168: seven plot points timed in the ten highest-rated films on IMDb (September 2025). Accolade uses the median and the lowest to the highest film from its per-film table (its summary table's means and SDs do not all agree with that table): inciting incident median 10.5%, 4–17%; midpoint median 50%, 49–53%; low point median 69%, 66–74%; climax median 91.5%, 82–94%. Ten films, one person's reading of them: a narrow sample.

Which beats are measured: **Three Acts**, all of them (Inciting Incident, the two turning points and the acts they open, Midpoint, Climax, Resolution as the epilog). The **15 beats**: Catalyst (as the inciting incident), Break into Two and Act Two, Midpoint, All Is Lost (as the low point), Break into Three and Act Three; the rest are the convention. **Sequences**: the act breaks and Sequence 5 (from the midpoint); the rest convention. **Hero's Journey**, **Story Circle** and **Kishōtenketsu**: no study times their stages, so all convention.

Two things to keep in mind. The studies time films in minutes, Accolade places sections by pages: a page runs about a minute, a little under (Stephen Follows, 2026, matched 2,520 scripts against their films' running times: about 55 seconds a page), so the shares come out about the same. And the studies time the event, Accolade the section's start: start a section where its event happens and the two agree.

## Moving it

The moves take what is at the cursor as one unit: a section with all it holds, a scene, or a paragraph, and a synopsis paragraph is a paragraph.

- **Alt+Up / Alt+Down** (`:move-up`, `:move-down`): past the one before or after it, of its kind.
- **Shift+Alt+Up / Shift+Alt+Down** (`:into-previous`, `:into-next`): a paragraph into the section or scene beside it. A section's own synopsis goes into the section before, after that section's own; or into the next one, before its own. A scene's paragraph goes into the scene before or after.
- **Alt+Left / Alt+Right** (`:promote`, `:demote`): a section a level up or down, a scene out of its section or into the next one.
- **Move… and Swap…** (`:move 3..5 after 9`, `:swap`): runs of scenes, and whole sections, by name.
- On the cards, drag a scene's card, a section's header, or a synopsis card: a synopsis dropped on another goes before it (on its right half, after it); on a scene's card, before the scene (on its right half, into it); on a section's header, first in that section.

## Growing it into scenes

New Scene… (Format menu, Write on the ribbon, `:new-scene`) with the cursor on a synopsis standing on its own puts the new scene's heading right above it: the synopsis becomes the scene's own, and the cursor waits after it for the first line of action. Anywhere else, the new scene goes after the scene you are in.

Fountain never closes a section or a scene: whatever you write after a heading belongs to it until the next heading. So once a beat has grown into a scene, the beats still below it are in that scene. They stay visible as lines of their own, in the outline and on the cards, and New Scene on each of them starts the next scene there.

## Printing

The PDF, Print, Word, ODT, HTML and Final Draft exports leave sections and synopses out, and notes too, so the structure can stay in the script while you write the scenes. To have it on paper or to share it, export a Treatment (PDF or Word: the structure alone, see below), an Outline (OPML), Fountain (as written), or one of the text formats made with pandoc (Markdown, RTF, Org…), which keep it; Export Clean Fountain… asks whether to keep the sections and synopses.

## OPML: Outlines In and Out

OPML is the file outliners such as Workflowy, OmniOutliner and Dynalist export and open. Accolade reads and writes it, so a plan made in an outliner becomes the script's structure, and the structure goes back out to one.

- **Out:** File > Export Outline (OPML) — for Workflowy, OmniOutliner…, or Export… and the format of that name, or `:export opml`. Each section is an item holding what is in it, each scene's heading an item, a heading's own synopsis its note, and each synopsis or note standing on its own an item where it is. The scenes' own text (action, dialogue) stays in the script: an outline holds only the plan.
- **In:** File > Open an outline (`.opml`; one saved under another name is known by what is in it). It opens as a new script, not yet saved: the items nested as sections (`#`, then `##` inside them), an item's note as its synopsis touching its heading, the items with nothing under them as beats, and an item that reads as a scene heading as a scene. Save keeps it as a script, never over the outline.
- **Paste as Structure** (Edit menu, the palette): an outline copied from an outliner, as indented lines or as OPML, put into the script at the cursor's section boundary, as Insert Structure puts a structure.
- **Round trip:** Accolade's own outline, opened again, gives the same sections, synopses and notes, as written.

To read rather than outline: File > Export Treatment (PDF, Word)… writes the same structure as a document: the sections as headings, the scenes' headings under them, each synopsis and beat as a paragraph, the notes in italics.

## One way to work: from beats to script

This is Accolade's answer to a beat board: the beats are lines in the script, and they grow into it.

1. **Write the beats.** A section per act, a synopsis per beat:

   ```
   # Act One

   = The world before, and what is missing.

   = Mara finds her father's letter in the lamp room.

   = She decides to stay the winter.

   # Act Two

   = The storm tears the roof from the boathouse.
   ```

   The first synopsis under each act says what the act does; the ones after it are the beats.
2. **See them** in the outline and on the cards.
3. **Rearrange** with Alt+Up and Alt+Down; take a beat into the next act with Shift+Alt+Down; move whole acts with Alt+Up and Alt+Down on their heading.
4. **Fold** the acts you are not working on.
5. **Grow a beat**: the cursor on it, New Scene…, write the scene. Its second scene: New Scene… again, from inside the first.
6. **Print**: the beats stay in the script, out of the PDF.

There are other ways to use the same tools. Write scenes first and add sections around them later, to see the shape you have. Keep the plan in notes (`[[ ]]` or the Notes pane) and turn the ones that hold into synopses. Or skip the planning: nothing here has to be filled in.

## Starting from a structure

File > New from Structure starts a script laid out in one of these. Insert Structure (Format menu, the palette, `:structure name`) puts one into the script you have, in one undo step: after the section the cursor is in, before the next `#` section (on a `#` heading, before it; with none after it, at the end), so nothing already there comes to be in another section. Each one is sections and synopses with a note on how to use it, nothing more: write over the synopses, delete what you don't need, and grow the rest into scenes.

- **Three Acts** — the acts, with their turning points, midpoint and climax as `##` parts.
- **Sequences** (after Frank Daniel) — eight sequences in three acts.
- **Hero's Journey** (after Joseph Campbell and Christopher Vogler) — twelve stages in departure, initiation and return.
- **Story Circle** (after Dan Harmon) — eight steps, from order into chaos and back.
- **Kishōtenketsu** — four parts: introduction, development, twist, reconciliation.
- **Beat Sheet (15 Beats)** (after Blake Snyder) — fifteen beats in three acts.
- **Beats First** — three acts, each with a line for what it does and three empty beats.

See also [Fountain Reference](fountain.md) for the syntax, [Keyboard Shortcuts](shortcuts.md) and [Ex Commands](ex.md).
