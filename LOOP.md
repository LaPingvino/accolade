# Accolade in the hourly loop

Joop, 2026-11-19: Accolade is in the hourly tick, next to goloco, after Rebekah's WEquilibrium notes.

## What the tick works on

- The adaptation plan, `docs/ADAPTATION.md`, in its Order. Each step is one slice, or one background agent in a worktree.
- Only one Accolade agent runs at a time.

## Rules

- **Never cd:** use `git -C` and `go -C`.
- **Tests:** `go test ./...` and `go test -race .` must pass. Commit only inside an `if` on success.
- **Pushing to main** redeploys the public browser version (pages.yml), so main must always work in the browser too.
  Anything that needs the desktop (pandoc, `exec`) has a browser fallback, or is left out there with a clear message.
- **Lexington:** changes it needs go into `~/Projects/lexington` first and are tagged there. Accolade then updates
  its go.mod, as for the earlier export work.
- **Screenshots:** anything visible is checked with a screenshot, as `screenshot_test.go` makes them.

## Tick log

- 2026-11-19: added to the loop. Plan in docs/ADAPTATION.md. Next: step 1, the Markdown mode.
- 2026-11-19: step 1, Markdown mode (branch markdown, not merged). A .md/.markdown file opens with Markdown colouring
  (syntax/markdown.go: line kinds and inline emphasis), an outline of its headings, a live preview beside the editor
  (Fyne's RichText, goldmark inside; Ctrl+Shift+P), notes in `<!-- Notes -->`, list continuation on Enter, Print as a
  goldmark web page. Elements, completion, title page, export, cards and colours are put away in Markdown. File > New
  Markdown Document; Open offers .md (browser too). Next: step 2, import and export.
- 2026-11-19 (Joop): renaming to .md / .fountain switches the mode (checked by him) — fine as a last resort. To add: a direct switch (View > Document type: Screenplay / Markdown) that keeps the file name, saved per file. Small; before step 2.
- 2026-11-19: done, the direct switch (branch doctype, not merged): View > Document Type > Screenplay / Markdown (checked
  items), keeping the name and the text as they are; a notes block in the other form is carried over (only its marks
  change). The choice is kept per file in settings.json ("document-types", only where it differs from the extension;
  the browser keeps its settings folder too) and wins over the extension on reopen. Renaming to .md/.fountain still
  switches; a rename to .txt or another name keeps the type. Peers get the mode as before. Next: step 2.
- 2026-11-19: step 2, import and export (branch mdimport, not merged). File > Import as Markdown: DOCX, ODT, HTML and
  text read by internal/prose (pure Go: one block/span model, readers and writers together; headings by style name or
  outline level, lists from numbering, quotes, tables, bold/italic/strike/code, links, footnotes as [^n], images left
  out with an `<!-- image left out -->` comment), HTML through golang.org/x/net/html; PDF as prose through Lexington's
  new pdfin.ReadProse (OCR included). Export in Markdown mode is its own dialog: HTML (goldmark, footnotes now on),
  DOCX and ODT (Word's/LibreOffice's own styles, real lists and footnotes), PDF (headless Chromium, else the print
  dialog's Save as PDF; the browser prints), Markdown as written, plain text. With pandoc on the desktop: File >
  Import with pandoc (EPUB, RTF, LaTeX, Org, RST, DOCX/ODT/HTML too) and EPUB/RTF/LaTeX/DOCX/ODT/Org exports; hidden
  in the browser and without pandoc. go.mod has `replace github.com/LaPingvino/lexington => ../lexington` until
  Lexington's v1.6.0 tag is pushed (then go get it and drop the replace). Next: step 3, the source pane.
- 2026-11-19 (Joop): support the TextBundle family for Markdown and Fountain — .textbundle (a folder: info.json, text.md / text.fountain, assets/; desktop only), .textpack (the same zipped) and Highland 2 / Pro's .highland (a zipped, customised TextBundle: text.fountain / .markdown / .txt, info.json, assets/, maybe sprints.json, characters.json — https://quoteunquoteapps.com/highland-2/highland-format; no published spec, so keep unknown files as they were on save). Open, save and Save As; the text file's extension sets the document type; assets/ kept untouched; info.json's keys preserved. A natural home for the adaptation source (assets/ or a second text file) — decide with step 3's layers. Browser: zips only. Next after the source pane.
  Research for it: docs/HIGHLAND.md (layouts seen, info.json, readers to compare; samples wanted from Joop).
- 2026-11-19: step 3, the source pane and its links (merged). Where the source is, in
  layers, looked for in this order (Joop: the browser has no folder beside the script): (1) in the script, a
  `/* Source … */` boneyard block before the notes (Format > Keep Source in the Script moves it in or out, off by
  default); (2) a copy Accolade keeps: File > Choose Source… reads .md/.txt/.docx/.odt/.html (internal/prose) and keeps
  it as Markdown in the data folder (~/.local/share/accolade/sources; the browser: the kept settings folder),
  remembered per script in settings.json "source-files", following a rename/Save As; File > Save Source As… gets it
  out (the browser downloads it); (3) story.md beside story.fountain. The pane names the one in use. The Notes pane
  has two tabs, Notes and Source (View > Source): the source as passages (paragraphs, headings, quotes, each list
  item); Edit opens it as text in the pane, Done saves it where it is kept; a kept/beside file changed elsewhere is
  read again when its time or size changes, checked as the pane follows the cursor. Links are
  `[[from: "first eight words…"]]`, quoting more until unique, written by Format > Link to Source Passage at the
  cursor (an empty line, inline after dialogue, a line below a heading). Matching: normalised words (case, quotes,
  punctuation), exact prefix; else near (word edit distance ≤ a quarter of the words, link ≥ 4 words, one best);
  ties ambiguous, none missing — said in the pane, candidates tinted. The cursor's scene lights its passages
  (scrolled to if none in sight); a tap on a passage picks it and goes to its scene, the next on each tap. Format >
  Source Headings as Sections: all at the cursor as an outline, or the missing ones in their places, never twice,
  one undo step. Passages are keyed by their words so statuses can attach. Rough: the editor colours the source
  block as Fountain (boneyard has no colour of its own yet, the notes block neither). Next: TextBundle / .highland (above), then step 4, statuses and
  Next passage.
- 2026-11-19: the TextBundle family (branch textbundle, not merged; bundle.go, docs/HIGHLAND.md). Open, Save, Save As,
  Open Recent, rename, autosave and the browser's picker take .highland and .textpack (zips) and .textbundle (a
  folder, desktop only: Open goes into it and opens its text). The type comes from the text file's name, then
  info.json's `type`. Both zip layouts are read (root, or one Name.textbundle/) and written back as read. A new zip
  gets the folder layout, for .textpack too: TextBundle's own sample, writer and reader all use it (the spec is
  silent). Saving changes only the text and Accolade's key; every other entry is copied raw in order, the zip comment
  kept. info.json is edited as text, so its trailing commas, `\/` and key order survive. A non-zip .highland is
  plain Fountain; a bundle with no text is an error. Save As from plain makes a minimal bundle. The source is a new
  layer 2: assets/source.md in the bundle, written at once by Choose Source and the pane. Tests: generated bundles
  (both layouts, assets, sidecars, unknown file, __MACOSX); TextBundle's MIT samples in testdata/textbundle; any real
  files in testdata/highland (gitignored). A same-path save re-reads the bundle first, so other apps' changes are kept.
  Rough: no real .highland tested yet; assets/ for the source is a guess. Fyne's save dialog refuses an existing
  folder's name, so Save As can't overwrite a .textbundle. Each save, autosave too, rewrites the whole zip. Next: step 4, statuses and Next passage.
- 2026-11-20: step 4, statuses and Next passage (branch statuses, not merged; adaptation.go, adaptation_pane.go).
  Each counted passage (not a heading or break) is adapted (a link resolves to it; ambiguous ones don't), cut on
  purpose, or to do. Cuts live in the script: `/* Adaptation` block after the script, before Source and Notes
  (either order with Source is read), `cut:` then one `"first words…"` per line, `(2)` for the second of alike
  passages; read leniently (on the key's line, lists, bare words), other lines and unresolved entries kept. The pane:
  a mark left of each passage (green solid adapted, red dashed cut, thin grey line to do), the count "50% adapted ·
  1 cut · 3 to go" (% of the passages not cut), Next and Mark as Cut / Unmark Cut (one undo step; linking a cut
  passage drops its entry in the link's step). Format > Next Source Passage (Ctrl+Alt+J), Mark Source Passage as
  Cut. Next: after the cursor's scene's last linked passage (else the nearest scene before with links, or the one
  Next showed), the first to do, wrapping; starts a scene after the cursor's (chapter section if the script enters a
  new chapter without one, `INT. PLACE - DAY` selected, the link), one undo step; Preferences > Script > Adapting
  "Next starts a scene" (default on) or only shows and offers Start a Scene. Next again with the shown one still
  unlinked asks in the pane: mark it as cut? Rough: in insert mode the skip question only comes after an undo;
  a duplicate passage can be cut but Next's link to it is ambiguous, so it stays to do; placeholder is English-only
  ("…" in other languages). Next: step 5, the characters and dialogue heuristics.
- 2026-11-20 (Joop): the bundle as the project container. A .highland that comes in is saved as it is; Accolade's own format is the same TextBundle plus what Accolade adds (production sidecars, the adaptation source, …), saved as .highland (Highland-compatible), as .accolade (same structure, our extension), or as individual files when needed; with export options for on set. Sidecars use existing standards where any fit (the production research, docs/PRODUCTION.md when written). To test with a real Highland: does it keep unknown files on its own save?
- 2026-11-20: production research done — docs/PRODUCTION.md (what exists, open formats to adopt: FDX, OSF, iCalendar, vCard, OTIO, Fountain #n#; the .accolade container with assets/production/; top gaps and an Order of 12 steps with a guided Production pane). Lane order: finish ADAPTATION steps 5–7, then PRODUCTION from its step 1 (unless Joop reorders).
