# Build Status

What works in the Fyne version of Accolade, what does not yet, and how to
build and test it. Last reviewed October 2026.

## Building

Accolade needs Go (see `go.mod`), a C compiler and the usual OpenGL/X11
development libraries for Fyne (see the README).

```bash
CGO_ENABLED=1 go build -o accolade .
```

Lexington (Fountain parsing and PDF/HTML/FDX output) comes from its
v1.3.0 release through Go modules.

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
| Editing | Works | Accolade's own editor (internal/editor) on a Fyne TextGrid: wraps at the window width, screenplay font and size from preferences |
| Formatting on Enter | Works | Scene headings uppercased; character, parenthetical, dialogue and transition indents; one undo step; off with "auto-indent" |
| Find / replace | Works | Case, whole word, regex with `$1` groups; all matches highlighted; each replace (and Replace All) one undo step |
| Undo / redo | Works | By word while typing; Ctrl+Z, Ctrl+Y, Ctrl+Shift+Z |
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
| Fountain colouring | Works | Scene headings, characters, parentheticals, transitions, notes |
| Line numbers | Works | "Show line numbers" preference; numbers on the first row of each line |
| File drop, file type filters in dialogs | Not yet | |

## Known limitations

- The editor is monospace only (screenplays are), and always wraps at
  the window width; the old word-wrap setting no longer applies.
- Double-width characters (CJK) are counted as one column.
- `lexington_converter.go` still shells out to a `lexington` binary and
  is not used by the app; the built-in lexington library is.
