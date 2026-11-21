# The main editor and the dock panes made equivalent

LOOP.md, Joop 2026-11-21: "make it all equivalent? you have move to top and move to bottom … a swap with main
added to that." Every dock tab's menu has **Swap with Main**: that pane's document or view goes into the main
area, and the script takes the pane's place as a **Script** tab. Swapping again puts them back.

Before that could be built, everything that took "the main editor" to mean "the script" had to ask which
document the main editor holds. This file is that audit, the small document abstraction it led to, and how the
abstraction leaves room for named Docs (COLLAB.md, What is left, step 3).

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

- `w.main()` is the document in the main editor: the script, or the source or notes swapped in. A view swapped in
  (the outline, help, Production, the cards' full view) leaves no document there: nil.
- `w.mainEditor()` is that document's editor (nil for a view). `w.writeEditor()` is what the writer's commands
  act on: the main editor, else the script's. `w.focusMain()` gives the main area the keyboard.
- `w.editorMarkdown()` is the main editor's kind. `w.markdown` stays the **file's** kind: a swap never calls
  `setMarkdown`, which sends the type to peers, makes `sourceRef()` return none, hides the Source pane and
  resets the cards and the preview.
- `cmdEnv` gained `away` (the script is not in the main editor) and `swappedDoc` (a Markdown document of its
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
- A source kept **elsewhere** is written to its home when the script is saved (File > Save, auto-save), when it
  leaves the main editor, and on the Source pane's Done. Until then it is unsaved: the title marks it (•), and
  closing the window asks (`unsavedDocs`).
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
document's own. The swap code needed no change. The status line's "yours only, not shared" is left for a source
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
| `header_bar.go` editing row | `setMarkdown` chose the row | `showEditingRow(editorMarkdown())` on every swap |
| `ribbon.go` Write tab: elements, the format's buttons; the build key | `env.markdown` | `scriptEditing(env)`; `away` and `swappedDoc` in the key |
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
| cursor commands: Omit/Restore Scene, Mark Scene Done, Tag Selection, Untag, Scene Color, Link to Source Passage, Next Source Passage, Mark Source Passage as Cut, Source Headings as Sections | the script's cursor is the writer's | `scriptAway`: the status says to swap back first (the Cards view keeps its own Color by the card) |
| outline `goTo`, Production `goToLine`, Changes, the Script tab | move the script's cursor and focus it | `goToScript`: focused if the script is in the main editor, else the Script tab lights the scene |
| `dock.go` focus targets, F6, Down/Enter into a tab, Esc out of one, the pane's menu, Hide, View toggles, Side Panes | each pane's own box and focus | `boxOf`, `focusOf`, `titleOf`, `shownNow`: the Script tab in the swapped pane's place. Hiding it, or the pane's View command, puts the swap back |
| `dock.go` `paneInSight` (the panes ask it before following) | in front of its slot | a swapped pane is in sight only if its own box is in the main area (the outline, help), not when its document or full view is |
| `palette.go`, `ribbon.go` (Esc, key tips), `ex_window.go`, `markdown_format.go`, full screen | the focus back to `w.textEditor` | `focusMain()` |
| `help_pane.go` cheat sheets and the shortcuts page | the file's kind | the main editor's kind |
| `stripboard.go` | it and the cards take turns in the main area | it puts a swap back first |
| `watchdog_js.go` | the script's vim mode | the main editor's |

Kept as they are, and why:

- Export, Print, Statistics, Title Page, Script Language, Production, Adaptation's whole-script commands: these
  act on the script as a document, wherever it is shown.
- Rename and Save As: they are the script's file. Its documents are written and the swap is put back first.
- The drafts store: `sourceText` gives it the live source. Restoring a draft replaces the script's text, and
  the documents kept in it follow.
- Outline and Cards follow the script's text and cursor as before. The Script tab follows the main cursor in
  Swap mode.

## What was built on it

- **Swap with Main** (swap.go) on every dock tab's menu (right-click, long press, Shift+F10), beside Move to
  Top/Bottom, and as View > Swap Pane with Main (`view.paneSwap`, the pane picked last or with the keyboard).
  The source and the notes become Markdown documents in the main editor. The cards become their full view. Any
  other pane (Outline, Production, Help, Changes) puts its box in the main area. The script is then the
  **Script** tab in the pane's place: one row a scene or section. With the source in the main editor it follows
  that cursor the other way round. The scenes linking to the passage under the cursor are lit, the passage's
  status is said above them ("1 scene from it", "not adapted yet", "cut on purpose"), and a tap puts the source's
  cursor on the scene's passage and the script's cursor on the scene. With anything else it lights the script
  cursor's scene. The swap is put back by swapping again (the tab's menu says "put back"), by the pane's View
  command, by hiding the Script tab, or by Side Panes. One pane at a time: swapping another puts the first
  back first. Passages have no places in prose's model, so `passageLines` finds each passage's line by its
  first words, after the one before it, within `passageReach` lines.
- **Adaptation > Swap Source and Script** (`source.swap`, also on the Adaptation ribbon tab): the Source pane is
  brought in if it is not there, then swapped, or put back.
- **The Cards pane** (cards_pane.go), View > Cards (Shift+D), on every screenplay. It is one compact column: each
  scene's and section's heading with its colour, and its synopsis to write. A drag moves the scene (the
  outline's drag row, now `dragList`). A tap goes to the scene. The card of the cursor's scene is lit, or with the
  source in the main editor, those of the passage's scenes. Swapped with the main area it is the full Cards
  view, which the outline's Cards button and the view's Back to the Script reach (`toggleCards`).
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

- The Script tab is a list, not an editor of the script: writing goes back through the swap.
- Passages are found by their opening words. A passage whose opening is not on its first line (a table) is
  not found, and the cursor in it counts as being in the passage before.

## What a key costs (BenchmarkSwappedSourceKey, the PERF-SOURCE novel)

Typing in the source kept in the script cost 175–213 ms a key (200 keys, 3 runs): each key wrapped and coloured
the whole script with the novel in it for an editor not on screen (≈42 ms), counted the novel's words four times
(≈40 ms) and classified its lines four times for the status bar's element (≈17 ms), and wrote the whole block back
the long way (≈9 ms). Now:

- the script's editor out of sight is laid out only when asked or back in sight (`ScriptEditor.SetOffstage`);
- the status bar counts a text once (`w.markdownStats`) and takes the line's kind from the editor's own layout
  (`MarkdownKinds`); the script's handlers leave the status to the document that wrote (`docWriting`);
- the key goes into the script as its span (`spanInto`), the script's cursor kept;
- the Script tab reads the scenes again after a pause (`scriptTabTimer`), a swapped-out pane when it is back;
- a joined undo step grows in place (`Buffer.JoinSince`), so a long run of keys stays linear.

After (same 200 keys, 3 runs): 63–75 ms, the script's own key 54–56 ms (before 53–57). What is left is the
document's own layout (≈29 ms, as the script's), its word count (≈10 ms), the script's text made as a string for
its change handlers (≈10 ms, as the script's own key) and the fold marks' outline (≈5 ms). No growth over a long
run (55 ms a key over 2000). Beside the script 47–51 ms (before 61–68); the notes with the novel in them 61–71 ms;
what waits for the pause (the Script tab) ≈3 ms.

## Tests (swap_test.go)

- `TestSwapSourceIntoMain`: Markdown's commands and not the elements; a heading through `do`; the source's own
  Undo; Find and Replace All in the source and not the script; nothing written before Save; the title's • and
  the close question; Save writes it; swapping back writes again and brings the screenplay's commands back.
- `TestSwapSourceInsideScript`: three edits are one script undo step; a change to the block made in the script
  comes into the source outside its Undo; the script's Undo takes the source's writing back as one step.
- `TestSwapNotes`: Fountain's notes block, the Notes pane following, and a Markdown document's HTML comment.
  Markdown has no source or cards to swap.
- `TestSwapBlocked`, `TestSwapLeftForAnotherDocument` (Open and the document type), `TestSwapAutoSave`.
- `TestSwapView`: the outline in the main area, nothing to write in, the Script tab lighting the outline's
  scene, a cursor command refused with its reason, the View command putting it back.
- `TestScriptTabFollowsSource`, `TestPassageLines`: the passage under the cursor, its linked scenes lit, its
  status, a tap to the scene's passage.
- `TestSwapTogether`: two windows over the loopback; the source kept in the shared script, typed in the main
  editor on one side and edited in the script on the other, goes both ways.
- `TestSwapPhone`: the sheet's Script tab and its menu.
- `TestSwapSourceAndScriptCommand`: Adaptation's command and View > Swap Pane with Main.
- `TestCardsPane`: the cards, the cursor's card lit, a synopsis written, a drag, the passage's cards lit, the
  full view swapped in and a card opened from it.
- `TestLinkFixes`, `TestLinkOffer`: the edits for an opening changed (none when nothing changed, a split
  passage); the notice when the cursor leaves the passage, Update Links as one undo step, Not Now, the pane's
  Done.
- `TestSwapScreenshots` (SHOT_DIR): desktop and phone, before, the source with the Script tab, the link offer,
  the notes, the Cards pane, the full Cards view, the outline in the main area.
- `TestSwapSpanWrite` (swap_typing_test.go): the source inside the script and the notes, 400 random edits each
  (inserts at the start, the end and anywhere, with `*/`, `* /`, `/*`, `-->`, new lines and wide letters;
  deletions; typing; Undo; Redo; the notes emptied and typed again): after each the script is what writing the
  whole stretch makes, the home as seen is the home, the script's cursor has not moved.
- `TestSwapTypingTogether`: two windows, real timers; a key at a time typed in the swapped source, the co-writer's
  edit landing between the keys (before the block, in it before and after the cursor, at the script's start);
  both scripts alike at the end, every key and edit there, the cursor after the keys.
- `TestSwapFolds`: Fold Here, Unfold All, Tab at a heading, vim's zR in the source swapped in; the script's folds
  untouched; a view in the main area has nothing to fold.
- `TestLinkFixesCuts`: a passage's opening edited, its cut: entry written for the new words with the link, the
  other entry left.
- `TestTakeChange` (internal/editor/buffer), `TestOffstage` (internal/editor).
