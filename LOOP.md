# Accolade in the hourly loop

Joop, 2026-11-19: Accolade is in the hourly tick, next to goloco, after Rebekah's WEquilibrium notes.

## What the tick works on

- The adaptation plan, `docs/ADAPTATION.md`, in its Order. Each step is one slice, or one background agent in a worktree.
- Only one Accolade agent runs at a time.

## Rules

- **Never cd:** use `git -C` and `go -C`.
- **Tests:** `go test ./...` must pass before each commit and before merging (commit only inside an `if` on
  success), plus the wasm build. The race suite runs in CI on every push (.github/workflows/ci.yml), not locally
  (Joop 2026-11-20: "those tests take long"); after a push, check the CI run (`gh run list -R LaPingvino/accolade`)
  and fix a red one first. Agents run only targeted `-race -run …` tests when fixing a race.
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
- 2026-11-20 (Joop): drafts in the bundle — keep earlier versions of both the script and the source inside .accolade, so one can go back. Writing drafts (File > Save Draft…, named, any time, also before lock) and the production issues (White, Blue … from PRODUCTION step 4) share one store: assets/drafts/NN-name/ holding text.fountain (and source.md when it changed) + drafts.json (name, date, kind: draft | issue, colour). Identical texts are not stored twice. Browse / Compare (a diff, like Highland's Compare…) / Restore as a new draft (never overwriting). Flattening to plain .fountain keeps them in the Name.production/ copy. Goes with PRODUCTION step 3 (the container), before step 4 (issues build on it).
- 2026-11-20 (Joop): "lock the writer's source, start adding on top" — each stage locks what came before and adds layers on top, never editing it: (1) the source story, locked when adaptation starts (edits make a new source draft; links stay anchored to the locked one, with a "source changed since" notice per passage); (2) the writer's draft, locked at PRODUCTION's Lock (= the White issue); (3) the shooting script = the writer's draft + production layers (scene numbers, OMITTED, revisions as issued drafts, tags/breakdown/schedule in sidecars). Export either the clean writer's version or the shooting version from the same bundle. Drafts store (above) holds every locked stage.
- 2026-11-20 (Joop): external merges — someone edits a copy outside Accolade (a co-writer in Highland/Final Draft/any editor, an FDX from production, the flattened .fountain) and it comes back. Because every locked stage and draft is kept, Accolade has the common ancestor: File > Merge Changes… takes the outside file (fountain, highland, fdx, …), finds its base among the stored drafts (by the bundle/draft id when present, else the closest draft by text), and does a three-way merge per scene/element into a new draft: clean changes go in (marked as that revision's changes), conflicts shown side by side to pick. Links, cut marks, scene numbers and production sidecars follow the merged text (anchored by words; scenes matched by heading + content). Also the same for the source story. Exports carry the draft id (in info.json / an FDX header field / a boneyard line) so a returning file finds its base exactly.
  Compare and partial merge (Joop): Merge Changes… opens a side-by-side compare first, per scene/element, each change with Take / Leave (and Take all from this scene); only what is taken goes into the new draft. The same compare works between any two stored drafts, or a draft and an outside file, without merging.
  PDFs too (Joop): a script PDF someone sends back (their edits, or a marked-up print) goes through the same compare via Lexington's pdfin (OCR for scans): its text against the draft it came from, each difference offered as Take / Leave, or kept as a suggestion — a note in the script ("[[suggested by NAME: …]]") or a desk-style list of suggestions to work through — so their ideas are visible without being merged. Scene numbers and page numbers in the PDF help find the base draft and line up scenes.
- 2026-11-20: step 5, the reading helpers (branch heuristics, not merged; internal/storyread, source_reading.go).
  Heuristics only, pure Go, browser too. Word lists per language in internal/storyread/lang/xx.txt (en nl pt es fr
  de eo; one file each, [speech] with "= manner", [thought], [titles], [place-before], [interior]/[exterior],
  [time], [shift], [not-names] …; an unknown language reads with English's plus the punctuation rules). The source's
  language is detected from its frequent words (Lexington has no pt/es), else the script's. Dialogue: “ ” " " ‘ ’
  « » „ “ » «, paragraph dashes (—, –, --, -Para) with toggling inner dashes and incises (", disse eu"); a "-" list
  run counts as dialogue only if it reads as talk. Speaker: verb + name/pronoun/description next to it (said Anna,
  dit-il, cried his wife), else the sentence's nearest name; the paragraph's; a beat; "les dijo:" before; else the
  other of the two talking (vocatives steer it). Manner from the verb (whispered → whispering) or an -ly/-mente
  adverb. Characters counted with titles, speech and vocatives as weight; places after in/at/to or with Park, rua…;
  aliases separate. Scenes at headings, breaks, "III."/"CHAPTER I" lines and shift phrases, with a suggested
  heading (INT./EXT. from room/street words, place name, time). Thought/feeling verbs flagged. The pane: "What the
  source shows" under the passages for the picked one (closable). Tests: 14 public-domain Gutenberg excerpts in
  internal/storyread/testdata with hand-read speakers (.want, written before any run; the -b ones held out,
  though their misses were seen once and four general fixes followed): lines found 187/188. Right speaker over
  all expected lines 61/92 (66%) tuned-on, 53/96 (55%) held-out; of the lines it names a speaker for, 87% and
  69% (counting "his wife" for Mrs. Bennet; the exact name 53% and 39%). Per language, right of expected: en
  42/50, eo 16/17, es 10/13, de 11/19, pt 23/48, nl 3/9, fr 9/32 (fr-dumas-a 2/17: two unnamed men). A
  language without lists (Italian, Latin) is not detected as a listed one (hit-rate floor) and reads with
  English's. Whole Gutenberg novels read in 0.03–0.5 s natively, no panics. Rough: French narrators' "l'inconnu"/"l'hôte" pairs, three-way talk, pronoun gender, German nouns
  as names, the narrator-as-addressee ("disse-me"). Next: step 6, suggestions while writing.
- 2026-11-20: NEXT, before step 6 (Joop: "loading and editing source files kinda freezes the web app"): the fixes in docs/PERF-SOURCE.md, worst first; with them the autosave/theme data race seen under -race.
- 2026-11-20 (Joop): after the PERF fixes, before step 6 — work through every combination of source × screenplay and make each feel natural: source none / exists (beside, kept, in the script, in a bundle) / partial; screenplay none (new, empty) / exists / partial; and both partial. For each: what a writer sees first, what they can do next, where it is awkward. Likely gaps to look for: starting a screenplay from only a source (a new script with the source attached and its chapters as sections, Next ready); adding a source to an existing screenplay with no links (suggest links by matching scenes' dialogue/action to passages, accept per scene); a screenplay that already covers part of the source (statuses right from the start, Next continuing after the covered part); a source that is still being written (passages added later, links holding). A scenario test per combination (fyne test driver), screenshots of the first view for each, fixes for what is awkward.
- 2026-11-20: PERF-SOURCE 1–9 on branch perfsource (not merged). The source is edited in our editor; the tail blocks
  are found by a scan of the text's end, cached per text; link matches are cached; the pane follows edits after a
  200 ms pause; rows are estimated, then measured in stretches; the source is split into passages, and Choose
  Source and Import convert, in the background, yielding every ~20 ms (internal/pace); the draft is kept after a 1 s
  pause; old kept copies are deleted; stats and the editor layout are cached; a window's timers stop with it (the
  auto-save race). Per keystroke with the pane shown: 173→20 ms, with the source in the script 799→60 ms. Cursor
  move: 39→6 ms, inside 365→9 ms. First open: 3.2 s→29 ms on the UI goroutine. Numbers: docs/PERF-SOURCE.md.
  Left: incremental wrap in the editor; bundle zipping off the UI goroutine.
