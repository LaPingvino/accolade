# Fixing an Imported Script

A script read from a PDF, a scan or another app's file comes out as well as Accolade can read it, but older and unusual layouts have their own ways: headings written "Scene 6 - int. day -", names in mixed case with the parenthetical on the same line, a typescript's double spacing, page numbers in the text. Instead of fixing them one at a time, fix every line of a kind at once.

## Fix Lines Like This

Put the cursor on one line of the kind and choose **Edit > Fix Lines Like This…** (also in the editor's right-click menu, the Adaptation menu, and on the ribbon under Edit > Fix-Up). Right after an import from a PDF or Final Draft, the ribbon has a **Fix-Up** tab in front, until the script is saved or you choose Done Fixing: Fix Lines Like This…, Apply the Kept Rules Again…, and for a PDF Read with OCR Again… and Check for Garbled Text, which says how many pages are scans and how many have text that looks garbled.

1. **Line like this** is the cursor's line. Change it to start from another.
2. **Pattern** is the shape Accolade proposes for it, in plain words: `Scene <number> - <text>` for "Scene 6 - int. day -". It is matched against each whole line, its marks (`!`, `.`, `@`, `>`) taken off, words in any case, any dash for a dash. Edit it: the count under it and the lines lit in the script follow as you type.
3. **Make them** is what the matching lines become, proposed from the line:
   - **Scene headings**, written from **Result** (`<text> #<number>#` makes "Scene 6 - int. day -" into `INT. DAY #6#`); in capitals, with a point in front when the heading starts with no INT or EXT, as Fountain needs.
   - **Character cues**: the name in capitals, a parenthetical after it put on its own line (an extension, V.O. or CONT'D, stays with the name), the blank line between the name and its speech taken out and the speech's centring with it.
   - **Split glued cues NAME(paren)**: the same, for `THE WOMAN(very calmly)`.
   - **Transitions**, in capitals (`>` in front unless it ends in TO:).
   - **Narration (V.O.)**: a speech without a name gets one, from Result (`NARRATOR (V.O.)` at first).
   - **Drop them (page debris)**: page numbers, (MORE), a running title; one blank line is left where they stood.
   - **Join across blank lines**: a paragraph or a speech typed double-spaced made one again. The line and the next after a blank line are joined while the line does not end a sentence (`<unfinished>`) and the next is not a heading, a cue or a transition.
   - **Join words broken by hyphens**: "sol- diers" as an import joins its lines; the script's own words say how: a word it has whole elsewhere is joined, one whose second part it has as a word of its own keeps its hyphen (self-portrait).
   - **Collapse runs of spaces**: words spread by a justified typescript, one space apart again.
4. **Preview** lists each change, the line as it is and what it becomes. Nothing changes until **Apply**, and one Undo takes all of it back.

The parts a pattern can have: `<number>` (12, 12A), `<int/ext>` (INT, EXT, INT./EXT., I/E), `<CAPITALS>` (words in capitals), `<Name>` (one to four capitalised words, Mrs. Schultz), `<unfinished>` (text that does not end a sentence), `<spaces>` (two spaces or more) and `<text>` (anything). In Result, a part is written as in the pattern, a second `<text>` as `<text 2>`, and `<line>` is the whole line.

Under **Pattern parts, regular expression** the pattern is shown as a regular expression; written by hand, it is used instead (its groups go in Result as `<1>`, `<2>`, or by their names). One that does not read says so and changes nothing.

## The rules stay with the script

With **Keep the rule with the script** ticked (as it starts), each rule applied is kept in the script's `/* Fix-Up */` block at its end, one a line:

```
/* Fix-Up
heading: Scene <number> - <text> → <text> #<number>#
cue: VOICE OVER → NARRATOR (V.O.)
join: <unfinished>
*/
```

Fix Lines Like This… lists them: tap one to use it again, or **Apply Them All Again…** to run them all in order, after a count of what each changes, in one undo step. When you import the same file again (File > Open of the PDF, with this script in the window), the new import keeps the rules and Accolade asks whether to apply them; it never applies them by itself. Export Clean Fountain leaves the block out, as Accolade's other blocks.

## What the PDF import does by itself

Some of this is read on import already, where it is sure: headings numbered "Scene 12 - …" (the number kept as the scene number) or with the number on the line above; a name with a blank line before its speech, a short speech centred under it, a parenthetical under the name wherever it stands; a name at its speech's indent, or glued to its parenthetical; a scan's stray marks between the lines, which no longer make every line a paragraph of its own; a word broken at a line's end, joined where the script has the whole word elsewhere. Where it is not sure, the line is left as it was printed, for the fixer.

A scan that came with a text layer of its own (made by the scanner's OCR) is read from that layer. If that text looks garbled — stray marks at the lines' ends, words of one line at two heights — Accolade asks first whether to read the scanned pages with OCR instead (Read with OCR), or to keep the text as it is (Keep the Text); it never does so on its own. Either way, what is left is what the fixer is for.
