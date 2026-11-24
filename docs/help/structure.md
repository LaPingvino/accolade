# Structure: Sections, Synopses and Notes

A script's structure is written in the script itself, in three kinds of lines that never print:

- **Sections**, lines starting with `#`: acts, sequences, chapters, whatever parts you think in. `##` is a part inside a part.
- **Synopses**, lines starting with `=`: what happens. A synopsis right under a heading, touching it (no blank line between), is that heading's own. Every other synopsis paragraph stands on its own: a beat, a moment you are planning. So the first beat after a blank line stays a beat.
- **Notes**, `[[like this]]`: questions, reminders, anything for you alone. A longer note goes in the Notes pane, or in a `/* … */` boneyard.

Everything below works on these lines. There is no separate board to keep in step with the script: the outline and the cards are views of the text, and each change you make in them is an edit of the text, undone in one step.

## Seeing it

- **Outline** (Ctrl+Shift+O): sections and scenes, each with its own synopsis under it. A synopsis standing on its own is a line of its own, `= …`, under the section or scene it is in. The filter finds words in synopses too. Tap a row to go there.
- **Cards** (Ctrl+Shift+D): a group for each section, with its synopsis on the header. A section's own synopses come first, as outlined cards marked `=`, then its scenes as index cards, then a card for a new scene. A synopsis standing on its own inside a scene gets a card after that scene's. Tap a card to go to its line.
- **Folding**: Fold or Unfold Here (Ctrl+Shift+., or Tab at a heading's start) folds the section or scene at the cursor; Shift+Tab goes from the sections alone, to all headings, to everything. `:fold-sections`, `:headings-only` and `:unfold-all` do the same from the `:` line, and in vim mode so do `za`, `zM` and `zR`.

## Moving it

The moves take what is at the cursor as one unit: a section with all it holds, a scene, or a paragraph, and a synopsis paragraph is a paragraph.

- **Alt+Up / Alt+Down** (`:move-up`, `:move-down`): past the one before or after it, of its kind.
- **Shift+Alt+Up / Shift+Alt+Down** (`:into-previous`, `:into-next`): a paragraph into the section or scene beside it. A section's own synopsis goes into the section before, after that section's own; or into the next one, before its own. A scene's paragraph goes into the scene before or after.
- **Alt+Left / Alt+Right** (`:promote`, `:demote`): a section a level up or down, a scene out of its section or into the next one.
- **Move… and Swap…** (`:move 3..5 after 9`, `:swap`): runs of scenes, and whole sections, by name.
- On the cards, drag a scene's card or a section's header.

## Growing it into scenes

New Scene… (Format menu, Write on the ribbon, `:new-scene`) with the cursor on a synopsis standing on its own puts the new scene's heading right above it: the synopsis becomes the scene's own, and the cursor waits after it for the first line of action. Anywhere else, the new scene goes after the scene you are in.

Fountain never closes a section or a scene: whatever you write after a heading belongs to it until the next heading. So once a beat has grown into a scene, the beats still below it are in that scene. They stay visible as lines of their own, in the outline and on the cards, and New Scene on each of them starts the next scene there.

## Printing

The PDF, Print, Word, ODT, HTML and Final Draft exports leave sections and synopses out, and notes too, so the structure can stay in the script while you write the scenes. To have it on paper or to share it, export Fountain (as written), or one of the text formats made with pandoc (Markdown, RTF, Org…), which keep it; Export Clean Fountain… asks whether to keep the sections and synopses.

## One way to work: from beats to script

This is Accolade's answer to a beat board: the beats are lines in the script, and they grow into it.

1. **Write the beats.** A section per act, a synopsis per beat:

   ```
   # Act One
   = The world before, and what is missing.

   = Mara finds her father's letter in the lamp room.

   = She decides to stay the winter.

   # Act Two
   = The storm, and what it takes.

   = The storm tears the roof from the boathouse.
   ```

   The synopsis touching each act's heading says what the act does; the ones after a blank line are the beats.
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
