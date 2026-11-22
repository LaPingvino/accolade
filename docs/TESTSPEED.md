# Test speed

Why the local gate takes as long as it does, and what was done about it
(LOOP.md, 2026-11-22: "55 minutes Accolade gates? … I think we need some
cleanup"). Measured on the VPS (8 cores, load 3–6 from other work: ±15 %
between runs), `go test -count=1 -json -skip 'Focus(Is)?Visible$' .`.

## Before (main at 098bff3)

Root package: 669 tests, 187 s (serial: the tests share Fyne's current app
and the process's settings, so no t.Parallel). The distribution is flat —
median 0.17 s, the 19 tests over 1 s only 50 s of it — so the cost is the
fixture, not a few slow tests: every window a test builds cost ~0.15 s, and
the CPU profile put `NewMainWindow` at 92 of the 158 s the tests ran,
65 s of it in `setColorScheme` (Fyne re-theming the whole window tree that
had just been built).

The 40 slowest (seconds):

```
   8.91 TestSampleFresh 
   5.50 TestFootnoteReflowInPreview 
   5.05 TestSwapSpanWrite 
   3.82 TestRibbonFitsDesktop 
   3.33 TestMarkdownCommands 
   2.58 TestPreviewFollowsATypedKey 
   2.46 TestTailBlocksAsBefore 
   2.36 TestBundleFromPlain 
   1.96 TestOpenScannedPDF 
   1.86 TestKeyFlowTransitions 
   1.82 TestBundleZipRoundTrip 
   1.74 TestRenameTogether 
   1.42 TestExportKindle 
   1.27 TestLearnNotFromAnotherScenesLinks 
   1.25 TestWithPandoc 
   1.24 TestRibbonKeyTips 
   1.19 TestRibbonSettings 
   1.09 TestLearnPlace 
   1.00 TestBundleLanguage 
   0.95 TestScheduleAnchors 
   0.95 TestFlattenTwice 
   0.93 TestStripboardDrag 
   0.92 TestNewDocumentLeavesSharing 
   0.92 TestHelpTab 
   0.91 TestTypingDuringCatchUp 
   0.89 TestScheduleOrder 
   0.88 TestPreviewTab 
   0.87 TestComboPartial 
   0.87 TestBundleSaveAsAnother 
   0.84 TestSwapTypingTogether 
   0.82 TestSourceJoinMidEdit 
   0.79 TestSwapNotesTogether 
   0.79 TestPreviewReflowsMovedBreaks 
   0.77 TestHelpLinksInApp 
   0.77 TestCarriedConfRoundTrip 
   0.76 TestLearnTypedCue 
   0.75 TestPreviewTypingCancelsTheReflow 
   0.74 TestCharactersPane 
   0.73 TestScheduleGroup 
   0.73 TestPreviewFreshPDF 
```

The rest of the gate, before:

| step | time |
|---|---|
| `go test -count=1 -skip 'Focus(Is)?Visible$' .` | 191 s |
| `go test -p 1 ./...`: root again, *with* the focus tests (no `-skip`), + subpackages 22 s | ~330 s (187 + 123 + 22, estimated; 232 s measured after the first fixes) |
| `go test -run 'Focus(Is)?Visible$' -count=1 .` | 126 s |
| `go vet -race .` | 4 s |
| wasm build | 11 s |
| no-pandoc goldens (`TestRibbon\|TestCommandsGolden`) | 18 s |
| **total** | **~11 min** |

Focus tests (s): TestFocusIsVisible 35.9, TestSaveAsFocusVisible 18.1,
TestCharactersFocusVisible 14.2, TestPageViewFocusVisible 12.5,
TestAdaptFocusVisible 10.6, TestProductionFocusVisible 7.5,
TestDockFocusVisible 6.3, TestStripboardFocusVisible 6.2,
TestRibbonFocusVisible 3.8 — two full software-rendered captures per
focusable widget, most of it Fyne's shadow blur (bild convolution).

Subpackages (s): collab 5.6, prose 3.5, the rest under 0.4 each.

## What was done (branch testspeed)

Every test keeps checking what it checked; nothing moved behind
`testing.Short()` (CI runs without `-short`, but the slow remainders are
the OCR and Calibre tests, which CI's runners skip anyway: locally they
are the only place they run).

| change | tests it speeds | why coverage holds |
|---|---|---|
| **The theme is set before a window is built**, and only when the scheme changes (`NewMainWindow`, `Application.setColorScheme`). Production code: a second window no longer re-themes the first either. | every test that builds a window (~470) | The same theme ends up applied; built under it instead of re-themed after. New `TestNewWindowTakesTheSavedTheme`; the focus tests and the no-pandoc goldens pass unchanged. |
| **Format files cached** (`presetsIn`, script_format.go): decoded again only when a `.toml` file comes, goes or changes, keyed by name and content (each test's config folder has the same built-in formats). Production code. | all (each header/ribbon/pane refresh read and decoded every format file) | Same list as `rules.PresetsIn`; callers get a copy. New `TestPresetsInFollowsTheFiles` (new/changed/removed file, another folder, copies, goroutines; passes under `-race`). |
| **The link offer** re-lays the window out only when it comes up or its words change (source_links_offer.go). Production code: before, every key in a swapped-in source refreshed the whole window. | TestSwapSpanWrite 5.0 → 2.5 s (3.2 under `-json`) | The same text shown, the same refresh when it changes. |
| `heldDebounce` (background_test.go): a burst of typing with the debounced work held, each slot's last function run once at the end, shortest pause first — what the app does after a pause. `makeSample` types its new scene with it. | TestSampleFresh 8.9 → 3.6 s | The sample made is the same file but for dates and random ids (diffed: `UPDATE_SAMPLE=1` from main vs this branch, unzipped); `checkSample` checks the same features. |
| TestFootnoteReflowInPreview finds the paragraph count that moves the reference by bisection (memoised), not a linear scan rendering most counts twice. | 5.5 → 1.2 s | The same n (the page grows with n), then the same two checks as before (page 1 at n, page 2 at n+1). |
| TestTailBlocksAsBefore: the old regexp (the oracle) compiled once per marks, not once per text. | 2.5 → 0.13 s | The same 20 000 texts, the same comparisons. |
| TestPreviewFollowsATypedKey: the fresh whole-script layout that only timed a log line runs only with `-v`. | −0.7 s without `-v` | Its comparison drawing after the key is made as before. |

Not done, with the reason:
- **GOGC**: 300 gave ~10 % on a subset but nothing on the full run (86 vs 87 s at GOGC 200/100), for 1.5–2 GB of heap instead of 0.9.
- **t.Parallel**: ruled out by `t.Setenv`, the process's settings and `fyne.SetCurrentApp`.
- **Moving parts out of the root package** (step 3 of the item): the cost is building windows, spread over ~470 tests (median 0.17 s, 19 tests over 1 s), and the gate runs packages with `-p 1`; a page-view or reports package would still build its windows. Nothing to gain that justifies the churn.
- **The focus tests' captures** (below).

## After (6eae960)

Root package, the exact command (`-json`, which also means `-v`), three
runs back to back at load 2.4–3.6: **93.2, 96.0, 95.2 s** (from 187 s);
without `-json`, in the gate: 93 s (earlier runs 85–88 s at lower
load). So: halved, at the 90 s line rather than safely under it.

The 40 slowest after (seconds, run 2):

```
   3.61 TestSampleFresh 
   3.57 TestRibbonFitsDesktop 
   3.19 TestSwapSpanWrite 
   2.31 TestPreviewFollowsATypedKey 
   1.71 TestOpenScannedPDF 
   1.71 TestKeyFlowTransitions 
   1.47 TestExportKindle 
   1.24 TestRenameTogether 
   1.23 TestFootnoteReflowInPreview 
   1.20 TestRibbonKeyTips 
   1.13 TestMarkdownCommands 
   0.92 TestScheduleOrder 
   0.89 TestHelpTab 
   0.86 TestWithPandoc 
   0.82 TestScheduleAnchors 
   0.81 TestRibbonSettings 
   0.81 TestPreviewTab 
   0.77 TestLearnNotFromAnotherScenesLinks 
   0.75 TestIssueLocksPages 
   0.73 TestScheduleReports 
   0.69 TestStripboardDrag 
   0.67 TestLearnPlace 
   0.67 TestFollowEditsOffAndRefresh 
   0.66 TestPreviewReflowsMovedBreaks 
   0.66 TestHelpLinksInApp 
   0.63 TestScheduleRevisionFlag 
   0.63 TestQuickPhoneTwoRows 
   0.60 TestScheduleFollowsAfterPause 
   0.59 TestProductionFollowsAfterPause 
   0.58 TestSwapTypingTogether 
   0.58 TestQuickNarrowKeepsPreview 
   0.57 TestPreviewTypingCancelsTheReflow 
   0.57 TestMarkdownLivePreviewPages 
   0.56 TestCharactersPane 
   0.56 TestBrowserPDF fail
   0.55 TestRenumberDropsOmittedKeys 
   0.55 TestPreviewFreshPDF 
   0.55 TestBreakdownFollowsAfterPause 
   0.54 TestNewWindowTakesTheSavedTheme 
   0.53 TestStripboardPoolCastKeys 
```

The gate, before and after (each step's exit code checked; the only
failures are TestBrowserPDF and TestBrowserPDFMarkdown, which fail only
inside the sandbox — no Chromium there):

| step | before | after |
|---|---|---|
| root, `-skip` focus | 191 s | 97 s |
| `go test -p 1 ./...` (root again with the focus tests, + subpackages) | ~330 s | 254 s |
| focus tests | 126 s | 123 s |
| `go vet -race .` | 4 s | 4 s |
| wasm build | 11 s | 12 s |
| no-pandoc goldens | 18 s | 16 s |
| **total** | **~11 min** | **~8 ½ min** |

### Proposed: step 2 without the root package

Step 2 runs the root package a second time, focus tests included —
which CI keeps apart because they have failed only inside full runs.
Steps 1 and 3 already run every root test, so step 2 as
`go test -p 1 ./docs/... ./internal/...` (22 s uncached) loses nothing:

| gate with that step 2 | 97 + 22 + 123 + 4 + 12 + 16 = **~4 ½ min** |
|---|---|

Not changed in LOOP.md's Rules: it is the coordinator's gate.

### What still stands between that and 4 minutes: Fyne

Three findings, all in the Fyne fork (LaPingvino/fyne, pinned in go.mod;
not changed here):

1. **The software painter blurs every shadow on every capture**
   (`internal/painter/software/draw.go` drawShadow → bild Gaussian). The
   focus tests make 439 focusables × 2 full captures (≈130 ms each), and
   the blur is ~60 % of their CPU. A cache of blurred shadow images by
   size, radius and colour would roughly halve the focus step (~60 s).
   Reusing one unfocused capture per canvas in `invisibleFocus` would
   halve it too, but would let a side effect of focusing pass as a
   visible focus: left as it is.
2. **Test windows never release their widgets.** A bare `widget.Entry`
   in a closed test window is never collected (probe with finalizers): the
   renderer/canvas caches keep every object until expiry plus a
   `Clean(true)`, and the test window's `Close` does not call
   `cache.CleanCanvas` as the GLFW and mobile windows do. The root run's
   live heap climbs steadily to ~500 MB; GC marking is about half of
   the run's CPU (on other cores, so less of its wall time). A
   `cache.CleanCanvas` in the test window's `Close` (and the fixtures
   closing their windows) would bound it — also CI's race job's ~1.5 GB.
3. **`test.NewApp` clears the font cache** (the fork's
   "ClearFontCache lets old fonts go"), so every test parses its fonts
   again: `font.ParseTTF` ~5–12 s CPU over the run.

Also seen, in Lexington (changes go there first): `pdf.AddFont` parses
the fonts again for every report and measure (`charactersPane.read` →
`scriptReport` → `PDFWriter.Measure`), ~3–5 s over the run.
