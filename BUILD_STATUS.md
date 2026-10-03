# Build Status

What works in the Fyne version of Accolade, what does not yet, and how to
build and test it. Last reviewed October 2026.

## Building

Accolade needs Go (see `go.mod`), a C compiler and the usual OpenGL/X11
development libraries for Fyne (see the README).

```bash
CGO_ENABLED=1 go build -o accolade .
```

**Lexington checkout:** `go.mod` currently has

```
replace github.com/LaPingvino/lexington => ../lexington
```

because the lexington fixes Accolade relies on (title pages, forced
transitions, FDX, HTML escaping) are not in a tagged release yet. Until
they are, clone lexington next to this repository
(`git clone https://github.com/LaPingvino/lexington ../lexington`). Once
lexington is tagged, replace the `replace` line with the new version.

## Testing

```bash
go test ./...          # headless, uses fyne.io/fyne/v2/test
go test -race .
```

The tests drive real widgets: they type screenplays with Enter, use the
menus and search bar, export PDF/HTML/FDX and read them back. PDF content
is checked with `pdftotext` when it is installed.

## Features

| Feature | Status | Notes |
|---------|--------|-------|
| Editing | Works | Screenplay font (Courier Badi), size from preferences |
| Formatting on Enter | Works | Scene headings uppercased; character, parenthetical, dialogue and transition indents; undoable; off with "auto-indent" |
| Find / replace | Works | Case, whole word, regex with `$1` groups; no highlight-all |
| Undo / redo | Works | Fyne's Entry undo; Ctrl+Z, Ctrl+Y, Ctrl+Shift+Z |
| Keyboard shortcuts | Works | On the main menu (File/Edit/View/Help) |
| Auto-save | Works | Interval from preferences; atomic writes |
| Title page dialog | Works | Generates and replaces Fountain title pages |
| Export PDF / HTML | Works | Via lexington |
| Export / open FDX | Works | Via lexington; opening converts to Fountain |
| Export TXT / Fountain | Works | |
| Themes | Works | system, light, dark, sepia |
| Preview pane | Partial | Basic rendering |
| Export DOCX | Not yet | Removed from the export dialog until implemented |
| Spell check | Not yet | `spell_checker.go` is a stub |
| Focus mode | Not yet | Menu/toolbar action logs only |
| Syntax highlighting, line numbers | Not yet | Needs a custom editor widget (see below) |
| File drop, file type filters in dialogs | Not yet | |

## Known limitations

- Fyne's text Entry has no API to select text or highlight ranges.
  `editor_cursor.go` works around this with cursor fields and key events.
  A dedicated editor widget (or upstream Fyne support) is the long-term
  fix and would also allow syntax highlighting and line numbers.
- Undoing a formatted or replaced line takes a few steps, because Fyne
  records typing over a selection word by word.
- `lexington_converter.go` still shells out to a `lexington` binary and
  is not used by the app; the built-in lexington library is.
