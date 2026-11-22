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
| `go test -p 1 ./...` (root again, ~190 s, + subpackages 22 s) | ~212 s |
| `go test -run 'Focus(Is)?Visible$' -count=1 .` | 126 s |
| `go vet -race .` | 4 s |
| wasm build | 11 s |
| no-pandoc goldens (`TestRibbon\|TestCommandsGolden`) | 18 s |
| **total** | **~9 ½ min** |

Focus tests (s): TestFocusIsVisible 35.9, TestSaveAsFocusVisible 18.1,
TestCharactersFocusVisible 14.2, TestPageViewFocusVisible 12.5,
TestAdaptFocusVisible 10.6, TestProductionFocusVisible 7.5,
TestDockFocusVisible 6.3, TestStripboardFocusVisible 6.2,
TestRibbonFocusVisible 3.8 — two full software-rendered captures per
focusable widget, most of it Fyne's shadow blur (bild convolution).

Subpackages (s): collab 5.6, prose 3.5, the rest under 0.4 each.
