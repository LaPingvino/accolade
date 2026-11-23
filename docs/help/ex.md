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

- **:m[ove]** `address` — Move the lines to after a line: :m0 to the top, :m$ to the end, :m-2 up one, :m+1 down one
- **:t** `address` — Copy the lines to after a line (:t. doubles the line)
- **:co[py]** `address` — Copy the lines to after a line, as :t
- **:sc[ene]** `heading or number` — Go to a scene or section: its number, or its heading (Tab lists them)
- **:move-scene-up** — Move the scene or section under the cursor, with all it holds, above the one before (Alt+K in vim)
- **:move-scene-down** — Move the scene or section under the cursor, with all it holds, below the one after (Alt+J in vim)
- **:promote-section** — Make the section under the cursor one level higher (one # fewer; Alt+H in vim)
- **:demote-section** — Make the section under the cursor one level deeper (one # more; Alt+L in vim)
- **:next-heading** — Go to the next scene or section heading (gj in vim)
- **:previous-heading** — Go to this scene's or section's heading, or the one before (gk in vim)
- **:go-parent-section** — Go to the section the cursor's heading is in (gh in vim)
- **:go-first-inside** — Go to the first heading inside the cursor's section (gl in vim)
- **:move-line-up** — Edit > Move Line Up
- **:move-line-down** — Edit > Move Line Down
- **:move-pane-top** — View > Move Pane to Top
- **:move-pane-bottom** — View > Move Pane to Bottom
- **:move-pane-middle** — View > Open Pane in the Middle
- **:move-pane-side** — View > Middle Tab Back to the Side

Moves are named `move-…-up` and `move-…-down`, so `:mo` lists them all; jumps `go-…`, `next-…` and `previous-…`.

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

The pages: `index`, `fountain`, `final-draft`, `adapting`, `fixing-imports`, `production`, `shortcuts`, `formats`, `settings`, `lexington`, `vim`, `ex`, `cheat-shortcuts`, `cheat-fountain`, `cheat-blocks`, `cheat-revisions`, `cheat-breakdown`, `cheat-dood`, `cheat-markdown`, `cheat-lexington`.

## Accolade's commands

Every command of the menus, the ribbon and the palette, by a name made from its label (Next Source Passage is `:next-source-passage`), shorter for the common ones. Where it lives is said after it.

### File

- **:new** — File > New
- **:new-empty** — File > New Empty
- **:new-window** — File > New Window (not in the browser)
- **:new-empty-window** — File > New Empty Window (not in the browser)
- **:new-markdown-document** — File > New Markdown Document
- **:open** — File > Open
- **:recent** `item` — File > Open Recent
- **:open-sample** — File > Open Sample
- **:import-as-markdown** — File > Import as Markdown
- **:import-with-pandoc** — File > Import with pandoc (not in the browser)
- **:start-screenplay-from-this-source** — File > Start a Screenplay from This Source (Markdown)
- **:save** — File > Save
- **:save-as** — File > Save As
- **:rename** — File > Rename
- **:undo-rename** — File > Undo Rename
- **:add-to-my-settings** — File > Add to My Settings (not in the browser)
- **:export** `[format]` — File > Export
- **:export-clean-fountain** — File > Export Clean Fountain (screenplay)
- **:page-setup** — File > Page Setup
- **:print** — File > Print
- **:language-for-new-scripts** `item` — File > New > Language for New Scripts

### Collaboration

- **:save-draft** — Collaboration > Save Draft
- **:drafts** — Collaboration > Drafts
- **:edit-together** — Collaboration > Edit Together
- **:join-shared-script** — Collaboration > Join a Shared Script
- **:compare** — Collaboration > Compare
- **:compare-drafts-side-by-side** — Collaboration > Drafts > Compare Drafts Side by Side

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
- **:select-all** — Edit > Select All
- **:script-language** — Edit > Script Language (screenplay)
- **:editor-keys** `item` — Edit > Settings > Editor Keys

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
- **:suggest-links-to-source** — Adaptation > Suggest Links to the Source (screenplay)
- **:update-source-links** — Adaptation > Update Source Links (screenplay)
- **:source-headings-as-sections** — Adaptation > Source Headings as Sections (screenplay)
- **:source-in-middle** — Adaptation > Source in the Middle (screenplay)
- **:what-accolade-learned** — Adaptation > What Accolade Learned (screenplay)
- **:source** — Source — Adaptation > Source Pane (screenplay)
- **:suggest-from-source-while-writing** — Adaptation > Options > Suggest from the Source While Writing (screenplay)

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

### View

- **:preview** — View > Preview PDF
- **:fold** — View > Fold or Unfold Here
- **:fold-sections** — View > Fold All (Sections Only)
- **:headings-only** — View > Headings Only
- **:unfold-all** — View > Unfold All
- **:statistics** — View > Statistics (screenplay)
- **:fullscreen** — View > Fullscreen
- **:document-type** `item` — View > Document Type
- **:working-as** `item` — View > Working As (screenplay)
- **:outline** — View > Outline
- **:notes** — View > Notes
- **:characters** — View > Characters
- **:cards** — View > Cards (screenplay)
- **:production** — View > Production (screenplay; working as Production)
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
- **:lock-ribbon-order** — View > Lock Ribbon Order (not in the browser)
- **:quick-row** — View > Quick Row (not in the browser)

### Help

- **:palette** — Help > Command Palette
- **:about** — Help > About Accolade
- **:cheat-sheet** `item` — Help > Cheat Sheets
- **:online-help** — Help > Online Help
- **:send-feedback** — Help > Send Feedback

### Preview

- **:fit-width** — Preview > Zoom > Fit Width (not in the browser)
- **:actual-size** — Preview > Zoom > Actual Size (not in the browser)
- **:zoom-in** — Preview > Zoom > Zoom In (not in the browser)
- **:zoom-out** — Preview > Zoom > Zoom Out (not in the browser)
- **:previous-page** — Preview > Pages > Previous Page (not in the browser)
- **:next-page** — Preview > Pages > Next Page (not in the browser)
- **:save-pdf** — Preview > Output > Save PDF (not in the browser)
- **:open-in-viewer** — Preview > Output > Open in Viewer (not in the browser)
- **:preview-page-setup** — Preview > Setup > Page Setup (not in the browser)
- **:preview-text** — Preview > Show > Text (not in the browser)
- **:preview-pages** — Preview > Show > Pages (not in the browser)
- **:preview-refresh** — Preview > Refresh > Refresh (not in the browser)
- **:preview-follow-edits** — Preview > Refresh > Follow Edits (not in the browser)
- **:close-preview** — Preview > Close > Close Preview (not in the browser)
