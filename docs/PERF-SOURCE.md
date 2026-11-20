# Why the source pane freezes the browser (audit 2026-11-20)

Joop: "loading and editing source files kinda freezes the web app". This is a read-only audit. Measurements were taken
natively and as js/wasm under node, which runs 1.4–5.5× slower than native.

The test fixture:
- a novel of 536 KB: 101k words in 2592 passages;
- a script of 160 KB: 200 scenes and 200 links (180 exact, 15 near, 5 missing).

The benchmarks are in source_bench_test.go now, with the fixture made in the test
(`go test -run '^$' -bench . -benchmem`). What was done, and the numbers after, are under "Done" at the end.

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

## Done (2026-11-20, branch perfsource)

Native numbers come from this VPS, which was under load, so they are rough. Wasm numbers come from node, with
`GOOS=js GOARCH=wasm go test -c`, a small page shim and 5 runs each. In the browser most of the remaining slow work
runs in the background now, and pauses for 1 ms every ~20 ms (internal/pace).

| Hotspot | Before | After |
|---|---|---|
| 1. Edit opens the novel / a key in it | 17.6 s / 1.8 s per key at 100 KB (Entry) | 51 ms / 44 ms per key for the whole 536 KB novel (internal/editor, Markdown mode) |
| 2. Tail-block scan, script plus source kept in it | 82 ms native, 765 ms wasm, ×7 per key | 2.5 ms native, 5 ms wasm for a new text; 0.1 µs for the same text again |
| 2. The same for the plain script | 5.4 ms | 0.33 ms |
| 3. Model rebuild per key | 110 ms (inside: 205 ms) | 4.9 ms native, 11 ms wasm; runs once, 200 ms after typing pauses |
| 3. One near or missing link | 3.4 ms | 0.85 ms (prefixDistance stops at len(a)+limit; matches kept per link) |
| 3. Keystroke with the pane shown (UI goroutine) | 173 ms (inside: 799 ms) | 20 ms (inside: 60 ms) |
| 3. The follow after the pause | — | 4.5 ms (inside: 5.7 ms) |
| 8./9. Cursor move, pane shown | 39 ms (inside: 365 ms) | 5.6 ms (inside: 8.8 ms) |
| 8./9. Cursor move, pane hidden | 35 ms (inside: 296 ms†) | 4.5 ms (inside: 13 ms) |
| 4. passagesOf(novel) | 206 ms | 93 ms native, 843 ms wasm; in the background |
| 4. First open on the UI goroutine | 3.2 s | 29 ms (rows estimated; measuring follows in stretches, rows in view first) |
| 4. A width change (split-drag step) | 1.76–2.2 s | 33 ms, after a 150 ms debounce |
| 5. Convert DOCX / ODT / HTML | 143 / 154 / 79 ms native; DOCX 619 ms wasm | 104 / 71 / 28 ms native; 507 / 294 / 124 ms wasm, in the background |
| 9. computeStats on a cursor move | 18 ms (inside: 87 ms) | 2 µs native, ~3 ms wasm (counted once per text) |

† measured for a keystroke with the pane hidden.

What changed:
1. **Source editor.** The source is edited in `sourceEditor`, which is internal/editor in Markdown mode inside a scroll.
2. **Tail blocks.** `notesMarks.at` is a plain scan of the text's end:
   - forward `strings.Index` only, since `LastIndex` was slow;
   - the old regexp is kept in tails_test.go and compared with the scan on 20 000 generated texts.

   `tailsOf` works in bytes and remembers the last text it was given. `Buffer.Text` is now made once per change
   (`Buffer.Rev`), so the many calls in one event all get the same string and hit that cache. The Notes block uses
   the same code.
3. **Links.**
   - `linkMatcher` keeps every link's match for as long as the passages stay the same, and finds exact matches through
     a first-word index.
   - `prefixDistanceIn` is bounded and reuses its row.
   - `followSource` follows a cursor move at once, and an edit once, after a 200 ms pause.
   - A direct `follow` cancels a pending one.
4. **Rows.**
   - The rows are estimated first, then measured a stretch at a time on the UI goroutine (`repeat`), the rows in view
     first.
   - Many `SetItemHeight` calls are made with the list briefly empty, because each call lays the whole list out again.
   - A row's RichText is parsed again only when its Markdown changed.
   - A source of 64 KB or more is split into passages in a goroutine.
   - `prose.collapse` is a loop now; it was half of `passagesOf`.
5. **Choose Source and Import** convert in `inBackground`. A result is dropped if another Choose Source or Import came
   after it, or if another script was opened meanwhile (`takeChosenSource`). The PDF progress callback yields.
6. **Browser.**
   - `keepDraft` waits for a 1 s pause in typing.
   - memfs.js no longer keeps `*.tmp` files and encodes in 32 KB chunks.
   - Choosing a new source deletes the copy Accolade kept of the old one, once no script uses it.
7. **Bundles.** `fresh` reads the zip again only if its file's time or size changed since Accolade last read or wrote it.
8. **Cursor moves.** A bundle's source is decoded once per bundle read. The file `Stat`s stay on the cursor path: a
   source changed in another app is still seen on the next cursor move.
9. **Stats and layout.**
   - The stats are counted once per text.
   - The editor keeps its wrap and classification while the text, the width and the cursor's line stay the same (a
     cursor move used to wrap the whole script again).
   - The status bar takes the line kinds from the editor's layout.

The data race: a window's timers (`after`, background.go) stop when the window closes, and in the tests at cleanup.
Before, the 30 s auto-save timer of an earlier test's window fired during TestBundleFromPlain. Under the test driver,
`fyne.Do` runs on the timer's goroutine. The tests run debounces and background work at once (background_test.go).
The real paths have their own tests in background_real_test.go, which does not build under -race.

Left:
- A keystroke still makes the editor wrap and classify the whole text again: 20 ms for the 160 KB script, 60 ms with
  the novel kept in it. An incremental wrap is the next step for long scripts.
- ~~Saving a bundle still zips on the UI goroutine.~~ Done with PRODUCTION step 3 (bundle_queue.go): every write
  into a bundle (Save, auto-save, the source's Done, Save Draft) is a job run in the background, one after another,
  each from the bundle the one before left; the zip loop calls pace.Yield.
- goldmark's parse and x/net/html's parse cannot yield inside. In wasm they are about 10% of a novel's
  conversion.
- The window benchmarks don't run under node yet: Fyne's wasm driver wants more of a page than the shim gives.
