# Why the source pane freezes the browser (audit 2026-11-20)

Joop: "loading and editing source files kinda freezes the web app". This is a read-only audit. Measurements were taken
natively and as js/wasm under node, which runs 1.4–5.5× slower than native.

The test fixture:
- a novel of 536 KB: 101k words in 2592 passages;
- a script of 160 KB: 200 scenes and 200 links (180 exact, 15 near, 5 missing).

The benchmarks are kept outside the repo (perf_bench_test.go, in the session scratchpad). They are to be added as
real benchmarks with the fixes.

In the browser, Go runs on the page's one thread. A goroutine only gives control back to the page when every
goroutine is blocked, and `runtime.Gosched` is not enough for that. So long work also has to block now and then,
for example with `time.Sleep(time.Millisecond)` every ~20 ms of work.

## Hotspots, worst first, each with its fix

1. **Edit/Done opens the whole source in a `widget.Entry`** (source_pane.go:117, :431–437).
   - Cost grows faster than the text: 100 KB takes 1.9 s to load and 1.8 s per key. The whole novel took 17.6 s and
     22 GB of allocations to load.
   - Fix: edit with Accolade's own editor (internal/editor) in Markdown mode, which handles a 700 KB buffer at
     about 50 ms per key, or edit one passage at a time.
2. **A regexp scans the whole text on every event to find the in-script `/* Source */` block** (notes.go:45
   `notesMarks.at`, via `tailBlocks` adaptation.go:48 and `sourceBlock` source_file.go:84).
   - One call takes 139 ms natively and 765 ms in wasm, and runs about 7× per keystroke, even with the pane hidden
     (through `computeStats`).
   - Fix: a plain scan instead (LastIndex of the opening mark, check the closing mark at the end), byte offsets,
     and bounds cached per buffer revision. The Notes block uses the same code and gets the same fix.
3. **Every keystroke rebuilds the source model, and `follow` runs twice** (window.go:873, :940;
   `newSourceModel` source.go:357).
   - The rebuild takes 171 ms natively. Most of that is fuzzy `matchLink` over every near or missing link
     (`prefixDistance` source.go:238 against every passage).
   - Fix:
     - cache matches by the link's words for as long as the passages are unchanged;
     - in `prefixDistance`, stop at `len(a)+limit` and reuse one row;
     - index the passages by their first word;
     - debounce `follow` by about 200 ms, and run it once per keystroke.
4. **Loading the source and measuring every row happen on the UI goroutine** (`load` source_pane.go:195,
   `measure` :176, `widthWatch.Resize` :497).
   - The first open takes 3.2 s, of which measuring the rows cold takes about 2.5 s.
   - Every width change measures all rows again (1.76 s), and that happens on every step of a split drag.
   - Fix:
     - split into passages in a goroutine;
     - make the list lazy: estimate each row's height and measure a row only when it becomes visible;
     - debounce width changes;
     - replace `prose.collapse`'s regexp with a loop (model.go:261).
5. **Choose Source and Import convert on the UI goroutine** (`chooseSource` source_pane.go:533 →
   `toMarkdown` md_import.go:53; `importDocument` md_import.go:167).
   - DOCX takes 619 ms in wasm, and with the load after it a DOCX freezes the browser for 2–3 s.
   - Fix: the `readPDFInBackground` pattern (a goroutine, progress, `fyne.Do`), plus the wasm yields. Check that
     the PDF path itself yields too.
6. **Browser persistence** (web/memfs.js:30; `keepDraft` browser_js.go:32, called from window.go:859).
   - The whole script is written to draft.json on every keystroke: 24–77 ms per write, plus base64 and a
     synchronous `localStorage.setItem`.
   - `.tmp` files are persisted too, so every save is encoded twice.
   - Old kept source copies are never deleted, and a few novel-sized ones will likely fill the ~5 MB
     localStorage quota. After that, saves fail with only a console warning.
   - Fix:
     - debounce `keepDraft` (about 1 s idle);
     - don't persist `*.tmp` files;
     - encode in chunks, or move to IndexedDB with raw bytes;
     - delete the old kept copy when a script gets a new source.
7. **Bundle autosave and Done rewrite the zip on the UI goroutine** (bundle.go:352, :378, :393 `fresh`, :454).
   - 94–187 ms in wasm.
   - Fix: skip `fresh` when the file's time and size haven't changed since our own write; do the zipping and
     writing in a goroutine.
8. **A cursor move with the text unchanged** costs 8 ms natively (20–30 ms in wasm), from `sourceRef` /
   `sourceText`, up to four `os.Stat` calls, and a `decodeScript` per move for bundles.
   - Fix: cache per buffer revision; check the file's time and size on a timer or on focus, not on every move.
9. **The baseline on every cursor move, even with the pane hidden, is 33 ms natively.** `computeStats` (stats.go:30)
   classifies the whole script every time, and `Buffer.Text()` copies the script several times per event.
   - Fix: cache the stats per text revision; recompute only the cursor's scene on a move.

No action is needed on `inSpan`, `count`, `spanAt`, `linked` (each a microsecond or less), `anchorFor` (57 µs) or
`list.Refresh` (0.46 ms, visible rows only).

Also to look at with these fixes: an occasional data race between the autosave timer and Fyne's theme code
(autosave.go), seen in the race test run of step 5.
