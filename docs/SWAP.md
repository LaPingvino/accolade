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
| source | in the bundle, a kept copy, the file beside it   | its own editor's     | none yet: `sharedAs() == ""` (named Docs, 3) |

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
  is written into the script at once, as the Notes pane does. Only what differs is replaced (`replaceLeast`), so
  a co-writer's edit elsewhere in the stretch and the undo steps around it are not touched. The writes are one
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
holds, `""` for a source kept outside it. Step 3 gives the source a Doc of its own named `"source"`. Then
`sharedAs` returns that name, `store` keeps writing the peer's own home, and `followHome` takes the source Doc's
remote edits the way it takes the script's now (Remote edits outside the undo history). The swap code then needs
no change. Until then, Swap with Main works in a shared session for every pane. A source kept outside the script
is this writer's alone, and the status line says so ("yours only, not shared").

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
| `editor.go` vim, menu shortcuts, brackets, Alt tips | made for the script's editor only | `attachVim` for every editor. A document's vim does not fold or move through the outline. The keys' reset (Preferences) reaches the documents' vims, and the fonts and line numbers their editors |
| `menus.go` Undo, Redo, Cut/Copy/Paste/Select All fallbacks | `w.textEditor` | `writeEditor()`: the source's own history in the main editor |
| `editor.go` Enter, Tab; `keyflow.go`; `suggest.go`; `complete.go` | the script's editor | still only its own hooks. A document's Enter continues lists, and its Tab is the editor's. No Tab hint while the script is away (`showSuggestionHint`) |
| `editor_folds.go` Fold commands | the script is in sight | the status says why not while it is away |
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
