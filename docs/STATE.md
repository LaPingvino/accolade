# State changes: the views derived from the document in front

LOOP.md, 2026-11-22 (Joop: the preview "gets stuck when clicking swap" — "check for similar state change issues and
ways to prevent similar bugs"); 2026-11-23 (Joop: "Outline might also need to refresh on context? (depending on Source
or Screenplay active in the middle?)").

Swap with Main is gone: the middle is a row of tabs (middle.go, docs/MIDDLE.md) and every change of the tab in front
goes through `middle.show`. Beside the tab in front, many views are *derived* from a document: they show something
of it and keep it until told otherwise. A view goes stale when what is in front changes and nobody tells it.

## The rule, and where it is kept (state.go)

- **What a view describes.** The **front document** is the document in front in the middle; with a view in front
  there (the cards' full view, the outline as a middle tab), the script (`frontDoc`). Most views describe the front
  document; the **script's views** (the cards, the characters, the source's links and Adapt…) describe the script
  whatever is in front.
- **Each view records what it shows**, as a `docKey`: the document and the script's **epoch**. The script's document
  is the same object for the window's life, so Open, New, a join or a type change would otherwise look like no
  change at all; the epoch moves on at the end of each of those.
- **One hook.** `mainDocChanged` goes through the views (`docViews`, one table) and brings along each one in sight
  whose record is not what it should describe. It runs at the end of `middle.show`, and at the end of every
  transition that replaces or changes the script, which is bracketed by `defer w.scriptChange()()` (nested brackets
  are one; the epoch moves on and the views follow when the outermost ends, so a view shown midway — of a script
  half read — is followed again).
- **One invariant test.** `TestViewsFollowTheFront` (state_test.go) drives every transition below, from the script
  in front and from the source in front, with the outline, the characters, the find bar (searching "the") and the
  preview shown, and checks after each: every view in sight has the record it should, and what it shows is of the
  front document as it is now (the outline's items, the find bar's count against the highlights of the editor in
  front and none elsewhere, the Live Preview's text, no script pages over another tab, the title naming the
  document in front). It fails when the outline is made to describe the script always.

The transitions the test drives: show the source / the notes / the cards / the script, the next tab, an edit in the
document in front, Preview, the cards then Preview, the outline to the middle, the outline there and back to the side
(Open in the Middle / Back to the Side), closing the source's and the notes' tab, closing a pane, Open, New, Open
Sample, New Markdown Document, the document type to Markdown and back, a rename, sharing, joining, sharing and
leaving, Production, Writer, Choose Source… with the Source tab open: 27 transitions × 2 fronts = 54 runs.

## The views

| view | describes | records | follows by |
|---|---|---|---|
| page view (the script's pages) | script, only with it in front | `pagesWant` at the request, `pagesComing` while one is made | made again (the pages there until then); hidden over another tab, and the Live Preview instead when the script became Markdown |
| Follow Edits' target | the pages' script text | with the pages (newPageFollow) | goes with the pages (stopped by hidePages) |
| Live Preview (and its Page Setup line) | front document | `previewPane.of` (`updateLivePreview`) | shown again of the front document; closed, and the pages instead, when that is a screenplay |
| find bar: count and highlights | front document | `searchBar.of`, `lit` the editor highlighted | matches found again; the highlights leave the editor they were in |
| outline | front document | `outline.of`, `outline.doc` | read again (`refresh`): the script's scenes, the source's chapters and passages, the notes' headings |
| Cards pane (lit scenes) | script; lit: the source's passage when it is in front | `cardsDock.of` | read again (`update`, which follows the cursor) |
| cards' full view | script | `cards.of` | read again |
| Characters | script | `charsPane.of` | read again |
| Source pane: lights, info, anchor caption | script (its links) | `source.of` (in `follow`) | `follow` |
| Adapt… (open) | script, as planned at its opening | `adapting.of` | closed, with a status line (only a peer or a background read can change the script under the modal dialog) |
| status bar (element, counts, sharing, peers) | front document | `statusOf` | `updateCurrentElement`, `updateStatusBar` |
| title (•, the document in front, the file) | front document | `titleOf` | `updateTitle` |
| menus, ribbon contextual tabs, editing row, cheat sheets | front document's kind | `commandsOf` (`editorKindChanged`) | `editorKindChanged` |

Not views with state of their own, so nothing to go stale: **Undo** (Edit > Undo acts on the focused editor, else
the main one: `editTarget`, at each call; `focusMain` moves the keyboard with the tab), **collab presence** (the peer
count is the status bar's, set by the network's callback through `uiDo` and by the status view; the editor's other
cursors are helix's own selections, not peers'), the **Notes pane** (a pane is in the dock or the middle, never both;
it reads the script's notes block on each edit).

## Each pair that could go stale, and its verdict

"Was" is before this audit (after the middle tabs); "now" is with the hook.

| # | view | transition | was | now |
|---|---|---|---|---|
| 1 | outline | Source to the front | the script's scenes beside the source | **fixed**: the source's chapters and passages; a click goes to the passage in the source's editor |
| 2 | outline | Notes to the front | the script's scenes | **fixed**: the notes' headings |
| 3 | outline | back to the script, a view (cards) to the front | — | the script's again (`frontDoc`) |
| 4 | outline | typing in the source / the notes | not refreshed (only the script's edits reached it) | **fixed**: `mainChanged` refreshes it after a pause (a source can be a novel; the script's stays at once) |
| 5 | outline | Choose Source… with the Source tab open | the old source (see 23) | **fixed** |
| 6 | outline rows | drag, colour, Cards button with the source in front | would have moved and coloured script scenes by the source's rows | **fixed**: script only (the drag says so, the colour menu and the Cards button are not there) |
| 7 | find bar | a view (cards, the outline) to the front | highlights cleared in the script, the count left ("3 of 11" with nothing lit) | **fixed**: found again in the editor the commands act on |
| 8 | find bar | another editor tab to the front | the old editor's highlights cleared, the count of the old document kept until the next key | **fixed**: matches found again in the document now in front |
| 9 | page view | Open, New, Open Sample, an import, a join, a rename, the role, with the **source** in front | the tab change put the script in front *before* the new text was there: the pages were made of the **old script** (with Follow Edits off they stayed so) | **fixed**: the request in flight is the view's (`pagesComing`): at the transition's end it is made again of the script as it is then |
| 10 | page view | sharing then leaving at once | the second change came while the first pages were being made, hidden: those (of before) were shown | **fixed** (same: a Preview being made counts as shown) |
| 11 | page view | Open/New/a rename with the script in front, Follow Edits off | the old script's pages until Refresh | **fixed**: made again at the transition's end |
| 12 | page view | document type to Markdown with the pages shown | a screenplay's pages of a Markdown document | **fixed**: the Live Preview instead |
| 13 | page view | Preview with a view tab in front | nothing visible (pages made behind the cards) | **fixed**: the script's tab comes forward with its preview |
| 14 | page view | another tab to the front, a PDF still being made | (fixed with the middle tabs: `previewGen`) | checked |
| 15 | Live Preview | the tab in front changes | (fixed with the middle tabs: a30e72d, Joop's report) | checked by the record and the text |
| 16 | Live Preview | document type to screenplay | closed by setMarkdown | the hook closes it too when it would be of a screenplay |
| 17 | status bar, title | every transition | each transition called them itself, some not at its end | the hook at the end of each; the calls at the end of Open, New, the imports and setMarkdown are gone |
| 18 | menus, ribbon | every transition | as 17 | the hook (`editorKindChanged`) |
| 19 | Cards pane, cards' view, Characters | Open, New, a join, the role | followed the text by `dock.edited`, unrecorded | recorded; followed at the end if not |
| 20 | Source pane caption, lights | Open, New, a rename, the source changing | followed by `followSource` on edits | recorded, followed at the end |
| 21 | Adapt… | the script replaced while it is open (a peer's type change, a join by a pasted link) | planned on the old script, its Insert into the new one | **fixed**: closed, with a status line |
| 22 | Page Setup line | role, type | the page view's line is made with each Preview; the Live Preview's refreshes with it | follows with 11, 12, 16 |
| 23 | Source tab's document | **Choose Source…** with the Source tab open (left open by the middle tabs) | the tab kept the old source; in a bundle (the same home) its text was what the pane read, and a later save could write it back over the new one | **fixed**: the old document is written to its home and let go first, the tab opened again on the new source, the tab that was in front kept there |
| 24 | stripboard | any tab change (left open: "the stripboard is not a tab") | — | by design: one view in the middle; a tab change closes it, its command puts the script's tab in front; it follows the text on edits. Not a view of the hook |
| 25 | Undo | a tab change | — | no state: the focused editor's, else the main one's |
| 26 | collab presence | join, leave | — | the status bar's; follows with 17 |
| 27 | everything | **Revert** | — | there is no Revert command; Restore Draft is one edit (`Replace`), which every view follows as an edit (pages with Follow Edits) |

27 pairs; 15 fixed (1, 2, 4–13, 21, 23 and the outline's rows), the rest checked or by design.

## Left

- The outline's passages carry no status (adapted / cut / to do); the source pane has it. A synopsis line with it
  would make the outline a second source list — wait for Joop.
- The Adapt dialog is closed rather than planned again when the script changes under it.
- The test drives each transition once from two fronts; it does not drive two transitions racing (a Preview from
  one finishing inside the next one's bracket is handled by `pagesComing`, not tested apart).
