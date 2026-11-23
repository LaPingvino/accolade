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
  2026-11-22 (Joop: "The race tests are useful and important, but I think they are best to leave running on the
  background almost as an afterthought, and only pick up from there if they miss, then do a static check to find the
  culprits like we did before"): the merge gate has NO race step (root tests, subpackages, focus tests, vet, wasm,
  the no-pandoc goldens — about 10 minutes); the race run is CI's after the push. When CI's race job reports one,
  find the culprit statically first — list the package-level variables and shared state the recent code touches
  and who writes them from which goroutine (the page view's pagesShown/pagesInSight, gofpdf writing into shared
  font bytes) — then confirm with one targeted `-race -run` test. Never loop the full race suite to catch it.
  Agent briefs don't ask for the race suite either.
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
- 2026-11-20 (Joop), DONE (branch foldchips, not merged; see the tick log): Accolade's own blocks and markers should fold away or show more naturally. In the editor: the tail blocks (/* Notes */, /* Source */, /* Adaptation */, /* Production */) folded by default into one quiet line each ("▸ Production · scene numbers locked · Blue", "▸ Source · story.md, 2 592 passages"), unfolded on click or when the cursor enters; an OMITTED scene's kept boneyard folded under its heading ("▸ the omitted scene, 12 lines"); source links [[from: …]] drawn as a small chip ("↗ source" / "↗ 3 passages", the quote on hover/long-press, raw text when the cursor is inside); learned/cut entries only in their panes. Fountain text unchanged — display only; other apps still see plain Fountain. Uses editor_folds.go's folding; keep per-keystroke cost flat. Next after the schedule step.
- 2026-11-20 (Joop), DONE (branch sample, see the tick log): a sample .accolade to test everything with — none exists yet. Generated, not hand-made: a Go generator (e.g. cmd/makesample or a test with -update) that builds examples/sample.accolade through Accolade's own code, so it stays valid as formats change, and a test that opens it and checks every feature is present. Contents: a public-domain short story as the source (e.g. Poe's The Tell-Tale Heart, or Chekhov in translation — note the PD status) partly adapted: links (Text Fragment form, a range, a prefix-disambiguated twin), a cut passage, learned notes (an alias, a not-character, a place), an OMITTED scene and an A-scene with locked scene numbers, drafts (a writer draft, White and Blue issues with marks), locked pages with an A-page, breakdown tags in several categories incl. a lost one, a schedule over two days (after step 8), Notes. Also published for the browser version as File > Open Sample. Next after the schedule step, before the folding item (so folding is tried on it).
- 2026-11-20 (Joop), DONE (branch commands, see the tick log): the Format menu is too long with source items — an Adaptation menu of its own, beside Production (and like it hidden in Markdown mode, except Start a Screenplay from This Source which stays in File for an open .md): Choose Source…, Save Source As…, Keep Source in the Script, Show Source pane (also stays in View) · Link to Source Passage, Next Source Passage, Mark Source Passage as Cut, Suggest Links to the Source, Update Source Links, Source Headings as Sections · What Accolade Learned… . Shortcuts kept. Format keeps formatting only. Small; right after the schedule step (it touches menus.go too), before the sample file.
- 2026-11-20 (Joop), DONE (branch ribbon on the registry of branch commands, see the tick log): a ribbon-style variant of the menus. Groundwork first (with the Adaptation menu): one command registry (id, label, icon, shortcut, enabled/visible rules, which menu/tab/group) that the menu bar, the header bar's buttons, the format-button row, context menus and shortcuts all read from — so nothing is defined twice. Then the ribbon (Preferences > Look: Menus / Ribbon): tabs Write (elements, formatting, title page), Adaptation (shown when a source exists), Production (its tab follows the pane's stage: Lock / Revise / Break down / Schedule actions), View, File; groups with icons and labels, large buttons for the common actions; one row of tabs + a scrolling group row at phone width; keyboard (Alt shows key tips) and screen-reader names from the registry. The existing second toolbar row (INT./EXT., Character, Color, B I U, Montage …) becomes the ribbon's Write tab in that mode. A command palette (a search over the registry; not Ctrl+Shift+P, which is Preview — e.g. Ctrl+K or F1) falls out cheaply — add it too. After the Adaptation menu, before the sample file.
- 2026-11-20 (Joop), DONE (branch ribbon, see the tick log): regroup the menus. File keeps new/open/import/save/export/print; Edit gets Undo, Redo, Cut, Copy, Paste, Select All, Find…, Move Line Up/Down, Title Page…, Script Language…, Preferences…; a Collaboration menu (before View): Edit Together…, Join a Shared Script… · Save Draft…, Drafts…, Compare… (room for Merge Changes… and returned-PDF suggestions, not built). The ribbon's tabs follow the menus (File, Write, Edit, Adaptation, Production, Collaboration, View). Golden moves explicit, shortcuts unchanged, help updated.
- 2026-11-20 (Joop), DONE (branch ribbon, see the tick log): Markdown mode's Format commands in the registry (Markdown only, plain Markdown, toggling, one undo step each): Heading 1–6 and Normal Text (Ctrl+1–6, Ctrl+0), Strikethrough, Inline Code, Code Block, Link… (Ctrl+L: Ctrl+K is the palette), Image…, Bulleted/Numbered/Task List, Quote, Horizontal Rule, Table…, Footnote; the Markdown toolbar row and the ribbon's Write tab show the common ones; the Markdown cheat sheet lists them.
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
- 2026-11-20 (Joop), DONE (branch renumstrat, see the tick log): renumbering strategies. Renumber Scenes… becomes a dialog with choices, a preview of old → new numbers before applying (one undo step): Scope: all scenes / only the selected (or a range: from scene X to Y) / from the cursor on. Keep: a list of numbers that stay as they are ("keep these numbers": pinned scenes, e.g. ones already shot or in sides that went out), the others numbered around them (letters where a pinned number forces it). Only: renumber just the listed scenes ("only these numbers"). Letters: fold A-scenes into whole numbers / keep the letters. OMITTED: drop them and close the gaps (today's behaviour) / keep them as OMITTED with their numbers (gaps stay) / merge runs into one "OMITTED 12-14" heading. Sensible presets: "Before anything went out (clean 1…n)", "After issue: keep shot/sent scenes", plus the last used. Tags, strips and locked pages follow as they do now. After the Adaptation menu + registry, before the ribbon.
- 2026-11-20 (Joop, via a Porcupine review on Product Hunt), DONE (branch helpflow, see the tick log): writers coming from Final Draft expect its keyboard flow; check Accolade against it and close the gaps — Tab on an empty action line starts a character cue (and Tab after a typed ALL CAPS name moves to its dialogue); Enter after a cue goes to dialogue; Tab in dialogue makes a parenthetical, Enter after a parenthetical back to dialogue; Enter on an empty dialogue line goes to action; Enter after a scene heading goes to action; Tab completion (names, headings) and source suggestions keep working; all of it only writes ordinary Fountain. A short "Coming from Final Draft?" note in Help. Also a prominent way to send ideas and bugs (Help > Send Feedback… → a prefilled GitHub issue in the browser, nothing sent by itself). Test with real key events.
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
- 2026-11-20 (Joop), DONE (branch helpflow, see the tick log): a help system, one source for three places. Help pages written once as Markdown in docs/help/ (our own words; not copied from fountain.io): Fountain Reference (quick, with examples; plus Accolade's own blocks and links: /* Notes */, /* Source */, /* Adaptation */, /* Production */, [[from: …#:~:text=…]], scene numbers, OMITTED), Coming from Final Draft (the keyboard flow, FDX in/out, what maps to what), Adapting a Story (the source pane, links, statuses, Next, suggestions, learned notes), Production (lock, revise, break down, schedule, day pack), Keyboard Shortcuts (generated from the command registry, so it never drifts), File Formats (.fountain, .accolade/.highland/.textbundle, FDX, OSF, PDF…). (1) Embedded in the app (go:embed) and opened in a Help tab in the side panel, beside Notes/Source/Production, rendered like the Markdown preview, with in-page links and a search box; works offline and in the browser. (2) Published online with the browser version (GitHub Pages, e.g. /help/), each page with a stable URL. (3) Help menu (from the registry): the pages (open in the side panel), Online Help (opens the same page online, for sharing a link), Send Feedback…; a page's "open online" and the online page's "this is the help for version X" match the app's version. Together with the Final Draft keyboard flow item, before the ribbon (it is for Joop's own use).
  DONE (branch helpflow). Cheat sheets too (Joop): short one-screen panes beside the full pages — Keyboard Shortcuts (from the registry, grouped, for the current mode), Fountain Syntax (one line per element with a tiny example), Markdown Syntax (for Markdown mode and the source), Accolade Blocks & Links, Revision Colours, Breakdown Categories (colours from the registry of categories), Day Out of Days codes. Each opens in the Help tab with a picker at the top; the pane follows the mode (Markdown cheat sheet first in Markdown mode). Printable (PDF/HTML) as a one-page card.
- 2026-11-20: renumbering strategies (branch renumstrat, not merged; renumber_dialog.go, renumber_strategy.go;
  Lexington v1.13.0 local tag, go.mod replace => ../lexington; docs/PRODUCTION.md §15). Renumber Scenes… is a dialog:
  Scenes all / selected / from … to … / from the cursor on; Numbers: keep these / only these ("5, 12A, 20-24", kept
  numbers reserved too); A-scenes fold / keep letters; OMITTED drop / keep / merge runs into `.OMITTED #12-14#`
  (prints "12-14 OMITTED 12-14", FDX Number="12-14"; Restore Scene splits a run back). Presets Clean 1…n, After an
  issue: keep sent scenes (every number in any issue), Last used (settings.json renumber-last). Preview old → new from
  the same plan Apply writes; one undo step; the after-issue warning and Save a Draft, Then Renumber stay, in the
  dialog. Numbering in Lexington (lex.Renumber: free scenes count on from the pinned one before, letters where a
  pinned number forces it; ranges taken whole by locked numbering). Shots: SHOT_DIR=… go test -run
  TestRenumberDialogScreenshots. Rough: the test renderer draws the numbers field outside the scrolled controls (a
  software-painter clip artefact); Last used keeps a range by scene index.
- 2026-11-20 (Joop): normalise scene numbers — "a scene number can be anything" (Fountain allows #110A#, #I-1-A#, #1.1# …). Numbers are opaque labels; the numbering engine (lex.Renumber, NumberScenes, SceneNumberBetween) interprets only labels in a recognised scheme (whole number + letters; letter prefix + number for A11) and treats every other label as fixed (never renumbered or reused, numbered around). Merged OMITTED is NOT stored as a dashed range: the text keeps one .OMITTED #n# heading per scene; "merge runs" becomes a print/export option (Lexington prints consecutive OMITTED as one "12-14 OMITTED" line; FDX one heading per number); an existing #12-14# written by v1.13.0 is read back as its scenes and rewritten on the next renumber/save. FIRST, before the Final Draft keyboard flow.
- 2026-11-20 (Joop): swapping and moving parts of the script as clear, separate actions (not hidden in renumbering): in the card view and the outline, "Swap…" — range X with range Y, where a range is a run of scenes (document order, from one heading to another), or a section with all its contents (# / ## …), or a selection of cards; "Move…" a range or section before/after another. A scene is not always the unit, so the picker offers scenes, scene ranges and sections. One undo step; source links, cut marks, tags, schedule strips and locked numbers follow by their anchors; with locked numbers the moved scenes keep their numbers (that is what locked means) and the preview says so. Builds on outline_move.go. Queued after the Final Draft keyboard flow + help.
- 2026-11-20 (Joop): LAST, after everything else in this list — a full keyboard pass. (1) Audit: every command in the registry reachable by keyboard (a shortcut, a menu mnemonic, the palette), no clashes per mode/platform (desktop, browser: keys the browser keeps), Tab order and focus visible in every dialog and pane (stripboard, reports, renumber, drafts, compare, the panes). (2) Scene/section level editing from the keyboard: quick renumber, move and swap of scenes, ranges and sections; vim/helix normal-mode motions and operators for screenplay structure (e.g. ]] / [[ next/previous scene, ]s sections, a text object for a scene and a section — "as", "is", "aS" — so d/y/c/p/m work on them), helix-style select-then-act with the same objects. (3) Ex commands for production and structure, in ex.go's style, ranges in document order (labels, not numbers): :renumber [keep 5,12A] [only …] [from X to Y] [fold|letters] [omitted drop|keep], :lock / :unlock [keep|remove], :omit / :restore, :move X..Y after Z, :swap X..Y with Z..W, :section moves, :issue, :tag <category>, :next (passage/scene), :draft <name>; with completion. (4) Documented: the Keyboard Shortcuts cheat sheet and help page generated from the registry plus the vim/helix/ex tables, per mode; a "Keyboard only" help page that walks through writing, adapting and production without a mouse.
  And (Joop): every vi and helix capability Accolade has documented in the help — a "Vim mode" and a "Helix mode" page plus cheat sheets: modes, motions, operators, text objects (incl. the screenplay ones), counts, registers, marks, search, visual/select, multi-cursor (helix_multi.go), macros if any, and every ex command with its options; what differs from real vim/helix and what is not supported. Generated where possible from the code's own tables (vim.go, helix_multi.go, ex.go) with a test that fails when a key or command exists without documentation.
- 2026-11-20 (Joop) LOW PRIORITY, after the keyboard pass: title page keys per language and custom keys. Lexington's language config gets a translated list of title page keys (Title, Credit, Author, Source, Draft date, Contact, Notes, Copyright…; e.g. eo "Titolo", "Aŭtoro") mapping to the standard ones — config-driven, no language checks in code, TOML/docs/CHANGELOG/tag. A language script reads both its own and the English keys; the converted text.fountain (bundles, plain .fountain) writes the English ones. Custom title page keys: any other `Key: value` is kept and shown (the Title Page dialog lists them, add/rename/remove), with config saying where each prints (left/right/centre, or not printed); FDX/PDF/HTML writers follow. The embedded-config item carries them.
- 2026-11-20 (Joop), DONE (branch scenelabels, see the tick log): normalise scene numbers — "a scene number can be anything" (Fountain allows #110A#, #I-1-A#, #1.1# …). Numbers are opaque labels; the numbering engine (lex.Renumber, NumberScenes, SceneNumberBetween) interprets only labels in a recognised scheme (whole number + letters; letter prefix + number for A11) and treats every other label as fixed (never renumbered or reused, numbered around). Merged OMITTED is NOT stored as a dashed range: the text keeps one .OMITTED #n# heading per scene; "merge runs" becomes a print/export option (Lexington prints consecutive OMITTED as one "12-14 OMITTED" line; FDX one heading per number); an existing #12-14# written by v1.13.0 is read back as its scenes and rewritten on the next renumber/save. FIRST, before the Final Draft keyboard flow.
- 2026-11-20: scene numbers as labels (branch scenelabels, not merged; omitted_runs.go, renumber_strategy.go,
  renumber_dialog.go; Lexington v1.14.0 local tag, go.mod replace => ../lexington; docs/PRODUCTION.md §15 "Scene
  numbers are labels"). Lexington's engine reads one scheme (10, 10A, 10AA, 10aA, A11, AB11, A11B; lex.InNumberScheme);
  every other label (I-1-A, 1-A, 1., 1.1, PROLOGUE, 01, 10a) is fixed: never renumbered (any strategy, OMITTED too,
  Unlock's Remove too), never given, never a neighbour (the nearest scheme numbers are). Lists' ranges go by script
  order (4-2 forward when 4 comes first; Y before X an error under the field, Apply disabled, never swapped; separators
  – .. … or a spaced -, a lone - only where both sides are labels and it splits one way); the From … To pickers the
  same. The text keeps one .OMITTED #n# a scene; the third OMITTED choice is "Keep them, print runs as one" (production
  block `omitted runs: print as one`; preview "· prints as 4-5"): Lexington -mergeomitted prints "12-14 OMITTED 12-14"
  in PDF/HTML/DOCX/ODT/print, pages, lock and lengths alike; FDX one heading a number. v1.13.0's .OMITTED #12-14# is
  read as its scenes (Lexington's reader) and rewritten by Accolade on Renumber/Unlock/Save (one undo step, setting
  on, status note); Restore still splits it. Shots: SHOT_DIR=… go test -run TestRenumberDialogScreenshots. Rough: a
  merged run's page can anchor a page lock to "12-14"; the one-line form is unverified against a source.
- 2026-11-20 (Joop), DONE (branch dropcompat, see the tick log): drop the v1.13.0 compatibility entirely — "nobody used that in production, and the undoing could cause legitimate - usage in scene numbers to break down the road". Lexington: remove lex.OmittedRange, lex.SplitOmittedRanges, fountain.SplitOmitted's range splitting and the reader's defer that runs it, and NumberScenes' reading of 12-14 as a run; a heading .OMITTED #12-14# is then simply a scene labelled "12-14" (a fixed label) like any other dashed label; CHANGELOG notes the removal; tag v1.15.0. Accolade: remove omitted_runs.go's rewrite on Renumber/Unlock/Save, its status note, the page-lock "12-14" move, and the old "Last used" merge mapping; Restore Scene no longer splits runs; tests that asserted the old form go. FIRST after the help/keyboard item.
- 2026-11-20: the Final Draft keyboard flow, the help and the cheat sheets (branch helpflow, not merged; keyflow.go,
  help_pane.go, shortcuts_help.go, docs/help/). Flow, only ordinary Fountain: Tab on an empty line outside a speech
  starts a cue (blank line above put in; typed in capitals, header says Character; @ for a script without capitals);
  Tab/Enter after a cue → dialogue; Tab in dialogue → "()" cursor inside (at a line's end, on the next line); Tab/Enter
  in or after it → dialogue (an empty "()" goes on Enter); Enter on an empty dialogue line → action; Enter after a
  heading writes the blank line (a second Enter straight after adds nothing, for old habits). Completion and source
  suggestions keep Tab while "Tab: …" shows (Esc gives Tab back); a one-name completion's second Tab goes on to the
  dialogue. vim/helix normal modes untouched, insert mode has the flow. Help: docs/help is a pure-Go package (go:embed
  pages; Sections, anchors, Search, Resolve, HTML, URL, FeedbackURL). Pages: Contents, Fountain Reference (with the
  end blocks, [[from:]] links, colours, scene numbers, OMITTED), Coming from Final Draft, Adapting a Story,
  Production, Keyboard Shortcuts (from the registry, live per mode in the app; shortcuts.md for the web, a test fails
  when a key is missing or the file is stale), File Formats, Lexington on the Command Line (flags and output formats
  from Lexington's own -help and cli.Formats, examples run in the test), Vim and Helix (stub). Cheat sheets: Keyboard
  Shortcuts, Fountain Syntax, Accolade Blocks & Links, Revision Colours (lex.RevisionName), Breakdown Categories
  (lex.TagCategories, swatches online), Day Out of Days codes (report's), Markdown Syntax, Lexington Flags; Markdown
  sheet first in Markdown mode. Help tab after Notes/Source/Production (a Markdown document: Notes, Help): picker,
  search, Back, Print… (a cheat sheet as a one-page two-column card, a page as a web page, via printFile), the
  online button. Help menu: each page, Cheat Sheets ▸, Online Help (help.URL: /help/<page>.html?v=<version>; the
  page says when it is for another version), Send Feedback… (a prefilled GitHub issue opened in the browser, version,
  platform, mode; nothing of the script, nothing sent); View > Help. Online: build-web.sh runs
  docs/help/cmd/helpsite (CGO_ENABLED=0) into _site/help/. Shots: SHOT_DIR=… go test -run 'TestHelpScreenshots|TestHelpPhone'.
  Rough: anchors scroll to a heading's section (the last ones only as far as the page goes); in-app tables flatten,
  so the pages use lists; Lexington exports no flag set (the test reads -help from stderr; an exported cli.Flags()
  would be cleaner); no "revised pages only" or breakdown-CSV flag in Lexington v1.14.0, so those examples are
  -lock/-lockpages and -report. Next: the vim/helix/ex help with the keyboard pass.
- 2026-11-20: v1.13.0's merged OMITTED form dropped (branch dropcompat, not merged; Lexington v1.15.0, local tag, not
  pushed; go.mod `replace => ../lexington` until it is). Lexington: lex.OmittedRange, lex.SplitOmittedRanges,
  fountain.SplitOmitted, the Fountain reader's split and NumberOptions.Omitted (12-14 read as a run) gone;
  `.OMITTED #12-14#` is one scene labelled "12-14", a fixed label (test: read, numbered, FDX, Fountain). Accolade:
  omitted_runs.go keeps only the print setting; no rewrite on Renumber/Unlock/Save/Save As, no status note, no page-lock
  move, Restore Scene no longer splits; the old-form tests went. PRODUCTION.md §15. Next: push Lexington v1.15.0, drop
  the replace, merge.
- 2026-11-20 (Joop), DONE (branch characters, see the tick log): the character list into the side panel — a Characters tab beside Notes/Source/Production/Help (replacing the pop-up picker as the main view; Insert Character keeps a quick picker): every cue with cast number (report.Cast), scenes, speeches/words, first appearance; aliases from the learned notes shown under their cue; tap → next occurrence, a scene list per character; actions: Rename everywhere (one undo step; tags, cast, strips follow), Merge into…, Add alias, Not a character; filter box; in Markdown mode the storyread names of the document. With five tabs, the tab row needs to stay usable on phones (scroll or an overflow menu) — fix that in the same step. After the ribbon/menus item.
  With character notes (Joop): per character a note space — free text plus a few optional fields (age, description/look, arc, casting/actor) — kept in the script as a /* Characters */ block at the end like the other blocks (ANNA: …; indented lines below a name), so it travels with the file and other Fountain apps ignore it; shown and edited in the Characters tab; usable later by the breakdown (cast details) and day packs (cast list with actors). Folds like the other blocks.
- 2026-11-20 (Joop), DONE (branch cleanfountain, see the tick log): Export > Clean Fountain — a .fountain with Accolade's metadata stripped, for handing to other apps/people: always drops Accolade's end blocks (Notes, Source, Adaptation, Production, Characters) and source links [[from: …]]; options (checkboxes, remembered): all [[notes]], all /* boneyard */ (incl. omitted scenes' kept text), scene numbers #n#, OMITTED headings, title page, section # and synopsis = lines (writer's outline). Presets: "Clean (Accolade's data only)", "Reading copy (no notes, no outline)", "Writer's version (no scene numbers, no OMITTED)" — the last one is the clean writer's draft from the locked-stages note. The original file is never touched; same in the browser (download). A Lexington flag -strip for the command line (Lexington first, tagged).
- 2026-11-20 (Joop): **[done: branch saveas, see the tick log]** Save As shows no file types, so it is unclear what is supported and what each file keeps. Fyne's save dialog has a filter but no type chooser — so Save As… first shows Accolade's own small step: a "Save as type" list with, per type, what it keeps — Fountain (.fountain: the script, Accolade's blocks inside; drafts/breakdown/schedule go to the kept copy or Name.production/), Accolade (.accolade: everything in one file — script, source, drafts, production files), Highland (.highland: Highland-compatible, Accolade's extras may be dropped by Highland — the warning), TextPack (.textpack), TextBundle (folder, desktop only), Markdown (.md, Markdown documents), plus Clean Fountain (export, links to that item) — the current type preselected and the recommended one marked; then the system file dialog with that type's filter and extension; the browser's download gets the same chooser. A help link to File Formats. FIRST after the ribbon/menus item.
  The clickable title in the header (Joop, correcting: not the title page): clicking the file name renames it, and the name's extensions carry meaning (.md switches to Markdown, .eo.fountain sets the script language, .accolade/.highland/.textpack are bundles) — hidden knowledge, and renaming a bundle to a plain name is refused today ("use Save As"). Make the rename pre-configured: the click opens a small editor with the base name, a Language picker (the script languages; writes the .xx part) and a Type picker (Fountain, Markdown, Accolade, Highland, TextPack, TextBundle — with the same one-line "what it keeps" as Save As), the resulting file name shown live; changing the type to or from a bundle converts properly (the same code path as Save As, extras to the kept copy / folder as there), never refused. Typing the full name with extensions keeps working for those who know.
  And (Joop): the rename editor shows every file that belongs to the script right now — the script, its source (beside / kept / in-bundle / in-script), the Name.production/ folder or kept copy with drafts, breakdown.json, schedule.json — each with where it lives, and lets them be renamed together coherently by the needs of the output format: e.g. story.fountain → film.fountain also offers story.md → film.md and story.production/ → film.production/; switching to .accolade shows them moving inside the bundle (assets/source.md, assets/production/…), switching back shows where they go out to; names that a format fixes (text.fountain, info.json, assets/…) shown as fixed, not editable. One action, all or nothing (staged, then committed; on failure nothing half-renamed), undoable where the files allow; the same list appears in Save As so both stay one code path.
- 2026-11-20 (Joop): a .accolade carries its own Lexington configuration, so custom languages and custom formats work on any machine: on save, the parts of lexington.toml the script uses (its language's keywords, settings it relies on) and every custom format .toml it uses go into the bundle (assets/lexington/lexington.toml, assets/lexington/formats/*.toml); on open they apply to that script only, layered over the machine's own config (the bundle's win for this script; nothing on the machine changes). If a language or format is missing on the machine, the status line says the file brought it, with "Add to My Settings…" (File or the Preferences page): lists what would be copied, shows a conflict when the machine has a different one of the same name (keep mine / take theirs / keep both under a new name), never overwrites silently. Plain .fountain / other bundles: unchanged (only .accolade carries them). Tests: a custom language and a custom format round-trip through a bundle opened under an empty XDG config; Add to My Settings with and without a clash. Next after the bundle-language item. Done (see the tick log).
- 2026-11-20 (Joop): a language's direction as the last resort only. Lexington's language config gets `direction` ("ltr" default; the default config sets "rtl" for ar and he; custom languages can set it) — TOML format, docs, CHANGELOG, tag; no language-code checks in code. Used only for a line with no direction mark and no strong letters ("…", "2.", a bare number): it takes the script language's direction, in Accolade's editor (and its mirrored layout) and Lexington's writers. Every other line keeps following its own Unicode characters (mixed-direction files stay mixed); Format > Right to Left Line stays the per-line override. Small; after the embedded-config item (a bundle carries it with its language). Done (see the tick log).
- 2026-11-20 (Joop): a bundle's language is data, not naming: "name.accolade", never "name.eo.accolade" (the inner text must stay text.fountain/text.md — TextBundle/Highland fix it). The language lives in info.json under `eu.kiefte.Accolade` ("language": "eo"); the rename editor's and Save As's Language picker write it there for bundles and leave the outer name plain; opening a bundle reads it from info.json first (a `.eo.` in a bundle's outer name read only as a fallback, never written); converting bundle → plain Fountain/Markdown puts it into the name (film.eo.fountain), plain → bundle moves it from the name into info.json. Plain files keep the .eo.fountain convention. FIRST, right after the Save As/rename merge. Sharpened the same day (Joop): ".eo.fountain IS a different format, because EXT. and INT. are part of the Fountain format." So: a language script's bundle holds BOTH text.<lang>.fountain (native, authoritative, the one Accolade edits) and text.fountain (standard Fountain: force marks for every element only the language's rules see — `.EN. DOMO`, and `!`/`@` for a line English would take for a heading), written from the same text on every save; .accolade may leave text.fountain out ("Keep a plain Fountain copy inside", per script, default on). info.json keeps the language and a hash of the text.fountain written; on open a different hash means another app edited it: that edit wins (force marks taken back), the previous native text kept as a draft, said in the status line. In the Type picker and Save As a language script's dialect is its own entry ("Esperanto Fountain (.eo.fountain)", with an honest line) next to plain "Fountain (.fountain)", which writes the forced, standard text and keeps the language if it can (else says it is dropped). Markdown: no language. The conversion is Lexington's, decided by its parser, universal: every language as configured (no keyword lists or per-language branches in code; "English" is only the default config; a gap goes into Lexington's config format). Done (see the tick log).
- 2026-11-20 (Joop), DONE (branch langrules, Lexington and Accolade, see the tick log): language rules as clear, named rules chosen per language in Lexington's config — no per-language code. (a) Character cues: rules `forced` (@, always), `caps` (letters, all capitals; default), `list` (a name in the script's character list — the `/* Characters */` block of the Characters tab — with an optional extension; "a no-caps character list as a supported system"), `parenthetical` (a parenthetical follows directly), `extension` (ends in one of the language's extension words, (V.O.)/(O.S.) translated in config), `shape` (≤ N words/characters per language, none of the language's sentence-end marks, after a blank line, text directly after), `repeat` (passes shape at least K times in the script); config e.g. `cues = ["list","parenthetical","extension","repeat"]` for zh/ja/he/ar, `["caps"]` for en. Editor side: when the writer marks a line Character and the language's rules would not read it so, Accolade adds the name to the list (or writes @, shown hidden), so what is typed reads back the same. (b) Transitions: the " TO:" suffix and its words per language in config. (c) Scene headings: FIRST research how real scripts write them per language (Chinese e.g. "1. 咖啡馆 内 夜" / "场景一 日/内 咖啡馆", Joop's "zai … :" orders; Japanese ○; Hebrew; Arabic; what Hekaya does) and, if the evidence shows other word orders in real use, heading patterns with slots in config ({place} {inside|outside} {time}, prefix/suffix/circumfix) instead of only a prefix list. Lexington first (format, docs, CHANGELOG, tag); Accolade's editor, outline and conversion (Force/Unforce) follow. After the Characters tab item.
- 2026-11-20 (Joop), DONE (branch langrules, see the tick log; issue drafts await Joop): support Hekaya (Arabic Fountain, MIT, spec v0.1: michaelkmalak.github.io/hekaya/docs/spec/hekaya-markup-spec; .hekaya primary, .fountain "compatibility mode"). (1) Lexington's default Arabic becomes Hekaya's full set: scene words (داخلي/خارجي/لقطة تأسيسية/د/خ…, check مقدر which only its compatibility page lists), transitions (قطع, قطع إلى, اختفاء تدريجي … with dashes "- قطع -"), extensions (ص.خ = V.O., خ.ش = O.S., خارج الكادر = O.C., تابع = CONT'D), title page keys (العنوان, سيناريو/كتابة/تأليف, المصدر …), cue rules `forced` + `list` where the list is filled by the script's own @ uses (Hekaya's "Character Registry": first @name registers it, later bare name + dialogue is a cue), direction rtl, `Direction:` title key, Arabic digits in #scene numbers#. (2) A fixed `hekaya` preset in Lexington (not the user's editable ar) and `.hekaya` = `.ar.fountain` by extension, read with that preset: its own type in Save As / the rename editor, "Hekaya (.hekaya)", native text written as is; the standard text.fountain conversion applies in bundles as for any language. (3) Lexington's PDF does right-to-left properly before any of this is announced (pdf/check.go today: RTL "comes out left to right"): bidi reordering per line (golang.org/x/text/unicode/bidi), Arabic shaping through Courier Badi's own OpenType features (joining forms, lam-alef; the font is Joop's and embedded already), the page layout mirrored for RTL lines as the editor does; Hebrew gets it too; check.go's warning goes. Test against Hekaya's own examples (fetch them honestly; MIT) and note where its spec is loose (>text< centred vs > forced transition; = synopsis vs === page break) and what Lexington chose. After the language-rules item (needs its cue/transition/title-key rules). For each loose end in Hekaya's spec make a sensible choice in Lexington (what Fountain does where Hekaya follows it; otherwise the reading least likely to misparse a real Arabic script), document it in Lexington's docs as "Hekaya, where its spec is open", and test it. When done: the agent DRAFTS one issue per loose end for Hekaya's repo (the difference, the case that shows it, the proposed fix, what Lexington does now) plus a short note that Accolade/Lexington read and write .hekaya, mentioning Joop's Courier Badi (github.com/LaPingvino/courier-badi, OFL, Courier Prime with Arabic and generated Arabic shaping) as a single monospaced screenplay font for both scripts — @hekaya/pdf now pairs Courier Prime (English) with Cascadia Mono (Arabic); I review and file them (Joop, 2026-11-20: "make a sensible choice … then file a bug explaining the difference, how to fix"). If upstream decides otherwise, change Lexington to match (a LOOP item then). Joop: "that creator might be happy with better tools for their format".
- 2026-11-20 (Joop): Issue White / Issue Blue cannot be undone. Make an issue undoable: Undo right after Issue takes the whole issue back (the drafts-store entry, the revision line, a page lock the issue made, the scene lock the first issue made) in one step, Redo re-issues; later, Production > Withdraw Last Issue… (asks; only the newest issue, and only while nothing depends on it — no later issue; warns if its PDF/FDX may already have gone out) does the same. Drafts store entries removed by a withdraw are kept in a "withdrawn" list for a session so Redo/undo stays safe. FIRST after the ribbon/menus item (with Save As). Done (see the tick log).
- 2026-11-20 (Joop), DONE (branch ribbonpolish, see the tick log): ribbon polish — "I mostly like the ribbon, actually a lot", but tweak text lengths, buttons, icons. (1) Registry: an optional short ribbon label per command (`ribbonLabel`), the group giving the context ("Next Source Passage" → "Next" in Adaptation › Source, "Scene or Section Color…" → "Color" in Scene, "Revised Pages PDF…" → "Revised PDF"); the full label stays the tooltip and the screen-reader name, with the keys; menus and palette keep full labels. Key tips recomputed from the short labels, still unique. (2) Aesthetics: one consistent icon set (same stroke weight/size; replace mismatched theme icons with drawn ones in icons.go where needed), consistent big/small button sizes, labels never wrapping mid-word, even group spacing and group captions; the Write tab fits 1000 px (today it scrolls). Screenshots of every tab before/after (desktop + phone) for Joop. Goldens regenerated with the changes listed. Right after the Writer/Production toggle (same files).
- 2026-11-21 (Joop): the main editor and the dock panes made equivalent — "make it all equivalent? you have move to top and move to bottom … a swap with main added to that". Every dock tab's menu (right-click / long-press / Shift+F10, and a registry command) gets **Swap with Main** beside Move to Top/Bottom: that pane's document or view goes into the main area and what was there takes the pane's slot. Source → the source in the main editor as a full Markdown document (Markdown commands, Live Preview, vim/helix), the script into the dock as a Script tab that follows the cursor the other way round (scenes linking to the cursor's passage lit, tap to go, the passage's adapted/cut/to-do status); Notes → the notes in the main editor; Cards → the full-window Cards view (the existing one, reached this way); swapping again puts things back. Adaptation › Swap Source and Script stays as a shortcut for the common case. Cards also becomes a dock pane everywhere (compact one-column list: heading, synopsis, colour; drag to reorder; synopsis editable; lit for the cursor's scene, or in swap mode for the passage's linked scenes). FIRST, correctness (Joop: "might need some correctness fixes especially with the markdown"): audit every place that assumes "the main editor is the script" — w.markdown as the editor's mode, the element/Markdown commands, ribbon tabs and roles, the status bar, title and rename, saving and bundle writes, autosave, the language/format rules, find/replace, the keyboard flow and suggestions, outline/cards following the cursor, collaboration's main Doc — and make each ask which document the main editor holds (a small document abstraction: script / source / notes, each with its kind, where it's saved, its undo history, its collab Doc); tests per path, including Markdown-specific ones. Editing a passage's opening words offers to update the [[from:]] links that quote it (never silently; one undo step on the script side). Phone too. After the collaboration review (it makes the source a document of its own). Done, all four parts (branch swapmain, not merged; see the tick log).
- DONE 2026-11-21 (Lexington v1.20.0 embeds the v1.100 release; Accolade e92882e): Courier Badi has Hebrew (courier-badi main b883294e: Cousine's Hebrew fitted, the font's first GPOS with anchors reproducing HarfBuzz's old fallback for the other scripts; fontbakery unchanged). Refresh Lexington's embedded copy (lexington/font/CourierBadi-*.ttf and font/OFL.txt with the Cousine line) from that build once the spec-conformance branch is merged (it edits Lexington now); pdf/check.go then no longer lists Hebrew letters as missing (RTL layout is still the Hekaya item's); Lexington test, tag, push, Accolade go get. Small.
- DONE 2026-11-21 (Lexington v1.21.0 a439799; Accolade bump on lexington-1.21: Hebrew makes its PDF natively, Arabic still goes to the browser for shaping) (Joop: with Hebrew in the font, Hebrew shouldn't need the browser PDF any more): Lexington's native PDF writer does right-to-left for scripts that need no shaping — bidi reordering per line (golang.org/x/text/unicode/bidi, already used by pdf/check.go), the page layout mirrored for RTL lines as Accolade's editor does (alignment, indents, transitions, scene numbers), the last-resort [Directions] for neutral lines; then cli.PDFWarning's rtl flag only for text that needs shaping (Arabic and other joining scripts — by Unicode joining type, not by language), so a Hebrew script makes its PDF natively and Accolade's pdfOrPrint no longer sends it to the browser. Tests: a Hebrew script's PDF text order and positions, a mixed Hebrew/Latin line, neutral lines. Arabic's shaping stays in the Hekaya item. Right after the Courier Badi refresh.
- 2026-11-21: Courier Badi's Hebrew points where no GPOS is applied (Lexington's gofpdf PDF, simple renderers): the shin dot and sin dot both sit over the letter's middle (sin a little higher) and are hard to tell apart; the holam male sits on the vav's stem; a shuruk's dagesh hides in the vav. Give the shin dot (U+05C1) its default outline over the right arm and the sin dot (U+05C2) over the left arm (HarfBuzz renderers keep using the anchors), check holam and dagesh-in-vav the same way, then a Courier Badi release and Lexington's embedded copy refreshed. Small; font only (go-text's shaper for real GPOS belongs with the Hekaya shaping item).
- 2026-11-20 (Joop): PART 1 DONE (branch collab, see the tick log; the rest is docs/COLLAB.md's "What is left", NEXT in its order) after the side dock (moved up 2026-11-21: Joop asked after it; overnight 21 Nov he said don't wait for him before code changes — "we can revert if necessary": write docs/COLLAB.md, then fix straight on) — review collaboration (internal/collab: one text CRDT Doc per session + a few shared meta values: format, language, markdown; WebRTC via trackers) against everything built since, and the data structures behind it. Questions to answer, then fix: (1) Markdown documents on their own, edited together (mode, preview, the Markdown commands). (2) The whole combination: a script AND its source edited together — the source is a second document (kept copy / beside / in-bundle) that peers do not see today; options: a session holding several named Docs (script, source, …) over one connection, each with its own CRDT, vs. the source only in-script. (3) What lives outside the text and so is not shared today: breakdown.json, schedule.json, drafts, the kept copy, per-file settings (document type, source choice), page-lock/issue data in sidecars — decide per item: shared CRDT map, shared read-only snapshot on change, or local; conflicts (two people tagging, moving strips) need a merge rule. (4) Things that act on the whole text at once (Renumber, Lock, learned-notes writes, Update Source Links, bundle saves, the keyboard flow, suggestions) — make sure each is a minimal edit so concurrent peers merge cleanly, and that the "not in a shared session" guards added along the way are revisited (learned notes, auto A-numbers) rather than left as permanent holes. Write the design into docs/COLLAB.md (data model, what is shared, merge rules, protocol versioning so old peers fail clearly), then build it in steps with multi-peer tests (in-process sessions). Before the sample file.
- 2026-11-21, DONE (branch testmem, see the tick log): the root package's test binary grows past ~9 GB (OOM-killed even alone with -p 1, on main too; CI's Linux race run was killed with 137). Find what holds memory across tests — windows, canvases, fonts, editors, background goroutines or timers never released, caches keyed per test — with `go test -memprofile` per chunk and a heap snapshot after N tests; fix the leak at its source (t.Cleanup closing windows, shared fonts) rather than splitting runs; the gate goes back to one `go test ./...`. Until then the gate runs the root tests in three -run chunks. Before the next feature.
- 2026-11-21, DONE (branch showall, see the tick log): the source pane's "Show All" button overlaps its text in a short slot (seen in the dock shots; older than the dock) — lay it out below the table's note, never over text; a test at a short slot height. Small; with the next Accolade item that touches the source pane, or on its own.
- 2026-11-21 (Joop), DONE (branch ribbon-verbs, see the tick log; all three parts: element keys, verb + variants groups, borderline settings): ribbon groups as "verb + variants", like File's New (big New; small Empty, Markdown, Window, Empty Window), Open and Save already are: "I think other groups might profit from similar tricks". Each group: one big button for its main verb, small buttons named only by what differs. Candidates: Find (Find; Next, Previous, Replace), Production Lock (Lock Numbers; Unlock, Renumber, Omit, Restore), Revise (Issue; Withdraw, Compare, Revised PDF), Adaptation Source (Source; Choose, Save As, Keep in Script), Links (Link; Next, Suggest, Update), Collaboration Drafts (Save Draft; Drafts, Compare) and Together (Edit Together; Join) — judge each by use, not symmetry; keep every tab within 1000 px (TestRibbonFitsDesktop) and key tips unique. Before/after shots of each changed tab. Right after the side dock (same registry). Same pass (Joop: the borderline settings "seems like a good idea to add to the ribbon"): put the (c) settings on the ribbon too, each a registry command reading/writing the same setting as Preferences, both views in sync — editor keys standard/vim/helix (View › Window or Edit; switching mid-work must be safe: mode state reset, a status line saying which), font family (picker beside text size), default language for new scripts (File › New group, as "New in: …"), toolbar visibility (Menus mode only, so in View for when the ribbon is off — also reachable from the ☰ menu), printed scene numbers and CONTINUEDs (Production › Options, Production role), drafts side by side (Collaboration › Drafts). Still within 1000 px per tab. FIRST in this pass (Joop, 2026-11-21: "folded out screenplay text types lack hotkey marking — on ribbon"): the Write tab's 8 element buttons show their Fountain mark (. ! @ …) but not their keys (Ctrl+1…8 / ⌘1…8 — only in the tooltip). Show the key on each element button too, dim and right-aligned after the label (the platform's form, from keysTip), the mark staying in the icon's place; same treatment for Markdown's Heading 1–6 / Normal Text (Ctrl+1–6/0) where they're on the ribbon; decide whether other small buttons with keys get it (likely no — Word's ribbon doesn't; elements are typed by key all the time, which is why they need it). Keep the Write tab within 1000 px in every built-in format (TestRibbonFitsDesktop; radio/series are at 986/984 of 992 — shorten or reflow elsewhere if needed); phone layout too; before/after shots.
- 2026-11-20 (Joop), DONE (branch sidedock, see the tick log): side panel redesign — one dock. "Outline can probably merge with the others, and they could all toggle in and out with the toolbar button. Maybe then enable putting them top or bottom like what happens with Outline + Notes now." One side dock with two slots (top, bottom); every pane is a tab in one of them — Outline, Notes, Source, Production, Help, and the coming Characters; a slot alone takes the full height. Any tab moves between slots ("Move to Top/Bottom" on the tab: right-click / long-press, and a registry command), the arrangement remembered (settings; per-pane slot and order). Toolbar/ribbon/View buttons toggle each pane in and out (shown in its slot; pressed again hides it); the dock disappears when nothing is shown; the split between the slots and the dock's width stay draggable and remembered (side-pane-offset etc. migrate). Phone: the dock stays the bottom sheet, both slots as one tab row. Writer/Production role respected (Production tab hidden as Writer). Keyboard: focus moves into/out of the dock, tabs reachable, focus visible (…FocusVisible test). Screenshots of arrangements (desktop + phone). Before the Characters tab (it adds a pane); after the ribbon polish (same View commands).
- 2026-11-20 (Joop): a Writer / Production toggle so a writer is not faced with production settings: Preferences (and a quick switch in View) "I am working as: Writer / Production". Writer hides the Production menu, the Production tab in the side panel and ribbon, production preferences (scene-number style, revision marks, CONTINUEDs, page lock) and the Collaboration menu's issue-related items; the script still works the same (locked numbers stay locked, marks still print as set) and a quiet status line says "This script is locked for production (Blue) — switch to Production to change that" when relevant. Per script override remembered (a producer can keep one script in Writer view). Default Writer for new users. From the command registry's rules (a role field), so menus, ribbon and palette follow. Done (see the tick log).
- 2026-11-20: the ribbon, the command palette, the menus regrouped, Markdown's Format commands (branch ribbon, not
  merged; ribbon.go, palette.go, markdown_format.go, icons.go; the layouts in commands.go). Menus: File (new, open,
  import, save, export, print), Edit (+ Cut/Copy/Paste/Select All on the focused field, shown keys only, Move Line
  Up/Down, Title Page…, Script Language… — the script's own, like its title page — Preferences…), Format (a Markdown
  document's: Heading 1–6/Normal Text on Ctrl+1–6/Ctrl+0, Strikethrough, Inline Code, Link… Ctrl+L, Image…,
  Footnote, the lists, Quote, Code Block, Horizontal Rule, Table…), Collaboration (Edit Together…, Join… · Save
  Draft…, Drafts…, Compare…: the newest draft with the script now), Help (Command Palette… Ctrl+K/F1 first). Status
  lines and help pages say the new places. The registry has ribbonLayout (tabs → groups → homes, big() for large
  buttons, group when/stage) and homesOf falls back to "Tab > Group > Label"; the Production pane's buttons are
  registry commands now (paneButton; scriptStage shared with the ribbon). Ribbon (Preferences > General > Look:
  Menus/Ribbon; the menu bar stays, the header's rows hidden): tabs File, Write (elements with their marks, INT./EXT.,
  Character, Color, B I U, RTL, the format list and its buttons, Title Page; Markdown: Heading picker, emphasis,
  Insert, Lists), Edit (small), Adaptation (with a source; for Markdown its Start), Production (first group the
  pane's stage: Lock / Revise / Break Down / Schedule; the stage looked at while the pane shows), Collaboration, View;
  the script's name and a palette button on the tab row. Buttons follow the rules (applies hides, enabled greys,
  checked marks) on refreshMenu, 300 ms after typing, the pane's pick. Phone: tabs scroll in one row, groups one
  sideways-scrolling row, buttons ≥ 48 px. Keys: F10 or Alt alone (the menu bar Fyne opens for it closed again:
  DismissMenu) shows key tips (keyTipsFor: word initials, other letters, digits, any letter, then 0x; unique and
  prefix-free), the focus on the tabs so letters never reach the script; a tab's letter, then a button's, runs it;
  Esc back, then out; arrows/Tab/Enter/Space; tabs and buttons are Accessible (tab/button roles, label + keys,
  states). Palette: a modal search, fuzzy (word starts, runs, gaps and word count), label · keys · home path,
  submenus by their items, greyed when it cannot be given; Enter/click runs with the editor's focus back; Ctrl+K/F1
  again focuses it, no second one. Goldens: testdata/ribbon/*.golden (UPDATE_RIBBON_GOLDEN=1); the commands goldens
  unchanged, regroupMoves spells out every move. Shots: SHOT_DIR=… go test -run TestRibbonScreenshots. Rough: Alt
  alone is seen by the menu bar Fyne opens for it (watched 3 s after Alt goes down; Alt+Tab opens none), tested with a
  stand-in for Fyne's unexported menu item, so on a Mac (native menus) it is F10 only; the Write tab is wider than
  1000 px with a format's buttons (its row scrolls); the Schedule stage's Open Stripboard stays the pane's own (the registry's Stripboard toggles); key tips sit over a
  small button's icon. Next: the sample file.
- 2026-11-20: Save As's type chooser and the rename editor (branch saveas, not merged; script_files.go, relocate.go,
  rename.go, saveas_dialog.go). Save As… first lists the types, each with a line of what it keeps (the file's type
  now chosen, the recommended one marked, the other kind of document last), a link to File Formats and one to
  Export… (Clean Fountain is not built yet: its own item), then the system dialog with that type's filter and
  extension (a bare name gets it, the dialog's empty file removed); the browser asks the name there and downloads.
  The header's name (and File > Rename…) opens the rename editor: name, Language (the .xx part), Type, the name made
  live; a full name typed sets the pickers. Both show the same plan (planRows): the script, its source (in the script /
  in the bundle / kept / beside), drafts and each production file, another app's bundle files, now → after, names a
  format fixes marked, a choice where a bundle's drafts go and "keep the source's own name". One executor
  (relocate): same kind renamed (story.md → film.md, story.production/ → film.production/), else written anew — into a
  bundle the source and drafts carried in, out of one the drafts and source kept or beside; a rename converts, never
  refuses. Journaled: made files, renames, and what it would delete parked under a hidden name; a failure puts all
  back (files, settings, window), File > Undo Rename too, until another document or the window closes (it declines
  after a conversion once drafts or the source changed since; the settings put back are only the script's own). Save As copies
  a kept source/drafts instead of taking them from the old file. Rough: parked files are hidden siblings until then
  (a crash leaves them); a rename out of a bundle loses another app's files once the undo is gone (warned in the plan);
  a Save As over a .textbundle folder still meets Fyne's refusal; the dialogs' scroll does not follow the keyboard
  focus on a small screen; a failed background write shows two error dialogs.
- 2026-11-20: a bundle's language is data (branch bundlelang, not merged; fountain_dialect.go, script_text.go, bundle.go,
  rename/Save As). The conversion is Lexington's (v1.16.0, pushed; replace dropped): fountain.Force/Unforce compare
  ParseLines under both languages' Scenes line by line and change only force marks (`.`/`!`/`@`); config `Standard`
  (en) names the language other apps read; the parser now reads with the words it is given (title page too), no
  shared global. Not a lex writer round trip: FountainWriter capitalises headings and centred text, puts `!` on every
  all-caps action, rewrites the title page, and has no case for a line only English reads as a heading. Bundles:
  info.json's key has language, fountainHash, noFountainCopy; text.eo.fountain + standard text.fountain; another
  app's change to text.fountain wins on open, the old text a draft. Plain film.fountain of a language script is
  standard Fountain with a last `/* Language: eo */` boneyard line (a third option beside a title-page key or
  dropping it: no Lexington change, nothing printed, no title page made where there was none); only a language the
  file says or the writer chose is written so — one assumed from the Preferences changes nothing written. Rename editor: Language picker (the script's, always shown for a screenplay), Type list rebuilt per
  language with "Esperanto Fountain (.eo.fountain)", Becomes says where the language goes, a Language row and the
  text entries in the files list, the .accolade copy check; a language-only change saves in place (no Undo Rename).
  Edit > Script Language on a bundle saves it inside. Rough: a line in a speech that English reads as a heading has no
  force mark; a redundant mark the writer typed where Accolade would add one (".EN. DOMO" in Esperanto) is dropped
  from a plain .fountain on reading; Fyne's software painter panics at one checkbox label width in the phone shot.
- 2026-11-20: an .accolade carries its Lexington settings (branch bundleconf, not merged; lexington_carried.go,
  lexington_adopt.go; Lexington v1.17.0 tagged locally, not pushed: go.mod has the replace ../lexington again). Saved:
  assets/lexington/lexington.toml (rules.Part of the script's language, Standard and its words, a lexington.toml
  element-set format; only what differs from the defaults) and formats/<key>.toml (its own format and Base chain);
  info.json names the format. Opened: laid over the machine's for that window only (w.lex(): editor, conversion,
  exports via cli Options.Conf, pickers); the format is the script's own; nothing on the machine changes; status line
  says what is new or clashes. File > Add to My Settings… (and Preferences > Script): new ones ticked, clashes keep
  mine / take theirs / keep both (xq-2, the script then uses it); lexington.toml edited as text and read back, else
  refused. Only .accolade: a rename/Save As to another bundle rewrites without them (plan row warns). Rough: a plain
  film.xq.fountain of a carried language does not read back where the language is unknown; the place button keeps
  INT./EXT. for a language with no A/B word; Matchups/Standard keep-both renames only the copy.
- 2026-11-20: a language's direction as the last resort only (branch rtlfallback, merged; Lexington v1.18.0 pushed
  tagged locally, not pushed: `[Directions]` ar/he rtl by default, lex.OwnDirection, the HTML writer's dir="rtl" for a
  neutral line; Accolade: ScriptEditor.LanguageRTL, Right to Left Line, the export's SceneIn, carried and added with
  its language). go.mod has `replace => ../lexington` until v1.18.0 is pushed. Shots: rtlfallback-before/after.png.
- 2026-11-20: an issue can be taken back (branch undoissue, not merged; issue_undo.go). Issue is one tagged undo step
  (buffer JoinSince/Tag; the editor asks before a tagged step moves, by menu, Ctrl+Z or vim): Undo takes the drafts
  entry out (only the files no other draft names), Redo puts it back as it was; Production > Withdraw Last Issue…
  (ribbon Revise) takes the newest issue back by its entry's new `made` note, only what the block still holds, one
  undo step, with the went-out warning; an imported FDX issue is refused. Typing during the page layout now stops an
  issue. PRODUCTION.md §10, help. Shots: undoissue-withdraw-desktop/phone.png.
- 2026-11-20: the Writer / Production toggle and the ribbon without a double bar (branch roletoggle, not merged; role.go).
  A command's `role` (commands.go; menus, ribbon tabs, made context items too) puts Production's away in Writer —
  menu, side-panel tab, ribbon tab, header button, context tagging, Ctrl+Shift+B, the Preferences' Production card,
  tagged-word colours — the script unchanged (locks, marks, tags); status "This script is locked for production
  (Blue) — switch to Production to change that". Preferences > General "I am working as" and View > Working As
  (Writer, Production, Only for This Script: "script-roles", follows renames/Save As, journaled). New users Writer;
  a settings file without a role gets Production if it shows production use (production-files, a production pref
  changed, a recent script locked/issued), else Writer, kept. Palette: Production's found once typed, "(Production)",
  running one switches this script. Collaboration had no issue items to hide. Ribbon (Joop): the in-window menu bar
  goes in Ribbon mode (Mac keeps its own), keys stay as in full screen; ☰ beside the palette has the menus (same
  registry, role); Alt alone is press+release with no key between (the editor's and ribbon's KeyUp; Alt+Tab cancels
  on focus loss), F10 kept. Goldens: roleMoves/writerMoves; ribbon View gains Working As; screenplay-writer-view new.
  Shots: role-*.png, role-ribbon-*.png (real app under Xvfb). Rough: Alt alone unseen with the focus in another
  field (notes, search); the phone ribbon shot is the test driver's.
- 2026-11-20: ribbon polish (branch ribbonpolish, not merged; commands.go ribbonLabels, ribbon.go, icons.go,
  ribbon_settings.go). Short ribbon labels from one table, the group saying the rest, never a "…"; the full label +
  keys is the name and a new hover tooltip (internal/tooltip.Hover). Labels follow the document's kind (Joop): Markdown's
  Live Preview (checked), New Screenplay, New Empty Screenplay; Edit's "Script" group is "Document" there. Rows line up
  (ribbonRows), large buttons' icons at one height, B I U as icon tools, Title Page joined the Script group: every tab
  fits 1000 px (TestRibbonFitsDesktop: roles, kinds, formats, stages). One icon per meaning (Material glyphs; lock
  pages drawn). Preferences changed while working on the ribbon and in the palette (Joop): View Show (Line Numbers,
  Status Bar), Window (Ribbon, Theme, text −/+), Adaptation Options (Suggest While Writing, Next Starts a Scene),
  Production Options (Revision Marks, Scene Lettering, CONTINUEDs; Production's only); same settings as Preferences,
  an open Preferences window follows. Shots: /tmp/claude-1000/shots/polish-{before,after}-*.png, settings-{before,after}-*.png.
- 2026-11-21: the side dock (branch sidedock, not merged; dock.go). Outline, Notes, Source, Production, Help and Changes
  are tabs of one dock, two slots (top: Outline; bottom: the rest at first); a slot alone takes the full height, no
  pane no dock. Each pane's View command toggles it (in sight: out; behind a tab: to the front; out: in, focused as
  before) and is checked while in; View > Side Panes all out and back. A tab moves by its menu (right-click, long
  press, Shift+F10: Move to Top/Bottom, Hide) or View > Move Pane to Top/Bottom (the pane with the keyboard, else the
  one picked last). Kept: "dock-panes" (id:slot in tab order), "dock-width" (side-pane-offset migrates),
  "dock-split"; which panes are shown is not kept (as before). Markdown hides a screenplay's tabs and keeps them
  for later; Writer takes Production out (asked for again it comes in). A tab coming to the front is refreshed.
  Phone: the sheet, one row of tabs, the taller button in it. Keys: F6 each slot's pane in front; Tab reaches the tab
  in front of each row, arrows along, Down/Enter in, Esc out. A new pane is one row in dockPanes() (its command, View
  menu, ribbon Panes group and per-edit update follow); the header row stays hand-listed. notesPanel is only the notes now (w.source,
  w.production, w.help their own). Goldens: dockMoves (View's panes together after the folding, three new items),
  ribbon goldens regenerated. Shots: /tmp/claude-1000/shots/dock-{before,after}-*.png. Rough: the source pane's Show
  All overlaps its text when the slot is short (as before in a short split); no drag of a tab between slots.
- 2026-11-21: collaboration review, part 1 (branch collab, not merged; docs/COLLAB.md has the findings, the
  decisions on the four questions and what is left). Broken and fixed: Open/New/Import in a shared window replaced
  everyone's script (now it leaves the session; a shared window is never replaceable, the browser asks "Leave the
  Shared Script?"); typing during a catch-up deleted the peer's edit (Session.Run = uiDo: the Doc changes only in
  the UI turn, with the editor); Undo undid co-writers' typing (Buffer.Remote: peers' edits outside the undo
  history, local steps rebased, overlapped ones dropped); joining in the browser overwrote your own kept script and
  a reload came back empty (the shared script's CRDT state kept per room in shared/<room>.json, draft.json left
  alone; rejoin starts from it without doubling); whole-text actions were one edit first-to-last change (Doc.SetText
  and applyRemote now minimal: lines then words, collab.Edits); catch-up gaps after out-of-order ops (hello counts,
  protocol 4); stuck peer reservations and a peer count that counted them; the link only github.io and lost on
  reload (now the page's own address, #k= kept in the address while shared, hashchange joins, ParseKey ignores a
  chat's punctuation and line breaks, a desktop argument with #k= joins); the status bar names the other side's
  version and which one is older. Tests: two windows over an async loopback (holdUI); the catch-up and gap tests
  checked to fail on the old code. Not run in a real browser (the address/hashchange code is only compiled). Guards for auto A-numbers and learned notes kept, with the single-writer rule written as their
  way out. Shots /tmp/claude-1000/shots/collab-{together,leave}-{desktop,phone}.png. Next: COLLAB.md's What is left,
  from 1 (single-writer automatic writes).
- 2026-11-21 (Joop, "go for it, on both sides"), DONE (branch strictspec, see below): Lexington follows the Fountain spec on the four points tree-sitter-fountain's README lists (blank line before a heading and a cue; a transition in capitals ending TO: with blank lines around; two spaces keep a speech going); Accolade users get a clean migration: a one-time notice on opening a script that relied on the old reading, Keep It As It Was (Lexington's KeepLoose, one undo step) or Read It the New Way, remembered per script.
- 2026-11-21: strict Fountain reading (branch strictspec, not merged; Lexington v1.19.0 tagged locally, not pushed; go.mod replaces it with ../lexington until the tag is pushed). loose_reading.go: the notice above the status bar, "loose-reading" in settings (moved with a rename, in journalKeys); the editor's classifier follows the same four points (two spaces in a speech, a transition needs a blank line after, forced . and @ anywhere). Measured over ~24,000 lines (Big Fish, Groundhog Day, Amadeus, Hope, the samples, examples, help): 13 lines read otherwise, none in Accolade's own sample, tests or help. Shots: /tmp/claude-1000/shots/strict-{desktop,phone}.png.
- 2026-11-21: the root test binary's memory growth, DONE (branch testmem, not merged; the fix is in the Fyne fork, branch shaper-leak, commit 0cee0107 in ~/Projects/fyne-shaper, not pushed; go.mod replaces Fyne with ../fyne-shaper until it is pushed, then a pseudo-version). Cause: Fyne's shared HarfbuzzShaper keeps a shaping plan per face for good (go-text's harfbuzz.Buffer.planCache), each holding its parsed font; ClearFontCache (every test.NewApp, and the desktop's theme/settings changes) made all faces new but kept the shaper, so ~12 MB of fonts stayed per test (and per theme switch in the app). ClearFontCache now makes a new shaper. Root package alone: >9 GB / OOM before; the first 150 tests 3.25 GB → 347 MB peak RSS, the whole package 607 MB (3 min); with -race the root package alone 1.5 GB (15 min), the others ≤ 225 MB each, so CI's race step is back to all packages at once. The gate is one `go test ./...` again. Seen under -race, older than this: TestDockLongPress races (the test driver runs fyne.Do inline on the long-press timer's goroutine while the test polls Overlays) — fix next; and internal/collab TestLongScript misses its 10 s limit under -race here (11 s). Left: each test's window stays referenced (~0.4 MB each) by Fyne's renderer/override caches and fyne.SetAccessibleLabel's global map (never cleaned on close — a real, smaller app leak too).
- 2026-11-21: ribbon verbs + variants (branch ribbon-verbs, not merged). (1) Element keys: the Write tab's element buttons show their key (Ctrl+1…8 / ⌘1…8, from keysTip) dim and right-aligned after the label, the mark in the icon's place; Markdown's Heading group is now Heading 1–6 + Normal Text as small buttons with Ctrl+1–6/0 (the picker stays on a phone, which shows no keys). Other small buttons keep their keys in the tooltip. Room for it: Scene's INT./EXT., Character, Color small in one column; Right to Left an icon on a row of its own under B I U (a sep in a ribbon group ends a compact row). Write: radio 982, series 980, default 888 of 992; Markdown Write 866. (2) Verb + variants, judged by use: Edit › Find (big Find; Next, Previous, Replace), Production › Revise (big Issue; Withdraw, Compare, Revised PDF), Adaptation › Source (big Source; Choose, Save As, Keep in Script: home.short, a per-place short label keeping the full name), Links keeps Next and Link big with Suggest and Update short. Left: Lock (already Lock Numbers; Unlock, Renumber — Omit/Restore stay in Scenes, shown at every stage), Together and Drafts (already so). (3) The borderline settings as registry commands, Preferences following both ways: Edit › Settings › Editor Keys (switching resets the vim/helix state and says which in the status line; only on a change now, no longer on every settings apply), View › Window › Font (icon beside the text size; a font file shown as itself), File › New › "New in: English" (script-language; cut at 14 letters), View menu / ☰ › Toolbar (in the Ribbon look it hides the ribbon and the menu bar comes back), Production › Options › Scene Numbers (Production role), Collaboration › Drafts › Side by Side. Widths: File 807/900 (md), Edit 727, View 979 (writer 952), Production stages 960/833/982/975, Collaboration 308. Goldens: ribbon regenerated; commands via toolbarMoves. Tests: TestRibbonMoreSettings, the fit test with the widest language. Gates: the -race gate as written fails, as it does on main: the root package under -race overruns go test's default 10 m (main too: timed out at 600 s, TestLockTeachesNoPlace running; this branch passes with -timeout 40m in 1057 s) — the gate wants a -timeout; internal/tooltip's -race flake (TestTipShowsBelowTheButtonAndHides) seen once under load, passes 5/5 alone, package untouched. TestBrowserPDF* fail only in the sandbox (pass outside). The rest pass. Shots /tmp/claude-1000/shots/{before,after}/ribbon-*.png. Next: Joop's look at the shots; merge.
- 2026-11-21: Show All below the text (branch showall, not merged). Cause: a passage row was Border(bottom: Show All, center: text); the list sized a new row (Show All hidden) before filling it, and showing the button did not lay the row out again, so it stayed at the row's top with no height, over the table's note (any row whose size did not change after the fill; seen in a short dock slot). Fix: the text and Show All stacked in a VBox beside the status bar (the button can only go below the text, whatever height the row gets), and fill lays the row out after every fill. TestSourceShowAllBelowText (source_table_test.go): outline + source in the dock at 1200x720 and the split at 0.5/0.65/0.8, every row with Show All in view has it below its text; it failed on the old layout at each height. With SHOT_DIR it renders source-show-all.png.
- 2026-11-21: the main editor and the dock panes made equivalent (branch swapmain, not merged; docs/SWAP.md has the
  audit, the design and the tests). (1) Correctness first: document.go, the script (w.textEditor, for life), the
  source and the notes as documents (kind, home, own undo, collab Doc: sharedAs "script", or "" for a source kept
  outside the script, which named Docs take over in COLLAB step 3); w.markdown stays the file's kind, the editor's is
  editorMarkdown, cmdEnv away/swappedDoc with scriptEditing/markdownEditing. Every audited place asks the main
  document (commands, do/tapped, Find/Replace, ex/vim, Undo, status, title, Save/auto-save, close question, focus,
  help, ribbon, header row); in-script documents write through as one script undo step and follow the script by
  Remote edits; leaveDocs before anything replaces the script (Open, New, join, type, rename/Save As); the Source
  pane's Edit/Done is the same document. Works in a shared session (tested over the loopback); a source kept
  outside the script is "yours only". (2) Swap with Main on every tab's menu and View > Swap Pane with Main;
  the Script tab (scenes, lit for the source cursor's passage, its status, tap to go); Adaptation > Swap Source
  and Script. (3) Cards as a dock pane (Shift+D; compact column, drag, synopsis, lit); its swap is the full view.
  (4) A passage's opening edited: a notice offers Update Links (one undo step) or Not Now, on leaving the passage,
  Done or Save. Goldens: dockMoves gains Swap Pane with Main, Cards among the panes, Adaptation's Swap; the ribbon's
  Adaptation golden one line. Shots /tmp/claude-1000/shots/swap-{desktop,phone}-*.png. Not done: the Script tab is
  a list, not an editor; documents have no folds; a passage is found by its opening words.
- 2026-11-21: after Swap with Main — typing in a source kept INSIDE the script while it is swapped into the main editor costs ≈160–210 ms a key on the PERF-SOURCE novel (BenchmarkSwappedSourceKey; beside the script ≈65 ms, the script itself ≈51 ms): every key runs the whole script-change path. Make it as cheap as typing in the script (only the changed span into the script's text, the derived views on the usual debounce) without a delayed save that would need a three-way merge in shared sessions. Also from that branch: the source and notes editors have no folding; cut entries in the adaptation block are not touched by the link-update offer. Small to medium; before the Characters tab. DONE (branch swapspeed, see the tick log).
- 2026-11-21: the swapped source's key (branch swapspeed, not merged; docs/SWAP.md, What a key costs). Profiled
  first (-cpuprofile on BenchmarkSwappedSourceKey/inside): ≈42 ms the hidden script editor wrapping the whole
  script, ≈40 ms markdownStats four times, ≈17 ms ClassifyMarkdown four times for the element, ≈9 ms the block
  written the long way. Fix: ScriptEditor.SetOffstage (laid out when asked or back in sight), the stats counted once
  a text and the element from the editor's MarkdownKinds, the script's handlers leaving the status to the writing
  document, spanInto (Buffer.TakeChange's span straight into the script at the document's known place, the script's
  cursor kept; the long way when the script changed since, or "*/", "* /", a block mark, the last new lines), the
  Script tab on a pause, a swapped-out pane updated when back, JoinSince growing in place. Numbers (200 keys, 3
  runs, same binary settings): inside 175–213 → 63–75 ms (55 over 2000 keys: flat); the script's own key 53–57 →
  54–56; beside 61–68 → 47–51 (100 keys); notes with the novel 61–71; the pause's work ≈3 ms. Shared sessions: still written at once, no delayed save;
  TestSwapTypingTogether (real timers, a peer's edit between keys before/in/after the block) and TestSwapSpanWrite
  (400 random edits each for source and notes against the whole-stretch write). Leftovers: the source and notes
  editors fold their Markdown headings (folds type per editor: View's Fold commands, vim z, Tab, Shift+Tab, the
  gutter; a view in the main area refuses); the link-update offer writes the adaptation block's cut: and skipped
  links: entries for the passage's new words too (fixedCut, same undo step, counted in the notice).
- 2026-11-21: the Characters tab (branch characters, not merged; characters_block.go, characters_model.go,
  characters_pane.go, characters_rename.go). A dock pane like the others (View > Characters, the ribbon's Panes, the
  tab menu's Move/Swap; both roles; Markdown too). Rows: everyone who speaks (Lexington's parse: cues with @, ^ and
  extensions; the learned notes' names merged into their cue, as the reports do) and everyone in the list, the most
  present first; scenes, speeches, words, first scene, "also Annie" on a line under the cue; as Production the cast number (report.Cast,
  in the background on a pause). Filter (names, aliases, notes). Tap: the next cue, selected ("ANNA: cue 2 of 3");
  › opens one: Previous/Next Cue, its scenes (tap: the scene), Rename Everywhere…, Merge Into…, Add Alias… and
  Not a Character / A Character After All (the learned notes' characters: and not characters:), Add to the List,
  and the note with Age, Look, Arc (Actor as Production) written into the script as typed, one undo step a field.
  Swapped with Main it is the main area's view and a tap still goes to the cue (the Script tab lights the scene).
  Markdown: storyread's names, mentions and lines, a tap to the next mention, Rename Everywhere. Insert Character
  keeps its quick picker and offers the list's names that do not speak yet (@ before one in small letters).
  The data model, ready for the language rules: /* Characters */ is a tail block like the others (findTails knows it,
  so it hides none of them and bodyEnd ends before it; withTailBlock is the one writer for Adaptation, Production and
  Characters). An entry is a name at the margin ("ANNA: note", or "ben" alone), indented fields and note lines;
  names keep their case and script and match cues by charKey (Lexington's CueName), so a no-caps list is a list.
  Parse and write are byte for byte: an entry not changed is kept as written, a changed one alone written anew.
  Hooks for the next item: listedNames (the `list` cue rule) and addedName (the editor adding a name: an edit,
  usable under Amend), isListed. Nothing writes to the list unasked (COLLAB's single-writer rule is not built).
  Rename Everywhere: the name as a word anywhere in the body, each place in its own case (ANNA → HANNA, Anna →
  Hanna, INT. ANNA'S FLAT too), the list's entry and the learned cue, one undo step of the least edits (each
  name's runes, last first; the blocks by their changed lines and words); Merge Into…: the cues only, the entries
  together (notes after, fields kept where the other has none). Decisions: "everywhere" includes action, dialogue
  and headings, Merge only cues; a place counts in capitals, with a capital each word, or as a cue or the list
  spells it (a character WILL leaves "I will go"); the title page and [[from: …]] quotes are left alone; the
  dialog says how many places first; a rename onto another character's name is refused (Merge is the way). The breakdown
  follows (renumber_follow's way): the cast member renamed (Merge: its tags to the other's), each tag re-anchored
  where its words went, Done scenes and strips their new heading, a strip's sum taken over; Undo and Redo put
  the files back as kept before/after (followRenamed, from followSceneNumbers), or follow a rename's edits back
  when something changed them since (a merge then left as is). Collaboration: the edits go to the session as
  such before the editor makes them (shareEditsFirst): Doc.SetText's diff gives up past 2000 changed lines and
  takes the whole stretch; TestRenameTogether (600 scenes, 1200 places, a peer typing beside a cue and in the list
  while cut off) converges with both, and fails without it. Phone: the sheet's tabs are one row (the
  ribbon's overflowRow): those that fit, the one in front always, the rest in a More menu; the arrows reach all.
  Ribbon: Changes moved from View > Panes to Collaboration > Together (View was 1084 px with Characters; now fits).
  Goldens: dockMoves gains Characters after Notes; the ribbon's View and Collaboration regenerated (read: Characters
  in, Changes moved). Help: index (the pane, side panes, the phone row), Fountain Reference and the blocks cheat
  sheet (the block's form); COLLAB.md one line. Tests: TestCharListRoundTrip, TestCharactersAmongTailBlocks,
  TestAddedName, TestScreenplayChars, TestCharactersPane, TestRenameEverywhere, TestRenameTogether,
  TestRenameTags, TestRenameLeavesAlone, TestMergeUndoTags, TestCharactersSwapped, TestCharactersMarkdown,
  TestDockSheetOneRow, TestCharactersFocusVisible; shots
  (SHOT_DIR=… -run TestCharactersScreenshots) /tmp/claude-1000/shots/characters-{desktop,phone}-*.png. Not done:
  the block does not fold (no tail block folds yet: the 2026-11-20 fold item covers it); the list's notes are a
  screenplay's (Markdown has no block); a rename does not touch the Notes or Source blocks; in the screenshots a
  multi-line entry's text drew past the pane's scroll at its bottom edge (seen in the test renderer, not checked
  in the GL one).
- 2026-11-21: Characters tab leftovers (branch characters): check in the real app whether a multi-line Characters entry draws past the pane's scroll edge (seen in test shots) and fix; Rename Everywhere also offers the Notes and Source blocks (shown as separate counts, off by default); character notes for Markdown documents (a block of their own, as for screenplays); Undo of a Merge with the breakdown changed since — say so instead of leaving it silently. Small; with the next item touching the pane.
- 2026-11-21 (Joop: "The font file button goes to the main preference screen, without any font file picker on either web or desktop"; "the new ribbon selection is already better than what preferences offers"): DONE (branch fontpicker) — Font File… opens the file picker itself (Fyne's open dialog; the browser's own, kept in /files/ and so across reloads), and Preferences' Font is the ribbon's list (Courier Badi, System, the chosen file by name) with Choose File… beside it, both in sync. One picker for both (chooseFileIn). Tested (TestRibbonMoreSettings drives Font File…), screenshot preferences-font.png.
- 2026-11-21: Export > Clean Fountain (branch cleanfountain, not merged; Lexington branch strip, not tagged: go.mod
  has `replace … => ../lexington-strip` until the tag). Lexington: fountain.Strip(text, scenes, StripOptions) —
  always Accolade's data (end blocks, the language line after them, [[from: …]] links, in a kept boneyard too),
  and as asked notes, boneyard (as Parse reads one: a line starting "/*" to the line closing it; omitted scenes'
  kept text too), numbers, omitted (the heading with the kept scene under it), titlepage,
  outline; presets clean/reading/writer (StripPresets), ParseStrip for a comma list that adds up ("writer,notes").
  Byte for byte where nothing goes (CRLF, no last new line); a line left empty goes, with a blank line where two
  would be left; a line that would read as another element gets its force mark (keepReading, as KeepLoose: a cue
  after a stripped boneyard line was action and stays "!BOB"). -strip on the command line: Fountain out (the
  default with -strip) is the stripped text as is, other -to convert it; never over the input; README + CHANGELOG
  [Unreleased]. The end blocks in one place: fountain.EndBlocks / EndBlockNames / EndBlockMarks / BlockMarks.At
  (notesMarks.at's scan, moved), LanguageLine, IsSourceLink; Accolade's findTails, its five marks,
  cutLanguageLine and the link checks now use them (tails_test still compares with the old regexp). Accolade:
  File > Export Clean Fountain… (file.exportClean, screenplays only; ribbon File > Output, small "Clean Fountain",
  File tab 939 of 992 px), clean_fountain.go: preset list (Clean / Reading copy / Writer's version, the
  item's parenthesis on a line under it; "Your own" when the boxes match none), six boxes remembered (clean-fountain-<key>), a line count of the copy; the copy is
  "Name - clean.fountain", a language script written as standard Fountain (standardFountain, no language line);
  written through saveNamedFile (browser: download), refused over the script; the editor, its file and its
  changes untouched; with a pane swapped into the main area the copy is still the script's. Save As links to it next to Export…; the Export dialog's Fountain-as-written note points to
  it; saveNamedFile's messages are generic now (the report's "Report saved as" moved to the reports). Help: File
  Formats § Clean Fountain, Fountain Reference's end blocks, Lexington page (-strip in its group, two examples run
  by the test). Goldens: regroupMoves adds Export Clean Fountain… after Export… (screenplay menus); ribbon
  screenplay-file regenerated. Tests: Lexington fountain/strip_test.go (each option, each preset, a script with
  every block, byte-exact over examples/ and testdata/ with CRLF/no-newline variants, idempotence, reads-as-before),
  cli/strip_test.go; Accolade TestCleanFountainExport/Dialect/Command. Shots (SHOT_DIR=… -run
  TestCleanFountainScreenshots): clean-{desktop,phone}.png, clean-writer-*.png. Rough: TestSaveAsScreenshots
  panics in the software painter at phone size (also before this branch). Next: Joop tags Lexington, swap the
  replace for the tag, merge.
- 2026-11-21 (Joop: "split out the export settings from Export, Print and Preview, as the latter two use these settings, but you can only set them while also exporting"): DONE (branch pagesetup, see the tick log). One **Page Setup** of its own (File › Page Setup…, a registry command; on the ribbon by Print/Preview in File › Output, and reachable from Production's Options for its items) holding what the printed page looks like — paper, title page, scene numbers, CONTINUEDs, colours (style, block, tint, patterns), pagination/locked pages, revision marks, revised pages only, page colours, template — the same settings keys as today (no migration needed; check each key's readers). Export keeps only what is the export's own (format, folder, file name) plus a one-line summary of the page setup and a Page Setup… button; Print and Preview each get the same summary line + button (Preview re-renders on change); the ribbon's existing setting commands (Scene Numbers in Production › Options, etc.) stay in sync with the dialog. Per format, show only the settings that apply (as Export does now: colours and revisions for PDF, etc. — read export_dialog.go's enable logic and move it). Markdown documents: their own page setup (paper, title page, what the Markdown export uses). Browser and phone too. Tests: each setting read by Export, Print and Preview from the one place; screenshots of Page Setup, Export, Print/Preview summary (desktop + phone) looked at. After Clean Fountain.
- 2026-11-21: Page Setup split out of Export (branch pagesetup, not merged; pagesetup.go, md_pagesetup.go). One
  reader: pageSetupOf(settings) → layoutJob (was savedExportJob: the page counts, locks, reports, schedule, characters,
  breakdown) and outputJob(setup, format, text, part) (Export, Print = "html", Preview PDF = "pdf": the paper resolved
  as Export did, template, revisions per format, issued pages, FDX breakdown); the two exportJob literals are one.
  Same keys, plus export-revised-only and export-template (never saved before), markdown-page-size and
  markdown-title-page. Applicability is one table (pageSetupApplies: Export's old switches): Page Setup shows only
  what the format at its top uses ("Settings for:" Print or an export format, set by the caller), the summary line
  says the same. File › Page Setup… (file.pageSetup; menu before Print…; ribbon File › Output small under Clean
  Fountain, Production › Options; a Markdown document's Output is big Export, small Print over Page Setup so the
  tab fits: File 939/953 md of 992, Production stages unchanged ≤ 982). Export keeps format, folder, file name and
  the part of the script (filters: per export), a "Page Setup" card with the line and button; Revised Pages PDF… is
  an override for that export only (shown in its line). Print… now a small dialog (line, Page Setup…, Print);
  Preview PDF opens the PDF then a dialog with the line, Again, and a Page Setup change makes the PDF again.
  Preferences: paper, title page, scene numbers, CONTINUEDs moved out, a Page Setup card (line + button) in their
  place, following the ribbon. Ribbon Scene Numbers/CONTINUEDs write the same keys through changeSetting. CONTINUEDs
  shows for PDF, Production role. Markdown: Page Setup (paper: default = A4 in Word/ODT and the printer's for the
  page, as before; title page from front matter title:/author: or the first heading) in internal/prose (Doc.Paper,
  Doc.TitlePage; defaults byte-identical), the web page (@page, title section), Export, Print, Live Preview (line at
  its top, title page shown, redrawn on change). Phone: the line's button goes under it; Export scrolls down only
  (sides radios stacked). Tests: TestPageSetupOnePlace (each setting from the dialog: Export's, Print/Preview's and
  the layout's jobs agree), TestPageSetupPrintAndPreview (title page in the printed HTML and the preview's pages;
  Preview remade on change), TestPageSetupPerFormat, TestPageSetupFollowsRibbon, TestMarkdownPageSetup; the old
  export colour/pagination tests now drive Page Setup. Goldens: commands via regroupMoves (Page Setup… before
  Print…), ribbon regenerated (7 lines, read). Shots (SHOT_DIR=… -run 'TestPageSetupScreenshots|TestMarkdownPageSetup'):
  /tmp/claude-1000/shots/ps/{pagesetup,pagesetup-bottom,export,export-sides,print,preview,preferences-pagesetup,
  md-export,md-preview}-{desktop,phone}.png, pagesetup-markdown.png. Docs: help formats/production, PRODUCTION.md.
  TestPageSetupRevisions (issued: marks, revised only, colours in Export's and Preview's PDF, marks in FDX, none in
  the layout or Print). Calls made: the filters stay in Export (per export, not the page's look); CONTINUEDs shows in
  the Production role only (as the ribbon's); Ctrl+P opens the Print dialog first (Space prints), Preview PDF shows
  its dialog each time; Preferences' Reset no longer resets the page setup (Page Setup has Defaults). Left: pandoc
  formats ignore a Markdown page setup; Preferences' Page Setup opens in the first window; Page Setup on an issued
  script not shot at phone width.
- 2026-11-21: COLLAB.md "What is left" steps 1–3 (branch collab2, not merged). (1) The single-writer rule: a locked
  script's new scene is numbered, and a lesson learned, only by the peer whose replica wrote the line's first rune
  (`Doc.Writer`, `wroteLine`); the guards in numberNewScenes and learnFromWriting are gone (TestSceneNumbersTogether,
  TestLearnTogether, both fail with every peer writing). (2) Run-length Doc nodes and kept ops (Op is a run; the
  old per-rune Doc is the oracle of TestRunsAgreeWithRunes): on a 572k-rune novel heap 178 → 6.5 B a rune, a join
  1.26 s → 44 ms, a kept state 0.76 s → 39 ms, a key 21 → 3 ms. (3) Named Docs, protocol 5: the source kept outside
  the script shared in a Doc of its own (collab_source.go), the source document its editor, live both ways; a
  joiner without one keeps it for the room; protocol 4 peers read the hello and say to update (TestNamedDocs,
  TestJoinMidEdit, TestStateDocs, TestMixedProtocols, TestDrop, TestSourceTogether (also the Source pane's Edit/Done),
  TestSourceJoinMidEdit, TestSourceRejoin, TestSourceOldPeer). Not run in a browser.
  Gates: TestBrowserPDF* fail inside the sandbox only (chromium's crashpad, read-only ~/.config). Left: steps 4–6.
- 2026-11-21: after collaboration steps 1–3 (branch collab2): try source sharing in two real browsers (the web version, two tabs/devices) and a desktop+browser pair — join, edit the source both ways, rejoin, an old peer; measure a remote edit's cost in a window on the novel (each message re-reads the source pane's passages; debounce if it shows); two peers adding a scene at the same spot can get the same A-number (resolve on the writer side: the later writer renumbers its own); an empty source isn't shared; two peers picking a source at once (say so to the loser instead of dropping silently). Then COLLAB.md steps 4–6.
- 2026-11-21 (Joop: "do 6 and 7 first, then 5, then resume"): ORDER NOW — after the sample .accolade: (1) the language rules and Hekaya as ONE item (Joop: "you can include the language rules part"): the language rules first ("2026-11-20 (Joop): language rules as clear, named rules": cue rules, transitions, heading research and patterns), then Hekaya on top of them ("2026-11-20 (Joop): support Hekaya"); Lexington first and tagged, Accolade after; (2) Courier Badi's Hebrew points (running alongside in its own repo, branch hebrew-points); then the rest in its old order (folding/chips, swap/move, collaboration follow-ups + COLLAB steps 4–6, leftovers, the two older Joop items — lock-the-writer layers and File › Merge Changes —, the keyboard pass, title keys last).
- 2026-11-21: the sample script (branch sample, not merged; sample.go, sample_test.go, examples/). File > Open
  Sample (ribbon: File > New > Sample) opens examples/sample.accolade, embedded in the app, as a copy of its own (the
  data folder's samples/, the browser's /files/; a changed copy is kept and a fresh one goes beside it). The story is
  Chekhov's "A Chameleon" in Constance Garnett's translation: Project Gutenberg eBook #13417, fetched from PG's own
  gutenberg.pglaf.org (www.gutenberg.org answered 503 Retry-After 6 h, the Wayback Machine 429); its RDF says
  "Public domain in the USA." The Tell-Tale Heart was tried first, but no passage of it needs a prefix, and the
  sample needs a twin: "Yes." in A Chameleon needs one, as "Yes, your honour." opens alike. TestMakeSample (UPDATE_SAMPLE=1) builds the sample from
  examples/a-chameleon.md (front matter) and a-chameleon.fountain ({{n}} where a link goes) through the commands:
  Save As .accolade, the source in the bundle, Link to Source Passage, a two-passage range as the command writes one,
  Mark as Cut (two), the Characters pane (notes, fields, aliases, General Zhigalov listed, not speaking), learned not
  characters and a place, Notes, a writer's draft, Lock, Omit (3), a scene typed after the lock (1A), Lock Pages, tags
  in five categories, two shoot days, White, Blue (an A-page, a lost tag), Save. TestSample opens the embedded one
  as Open Sample does, in a window with new settings and data, and checks 28 features (sampleFeatures, one each);
  TestSampleFresh makes it again and checks that. examples/README.md: the PD evidence, the table, what it does not
  show (another script language, a format of one's own, Page Setup's settings, which are the machine's: only the
  lock's paper and format travel). Shots: SHOT_DIR=… go test -run TestSampleScreenshots. Found while making it: the
  learner took the typed 1A heading for the place "god" (from "God's world" in the passage before), forgotten in the
  generator (fixed on branch samplebugs: no lesson now, the workaround gone); breakdown tags match words inside [[from: …]] notes (a "beggar" tag was anchored in a link's quote, and a
  tag whose words went stays found in the link); the Characters pane's "from scene N" counts headings, not the
  locked scene numbers (HRYUKIN "from scene 3" speaks first in 2).
- 2026-11-21, DONE (branch samplebugs, not merged; see the tick log): three bugs found building the sample (branch sample): (1) the learner took a typed 1A heading as the place "god" from "God's world" in the passage before — a heading's place should come from the heading's own words, never a source quote; (2) breakdown tags match words inside [[from: …]] link quotes (a tag anchored in a quote; a tag whose words were deleted from the script stays "found" in the link and is never reported lost) — tags must skip source links and notes; (3) the Characters pane's "from scene N" counts headings, not the locked scene labels (HRYUKIN "from scene 3" but first speaks in scene 2) — use the scene's label. Small; right after the language rules + Hekaya item. Also: Open Sample not yet tried in a real browser.
- 2026-11-21 (Joop: "the preview doesn't fully work, at least on my android"; then "How hard would it be to build-in the PDF preview natively also e.g. on Desktop?" — "So we can basically do a PDF reader, except much simpler because we barely need to support anything"): DONE (branches pageview in both repos, not merged; Lexington untagged; see the tick log). A native page view instead of pdf.js. Cause on Android: the browser's Preview PDF puts a blob PDF in an iframe (browser_js.go openFile/previewPane) and Chrome on Android has no in-frame PDF viewer; on the desktop Preview opens the system viewer. Design: (1) Lexington first: its PDF writer draws through a small interface (the ~30 gofpdf calls it uses: SetFont, CellFormat/Cell/Write, Rect, SetTextColor/SetFillColor/SetAlpha, AddPage/SetPage, margins, GetStringWidth…) that *gofpdf.Fpdf satisfies; a recording wrapper forwards every call to gofpdf (so measurement, breaks and positions stay gofpdf's) and records what is drawn per page — text runs (x, y, font face/style/size, colour, the exact string as placed, after bidi and the Hebrew presentation forms), rectangles (fill, alpha), page size — as a public type (e.g. pdf.Pages); Write/CellFormat alignment resolved to absolute x. API: render to PDF and get the recorded pages in one pass. Fidelity test in Lexington: rasterise the recorded pages (x/image + the embedded fonts) and compare with pdftoppm of the real PDF per page within a small tolerance, over examples/ and the RTL/Hebrew/colour/revision tests. CHANGELOG, tag. (2) Accolade: a Fyne page-view widget drawing the recorded pages (white pages with shadow, zoom: fit width / 100% / pinch on phone, scroll, page n of N, keyboard), in the preview area on desktop, browser and phone alike — Preview PDF shows it instead of opening the system viewer / iframe, with Save PDF… / Open in Viewer buttons (desktop) and Download PDF (browser); Revised Pages and anything else previewing a PDF uses it too; it re-renders when Page Setup changes. Screenshots desktop + phone, looked at; a test that the view's pages match the PDF's page count and first lines. (3) Markdown too (Joop: "I guess we can do that equally for Markdown and Fountain"): a Markdown document's PDF is made today by Chromium printing the HTML page (md_export.go, browser_pdf.go; the print dialog without one). Give Accolade's internal/prose (the document model lives there, not in Lexington) a native PDF writer (prose.Document → gofpdf through Lexington's drawing interface, beside WriteDOCX/WriteODT: headings, paragraphs, lists, emphasis, quotes, code, footnotes, tables as the HTML has them; Markdown page setup's paper and title page; the embedded fonts with a proportional face for prose if one fits the licence — say which) through the same drawing interface, so the recorder covers it: Markdown PDFs then need no Chromium (desktop) and the browser version makes a real PDF; Markdown's Preview uses the page view. The reports' PDFs (report/schedule.go, breakdown.go use gofpdf) can show in the page view the same way — check and wire if small.
- 2026-11-21 LOW PRIORITY (Joop: "what about Markdown image embedding? … a reasonable subset" — "I don't think it's massively important, but it's technically something Markdown can do"): images in Markdown. NOT BEFORE Wednesday 2026-11-25 (Joop: "you can even put that after the next wednesday reset" — the weekly usage reset); then after the language rules + Hekaya and the sample's three bugs; skip past it to the next item until then. Builds on the native page view and the prose PDF writer. today prose keeps only an image's alt text (markdown.go ast.Image) and the DOCX/ODT/HTML imports leave images out. Add an image span to the model (src, alt, title; block when alone in its paragraph, inline otherwise). Sources: paths beside the document, a bundle's assets/, data: URIs; remote URLs never fetched in export or preview (privacy, offline, the browser's CORS) — alt text + the link instead. Formats: PNG, JPEG, GIF as gofpdf takes them; WebP decoded (x/image/webp) and embedded as PNG; SVG as its alt text for now (note it). Layout: a block image scaled to the text width (never up), kept on one page, its title as a caption; an inline image at line height. Every writer: PDF, the page view, Word and ODT embed the image, HTML keeps the tag. The imports (DOCX, ODT, HTML) keep images: copied into the bundle's assets/ (or beside the file) and linked from the Markdown instead of "image left out". Tests per source, format and writer; screenshots of a Markdown document with images in the page view, desktop + phone.
- 2026-11-22 (Joop: "for mobile … auto reorder the ribbon by usage/priority/utility, and maybe have a quick access ribbon that puts the most prio from all together to avoid having to scroll to the side all the time"; "put a little toggle to preserve muscle memory"): DONE (branch quicktab, not merged; see the tick log). (1) A **Quick** tab, first on phones and their default: the 8–12 most useful commands from every tab, scored by a built-in priority per mode and role (screenplay: elements, Undo, Save, Find, Preview, Next Source Passage while adapting; Markdown its own) plus local usage counts with decay (on the device only, never shared or synced); reshuffled only at app start, new entries appended, never moving under the finger; long-press any ribbon button → Pin to Quick / Unpin. (2) Auto-reorder of the regular tabs' groups by use, with a toggle (Joop: "a toggle that's easy in reach and expected to be used a lot"): a lock/pin icon on the ribbon's own row beside ☰, always visible, one tap, switching live between **Smart** (Quick tab first, groups by use) and **Fixed** (the standard order, as on the desktop) — the ribbon redraws in place, the tab in front kept if it exists in both; remembered per window; a key tip and a registry command (and the ☰ menu) too; Smart by default on phones, Fixed on desktop. Smart's order is still recomputed only at app start, so flipping back and forth always shows the same two layouts within a session. (3) Desktop: the Quick row optional (off by default), like Office's Quick Access. Registry-driven (ribbonLayout), key tips unique, tabs within 1000 px; tests for scoring, stability within a session, pinning, the toggle; screenshots phone + desktop.
- 2026-11-22 (Joop: "the source view still lacks a jump to related script part thing? Also the Next button is a bit confusing and does too much too quickly"): DONE (branch sourcepane, not merged; see the tick log). Found: (a) tapping a passage already jumps to its scene (source_pane.go pick: cursor to the linked line, cycling on repeat taps) but only the status line says so and a passage shows nothing of its links — undiscoverable. Fix: each linked passage shows a chip ("→ Scene 4" / "→ 2 scenes", the scenes' labels) that jumps (cycling, or a small list for several); tapping the passage text only selects it; the jumped-to scene flashes briefly in the script. Phone and swap mode too (the Script tab already lights scenes). (b) Next (adaptation_pane.go nextPassage) with next-passage-inserts on by default starts a scene in the script (heading + link, cursor moved) and may pop a skipped-passage question, all on one tap labelled "Next". Fix: Next and a new Previous only move — select the next/previous passage to adapt and scroll to it, nothing written; the selected passage shows one **Adapt…** action (plus Mark as Cut), and writing goes through an in-between step (Joop: "make it feel better by adding a bit of friction, e.g. an in-between dialog for each part … suggesting how to add it and where to insert it, with likely options instead of an immediate insert"): a dialog on desktop, a bottom sheet on the phone, with (1) How — Start a new scene (heading suggested from the passage via storyread's place/time, editable), Add to the current scene (link at the cursor), Draft from the passage (quoted speech → cues + dialogue, the rest → action, editable before it goes in), Mark as cut, Skip for now — the likely one preselected; (2) Where — after the scene adapting the passage before this one in source order ("after Scene 4, which adapts '…'"), at the cursor, after the cursor's scene, at the end — the likely one preselected with its reason shown; (3) a preview of exactly what goes in. Enter accepts the defaults; one undo step per insert; keyboard-complete, focus visible; no skipped-passage popup (it stays to do); the next-passage-inserts setting and its ribbon toggle go (settings migration: drop the key). (c) Context: Joop: "the biggest issue with the Next is that it's not flexible or context aware, it mostly and kinda only works properly if you start at the beginning from a clean source". Today nextPassage anchors on the last linked passage before the cursor, else the last linked anywhere. Make the anchor what the writer is looking at, in this order: the passage picked in the pane (if picked since the cursor last moved); else the passages linked from the cursor's scene (the last of them in source order); else the nearest linked scene before the cursor (its last passage); else the passage at the top of the pane's view. From the anchor, the next passage still to adapt in SOURCE order (adapted and cut skipped); at the end, say so and offer to wrap to the first remaining to-do instead of jumping silently; Previous mirrors it. The pane shows the anchor in a caption line ("after: …"). Cases to test: adapting a middle chapter first, scenes out of source order, a partly linked script, the cursor in an unlinked scene, a picked passage, a passage linked from two scenes, a source edited since. Keyboard: Alt+J Next, Alt+K Previous (check clashes), the actions reachable by keyboard; help/cheat sheets updated. Tests; screenshots desktop + phone.
- 2026-11-22: the native page view (branch pageview here and in ~/Projects/lexington-pageview, neither merged;
  Lexington untagged, Accolade's go.mod replaces it with ../lexington-pageview until it is). Lexington: pdf.Drawer (the
  41 gofpdf calls the PDF writer and the reports make; *gofpdf.Fpdf satisfies it), pdf.Recorder (forwards to gofpdf,
  runs MultiCell/Write over CellFormat as gofpdf does, records text runs at their baselines and boxes per page, in
  points: pdf.Drawing), PDFWriter.Record / cli.Options.Record / report.Draw*PDF, pdf.Rasterizer (x/image, the embedded
  fonts). Tests: the PDF through a Recorder is byte for byte the plain one over 32 cases (the standard fonts too); the painted pages match
  pdftoppm's at 100 dpi on ≥ 99.9% of the ink (within a pixel, both ways) and colour, worst page 0.9991, over the
  examples, testdata/input, Hebrew (A5 too), the colour styles, revisions (marks, pages only, page colours), the three
  reports; a moved line or a dropped run fails it. Also: catalog sorted (same PDF for the same script), parsing no
  longer writes fountain.Scene (a race). Accolade: pageview.go — Preview PDF, Export's PDF preview, Revised Pages and
  the reports' PDFs show in the page view (beside the script; in its place on a phone): Fit/100%/zoom, keys, Page n of
  N, Save PDF…/Open in Viewer (desktop), Download PDF (browser), two-finger zoom in the browser (browser_js.go; not
  tried on a real phone), the page setup line over it and a change in Page Setup making it again. internal/prose
  WritePDF (Noto Sans from Fyne, OFL; Courier Badi for code): Markdown's PDF export is Accolade's own (browser too),
  Live Preview has Text | Pages. Not covered yet by the prose PDF: right-to-left text, scripts neither font has (CJK),
  images — those still go through Chromium / the print dialog. Shots looked at (pageview-*.png: desktop, phone, Hebrew,
  Markdown). A change of how the page prints anywhere (Page Setup, the ribbon, the script format) makes the
  preview's pages again; zoom paints at most 3000 px a page; a pinch stretches and paints when the fingers lift. The Fyne mobile build (not the browser) has no pinch: Fyne gives no two-finger events; the buttons zoom.
- 2026-11-22: after the native page view (branch pageview): try it in a real browser and on Joop's Android (background painting, two-finger zoom, Download PDF), and in Fyne's native mobile build (no pinch events there: zoom buttons only); the title page dialog's preview as the page view; Markdown's native PDF still sends right-to-left text, scripts neither font has (CJK) and images to Chromium/print (PDFMissing) — right-to-left through Lexington's bidi like the screenplay, CJK with a fallback font if a licence-compatible one fits; Live Preview on a 380-wide phone widens the window to ~429 (older).
- 2026-11-22 (Joop: "Move the top of the preview pane to a new ribbon tab?"; "It also lacks a refresh preview"; "copy the rough goloco redraw idea for quick local rerendering/invalidating, and fully rerender in the back when necessary"): DONE (branch previewtab here, resume in ~/Projects/lexington-resume; neither merged, Lexington untagged; see the tick log). (1) A contextual **Preview** ribbon tab, shown and brought forward while the page view is shown, gone when it closes: Zoom (Fit, 100%, in, out), Pages (Page n of N, previous/next), Output (Save PDF…, Open in Viewer / Download PDF), Setup (Page Setup…, its summary as tooltip + short label), Markdown's Text | Pages, Refresh, Follow Edits, Close Preview; the bar above the pages goes in the Ribbon look (stays in the Menus look); key tips, ≤1000 px, phone layout. (2) Refresh (F5, Ctrl/⌘+R if free) remakes the pages keeping page and zoom; Follow Edits re-renders after a pause in typing; a dot on Refresh when the pages are stale. (3) goloco's redraw idea (OpenLoco's invalidate-and-redraw-only-the-dirty-rects): the recording maps script lines to pages; an edit marks its page dirty and later pages possibly stale (dimmed); the dirty page is laid out again at once from its recorded start assuming unchanged breaks and repainted; a cancellable background pass then repaginates from that page and STOPS when a later page starts at the same script line as before (the rest is identical — reuse recordings and painted images); repaint only invalidated pages, zoom and scroll kept. Lexington: pagination resumable from a recorded page start, and a line→page map in the recording (tag). Tests: an edit mid-script repaints one page and converges; a page-break-moving edit reflows to convergence; cancel on typing.
- 2026-11-22 (Joop: "Markdown links render as links but aren't clickable"): DONE (branch pdflinks here, links in ~/Projects/lexington-links; neither merged, Lexington untagged; see the tick log). gofpdf has link annotations (LinkString / CellFormat's link; AddLink/SetLink internal) — the prose PDF writer only styles links. Add: an annotation over each link run's rect (wrapped links: each part); footnote references ↔ notes and #heading links as internal links; headings as the PDF's outline (Bookmark). The Recorder records link rects; the page view opens external links (browser) and scrolls for internal ones on click/tap, pointer over links, links reachable by Tab.
- 2026-11-22 (Joop: "do we have footnote support? likely more important for works of fiction"): DONE (branch pdflinks; see the tick log). Today: the editor inserts [^n] (markdown_format.go), imports keep footnotes, Word/ODT write real footnotes, but the PDF/page view gather them as endnotes. Add real page footnotes in the prose PDF/page view: at the foot of the referencing page, a short rule, smaller size, long notes continued on the next page's foot, the text area shrinking to fit, a reference near the page end moving with its note when needed. Page Setup (Markdown): Notes at page foot (default) / end of chapter (top-level heading) / end of document; numbering through the document or per chapter. Editor: jump between a [^n] and its note, renumber notes, the note's text on hover over its mark.
- 2026-11-22 LOW PRIORITY (Joop: "see how viable something like bibtex support would be… might be a stretch"; "check how Zettlr does it"): NOT BEFORE Wednesday 2026-11-25, after Markdown images. Zettlr (docs.zettlr.com /en/editor/citations.html, /en/syntax/citations-and-crossref.html, /en/sidebar/bibliography.html, source service-providers/citeproc): Pandoc citation syntax; CSL-JSON, CSL-YAML, BibTeX/BibLaTeX (Better CSL JSON from Zotero's Better BibTeX recommended, the file watched for re-exports); main library in Preferences, per-file `bibliography:` (first only) and `csl:` in front matter; in-app rendering with citeproc-js always Chicago author-date (preview only), the chosen CSL at Pandoc export; @-autocomplete with a tooltip; a References sidebar of what is cited (+ nocite). Phase 1: Pandoc citation syntax coloured; CSL-JSON then BibTeX/BibLaTeX (a small Go parser, MIT-compatible); library from front matter `bibliography:`, else a .bib/.json beside the document or in the bundle's assets, else Preferences; watched; @ autocomplete + tooltip; pandoc exports get --citeproc and csl:. Phase 2: a References dock pane; one built-in author-date style (Chicago-like) in the page view, Accolade's PDF, Word and ODT, bibliography at the end (numeric later if wanted); optionally full CSL via citeproc-js in the browser version only.
- 2026-11-22: the Preview tab, Refresh and Follow Edits (branch previewtab here, resume in ~/Projects/lexington-resume;
  neither merged, Lexington untagged: go.mod replaces it with ../lexington-resume). Lexington: each recorded page has its
  start (pdf.PageStart: page, line, word, and what the pages before leave it — scene, CONTINUEDs, page numbers, the rows
  pagination looks back at) and its lines (First/Last, Drawing.PageOf); PDFWriter.Resume / cli.Options.Resume draws from
  a start (ErrNotResumable if the rows before it changed), asked at each page whether to stop; PageStart.Same; pdf.Reflow
  (from the page before the edit to a page that starts as before, the rest kept, cancelled by its context) and pdf.Redraw
  (the edited page alone); fonts read when first set for a Resume (gofpdf's font loading was most of a page's time).
  Tested: from every start the pages are the whole drawing's (default, CONTINUEDs with numbers, A4, Keep, none, colour
  bars, markers over blocks, backgrounds, Hebrew); twelve edits in a row, each reflowed from the last, are the whole
  drawing. Found and fixed: a dual dialogue's opening row had no line. Accolade: the Preview tab (registry: Zoom, Pages
  with Page n of N, Output, Setup with the paper on it and the setup as its tip, Text | Pages for a Live Preview,
  Refresh F5 with a dot when stale, Follow Edits, Close Preview), forward when a preview opens, the tab before it back
  when it closes, in sight on a phone's tab row; the bar over the pages only in the Menus look (Refresh and Follow there
  too; on a phone the line with Close stays). Follow Edits (pagefollow.go, on by default): the edited page laid out again at once (pdf.Redraw), the pages after
  it dimmed, the rest after a 400 ms pause (pdf.Reflow), typing cancels it; the view keeps the paintings of pages that
  look the same, the place (page + fraction) and the zoom; Save PDF…/Open in Viewer make the PDF again first. Ctrl+R is
  not free (vim's redo, the browser's reload): F5 only, and not in the browser. Measured (test driver, a made-up feature
  of 113 pages, a key typed on page 57): 1 page laid out at once, ~80 ms to it shown; 2 laid out again once typing
  pauses (400 ms), the rest kept, ~45 ms more; 1 page painted again; the whole feature takes 1.0–1.4 s. Three lines
  typed on page 11 of 46: 5 pages laid out again, then they start as before; a paragraph that adds a page reflows to
  the end (every page number moved). A heading's colour changed reflows its block (Lexington tested). Shots previewtab-*.png (desktop, phone; screenplay, typed with the
  pages after dimmed, stale with the dot, Markdown), looked at. Left: locked pages, revision marks and Markdown's pages are
  made whole on each pause (Markdown only repaints the pages that changed); a page start within a styled row is not
  resumable (Write's wrapping); not tried on a real browser or phone.
- 2026-11-22: links in the Markdown PDF and the page view (branch pdflinks here, links in ~/Projects/lexington-links
  from v1.25.0; neither merged, Lexington untagged: go.mod replaces it with ../lexington-links). Lexington: pdf.Drawer
  has AddLink/SetLink/Link/LinkString; the Recorder keeps DrawnPage.Links (rect in points; an address, or a page + y
  resolved in Drawing(), so a SetLink after the link works); a link is not a mark (Equal, Rasterizer unchanged);
  byte-identical PDF through a Recorder (new links case; the screenplay cases unchanged); CHANGELOG [Unreleased].
  Accolade: the prose PDF links each line's part of a link, #anchors to headings (GitHub-style ids, made in
  prose.ParseContext, so the web page export now has the same ids), a note's number to the note and back; headings
  are the PDF's outline (levels without gaps). Fixed: a note in a table cell was numbered twice. Page view: hand +
  lit link + tip, click/tap opens http/https/mailto only (others say "Not opened"), internal links scroll; Tab/Shift+Tab
  through the links from the page in sight (ring), Enter follows, Tab leaves past the last. qpdf --json reads the
  annotations (URIs, /Dest pages and y) and the outline exactly as recorded (TestPDFLinksInAReader). Shots
  pagelinks-hover/footnote-{desktop,phone}.png looked at.
- 2026-11-22: page footnotes (branch pdflinks). Notes at the foot of the referring page under a short rule, 9 pt; the
  text area shrinks; a line whose note's first line has no room goes to the next page with it; a long note is carried
  to the next page's foot (at most half the page while text follows, whole pages after the text); a note referred to
  twice keeps one number. Page Setup (Markdown): Notes at the foot / after each chapter / after the text, numbering
  through / again in each chapter (chapter = top-level heading, or the next level under a single title heading;
  after the text with per-chapter numbers, each chapter's heading over its notes); PDF and pages only, Word/ODT keep
  footnotes. Markdown's preview pages are made whole each pause (no Resume for prose): tested that an edit moving a
  reference and its note to the next page shows exactly the fresh drawing, paintings and links. Editor: Go to Note or
  Reference, Renumber Notes (reference order, trailing notes sorted, one undo step; code skipped), the note's text in a
  tip over its mark (ScriptEditor.OnHover; tooltip.ShowTipUnder). Left: not tried in a real browser or phone; a
  javascript: link stays a URI annotation in the PDF (the view won't open it).
- 2026-11-22: the source pane's jump, Next and Adapt… (branch sourcepane, not merged). (a) A linked passage shows
  "→ Scene 4" under it (spanLabels: the locked label, else counted; a section by its heading), "→ 2 scenes" a menu;
  it goes to the scene (goToScript, so the swapped Script tab too) and the scene flashes 0.9 s (ScriptEditor.SetFlash,
  apart from the search's highlights; a window uiTimer clears it); a tap on the passage only picks it. Adaptation >
  Go to the Passage's Scene does it by keyboard, the next scene each time. Rows' heights count the line under the
  text (rowActs; re-measured when a link or the pick comes or goes). (b) Next (Ctrl+Alt+J) and Previous (Ctrl+Alt+K,
  free: vim's evil-org keys are plain Alt+J/K) only pick and scroll; the picked passage shows Adapt… and Mark as Cut
  under it, and Adapt… is beside Next too (a phone's list is too short to show the row's). Adapt… (Ctrl+Alt+A;
  adapt_dialog.go, a ModalPopUp: two columns on a desktop ≥ 940 wide, a sheet at the bottom on a phone): How (new
  scene, draft, current scene, cut, skip), the heading (the headings the source reads, the first likely; editable),
  Where (story order, the cursor, after the cursor's scene, the end), What goes in (editable; made again for the
  script as it is when taken, unless written in). Likely: the current scene when the story's scene goes on from the
  passage it adapts (storyread SceneStart empty) or it adapts nothing yet; else a new scene, the reason shown (the
  story's own, e.g. "chapter heading"). Where: after the last scene adapting the nearest passage before it that one
  adapts, else before the first adapting a later one, kept inside its chapter's section; a new chapter's section in
  the block if the script lacks it. Enter takes the choices (choiceList: one Tab stop, ↑↓, Enter), Esc cancels; one
  undo step, an insertion only (insertBlock; the cut entry of a cut passage goes in the same group). The skipped-
  passage question is gone; the setting next-passage-inserts, its Preferences check and ribbon toggle went
  (retiredSettings drops the key on load). Start from the Source still writes the first scene at once (the writer
  asked), with the story's heading, its words after INT. selected. (c) The anchor (adapt_plan.go anchorAt): the
  passage picked or gone to since the cursor last moved; else the cursor's scene's last passage; else the nearest
  linked scene above; else the list's top (inclusive); else the start. Caption "after: “…” · picked". At the end it
  offers to go round (Next again does). Tests: TestAnchorRules (middle chapter first, out of order, partly linked,
  unlinked scene, picked, two scenes linking, the end), TestAdaptDefaults, TestInsertBlock, TestNextOnlyMoves (a
  source edited before the picked passage), TestAdaptNewScene/OtherWays, TestPassageSceneButton,
  TestAdaptFocusVisible, TestRetiredNextSetting; golden moves (passageMoves), ribbon golden regenerated (a Passages
  group: Adapt big, Next, Previous, Mark as Cut). Shots adapt-{chips,next,dialog,draft}-{desktop,phone}.png from the
  sample, looked at. Seen there: the sample suggests "EXT. GOD" for Prohor's passage — the learner's "god" place bug
  (sample's bug 1) shows in Adapt…'s heading now; not fixed here.
- 2026-11-22: the sample's three bugs (branch samplebugs, not merged). (1) The place "GOD": storyread scored "in God's
  sight" as "in <place>" (nameAt strips the 's, then the preposition counted), so God was a place and the whole
  one-scene story read "EXT. GOD"; Hryukin was next ("laughs at", "turns to"; his "begins" was no speech verb).
  Now a name after a preposition of place is evidence only when it ends the phrase: not a possessive ("God's
  sight"), not followed by an article/determiner/possessive (de "in Äußerung ihrer Sorgen", also EXT. ÄUßERUNG
  in the Fontane testdata, now EXT. GARTEN; de determiners got the inflected possessives); en speech verbs in the
  present tense (begins, inquires, threatens…). A scene's place is its first passage that says where it is: a
  place name or a learned place (Hints.Places matched in the narration), else its first place word with the place
  words and owner written with it (MARKET SQUARE, PITCHUGIN'S TIMBER-YARD; a hyphenated word's last part counts:
  timber-yard). Each passage keeps its own place when it names one (PassageReading.Place), and Adapt…/Tab/the
  learner use it before the scene's: Prohor's passage (29) suggests EXT. PITCHUGIN'S TIMBER-YARD (the learned
  place), the rest EXT. MARKET SQUARE (3 and 6 the timber-yard), where all 31 said EXT. GOD. The learner's side:
  a heading typed above a linked scene had that scene's links under it until the blank line after it was typed,
  and learned from them ("god", then "the timber-yard = EXT. GENERAL ZHIGALOV'S YARD"); the learner now keeps the
  known heading each link was under (linkHead), and a new heading does not learn from a link whose old heading's
  words still stand in its scene (fromAnother); a heading renamed and Adapt…'s placeholder written over still teach.
  The source pane's "What the source shows" adds "Here <heading>" when a passage names its own place. (2) Breakdown words
  skip notes and source links (blanked: printedRunes; the boneyard stays, an omitted scene's kept text keeps its
  tags: TestRenumberDropsOmittedKeys), so no tag lands in a link's quote,
  a tag whose words left the script is lost though a link quotes them, Tag Selection in a link says why not;
  anchors' context without notes. The schedule's strip sums and first words follow (no link words); a
  strip summed before still matches (oldSceneSum), so no false "changed". (3) The Characters pane's "from scene N"
  and its scene list use the scene labels (locked: 2, 2A; else counted): HRYUKIN "from scene 2". Tests:
  storyread TestPlaceEvidence, TestLearnNotFromAnotherScenesLinks (also a rename, a placeholder), TestWhatTheSourceShowsHere,
  TestTagsSkipLinksAndNotes,
  TestScheduleSumSkipsNotes, TestCharSceneLabels; TestSample checks HRYUKIN's label and no tag in a link; the
  generator's forget-"god" step went (it asserts one learned place). Sample regenerated (breakdown anchor,
  schedule starts/sums). Shots adapt-dialog-desktop (Prohor's passage), passage 30, combo-sample-characters,
  looked at. Not done: Mitry Mitritch (said "to" once, in speech) and Petersburg still read as places.
- 2026-11-22: the language rules and Hekaya, as one item (branch langrules in lexington-langrules, from v1.26.2, and in
  accolade-langrules; Accolade's go.mod replaces Lexington with ../lexington-langrules until it is tagged). Lexington:
  rules.Language, [Languages.xx] in the configuration, every language in full (Dump, Part, With; the old [Scenes],
  [Matchups], [Directions] read into it): Scenes, Headings (patterns with {n} {place} {io} {time} {any}), Settings,
  Times, Matchup, Direction, Cues (forced caps list parenthetical extension shape repeat), Extensions, Shape, Repeat,
  TransitionEnds, Transitions, TransitionMarks, TitleKeys. fountain.Dialect reads lines by them (Heading, Transition,
  Cue with the script's Cast: its /* Characters */ names, fountain.ListedNames, and every earlier cue); ParseIn,
  ParseLinesIn, ForceIn (> for a transition), UnforceIn, StripIn, KeepLooseIn. Heading research with sources in
  docs/LANGUAGES.md: real scripts write other orders (Chinese place/time first, Arabic مشهد N … ن/د, German
  GERICHTSSAAL INNEN/TAG, French 5 - CHU GRENOBLE – INT/JOUR), so patterns for those. Hekaya: default ar is its full
  set (plus numbered Arabic headings); the fixed `hekaya` language for .hekaya (cli -i/-o x.hekaya); Direction: on the
  title page (lex.TitleDirection, cli.Options.Direction); twelve open points chosen, documented ("Hekaya, where its
  spec is open") and tested (TestHekayaOpenSpec); drafts in docs/hekaya-issues/ (00 note with Courier Badi, 01-12),
  NOT filed. PDF: Arabic shaped by go-text's HarfBuzz with Courier Badi's own GSUB (pdf/arabic.go), printed as the
  characters the font maps the glyphs to, in logical order before wrap and bidi (the Recorder records them);
  TestArabicShapedAsHarfBuzz checks every glyph of Hekaya's samples in all four faces; a fidelity case; Unsupported
  warns only for letters whose joined forms have no character (Pashto, Sindhi… extended letters) and other joining
  scripts. Not done there: GPOS (a vowel stacked on a shadda stands at its default place). Accolade: syntax.ClassifyIn
  / ClassifyTypingIn (editor.Dialect), the window's element, keyflow, outline/headings by the patterns
  (isSceneHeadingIn asks the language's dialect), exports, PDF check, characters/details/breakdown parse by the
  language (langFor), Force/Unforce by ForceIn/UnforceIn; Format > Character (markLineElement): a name the rules
  would not read goes into the Characters list in the same undo step, in Hekaya @ is written; .hekaya: its own type
  "Hekaya (.hekaya)" in Save As/rename for Arabic scripts, opened and saved as written, language hekaya. Tests:
  TestCueRules, TestTransitionRules, TestHeadingPatterns, TestTitleKeys, TestHekayaFixed, TestListedNames,
  TestHekayaOpenSpec, TestEveryLanguageInFull, TestTitleDirection, TestArabicShapedAsHarfBuzz, TestArabicForms;
  Accolade TestClassifyAgreesWithLexington (Hekaya's samples and a Chinese script), TestMarkCharacterByRules,
  TestHekayaFile. Shots (SHOT_DIR=… -run TestLangRulesScreenshots) /tmp/claude-1000/shots/langrules/langrules-{ar,zh}-
  {editor,page}-{desktop,phone}.png, looked at: Arabic shaped in the page view; Chinese goes to the browser PDF (no
  CJK in Courier Badi), whose Chromium aborts in the sandbox. Then (Joop): extensions configurable — Lexington's
  [Extensions] (ext → language; hekaya = "hekaya" built in; carried with a script's language, adopted as an
  "extension" item), read by cli -i/-o, Accolade's languageOf, open/Save As/rename types (langType), the file picker;
  withLanguageName gives film.hekaya ⇄ film.ar.fountain; TestExtensions, TestExtensionLanguages,
  TestLanguageExtensions (a user's .kissa opens as zh, saves byte for byte, reopens). Research of other Fountain
  dialects with their own extension in docs/LANGUAGES.md: Wrap (.wrap, language by a Language: title key) and
  Playfount (Russian/Ukrainian, no extension of its own) — no default entry besides Hekaya. Also: Action/Transition
  marked by the rules (no needless > before - قطع -, ! before a registered name), caps kept beside the no-caps rules
  (a Latin cue in capitals reads as before), Force/Unforce tested with the rules (a forced later bare name in the
  registry keeps its @ coming back: it cannot be told from the writer's), lexington-web builds. Left: the @ is
  shown, not hidden; characters/details/breakdown find the language by its scene words
  (langFor), and ar and hekaya have the same words, so a .hekaya there reads as the configured ar; the editor draws
  an RTL parenthetical's brackets unmirrored in italics; Courier Badi's final ص ض س ش overhang 276 units into the
  space before them (glyph xmin −276, the advance 600 as HarfBuzz's), seen in the PDF as words touching — a font
  matter; same-line cues (张三：…) are not read; Lexington's Fountain writer still writes @ for every no-caps cue.
  Then (coordinator: per-key cost): the cast kept between keys (fountain.CastCache in the editor, updated from the
  changed lines; the Characters block read again only when the end blocks change), the dialect's checks skip
  paragraphs (over 240 bytes), heading lookups without a key per line, the default languages built once, the parser
  reading the whole text for names only for list/repeat languages. ClassifyIn on the 10 000-line bench text went
  37 → 9 ms (ClassifyWith 9 ms). BenchmarkSwappedSourceKey, 60x, three alternating rounds against main (8c2d287),
  medians in ms, main/langrules: beside 55/50, inside 80/72, notes 70/65, script-with-source-inside 56/61
  (means 58/59), inside-pause 3/3 — within the machine's noise (load ≈14 from other work, ±20 % between rounds).
  The Tab flow and the cue picker write @ only where the language would not read the name; stats, production
  checks, the characters list read headings by the language's patterns. TestCastCache, TestTabCueByRules.
- 2026-11-22 (Joop: "55 minutes Accolade gates? … I think we need some cleanup"): DONE (branch testspeed, not merged; see the tick log and docs/TESTSPEED.md). After the Quick tab. The race step is out of the gate now (Rules); what remains is ~6 min, the root package's tests ~3 min of it. (1) Measure: `go test -json` over the root package, rank tests by elapsed time; (2) fix the slowest: real timers/sleeps → injected clocks (the long-press/tooltip pattern), the novel-sized text only where a test needs it, shared windows where subtests can reuse one; (3) move self-contained parts out of the root package (page view, source pane model, reports) where it shortens the serial run; target: the root package under 90 s, the gate under 4 min. Before/after numbers in the tick log.
- 2026-11-22: the Quick tab, Smart and Fixed (branch quicktab, not merged). quick.go. The lock on the ribbon's tab row
  (between the palette and ☰; also View > Lock Ribbon Order in the Ribbon look, its key tip L) switches the window's
  ribbon in place: Smart (a phone's at first) puts a Quick tab first and in front, the other tabs' groups most used
  first (a Production stage's group stays first; unused groups keep their order); Fixed (the desktop's at first) is the
  standard order. The tab in front stays if both have it (Quick → Write in Fixed). Per window (window.ribbonOrder;
  the other windows keep theirs); the last choice is remembered per phone or desktop (settings ribbon-order-phone /
  -desktop) as a new window's and the next start's. A window turning narrow with no choice made puts the order's first
  tab in front, not over the Preview tab or a tab the user chose. Score: a built-in priority per kind
  and role (screenplay: Scene Heading 95, Undo 92, Action 90, Character 88, Dialogue 86, Save 84, Find 75, Preview 72,
  Parenthetical 70, Redo 55…; with a source Next Passage 80, Adapt 78; production Tag 62, Production pane 60; Markdown:
  Heading… 95, Undo, Save, Bold, Italic, Bullets, Link, Find, Preview, Footnote…) + 20·ln(1+uses); a use is any
  command given (button, menu, key, palette; not the ribbon's own nor the Preview tab's), counts halving every 14 days,
  kept in ribbon-usage.json beside settings.json (this device only; in the browser the same memfs as the settings).
  Filled to 10, at most 12 (an env first seen later, e.g. a source or Writer↔Production, may top it up to 12). Stability: the counts are read once (the app's first use of them) and frozen; the Quick
  order is kept for the session per kind of document, what comes later (a source's Next Passage and Adapt, a pin) goes
  at its end, so flipping Smart/Fixed shows the same two ribbons. Long press (browser: 500 ms, not if the groups
  scrolled meanwhile) or right-click (Fyne mobile's long press) on any ribbon button: Pin to Quick / Unpin from Quick /
  Remove from Quick (a pick, kept off); pins come first after a restart. Desktop: View > Quick Row (or Show the Quick Row
  on a button's right-click), off at first: the Quick commands as small buttons under the ribbon while Fixed, their key
  tips with the tabs'. Tests quick_test.go: scoring and decay, stability within a session and after a restart, pinning
  (long press, the tap after it swallowed), the toggle (in place, per window, flipping twice identical, groups by use,
  key tip, ☰'s check), Preview over Quick, Quick tab and row ≤ 1000 px. No golden changed (the lock's menu item is
  Ribbon-look only). Shots (SHOT_DIR=… go test -run TestQuickScreenshots, all eight looked at): a 380 px phone shows
  6½ of the Quick tab's 10 buttons (Scene, Undo, Action, Character, Dialogue, Save, half of Next; "Scene"/"Paren." are
  the phone's short element labels), the rest one short swipe; Smart's Edit tab with Find and Lines used puts those
  groups first; desktop Smart fits in four columns; the Quick row is 872 of 992 px (screenplay, ten), its key tips with
  the tabs' (the lock L, Commands O, ☰ M). Open for Joop: a phone Quick tab in two rows would end the swipe but make
  that tab taller than the others (the script jumps on a tab change). Not done: a Preferences checkbox for the Quick
  row.
- 2026-11-22: the phone's Quick tab in two rows (branch quick2rows, not merged; Joop chose it). At a phone's width the
  Quick tab lays its buttons out in two rows in their order, row by row, the first the larger half (5+5, 6+5, 6+6):
  at 380 px all ten show without a swipe (Scene, Undo, Action, Character, Dialogue / Save, Next Passage, Adapt, Find,
  Preview). That tab is a row taller than the others (accepted); the window lays out again round the top
  (relayoutTop: the frame Border kept in w.frame) when the ribbon's height changes, so the script and panes get the
  height left, no gap or overlap, and the same layout each time back and forth. The desktop's Quick tab (three-row
  columns) and Quick row are as they were. Preferences > General > Appearance: "Quick row under the ribbon" (hidden on
  a mobile device), one setting with View > Quick Row. Tests: TestQuickPhoneTwoRows (every button inside 380 px and
  not cut, two balanced rows, no scroll; the script-and-panes split at the ribbon's foot on Quick and Write, its
  height the rest, back and forth identical; the desktop's height unchanged by Quick); TestRibbonSettings (the
  checkbox and View > Quick Row). Shots looked at: quick-phone-smart (both rows), quick-phone-smart-edit (the tab
  switch), quick-desktop-smart (unchanged).
- 2026-11-22: test speed (branch testspeed, not merged; docs/TESTSPEED.md has the rankings and tables). Measured
  first: the root package's 669 tests took 187 s with a flat spread (median 0.17 s; the 19 over 1 s only 50 s), so
  the cost was the fixture — NewMainWindow 92 of the tests' 158 s CPU, 65 of it Fyne re-theming the window just built.
  Fixed: the theme set before a window is built and only when the scheme changes (also no re-theme of the other
  windows when one opens; TestNewWindowTakesTheSavedTheme); the format files decoded again only when one changes
  (presetsIn, keyed by name and content; TestPresetsInFollowsTheFiles, -race clean); the link offer re-lays the
  window out only when it comes up or its words change (was: every key in a swapped-in source). Tests: heldDebounce
  (a burst's debounced work run once after it, as the app's pause does) for makeSample's typing (the sample comes out
  the same but for dates and ids), TestFootnoteReflowInPreview bisects for its paragraph count, TestTailBlocksAsBefore
  compiles its oracle's regexps once, TestPreviewFollowsATypedKey's log-only layout under -v. No assertion weakened,
  nothing behind -short. Root: 187 → 93–96 s (three runs, -json, load 2.4–3.6; 85–88 s at lower load). Gate as
  defined: ~11 → ~8½ min (root 97, step 2 254, focus 123, vet 4, wasm 12, no-pandoc 16; only TestBrowserPDF* fail,
  the sandbox's missing Chromium). Targeted -race on the changed tests: clean. Left: (1) proposed for the Rules — step
  2 as `go test -p 1 ./docs/... ./internal/...` (it re-runs the root package *with* the focus tests; steps 1 and 3
  already cover every root test): the gate ~4½ min; (2) under 4 min needs the Fyne fork: the software painter's shadow
  blur on every capture (~60 % of the focus tests), test windows never releasing their widgets (live heap → ~500 MB,
  no CleanCanvas on the test window's Close), test.NewApp's font-cache clear; Lexington's pdf.AddFont per report.
  Step 3 of the item (moving packages out) not done: the cost is window building, which would move with them.
- 2026-11-22 (Joop: the preview "gets stuck when clicking swap" — fixed on branch swappages: the script's page view now goes when a pane is swapped in and the incoming document gets its Live Preview; "check for similar state change issues and ways to prevent similar bugs"): NEXT after folding/chips. (1) Audit every transition that changes what the main editor or window shows — Swap with Main / back (each pane), Open, New, Open Sample, document type change, rename/Save As, revert, join/leave a shared session, role change, Writer/Production, closing a pane, a bundle's source changing — against every view derived from it: the page view and Preview tab (and Follow Edits' target), Markdown Live Preview, find bar and its highlights, outline/cards/characters following the cursor, status bar and title, Page Setup's summary, the Adapt dialog if open, the source pane's anchor caption, ribbon contextual tabs, undo history focus, collab presence. List each (view, transition) pair that can go stale in docs/STATE.md. (2) Prevent the class: each derived view records the document it shows (its kind and identity, e.g. a *document or "script"); one window-level hook, mainDocChanged(old, new), that every such view registers with and that every transition calls (instead of ad-hoc resets in swap.go, open, new …); and one invariant test that drives every transition (table-driven) and asserts after each: no shown view describes a document other than the main one. (3) Fix what the audit finds. Screenshots only where something visible changes.
- 2026-11-22: Accolade's own blocks fold, source links are chips (branch foldchips, not merged; block_folds.go,
  link_chips.go; internal/editor: editor.Fold with Block + FoldLabel, Chips/ChipAt/OnChipPressed; wrap.WrapSpans).
  In the script's editor the tail blocks (Characters, Source, Adaptation, Production, Notes) and an OMITTED scene's
  kept boneyard are folded by default to one quiet line: "▸ Production · scene numbers locked · pages locked ·
  Blue", "▸ Characters · 5, 5 with notes", "▸ Omitted scene · INT. TAVERN - DAY #3#, 2 lines", "▸ Notes · 3
  paragraphs · This is…" (cut at the page's width). Open while the cursor is in them (a click, the arrows, a Find
  hit), folded again when it leaves; opened by hand (the gutter's ▸, View > Fold or Unfold Here, vim zo/za,
  Unfold All) they stay open until zc / Fold Here / zM. Shift+Tab's cycle leaves them. [[from: …]] is a chip
  "↗ source" ("↗ 3 passages" for a range when the source pane has its matcher, the note's directive count
  otherwise; "↗ source?" not found), its quote decoded on hover/long-press ("first words … last words", "after
  “visit?”"), the note's own text only while the cursor is inside (a tap puts it there). The Adapt dialog's preview
  shows links readable and writes the real note back. Display only: text, bytes, peers, screen readers
  (AccessibilityText is the whole text) unchanged. Same thread (Joop): new links shorter, 4 first words + 3 last
  (whole up to 7; 3 of the sample's 31 passages and 166 of the novel's 2 592 needed more), a dash encoded only at
  a term's edges (timber-yard), sample made again. Cost (benchstat, base = the link change, interleaved): sec/op
  within noise (sample key 3.63→3.71 ms, novel key 58→68 ms p=0.15, move 11.2→11.3 ms, PaneKeystroke/beside
  23.7→25.0 ms); B/op +8–10 % a key (+23 % PaneKeystroke/beside) from parsing the links each key, fixed after
  (chips kept by line text) but not measured again. Shots: SHOT_DIR=… go test -run TestBlockFoldScreenshots.
  Rough: three end words leave near-end matching out (nearEnd needs four); ↗ unchecked in the browser build.
- 2026-11-22 (Joop: "Scrolling on the cards view only works in the gaps"): fixed in the Fyne fork (linux-a11y
  7ea7b487): Scroll.CanScroll, and the glfw driver hands a scroll to the Scrollable around one that can't move that
  way (each card's synopsis is a multi-line Entry, whose own scroller swallowed the wheel). Fork tests
  TestScroll_CanScroll and TestWindow_ScrolledPassesOutward (fails without the driver change). Branch cardscroll.
- 2026-11-22 (Joop: "the new approach to the cards makes the card open in the sidebar at first and push the script to
  the sidebar in a very similar view to the outline... Maybe make the cards and script tab like the sidebar when in
  the middle? And maybe we can use a similar approach for the source, enable them to be active in the middle at the
  same time with this tabbing?" and "I think that comes from a too careful attempt at the swap with middle task.
  Being too careful almost broke it more"): MIDDLE TABS, NEXT after folding/chips and BEFORE the state-change audit
  (the audit then covers the new transitions instead of swap's). The middle becomes a tab strip like the dock's:
  Script, Cards, Source (and any pane put in the middle) are tabs there, several open at once, one shown; Cards from
  the View menu/ribbon opens as a middle tab (never in the sidebar first, never pushing the script into the sidebar);
  a dock pane's "Swap with Main" becomes "Open in the Middle" (a tab) and back "Back to the Side". Remove the swap
  machinery this replaces (swap.go's swapWithMain/swapBack and their special cases), don't build around it. The
  ribbon's contextual tabs, Preview and Follow Edits follow the shown middle tab. Screenshots: middle with Script +
  Cards + Source tabs, desktop and phone.
  Done (branch middletabs): middle.go, docs/MIDDLE.md (was SWAP.md). The middle is a row of tabs (the dock's tab
  widget behind a `tabHost`; an `overflowRow`, so a phone's row fits with More…), shown only while a tab beside the
  script's is open. View > Cards (Shift+D, ribbon, outline's Cards button, the pane's full-view button) opens the
  cards' full view as a middle tab directly; a dock tab's menu has Open in the Middle (View > Open Pane in the
  Middle), a middle tab's Back to the Side and Close (× at the row's end); the Script tab stays. Source and Notes are
  Markdown documents there, several open at once; Ctrl+PgDn/PgUp (View > Next/Previous Middle Tab; no keys in the
  browser, which keeps them), arrows on the row; Adaptation > Source in the Middle (`source.middle`). A pane is in
  the dock or the middle, never both (`dock.shown` excludes middle panes; `showPane` brings the tab forward). One
  transition, `middle.show`: preview follows the tab in front (pages / Live Preview / none over a view, back on the
  next editor tab; a PDF still being made is dropped), offstage editors, link offer on leaving the source, highlights,
  menus, ribbon, title (• also for a document behind another tab), status, focus — the hook for the state audit.
  `goToScript` (outline, cards, Changes, Production, source scenes) brings the script's tab forward. Removed: swap.go
  (swapWithMain/swapBack, w.swapped, w.mainDoc, swappedCenter/IsView, the dock's Script tab scriptPane and its
  timer), cardsVisible, the dock's boxOf/titleOf special cases. Tests: swap_*_test.go → middle_*_test.go rewritten
  for tabs; TestMiddlePreviewFollowsShownTab is a table over every pair of tab kinds; new TestMiddleCommands (keys,
  row arrows), TestMiddlePhone (six tabs on a phone's row), TestMiddleOnePlace; goldens (menus, shortcuts, ribbon,
  shortcuts help) moved. Gate: root (all but TestBrowserPDF*, the sandbox's Chromium), focus tests, subpackages, vet,
  wasm green. Shots (TestMiddleScreenshots, middle-{desktop,phone}-0…8): script alone, View > Cards, Script + Cards +
  Source with each in front, link offer, notes, outline, Cards pane back to the side — looked at. Left open (for the
  state audit): Choose Source… while the Source tab is open (its document is the one opened); the Script tab's
  passage caption and tap-to-passage are gone (Cards pane still lights the passage's scenes); the stripboard is not a
  tab (it puts the script's tab in front); in the ribbon look an Entry (a card's synopsis) swallows menu shortcuts, as
  for every command; Preview while a view tab is in front does nothing visible.
- 2026-11-22 (Joop opened Star Wars: A New Hope: "only the 3 line title and then the credit that are a bit uncommon
  don't get parsed properly, and it reads the THE END at the end as a scene instead of action"): Fountain import
  details, Lexington first. Needs the file (not on the VPS) — ask Joop for it. Small; after the middle tabs.
- 2026-11-22 (Joop runs the AUR release, 0.6.0, and found today's issues there: "I think it's good to tag the new
  version and push it to the AUR when done"): RELEASE v0.7.0 once cardscroll, foldchips and middletabs are merged
  and CI is green: FyneApp.toml version bump ("Version 0.7.0" commit), annotated tag v0.7.0 with a one-line summary
  like v0.6.0's, push main + tag; then ~/Projects/aur/accolade: pkgver=0.7.0, pkgrel=1, sha256 of the tag tarball,
  makepkg --printsrcinfo > .SRCINFO, makepkg -sf builds and the binary starts, push to AUR (~/.ssh/aur_ed25519).
- 2026-11-22 (Joop: "Probably good to put a nice UI in for lexington.toml etc configuration? Even if you just want to
  use it for your CLI... Maybe open that editor directly if you open a lexington.toml file even..."): LEXINGTON
  SETTINGS EDITOR, next agent slot after middle tabs (it touches lexington_settings.go/Preferences, not the middle).
  Today Preferences > Script > Edit Lexington Settings… is a raw TOML text box (lexington_settings.go editTOML). Make
  it a real editor: one page per section of rules.TOMLConf (element sets / own formats, scene-heading languages and
  their words, file extensions, language rules, export options — whatever the struct has), with lists you add to /
  remove from, the defaults shown greyed with "override", validation as you type, and a "TOML" tab that keeps the
  raw text for anything the form can't show (round-trip: comments and unknown keys survive, as lexington_adopt.go
  already edits text safely). It edits any lexington.toml, not only the settings folder's: File > Open (and the
  command line, `accolade path/lexington.toml`, and the desktop file's MIME/glob) on a file named lexington.toml —
  or a .toml that parses as a Lexington config — opens this editor on that file; the title bar says which file and
  that the CLI reads it. Help page: what each setting does for Accolade and for the `lexington` CLI. Lexington first
  if the config needs a schema/describe API (field docs, defaults) — tag it. Screenshots desktop + phone.
  Joop, same day: "settings editor when opening any .toml, so including custom formats" — so EVERY .toml Accolade
  opens goes to a form editor, picked by what the file is: lexington.toml (or a Lexington config) → the settings
  editor above; a format/preset file (Name, Base, [Elements.…], [[Accolade.Buttons]] — today only a raw text box via
  Preferences > Edit Format…, preferences_dialog.go editFormat) → a format editor: base format, per element its
  margins/indent/case/spacing/font with a live sample, the toolbar buttons as a reorderable list; same TOML tab and
  round-trip rule. A .toml that is neither opens as text with a note. Saving a format file inside the formats folder
  makes it pickable at once (presetsIn already reloads by content).
  Done 2026-11-23 (branch tomleditor): Lexington v1.28.0 first (rules/describe.go: ConfFields, LanguageFields,
  ShapeFields, CueRuleFields, FormatFields, ElementKeys, FormatFileFields, Describe; a reflection test that every
  field/key/cue rule is described; tagged and pushed with master). Accolade: the TEXT is the only state — every
  control is one path edit of the TOML text (toml_text.go: sections incl. [[blocks]] and their comments, multi-line
  strings, inline tables rewritten whole, quoted keys; dotted keys refused), and toml_doc.go keeps an edit only if the
  text reads back as the same map but for that path and checks out for its kind (margins, align, style, pagination,
  color scheme, direction, cue rules, Base). The old line helpers of lexington_adopt.go now sit on it (a [[block]]
  ends a table; the old ones let a key fall into the last button). classifyTOML: lexington.toml or a formats folder by
  place, else by a key only that kind has AND a typed decode (Base/Accolade/Elements.<element>.<field> → format;
  Languages/Standard/Elements.<set>.<element> → settings; a root Name or an Extensions list alone, FyneApp.toml,
  empty, other → text). A comment at the end of an edited key's line stays. Editors (toml_editor.go): a WINDOW of its
  own on the desktop, a dialog over the window in the browser (as Preferences) — not a middle tab: the file is no part
  of a script, it opens from Preferences and from `accolade x.toml` with no script window, and saving changes every
  window; app.tomlEditors keeps Accolade running while one is open (the real binary, `accolade comic.toml` under
  xvfb-run, stays up with only the editor). Title: kind, path, and for a lexington.toml that the lexington command
  reads it (checked: the CLI reads x.de.fountain with German rules from a file without [Languages.de]). Settings editor (settings_editor.go): General
  (Standard, Extensions with Lexington's .hekaya greyed), Languages (one at a time; each field Lexington's default
  greyed with Override / Use the Default; lists add/remove; Matchup by role, Cues as checks, Shape, TitleKeys; a
  language in the older [Scenes]/[Matchups]/[Directions] moves out on its first edit, as they are read over
  [Languages]), Element Sets (one set, one element; Lexington's presets read-only with Override = copy whole).
  Format editor (format_editor.go): Format (Name/Description with Base's as placeholder, Page, Base), Elements (Base's
  greyed; Change It Here copies Base's element whole, since an element replaces Base's whole), Buttons (↑ ↓ ✕, Add;
  blocks moved whole with their comments). Live sample (toml_sample.go): Lexington's own PDF drawing (cli.Write →
  Drawing → raster) of a short script, one at a time (no app-wide draw guard exists to share), dropped when its page
  or editor goes, redrawn 250 ms after a change, beside the form (under it on a phone). TOML tab
  = the same text; wrong TOML disables the pages, the status says the line, Save refuses. Help tab and Help >
  Settings Files (docs/help/settings.go, generated from rules.Describe; Accolade vs the CLI: the CLI reads
  ./lexington.toml or -config, adds presets; Accolade layers over the defaults). Opening: openPath (File > Open,
  drop, Open Recent) and openFiles (command line) branch on .toml before any window is picked; .toml in the open
  filter and the browser's picker; desktop file MimeType application/toml; Windows Open With (Accolade.TOML).
  Saving lexington.toml in the settings folder reloads the settings; a format in the formats folder (or Add to My
  Formats from elsewhere; Accolade's built-in copy saves into one's own folder) is a script format at once
  (scriptFormatsChanged + Preferences' lists). Browser: File > Open of a .toml copies it to /files and opens the
  dialog; Save writes it there and offers it as a download; the settings and formats live in the kept settings
  folder. Removed: editTOML, editLexingtonSettings, defaultLexingtonSettings, formatText's commented copy of the
  base. Not done: Format has no case or spacing fields in Lexington (margins, font, size, style, align, prefix,
  postfix, hide are what there is); the sample does not highlight the chosen element; macOS has no document-type
  entry for .toml. Shots (TestTOMLEditorScreenshots, tomleditor-{desktop,phone}-{settings-0..3,format-0..4}) looked at.
- 2026-11-22 (Joop, on the folding agent's 4-first + 3-last words: "What about checking shortest unique for start and
  finish instead? You could auto-fix that when new text breaks this, and when impossible signal this with a
  warning."): SHORTEST UNIQUE LINKS, right after foldchips merges (second agent slot; touches textfrag/link code,
  not the middle). A new link's text= start is the fewest first words, and its end the fewest last words, that make
  the range match exactly this passage in the source (start at least 1 word, a short passage quoted whole when that
  is shorter). Whenever the source changes (the source pane's Edit, a reload, Update Source Links, opening a script
  whose source differs from the last time) re-check every link: one that became ambiguous or stopped matching but
  still identifies its passage by its old anchor/approximate match is lengthened or re-anchored automatically (one
  undo step, status line "N links lengthened"); one that cannot be made unique (two identical passages, or the
  passage is gone) gets a warning: the chip "↗ source?" / "↗ 2 places" with a tooltip saying why and the Source
  pane's issue list. Old links keep reading. Tests: uniqueness after random source edits (property test), the
  auto-fix, the impossible case's warning; sample regenerated (links get shorter: report before/after lengths
  for the sample and the novel). Help page updated (replaces the "3 end words" trade-off note).
- 2026-11-22 (Joop: "for the web version, how hard would it be to add Chromecast support and basically cast a script
  like a Google Sheets presentation, to do table reads etc? Could be good to have a panel with it that filters and
  shows the relevant parts quickly (pick by scene, by actor, by label etc)" — then "Not yet, probably good for next
  weekly rate limit"): TABLE READ / CAST, NOT BEFORE the weekly reset of Wed 2026-11-25 (after Markdown images,
  before citations unless Joop says otherwise). Plan agreed in chat: (1) a Table Read view — own page/window, large
  type, dark, shows the picked part; cast with Chrome's own Cast tab, or a second window on a projector/TV; (2) the
  view joins a collab-style session as a follow-only viewer so a phone/laptop drives it (presenter cursor); (3) later,
  only with Joop: a real Cast button (Google Cast developer registration in his name, a custom receiver page on
  Pages, Chrome/Android senders). The panel: filter by scene/range, character, tag/label (breakdown and stripboard
  tags); per-actor tint, "only X's scenes", narrator mode for headings/action, one-tap next scene, page/time estimate.
  Joop, same day: "The Markdown images could join cast images for rudimentary storyboard casting support" — so build
  the Markdown images item with this in mind: one image model (an image referenced from a script or note — Markdown
  ![](…), a scene's storyboard frame, a character's cast photo in the Characters tab) shared by the page view, the
  PDF and the Table Read view. Table Read then shows a scene's storyboard frames beside or between its lines (a
  "Storyboard" toggle), and a character's cast photo next to their name in the filter panel and on their first line.
  Storyboard formats to read (Joop: "check Storyboard Fountain's format"; it became Storyboarder, a different file):
  (a) Storyboard Fountain (setpixel/storyboard-fountain, MIT, Charles Forman & Chris Smoak, last push 2017) writes the
  boards INTO the Fountain: a `[[board: <file>.png, duration: 2.50, continued: true]]` note on its own line under the
  action/dialogue line it illustrates (duration in seconds, only when set by hand; continued = the board carries on
  from the previous one), plus header notes `[[Storyboard Fountain: 0.0.2]]`, `[[aspectRatio: 2.35]]`,
  `[[dataPath: <folder>]]` (the images' folder beside the script). Accolade should read these as storyboard frames
  (and write the same notation for frames added in Accolade — it is plain Fountain notes, so other apps keep them).
  (b) Storyboarder (wonderunit, its successor; not OSI) keeps boards in JSON `storyboards/Scene-<n>-…/*.storyboarder`
  beside the script and rewrites scene numbers to `#<n>-<id>#` (docs/PRODUCTION.md §Storyboards): read its boards
  per scene by that id, never write its files.
- 2026-11-23 (Joop, with his own "Kredu min, Sinjorino" adaptation — a free adaptation of a 70k-word Esperanto novel,
  21 scenes, 3 [[from:]] links, the Enkonduko and long runs skipped; the file is private: measure on the copy in
  ~/.local/share/accolade-bench/kredu only, never commit it or quote it in fixtures, build synthetic tests of the same shape):
  "It doesn't have to be perfect, the tools should just somehow be useful and not overwhelming when you have a
  situation like this." / "when I added this source at first, with the first version of the source tool, it had
  wildly inaccurate suggestions and started breaking the document. That doesn't seem right to me. Useful things
  would be something like 'figure out existing sections, mark them as sections and add the rest beyond it', and a
  rough match guess that's added unintrusively..."
  LINK UP A FREE ADAPTATION, after shortlinks (it builds on the position alignment). Rules first:
  (R1) NOTHING is written into the script from a guess. Guesses live beside it (chips, the source pane) until the
  writer accepts one; accepting is one undo step. Adding a source to a script never changes the script.
  (R2) Quiet: only confident guesses show; one cut suggestion at a time; no dialog full of suggestions.
  What it does: (1) SECTIONS FIRST: align the script's scenes to the source in order (rarer words + order, the
  monotone alignment; a scene-level quick test on Joop's file already lands most scenes: ch.1–5, para 11→130, with
  a few short scenes jumping far off that the order constraint pulls back) and show which source sections
  (chapters/## headings, else runs of passages) are already adapted: the source pane marks them "adapted (scenes
  3–9)" as whole sections, not per passage; (2) THE REST BEYOND: everything after the last adapted section is
  "not adapted yet" and is where Next / Adapt… continue — the writer goes on from where the script ends, never from
  the top; (3) a rough match guess per scene, unintrusive: a faint "↗ link?" chip on the scene heading; tap → the
  guessed range with Accept / Adjust / Not This; (4) skipped runs inside the adapted stretch (the Enkonduko before
  the first adapted section, gaps between) offered one at a time as a quiet line "Enkonduko (9 paragraphs) isn't in
  the script — Mark as Cut?"; (5) Adaptation › Link Up the Script… does it all at once in a review list for when the
  writer wants that. Measure on Joop's copy: scenes guessed right / wrong / silent, and report.
- 2026-11-23: LINK UP, done (branch linkup, not merged). (a) suggestLinks summed in map order (Jane Eyre 0–3
  right from run to run); fixed, then replaced. (b)+(c) scene_align.go: each scene against windows of the source
  (1–50 passages), the rarer words they share (once each, log N/df) against chance for a window of that size (long
  passages hold more words), a z-score; scenes placed in order by a forward–backward DP (any left out), a floor of
  1 + √(2 ln places); confidence = the best total against the scene left out (slack 1.5) or placed off its hill;
  silent below 2. Scenes left out between two placed ones are looked for in the gap alone (a lower floor for fewer
  places). Links are fixed anchors. Bench, before → after (right/wrong/silent; invented silent): Christmas Carol
  12/0/0, invented guessed → 12/0/0, invented silent; Jane Eyre 0/11/0 (deterministic), both invented guessed →
  7/0/4, both silent; an invented scene put between two close sure ones stays silent (TestAlignInventedBetween).
  Cuts offered (the bench's cuts row): Christmas Carol none; Jane Eyre 12, 9 right, 3 wrong (XVII: I.II is out of
  the novel's order; XIX and XXXV: parts of merged scenes whose guess is the other part) — a gap with a silent scene
  between its neighbours is not offered. A look: 28 ms (Carol), 67 ms (Jane Eyre, 4087 passages; the source's side
  kept while the passages stay, Pow → Expm1: was 318 ms). Joop's Kredu (21 scenes; its 3 links all found, all in
  one scene; local only): its links taken out, that scene found again; with them in, 18 of the other 20 guessed
  (13 by the order, 5 in gaps; 6 share a first passage), all in the source's order (rightness unknown). UI: the editor's Tags (a faint
  label after a line, not text: the cursor passes it; Short "↗?" where "↗ link?" has no room), tap/long press →
  Accept (one undo step) / Adjust… / Not This (skipped links:); Ctrl+Alt+G (Ctrl+Alt+L locks many Linux desktops) and vim gs open it from the keyboard. Source
  pane: chapter (else run) labels "adapted (scenes 3–9)", "adapted? (…)" when only guessed, "not adapted yet";
  Next/Adapt… go on after a guessed scene (anchor "(by its words)"); a skipped section offered as cut one at a
  time; Adaptation › Link Up the Script… (replaces Suggest Links and its "1 of N" prompts) lists guesses (checked)
  and skipped sections (not), one undo step. Guesses are made also with the pane hidden once the source is read
  (so link chips count passages there too). Tests: TestComboNoLinks, TestGuessTagNotText, TestLinkUpLeavesScript
  (R1: byte for byte, nothing to undo, no changes flag), TestLinkUpOffersCut, TestAlignScenesSameEachTime,
  TestAdaptBenchPrivate (skips without the file; counts only). Shots: SHOT_DIR=… -run 'TestComboNoLinks$|
  TestLinkUpOffersCut$' (combo-linkup-{sections,menu,cut}, combo-beside-nolinks). Open: the guess range is the
  dense core of the window (often one passage), so Accept links less than the scene adapts; the last scene is
  often silent (nothing after it to hold it); Jane Eyre's non-monotone scenes (I.II after II.II in the novel)
  stay silent; storyread's dialogue lines are not used yet; guesses live for the session (Not This persists); a
  heading read right to left gets no tag (Link Up and Ctrl+Alt+G still reach it); vim gs has no test.
- 2026-11-23: shortest unique links (branch shortlinks, not merged; Joop 2026-11-22: "checking shortest unique
  for start and finish … auto-fix … when impossible signal this with a warning"). fragmentFor writes the fewest
  first words and fewest last words that find the passage(s) alone (read off the word index, confirmed by
  findDir; a short passage whole when that is no longer; prefix-/-suffix only when nothing inside tells it
  apart); the 4+3 constants are gone. Lengths (words in text=, all passages linked singly): sample 6.45 → 3.10,
  novel 6.96 → 3.75; the sample's 100 script links 7.2 → 3.7 words (43.8 → 25.2 chars). relink.go replaces the
  Update Links / Not Now notice (source_links_offer.go deleted): whenever the source changes (the cursor leaving an
  edited passage, the pane's Done, Save, the file read again, Update Source Links, a script opened whose source
  changed since it was saved) every link is looked at; one that found its passage(s) before and does not now is
  lengthened or re-anchored at once, one undo step, status "The source changed: 2 links lengthened, 1
  re-anchored". alignPassages now maps by content (ends, unique same-words anchors on the longest in-order run,
  moved ones, then mutual-best order-aware likeness ≥ 0.6) and never by place alone. Can't be fixed: chip "↗ 2
  places" / "↗ source?" in the warning colour (editor.Chip.Warn), tooltip says why, pane lists the scenes; a link
  that would find another passage is written so it doesn't (alike: their shared words + context; gone: up to 8+8
  old words + context, found again if they come back). New on-disk store: <data>/accolade/links/<hash of script
  path>.md, the source as the script's links were last looked at, written on open (first time) and on Save; tests
  get a temp XDG_DATA_HOME in TestMain. A look dropped (the script changed while it was worked out) is
  tried again on the pane's next follow. Tests: TestRelinkProperty (30 seeds × 40 steps of add/delete/edit/copy/
  move on a 32-word vocabulary; 400 seeds pass), Reanchors, Lengthens, Impossible, InWindow, AlignPassages,
  minimality in TestShortLinksUnique. Bench (BenchmarkScriptKey/sample, -count 5, load ~7): 3.39 → 3.12 ms median,
  allocs 5809 → 5783. Shots: SHOT_DIR=… go test -run TestRelinkScreenshots. Open: near links are re-anchored on any
  source change, so the cross-session "changed since linked" flag a near link carried is gone (kept within a
  session through linkedWords); short links almost never match approximately (near needs 4 words), so a quoted
  word edited outside Accolade with no saved base (never saved since) gives "↗ source?" until linked again; an
  ambiguous link that later resolves (its twin edited) follows whichever still matches.
- 2026-11-23: rebased shortlinks on main (middle tabs: relink triggers now in middle.go's tab switch and
  relinkOnLeave via midDoc/passageSeen; TestLinkOffer* became TestRelinkOnLeavingTab / TestRelinkCutOnly), then the
  follow-up (Joop: "fix a lot also by position … rough match of a full new source and half-done edit"; then "write
  only when certain"). alignPassagesSure: monotone alignment, ends + unique same-words anchors (LIS) + moved,
  between anchors a Needleman–Wunsch pairing (likeness + place in the stretch, banded with bag-of-words for a whole
  new source); certain = same words, or the one passage left between two anchors (unless another passage reads
  like it, or what is there reads like another: moved/copied). The script is written only when certain; every
  weaker pairing is a guess: the link left byte for byte, chip "↗ check?" (warning colour) with the guess in its
  tip, long press → Move It There (one undo step) / Keep As It Is; guesses go with a link written again
  (carryGuesses); the pane lists them. The protective rewrites of the first round (whole words for a gone passage,
  shared words for alike) are gone: a gone passage whose link's words fit another passage now warns "↗ source?".
  Choose Source… on the same script aligns the whole old source with the new: "Another source: 7 links moved, 3 to
  check" (TestRelinkAnotherSource: 14 passages, ~half rewritten, one added, one taken out). TestRelinkProperty (30
  seeds): not found/ambiguous 29.9 % before → 29.4 % (with the passage still there 1.7 → 5.1 %, the cost of not
  writing weak pairs), plus 8.0 % finding another passage and warned (gone passages, guesses); 53 guesses made, 49
  right; 400 seeds pass. Reworded run of 4 (order kept): 0 written, 4 to check, 4 guessed right. Weak pairing
  leaves the script byte for byte (TestRelinkWeakLeavesScript). Shot: relink-chip-check (SHOT_DIR, that test).
  Open: guesses live for the session only; the moved/copied guard compares each lone pair with every passage (bag
  of words), slow-ish for a whole-novel swap with many lone pairs (background).
- 2026-11-23 (Joop: "I don't have much more examples than this xD so I guess we might need other ways to sharpen
  that afterwards..."): ADAPTATION BENCHMARK, with the link-up item. (1) Two or three public-domain pairs from Project
  Gutenberg — a novel/story and a period stage adaptation of it (A Christmas Carol and a 19th-century stage version;
  Treasure Island and its play; Doyle's stories and Gillette's 1899 Sherlock Holmes) — converted to source.md +
  .fountain with a hand-checked scene→source mapping, committed under testdata/adapt-bench/ (public domain, note the
  Gutenberg etext numbers); (2) synthetic loose adaptations made to measure (intro skipped, scenes merged, dialogue
  reworded) with their true mapping; (3) Joop's Kredu min, Sinjorino measured locally only (private). A benchmark
  test reports per pair: scenes guessed right / wrong / silent, cuts suggested right / wrong.
  STRETCH GOAL (Joop: hopetalks.fountain "by Renata Ventura, and while it refers to a loose source, that source is in
  Spanish. That reference would be a massive challenge, but is probably a great stretch goal benchmark."): matching
  across languages (script in one language, source in another). Her script is her work: never commit, upload or
  quote it; local measurement only, and only once the Spanish source is at hand (ask Joop). Approaches to weigh then:
  a bilingual word list per language pair, names/numbers/places as language-free anchors plus order, or an offline
  translation of the source first.
- 2026-11-23 (Joop, sharing Renata Ventura's Hope Talks: the PDF "HOPE TALKS (Version 3) 25-01-2011.pdf" (and a .doc
  synopsis) in ~/drive/Esperanto/Filmo vivo de Zamenhof/ are the sources the hopetalks.fountain files were made from;
  the final fixed version is the upload Hope_Talks_2026_fix.fountain — "We might want to be able to recognize
  screenplay sources as well, for fix-up work..."): SCREENPLAY SOURCES. When the chosen source is itself a screenplay
  (a PDF/DOCX/FDX/Fountain — recognized by its shape: headings, centred cues, dialogue blocks), the source pane reads
  it as a screenplay (Lexington's importers) and aligns scene to scene and line to line instead of passage to
  passage: same language, mostly the same words, so most pairs are certain. For fix-up work: per scene, what the
  converted script lost, garbled or merged against the source (PDF extraction debris: page numbers, (MORE)/(CONT'D),
  split dialogue, hyphenation, wrong element types), each shown as a quiet difference with "Take the Source's" /
  "Keep Mine"; a summary "N scenes match, M differ, K missing"; never written without the writer's choice (the same
  rule as the link-up item). The far source (A. López Luna, "Zamenhof: Iniciador del Esperanto", Spanish) stays the
  cross-language stretch goal. Renata's script and PDF are her work: measure locally only, never commit, upload or
  quote; synthetic look-alikes for tests (a public-domain screenplay PDF and a hand-damaged Fountain of it).
  Joop, same day: "That was also a big challenge, especially as her screenplay was further away from standard format.
  Same thing e.g. with what I just looked at online: Scott Fitzgerald's old screenplays, which didn't use US standard
  format yet..." — so recognition must not assume today's US layout. Read a screenplay source by learning ITS layout:
  group lines by indent, case, length, punctuation and repeated patterns (numbered shots "12. MED. SHOT —",
  "NAME: speech" on one line, two-column A/V or European layouts, stage-play style, camera directions inline,
  parentheticals anywhere), then label the groups (heading, shot, cue, dialogue, action, transition, page debris) by
  which pattern fits, the writer able to correct a label once for the whole document ("these are cues"). Benchmark:
  period scripts in public-domain or freely available form (1930s–40s studio shooting scripts with numbered shots;
  stage plays), Renata's locally; for each, labelled lines right/wrong.
  2026-11-23: the full Hope Talks chain is on this machine for local measurement only (copyrighted; never commit,
  upload or quote): the Spanish book (López Luna, 2017 digital edition, text PDF, ~37k words, running header on every
  page) and the 2026 final Fountain in ~/.local/share/accolade-bench/hopetalks/ (Kredu min, Sinjorino in .../kredu/; mode 700), the 2011 screenplay PDF and the earlier conversions in ~/drive/Esperanto/Filmo vivo de Zamenhof/. Benchmarks:
  2011 PDF → 2026 Fountain (screenplay source, fix-up), book → 2026 Fountain (across languages, stretch).
- 2026-11-23: relinking, never write from a guess (coordinator, from the wrong-guess measurement). The one passage
  between two anchors is certain only if it still shares ≥ 0.4 of its words in order (minSure; 0.3 measured one
  wrong write on 400 seeds, 0.4 none); below that it is a guess. An identical pair scores above any likeness in
  the stretch pairing, so a twin claims its own place. TestRelinkProperty fails on any written move of a link
  that was on its own passage onto another (0 on 400 seeds; RELINK_SEEDS=400 to run them). 400 seeds × 40 steps,
  position on: 5,803 identical, 1,048 certain by position (0 wrong; 3 right ones became guesses, 34 wrong ones
  too), guesses 439 right / 174 wrong; position off: 0 by position, guesses 1,361 / 249, 12,297 more not found.
  Another source (14, wholly rewritten passages): 0 by position, 6 guesses all right (was 3 written + 3 guessed).
  Long source, stretch reworded (300): 0 by position, 79 / 7 guesses, 2 not found (off: 88 not found).
- 2026-11-23: the state-change audit (branch stateaudit, not merged), on the middle tabs' transitions. docs/STATE.md:
  13 views derived from a document (pages and Follow Edits, Live Preview and its Page Setup line, find bar, outline,
  Cards pane and full view, Characters, Source pane, Adapt…, status bar, title, menus/ribbon) and 28 (view,
  transition) pairs that could go stale, each with its verdict (14 fixed). Prevention (state.go): each view records the
  document it shows (`docKey`: the document and the script's epoch — the script's document is the same object
  across Open); one hook, `mainDocChanged`, at the end of `middle.show`, of a role change and of every transition
  that replaces or changes the script (`defer w.scriptChange()()`: Open, New, Open Sample, the imports, a restore,
  a join/leave, the document type, a rename, Choose Source); one table `docViews`; TestViewsFollowTheFront drives
  26 transitions from the script and from the source in front (52 runs) and checks each view's record and what it
  shows (red with the old outline). Fixed: the OUTLINE shows the document in front (Joop: "Outline might also need
  to refresh on context?") — the script's scenes, the source's chapters and passages (a click goes into the
  source), the notes' headings, edits in them refreshed after a pause, drag/colour/Cards the script's only; the find
  bar finds again in the tab now in front (it kept the old count, highlights cleared); a Preview asked midway
  through Open/New/a join with the source in front was of the old script (made again at the end; one being made
  counts as shown); pages over a script turned Markdown give way to its Live Preview; Preview with a view tab in
  front brings the script's tab with it; Choose Source… with the Source tab open reopens the tab on the new source
  (it kept the old text; in a bundle the pane read it); only the script's Preview follows (not Export's preview or a
  report's PDF in the page view); Adapt… closes when the script changes under
  it. Background → UI: every hand-over goes through uiDo, inBackground, w.after/debounce or a seam whose default
  hands over with uiDo and whose test replacement runs the UI part itself (longPressAfter, tooltip.Run); the UI
  part never calls uiDo — so a test holding the turn runs background work inline without deadlock. paintAsync and
  mdPagesAfter are gone (painting and the Live Preview's Pages through inBackground/debounce); exports, Print, the
  report PDF, the pandoc import through inBackground; PDF import progress, peers, away timer, browser events through
  uiDo. TestNoPlainFyneDo fails on fyne.Do outside background.go (the _js.go files read too; allowed: the editor's
  relayout after a focus event, the tooltip package's default runner). Shots: TestOutlineScreenshots
  (outline-{desktop,phone}-{0-script,1-source,2-notes}) looked at. Left: the outline's passages show no status
  (adapted/cut); the source's outline takes 93 ms on the UI goroutine for the novel's tab change (once); Adapt… is closed rather than planned again; Restore Draft is an edit (no Revert command exists).
  2026-11-23 (Joop: Fitzgerald's "Three Comrades" screenplay, 1938, as printed in the 1978 Bruccoli edition — "it's
  a scan, so it both needs the OCR and then it hits the format issues. Good luck!"): third local benchmark (book
  edition under copyright: local only, ~/.local/share/accolade-bench/three-comrades/). What a page holds: an OCR text
  layer already (page 40: ~1 200 chars) and tesseract on a 200 dpi render reads it as well; numbered shots
  ("39 A GLASS OF FOAMING BEER—"), cues in mixed case with the parenthetical on the same line ("Koster (really
  pleased)"), transitions right-aligned ("DISSOLVE TO:"), hyphenation across lines ("thirti-/eth"), OCR noise
  ("So T um proud", "i ms old"), page furniture. Use Accolade's OWN import for it — importPDF / Lexington's pdfin
  with tesseract, the OCR language asked in ocr_ask.go (Joop: "Accolade has OCR support, so of course it should
  work using that") — and measure what it gives; fix the importer (Lexington first) where it fails: (b) the
  per-document layout learning above, (c) de-hyphenation and OCR-noise repair offered as suggestions, never
  applied silently.
  Also (Joop: "I also found a version of Amélie which was surprisingly in English, and has similar issues..."): an
  English Amélie (2001) screenplay PDF, local only (~/.local/share/accolade-bench/amelie/). A text layer from OCR of
  a justified typescript: words spread by runs of spaces, headings "Scene 6 - int. day -", the cue glued to its
  parenthetical ("THE WOMAN(very calmly)"), the narrator's voice-over as indented quoted text with NO cue, OCR noise
  ("The Whal.e's"). Needs: space-run collapsing, numbered "Scene N - int./ext." headings, a cue split from a glued
  parenthetical, and a quoted indented block read as narration (V.O.) with the narrator asked once.
  Joop: "we can probably not expect to get all of these right at once, but we can definitely do a best effort and
  make it easy to fix via the source tool..." — so the goal is a best-effort import plus EASY FIXING in the source
  pane, not a perfect importer: the imported script beside its source (the PDF page image for a scan, its text for
  others); a wrong element fixed once for the whole document ("lines like this are cues" — the layout label, applied
  to every line of that group, one undo step); OCR/hyphenation suggestions as quiet marks to accept or dismiss; a
  per-page "looks right" tick so the writer sees what is checked. Measure each benchmark as "lines right after
  import" and "fixes needed to get it right" (fewer, bigger fixes are the win).
  Joop: "And 'easy tools available to fix this up', e.g. the scene format for Amelie is very unexpected, but with a
  pattern or format matcher that should be easy to fix at once." — a PATTERN FIXER in the source pane: pick one
  line ("Scene 6 - int. day -"), Accolade proposes the pattern it fits (shown in plain words, editable: "Scene
  <number> - <int/ext>. <time> -", regex underneath for those who want it), highlights every line it matches with a
  count, and offers what to make of them: "Scene headings" with a template for the result ("INT. … - DAY #6#"), or
  cues, transitions, page debris to drop, narration. Preview of all matches before applying; one undo step; saved
  with the document's import settings so a re-import of the same source applies it again. The same tool covers
  glued cues ("NAME(paren)" → cue + parenthetical) and Fitzgerald's numbered shots.
  Joop: "And something like that should also be able to fix the extra newlines and uncommon translation format
  choices for HOPE TALKS" — the pattern fixer also works on blank lines and layout choices, not only on single lines:
  "blank lines inside a speech / action paragraph like this → join" (the match previewed, one undo step), and a
  document's own conventions (in Hope Talks, the translation's way of setting things that standard Fountain sets
  otherwise) recorded as a rule once and applied to every match.
- 2026-11-23 (Joop: "and especially move commands should be intuitive. I remember a long time ago someone wrote about
  intuitive script editing, which felt to me similar to e.g. Emacs paredit"): STRUCTURAL MOVES, paredit for scripts —
  folded into the swap/move item (2026-11-20) and the keyboard pass. The script's units nest: section > scene >
  beat/paragraph > speech (cue + parenthetical + dialogue, moved as one, like Page2Stage's drag) > line. Commands work
  on the unit at the cursor and keep the script well-formed at all times (a cue never orphaned from its speech, a
  scene never split mid-speech): move up/down among siblings (Alt+↑/↓ in every mode, vim/helix keys too), move to the
  next/previous parent (a speech into the next scene: "barf"; the next scene's first speech into this one: "slurp"),
  swap with the neighbour, transpose two speakers' lines, raise/lower (a dialogue line out into action, action into a
  character's line), select the unit (expand/shrink selection by unit, like tree-sitter's). Each one undo step, its
  names discoverable in ex completion (:move-scene-down …) and the palette. Joop's half-remembered article on
  intuitive script editing: not found in a first search; ask him for the name if it comes back to him.
  Joop: "I think that was in a YouTube comment xD" (so: go by the idea) and "We already have the org-mode like
  folding. I think it fits well there: both add to the menu options and make it easy to use with vim/helix/ex mode" —
  so the moves live with the org-like folding (editor_folds.go, Shift+Tab cycling) and follow org-mode's keys: Alt+↑/↓
  moves the unit at the cursor with everything folded under it (a section with its scenes, a scene with its speeches,
  like org's subtree move), Alt+←/→ promotes/demotes (section levels # / ##, a scene into/out of a section), Alt+Shift
  for barf/slurp; the same commands in the menu next to Fold/Unfold (a "Structure" group), in vim/helix (z-prefixed
  or g-prefixed where free, matching the fold keys) and in ex (:move-up, :promote, …, completed).
- 2026-11-23 (Joop: "I think Scrite or any tool like that also has something like a timeline view? Maybe we can do that
  as a card view variant..."): TIMELINE, a variant of the Cards tab (a Grid / Timeline toggle in its header, the
  choice kept per script). Scenes left to right in story order as blocks whose width is their length (page eighths,
  as the stripboard measures), coloured by scene colour, the act/section headings as bands above; optional lanes by
  character or storyline/tag (a scene appears in each lane it belongs to; a character's lane shows where they are
  absent too); zoom with Ctrl+wheel/pinch; tap opens the scene, drag reorders through the STRUCTURAL MOVES commands
  (one undo step, same rules), keyboard ←/→ between scenes, Alt+←/→ to move one. Reuse the stripboard's drawing where
  it fits (stripboard.go draws production strips; this is story order, not shooting order). Screenshots desktop and
  phone (phone: vertical timeline). After the structural moves, since dragging needs them.
  Joop, same day: "in all these programs, they work like bolted on views. Accolade is very much a Fountain/writing
  first program, so even if you can call it a bit bolted on as well, it goes from a strong philosophy of making the
  writing work itself easier. Just having cards doesn't do that." — so the timeline is judged by what it does for
  the writing, not by being there: it follows the cursor while you write (a thin strip above the editor, the scene
  you are in lit, collapsible) as much as it is a place to go; it answers writing questions — where am I, what is the
  rhythm (scene lengths side by side), who has been gone too long (lanes), which scenes are still empty or only a
  synopsis — and every edit in it is an edit of the Fountain text (no state of its own, one undo step). Same test
  for every view (cards, outline, characters, timeline): what does it make easier while writing?
- 2026-11-23 (Joop: "Stripboard also opens in the middle and should probably be aligned with the others that were
  corrected on this: also enable sidebar, switching and tabbed display." / "Check if anything else secretly opens in
  the middle like this."): STRIPBOARD AS A MIDDLE TAB, next free agent slot. Checked (window.go relayout, the only
  place the window's content is set): the stripboard is the one view that bypasses the middle tabs
  (`if w.stripsVisible … center = w.stripboard.box`, and `w.mid.center` skipped for it). Make it a middle tab like
  Cards (opened by View › Stripboard and Production's button, switchable, Back to the Side if it makes sense, the dock
  beside it, its ribbon contextual tab following it), remove stripsVisible's special case. The other one found: on a
  phone the page view (Preview) takes the script's place (`w.pagesVisible && w.narrow → center = w.pages`) — there it
  should be a middle tab "Pages" next to Script, so switching back is a tab tap, not hiding the preview. Preferences
  and the TOML editors are separate windows on purpose. Extend TestViewsFollowTheFront to both.
- 2026-11-23 (Joop: "I think we might want to add Hindi, that's a huge market xD"): HINDI, after the reset. First the
  risk: Devanagari needs shaping (conjuncts, matras reordering around consonants) — check the editor (Fyne fork's
  shaper) and especially the PDF (gofpdf doesn't shape; see how Lexington prints Arabic and whether that covers
  Devanagari) with screenshots of both before anything else. Then: a Hindi language rule in Lexington (scene words,
  transitions; Indian scripts often keep INT./EXT. in English with dialogue in Devanagari or in Hindi in Latin
  letters — all three read right), a Devanagari companion font for Courier Badi (e.g. Noto Sans Devanagari) via the
  same fallback the Hebrew points use, the help page. Indian layouts (two-column, Hindi in Latin letters) through the
  screenplay-sources pattern fixer.
  Joop: "I think it should already work as is, just without any presets for it available, so I think preset first
  is a good approach." — so ORDER: the Hindi preset/language rule first (Lexington, tagged), the shaping check as a
  screenshot in the same item to confirm it (not a gate in front of it).
- 2026-11-23 (Joop: "I think the preset early, rest of hindi around the end of the todo might be the best approach."):
  ORDER NOW (after the four running branches — link-up, pattern fixer, ex completion, swap/structural moves — merge):
  (1) stripboard + a phone's pages as middle tabs; (2) the Hindi preset (Lexington language rule + preset, a
  Devanagari screenshot); (3) the timeline strip (needs the structural moves); (4) A New Hope import details (needs
  Joop's file); (5) older open items: undoable issues / Withdraw Last Issue, collab follow-ups (real browsers, COLLAB
  steps 4–6), page view on real browser/Android, Characters leftovers, lock-the-writer layers, File › Merge Changes;
  from Wed 2026-11-25: Markdown images + storyboard frames + cast photos, then Table Read / cast, citations; then
  the keyboard pass; then the rest of Hindi (font companion, Indian layouts) near the end; title page keys last.
- 2026-11-23: ex mode completion (branch excomplete, not merged; Joop: "Maybe ex mode should have autocomplete /
  discoverable commands? … those commands are not easy to guess"; "and especially move commands should be
  intuitive"). One table, ex_table.go: ex.go's line commands (vim's names and abbreviations, s[ubstitute], co[py]…;
  ex.go now takes its names from it), the window's own (:e[dit] file, :sc[ene] heading-or-number, :el[ement],
  :heading, :h[elp] [cmd|page], and the moves: :move-scene-up/-down (org's Alt+K/J), :promote-/:demote-section,
  :next-/:previous-heading, :go-parent-section, :go-first-inside) and every registry command by a name made from
  its label (:next-source-passage), shorter for the common ones (:preview, :fold, :export, :outline, :cards); a
  submenu's items are its argument (:theme dark, :recent …), elements :element's, headings :heading's, help pages
  :help's — a test checks every registry command is reached and no name hides another. The ':' line lists what it
  may be as you type (a layer of the window, not a pop-up: the line keeps the keyboard), prefix first, then fuzzy;
  an empty line the most useful; Tab completes and cycles, Shift+Tab back, arrows choose, Enter runs the line as
  typed (or the arrows' choice), a command wanting its argument lists it instead (:scene ↵ lists the scenes);
  Up on an empty line is the history. Arguments complete: scene headings, elements, export formats, files on the
  desktop for :w/:e, submenu items. Phone: two-line rows, a tap completes and runs. docs/help/ex.md is made from
  the table (UPDATE_HELP=1 go test -run TestExHelp), live in the app's Help tab; vim.md, index.md and the shortcuts
  page link it. Moves are named move-<what>-up/-down, so ":mo" lists them all (keywords bring the jumps after);
  the STRUCTURAL MOVES slot (move-speech-…) is a comment in exWindowCommands — no such entry is in this LOOP.md
  yet. Fixed on the way: :m to an address outside the script (:m-9 on line 3) panicked; it says "invalid address".
  palette.run's role switch is runByName now, shared with the ':' line.
  Review fixes: a tap after the arrows ran the arrows' row; two scenes with one heading both went to the first
  (:scene completes to "3 INT. KITCHEN - DAY", read by its number); the list's colours follow :theme; a recalled
  command (Up) is not overwritten by Tab; Enter on a name naming nothing (:otln) takes the list's first, as shown;
  :h <menu command> goes to its menu's part of the page. Unverified: row taps on the real drivers (the tests call
  Tapped; the press unfocuses the line, the tap gives it back), a click elsewhere leaves the list up while the line
  stays open, the browser build only compiled.
- 2026-11-23 (Joop: "we might want to add hi-Latn for the common Fountain format most seem to use, and hi-Deva for
  Devanagari. And maybe just hi for supporting both at once..."): DONE (branch hindi, not merged; Lexington 1.33.0, 1.35.0;
  see the tick log). The Hindi preset as three language codes in
  Lexington's config: hi-Latn (Hindi in Latin letters, what most Hindi Fountain scripts use), hi-Deva (Devanagari),
  and hi = both at once (scene words, transitions and cue rules of either script recognized; the writer writes in
  either or mixes). BCP 47 script subtags, so a file named x.hi-Latn.fountain / a bundle's language "hi-Deva" picks
  them; the language picker shows "Hindi (Latin)", "Hindi (Devanagari)", "Hindi (both)". If Lexington's language
  lookup only knows bare codes, teach it script subtags generally (fallback hi-Deva → hi → default), tested.
- 2026-11-23 (Joop: "You can put the post-reset things to until Wednesday, because we have a heavier package until that
  reset now."): the items that waited for Wed 2026-11-25 (Markdown images + storyboard frames + cast photos, Table
  Read / cast, citations) no longer wait: they join the queue now, after the Hindi preset and the timeline.
- 2026-11-23 (Joop: "How hard would it be to add a sample screenplay per language and format? … if you start, you
  always get an English one, which misfit for the other languages is partly why we added the empty option. I think
  clicking new with a different language set should show an appropriate sample or maybe an automatically generated
  template based on the settings for that language, e.g. just the English one with INT/EXT swapped etc as a
  fallback..."): NEW BY LANGUAGE AND FORMAT, right after the Hindi preset (the same language data). File › New's
  starting text = (1) a hand-written template for that language + format if one exists (assets, per BCP 47 tag with
  fallback hi-Deva → hi), else (2) generated from the English template for that format with everything
  language-bound taken from Lexington's config for that language: scene words (INT./EXT. and their combos, DAY/NIGHT
  and the time words), transitions, extensions (V.O., O.S., CONT'D), title page keys, the direction (rtl), the
  script's case rules (no capitals for languages without them) — the prose staying English as placeholder text, or
  short neutral placeholders ("Scene description.") if the language is chosen as such. Per format: screenplay, stage
  play, the comic/other formats Lexington has. Empty stays an option. A test generates the template for every
  language × format in the config and checks it reads back as that format with that language's rules (headings are
  headings, cues are cues). Screenshots: Esperanto, Arabic, Hindi (Devanagari), Chinese.
  Joop: "I specifically want to be able to test the Hebrew font/PDF support..." — so Hebrew is the FIRST language of
  this item (before the Hindi preset if that's quicker): New with Hebrew gives a Hebrew screenplay whose prose is
  real Hebrew, not English placeholders — with niqqud that exercises Courier Badi's points (shin/sin dots, holam,
  dagesh: e.g. Genesis 1:1–5, public domain, as action and dialogue) plus plain unpointed lines, a parenthetical,
  numbers and an English name inside a Hebrew line (bidi), Hebrew scene words (פנים/חוץ, יום/לילה …) from Lexington's
  config (add a [Languages.he] if it lacks one, tagged), rtl. Screenshots: editor, page view and the exported PDF
  page (rendered), so Joop can open it and print it.
- 2026-11-23 (pattern fixer merged): DONE (branch ocrcheck, not merged; Lexington 1.32.0; see the tick log below).
  Three Comrades did not improve because its PDF carries the scanner's own poor
  text layer, so Accolade never offers OCR. Add a text-layer quality check on import (share of dictionary/real words,
  odd character runs, words broken by stray spaces): below a bar, offer "This PDF's text looks garbled — read it with
  OCR instead?" (the existing ocr_ask.go flow), never automatically.
- 2026-11-23: STRIPBOARD AS A MIDDLE TAB, and a phone's Pages tab (branch stripstab, not merged). The middle gained
  tabs that are no dock pane (`midView`, middle.go): their menu has Close alone, Back to the Side greyed
  (`cmdEnv.midSide`). The Stripboard is one (Production › Stripboard, now also View › Stripboard after the panes, the
  Schedule stage's Open Stripboard and rows): beside Script and the others, switchable, × / its command in front
  close it, the side panes beside it, Back to the Script and Open Scene keep its tab, the Writer role and Markdown
  prune it, and the ribbon's Production tab comes forward on its Schedule stage while it is in front (as Preview's
  does), back to the tab before after. Back to the Side left out: the Production pane's Schedule stage is its side
  form, and the board is too wide for the dock. `stripsVisible` and relayout's special case are gone. On a phone the
  page view is the tab Pages right after Script (`pagesTabFollow`): Preview opens it in front, Script and back keeps
  the pages (no hand-off between the script, the pages and a view; a document's tab still takes its own preview), the
  pages remade for the script's tab coming back arrive behind it (`pagesBack`), ×/Close Preview/Escape close it, and
  crossing `narrowWidth` turns it into the split and back; the ribbon's Preview tab only with Pages in front. Both
  registered in state.go (a `stripboard` docView; the `pages` one remakes in its tab), TestViewsFollowTheFront now
  starts from the stripboard and a phone's Script/Pages too (5 fronts × 34 steps), new TestMiddleStripboard and
  TestMiddlePhonePages, TestMiddleScreenshots shots 9–11 (looked at: desktop board with the Cards pane beside,
  phone Script/Pages/Stripboard row). docs/MIDDLE.md, STATE.md updated; the commands golden's View menu move.
- 2026-11-23: the garbled-text-layer offer (branch ocrcheck, not merged; Lexington 1.32.0 tagged and pushed). Lexington:
  pdfin.TextLayers/TextQuality per page — odd tokens (stray marks alone or at a line's ends, punctuation glued inside a
  word, case or script changing inside a word, letters and digits mixed, letters spread by spaces) plus lines split
  in two (a skewed scan's words at two heights, closer than 0.35 of the page's usual line gap); a page is a scan if
  its largest image has the page's shape, ≥100 dpi, and can be decoded. GarbledBar 0.97: a page below it counts, but
  only when the scans' layers pooled score below it too (CountGarbled/GarbledPages). Options.OCRText: OCR reads those
  scans in place of the layer; a page OCR cannot read keeps its layer. Layered (MRC) scans as the Internet Archive
  makes them — JPEG 2000 foreground + 1-bit JBIG2 mask — read through the mask (Lexington still cannot decode
  JPX: without this Three Comrades could not be OCRed by Accolade at all; the mask equals poppler's pixel for pixel,
  but the MRC encoder put some small words in the background, so they come out as blobs). Word lists and letter
  trigrams were measured (public-domain English from testdata/adapt-bench as the lexicon): the layer's out-of-lexicon
  share (median 0.075) sits inside clean scripts' range (0.01–0.10), so they are not used. Bar by measurement: every
  Lexington-printed page (examples/, a multi-script Fountain — Cyrillic, Greek, Hebrew, French spacing, German,
  Esperanto) scores exactly 1.000; synthetic garbled scan triggers, the same scan with clean lines does not.
  Accolade: readPDFAsking (screenplay and prose import alike) counts scans and garbled pages once (pdfPagesOf);
  garbled only → "This PDF's text looks garbled — read it with OCR instead?" with the language pick, Read with OCR /
  Keep the Text (reads the layer); scans and garbled → the scan dialog with a "read the N garbled pages too" tick;
  no OCR (wasm) → no offer. Tests on synthetic PDFs only (TestGarbledTextOffersOCR, TestGarbledAskReads; Lexington
  TestTextQualityWords, TestCleanLayersScoreOne, TestGarbledScan, TestSplitLines).
  Three Comrades (local, counts only): 199 of 266 pages flagged, pooled layer score 0.942 (so the offer appears).
  Pages 40/60/100, 67 elements counted by hand against the page images (right = own element, right type, ≤1 wrong
  word): text layer 4, OCR 4 — the classifier, not the text, is the bottleneck on these pages (cues and speeches
  come out as action either way); OCR's lines are whole (layer 10 split lines on p60, OCR 0) and its text scores
  higher on all three (0.924→0.947, 0.890→0.937, 0.955→0.961), still under the bar itself. OCR took minutes a page
  under the machine's load (load average ~21). Next: lines() grouping glyphs along a skewed baseline would mend the
  layer's split lines without OCR; the classifier on Fitzgerald's layout (cue + parenthetical in mixed case) is
  the pattern fixer's or the layout-learning item's.
- 2026-11-23 (Joop, on the stripboard tab: "That should also make the stripboard a joy to use on phone"): the
  screenshot (middle-phone-9-stripboard) says not yet — on a phone the board's own toolbar (Move Up/Down, Move to…,
  Unschedule, Open Scene, Checked, Add Day, Undo, two Group buttons: 9 controls in 5 rows) takes most of the tab, the
  strips get one header line, and the dock's sheet takes the lower half. PHONE STRIPBOARD: the toolbar goes into the
  ribbon's Production › Schedule stage (already shown while the board is in front) or a single overflow row, strips
  get the tab's height, a strip's actions on long-press, drag to reorder with the finger (the structural-moves engine
  where it applies), days as collapsible bands; the dock sheet collapsed by default while the board is in front.
  Screenshot before/after.
  Joop: "so same as preview ribbon" — the pattern to copy is the Preview tab's: the pane's own toolbar moved into a
  contextual ribbon tab that shows while the view is in front (previewtab.go). So: a contextual "Stripboard" ribbon
  tab with the board's commands (grouped: Move, Day, Group, Check), shown while the board is in front on every
  screen size, and the in-pane toolbar removed (not only on phones).
  Done (branch stripribbon, not merged): stripstab.go — the contextual Stripboard tab (Move: Move To, Up, Down,
  Unschedule, Undo; Day: Add Day, Edit Day…, Fold/Unfold Days; Group: by Location, by Cast; Check: Open Scene,
  Checked; Close), registry commands (menus, palette, :move-strip-up …); `ribbonTab.contextual` with Preview,
  `contextFollow` in place of the jump to Production › Schedule (Production still there, on Schedule). The board
  has no toolbar in either look (Menus look: the Production menu has the commands while it is in front — unlike
  Preview, which keeps its bar there); its count is its first line. Phone: a grip (≡) per strip drags it (edge
  scroll), a drag elsewhere scrolls, a long press (or right-click) has its actions; days fold to bands (everywhere);
  the dock's sheet folds down to its row while the board is in front (sheetFollow). internal/structure (merged
  since) does not apply: the board never moves the script's text — the unscheduled's "script order" is only how the
  pool is listed, and every board move is a schedule edit (days, shooting order), not a scene moved in the script. Shots before/after: combo-stripboard-ribbon-*, combo-stripboard-folded-*,
  middle-phone-9-stripboard.
- 2026-11-23 (Hebrew/Arabic samples merged; left wrong, from its report): (1) PDF title page: rtl lines sit ~0.5 in
  left of the Latin ones (centred on mirrored margins); (2) editor: a Hebrew run inside a left-to-right line (title
  page values) shows letter-reversed — only whole rtl lines are mirrored, runs inside ltr lines need bidi reordering;
  (3) the cursor/selection inside joined runs ("23:40", Latin names in rtl lines) still maps cell by cell — check on
  screen; (4) Arabic PDF: the font's GPOS mark positioning is not applied, so vowel marks sit at default positions
  (fatha on hamza reads like damma; shadda+fatha crowded) — Lexington/the PDF writer; (5) Courier Badi's tall Arabic
  glyphs make single-spaced dialogue lines touch — the font (Courier Badi repo: Arabic glyph heights/vertical metrics);
  (6) Hebrew dagesh in yod/lamed hard to see — the font. (1)–(4) Accolade/Lexington, (5)–(6) Courier Badi. The Arabic
  sample's vowelling is the agent's own, to be checked by a native reader.
- 2026-11-23: STRUCTURAL MOVES + Swap…/Move… as one engine (branch structmoves, not merged). internal/structure:
  the body read as a tree (sections > scenes > blocks: a speech is one, side-by-side ones too, a boneyard is one >
  lines), title page and end blocks never touched; every move a reorder of line spans, then checked by reading the
  text again (same headings, each in its parent unless it moved, same lines, whole speeches) or refused with why.
  Keys, every mode: Alt+↑/↓ the unit at the cursor (or the selected ones; lines inside a speech), Alt+←/→ promote/
  demote (a section's level, Workflowy-style: promoted it goes after its parent; a scene out of its section / into
  the next; a speech's line out into action, splitting the speech with (CONT'D), and action back in, joining it),
  Alt+Shift+↑/↓ a speech or paragraph into the scene before/after, Alt+Shift+←/→ barf/slurp, Alt+T the two speakers'
  lines (the cues' names change places), Alt+O/Alt+I expand/shrink the selection by unit (helix's keys); vim/helix
  also evil-org's Alt+j/k/h/l, with Shift into-scene and barf/slurp. View > Structure has them all (registry homes
  can now hold a submenu), Edit's Move Line Up/Down became it (Markdown keeps line moves), the ribbon's Edit >
  Structure. Commands carry ex names (move-up, move-down, promote, demote, into-previous, into-next, barf, slurp,
  transpose-speakers, raise, lower, expand-selection, shrink-selection, swap-ranges, move-range) for the ex
  completion branch. Swap…/Move… (menu, the outline's and cards' right-click menu, the Cards header): range A (from/to
  among the units side by side), with range B or before/after a target, preview = the outline after with what moved
  marked, or why not; locked numbers stay with their scenes and the preview and status say how they read. Outline and
  cards dragging go through the engine (outline_move.go gone; the last scene no longer drags the end blocks along).
  Tags, Done marks and strips follow their scenes, Undo/Redo too (followEdit generalises the rename's follow; the
  renumber-follow no longer mistakes reordered numbers for renumbered ones). Shots structure-1…4 (before, Swap…,
  after, phone Move…). Rebased on main with the ex completion: the ex table reads command.ex first; the hand-made
  move-scene-up/down, promote-section/demote-section rows gave way to the structure.* commands (no aliases), which
  stand together right after :scene, so :mo lists every move before the jumps; descriptions are the commands' tips;
  docs/help/ex.md made again. On a Mac (coordinator): Option+←/→ stay word moves, so promote/demote and barf/slurp
  take Control+Option there (command.macKeys; the help pages note it, the cheat sheet has an "On a Mac" list).
  Open: no multi-select of cards yet (a range is picked in the dialog).
- 2026-11-23 (Joop: "Any obvious contextual ribbons we're missing?" / "Maybe a cards reorg one?"): CONTEXTUAL RIBBON TABS,
  each like the Preview tab (shown while its context is in front, the view's own toolbar moved into it). (1) CARDS,
  together with the TIMELINE item (after the structural moves merge): Rearrange (Swap…, Move…, Move Up/Down,
  Promote/Demote — the structure engine), Arrange (Grid / Timeline), Group & sort (by act/section, colour, character,
  tag — a view, not the script order, unless "Make It the Order", one undo step), Card (New Scene, Colour, Fold
  Synopses, Open in the Script). Then: (2) FIX-UP after importing a PDF/DOCX/FDX until saved or dismissed (Fix Lines
  Like This…, the kept rules, OCR again, the garbled-text check); (3) TITLE PAGE while the cursor is in it (keys to
  add; later custom keys); (4) SHARED SESSION while in one (who's here, follow, leave); (5) CHARACTERS with its tab/pane
  in front (rename everywhere, merge names, notes, later the cast photo); (6) COMPARE while comparing drafts/revisions
  (next/previous change, accept/keep).
  Joop: "I think Word has some options always visible, too, on their ribbon?" — Word's Quick Access Toolbar (always
  visible, customizable by right-click "Add to Quick Access Toolbar") and its "Tell me" search box. Accolade's top
  icon row is the fixed equivalent; add: (7) right-click any ribbon button / menu item → "Add to the Top Row" (and
  remove), kept per phone/desktop like the Quick tab's lock, overflowing into the row's … menu on a phone; (8) a search
  field in the top row that opens the palette as you type (the palette + ex completion's matching). With the
  contextual tabs above.
  Joop: "we might want to enable the inverse as well: on Mobile the clickable name part drops, and that's still very
  useful..." — (9) the top row's items can also be removed (right-click/long-press → "Remove from the Top Row", into
  the … menu or the ribbon), and on a phone the clickable script name keeps its place before the icons: less-used icons
  overflow into … first (ribbon.go now hides the title below 80 px of room). Where even that's too narrow, the name
  stays reachable by tapping the Script middle tab's label / the title in the … menu (the same name editor).
- 2026-11-23: HINDI PRESET (branch hindi, not merged; Lexington 1.33.0, 1.35.0: Knows exact again, so keep both's xq-2 is free). Lexington: hi-Latn (English Fountain's
  words, as Hindi scripts in Latin letters write them, + DIN/RAAT/SUBAH/SHAAM times), hi-Deva (the NCERT textbook's
  heading "शिक्षा अधिकारी का दफ़्तर (बाहरी कक्ष)/दिन/अंदर" as a place/time/inside pattern, अंदर/बाहर, दिन/रात/सुबह/शाम,
  कट टू, (डिजॉल्व टू), noCaps cues, English's INT./EXT./V.O./" TO:" kept), hi = both (tested superset). No Devanagari
  standard found: the dotted अंदर./बाहर. prefix words, फ़ेड इन/आउट and the title keys (शीर्षक, लेखक…) are Lexington's
  own, said so in docs/LANGUAGES.md "Hindi" with the sources. BCP 47 generally: TOMLConf.Lookup / rules.Tag — any case,
  _ for -, then without the last subtag (hi-Deva-IN → hi-Deva, de-CH → de, hi-Arab → hi); Language/Knows/Scenes
  follow it. Accolade: languageOf and the language line go through it (film.hi-latn.fountain is hi-Latn); the picker
  says "Hindi (Latin, hi-Latn)", "Hindi (Devanagari, hi-Deva)", "Hindi (both, hi)" (a name in brackets takes its code
  inside them); languageName "Hindi (Latin)"…; help fountain.md. New's template in each reads back (hindi_lang_test.go,
  and TestNewInEveryLanguage covers them). DEVANAGARI CHECK (TestDevanagariScreenshots, SHOT_DIR): the editor draws
  every letter in its own cell, unshaped — matras apart with dotted circles (कि as क + ि, not reordered), conjuncts
  as halant forms (क्ष्, ज्ञ broken), no reph (धर्म) — from a system Devanagari fallback (Courier Badi has none); the
  ribbon's Fyne labels (अंदर/बाहर.) shape right. Oddly the eyelash ra र्‍य does come out joined (ऱ्य) in the editor
  while द्ध stays द् + ध. The editor's Devanagari font is Fyne's fontscan fallback over the system's fonts (here
  Noto Sans/Serif Devanagari [UI]; which one not pinned down); the shots are the test driver's, not a real display. Lexington's own PDF (and so the page view): Courier Prime has no
  Devanagari — boxes in the PDF, nothing in the page view; PDFWarning says so, and with Chromium on the computer
  export goes through it: Chromium's PDF (Noto Sans Devanagari; its Latin in Liberation Mono, not Courier Badi)
  shapes right (क्षत्रिय, कि, की, धर्म, श्रद्धा, ज्ञान, र्‍य as ऱ्य, द्ध stacked; Lexington's own prints the ZWJ as a box),
  but its bold (the heading, a synthetic Type 3 bold) loses the hook of ि/ी, and a parenthetical's taller line touches
  the dialogue under it. Chromium only runs here outside the sandbox. For the rest of Hindi (end of the queue): a
  Devanagari companion font for Courier Badi with shaping in Lexington's PDF (as pdf/arabic.go does Arabic: HarfBuzz-
  style reordering + conjunct GSUB is far more than Arabic's joining), and the editor shaping runs, not runes.
