# TextBundle and Highland files: what is known

Collected 2026-11-19 for LOOP.md's TextBundle / .highland item. Highland has published no spec, so this is pieced
together from the developer's pages and from other open-source readers. Real Highland files are still wanted, as the
FDX work had them: put them in testdata/highland (never committed), where the tests pick them up.

## Sources

- The developer's pages: https://quoteunquoteapps.com/highland-2/highland-format and the Highland Pro page.
  A .highland file is a zipped, customised TextBundle holding:
  - the text, as .fountain, .markdown or .txt;
  - info.json;
  - assets/.

  The developer promises to stay compatible with TextBundle.
- A teardown of Highland 2.app (MIT): https://github.com/adamburnbox-beep/Passage-Screenwriting-Software---2,
  docs/reference/HIGHLAND2_TEARDOWN.md. It was made by static analysis of the app and its bundled samples.
- Readers to compare against (MIT):
  - nattiini45/caotang `packages/import-highland/src/highland.ts`;
  - intrusive-memory/SwiftCompartido `GuionParsedScreenplay+Highland.swift`;
  - ChrisTempel/CineSched `HighlandArchiveReader.swift`.

  The TextBundle spec is at http://textbundle.org (`org.textbundle.package`).
- TextBundle's own reference code and samples (MIT): https://github.com/shinyfrog/TextBundle,
  `TextBundleTests/Sample TextBundles/`. The `.textpack` sample and three small folder samples are in
  testdata/textbundle with the licence (taken at commit 28027bf); the sample with a 160 KB JPEG is left out.

## The layout, and where readers disagree

- **The teardown:** the zip holds one folder, `Name.textbundle/`, with the bundle's files inside it:
  - `text.fountain` (`text.md` for Markdown);
  - `info.json`;
  - `characters.json`, `navigatorFilters.json` and `sprints.json`;
  - `assets/`.

  The binary also names `characterHighlighting.json`, plus revisions, milestones, a bin, a scratchpad and includes,
  stored the same way.
- **caotang:** looks for `text.fountain`, `text.markdown` or `text.txt` at the zip's root, in that order.
- **TextBundle itself (checked 2026-11-19):** the spec says nothing about the layout inside a `.textpack`. Its
  own sample `.textpack` holds `Textbundle Example.textbundle/…`. Its writer (`TextBundleWrapper+TextPack.swift`)
  zips a `<name>.textbundle` folder, and its reader opens the first item it unzips as the bundle, so a
  root-layout `.textpack` would fail in it.
- **What Accolade does:** accepts both layouts, files at the root or inside a single top-level folder (the
  `*.textbundle/` one if there are several). It ignores macOS's `__MACOSX/` and `.DS_Store` when deciding, but
  copies them through. It writes back in the layout it read. A new `.highland` *and* a new `.textpack` get the
  folder layout, `Name.textbundle/`: that is what Highland's samples and TextBundle's own writer, sample and
  reader agree on. Save As to another zip keeps the layout, with the folder renamed for the new file.
- **Plain-text files named .highland:** some repositories' "samples" (intrusive-memory's fixtures) are plain Fountain
  text with a .highland name. Highland itself may never write such files, but any file that isn't a zip is read as
  plain text.

## info.json (the teardown's real example)

```json
{
  "com.quoteunquoteapps.highland2": { "lastScrollPosition": 190.5, "cursorPosition": 0,
    "templateName": "Screenplay", "printParagraphNumbers": false, "showSynopses": false,
    "currentTabIndex": -1, "sidebarWidth": 264 },
  "transient": false,
  "type": "com.quoteunquoteapps.fountain",
  "version": 2
}
```

The standard TextBundle keys sit at the top level, and the app's own state goes under a reverse-DNS key.

- **Our state:** Accolade keeps its own under `"eu.kiefte.Accolade"` and never touches the other keys.
  Possible candidates: the document type, and the adaptation source when it is kept in the bundle.
- **`type`:** gives Fountain or Markdown, alongside the text file's extension.

## Highland's markup beyond Fountain

These come from the teardown and are worth knowing for Lexington:

- emphasis marks: `+highlight+`, `~~strike~~`, `_underline_`;
- endnotes, `[^id]`;
- the term list, `Key :: Value`;
- directives: `{{HEADER: …}}`, `{{FOOTER: …}}`, `{{TOC}}`, `{{INCLUDE: …}}`, `{{PAGENUMBER: n}}`, `{{SERIES: n}}`,
  `{{WATERMARK: …}}`, and markers `{{%m}}`;
- lyrics, `~`;
- the page-break line, `===`.

Since Lexington 1.47 the directives are done, as Highland 2 does them (per highwater-too's parser, MIT, checked
against Highland's PDFs): includes, header and footer, page numbers, series, timestamp, TOC in print; markers
never printed. In Accolade an include is a chip (tap or `zo` shows the file, Open edits it), Page Setup's Header
and Footer are `{{HEADER}}`/`{{FOOTER}}` (the script's own wins and is edited in place), and markers stay
markers (chips, Next/Previous Marker): see highland_directives.go and docs/help/formats.md.

## What the samples taught

- **Invalid JSON:** the three folder samples' info.json end their object with a trailing comma
  (`"creatorIdentifier":"…",\n}`), which strict JSON readers refuse. Accolade edits info.json as text: it
  finds the top-level members and replaces, adds or removes only its own. A trailing comma is kept, and so
  are `\/` escapes, key order and layout. An info.json that cannot be read at all is left alone.
- **No info.json:** one sample has none (the spec calls it required). Accolade opens it all the same,
  taking the type from the text file's name, else the spec's default, Markdown.
- **No text:** a bundle without a `text.*` file is an error ("holds no text file"). It never opens as an
  empty document.

## Rules for Accolade

- **Opening:** reads the text; the document type comes from the text file's extension and info.json's `type`.
- **Opening:** the type comes from, in order, a choice made in View > Document Type, Accolade's key, the text
  file's extension (`.md`/`.markdown`, `.fountain`), then info.json's `type` (containing "fountain" or
  "markdown"). Failing those, a `.highland` is Fountain and other bundles are Markdown, the spec's default.
  The text file is picked as caotang picks it (`text.fountain`, then `.md`, `.markdown`, `.txt`, then any other
  `text.*`).
- **Saving:** for the same file, the bundle is re-read from disk first, so what another app changed in it since
  it was opened is kept. If the file is gone, emptied (a Save As dialog empties it), or now has another layout
  or text file, the bytes as read are used instead. Writes only the text file and Accolade's own key in info.json. Every other entry (assets, sidecar
  JSON, unknown files) is copied through byte for byte, in its original order. Zip entries go through
  `zip.Writer.Copy`, raw and not recompressed, and the archive comment is kept.
- **Accolade's key**, `"eu.kiefte.Accolade"`: `{"version":1}` plus `documentType` ("markdown" or
  "screenplay"), written only when the document is not what the bundle says, and `source`, the entry holding
  the adaptation source. When there is nothing to say the key is removed, or never written, so a plain
  open-and-save of a Highland file changes only its text.
- **The adaptation source:** layer 2 of source_file.go, after the script's own block and before Accolade's kept
  copy and the file beside it. It is `assets/source.md` (or the entry the key names, or a `source.md` beside
  the text). assets/ is a guess, not tested: the developer's page names assets/ as part of the format, so Highland
  seems likelier to keep it than an unknown file at the root, though the spec means assets/ for files the text
  refers to. Check this with a real Highland when one is at hand. The
  source is written into the bundle at once (Choose Source, the pane's Done, Keep Source in the Script off),
  with the text in the bundle as last saved. Saving the bundle as a plain file keeps the source as a copy
  Accolade keeps.
- **A new bundle** from a plain script (Save As, or a rename to `.highland`/`.textpack`) holds
  `info.json` (`version` 2, `type` com.quoteunquoteapps.fountain or net.daringfireball.markdown, `transient`
  false, `creatorIdentifier`), `text.fountain` or `text.md`, and `assets/`.
- **Zip or folder** (2026-11-23): reading and writing live in Lexington's `bundle` package, shared with its command
  line. Every type comes in both forms, the same files zipped or in a plain folder (`.textpack` ⇄ `.textbundle`,
  `Name.accolade` ⇄ the folder `Name.accolade/`, `.highland` likewise). The form is told by what is on disk, and a
  bundle is written back in the form it was read in. Save As (Accolade folder, TextBundle folder) and Rename switch
  it, in place too. A new folder is written whole and then moved into place, over an empty file a save dialog left,
  or over a bundle of either form, never over anything else. Open goes into a folder bundle and opens its text
  file; File > Open Bundle Folder… opens the folder itself. The browser saves zips only. docs/FORMAT.md describes
  the bundle roughly.
- **.accolade** (PRODUCTION step 3) is a `.highland` by another name: the same zipped TextBundle, a screenplay by
  default, never opened as plain text, plus Accolade's extras under `assets/drafts/` (and later
  `assets/production/`). Accolade's key gains `drafts` (the index, `assets/drafts/drafts.json`) and
  `highlandWarned`.
- **Accolade's files in a .highland:** a .highland can hold drafts and the source too, but whether Highland keeps
  them on its own save is untested. Saving a .highland that holds them (or the first draft saved into one) says so
  once, after the save; `highlandWarned: true` in the key keeps it to once per file. To test with a real Highland:
  open such a file there, edit, save, and list the zip (`unzip -l`).
- **No samples from others:** none is committed unless its licence allows it. Test fixtures are made by us, in both
  layouts. Real `.highland` files dropped into testdata/highland are picked up by `TestHighlandFiles`: each is
  opened and saved, and every entry but the text is compared byte for byte. That folder's .gitignore keeps
  them out of git.
