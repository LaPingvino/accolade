# An editor widget for Accolade

Status: proposal (October 2026). Recommends Accolade's own editor widget
built on Fyne's `TextGrid`, in the steps listed at the end.

## Why

Accolade edits screenplays in Fyne's `widget.Entry`. That works for typing,
but several features need things Entry does not offer publicly:

| Need | With Entry today |
|---|---|
| Select a search match | No API: `editor_cursor.go` moves the cursor with a binary search over wrapped rows and selects with simulated Shift+Right key events |
| Highlight all matches | Impossible: Entry draws one selection only |
| Fountain syntax colouring | Impossible: Entry renders a single text segment |
| Line numbers | Impossible to align: row geometry is not exposed |
| One-step undo for a replace or a formatted line | Fyne records typing over a selection as erase plus insert, word by word |
| Formatting without fighting the widget | Reformatting must go through simulated typing to stay undoable |

## Options

### A. Fork Fyne's Entry

Copy `widget.Entry` into Accolade and add `Select(start, end)`, highlight
ranges and styled segments.

- Entry (2,177 lines) works through unexported methods of `widget.RichText`
  (row boundaries, `insertAt`, `deleteFromTo`) and Fyne-internal packages
  (`internal/cache`, `internal/widget`, `internal/painter`), which code
  outside the Fyne module cannot import. A fork therefore has to take
  `entry.go`, `selectable.go` and `richtext.go` with their internals:
  about 4,000 lines, BSD-3 licensed (fine to copy with the notice).
- It keeps word wrapping, clipboard, IME-composed input and accessibility
  as Fyne has them, but every Fyne upgrade means re-applying the fork.
- Syntax colouring still needs the RichText segment model reworked.

### B. Accolade's own widget on `TextGrid` (recommended)

`widget.TextGrid` is public, renders a grid of cells with per-cell
foreground/background styles (`SetStyle`, `SetStyleRange`), has built-in
`ShowLineNumbers` and scrolling, and maps between positions and cells
(`CursorLocationForPosition`, `PositionForCursorLocation`). It has no text
input and no wrapping; those would be Accolade's code.

A screenplay suits a grid: it is set in a fixed-width font in fixed columns
(Accolade already lays out character names at 37 spaces, dialogue at 25,
transitions at 60). Selection, search highlights, Fountain colours and line
numbers all become cell styles.

Accolade would write: the text model (lines, cursor, selection, edits with
undo groups), soft wrapping into grid rows, keyboard input, mouse selection
and clipboard. Estimated 1,200-1,800 lines, all on public Fyne API and
testable with `fyne.io/fyne/v2/test`.

Costs: the editor is monospace only (right for screenplays, and the
current editor already uses a monospace font), and input-method
pre-edit display is not available (Fyne does not offer it in Entry
either; composed characters arrive through `TypedRune`).

### C. Wait for upstream Fyne

- fyne-io/fyne#1395 "Consistent Selection API" (open since 2020, 32
  comments) is about list/select widgets, not text ranges.
- fyne-io/fyne#4854 "Add a lower-level EditableText widget" (open since
  May 2024) is the closest: maintainers (andydotxyz, dweymouth) welcome a
  lower-level editable text component that could support line numbers and
  custom editors, but nothing is scheduled.

A small upstream PR adding `Entry.Select(start, end int)` and
`Entry.SelectionRange()` would remove the `editor_cursor.go` workaround
and could be proposed regardless, referencing #4854. It would not give
highlighting, colouring or line numbers.

## Recommendation

Build **B**, keep Entry as the editor until B is at least as good, and
propose the small selection API upstream (C) independently. If Fyne lands
an `EditableText`, B's model and wrapping can move onto it.

## Steps

Each step is one reviewable change with tests.

1. **Text model** (`internal/editor/buffer`): lines as rune slices, cursor
   and selection as offsets, insert/delete/replace with undo groups
   (one undo step per replace or formatted line), redo. Pure Go.
2. **Soft wrapping**: logical lines to visual rows for a width in columns,
   breaking at spaces, keeping leading indentation on continuation rows,
   with offset <-> (row, column) mapping. Pure Go.
3. **Widget, keyboard**: a `ScriptEditor` widget rendering wrapped rows
   into a `TextGrid`; focus; typed runes; arrows, Home/End, Page Up/Down
   (with Shift for selection); Backspace/Delete; Enter; clipboard and
   select-all shortcuts; undo/redo. `fyne/test` tests.
4. **Mouse**: tap to place the cursor, drag to select, double tap to
   select a word; scrolling follows the cursor.
5. **Styles**: selection, all search matches (`SetHighlights`), Fountain
   element colours (scene heading, character, dialogue, parenthetical,
   transition, notes) from Accolade's element detection; theme-aware.
6. **Accolade integration** behind a Preferences option ("New editor"):
   search and replace use `Select`/`ReplaceRange` (single undo step),
   formatting on Enter uses `ReplaceRange`, line numbers follow the
   preference. `editor_cursor.go` goes once Entry is no longer used.
7. **Switch the default** after the new editor has been used for real
   writing; remove the Entry-based editor.
