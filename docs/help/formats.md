# File Formats

Accolade's own files are plain text: a screenplay is Fountain, a story or any other document is Markdown. Everything else is opened, imported or exported through them.

## Screenplays

- **.fountain** — the usual Fountain file. See [Fountain Reference](fountain.md).
- **.spmd** — Fountain under an older name, opened the same way.
- **.txt** — opened as a screenplay too.

Accolade saves a file the way it found it: a Windows-1252 file stays Windows-1252, a byte order mark and Windows line endings (CR LF) are kept.

A language code before the extension says the script's language, which sets the words a scene heading starts with: `scene.eo.fountain` is in Esperanto, `scene.nl.fountain` in Dutch. Edit > Script Language… sets it for a script without one in its name.

## Markdown documents

- **.md** and **.markdown** open as Markdown documents: Markdown colours, an outline of the headings, a preview beside the editor (Ctrl+Shift+P), and their own Export. File > New Markdown Document starts one.

View > Document Type > Screenplay or Markdown switches a file without renaming it; Accolade remembers the choice for that file. Renaming a file to .md or .fountain switches it too. Syntax: [Markdown Syntax](cheat-markdown.md).

## Save As and Rename

**File > Save As…** (Ctrl+Shift+S) first asks for the type, each with a line saying what it keeps: Fountain, Accolade, Highland, TextPack, TextBundle (desktop only), and Markdown — the other kind of document last, since it switches the document. The type the file has now is chosen; the recommended one is marked (Accolade once a screenplay has drafts or a lock, else Fountain; Markdown for a document). Then comes the system's save dialog, showing that type's files and giving the name its extension. In the browser the name is asked in the same window, and the file is downloaded.

**Renaming:** click the name over the editor, or File > Rename…. The rename editor has the name, the script's **Language** (it writes the `.eo` part of `film.eo.fountain`) and the **Type**, and shows the resulting name as you type. Typing a whole name with its extensions (`film.eo.md`) works too: the pickers follow it. Changing the type to or from a bundle converts the file; nothing is refused.

Both show **every file that belongs to the script** and where each goes:

- the script itself;
- its source: in the script, inside the bundle (`assets/source.md`), kept by Accolade, or beside it (`story.md`);
- its drafts and production files (`breakdown.json`, `schedule.json`, `revisions.json`): inside the bundle under `assets/`, kept by Accolade, or in `story.production/` beside it;
- another app's files inside a bundle (Highland's `sprints.json`, pictures).

They move together by what the new type needs: renaming `story.fountain` to `film.fountain` renames `story.md` to `film.md` and `story.production/` to `film.production/` (untick *Keep the source's own name* to leave `story.md` as it is); making it `film.accolade` moves them inside the bundle; making a bundle plain again asks where its drafts go (kept by Accolade, or in a folder beside it, with the source as `film.md`). Names a format fixes — `text.fountain`, `info.json`, `assets/…` — are shown as fixed. Save As copies these files; a rename moves them.

A rename is all or nothing: if any step fails, everything is put back as it was. **File > Undo Rename** takes the last rename back — the old files return exactly as they were, the new ones go — until another document is opened or the window is closed. A plain file cannot hold another app's files from a bundle: the list warns before a rename leaves them out, and Undo Rename brings them back.

To hand a script to someone else, use File > Export… (PDF, Final Draft, Fountain as written); the Save As window links there.

## Bundles

A bundle keeps the text and everything that goes with it in one file. All of them are TextBundles inside: an `info.json`, the text (`text.fountain` or `text.md`), and an `assets/` folder.

- **.accolade** — Accolade's own: a Highland bundle plus Accolade's drafts and production files under `assets/`. Save As recommends it once a script has drafts or a lock.
- **.highland** — Highland 2's file. Accolade opens and saves it, and keeps everything Highland put in it. A .highland that is not a zip is read as plain Fountain. Saving drafts or production files into a .highland works, and Accolade says once per file that Highland may not keep them.
- **.textpack** — a TextBundle zipped.
- **.textbundle** — a TextBundle folder. Open goes into the folder and opens its text. Desktop only.

Whether the text is a screenplay or Markdown comes from the text file's name inside.

On save, Accolade changes only the text, its own entry in `info.json` and its own files. Every other file in the bundle is copied as it was, so another app's data survives. A script's source can live in the bundle too, as `assets/source.md` (see [Adapting a Story](adapting.md#where-the-source-is-kept)).

### Saving a bundle as plain Fountain

Save As… or a rename to a .fountain asks where the drafts and production files go (in the list of the script's files):

- **kept by Accolade,** found again when you open the .fountain here;
- **in a folder beside it,** `Name.production/` (desktop only);
- **left out,** after asking a second time (Save As only: the bundle keeps them).

Saving or renaming a plain script as .accolade carries them in, with its source.

## Opening other apps' scripts

File > Open… (in the browser: Open from Your Computer…) also takes these. They open as a new, unsaved script; Save asks where to put the .fountain and leaves the original alone.

- **Final Draft (.fdx)** — the whole script, with scene numbers and colours, notes, revisions, locked pages and breakdown tags. Details in [Coming from Final Draft](final-draft.md#opening-final-draft-files).
- **PDF of a screenplay** — read back into Fountain by its layout (Lexington's own, Final Draft's, Highland's, Fade In's, Word's). A scanned PDF is read with OCR: Accolade asks what language the pages are in, and uses tesseract if it is installed, else its built-in OCR, which reads English. Check the result: a PDF only shows how a script looked.
- **Beat** — a Fountain file from Beat opens as it is; Beat's breakdown tags in its data block are read into the breakdown.

## Importing documents

File > Import as Markdown… turns a document into a Markdown document, ready to use as a story source:

- **Word (.docx)** and **OpenDocument (.odt)** — headings, lists, quotes, tables, bold, italic, strikethrough, links and footnotes; images are left out, with a comment where they were.
- **Web pages (.html, .htm, .xhtml).**
- **PDF** — read as prose, with OCR for scanned pages as above.
- **Text (.txt, .md, .markdown).**

File > Import with pandoc… (desktop, with pandoc installed) reads many more: EPUB, RTF, LaTeX, Org, reStructuredText, FictionBook, Textile, DocBook, Typst, and DOCX, ODT and HTML too.

## Exporting a screenplay

File > Export… (Ctrl+E) writes:

- **PDF** — the printed script.
- **HTML**, **Word (DOCX)**, **OpenDocument (ODT)** — every element its own paragraph style, to restyle in Word or LibreOffice.
- **Final Draft (FDX)** — see [Coming from Final Draft](final-draft.md#saving-for-final-draft).
- **Fountain (as written)** — a copy of the file; **Fountain (rewritten by Lexington)** — tidied, every element marked; **Plain text**.
- **LaTeX** and **Lexington's own (lex)**.
- With other programs installed: PDF through HTML (wkhtmltopdf), PDF through LaTeX, EPUB, RTF, Markdown, reStructuredText and other text formats (pandoc), Kindle and other e-book formats (Calibre). A format that needs a missing program says so.

The dialog's options: the script format and paper, the title page, scene numbers, colours from the scenes' colour notes, revision marks, only the scenes with a tag or colour, and one character's sides. FDX and LaTeX take a template.

File > Print… (Ctrl+P) prints through a web page, with the system's or the browser's print dialog. It has no revision marks and no locked pages; use a PDF for those.

## Exporting a Markdown document

In a Markdown document, File > Export… offers:

- **Web page (HTML)**, with footnotes;
- **Word (DOCX)** and **OpenDocument (ODT)**, with real lists and footnotes in the programs' own styles;
- **PDF**, made with Chromium if it is installed, else through the print dialog's Save as PDF;
- **Markdown (as written)** and **Plain text**;
- with pandoc on the desktop: EPUB, RTF, LaTeX, Org mode, and DOCX and ODT made by pandoc.

## Production files

- **Reports** (Production > Reports…): CSV, PDF, or printed.
- **Shoot days** (Production > Export Shoot Days (.ics)…): an iCalendar file for any calendar app.

See [Production](production.md).

## In the browser

The browser version is the same program, with these differences:

- **Files:** Open takes a file from the browser's file picker; Save, Save As and every export hand the file to the browser as a download. The script you are working on is kept in the browser, to come back to after a reload.
- **Formats:** .fountain, .spmd, .txt, .md, .markdown, .highland, .accolade, .textpack, .fdx and .pdf open. A .textbundle folder does not.
- **No OCR:** scanned pages of a PDF are left out; the desktop app reads them.
- **No other programs:** nothing that needs pandoc, Calibre, LaTeX or wkhtmltopdf. Import with pandoc… is not there.
- **PDF of a Markdown document** goes through the browser's print dialog.
- **No folder beside a file:** a bundle saved as .fountain keeps its drafts in Accolade's kept data, or leaves them out.
- **One window:** New Window is not there.

Formats that need no other program are offered, and their files are downloaded. Adapting and production work as on the desktop. For converting files without the editor, see [Lexington on the Command Line](lexington.md).
