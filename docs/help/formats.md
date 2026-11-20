# File Formats

Accolade's own files are plain text: a screenplay is Fountain, a story or any other document is Markdown. Everything else is opened, imported or exported through them.

## Screenplays

- **.fountain** — the usual Fountain file. See [Fountain Reference](fountain.md).
- **.spmd** — Fountain under an older name, opened the same way.
- **.txt** — opened as a screenplay too.

Accolade saves a file the way it found it: a Windows-1252 file stays Windows-1252, a byte order mark and Windows line endings (CR LF) are kept.

### A script in another language

A script's language sets the words a scene heading starts with: EN. and EKST. in Esperanto, BIN. and BUI. in Dutch (Lexington's settings, which you can change). INT. and EXT. are part of Fountain itself, so a script in another language is a dialect of Fountain: other Fountain apps read `EN. DOMO - TAGO` as action. Where the language is kept decides which one a file holds:

- **`film.eo.fountain`** — *Esperanto Fountain*: the name says the language, the text is as written, in Accolade's and Lexington's Esperanto keywords. Other Fountain apps won't see its scene headings.
- **`film.fountain`** — standard Fountain, for any app: a scene heading only the language knows gets Fountain's force mark (`.EN. DOMO - TAGO`), a line English would take for a heading but the language does not gets `!` (action) or `@` (a character), and the language is kept in a last line, `/* Language: eo */`, which no app prints. Accolade takes the marks and the line back when it opens the file, so you see the script as written; a mark you put on purpose stays. (A line inside a speech has no force mark: one starting INT. there is read as a heading by other apps.) A .fountain that says no language is read in the language set in the Preferences and saved exactly as it is: only a language the file says, or you choose (the rename editor, Edit > Script Language…, Save As), is written this way. Which words are "standard" is Lexington's setting (`Standard`, English unless changed), and every language's words come from its `[Scenes]` (Preferences > Script); Lexington's parser decides each mark, so a language you add there works the same way.
- **a bundle** (.accolade, .highland, .textpack, .textbundle) — the name never says the language; `info.json` does (Accolade's entry, `"language": "eo"`). The bundle holds the script twice from the same text: `text.eo.fountain` as written, the one Accolade reads, and `text.fountain` as standard Fountain for Highland and other apps, which ignore the other file. An .accolade may leave `text.fountain` out (*Keep a plain Fountain copy inside*, in the rename editor and Save As; on by default). If another app changed `text.fountain` since Accolade saved it, that change wins when you open the bundle: its force marks are taken back, the script as it was before is kept as a draft, and the status line says so (Collaboration > Compare… shows what changed). A bundle from before, named `film.eo.highland`, is still read in Esperanto.

A language of your own goes along in an .accolade, so the script reads the same on a machine without it (see [Settings that travel with an .accolade](#settings-that-travel-with-an-accolade)); a plain .fountain or another app's bundle does not take it along.

Each line runs the way its own letters say — Hebrew or Arabic right to left, Latin left to right — so a script that mixes them stays mixed; Format > Right to Left Line turns one line with a mark. Only a line with no letters of either kind (`…`, `2.`, a bare number) takes the script language's direction, from Lexington's `[Directions]` (`he = "rtl"`; Arabic and Hebrew are right to left unless changed, every other language left to right, and a language of your own can set it).

A Markdown document has no script language. Edit > Script Language… changes the language: a plain file's name follows (`film.eo.fountain`), a bundle is saved with the new one inside.

## Markdown documents

- **.md** and **.markdown** open as Markdown documents: Markdown colours, an outline of the headings, a preview beside the editor (Ctrl+Shift+P), and their own Export. File > New Markdown Document starts one.

View > Document Type > Screenplay or Markdown switches a file without renaming it; Accolade remembers the choice for that file. Renaming a file to .md or .fountain switches it too. Syntax: [Markdown Syntax](cheat-markdown.md).

## Save As and Rename

**File > Save As…** (Ctrl+Shift+S) first asks for the type, each with a line saying what it keeps: Fountain (and, for a script in another language, its dialect, e.g. Esperanto Fountain), Accolade, Highland, TextPack, TextBundle (desktop only), and Markdown — the other kind of document last, since it switches the document. The type the file has now is chosen; the recommended one is marked (Accolade once a screenplay has drafts or a lock, else Fountain; Markdown for a document). Then comes the system's save dialog, showing that type's files and giving the name its extension. In the browser the name is asked in the same window, and the file is downloaded.

**Renaming:** click the name over the editor, or File > Rename…. The rename editor has the name, the script's **Language** and the **Type**, and shows the resulting name as you type, with where the language will be kept. For a script in another language the types include its dialect, *Esperanto Fountain (.eo.fountain)*, beside standard *Fountain (.fountain)* (see [A script in another language](#a-script-in-another-language)); a bundle keeps the language inside and its name stays plain. Typing a whole name with its extensions (`film.nl.fountain`) works too: the pickers follow it. Changing the type to or from a bundle converts the file; nothing is refused.

Both show **every file that belongs to the script** and where each goes:

- the script itself, and where its language is kept (the name, a last line, or `info.json` with `text.eo.fountain`);
- its source: in the script, inside the bundle (`assets/source.md`), kept by Accolade, or beside it (`story.md`);
- its drafts and production files (`breakdown.json`, `schedule.json`, `revisions.json`): inside the bundle under `assets/`, kept by Accolade, or in `story.production/` beside it;
- another app's files inside a bundle (Highland's `sprints.json`, pictures).

They move together by what the new type needs: renaming `story.fountain` to `film.fountain` renames `story.md` to `film.md` and `story.production/` to `film.production/` (untick *Keep the source's own name* to leave `story.md` as it is); making it `film.accolade` moves them inside the bundle; making a bundle plain again asks where its drafts go (kept by Accolade, or in a folder beside it, with the source as `film.md`). Names a format fixes — `text.fountain`, `info.json`, `assets/…` — are shown as fixed. Save As copies these files; a rename moves them.

A rename is all or nothing: if any step fails, everything is put back as it was. **File > Undo Rename** takes the last rename back — the old files return exactly as they were, the new ones go — until another document is opened or the window is closed. A plain file cannot hold another app's files from a bundle: the list warns before a rename leaves them out, and Undo Rename brings them back.

To hand a script to someone else, use File > Export… (PDF, Final Draft, Fountain as written); the Save As window links there.

## Bundles

A bundle keeps the text and everything that goes with it in one file. All of them are TextBundles inside: an `info.json`, the text (`text.fountain` or `text.md`), and an `assets/` folder.

- **.accolade** — Accolade's own: a Highland bundle plus Accolade's drafts and production files under `assets/`, and the Lexington settings its script uses (below). Save As recommends it once a script has drafts or a lock.
- **.highland** — Highland 2's file. Accolade opens and saves it, and keeps everything Highland put in it. A .highland that is not a zip is read as plain Fountain. Saving drafts or production files into a .highland works, and Accolade says once per file that Highland may not keep them.
- **.textpack** — a TextBundle zipped.
- **.textbundle** — a TextBundle folder. Open goes into the folder and opens its text. Desktop only.

Whether the text is a screenplay or Markdown comes from the text file's name inside. A screenplay in another language than English keeps it in `info.json` and holds `text.eo.fountain` beside `text.fountain` (see [A script in another language](#a-script-in-another-language)).

On save, Accolade changes only the text, its own entry in `info.json` and its own files. Every other file in the bundle is copied as it was, so another app's data survives. A script's source can live in the bundle too, as `assets/source.md` (see [Adapting a Story](adapting.md#where-the-source-is-kept)).

### Settings that travel with an .accolade

A language or a script format of your own (Preferences > Script: Edit Lexington Settings…, New Format from This…) lives in your settings, so another machine would not know it. An .accolade takes along what its script uses, so it reads and prints the same anywhere:

- `assets/lexington/lexington.toml` — the script's language's scene words and matchup, which language is standard Fountain's (`Standard`) and its words, in full — even when they are Lexington's own, so the script reads the same on a machine whose words for that language were changed — and a format kept in your Lexington settings if it differs from Lexington's defaults;
- `assets/lexington/formats/name.toml` — the script's format, if it is one of your own, and any of your own it starts from (`Base`);
- `info.json` names the format (Accolade's entry, `"format": "my-radio"`): an .accolade keeps its own format, whatever the one chosen for new scripts.

Opening it, these are used for that script only, over your own settings: its own wins for it, other scripts and windows keep yours, and nothing in your settings changes. Saving it again takes them along again. If it brought something you do not have, or have otherwise under the same name, the status line says so.

**File > Add to My Settings…** (also on Preferences > Script while such a script is open) lists what the script brought and copies it into your settings. Something you have none of is added (untick it to leave it out). Something you have under the same name but otherwise shows both, yours and theirs, with a choice: **Keep mine** (the default: nothing of yours changes), **Take theirs** (replaces yours), or **Keep both** (theirs added under another name, `xq-2`, which this script then uses). Nothing is overwritten without that choice. Your lexington.toml is changed as text, its comments and the rest left as they are; if it is written in a way Accolade cannot add to safely, nothing is changed and you are told so.

Only an .accolade carries them: a .highland, .textpack, .textbundle or plain .fountain made from it leaves them out (the script there reads your settings), and Save As or a rename to one of those writes the file anew without them. In the browser, your settings are the ones it keeps.

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
