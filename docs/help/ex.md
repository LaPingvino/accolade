# Ex Commands

With vim's or helix's keys (Preferences > Editor), **:** opens the command line at the window's foot. This page is made from Accolade's own table of its commands, so none is missing; `:help` opens it, `:h` and a command's name at that command's part.

## Finding a command

- As you type, a list above the line shows the commands it may be, each with what it does: those it begins first, then those with its letters in order (`:nsp` finds `:next-source-passage`). An empty line lists the most useful ones.
- **Tab** writes the chosen one in and goes on to the next; **Shift+Tab** back. A command alone in the list is taken, and what it takes listed: scene headings for `:scene`, elements for `:element`, formats for `:export`, files and folders for `:w` and `:e`, a menu's items for a command with a submenu (`:theme`, `:recent`).
- **Up** and **Down** choose without writing it in; **Enter** runs the line as typed, or the one chosen. On an empty line **Up** goes back through the commands run before.
- **Esc** closes the line. On a phone, tap a row: it is completed, and run if it wants nothing more.
- Every name can be shortened to a beginning no other name has, and vim's abbreviations work as in vim (`:s`, `:d`, `:co`); the list shows them as `s[ubstitute]`.

## Ranges

Line commands take a range of lines before their name; without one they act on the cursor's line (`:g`, `:v` and `:w` on all).

- `12` a line, `.` the cursor's, `$` the last, `%` all of them, `'<,'>` the lines last selected in visual mode
- `/pattern/` the next line that matches, `?pattern?` the one before; each with `+n` or `-n` after it
- `from,to` the lines between; `:12` alone goes to line 12
- Patterns are vim's: `\(` `\)` groups, `\|` or, `\<` `\>` a word's ends, `\c` any case; `\v` takes Go's syntax as it is

## Lines

- **:s[ubstitute]** `/pattern/replacement/[gi]` — Replace what matches in the lines (the cursor's; % all): g every match, i any case; & is the match, \1 a group
- **:g[lobal]** `/pattern/command` — Run d, s, j, m or t on every line that matches (all lines if no range); g! on the others
- **:v[global]** `/pattern/command` — Run a command on every line that does not match
- **:d[elete]** — Delete the lines (into the register)
- **:y[ank]** — Copy the lines into the register
- **:j[oin]** — Join the lines into one (the cursor's and the next without a range)
- **:u[ndo]** — Undo the last change

## Moving and going to

- **:m[ove]** `address | [X..Y] after|before Z` — Move the lines to after a line: :m0 to the top, :m$ to the end, :m-2 up one, :m+1 down one; with after or before, scenes or sections: :move 3..5 after 9, :'<,'>move before #Act
- **:t** `address` — Copy the lines to after a line (:t. doubles the line)
- **:co[py]** `address` — Copy the lines to after a line, as :t
- **:sc[ene]** `heading or number` — Go to a scene or section: its number, or its heading (Tab lists them)
- **:move-up** — Move the section, scene, speech or paragraph at the cursor (or the selected ones) up past the one before (View > Structure > Move Up)
- **:move-down** — Move the section, scene, speech or paragraph at the cursor (or the selected ones) down past the next (View > Structure > Move Down)
- **:promote** — Out: a section a level up, a scene out of its section, a speech's line out into action (View > Structure > Promote) (screenplay)
- **:demote** — In: a section a level down, a scene into the next section, action into the speech beside it (View > Structure > Demote) (screenplay)
- **:into-previous** — The speech or paragraph at the cursor into the end of the scene before; a section's synopsis or note into the section before (View > Structure > Into the One Before) (screenplay)
- **:into-next** — The speech or paragraph at the cursor into the start of the next scene; a section's synopsis or note into the next section (View > Structure > Into the Next One) (screenplay)
- **:barf** — Barf: this scene's last speech or paragraph out into the next scene (View > Structure > Pass the Last Speech On) (screenplay)
- **:slurp** — Slurp: the next scene's first speech or paragraph into the end of this one (View > Structure > Take In the Next Speech) (screenplay)
- **:transpose-speakers** — This speech's lines to the speaker before, theirs to this one (the cues change places) (View > Structure > Swap Speakers' Lines) (screenplay)
- **:raise** — The speech's line at the cursor (or the selected ones) out into action (as Promote on a line) (View > Structure > Line to Action) (screenplay)
- **:lower** — The action at the cursor into the speech beside it, as its lines (as Demote on action) (View > Structure > Action into the Speech) (screenplay)
- **:swap-ranges** — Swap scenes, runs of scenes or sections with others (View > Structure > Swap) (screenplay)
- **:move-range** — Move scenes, a run of scenes or a section before or after another (View > Structure > Move) (screenplay)
- **:next-heading** — Go to the next scene or section heading (gj in vim)
- **:previous-heading** — Go to this scene's or section's heading, or the one before (gk in vim)
- **:go-parent-section** — Go to the section the cursor's heading is in (gh in vim)
- **:go-first-inside** — Go to the first heading inside the cursor's section (gl in vim)
- **:move-pane-top** — View > Move Pane to Top
- **:move-pane-bottom** — View > Move Pane to Bottom
- **:move-pane-middle** — View > Open Pane in the Middle
- **:move-pane-side** — View > Middle Tab Back to the Side
- **:move-strip-to** — Move the selected strip to a day, or back to the unscheduled (Production > Move Strip To) (working as Production; while the stripboard is in front)
- **:move-strip-up** — Production > Move Strip Up (working as Production; while the stripboard is in front)
- **:move-strip-down** — Production > Move Strip Down (working as Production; while the stripboard is in front)

Moves are named `move-…-up` and `move-…-down`, so `:mo` lists them all; jumps `go-…`, `next-…` and `previous-…`.

## Production and structure

- **:renumber** `[keep LIST] [only LIST] [from X to Y] [fold|letters] [omitted drop|keep|runs]` — Renumber the scenes as Renumber Scenes… would, its choices typed: keep 5,12A keeps those, only 20-24 renumbers those alone, from X to Y a part; fold or keep letters; what OMITTED scenes do. Without choices, the dialog (screenplay; working as Production)
- **:lock** — Lock the scene numbers (every scene numbered) (screenplay; working as Production)
- **:unlock** `[keep|remove]` — Unlock the scene numbers: keep them in the script, or remove the ones the lock wrote; without either, the dialog that asks (screenplay; working as Production)
- **:omit** `[X..Y]` — Make the scenes OMITTED, their numbers kept (the cursor's without a range): one undo step (screenplay; working as Production)
- **:restore** `[X..Y]` — Bring OMITTED scenes back (the cursor's without a range): one undo step (screenplay; working as Production)
- **:swap** `X..Y with Z..W` — Swap two runs of scenes or two sections (#Name): their links, tags and locked numbers go with them (screenplay)
- **:issue** — Issue a draft: the Issue Draft dialog (screenplay; working as Production)
- **:tag** `category` — Tag the selection (or the word at the cursor) in a breakdown category: :tag props (screenplay; working as Production)
- **:draft** `[name]` — Save a draft now, by that name (Draft n without one)

Scenes are named by their labels, in the script's order: a scene's number as written (`5`, `12A`, `I-1-A`), or in a script without numbers its place (`1`, `2`, `3`…; `(3)` names the third scene in a numbered one); `#Name` a section by the start of its heading; `.` the cursor's scene, `$` the last. `X..Y` runs from X to Y in the script's order (Y before X is an error, never swapped). Tab completes the labels and the words.

## Files and the window

- **:w[rite]** `[file]` — Save; with a file, save as that file
- **:wq** — Save and close the window
- **:x[it]** — Save and close the window, as :wq
- **:exi[t]** — Save and close the window, as :wq
- **:q[uit]** — Close the window (asks to save first); :q! closes it without saving
- **:e[dit]** `[file]` — Open a file (in a new window if this one has work of its own); without one, the Open dialog

## Writing

- **:el[ement]** `name` — Make the cursor's line an element: :element action, :el character…
- **:heading** `[0-6]` — A Markdown line's heading level (0: normal text); without one, the picker

The elements: `scene-heading`, `action`, `character`, `parenthetical`, `dialogue`, `transition`, `centered`, `lyrics`.

## Help

- **:h[elp]** `[command or page]` — This list in the Help tab; :h scene the line on :scene; :h fountain a help page

The pages: `index`, `fun`, `fountain`, `structure`, `final-draft`, `coming-from`, `adapting`, `fixing-imports`, `production`, `merging`, `table-read`, `together`, `shortcuts`, `keyboard`, `formats`, `settings`, `lexington`, `vim`, `ex`, `cheat-shortcuts`, `cheat-fountain`, `cheat-blocks`, `cheat-revisions`, `cheat-breakdown`, `cheat-dood`, `cheat-markdown`, `cheat-lexington`.

## Accolade's commands

Every command of the menus, the ribbon and the palette, by a name made from its label (Next Source Passage is `:next-source-passage`), shorter for the common ones. Where it lives is said after it.

### File

- **:new** — File > New
- **:new-empty** — File > New Empty
- **:new-window** — File > New Window (not in the browser)
- **:new-empty-window** — File > New Empty Window (not in the browser)
- **:new-markdown-document** — File > New Markdown Document
- **:open** — File > Open
- **:open-bundle-folder** — File > Open Bundle Folder (not in the browser)
- **:recent** `item` — File > Open Recent
- **:open-sample** `item` — File > Open Sample
- **:import-as-markdown** — File > Import as Markdown
- **:import-storyboarder-boards** — File > Import Storyboarder Boards (screenplay)
- **:import-pdf-as-screenplay** — File > Import PDF as Screenplay
- **:import-pdf-as-markdown** — File > Import PDF as Markdown
- **:import-with-pandoc** — File > Import with pandoc (not in the browser)
- **:start-screenplay-from-this-source** — File > Start a Screenplay from This Source (Markdown)
- **:save** — File > Save
- **:save-as** — File > Save As
- **:rename** — File > Rename
- **:undo-rename** — File > Undo Rename
- **:add-to-my-settings** — File > Add to My Settings (not in the browser)
- **:merge-changes** — File > Merge Changes
- **:export** `[format]` — File > Export
- **:export-storyboard-timeline-otio** — File > Export Storyboard Timeline (OTIO) (screenplay)
- **:export-clean-fountain** — File > Export Clean Fountain (screenplay)
- **:export-outline** — The sections, scenes' headings, synopses and notes as an outline (OPML) for an outliner: Workflowy, OmniOutliner, Dynalist… (File > Export Outline (OPML) — for Workflowy, OmniOutliner) (screenplay)
- **:export-treatment** — The sections, scenes' headings, synopses and notes as a document to read: PDF or Word (File > Export Treatment (PDF, Word)) (screenplay)
- **:page-setup** — File > Page Setup
- **:print** — File > Print
- **:new-from-structure** `item` — File > New from Structure
- **:language-for-new-scripts** `item` — File > Language for New Scripts
- **:export-writer** — The writer's draft as locked, without production's layers (File > Export Writer's Version) (screenplay)
- **:export-shooting** — The shooting script: scene numbers, OMITTED, revisions (File > Export Shooting Version) (screenplay)
- **:new-from-pdf** — File > New from a PDF

### Collaboration

- **:save-draft** — Collaboration > Save Draft
- **:drafts** — Collaboration > Drafts
- **:edit-together** — Collaboration > Edit Together
- **:join-shared-script** — Collaboration > Join a Shared Script
- **:compare** — Collaboration > Compare
- **:compare-drafts-side-by-side** — Collaboration > Drafts > Compare Drafts Side by Side
- **:table-read-follow-link** — A link for a TV or another screen: its Table Read follows yours, and it edits nothing (the script is shared first) (Collaboration > Copy the Follow Link) (screenplay)

### Edit

- **:title-page** — Edit > Title Page (screenplay)
- **:preferences** — Edit > Preferences
- **:redo** — Edit > Redo
- **:find** — Edit > Find
- **:replace** — Edit > Replace
- **:find-next** — Edit > Find Next
- **:find-previous** — Edit > Find Previous
- **:fix-lines-like-this** — Edit > Fix Lines Like This (screenplay)
- **:cut-selection** — Edit > Cut
- **:copy-selection** — Edit > Copy
- **:paste** — Edit > Paste
- **:paste-structure** — An outline copied from Workflowy, OmniOutliner… (indented lines or OPML) put in as sections and synopses (Edit > Paste as Structure) (screenplay)
- **:select-all** — Edit > Select All
- **:script-language** — Edit > Script Language (screenplay)
- **:editor-keys** `item` — Edit > Settings > Editor Keys
- **:scripts-language-for-new-scripts-by-default** — Edit > Script's Language for New Scripts by Default (screenplay)
- **:select-cue** — Select every character cue in the selection (none: the whole script), to type at all of them (Edit > Clipboard > More… > Select All Character Cues) (screenplay)
- **:select-speech** — Select every speech: cue, parentheticals and dialogue (each of a pair) in the selection (none: the whole script), to type at all of them (Edit > Clipboard > More… > Select All Speeches) (screenplay)
- **:select-dialogue** — Select the dialogue lines (a run between parentheticals as one) in the selection (none: the whole script), to type at all of them (Edit > Clipboard > More… > Select All Dialogue) (screenplay)
- **:select-paren** — Select the parentheticals in the selection (none: the whole script), to type at all of them (Edit > Clipboard > More… > Select All Parentheticals) (screenplay)
- **:select-action** — Select the action paragraphs in the selection (none: the whole script), to type at all of them (Edit > Clipboard > More… > Select All Action Paragraphs) (screenplay)
- **:select-heading** — Select the scene headings in the selection (none: the whole script), to type at all of them (Edit > Clipboard > More… > Select All Scene Headings) (screenplay)
- **:select-section** — Select the section headings in the selection (none: the whole script), to type at all of them (Edit > Clipboard > More… > Select All Section Headings) (screenplay)
- **:select-transition** — Select the transitions in the selection (none: the whole script), to type at all of them (Edit > Clipboard > More… > Select All Transitions) (screenplay)
- **:select-note** — Select the notes, [[ ]] and all in the selection (none: the whole script), to type at all of them (Edit > Clipboard > More… > Select All Notes) (screenplay)
- **:select-beat** — Select the beats: synopses standing on their own, sections marked a beat in the selection (none: the whole script), to type at all of them (Edit > Clipboard > More… > Select All Beats) (screenplay)

### Elsewhere

- **:find-replace** — Find and replace
- **:elements-list-or-buttons** — Elements as a list or as buttons (screenplay)
- **:menus** — Menus
- **:next-pane** — Next Pane
- **:previous-pane** — Previous Pane
- **:ribbon-key-tips** — Ribbon Key Tips

### Format

- **:right-to-left-line** — Format > Right to Left Line
- **:scene-or-section-color** — Format > Scene or Section Color (screenplay)
- **:bold** — Format > Bold
- **:italic** — Format > Italic
- **:underline** — Format > Underline (screenplay)
- **:storyboard-frame** — Format > Storyboard Frame (screenplay)
- **:new-storyboard-frame** — Format > New Storyboard Frame (screenplay)
- **:edit-frame-in** — Format > Edit Frame in (screenplay)
- **:include** — Another file printed where this line is: Highland's {{INCLUDE: file}}, relative to the script's folder (Format > Include File) (screenplay)
- **:marker** — A place to come back to, not printed: Highland's {{%m: label}} (Format > Marker) (screenplay)
- **:next-marker** — To the next {{%m: …}} marker (Format > Next Marker) (screenplay)
- **:previous-marker** — To the {{%m: …}} marker before (Format > Previous Marker) (screenplay)
- **:new-scene** — Format > New Scene (screenplay)
- **:number-scenes-in-headings** — Format > Number Scenes in Headings (screenplay)
- **:strikethrough** — Format > Strikethrough (Markdown)
- **:inline-code** — Format > Inline Code (Markdown)
- **:code-block** — Format > Code Block (Markdown)
- **:link** — Format > Link (Markdown)
- **:image** — Format > Image (Markdown)
- **:bulleted-list** — Format > Bulleted List (Markdown)
- **:numbered-list** — Format > Numbered List (Markdown)
- **:task-list** — Format > Task List (Markdown)
- **:quote** — Format > Quote (Markdown)
- **:horizontal-rule** — Format > Horizontal Rule (Markdown)
- **:table** — Format > Table (Markdown)
- **:footnote** — Format > Footnote (Markdown)
- **:go-note** — Format > Go to Note or Reference (Markdown)
- **:renumber-notes** — Format > Renumber Notes (Markdown)
- **:structure** `item` — A structure's beats (sections and synopses) put in after the cursor's section (before the next # section), one undo step (Format > Insert Beats / Structure) (screenplay)
- **:note** — A note at the cursor, never printed: [[…]] (Markdown: <!-- … -->); [[todo: …]] makes it a to-do (Format > Add Note)
- **:unsorted-note** — A note about the whole script, not tied to a place: kept at its end, listed in the Notes pane (Format > New Unsorted Note)

### Write

- **:inside-outside** — Write > Scene > Inside or Outside (screenplay)
- **:character-name** — Write > Scene > Character Name (screenplay)

### Adaptation

- **:choose-source** — Adaptation > Choose Source (screenplay)
- **:save-source-as** — Adaptation > Save Source As (screenplay)
- **:keep-source-in-script** — Adaptation > Keep Source in the Script (screenplay)
- **:link-to-source-passage** — Adaptation > Link to Source Passage (screenplay)
- **:next-source-passage** — Adaptation > Next Source Passage (screenplay)
- **:previous-source-passage** — Adaptation > Previous Source Passage (screenplay)
- **:adapt-source-passage** — Adaptation > Adapt Source Passage (screenplay)
- **:go-passage-scene** — Adaptation > Go to the Passage's Scene (screenplay)
- **:mark-source-passage-as-cut** — Adaptation > Mark Source Passage as Cut (screenplay)
- **:link-up-script** — Adaptation > Link Up the Script (screenplay)
- **:link-guess-for-this-scene** — Adaptation > Link Guess for This Scene (screenplay)
- **:update-source-links** — Adaptation > Update Source Links (screenplay)
- **:source-headings-as-sections** — Adaptation > Source Headings as Sections (screenplay)
- **:source-in-middle** — Adaptation > Source in the Middle (screenplay)
- **:what-accolade-learned** — Adaptation > What Accolade Learned (screenplay)
- **:source** — Source — Adaptation > Source Pane (screenplay)
- **:suggest-from-source-while-writing** — Adaptation > Options > Suggest from the Source While Writing (screenplay)
- **:lock-source** — Keep the source as it is now as a locked source draft (Adaptation > Lock Source) (screenplay)
- **:unlock-source** — Take the newest source lock out of the drafts store (Adaptation > Unlock Source) (screenplay)
- **:relink-from-here** — Link the script to the source as it is now, and lock that (Adaptation > Relink From Here) (screenplay)

### Production

- **:lock-scene-numbers** — Production > Lock Scene Numbers (screenplay; working as Production)
- **:unlock-scene-numbers** — Production > Unlock Scene Numbers (screenplay; working as Production)
- **:renumber-scenes** — Production > Renumber Scenes (screenplay; working as Production)
- **:omit-scene** — Production > Omit Scene (screenplay; working as Production)
- **:restore-scene** — Production > Restore Scene (screenplay; working as Production)
- **:issue-draft** — Production > Issue Draft (screenplay; working as Production)
- **:withdraw-last-issue** — Production > Withdraw Last Issue (screenplay; working as Production)
- **:lock-pages** — Production > Lock Pages (screenplay; working as Production)
- **:unlock-pages** — Production > Unlock Pages (screenplay; working as Production)
- **:revised-pages-pdf** — Production > Revise > Revised Pages PDF (screenplay; working as Production)
- **:compare-with-last-issue** — Production > Revise > Compare with the Last Issue (screenplay; working as Production)
- **:tag-selection** — Production > Tag Selection (screenplay; working as Production)
- **:untag** — Production > Untag (screenplay; working as Production)
- **:show-tagged-words** — Production > Show Tagged Words (screenplay; working as Production)
- **:break-down** — Production > Break Down (screenplay; working as Production)
- **:next-scene-to-break-down** — Production > Break Down > Next Scene to Break Down (screenplay; working as Production)
- **:mark-scene-done** — Production > Break Down > Mark Scene Done (screenplay; working as Production)
- **:breakdown-sheets** — Production > Break Down > Breakdown Sheets (screenplay; working as Production)
- **:stripboard** — Production > Stripboard (screenplay; working as Production)
- **:schedule** — Production > Schedule (screenplay; working as Production)
- **:add-day** — Production > Schedule > Add Day (screenplay; working as Production)
- **:oneliner** — Production > Schedule > One-Liner (screenplay; working as Production)
- **:day-out-of-days** — Production > Schedule > Day Out of Days (screenplay; working as Production)
- **:export-shoot-days-ics** — Production > Export Shoot Days (.ics) (screenplay; working as Production)
- **:reports** — Production > Reports (screenplay; working as Production)
- **:revision-marks-since** `item` — Production > Options > Revision Marks Since (screenplay)
- **:new-scenes-number** `item` — Production > Options > A New Scene's Number (screenplay; working as Production)
- **:continued-where-scene-breaks** — Production > Options > CONTINUED Where a Scene Breaks (screenplay; working as Production)
- **:number-scenes-in-exports** — Production > Options > Number the Scenes in Exports (screenplay; working as Production)
- **:unschedule-strip** — Production > Unschedule Strip (working as Production; while the stripboard is in front)
- **:undo-board-change** — Take the board's last change back (Production > Undo Board Change) (working as Production; while the stripboard is in front)
- **:edit-day** — Production > Edit Day (working as Production; while the stripboard is in front)
- **:fold-all-days** — Production > Fold All Days (working as Production; while the stripboard is in front)
- **:unfold-all-days** — Production > Unfold All Days (working as Production; while the stripboard is in front)
- **:group-by-location** — Production > Group by Location (working as Production; while the stripboard is in front)
- **:group-by-cast** — Production > Group by Cast (working as Production; while the stripboard is in front)
- **:open-strip-scene** — The selected strip's scene in the script (the board's tab kept) (Production > Open Strip's Scene) (working as Production; while the stripboard is in front)
- **:strip-checked** — The selected strip's scene changed since it was scheduled: checked, its flag taken away (Production > Strip Checked) (working as Production; while the stripboard is in front)
- **:close-stripboard** — Production > Close Stripboard (working as Production; while the stripboard is in front)

### View

- **:preview** — View > Preview PDF
- **:fold** — View > Fold or Unfold Here
- **:fold-sections** — View > Fold All (Sections Only)
- **:headings-only** — View > Headings Only
- **:unfold-all** — View > Unfold All
- **:timeline-strip** — A strip of the story over the script: where you are, its rhythm, who has been gone (View > Timeline Strip) (screenplay)
- **:statistics** — View > Statistics (screenplay)
- **:fullscreen** — View > Fullscreen
- **:document-type** `item` — View > Document Type
- **:working-as** `item` — View > Working As (screenplay)
- **:expand-selection** — Select the unit around the selection: the line, its speech, its scene, its section (View > Structure > Expand Selection) (screenplay)
- **:shrink-selection** — Back to the selection before Expand Selection (View > Structure > Shrink Selection) (screenplay)
- **:outline** — View > Outline
- **:notes** — View > Notes
- **:characters** — View > Characters
- **:cards** — View > Cards (screenplay)
- **:production** — View > Production (screenplay; working as Production)
- **:references** — View > References (Markdown)
- **:help-pane** — View > Help
- **:changes** — View > Changes
- **:side-panes** — View > Side Panes
- **:next-middle-tab** — View > Next Middle Tab
- **:previous-middle-tab** — View > Previous Middle Tab
- **:theme** `item` — View > Window > Theme
- **:larger-text** — View > Window > Larger Text
- **:smaller-text** — View > Window > Smaller Text
- **:line-numbers** — View > Show > Line Numbers
- **:status-bar** — View > Show > Status Bar
- **:ribbon** — View > Window > Ribbon
- **:font** `item` — View > Window > Font
- **:toolbar** — View > Toolbar
- **:tip** — A tip for where the cursor is, in the status bar (View > Show > More… > Show a Tip)
- **:tips-in-status-bar** — View > Show > More… > Tips in the Status Bar
- **:add-to-top-row** — View > Add to the Top Row
- **:reset-top-row** — View > Reset the Top Row
- **:stages** — The script's locked stages: the source, the writer's draft, the issues (View > Stages) (screenplay)
- **:table-read** — The script read aloud: large type on dark, a scene at a time, each actor's lines in a tint of their own (a window for a TV or projector; in the browser, a page to cast) (View > Table Read) (screenplay)
- **:lock-ribbon-order** — View > Lock Ribbon Order (not in the browser)
- **:quick-row** — View > Quick Row (not in the browser)

### Help

- **:palette** — Help > Command Palette
- **:about** — Help > About Accolade
- **:cheat-sheet** `item` — Help > Cheat Sheets
- **:online-help** — Help > Online Help
- **:send-feedback** — Help > Send Feedback

### Preview

- **:fit-width** — Preview > Zoom > Fit Width (while the Preview tab is there)
- **:actual-size** — Preview > Zoom > Actual Size (while the Preview tab is there)
- **:zoom-in** — Preview > Zoom > Zoom In (while the Preview tab is there)
- **:zoom-out** — Preview > Zoom > Zoom Out (while the Preview tab is there)
- **:previous-page** — Preview > Pages > Previous Page (while the Preview tab is there)
- **:next-page** — Preview > Pages > Next Page (while the Preview tab is there)
- **:save-pdf** — Preview > Output > Save PDF (while the Preview tab is there)
- **:open-in-viewer** — Preview > Output > Open in Viewer (while the Preview tab is there)
- **:preview-page-setup** — Preview > Setup > Page Setup (while the Preview tab is there)
- **:preview-text** — Preview > Show > Text (while the Preview tab is there)
- **:preview-pages** — Preview > Show > Pages (while the Preview tab is there)
- **:preview-refresh** — Preview > Refresh > Refresh (while the Preview tab is there)
- **:preview-follow-edits** — Preview > Refresh > Follow Edits (while the Preview tab is there)
- **:close-preview** — Preview > Close > Close Preview (while the Preview tab is there)

### Cards

- **:cards-grid** — The cards in rows, grouped (Cards > Grid) (while the Cards tab is there)
- **:cards-timeline** — The scenes in story order, as long as they are, under their acts (Cards > Timeline) (while the Cards tab is there)
- **:lanes-characters** — A lane for each who speaks: where they are, and where they are gone (Cards > Lanes by Character) (while the Cards tab is there)
- **:lanes-tags** — A lane for each tag on the headings ([[London]]): a storyline's scenes (Cards > Lanes by Tag) (while the Cards tab is there)
- **:timeline-zoom-in** — Cards > Zoom In on the Timeline (while the Cards tab is there)
- **:timeline-zoom-out** — Cards > Zoom Out on the Timeline (while the Cards tab is there)
- **:make-it-the-order** — Put the scenes in the order the groups show, inside each section, as one undo step (Cards > Make It the Order) (while the Cards tab is there)
- **:new-card** — A new scene after the current card's (Cards > New Scene) (while the Cards tab is there)
- **:fold-synopses** — The cards as their headings and how they start, without their synopses (Cards > Fold Synopses) (while the Cards tab is there)
- **:open-card** — The current card's scene in the script (Cards > Open in the Script) (while the Cards tab is there)
- **:back-to-script** — Cards > Back to the Script (while the Cards tab is there)
- **:group-section** — Cards > Group By Section (while the Cards tab is there)
- **:group-color** — Cards > Group By Color (while the Cards tab is there)
- **:group-character** — Cards > Group By Character (while the Cards tab is there)
- **:group-tag** — Cards > Group By Tag (while the Cards tab is there)

### Fix-Up

- **:apply-kept-rules-again** — Fix-Up > Fix > Apply the Kept Rules Again (while the Fix-Up tab is there)
- **:read-with-ocr-again** — Fix-Up > PDF > Read with OCR Again (while the Fix-Up tab is there)
- **:check-for-garbled-text** — Fix-Up > PDF > Check for Garbled Text (while the Fix-Up tab is there)
- **:done-fixing** — Fix-Up > Done > Done Fixing (while the Fix-Up tab is there)

### Title Page

- **:to-script** — Title Page > Page > To the Script (while the Title Page tab is there)
- **:add-title** — Title Page > Add > Add Title (while the Title Page tab is there)
- **:add-credit** — Title Page > Add > Add Credit (while the Title Page tab is there)
- **:add-author** — Title Page > Add > Add Author (while the Title Page tab is there)
- **:add-source** — Title Page > Add > Add Source (while the Title Page tab is there)
- **:add-draft-date** — Title Page > Add > Add Draft Date (while the Title Page tab is there)
- **:add-contact** — Title Page > Add > Add Contact (while the Title Page tab is there)
- **:add-notes** — Title Page > Add > Add Notes (while the Title Page tab is there)
- **:add-copyright** — Title Page > Add > Add Copyright (while the Title Page tab is there)

### Characters

- **:rename-character** — The character renamed wherever the name stands, each in its own case, one undo step (Characters > Character > Rename Everywhere) (while the Characters tab is there)
- **:merge-character** — The character's cues take another's name, their notes together (Characters > Character > Merge Into) (while the Characters tab is there)
- **:add-alias** — Another name the story or a cue has for the character (the learned notes) (Characters > Character > Add Alias) (while the Characters tab is there)
- **:not-a-character** — Left out of the characters and the source's reading (the learned notes); again: a character after all (Characters > Character > Not a Character) (while the Characters tab is there)
- **:character-note** — The character opened in the pane, their note to type in (kept in the character list at the document's end) (Characters > Notes > Their Note) (while the Characters tab is there)
- **:next-cue** — The script's cursor on the character's next cue (Characters > Places > Next Cue) (while the Characters tab is there)
- **:previous-cue** — The script's cursor on the character's cue before (Characters > Places > Previous Cue) (while the Characters tab is there)
- **:their-scenes** `item` — The scenes the character speaks in: one to go to (Characters > Places > Their Scenes) (while the Characters tab is there)
- **:close-characters** — Characters > Close > Close Characters (while the Characters tab is there)

### Revision

- **:next-item** — The next to-do, cut, note, beat, revision note or finding of the kinds shown, from the cursor (round at the end) (Revision > Step > Next Open Item)
- **:previous-item** — The open item before the cursor's, of the kinds shown (Revision > Step > Previous Open Item)
- **:open-items** — Where the cursor is among the open items: the Notes pane lists them (Revision > Step > Open Items)
- **:keep-finding** — The finding at the cursor written into the text as a revision note ([[revision: …]] at its line's end; in Markdown <!-- revision: … --> under its paragraph): saved with it, picked up later (Revision > Item > Keep as Revision Note)
- **:dismiss-item** — A finding out of sight, a line beside the document or in its notes, as Dismissed in Doc says (opening the file again keeps it so; deleting the line brings it back); a note, to-do or revision note taken out of the script (Undo brings it back) (Revision > Item > Dismiss Item)
- **:todo-done** — The to-do at the cursor written as done, [[done: …]] (again: to do again) (Revision > Item > Mark To-do Done)
- **:restore-cut** — The boneyard at the cursor back in the script: its /* */ taken away (an omitted scene with its number) (Revision > Item > Restore Cut)
- **:beat-scene** — The beat at the cursor grown into a scene: New Scene's form, the beat its synopsis (Revision > Item > New Scene from Beat) (screenplay)
- **:edit-item** — The note, to-do or cut at the cursor in the Notes pane, its text to type in (Revision > Item > Edit Item)
- **:remove-done** — Every done to-do taken out of the script, one undo step (Revision > All at Once > Remove Done To-dos)
- **:restore-scene-cuts** — Every cut in the cursor's scene back in the script, one undo step (Revision > All at Once > Restore Cuts in This Scene) (screenplay)
- **:dismissed-in-document** — On: dismissed findings are a paragraph in the document's notes block. Off (the default): beside it (in a bundle, or a .dismissed file next to the file), the document kept clean. Switching moves them (Revision > All at Once > Keep Dismissed Findings in the Document)
- **:show-to-dos** — Step through to-dos too: the Notes pane's filter, the same (Revision > Show > Show To-dos)
- **:show-cuts** — Step through cuts too: the Notes pane's filter, the same (Revision > Show > Show Cuts)
- **:show-notes** — Step through notes too: the Notes pane's filter, the same (Revision > Show > Show Notes)
- **:show-beats** — Step through beats too: the Notes pane's filter, the same (Revision > Show > Show Beats)
- **:show-revision-notes** — Step through revision notes too: the Notes pane's filter, the same (Revision > Show > Show Revision Notes)
- **:show-findings** — Step through findings too: the Notes pane's filter, the same (Revision > Show > Show Findings)
- **:where-beats-land** `item` — Each section's page and place in the script, beside its structure's usual range; each goes there (Revision > Beats > Where the Beats Land) (screenplay)
- **:next-off-range** — The next section landing outside its structure's usual range, and what to do about it (Revision > Beats > Next Section Off Its Range) (screenplay)
- **:previous-off-range** — The section before the cursor's landing outside its usual range (Revision > Beats > Previous Section Off Its Range) (screenplay)
- **:every-section-beat** — On: every # section is a beat (in the Notes pane, the counts, the cards), marked [[beat]] or not. Off (the default): only sections marked [[beat]] and synopses standing on their own. The script's own, kept in its .accolade (Revision > Beats > More… > Every Section Is a Beat) (screenplay)

### Session

- **:follow** `item` — Session > Follow (while the Session tab is there)
- **:stop-following** — Session > Stop Following (while the Session tab is there)
- **:copy-link** — Session > Copy the Link (while the Session tab is there)
- **:leave-shared-script** — Session > Leave the Shared Script (while the Session tab is there)

### Compare

- **:previous-change** — The changed scene before (Compare > Previous Change) (while the Compare tab is there)
- **:next-change** — The next changed scene (Compare > Next Change) (while the Compare tab is there)
- **:take** — Take this change into the script, then the next (Compare > Take) (while the Compare tab is there)
- **:leave** — Leave the script as it is here, then the next change (Compare > Leave) (while the Compare tab is there)
- **:take-scene** — Take every change in this change's scene (Compare > Take All From This Scene) (while the Compare tab is there)
- **:merge-taken** — Write what is taken into the script, as a new draft (Compare > Merge Taken Changes) (while the Compare tab is there)
- **:close-compare** — Compare > Close Compare (while the Compare tab is there)

### Table Read

- **:table-read-previous** — The scene picked before this one (←, Page Up) (Table Read > Previous Scene Read) (while the Table Read tab is there)
- **:table-read-next** — The next scene picked (→, Page Down, Space) (Table Read > Next Scene Read) (while the Table Read tab is there)
- **:table-read-pick** — The panel: the scenes, a character's scenes, a tag's; narrator, storyboard, tints; who to follow (Table Read > Pick What Is Read) (while the Table Read tab is there)
- **:table-read-narrator** — Headings and action read by one voice, marked NARRATOR (Table Read > Narrator Mode) (while the Table Read tab is there)
- **:table-read-storyboard** — The scene's storyboard frames between its lines (Table Read > Show Storyboard Frames) (while the Table Read tab is there)
- **:table-read-beside** — The storyboard's frames beside the lines they follow, each with its time (else between them) (Table Read > Storyboard Beside the Lines) (while the Table Read tab is there)
- **:table-read-tints** — Each actor's lines in a colour of their own (Table Read > Actor Tints) (while the Table Read tab is there)
- **:table-read-fullscreen** — The Table Read filling the screen (in the browser, the page: cast it with Chrome's Cast…) (Table Read > Table Read Full Screen) (while the Table Read tab is there)
- **:table-read-follow** `item` — The Table Read goes where they are reading (the script shared: who is here) (Table Read > Follow in the Table Read) (while the Table Read tab is there)
- **:table-read-close** — Table Read > Close Table Read (while the Table Read tab is there)
