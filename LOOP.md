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
- 2026-11-20: branch srctable (not merged). Phone fixes. A wide table stopped the source pane scrolling: Fyne puts a
  Markdown table in a scroller of its own, and a touch drag (a wheel scroll in the browser, turned sideways) went to it.
  Rows (and the Markdown preview) show a table as lines now, "Column: cell · …", with its size; a long row is cut
  with Show All; no row holds a scroller or is wider than the pane. Under 600 px wide the panes go under the script
  in a split (half each; an arrow over the panes makes them taller, as a finger cannot drag the split in the
  browser); a toolbar button toggles the source and stays on the row however narrow. Export, Title Page and
  Preferences dialogs fit a phone's width. Phone shots: SHOT_DIR=… go test -run TestPhoneScreenshots.
- 2026-11-20: every source × screenplay combination (branch combos, not merged; combos_test.go has a scenario per
  combination, SHOT_DIR=… go test -run TestCombo renders each first view at 1100×650 and 380×800). Found and fixed:
  no source showed a blank pane (the empty state never came up on first show) with Next and Cut still there: it says
  what to choose now, the tools hidden; a source with no passages says Edit writes it. A Markdown story (opened or
  imported) had no way on: File > Start a Screenplay from This Source and a toolbar button (the import's status says
  so) make a new script, a title page naming the story (Source: Based on …), its chapters as sections, the first scene
  started; a saved .md on the desktop stays open and is the source itself (it can go on being written), an import or
  the browser's is kept. An empty script given a source is offered the same in the pane. A script without links, a
  source added: links are suggested scene by scene (a speech storyread finds in the passage said in the scene, or the
  passage's words in the scene's, rarer words weighing more), Link / Skip / Not Now, never put in by itself; Format >
  Suggest Links to the Source asks again. Next from the top of a part-adapted script (as it opens) went on from the
  first passage and put the scene under the title page: it goes on after the last passage linked, after that scene;
  with the chapters' sections there, a scene goes under its own chapter's. A linked passage changed in the source
  shows its mark in the warning colour, "N changed since linked" in the count, and the scene's info says to link
  again. A new script or import no longer inherits the last unsaved script's source; the browser's kept draft keeps
  an unsaved script's source over a reload. A long status message made the window wider than a phone: it wraps now,
  the statistics give way. Settings leaked between tests (next-passage-inserts): restored after them.
  Ideas, not built: (1) "changed since" sees the quoted words always, the rest only within a session (it starts
  from the words first seen linked): store a fingerprint of the whole passage with each link (goes with the locked
  source, above) and a Relink that updates a link's quote; (2) suggestions: stems (tore/tears, climbs/climbed) from storyread's word lists, the story's order
  as a hint, a passage in two scenes, Skip remembered (step 7's notes), cuts offered for skipped passages between
  linked ones; (3) the first scene's heading from storyread's suggested INT./EXT. PLACE - TIME instead of the
  placeholder; (4) an open .md that is some script's source could say so and open that script; (5) the settings
  object is one for the process (GetSettings, globalSettings), shared by all tests in a run: give each test its own.
  Follow-ups (second commit): a passage changed after the words its link quotes counts as changed too, against its
  words when the session first saw it linked; the epigraph case (a passage under the story's title before Chapter 1:
  sections keep the title's heading, a scene goes before the next section); a long source (≥ 64 KB) is asked first
  before its links are suggested, and the matching yields (the bench novel: read 0.13 s, match 0.08 s natively); a
  stale question is dropped when the source changes. background_real_test.go flaked (main 1 in 30, this branch 3 in
  30: a source list refreshed by a background result while a measuring timer set its heights, side by side under
  the test driver); timers' and background results' functions now take turns (background.go uiTurn): 0 in 40.
- 2026-11-20: step 6, suggestions while writing (branch suggest, not merged; suggest.go). From the passages linked in
  the cursor's scene (else the one Next showed), in the status bar as Tab's hint ("Tab: TOM’s line: “Out. Is there
  tea?” · from the source · 1 of 2"); Tab takes it (one undo step), Tab again the next, round; Esc puts it away for
  that line until the text changes; typing on, the next comes from where the cursor is. A cue at the end of its line
  (or the empty line under it, or under its parenthetical): that character's next line not in the scene, the manner
  as a parenthetical, then the lines no one is named for (those only when the line is no name or heading being
  typed: AN, EXT still complete). A name Tab completed to the only one: Tab again gives its line. A cue being typed: the passages' speakers first (mapped to
  the script's cue when its words are the name's or part of it: ANNA K for Anna; no dots, MRS BENNET), then the
  script's own. An empty line: the next piece not in the scene, a line with its cue (CHARACTER, selected, when no one
  is named), or the narration as action as written ("not seen on screen" for thoughts); a blank line put before or
  after where needed. A heading being typed: storyread's INT./EXT. PLACE - TIME for the scene's passage (links below
  it, Next's, else the first to do after the last link above), the setting read first; none without a place. Used: a
  piece's normalised words in a row in the scene (one or two words: a whole line), notes and boneyard left out.
  storyread marks the sentence that says who speaks as a Tag piece ("said Tom, shaking his coat."), left out; a
  line's pieces are joined with a full stop where the narration between ended a sentence. Preferences > Script >
  Adapting "Suggest from the source while writing" (default on). The source is read for them with the pane hidden
  too (again a second after typing; the pane is made only for a script with a source). Cost: read once per source
  (in the background from 64 KB, storyread.Read now yields: pace); per revision and line the scene's lines only. On
  the bench script, the cursor where a suggestion shows: the hint 0.66 ms on, 0.98 ms off (the old completion hint),
  the text's copy apart; a whole keystroke 36 vs 38 ms (noise). Phones (Joop: yes): every "Tab: …" hint, the
  completions' too, starts with a small "Tab ⇥" chip (accessible: "Take the suggestion"), there only with a hint; a
  tap on it is Tab (takes, then the next; same undo step), the editor focused again; the message wraps beside it.
  Rough: the chip is caption-sized (about 50×28, under a finger's 44 px); no grey text after the cursor (the editor has no ghost text; the status bar is small and
  grey); a line edited after it was taken is offered again; speakers storyread misses come as CHARACTER; a tag with
  action in it ("shaking his coat") loses that action; the cue's cycle and the name completions' cycle are separate
  (after completing a name, Enter then Tab gives the line).
- 2026-11-20: TestFocusIsVisible is skipped under -race (race_on_test.go): on CI's Linux race job the Find/Replace bar's and the outline filter's focus never shows in captures, even after waiting ~4.5 s; plain runs on all three systems pass. To find: what differs under race (a renderer cache? a focus animation?). Also: run TestFocusIsVisible alone before every merge (it caught the Preferences page growing).
- 2026-11-20: step 7, the learned notes (branch learned, not merged; adaptation_learned.go the block, adaptation_learn.go
  the learning, adaptation_learned_view.go the view; format in ADAPTATION.md section 6). The adaptation block gains
  `characters: ANNA = Anna, Annie`, `not characters: Monday, Lord`, `places: the house = INT. KOVAČ HOUSE` and
  `skipped links:` (heading = "passage quote"), read leniently (keys in any case and a few synonyms, entries inline or
  below, `;`/`,`, `=`/`→`; unread entries and other lines kept), written in that order before `cut:`. Learned on the
  pane's follow (after typing pauses; the pane hidden too, on the suggester's 1 s reload), cursor's scene only, from
  actions not standing state (what the script had when opened teaches nothing; a cue/heading is looked at once the
  cursor left it): a cue Tab put in renamed, or a line the source reads as X typed under another cue → alias, unless
  the old name's own cue still speaks elsewhere in the script (a correction); a taken cue's name deleted (line left
  cueless) or renamed against a learned alias, twice → not characters (a line deleted whole or reworded: nothing); a new heading over a linked scene (Next's placeholder, or
  heading then link) unlike any the read place makes → places (time stripped); Skip on a suggested link → skipped
  links (not offered to that scene again). Quiet: only the block changes, joined to the writer's last undo step
  (buffer.Amend), cursor/selection kept (on the empty line before the block, a blank line kept after it); not while
  Redo is possible, a Tab cycle runs or text is selected. storyread.Hints now merge aliases into one speaker and
  character, drop not-characters as speakers, and give a place its heading (Scene.Given); readings and offers are
  re-read when the hints change; suggestions use the cue, the place's heading first. The pane: "What Accolade learned
  (n)", folded, rows with a remove button each (one undo step; not relearned this session). Also: tail blocks' rune
  places counted in one pass. Bench (base vs branch, n=8): SuggestKeystroke 22.4 vs 22.5 ms (whole keystroke
  unchanged); the hint's own share about +0.1 ms (0.49 vs 0.63 ms, a first run +18% p=0.004); LearnedKeystroke none 21.8 / learned block 20.4 ms; one learning tick 0.56 ms. Rough: strikes and
  "forgotten" are per session; no learning while shared live (collab); a correction made before the old name has a
  cue of its own elsewhere is still learned as an alias; the typed-cue alias rides on storyread's attribution (guarded, still possible to
  mislearn); a link added under a heading the script had when opened teaches no place; open on a phone the list
  takes most of the pane (it scrolls, folded by default); extra blank line kept before the block when it is first
  written with the cursor at the script's end. Next: step 8 (AI) only if wanted; else PRODUCTION from step 1.
- 2026-11-20: PRODUCTION step 1, locked scene numbers (branch scenelock, not merged; production.go; decisions in
  docs/PRODUCTION.md §7). Lexington v1.7.0 (tagged locally, not pushed): `lex.NumberScenes` (count, or locked: 10A,
  10B, 10AA between 10A and 10B, 10aA between 10 and 10A; or A11, B11, AA11, AB11; A1 before the first, the next
  number after the last; nothing taken twice), `fountain.OmitScene`/`RestoreScene`, `-number`, FDX OMITTED as the
  heading with its Number, and FDX no longer prints the boneyard. Accolade: a Production menu (not in Markdown) with
  Lock Scene Numbers (numbers every heading, the lock in a `/* Production` block: `scene numbers: locked, 10A`),
  Unlock Scene Numbers… (asks; the numbers stay), Omit Scene (`.OMITTED #12#`, the scene in `/* Omitted 12 … */`
  under it) and Restore Scene, one undo step each; a new scene while locked is numbered once the cursor left its
  heading and typing paused (part of the last undo step); Preferences > Script > Production: 10A or A11 (default
  10A, Joop; Wikipedia calls 10A the UK way, Final Draft uses it). Export numbers a locked script always; the
  Export dialog's own numbering is lex.NumberScenes now. go.mod has `replace github.com/LaPingvino/lexington =>
  ../lexington` until v1.7.0 is pushed (then go get it and drop the replace). Shots: combo-production-*.png
  (SHOT_DIR=… go test -run TestProductionScreenshots). A number left mid-line by text typed after it goes back to
  the end; the learner keys known headings without numbers (Lock taught it places). Rough: an omitted scene's
  boneyard shows as plain text. Next: PRODUCTION step 2, the basic reports.
- 2026-11-20 (Joop): source links in the browser's Text Fragments syntax (#:~:text=, WICG spec, supported by Chrome, Safari and Firefox): [[from: story.md#:~:text=She opened the door,stepped out]] — textStart[,textEnd] covers a passage range, prefix-/-suffix disambiguate instead of our "(n)", and a file name in front allows more than one source. Write it readable: percent-encode only what the grammar needs (, - & and %), spaces kept; read both encoded and plain, and keep reading the old [[from: "…"]] form (convert on the next save, or with a command). Use the same form in the Adaptation block (cut:, skipped links:) and in exports: an HTML/PDF of the source can then be opened at the exact passage from a link. Next after the reports step.
- 2026-11-20: PRODUCTION step 2, the basic reports (branch reports, not merged; reports.go; decisions and the CSV
  layout in docs/PRODUCTION.md §8). Lexington v1.8.0 (tagged locally, not pushed): `pdf.PDFWriter.Measure` (each
  scene's printed page and length, from the real paginator), `lex.EighthsOf` (nearest eighth, at least 1/8; written
  4/8, 1, 1 3/8), the `report` package (scenes, locations, characters, `Report.Cast()` numbered by size; CSV and a
  page to print) and `-report`. Accolade: Production > Reports…, one dialog: Report and Sort by (a table with
  sorting headers on a desktop, cards on a phone), CSV… and PDF…/Print…; measured with the export's settings, every
  scene numbered as Export would, aliases from the adaptation block merged. go.mod has `replace => ../lexington`
  until v1.8.0 is pushed. Shots: reports-*-wide/phone.png (SHOT_DIR=… go test -run TestReportsScreenshots). Next:
  PRODUCTION step 3, the `.accolade` bundle.
- 2026-11-20: source links in the Text Fragments syntax (branch textfrag, not merged; textfrag.go, textfrag_update.go;
  format in docs/ADAPTATION.md §3 "The link format"). `[[from: #:~:text=She opened the door to the garden and,out
  into the rain.]]`: after `from:` an optional file, `#:~:`, `text=[prefix-,]start[,end][,-suffix]` joined by `&`
  (one link each). start..end is a passage range (from start's passage to that of the first end after it); a
  short passage is all start, a longer one its first 8 words and last 4 (a quote's or list item's first and last
  block: a term is one block in a browser). prefix-/-suffix only when more words do not tell it apart, replacing
  "(n)" (twins with twin neighbours: ambiguous; Mark as Cut/Skip then write the old `"…" (2)`). Written readable
  (only , - & % [ ] encoded, the source's own characters as the page shows them: prose.Doc.Shown, no link
  targets or list marks; matched on those words too), read encoded or plain (a dash inside a word, a stray
  &). Matching is the spec's on the passages (terms within one passage as whole normalised words, prefix right
  before, suffix right after, the first end after the start; a word index per source, built on the first new-form
  link), but ambiguous where a browser takes the first, and near kept (start as before, end near one of the next 40
  passages' ends, else the start's passage alone). Old `[[from: "…"]]` and `"…" (2)` read as before; Format > Update
  Source Links writes every link and cut:/skipped links: entry that is found in the new form (one undo step; near
  ones keep their words; ambiguous/missing left as written, deliberately). New links (Link, Next, Start from the
  Source, suggestions: neighbours one range link) and block entries are the new form; no file part is written, a
  named one is the source's if any layer has that name (extension aside, so story.html too), else the pane says
  so and reads it against the source. Exports: the source exported/printed as HTML is where the fragments open
  (TestFragmentOpensInExportedSource runs the spec's algorithm on the exported page for every passage). Bench (main
  vs branch, n=6, old-form links): SourceModelKeystroke 4.0 vs 4.7 ms (~, p=0.18, noisy VPS), allocs 3623 vs 3624,
  B/op +15% (a bigger sourceLink); PaneKeystroke 21.2 vs 20.6 ms (~). New-form links: keystroke 3.4 ms, cold 35 ms
  (vs 19: the word index once per source read), a missed link 0.05-0.6 ms (old near link 0.76 ms). Not done: the
  script's own HTML export has no hyperlinks (Lexington drops notes); the pane links one passage (no range
  selection). Rough: accents count in matching (the spec ignores them); "fits N passages: quote more of it" says
  nothing of prefix-; a footnote's mark (the page's number right after the word) and an image's description are
  not in the shown text, so a link's words around them miss in a browser.
- 2026-11-20: PRODUCTION step 3, the .accolade bundle, the extras' homes and the drafts store (branch bundle3, not
  merged; extras.go, drafts.go, drafts_view.go, saveas.go, flatten.go, bundle_queue.go, internal/scriptdiff;
  docs/PRODUCTION.md §9). `.accolade` = the zipped TextBundle of a .highland plus assets/drafts/ (and later
  assets/production/); Save As offers it once a script has drafts or a lock; .highland saves keep working and warn
  once per file (info.json `highlandWarned`); a bundle saved as .fountain asks where its drafts go (kept by
  Accolade / Name.production/ beside it, desktop / left out after a second ask), and the plain file finds them
  again. File > Save Draft… and Drafts… (Compare any two or with now, by scene and element, word by word, side by
  side or stacked on a phone; Restore… keeps the script first, one undo step). Identical texts and sources stored
  once (drafts.json names each one's files). Bundle writes now run in the background, one after another
  (PERF-SOURCE item 7 done); the browser's Save As asks for a name. Still to do: test with a real Highland.
- 2026-11-20: PRODUCTION step 4, issues and revision sets (branch revisions, not merged; revisions.go,
  revisions_view.go; docs/PRODUCTION.md §10). Lexington v1.9.0 (tagged locally, not pushed; go.mod has `replace =>
  ../lexington` until it is): `lex.Line.Revision`, `lex.Revisions` (PDF: * at 7.75" on each printed line of a
  changed row, the changed pages' header "Blue Revision – 20/11/26", revised pages only, coloured paper), the WGA's
  colour order (`lex.RevisionName`: White … Cherry, then Second Blue; the brief's "double white" is in no
  WGA-attributed list), FDX `<Revisions>`/`RevisionID` out and `fdx.ParseRevisions` in (DeletedText places too),
  `fountain.ParseLines`. Accolade: Production > Issue Draft… (date; the first, White, locks the scene numbers; the
  issue in the drafts store, kind issue + colour + issued date; `revision: Blue, 2026-11-20` in the production
  block); marks never in the text, from diffing the issues' bodies line by line (scriptdiff.Lines), a deletion
  marking the line after; the open set in the next colour; marks of the last issue or of all issues (Preferences);
  a coloured bar in the editor's margin; Export > PDF "Revision marks", "Revised pages only", "Coloured paper", FDX
  "Revision marks"; an FDX with revisions is kept as an imported issue with its marks in
  assets/production/revisions.json until the next Issue. Shots: combo-revisions-*.png (SHOT_DIR=… go test -run
  TestRevisionScreenshots, which writes revisions.pdf too). Print (HTML) has no marks. Next: PRODUCTION step 5,
  locked pages and A-pages.
- 2026-11-20: PRODUCTION step 5, locked pages, A-pages and CONTINUEDs (branch lockedpages, not merged;
  lockedpages.go; docs/PRODUCTION.md §11). Lexington v1.10.0 (tagged locally, not pushed; go.mod has `replace =>
  ../lexington` until it is): `lex.PageLock` anchored by words (scene, first words, words before), `pdf.PDFWriter.Lock`
  (A-pages named as A-scenes: 12A, 12B, 12AA, 12aA; a page whose text is gone makes "12-13"; short pages stay short;
  pages after the last count on), `LockPages` (again: A-pages kept), `Pages`, revised pages counting A-pages,
  `Continueds` ((CONTINUED) / CONTINUED: (2)), FDX `<LockedPages>` out and in (Position as DeletedText's offset,
  checked on one Final Draft file; LockLevel for A-pages a guess), `-lock`, `-lockpages`, `-continueds`. Accolade:
  Production > Lock Pages / Unlock Pages… (one undo step each; the lock in the production block, `pages: locked,
  letter, format …, pagination …` and a line per page, for bundles too), export/preview/FDX in the lock's layout; the
  block holds the last issue's pages (every Issue locks again, A-pages kept; the first revision's Issue offers to lock
  the pages as White printed them) and a PDF of the script as last issued prints against the issue before's pages,
  Preferences "CONTINUED where a scene breaks over pages", an FDX's locked pages kept on import. Shots: Lexington's
  `LOCKED_SHOT=… go test -run TestLockedShot ./pdf` (page 12 with CONTINUED:, 12A with its Blue header), the Issue
  dialog's offer `SHOT_DIR=… go test -run TestIssueOffersLock` (lockedpages-issue-wide/phone.png). Print (HTML) keeps
  no locked pages. Rough:
  reports give the current pages, not locked labels. Next: PRODUCTION step 6, the Production pane.
- 2026-11-20: lockedpages_test.go is built without -race (as background_real_test.go): its tests run real background work, and under the test driver fyne.Do runs callbacks on the background goroutine, which -race reports (TestLockPagesUndo: writePageLock vs undo). To look at: drive these tests through one UI goroutine (a test helper that runs every UI call via fyne.DoAndWait), so they can run under -race again. Done (2026-11-20, branch racesafe): see the tick log.
- 2026-11-20: PRODUCTION step 6, the Production pane (branch prodpane, not merged; production_pane.go,
  production_checks.go; docs/PRODUCTION.md §12). A third tab beside Notes and Source (View > Production, a toolbar
  button that goes into the "more" menu on a phone; none in Markdown). Strip: 1 Lock, 2 Revise (the shown one
  highlighted, Lock ticked once issued), "3 Break down · 4 Schedule · 5 Day pack: coming" greyed. Stage from the drafts
  store (an issue → Revise). Lock: "Not locked yet · 6 scenes · 9 to check" / "Locked 1 Nov 2026 · 112 pages · 87
  scenes"; the check before the lock (no INT./EXT. or time, duplicate/missing/stray numbers, /* and [[ not closed or
  stray, a cue that is a learned alias or the same name spelled otherwise, scenes over ~3 pages, no title page), each
  row a tap to its line; Lock Scene Numbers, Lock Pages, Issue White… (whose dialog now offers to lock the pages as
  White prints them, checked). Revise: "Blue (open) · 1 page changed · scenes 2, 3" (pages counted by Lexington's
  pagination in the background, = revised pages only, A-pages included), the changed scenes in the set's colour;
  Issue Blue…, Revised Pages PDF… (Export set up), Compare with White. Follows the revision marks' 400 ms pause, never
  per key. On a phone the buttons go at the list's end. Shots: SHOT_DIR=… go test -run TestProductionPaneScreenshots.
  Rough: time/long-scene checks are heuristics; the page count lays the script out without yields in the browser.
  Next: PRODUCTION step 7, breakdown.
- 2026-11-20: race-safe real-background tests (branch racesafe, not merged). The tests of real timers and background
  work hold the UI turn for the whole test (holdUI, waitfor_test.go) and give it up only while they wait (waitFor,
  uiSleep): the timers' and results' callbacks take uiTurn (uiDo, background.go; lockedPagesOf now uses it too), so
  they run between the test's steps, as on a real driver's UI goroutine. No more `//go:build !race` on
  autosave_timer_test, background_real_test, lockedpages_test, production_pane_test, production_pane_real_test:
  their 23 tests run in CI's race job (-race -count=3 clean). A new test of real background work: holdUI(t) first.
- 2026-11-20: PRODUCTION step 7, the breakdown (branch breakdown, not merged; breakdown.go, breakdown_view.go,
  breakdown_io.go, production_breakdown.go; Lexington v1.11.0 local tag, go.mod replace; docs/PRODUCTION.md §13).
  Categories: Final Draft 13's ids/names (= Movie Magic's), colours from a Final Draft v5 file and OSF 4.0 (the
  same); one's own addable. Tags in assets/production/breakdown.json (bundle, kept copy, Name.production/), anchored
  by scene (number, else heading + nth) and words (a Text Fragment directive); lost tags listed, never dropped.
  Production > Tag Selection… (Ctrl+Shift+B; picker with filter/new category), Untag, Show Tagged Words, the
  editor's right click; tagged words coloured and underlined, found again 400 ms after typing pauses. Pane stage 3
  "Break down": "33% broken down · 2 scenes to go", the scene's eighths, location, cast (from cues, numbered; and
  tagged), elements by category, suggestions, lost tags; Next Scene, Mark Scene Done, Breakdown Sheets…. Reports:
  Breakdown sheets (CSV; PDF in the classic layout via gofpdf) and Elements. FDX out: SceneProperties Length/Page
  always, TagData + TagNumber with a breakdown (cast from cues tagged); FDX and Beat tags in. MMS reads only
  sluglines and speaking characters from FDX (its docs): tags reach it only via .sex. Shots: SHOT_DIR=… go test -run
  TestBreakdownScreenshots. Next: PRODUCTION step 8, the schedule.
- 2026-11-20 (Joop): Accolade's own blocks and markers should fold away or show more naturally. In the editor: the tail blocks (/* Notes */, /* Source */, /* Adaptation */, /* Production */) folded by default into one quiet line each ("▸ Production · scene numbers locked · Blue", "▸ Source · story.md, 2 592 passages"), unfolded on click or when the cursor enters; an OMITTED scene's kept boneyard folded under its heading ("▸ the omitted scene, 12 lines"); source links [[from: …]] drawn as a small chip ("↗ source" / "↗ 3 passages", the quote on hover/long-press, raw text when the cursor is inside); learned/cut entries only in their panes. Fountain text unchanged — display only; other apps still see plain Fountain. Uses editor_folds.go's folding; keep per-keystroke cost flat. Next after the schedule step.
- 2026-11-20 (Joop): a sample .accolade to test everything with — none exists yet. Generated, not hand-made: a Go generator (e.g. cmd/makesample or a test with -update) that builds examples/sample.accolade through Accolade's own code, so it stays valid as formats change, and a test that opens it and checks every feature is present. Contents: a public-domain short story as the source (e.g. Poe's The Tell-Tale Heart, or Chekhov in translation — note the PD status) partly adapted: links (Text Fragment form, a range, a prefix-disambiguated twin), a cut passage, learned notes (an alias, a not-character, a place), an OMITTED scene and an A-scene with locked scene numbers, drafts (a writer draft, White and Blue issues with marks), locked pages with an A-page, breakdown tags in several categories incl. a lost one, a schedule over two days (after step 8), Notes. Also published for the browser version as File > Open Sample. Next after the schedule step, before the folding item (so folding is tried on it).
- 2026-11-20 (Joop), DONE (branch commands, see the tick log): the Format menu is too long with source items — an Adaptation menu of its own, beside Production (and like it hidden in Markdown mode, except Start a Screenplay from This Source which stays in File for an open .md): Choose Source…, Save Source As…, Keep Source in the Script, Show Source pane (also stays in View) · Link to Source Passage, Next Source Passage, Mark Source Passage as Cut, Suggest Links to the Source, Update Source Links, Source Headings as Sections · What Accolade Learned… . Shortcuts kept. Format keeps formatting only. Small; right after the schedule step (it touches menus.go too), before the sample file.
- 2026-11-20 (Joop), GROUNDWORK DONE (branch commands, the registry; the ribbon and the palette are next): a ribbon-style variant of the menus. Groundwork first (with the Adaptation menu): one command registry (id, label, icon, shortcut, enabled/visible rules, which menu/tab/group) that the menu bar, the header bar's buttons, the format-button row, context menus and shortcuts all read from — so nothing is defined twice. Then the ribbon (Preferences > Look: Menus / Ribbon): tabs Write (elements, formatting, title page), Adaptation (shown when a source exists), Production (its tab follows the pane's stage: Lock / Revise / Break down / Schedule actions), View, File; groups with icons and labels, large buttons for the common actions; one row of tabs + a scrolling group row at phone width; keyboard (Alt shows key tips) and screen-reader names from the registry. The existing second toolbar row (INT./EXT., Character, Color, B I U, Montage …) becomes the ribbon's Write tab in that mode. A command palette (a search over the registry; not Ctrl+Shift+P, which is Preview — e.g. Ctrl+K or F1) falls out cheaply — add it too. After the Adaptation menu, before the sample file.
- 2026-11-20 (Joop), DONE (branch renumber, see the tick log): locking scene numbers must be easy to take back for a writer who plays around. Today: Undo right after Lock removes the numbers; Unlock keeps them. Add: Unlock Scene Numbers… asks "Keep the numbers / Remove the numbers" — Remove is the default while nothing has been issued since the lock (no White yet), Keep once it has (they have gone out); Production > Renumber Scenes (clears A-numbers and gaps, numbers 1…n again) — free before the first issue, after an issue it warns that sides and schedules use the old numbers and offers a draft first; the pre-lock check and the Production pane's Lock stage show "Unlock / Renumber" next to Lock. Breakdown tags and the schedule follow by their anchors (heading+words), so renumbering does not lose them — test that. FIRST, before the Adaptation menu.
- 2026-11-20: PRODUCTION step 8, the schedule (branch schedule, not merged; schedule.go, schedule_state.go,
  stripboard.go, production_schedule.go; Lexington v1.12.0 local tag, go.mod replace; docs/PRODUCTION.md §14).
  Stripboard in the editor's place (built on the cards' drag): strips white/yellow/blue/green by INT/EXT × D/N
  (Wikipedia's Production board, StudioBinder; MMS swaps the nights), number, I/E, location, D/N, eighths, cast
  numbers; days with date/call/notes and black breaks "End of Day 1 — 4 6/8 pages"; the unscheduled pool; drag, or
  select + Move Up/Down (Alt+Up/Down), Move to, Unschedule (the phone's way: two-line strips, a drag scrolls); Group
  by Location / by Cast once; board Undo. assets/production/schedule.json (strips by number, heading, first words;
  a sum of the scene's words flags "Changed in Blue: check" from all issues' marks; lost strips kept). Pane stage 4
  "18 of 87 scenes on a day · 3 days"; Reports: One-liner and Day out of days (MMS codes SW/W/WF/SWF/H; CSV golden,
  gofpdf PDFs); Export Shoot Days (.ics), RFC 5545. Shots: SHOT_DIR=… go test -run TestScheduleScreenshots. Rough:
  no drop/pick-up or travel in the DOOD, no banners. Next: PRODUCTION step 9, day packs.
- 2026-11-20: CI runs TestFocusIsVisible on its own (a step of its own), and skips it in the full and race runs: it failed only inside full runs (Linux race, then Windows plain) on the Find bar and outline filter, never alone. To find someday: what earlier tests leave behind that hides those six widgets' focus. The race skip in focus_visible_test.go (raceDetector) can go once this holds.
- 2026-11-20: easy to take back: Unlock with Keep / Remove, and Renumber (branch renumber, not merged; renumber.go,
  renumber_follow.go; docs/PRODUCTION.md §15). Lock now notes the numbers typed before it (`numbered before the lock:
  7`, or `none`). Unlock Scene Numbers… asks Keep the Numbers / Remove the Numbers. Remove is the default with no
  issue in the drafts store, Keep once one exists. Remove takes out what the lock wrote, keeps the typed numbers
  (named in the dialog), and gives back the script byte for byte when none were typed. An old lock without the line
  loses every number, and the dialog says so. Production > Renumber Scenes (and the pane's button) numbers 1…n in
  order, the lock kept. A-numbers, gaps and OMITTED go: an OMITTED scene's heading goes and its text stays as
  `/* Omitted scene` boneyard. The sources are Wikipedia (Shooting script), John August and StudioBinder (Scene
  numbering). Renumber runs at once before an issue. After one it warns and offers Save a Draft, Then Renumber. Each
  is one undo step, with the page lock's scene numbers in the same step. The breakdown's tags and done marks and the
  schedule's strips (and the board's Undo) follow by place when only the numbers changed. That covers Lock, Remove,
  Renumber, and Undo/Redo, including the editor's own Ctrl+Z after the pause. Strips' sums are carried over, so
  nothing is flagged as changed. Nothing is renamed into a duplicate number (a retyped number waits), and an OMITTED
  scene that went takes its number with it. The pane's Lock stage has Unlock… and Renumber beside "Scene Numbers Locked", and
  its count says "Numbers locked, not issued". Shots: SHOT_DIR=… go test -run TestUnlockDialogScreenshots. Rough:
  the pre-lock check rows don't offer them; anchors keep old numbers as context words. Next: the Adaptation menu.
- 2026-11-20 (Joop): renumbering strategies. Renumber Scenes… becomes a dialog with choices, a preview of old → new numbers before applying (one undo step): Scope: all scenes / only the selected (or a range: from scene X to Y) / from the cursor on. Keep: a list of numbers that stay as they are ("keep these numbers": pinned scenes, e.g. ones already shot or in sides that went out), the others numbered around them (letters where a pinned number forces it). Only: renumber just the listed scenes ("only these numbers"). Letters: fold A-scenes into whole numbers / keep the letters. OMITTED: drop them and close the gaps (today's behaviour) / keep them as OMITTED with their numbers (gaps stay) / merge runs into one "OMITTED 12-14" heading. Sensible presets: "Before anything went out (clean 1…n)", "After issue: keep shot/sent scenes", plus the last used. Tags, strips and locked pages follow as they do now. After the Adaptation menu + registry, before the ribbon.
- 2026-11-20 (Joop, via a Porcupine review on Product Hunt): writers coming from Final Draft expect its keyboard flow; check Accolade against it and close the gaps — Tab on an empty action line starts a character cue (and Tab after a typed ALL CAPS name moves to its dialogue); Enter after a cue goes to dialogue; Tab in dialogue makes a parenthetical, Enter after a parenthetical back to dialogue; Enter on an empty dialogue line goes to action; Enter after a scene heading goes to action; Tab completion (names, headings) and source suggestions keep working; all of it only writes ordinary Fountain. A short "Coming from Final Draft?" note in Help. Also a prominent way to send ideas and bugs (Help > Send Feedback… → a prefilled GitHub issue in the browser, nothing sent by itself). Test with real key events.
- 2026-11-20: the command registry and the Adaptation menu (branch commands, not merged; commands.go). Every
  command once: id, label, tip, icon, keys, when it applies (Markdown / screenplay, pandoc, browser), when it can be
  given (a source, numbers or pages locked: cmdEnv), what it does (menu: only with the editor focused for editing
  commands; button: then focus back; pickers open by their button). Its homes are layouts in the same file: the menu
  bar, the header's top row and editing rows, the editor's context menu; keys not on a menu item (Alt+Enter, F6)
  go through keyPreview. Golden files from main's code (testdata/commands: menus, shortcut tables, full screen's,
  header rows with tips and accessible names, context menu); the test applies the Adaptation moves to them and
  must match. Adaptation menu (screenplay only, before Production): Choose Source…, Save Source As…, Keep Source in
  the Script, Source Pane · Link, Next (Ctrl+Alt+J), Mark as Cut, Suggest Links, Update Source Links, Source
  Headings as Sections · What Accolade Learned… (the pane with its list open). Status lines say Adaptation >.
  Shots: SHOT_DIR=… go test -run TestCommandMenuScreenshots. Rough: the menu bar does not grey by the "can be
  given" rules (the commands say why on the status line); the format's TOML buttons stay data, placed by the
  registry's row. Next: the ribbon and the command palette on the registry.
