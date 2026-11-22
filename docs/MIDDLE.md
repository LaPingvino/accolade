# The middle tabs: the script, the cards, the source and any pane side by side

LOOP.md, Joop 2026-11-22: "the new approach to the cards makes the card open in the sidebar at first and push the
script to the sidebar in a very similar view to the outline... Maybe make the cards and script tab like the sidebar
when in the middle? And maybe we can use a similar approach for the source, enable them to be active in the middle
at the same time with this tabbing?" and "I think that comes from a too careful attempt at the swap with middle
task. Being too careful almost broke it more."

The middle of the window is a row of tabs like the dock's (middle.go). The **Script** tab is always first and stays:
the script is the window's document. View > Cards opens the cards' full view as a middle tab, never in the dock
first. Every dock tab's menu has **Open in the Middle** (and View > Open Pane in the Middle, for the pane picked
last): the source and the notes become Markdown documents in an editor of their own there, the cards their full
view, any other pane (Outline, Characters, Production, Help, Changes) its box. Several are open at once, one in
front; a pane is in the dock or in the middle, never both (a Fyne object has one parent). A middle tab's menu has
**Back to the Side** and **Close**; the row's × closes the tab in front; Ctrl+PgDn and Ctrl+PgUp (View > Next and
Previous Middle Tab) go round the tabs, the arrow keys go along the row on a tab. The row is there only while a
tab beside the script's is open, so a window with the script alone looks as before. On a phone the row is one line
(`overflowRow`): the tabs that fit, the one in front always among them, the others in its More menu.

This replaced **Swap with Main** (2026-11-21): one pane at a time traded places with the script, which became a
**Script** list tab in the dock. The swap machinery (`swapWithMain`, `swapBack`, `w.swapped`, `w.mainDoc`,
`swappedCenter`, `swappedIsView`, the dock's Script tab `scriptPane` and its special cases in `boxOf`, `titleOf`,
`focusOf`, `shownNow`, `togglePane`, `hidePane`, `toggleAll`, `cardsVisible`) is gone; what it had built under it,
the document abstraction below, carried over unchanged.

## The model, and its one transition

`w.mid` (`middle`) holds the open tabs after the script's (`tabs`), the one in front (`shown`, "" the script) and
the documents the tabs edit (`docs`). Everything else is derived from it: `w.main()` (the script's document, the
document of the tab in front, or nil for a view), `w.midDoc()`, `w.scriptInMain()`, `w.cardsShown()`,
`paneInSight` (a pane in the middle is in sight when its own box is in front there), `cmdEnv.away`, `midDoc` and
`midTabs`. Each dock pane's row says what its middle tab shows: `doc` (the source, the notes), `full` (the cards'
full view), else its `box`; `midFirst` makes its View command open it in the middle (Cards); `toMiddle`/`toSide`
are what it does going there and back.

Every change of what the middle shows goes through `middle.show`: open (`openMiddle`), select (a tap, the View
command, Next/Previous, `goToScript`, the stripboard), close (`closeMiddle`, `backToSide`), Open/New/a join/a
rename (`leaveDocs` → `closeMiddleDocs`), a document type change (`middlePrune`), the source's block going
(`docsFollowScript`). `show` alone brings along what follows the middle:

- the preview: a preview shown (the script's pages, a document's Live Preview) goes with the tab leaving and the
  tab coming in gets its own (the fix of a30e72d, Joop: the preview "gets stuck when clicking swap"); over a view
  (the cards, the outline) none, and the next editor tab gets its own again; a script's PDF still being made when
  the tab changes is dropped (`previewGen`);
- the editors kept off stage while out of sight (`SetOffstage`), the script's and each document's;
- the link offer when the source's tab is left (`offerLinkUpdate`), the find bar's highlights;
- the menus, the ribbon's contextual tabs and build key, the editing row, the title, the status, the focus.

The next item (the state-change audit, LOOP.md) can hang its `mainDocChanged` on `show`.

## The document abstraction (document.go)

A window edits one **script**: its file, saved by File > Save, shared when editing together. Its editor is
`w.textEditor`, for the window's whole life. Beside it are documents of their own that the main editor can hold
instead: the script's **source** and its **notes**.

| kind   | home (where it is saved)                         | undo history         | collab Doc                                   |
|--------|--------------------------------------------------|----------------------|----------------------------------------------|
| script | the window's file (bundle, plain file)           | `w.textEditor`'s      | the session's, the script's                  |
| notes  | the notes block at the script's end              | its own editor's     | the script's: it is in the script's text     |
| source | `/* Source */` in the script                     | its own editor's     | the script's                                 |
| source | in the bundle, a kept copy, the file beside it   | its own editor's     | its own, named by the setting `source` (collab_source.go) |

`document` holds the kind, its editor (and with it its undo history), its vim state, its view (the editor in its
scroll, with the editor's fonts), where it is kept (`ref`, for a source), and the text as last read from or
written to its home.

- `w.main()` is the document in the main editor: the script, or the source or notes in front in the middle. A view
  (the outline, help, Production, the cards' full view) leaves no document there: nil.
- `w.mainEditor()` is that document's editor (nil for a view). `w.writeEditor()` is what the writer's commands
  act on: the main editor, else the script's. `w.focusMain()` gives the main area the keyboard.
- `w.editorMarkdown()` is the main editor's kind. `w.markdown` stays the **file's** kind: a middle tab never calls
  `setMarkdown`, which sends the type to peers, makes `sourceRef()` return none, hides the Source pane and
  resets the cards and the preview.
- `cmdEnv` gained `away` (the script is not in the main editor) and `midDoc` (a Markdown document of its
  own is). The rules `scriptEditing` and `markdownEditing` decide the commands that write in the main editor.
  `screenplayOnly` and `markdownOnly` stay the file's, so the Adaptation and Production menus, Export and Print
  remain the script's while it is away, and Adaptation is also the way back.
- A document kept **in the script** (the notes, a source inside it) is a stretch of the script's text. Each change
  is written into the script at once, as the Notes pane does. Only what differs is replaced, so a co-writer's edit
  elsewhere in the stretch and the undo steps around it are not touched. A key goes in as its span alone
  (`spanInto`): the document's buffer says what changed since its last write (`Buffer.TakeChange`), and the
  document knows where its text starts in the script while the script is at the revision its own write left. When
  the script changed since (Undo there, a peer's edit), or the text holds what the block writes otherwise (a `*/`
  kept as `* /`, a block's mark), or the change is in its last new lines, the whole stretch is written instead
  (`replaceLeast`, which finds the same span the long way), and the place is read again. Either way the script's
  cursor stays where it was. Nothing is put off: no delayed write that a shared session would have to merge. The writes are one
  undo step of the script's while nothing else wrote there between: the step on top is tagged with the document
  and the next write joins it. A change to the stretch made in the script (Undo there, a peer's edit, a draft
  restored) comes into the document's editor as `Buffer.Remote` edits, outside its own history
  (`followHome`, from `onTextChanged`). If the block itself goes (Undo of Keep Source in the Script), the
  document is let go as it is, not written back.
- A source kept **elsewhere** is written to its home when the script is saved (File > Save, auto-save), when its
  middle tab closes, and on the Source pane's Done; behind another tab it stays open. Until then it is unsaved:
  the title marks it (•), and closing the window asks (`unsavedDocs`).
- `w.sourceText(ref)` returns an open source document's live text, so the pane's passages, drafts, Save Source
  As and carrying the source on Save As all see what is being written. `homeSourceText` reads the home itself.
- The Source pane's Edit/Done edits the same source document (one editor constructor for every kind, with vim,
  the menu's shortcuts and closing brackets: `attachVim`). The pane's old bare `sourceEditor` is gone, and Done
  is the document's `store`.
- Anything that replaces the window's script first calls `leaveDocs`: the main editor gets the script back and
  the documents are written while they still belong to the script they came from. That covers `newDocument`
  (Open, New, imports, a restore), a join (`startSharing`), the document type (`setDocumentType`, `setMarkdown`,
  a peer's type change), and a rename or Save As (`relocate`, whose source moves with the script).

### Named Docs (COLLAB.md, What is left, step 3)

`document.sharedAs()` names the session Doc a document travels in: `"script"` for the script and for what it
holds; for a source kept outside it, the Doc of its own that the shared setting `source` names (`source:<id>`,
COLLAB.md step 3, built). `store` keeps writing the peer's own home; `followShared` takes the source Doc's remote
edits the way `followHome` takes the script's (Remote edits outside the undo history), and `changed` sends the
document's own. The middle tabs needed no change. The status line's "yours only, not shared" is left for a source
the session names no Doc for (one picked before this peer had heard from the others).

## The audit: each place, what it assumed, the fix

| Where | It assumed | Now |
|---|---|---|
| `markdown.go` `setMarkdown`, `w.markdown` | the file's kind is the editor's mode | the file's kind only. The editor's is `editorMarkdown()`. A kind change first puts the script back (`leaveDocs`) |
| `commands.go` element commands (`format.element.*`), Underline, `write.place`, `write.character`, `write.elementKind` | `screenplayOnly` means "the editor holds a screenplay" | `scriptEditing`: hidden while the script is away |
| `commands.go` Markdown commands (`md.*`, `md.headings`) | `markdownOnly` | `markdownEditing`: there for the source or notes in the main editor. Ctrl+1–6 go to the headings then |
| `commands.go` `do`, `tapped` | an editing command acts only while `w.textEditor` has the focus; afterwards the focus goes back to it | the main editor's focus. With a view in the main area, the status says there is nothing to write in |
| emphasis, Right to Left Line, Move Line Up/Down, `mdEdit`, Link…, Image…, Table… | wrote into `w.textEditor` | `writeEditor()`; the focus goes back to the main area |
| `view.preview` (Live Preview), `preview()` | a Markdown file's | the main editor's kind (`byEditor` labels, `editorMarkdown`); the preview shows the main document; back on a screenplay it is hidden again |
| `header_bar.go` editing row | `setMarkdown` chose the row | `showEditingRow(editorMarkdown())` on every change of the tab in front |
| `ribbon.go` Write tab: elements, the format's buttons; the build key | `env.markdown` | `scriptEditing(env)`; `away` and `midDoc` in the key |
| `window.go` `updateStatusBar` | the script's statistics | the main document's: its words, its kind and where it is kept, "yours only, not shared" in a session |
| `window.go` `updateCurrentElement` | the script's cursor | the main editor's (Markdown's names for a document); none for a view |
| `window.go` `updateTitle` | the script's name | the main document first, with its own • when unsaved |
| `window.go` close question | `w.hasChanges` | that, or a document not written (`unsavedDocs`) |
| `window.go` `SaveFile`; `autosave.go` | the script alone | the open documents are written first (`storeDocs`); auto-save is armed by a document's change too |
| `relocate.go` rename, Save As | the source beside or in the bundle moves with the script | the documents are written and let go first (`leaveDocs`), so nothing writes to the old place |
| `collab_window.go` `newDocument`, `startSharing` (join), `applyShared("markdown")` | the window's text is the only state to replace | `leaveDocs` first |
| `collab_window.go` `applyRemote` | the script's editor is where the peers' edits go | still so; documents kept in the script follow it (`followHome`) |
| `doctype.go` `setDocumentType` | the notes block's marks change in the script only | an open notes document is written and let go first |
| `source_file.go` `sourceText` | the home is the truth | an open source document's text first |
| `source_pane.go` Edit/Done, `loadSource` | its own bare editor; a file source is read again only when its time or size changes | the window's source document; while it is open, its text (in memory) |
| `source_pane.go` `toggleSourceInside` | the source has one home while open | an open source document moves with it (`rehome`) |
| `notes.go` `notesPanel.save` | the pane writes the block | `writeNotes`, shared with the notes document; only what differs is replaced |
| `search_bar.go` Find, Replace, Replace All, highlights | `w.textEditor` | `writeEditor()`: Find and Replace work in the source. The highlights are cleared in both editors on a swap |
| `ex_window.go` `:` commands, the yank register | `w.textEditor`, `w.vim` | the main editor and its vim (`mainVim`); `replaceTextIn` for any editor |
| `editor.go` vim, menu shortcuts, brackets, Alt tips | made for the script's editor only | `attachVim` for every editor. A document's vim folds its own headings (`foldsOf`) and does not move through the outline. The keys' reset (Preferences) reaches the documents' vims, and the fonts and line numbers their editors |
| `menus.go` Undo, Redo, Cut/Copy/Paste/Select All fallbacks | `w.textEditor` | `writeEditor()`: the source's own history in the main editor |
| `editor.go` Enter, Tab; `keyflow.go`; `suggest.go`; `complete.go` | the script's editor | still only its own hooks. A document's Enter continues lists; its Tab at a heading's start cycles that heading's fold, else it is the editor's. No Tab hint while the script is away (`showSuggestionHint`) |
| `editor_folds.go` Fold commands | the script's editor and outline | each editor has its folds (`folds`): the script's (`w.scriptFolds`, its outline), a document's (its Markdown headings). View's Fold commands, vim's z keys, Tab at a heading and Shift+Tab act on the main editor's; with a view there the status says there is nothing to fold |
| cursor commands: Omit/Restore Scene, Mark Scene Done, Tag Selection, Untag, Scene Color, Link to Source Passage, Next and Previous Source Passage, Adapt Source Passage…, Mark Source Passage as Cut, Source Headings as Sections | the script's cursor is the writer's | `scriptAway`: the status says to bring the script's tab to the front first (the Cards view keeps its own Color by the card) |
| outline `goTo`, Production `goToLine`, Changes, the source pane's scenes, a card opened | move the script's cursor and focus it | `goToScript`: the script's tab to the front, the cursor there, focused (the other tabs stay open). The Characters pane's cue walk does the same unless it is itself the tab in front |
| `dock.go` focus targets, F6, Down/Enter into a tab, Esc out of one, the pane's menu, Hide, View toggles, Side Panes | each pane's own box and focus | the dock's own again: a pane in the middle is not in the dock. F6 starts at the middle's editor, view or tab (`mainFocus`). The tab widget (`dockTab`) serves both rows through `tabHost` |
| `dock.go` `paneInSight` (the panes ask it before following) | in front of its slot | a pane in the middle is in sight when its own box is the tab in front (the outline, help), not when its document or full view is |
| `palette.go`, `ribbon.go` (Esc, key tips), `ex_window.go`, `markdown_format.go`, full screen | the focus back to `w.textEditor` | `focusMain()` |
| `help_pane.go` cheat sheets and the shortcuts page | the file's kind | the main editor's kind |
| `stripboard.go` | it and the cards take turns in the main area | it puts the script's tab in front first, the other tabs kept |
| `watchdog_js.go` | the script's vim mode | the main editor's |

Kept as they are, and why:

- Export, Print, Statistics, Title Page, Script Language, Production, Adaptation's whole-script commands: these
  act on the script as a document, wherever it is shown.
- Rename and Save As: they are the script's file. Its documents are written and their tabs closed first.
- The drafts store: `sourceText` gives it the live source. Restoring a draft replaces the script's text, and
  the documents kept in it follow.
- Outline and Cards follow the script's text and cursor as before.

## What was built on it

- **The middle tabs** (middle.go), above. Passages have no places in prose's model, so `passageLines`
  (source_passage_lines.go) finds each passage's line by its first words, after the one before it, within
  `passageReach` lines: the Cards pane lights the scenes linking to the passage under the source's cursor, and
  the link offer knows when the cursor leaves a passage.
- **Adaptation > Source in the Middle** (`source.middle`, also on the Adaptation ribbon tab): the source's middle
  tab to the front, opened if it is not; in front already, the script's.
- **The Cards pane** (cards_pane.go), View > Cards (Shift+D), on every screenplay. It is one compact column: each
  scene's and section's heading with its colour, and its synopsis to write. A drag moves the scene (the
  outline's drag row, now `dragList`). A tap goes to the scene. The card of the cursor's scene is lit, or with the
  source in the middle, those of the passage's scenes. In the middle the cards are their full view: View > Cards
  opens it there (not the pane), as do the outline's Cards button (`toggleCards`) and the pane's full-view
  button; the view's Back to the Script brings the script's tab to the front, Back to the Side on its tab makes
  it the pane again.
- **Updating the links** (source_links_offer.go). Say the source's text changed where a passage opens, and the
  writer then leaves that passage in the main editor, presses the Source pane's Done, or saves. Then
  `linkFixes` compares the passages before and after: the same ones before and after the change, and those
  between them in order (`alignPassages`). It finds the script's `[[from:]]` links that found a changed
  passage and no longer find exactly it. A notice above the status bar says how many there are and offers
  **Update Links** or **Not Now**. Update writes each such link in the new form for the passage's new words
  (`fragmentFor`), all in one undo step of the script's. A link that cannot be told apart from another place
  stays as it was. The adaptation block's entries naming a changed passage (cut:, skipped links:) are offered and
  written with the links, by the same rules (`fixedCut`), in the same undo step. Nothing is changed without the
  writer's yes.
- The header's editing row for a document in the main editor is Markdown's without Start a Screenplay
  (`docRow`): the window has a screenplay already.

Not done, and why:

- The dock's Script tab went with the swap. Its caption of the passage under the source's cursor ("1 scene from
  it", "not adapted yet", "cut on purpose") and its tap from a scene to the scene's passage are not anywhere now;
  the Cards pane still lights the passage's scenes. A place for them (the Cards pane's header, the outline) is
  open.
- The stripboard is not a middle tab: it takes the middle's place as before, the script's tab put in front
  first.
- Passages are found by their opening words. A passage whose opening is not on its first line (a table) is
  not found, and the cursor in it counts as being in the passage before.

## What a key costs (BenchmarkSwappedSourceKey, the PERF-SOURCE novel; measured with Swap with Main)

Typing in the source kept in the script cost 175–213 ms a key (200 keys, 3 runs): each key wrapped and coloured
the whole script with the novel in it for an editor not on screen (≈42 ms), counted the novel's words four times
(≈40 ms) and classified its lines four times for the status bar's element (≈17 ms), and wrote the whole block back
the long way (≈9 ms). Now:

- the script's editor out of sight is laid out only when asked or back in sight (`ScriptEditor.SetOffstage`);
- the status bar counts a text once (`w.markdownStats`) and takes the line's kind from the editor's own layout
  (`MarkdownKinds`); the script's handlers leave the status to the document that wrote (`docWriting`);
- the key goes into the script as its span (`spanInto`), the script's cursor kept;
- (then) the Script tab read the scenes again after a pause; it is gone with the swap;
- a joined undo step grows in place (`Buffer.JoinSince`), so a long run of keys stays linear.

After (same 200 keys, 3 runs): 63–75 ms, the script's own key 54–56 ms (before 53–57). What is left is the
document's own layout (≈29 ms, as the script's), its word count (≈10 ms), the script's text made as a string for
its change handlers (≈10 ms, as the script's own key) and the fold marks' outline (≈5 ms). No growth over a long
run (55 ms a key over 2000). Beside the script 47–51 ms (before 61–68); the notes with the novel in them 61–71 ms;
what waits for the pause (the Script tab) ≈3 ms.

## Tests (middle_test.go, middle_typing_test.go, middle_preview_test.go)

- `TestMiddleSource`: the source's tab, not also in the dock; Markdown's commands and not the elements; a heading
  through `do`; the source's own Undo; Find and Replace All in the source and not the script; nothing written
  before Save; the title's • and the close question; Save writes it; the script's tab in front brings the
  screenplay's commands back with the source kept open; Back to the Side writes it and puts the pane back.
- `TestMiddleSourceInsideScript`: three edits are one script undo step; a change to the block made in the script
  comes into the source outside its Undo; the script's Undo takes the source's writing back as one step.
- `TestMiddleNotes`: Fountain's notes block, the Notes pane following, and a Markdown document's HTML comment.
  Markdown has no source or cards for the middle.
- `TestMiddleBlocked` (the menu item greyed), `TestMiddleLeftForAnotherDocument` (Open closes the document tabs,
  keeps the outline's; the document type closes the cards'), `TestMiddleAutoSave` (the source behind the
  script's tab).
- `TestMiddleView`: the outline in the middle, nothing to write in, a cursor command refused with its reason, a
  scene tapped bringing the script's tab to the front, the View command showing the tab and closing it.
- `TestMiddlePreviewFollowsShownTab`: from each editor tab with its preview (the script's pages, the source's and
  the notes' Live Preview) to every other kind (the script, the source, the notes, the cards, the outline): the
  preview shown is of the tab in front, none over a view, and the next editor tab has its own again; none comes
  when none was shown.
- `TestMiddleCommands`: Source in the Middle, Open Pane in the Middle, Next and Previous Middle Tab (the commands
  and Ctrl+PgDn/PgUp in the script's editor), Back to the Side, the strip's arrow keys and Down.
- `TestMiddlePhone`: six tabs on a phone's row, the one in front on it, the rest in More, the row within the
  width; the source's editor scrolls to its cursor; the tab's menu (Back to the Side, Close; none for the script).
- `TestCardsPane`: View > Cards opens the middle tab, not the pane; the pane's cards, the cursor's card lit, a
  synopsis written, a drag, the passage's cards lit with the source in the middle; the full view, a card opened
  from it bringing the script's tab forward, the View command closing it.
- `TestMiddleTogether`, `TestMiddleNotesTogether`, `TestMiddleTypingTogether`: two windows over the loopback.
- `TestCharactersInTheMiddle`, `TestAdaptScriptAway` (the → to a scene brings the script's tab forward and
  flashes it), `TestKeepSourceInsideWhileOpen`, `TestMiddleRenameAndJoin`, `TestMiddleExVimPreview`,
  `TestMiddleFolds`, `TestMiddleSpanWrite`, `TestLinkFixes`, `TestLinkOffer` (also on leaving the source's tab),
  `TestLinkFixesCuts`, `TestLinkOfferCutOnly`, `TestPassageLines`.
- `TestMiddleScreenshots` (SHOT_DIR): desktop and phone: the script alone, View > Cards, Script + Cards + Source
  with each in front, the link offer, the notes, the outline in the middle, the Cards pane back to the side.
- `TestTakeChange` (internal/editor/buffer), `TestOffstage` (internal/editor).
