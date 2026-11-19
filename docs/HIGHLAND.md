# TextBundle and Highland files: what is known

Collected 2026-11-19 for LOOP.md's TextBundle / .highland item. Highland has published no spec, so this is pieced
together from the developer's pages and from other open-source readers. Real sample files are still wanted, as the
FDX work had them: ask Joop for any he has.

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

## The layout, and where readers disagree

- **The teardown:** the zip holds one folder, `Name.textbundle/`, with the bundle's files inside it:
  - `text.fountain` (`text.md` for Markdown);
  - `info.json`;
  - `characters.json`, `navigatorFilters.json` and `sprints.json`;
  - `assets/`.

  The binary also names `characterHighlighting.json`, plus revisions, milestones, a bin, a scratchpad and includes,
  stored the same way.
- **caotang:** looks for `text.fountain`, `text.markdown` or `text.txt` at the zip's root, in that order.
- **What Accolade does:** accepts both layouts, files at the root or inside a single top-level `*.textbundle/`
  folder, and writes back in the layout it read. A new file gets the teardown's layout, since it was seen in
  Highland's own samples.
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

- **Our state:** Accolade keeps its own under `"io.github.lapingvino.accolade"` and never touches the other keys.
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

## Rules for Accolade

- **Opening:** reads the text; the document type comes from the text file's extension and info.json's `type`.
- **Saving:** writes only the text file and Accolade's own key in info.json. Every other entry (assets, sidecar JSON,
  unknown files) is copied through byte for byte, in its original order.
- **No samples from others:** none is committed unless its licence allows it. Test fixtures are made by us, in both
  layouts.
