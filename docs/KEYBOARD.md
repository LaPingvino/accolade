# The keyboard audit

Made by `UPDATE_KEYBOARD=1 go test -run TestKeyboardAudit .` from the command registry (commands.go), the ribbon's key tips on the sample, the ex table (ex_table.go) and the tables of keyboard.go; the test fails when this file is stale. LOOP.md's keyboard pass (2026-11-20) asked for it.

## How every command is reached

Every command of the registry is in the command palette (Ctrl+K, F1) in some state of the window (TestEveryCommandByKeyboard), and on the `:` line by its ex name (vim's and helix's keys). Fyne's menus have no mnemonics (no Alt+letter into a menu, no underlined letters; the word is nowhere in the Fyne fork's source): the keyboard's way into the ribbon is its **key tips** instead — F10, or Alt pressed and let go (not on a Mac), then the tab's letter, then the button's (the column below; a disabled button has none at that moment, and the letters are given again when the buttons change). In the Menus look, the ribbon's ☰ and the palette are the way. The key tips below are the sample's (a script in production at the Schedule stage): the Production tab shows the stage the script is at first, so another stage's buttons have their tips when the script is at it.

Keys per platform: Linux and Windows have the registry's keys, a Mac its macKeys where set (Ctrl is Cmd there otherwise), the browser its webKeys where set (the browser keeps some for itself). Contextual commands (the Stripboard's, the Cards', the Table Read's …) are there while their view is.

| Command | id | Linux/Windows | macOS | browser | key tips | ex |
|---|---|---|---|---|---|---|
| New | `file.new` | Ctrl+N | Cmd+N | Ctrl+Alt+N | F N | :new |
| New Empty | `file.newEmpty` | Ctrl+Alt+N | Cmd+Option+N | Ctrl+Shift+Alt+N | F E | :new-empty |
| New Window | `file.newWindow` | Ctrl+Shift+N | Cmd+Shift+N | — | F W | :new-window |
| New Empty Window | `file.newEmptyWindow` | Ctrl+Shift+Alt+N | Cmd+Shift+Option+N | — | F P | :new-empty-window |
| New Markdown Document | `file.newMarkdown` |  |  |  | F M | :new-markdown-document |
| Open… | `file.open` | Ctrl+O | Cmd+O | Ctrl+O | F O | :open |
| Open Bundle Folder… | `file.openFolder` |  |  | — |  | :open-bundle-folder |
| Open Recent | `file.recent` |  |  |  | F R | :recent |
| Open Sample | `file.openSample` |  |  |  | F S | :open-sample |
| Import as Markdown… | `file.import` |  |  |  | F T | :import-as-markdown |
| Import with pandoc… | `file.importPandoc` |  |  | — | F H | :import-with-pandoc |
| Start a Screenplay from This Source | `file.startScreenplay` |  |  |  |  | :start-screenplay-from-this-source |
| Save | `file.save` | Ctrl+S | Cmd+S | Ctrl+S | F A | :write |
| Save As… | `file.saveAs` | Ctrl+Shift+S | Cmd+Shift+S | Ctrl+Shift+S | F V | :save-as |
| Rename… | `file.rename` |  |  |  | F 1 | :rename |
| Undo Rename | `file.undoRename` |  |  |  |  | :undo-rename |
| Add to My Settings… | `file.addSettings` |  |  |  |  | :add-to-my-settings |
| Save Draft… | `file.saveDraft` |  |  |  | C S | :draft |
| Drafts… | `file.drafts` |  |  |  | C D | :drafts |
| Merge Changes… | `file.mergeChanges` |  |  |  | C M | :merge-changes |
| Edit Together… | `file.share` |  |  |  | C E | :edit-together |
| Join a Shared Script… | `file.join` |  |  |  | C J | :join-shared-script |
| Title Page… | `file.titlePage` | Ctrl+Shift+T | Cmd+Shift+T | Ctrl+Shift+Alt+T | W G | :title-page |
| Export… | `file.export` | Ctrl+E | Cmd+E | Ctrl+E | F X | :export |
| Export Clean Fountain… | `file.exportClean` |  |  |  | F C | :export-clean-fountain |
| Page Setup… | `file.pageSetup` |  |  |  | F G | :page-setup |
| Print… | `file.print` | Ctrl+P | Cmd+P | Ctrl+P | F 2 | :print |
| Preferences… | `app.preferences` | Ctrl+, | Cmd+, | Ctrl+, | E 2 | :preferences |
| Undo | `edit.undo` | Ctrl+Z |  |  | E U | :undo |
| Redo | `edit.redo` | Ctrl+Shift+Z | Cmd+Shift+Z | Ctrl+Shift+Z | E R | :redo |
| Find… | `edit.find` | Ctrl+F | Cmd+F | Ctrl+F | E F | :find |
| Replace… | `edit.replace` | Ctrl+H | Cmd+Option+F | Ctrl+H | E L | :replace |
| Find and replace | `edit.findReplace` | Ctrl+F |  |  |  | :find-replace |
| Find Next | `edit.findNext` | Ctrl+G | Cmd+G | Ctrl+G | E N | :find-next |
| Find Previous | `edit.findPrevious` | Ctrl+Shift+G | Cmd+Shift+G | Ctrl+Shift+G | E E | :find-previous |
| Fix Lines Like This… | `edit.fixLines` |  |  |  | E I | :fix-lines-like-this |
| Cut | `edit.cut` | Ctrl+X |  |  | E C | :cut-selection |
| Copy | `edit.copy` | Ctrl+C |  |  | E O | :copy-selection |
| Paste | `edit.paste` | Ctrl+V |  |  | E P | :paste |
| Select All | `edit.selectAll` | Ctrl+A |  |  | E S | :select-all |
| Scene Heading | `format.element.Scene Heading` | Ctrl+1 | Cmd+1 | Ctrl+1 | W S | :element scene-heading |
| Action | `format.element.Action` | Ctrl+2 | Cmd+2 | Ctrl+2 | W A | :element action |
| Character | `format.element.Character` | Ctrl+3 | Cmd+3 | Ctrl+3 | W C | :element character |
| Parenthetical | `format.element.Parenthetical` | Ctrl+4 | Cmd+4 | Ctrl+4 | W P | :element parenthetical |
| Dialogue | `format.element.Dialogue` | Ctrl+5 | Cmd+5 | Ctrl+5 | W D | :element dialogue |
| Transition | `format.element.Transition` | Ctrl+6 | Cmd+6 | Ctrl+6 | W T | :element transition |
| Centered | `format.element.Centered` | Ctrl+7 | Cmd+7 | Ctrl+7 | W E | :element centered |
| Lyrics | `format.element.Lyrics` | Ctrl+8 | Cmd+8 | Ctrl+8 | W L | :element lyrics |
| Right to Left Line | `format.rtl` |  |  |  | W R | :right-to-left-line |
| Script Language… | `format.language` |  |  |  | E G | :script-language |
| Scene or Section Color… | `format.color` | Ctrl+Shift+L | Cmd+Shift+L | Ctrl+Shift+L | W O | :scene-or-section-color |
| Bold | `format.bold` | Ctrl+B | Cmd+B | Ctrl+B | W B | :bold |
| Italic | `format.italic` | Ctrl+I | Cmd+I | Ctrl+I | W 1 | :italic |
| Underline | `format.underline` | Ctrl+U | Cmd+U | Ctrl+U | W U | :underline |
| Storyboard Frame… | `write.storyboardFrame` |  |  |  | W F | :storyboard-frame |
| Include File… | `write.include` |  |  |  |  | :include |
| Marker… | `write.marker` | Ctrl+M | Control+M | Ctrl+M |  | :marker |
| Next Marker | `write.nextMarker` | Ctrl+Alt+M | Control+Option+M | Ctrl+Alt+M |  | :next-marker |
| Previous Marker | `write.previousMarker` | Ctrl+Shift+Alt+M | Control+Shift+Option+M | Ctrl+Shift+Alt+M |  | :previous-marker |
| New Scene… | `write.newScene` |  |  |  | W N | :new-scene |
| Number Scenes in Headings | `format.numberScenes` |  |  |  |  | :number-scenes-in-headings |
| Inside or Outside… | `write.place` |  |  |  | W I | :inside-outside |
| Character Name… | `write.character` |  |  |  | W H | :character-name |
| Elements as a list or as buttons | `write.elementKind` |  |  |  |  | :elements-list-or-buttons |
| Normal Text | `md.heading0` | Ctrl+0 | Cmd+0 | Ctrl+0 |  | :heading 0 |
| Heading 1 | `md.heading1` | Ctrl+1 | Cmd+1 | Ctrl+1 |  | :heading 1 |
| Heading 2 | `md.heading2` | Ctrl+2 | Cmd+2 | Ctrl+2 |  | :heading 2 |
| Heading 3 | `md.heading3` | Ctrl+3 | Cmd+3 | Ctrl+3 |  | :heading 3 |
| Heading 4 | `md.heading4` | Ctrl+4 | Cmd+4 | Ctrl+4 |  | :heading 4 |
| Heading 5 | `md.heading5` | Ctrl+5 | Cmd+5 | Ctrl+5 |  | :heading 5 |
| Heading 6 | `md.heading6` | Ctrl+6 | Cmd+6 | Ctrl+6 |  | :heading 6 |
| Heading… | `md.headings` |  |  |  |  | :heading |
| Strikethrough | `md.strike` |  |  |  |  | :strikethrough |
| Inline Code | `md.code` |  |  |  |  | :inline-code |
| Code Block | `md.codeBlock` |  |  |  |  | :code-block |
| Link… | `md.link` | Ctrl+L | Cmd+L | Ctrl+L |  | :link |
| Image… | `md.image` |  |  |  |  | :image |
| Bulleted List | `md.bullets` |  |  |  |  | :bulleted-list |
| Numbered List | `md.numbers` |  |  |  |  | :numbered-list |
| Task List | `md.tasks` |  |  |  |  | :task-list |
| Quote | `md.quote` |  |  |  |  | :quote |
| Horizontal Rule | `md.rule` |  |  |  |  | :horizontal-rule |
| Table… | `md.table` |  |  |  |  | :table |
| Footnote | `md.footnote` |  |  |  |  | :footnote |
| Go to Note or Reference | `md.noteJump` |  |  |  |  | :go-note |
| Renumber Notes | `md.renumberNotes` |  |  |  |  | :renumber-notes |
| Choose Source… | `source.choose` |  |  |  | A C | :choose-source |
| Save Source As… | `source.saveAs` |  |  |  | A A | :save-source-as |
| Keep Source in the Script | `source.keepInside` |  |  |  | A K | :keep-source-in-script |
| Link to Source Passage | `source.link` |  |  |  | A L | :link-to-source-passage |
| Next Source Passage | `source.next` | Ctrl+Alt+J | Cmd+Option+J | Ctrl+Alt+J | A N | :next-source-passage |
| Previous Source Passage | `source.previous` | Ctrl+Alt+K | Cmd+Option+K | Ctrl+Alt+K | A P | :previous-source-passage |
| Adapt Source Passage… | `source.adapt` | Ctrl+Alt+A | Cmd+Option+A | Ctrl+Alt+A | A D | :adapt-source-passage |
| Go to the Passage's Scene | `source.goScene` |  |  |  |  | :go-passage-scene |
| Mark Source Passage as Cut | `source.cut` |  |  |  | A M | :mark-source-passage-as-cut |
| Link Up the Script… | `source.linkUp` |  |  |  | A U | :link-up-script |
| Link Guess for This Scene… | `source.linkGuess` | Ctrl+Alt+G | Cmd+Option+G | Ctrl+Alt+G |  | :link-guess-for-this-scene |
| Update Source Links | `source.update` |  |  |  | A T | :update-source-links |
| Source Headings as Sections | `source.sections` |  |  |  | A H | :source-headings-as-sections |
| Source in the Middle | `source.middle` |  |  |  | A I | :source-in-middle |
| What Accolade Learned… | `source.learned` |  |  |  | A E | :what-accolade-learned |
| Lock Scene Numbers | `production.lockNumbers` |  |  |  |  | :lock |
| Unlock Scene Numbers… | `production.unlockNumbers` |  |  |  |  | :unlock |
| Renumber Scenes… | `production.renumber` |  |  |  |  | :renumber |
| Omit Scene | `production.omit` |  |  |  | P M | :omit |
| Restore Scene | `production.restore` |  |  |  | P R | :restore |
| Issue Draft… | `production.issue` |  |  |  |  | :issue |
| Withdraw Last Issue… | `production.withdraw` |  |  |  |  | :withdraw-last-issue |
| Lock Pages | `production.lockPages` |  |  |  |  | :lock-pages |
| Unlock Pages… | `production.unlockPages` |  |  |  | P U | :unlock-pages |
| Revised Pages PDF… | `production.revisedPages` |  |  |  |  | :revised-pages-pdf |
| Compare with the Last Issue | `production.compare` |  |  |  |  | :compare-with-last-issue |
| Tag Selection… | `production.tag` | Ctrl+Shift+B | Cmd+Shift+B | Ctrl+Shift+B |  | :tag |
| Untag | `production.untag` |  |  |  |  | :untag |
| Show Tagged Words | `production.showTags` |  |  |  |  | :show-tagged-words |
| Break Down… | `production.breakdown` |  |  |  | P B | :break-down |
| Next Scene to Break Down | `production.nextScene` |  |  |  |  | :next-scene-to-break-down |
| Mark Scene Done | `production.sceneDone` |  |  |  |  | :mark-scene-done |
| Breakdown Sheets… | `production.sheets` |  |  |  |  | :breakdown-sheets |
| Stripboard | `production.stripboard` |  |  |  | P S | :stripboard |
| Schedule… | `production.schedule` |  |  |  | P C | :schedule |
| Add Day | `production.addDay` |  |  |  | P A | :add-day |
| One-Liner… | `production.oneLiner` |  |  |  | P O | :oneliner |
| Day Out of Days… | `production.dood` |  |  |  | P D | :day-out-of-days |
| Export Shoot Days (.ics)… | `production.ics` |  |  |  | P E | :export-shoot-days-ics |
| Reports… | `production.reports` |  |  |  | P T | :reports |
| Compare… | `collab.compare` |  |  |  | C O | :compare |
| Preview PDF | `view.preview` | Ctrl+Shift+P | Cmd+Shift+P | Ctrl+Shift+P | V R | :preview |
| Fold or Unfold Here | `view.foldHere` | Ctrl+Shift+. | Cmd+Shift+. | Ctrl+Shift+. | V F | :fold |
| Fold All (Sections Only) | `view.foldSections` |  |  |  | V D | :fold-sections |
| Headings Only | `view.headingsOnly` |  |  |  | V E | :headings-only |
| Unfold All | `view.unfoldAll` |  |  |  | V U | :unfold-all |
| Timeline Strip | `view.timelineStrip` |  |  |  | V 1 | :timeline-strip |
| Statistics… | `view.statistics` |  |  |  | V T | :statistics |
| Fullscreen | `view.fullscreen` | Ctrl+Shift+F, Alt+Enter, Alt+Enter | Cmd+Shift+F, Option+Enter, Option+Enter | Ctrl+Shift+F, Alt+Enter, Alt+Enter | V 2 | :fullscreen |
| Document Type | `view.documentType` |  |  |  | V M | :document-type |
| Working As | `view.role` |  |  |  | V W | :working-as |
| Menus… | `view.menu` |  |  |  |  | :menus |
| Next Pane | `view.nextPane` | F6 | F6 | F6 |  | :next-pane |
| Previous Pane | `view.previousPane` | Shift+F6 | Shift+F6 | Shift+F6 |  | :previous-pane |
| Ribbon Key Tips | `view.keyTips` | F10 | F10 | F10 |  | :ribbon-key-tips |
| Command Palette… | `app.palette` | Ctrl+K, F1 | Cmd+K, F1 | Ctrl+K, F1 |  | :palette |
| About Accolade | `help.about` |  |  |  |  | :about |
| Move Up | `structure.up` | Alt+Up | Option+Up | Alt+Up | E M | :move-up |
| Move Down | `structure.down` | Alt+Down | Option+Down | Alt+Down | E D | :move-down |
| Promote | `structure.promote` | Alt+Left | Control+Option+Left | Alt+Left | E T | :promote |
| Demote | `structure.demote` | Alt+Right | Control+Option+Right | Alt+Right | E 1 | :demote |
| Into the Scene Before | `structure.intoPrev` | Shift+Alt+Up | Shift+Option+Up | Shift+Alt+Up |  | :into-previous |
| Into the Next Scene | `structure.intoNext` | Shift+Alt+Down | Shift+Option+Down | Shift+Alt+Down |  | :into-next |
| Pass the Last Speech On | `structure.barf` | Shift+Alt+Left | Control+Shift+Option+Left | Shift+Alt+Left |  | :barf |
| Take In the Next Speech | `structure.slurp` | Shift+Alt+Right | Control+Shift+Option+Right | Shift+Alt+Right |  | :slurp |
| Swap Speakers' Lines | `structure.transpose` | Alt+T | Option+T | Alt+T |  | :transpose-speakers |
| Line to Action | `structure.raise` |  |  |  |  | :raise |
| Action into the Speech | `structure.lower` |  |  |  |  | :lower |
| Expand Selection | `structure.expand` | Alt+O | Option+O | Alt+O |  | :expand-selection |
| Shrink Selection | `structure.shrink` | Alt+I | Option+I | Alt+I |  | :shrink-selection |
| Swap… | `structure.swap` |  |  |  | E W | :swap |
| Move… | `structure.move` |  |  |  | E V | :move-range |
| Outline | `view.outline` | Ctrl+Shift+O | Cmd+Shift+O | Ctrl+Shift+O | V O | :outline |
| Notes | `view.notes` | Ctrl+Shift+K | Cmd+Shift+K | Ctrl+Shift+K | V N | :notes |
| Characters | `view.characters` |  |  |  | V C | :characters |
| Cards | `view.cards` | Ctrl+Shift+D | Cmd+Shift+D | Ctrl+Shift+D | V A | :cards |
| Source | `view.source` |  |  |  | A S | :source |
| Production | `view.production` |  |  |  | P P | :production |
| References | `view.references` |  |  |  |  | :references |
| Help | `view.help` |  |  |  | V H | :help-pane |
| Changes | `view.changes` | Ctrl+Shift+H | Cmd+Shift+H | Ctrl+Shift+H | C C | :changes |
| Side Panes | `view.dock` |  |  |  | V I | :side-panes |
| Move Pane to Top | `view.paneTop` |  |  |  |  | :move-pane-top |
| Move Pane to Bottom | `view.paneBottom` |  |  |  |  | :move-pane-bottom |
| Open Pane in the Middle | `view.paneMiddle` |  |  |  |  | :move-pane-middle |
| Middle Tab Back to the Side | `view.paneSide` |  |  |  |  | :move-pane-side |
| Next Middle Tab | `view.midNext` | Ctrl+PgDn | Cmd+PgDn | Alt+PgDn |  | :next-middle-tab |
| Previous Middle Tab | `view.midPrev` | Ctrl+PgUp | Cmd+PgUp | Alt+PgUp |  | :previous-middle-tab |
| Help Contents | `help.page.index` |  |  |  |  | :help index |
| Fountain Reference | `help.page.fountain` |  |  |  |  | :help fountain |
| Coming from Final Draft | `help.page.final-draft` |  |  |  |  | :help final-draft |
| Adapting a Story | `help.page.adapting` |  |  |  |  | :help adapting |
| Fixing an Imported Script | `help.page.fixing-imports` |  |  |  |  | :help fixing-imports |
| Production Help | `help.page.production` |  |  |  |  | :help production |
| Drafts, Compare and Merging | `help.page.merging` |  |  |  |  | :help merging |
| The Table Read | `help.page.table-read` |  |  |  |  | :help table-read |
| Writing Together | `help.page.together` |  |  |  |  | :help together |
| Keyboard Shortcuts | `help.page.shortcuts` |  |  |  |  | :help shortcuts |
| Keyboard Only | `help.page.keyboard` |  |  |  |  | :help keyboard |
| File Formats | `help.page.formats` |  |  |  |  | :help formats |
| Settings Files | `help.page.settings` |  |  |  |  | :help settings |
| Lexington on the Command Line | `help.page.lexington` |  |  |  |  | :help lexington |
| Vim and Helix Modes | `help.page.vim` |  |  |  |  | :help vim |
| Ex Commands | `help.page.ex` |  |  |  |  | :help ex |
| Keyboard Shortcuts | `help.cheat-shortcuts` |  |  |  |  | :help cheat-shortcuts |
| Fountain Syntax | `help.cheat-fountain` |  |  |  |  | :help cheat-fountain |
| Accolade Blocks & Links | `help.cheat-blocks` |  |  |  |  | :help cheat-blocks |
| Revision Colours | `help.cheat-revisions` |  |  |  |  | :help cheat-revisions |
| Breakdown Categories | `help.cheat-breakdown` |  |  |  |  | :help cheat-breakdown |
| Day Out of Days Codes | `help.cheat-dood` |  |  |  |  | :help cheat-dood |
| Markdown Syntax | `help.cheat-markdown` |  |  |  |  | :help cheat-markdown |
| Lexington Flags | `help.cheat-lexington` |  |  |  |  | :help cheat-lexington |
| Cheat Sheets | `help.cheats` |  |  |  |  | :cheat-sheet |
| Online Help | `help.online` |  |  |  |  | :online-help |
| Send Feedback… | `help.feedback` |  |  |  |  | :send-feedback |
| Theme | `set.theme` |  |  |  | V 4 | :theme |
| Larger Text | `set.textLarger` |  |  |  | V 5 | :larger-text |
| Smaller Text | `set.textSmaller` |  |  |  | V X | :smaller-text |
| Line Numbers | `set.lineNumbers` |  |  |  | V L | :line-numbers |
| Status Bar | `set.statusBar` |  |  |  | V B | :status-bar |
| Ribbon | `set.ribbon` |  |  |  | V 3 | :ribbon |
| Suggest from the Source While Writing | `set.suggest` |  |  |  | A W | :suggest-from-source-while-writing |
| Revision Marks Since | `set.revisionMarks` |  |  |  | P V | :revision-marks-since |
| A New Scene's Number | `set.lettering` |  |  |  | P L | :new-scenes-number |
| CONTINUED Where a Scene Breaks | `set.continueds` |  |  |  | P N | :continued-where-scene-breaks |
| Editor Keys | `set.editorKeys` |  |  |  | E K | :editor-keys |
| Font | `set.font` |  |  |  | V 6 | :font |
| Language for New Scripts | `set.newLanguage` |  |  |  | F I | :language-for-new-scripts |
| Toolbar | `set.toolbar` |  |  |  |  | :toolbar |
| Number the Scenes in Exports | `set.sceneNumbers` |  |  |  | P 1 | :number-scenes-in-exports |
| Compare Drafts Side by Side | `set.draftsSide` |  |  |  | C B | :compare-drafts-side-by-side |
| Fit Width | `preview.fit` |  |  |  |  | :fit-width |
| Actual Size | `preview.actual` |  |  |  |  | :actual-size |
| Zoom In | `preview.zoomIn` |  |  |  |  | :zoom-in |
| Zoom Out | `preview.zoomOut` |  |  |  |  | :zoom-out |
| Previous Page | `preview.previousPage` |  |  |  |  | :previous-page |
| Next Page | `preview.nextPage` |  |  |  |  | :next-page |
| Save PDF… | `preview.savePDF` |  |  |  |  | :save-pdf |
| Open in Viewer | `preview.openViewer` |  |  | — |  | :open-in-viewer |
| Page Setup… | `preview.pageSetup` |  |  |  |  | :preview-page-setup |
| Text | `preview.text` |  |  |  |  | :preview-text |
| Pages | `preview.pages` |  |  |  |  | :preview-pages |
| Refresh | `preview.refresh` | F5 | F5 | F5 |  | :preview-refresh |
| Follow Edits | `preview.follow` |  |  |  |  | :preview-follow-edits |
| Close Preview | `preview.close` |  |  |  |  | :close-preview |
| Move Strip To… | `strips.moveTo` |  |  |  |  | :move-strip-to |
| Move Strip Up | `strips.up` |  |  |  |  | :move-strip-up |
| Move Strip Down | `strips.down` |  |  |  |  | :move-strip-down |
| Unschedule Strip | `strips.unschedule` |  |  |  |  | :unschedule-strip |
| Undo Board Change | `strips.undo` |  |  |  |  | :undo-board-change |
| Edit Day… | `strips.editDay` |  |  |  |  | :edit-day |
| Fold All Days | `strips.fold` |  |  |  |  | :fold-all-days |
| Unfold All Days | `strips.unfold` |  |  |  |  | :unfold-all-days |
| Group by Location | `strips.byLocation` |  |  |  |  | :group-by-location |
| Group by Cast | `strips.byCast` |  |  |  |  | :group-by-cast |
| Open Strip's Scene | `strips.open` |  |  |  |  | :open-strip-scene |
| Strip Checked | `strips.checked` |  |  |  |  | :strip-checked |
| Close Stripboard | `strips.close` |  |  |  |  | :close-stripboard |
| Grid | `cards.grid` |  |  |  |  | :cards-grid |
| Timeline | `cards.timeline` |  |  |  |  | :cards-timeline |
| Lanes by Character | `cards.lanesCharacters` |  |  |  |  | :lanes-characters |
| Lanes by Tag | `cards.lanesTags` |  |  |  |  | :lanes-tags |
| Zoom In on the Timeline | `cards.zoomIn` |  |  |  |  | :timeline-zoom-in |
| Zoom Out on the Timeline | `cards.zoomOut` |  |  |  |  | :timeline-zoom-out |
| Make It the Order | `cards.makeOrder` |  |  |  |  | :make-it-the-order |
| New Scene… | `cards.newScene` |  |  |  |  | :new-card |
| Fold Synopses | `cards.foldSynopses` |  |  |  |  | :fold-synopses |
| Open in the Script | `cards.open` |  |  |  |  | :open-card |
| Back to the Script | `cards.close` |  |  |  |  | :back-to-script |
| Group By Section | `cards.groupSection` |  |  |  |  | :group-section |
| Group By Color | `cards.groupColor` |  |  |  |  | :group-color |
| Group By Character | `cards.groupCharacter` |  |  |  |  | :group-character |
| Group By Tag | `cards.groupTag` |  |  |  |  | :group-tag |
| Add to the Top Row… | `view.topRowAdd` |  |  |  |  | :add-to-top-row |
| Reset the Top Row | `view.topRowReset` |  |  |  |  | :reset-top-row |
| Apply the Kept Rules Again… | `fixup.applyRules` |  |  |  |  | :apply-kept-rules-again |
| Read with OCR Again… | `fixup.ocrAgain` |  |  |  |  | :read-with-ocr-again |
| Check for Garbled Text | `fixup.checkText` |  |  |  |  | :check-for-garbled-text |
| Done Fixing | `fixup.done` |  |  |  |  | :done-fixing |
| To the Script | `titlepage.toScript` |  |  |  | T H | :to-script |
| Add Title | `titlepage.add.title` |  |  |  | T T | :add-title |
| Add Credit | `titlepage.add.credit` |  |  |  | T C | :add-credit |
| Add Author | `titlepage.add.author` |  |  |  | T A | :add-author |
| Add Source | `titlepage.add.source` |  |  |  | T S | :add-source |
| Add Draft Date | `titlepage.add.draftdate` |  |  |  | T D | :add-draft-date |
| Add Contact | `titlepage.add.contact` |  |  |  | T O | :add-contact |
| Add Notes | `titlepage.add.notes` |  |  |  | T N | :add-notes |
| Add Copyright | `titlepage.add.copyright` |  |  |  | T Y | :add-copyright |
| Rename Everywhere… | `chars.rename` |  |  |  |  | :rename-character |
| Merge Into… | `chars.merge` |  |  |  |  | :merge-character |
| Add Alias… | `chars.alias` |  |  |  |  | :add-alias |
| Not a Character | `chars.not` |  |  |  |  | :not-a-character |
| Their Note | `chars.note` |  |  |  |  | :character-note |
| Next Cue | `chars.nextCue` |  |  |  |  | :next-cue |
| Previous Cue | `chars.previousCue` |  |  |  |  | :previous-cue |
| Their Scenes | `chars.scenes` |  |  |  |  | :their-scenes |
| Close Characters | `chars.close` |  |  |  |  | :close-characters |
| Follow | `session.follow` |  |  |  |  | :follow |
| Stop Following | `session.stopFollowing` |  |  |  |  | :stop-following |
| Copy the Link | `session.copyLink` |  |  |  |  | :copy-link |
| Leave the Shared Script | `session.leave` |  |  |  |  | :leave-shared-script |
| Previous Change | `compare.previous` |  |  |  |  | :previous-change |
| Next Change | `compare.next` |  |  |  |  | :next-change |
| Take | `compare.take` |  |  |  |  | :take |
| Leave | `compare.leave` |  |  |  |  | :leave |
| Take All From This Scene | `compare.takeScene` |  |  |  |  | :take-scene |
| Merge Taken Changes | `compare.merge` |  |  |  |  | :merge-taken |
| Close Compare | `compare.close` |  |  |  |  | :close-compare |
| Stages… | `view.stages` |  |  |  | V G | :stages |
| Lock Source | `source.lock` |  |  |  |  | :lock-source |
| Unlock Source | `source.unlock` |  |  |  |  | :unlock-source |
| Relink From Here… | `source.relinkFromHere` |  |  |  |  | :relink-from-here |
| Export Writer's Version… | `file.exportWriter` |  |  |  | F 3 | :export-writer |
| Export Shooting Version… | `file.exportShooting` |  |  |  |  | :export-shooting |
| Table Read | `view.tableRead` |  |  |  | C T | :table-read |
| Previous Scene Read | `tableread.previous` |  |  |  |  | :table-read-previous |
| Next Scene Read | `tableread.next` |  |  |  |  | :table-read-next |
| Pick What Is Read | `tableread.pick` |  |  |  |  | :table-read-pick |
| Narrator Mode | `tableread.narrator` |  |  |  |  | :table-read-narrator |
| Show Storyboard Frames | `tableread.storyboard` |  |  |  |  | :table-read-storyboard |
| Actor Tints | `tableread.tints` |  |  |  |  | :table-read-tints |
| Table Read Full Screen | `tableread.fullscreen` |  |  |  |  | :table-read-fullscreen |
| Follow in the Table Read | `tableread.follow` |  |  |  |  | :table-read-follow |
| Copy the Follow Link | `tableread.followLink` |  |  |  | C F | :table-read-follow-link |
| Close Table Read | `tableread.close` |  |  |  |  | :table-read-close |
| Lock Ribbon Order | `view.ribbonLock` |  |  |  |  | :lock-ribbon-order |
| Quick Row | `set.quickRow` |  |  |  |  | :quick-row |

## The keys each platform keeps

A command never has one of these (TestNoKeyClashes).

- **Linux/Windows:** Ctrl+Alt+L (locks the screen (GNOME, KDE)); Ctrl+Alt+T (a terminal (Ubuntu)); Ctrl+Alt+D (shows the desktop (Ubuntu)); Ctrl+Alt+Delete (the log-out or security screen); Ctrl+Alt+Up (the workspace above (GNOME)); Ctrl+Alt+Down (the workspace below (GNOME)); Ctrl+Alt+Left (the workspace before (GNOME, KDE)); Ctrl+Alt+Right (the next workspace (GNOME, KDE)); Alt+Tab (the window switcher); Alt+F4 (closes the window); Alt+F2 (the run dialog (GNOME, KDE)); Alt+F1 (the applications menu); Alt+Space (the window menu)
- **macOS:** Cmd+H (hides the app); Cmd+Option+H (hides the other apps); Cmd+M (minimizes the window); Cmd+Tab (the app switcher); Cmd+` (the app's next window); Cmd+Space (Spotlight); Cmd+Shift+3 (a screenshot); Cmd+Shift+4 (a screenshot of a part); Cmd+Shift+5 (the screenshot tools); Cmd+Option+D (shows or hides the Dock); Control+Up (Mission Control); Control+Down (the app's windows); Control+Left (the Space before); Control+Right (the next Space)
- **browser:** Ctrl+N (a new browser window); Ctrl+Shift+N (a new private window); Ctrl+T (a new tab); Ctrl+Shift+T (the tab closed last, again); Ctrl+W (closes the tab); Ctrl+F4 (closes the tab); Ctrl+Shift+W (closes the browser window); Alt+F4 (closes the browser window); Ctrl+Tab (the next tab); Ctrl+Shift+Tab (the tab before); Ctrl+PgDn (the next tab); Ctrl+PgUp (the tab before)

The browser's are Chromium's reserved commands (chrome/browser/ui/accelerator_table.cc, kReservedCommandIds, as its accelerator table maps them; read 2026-11-23), which a page can never take. Firefox's list was not read: it is believed to be like it, unchecked.

## Clashes, per mode and platform

None: TestNoKeyClashes looks at every state of the window with commands of its own (screenplay, Markdown, screenplay, adapting and in production, the source in the middle, the stripboard, the cards, the cards' timeline, the pages, Markdown's Live Preview, the ribbon, Smart, the ribbon, Fixed, with pandoc, an import to fix, the title page, a character, a shared script, Compare, merging, the Table Read, a phone), on each platform, and fails on two commands with one key, a command with a key its platform keeps, and a command with a key vim's or helix's normal mode takes (vim: Ctrl+R, Alt+J, Alt+K, Alt+H, Alt+L, Shift+Alt+J, Shift+Alt+K, Shift+Alt+H, Shift+Alt+L; helix: Ctrl+R, Alt+J, Alt+K, Alt+H, Alt+L, Shift+Alt+J, Shift+Alt+K, Shift+Alt+H, Shift+Alt+L, Alt+S, Alt+,; the menus' shortcuts are seen first, so such a command would take the key from the mode). TestModeKeysDocumented types every key in each mode and fails on one that does something without a line in the modes' table (vim_keys.go), which the help pages are made from.

Found and fixed by this audit:

- The browser: New (Ctrl+N), Title Page (Ctrl+Shift+T) and the middle tabs (Ctrl+PgDn/PgUp) had keys the browser keeps; there they are Ctrl+Alt+N (New Empty: Ctrl+Shift+Alt+N), Ctrl+Shift+Alt+T and Alt+PgDn/PgUp. Ctrl+Tab out of the editor is the browser's too: F6 goes to the panes.
- A Mac: Replace's Cmd+H hid the app; it is Cmd+Option+F there (the Mac's own Replace).
- Inside or Outside… and Character Name… panicked when given from a menu, the palette or the `:` line (no button to open under): their menu opens under the cursor now.
- The markers (Highland's directives) had no keys: they have Highland's, Control+M (Marker…) and Control+Option+M (Next Marker) on a Mac, and Control+Shift+Option+M for Previous Marker beside them; elsewhere the same with Ctrl (Ctrl+M, Ctrl+Alt+M, Ctrl+Shift+Alt+M), none of which clashed.
- The help wrote Ctrl+Next and Ctrl+Prior for Ctrl+PgDn and Ctrl+PgUp.
- A dialog's scroll did not follow the focus (Tab onto a control below the fold left it out of sight): Tab now scrolls every scroll holding the control to it (scrollToFocus, keyboard.go; not when the focused widget keeps Tab, as the editor does).
- The Drafts list hid its focus under the selected draft: a line round it now.
- vim's Ctrl+Shift+R was redo too; it is Ctrl+R alone. The modes' keys with a modifier (modeShortcuts) are checked against what the modes take (TestModeShortcutsListed), on a Mac as Control and Option.
- The Table Read's reading area took the focus and did not show it: a line round it now.
- helix's `/` did nothing (vim's opened the search bar): it opens the search bar.

Left as they are: Ctrl+Alt+J/K/A/G (Adaptation) are AltGr+J/K/A/G on Windows with a keyboard layout that has AltGr (Polish, German …): a letter typed there may be taken by them; there is no Ctrl+Alt key free of that, and Joop chose these. Fyne's file dialogs (Open, Import, Save PDF) are Fyne's own and not audited.

## Tab order and focus in the dialogs, views and panes

TestDialogTabOrder opens on the sample every command whose label ends in … (Accolade's own dialogs), the views (stripboard, cards, Table Read, Compare, the merge view), the side panes and the settings editor, and fails when Tab does not reach one of its shown, enabled controls (a row of tabs is one stop: its tab in front, the arrow keys along it). These must open, or it fails: production.renumber, production.unlockNumbers, production.issue, production.reports, file.drafts, file.saveDraft, file.titlePage, structure.swap, structure.move, write.newScene, view.stages, app.preferences, settings editor, merge view, collab.compare, production.stripboard, view.tableRead, view.cards and the panes (changes, characters, help, notes, outline, production, source).

These open nothing over the window on the sample as it opens, so their Tab order is not audited there (the test fails when this list is not what happened):

- Find (`edit.find`): it needs the search bar (TestFocusIsVisible checks it)
- Replace (`edit.replace`): it needs the search bar (TestFocusIsVisible checks it)
- Adapt Source Passage (`source.adapt`): it needs a passage picked in the Source pane
- Link Up the Script (`source.linkUp`): it needs unlinked scenes that read like the source
- Link Guess for This Scene (`source.linkGuess`): it needs a guess for the cursor's scene
- What Accolade Learned (`source.learned`): it needs the Source pane's list (the panes are audited)
- Tag Selection (`production.tag`): it needs words selected
- Break Down (`production.breakdown`): it needs the Production pane's Break Down stage (the panes are audited)
- Schedule (`production.schedule`): it needs the Production pane's Schedule stage (the panes are audited)
- Page Setup (`preview.pageSetup`): it needs the pages in front (File > Page Setup… is audited)
- Move Strip To (`strips.moveTo`): it needs a strip chosen on the stripboard
- Edit Day (`strips.editDay`): it needs a day chosen on the stripboard
- Add to the Top Row (`view.topRowAdd`): it needs the palette's picker, a layer of the window (TestPalette)
- Apply the Kept Rules Again (`fixup.applyRules`): it needs an imported script with fix-up rules
- Read with OCR Again (`fixup.ocrAgain`): it needs a script read from a PDF
- Rename Everywhere (`chars.rename`): it needs a character opened in the Characters pane
- Merge Into (`chars.merge`): it needs a character opened in the Characters pane
- Add Alias (`chars.alias`): it needs a character opened in the Characters pane
- Relink From Here (`source.relinkFromHere`): it needs a source changed since its lock

TestDialogsFocusVisible checks pixel by pixel that the focus shows on every control Tab reaches in production.renumber, production.unlockNumbers, production.issue, production.reports, file.drafts, file.titlePage, structure.swap, write.newScene, view.tableRead, view.stages on a desktop, and in production.renumber, file.drafts, structure.swap, file.titlePage, view.tableRead on a phone too (Tab scrolls as the window does: TestTabScrollsToFocus goes through the window's key preview). The views and panes are TestFocusIsVisible's, TestDockFocusVisible's, TestStripboardFocusVisible's, TestRibbonFocusVisible's, TestPageViewFocusVisible's and the Characters', Adapt and Save As tests'.

Not opened by the audit:

- Edit Together (`file.share`): shares the script over the network as it opens
- Export Clean Fountain (`file.exportClean`): Fyne's file dialog
- Export Shoot Days (.ics) (`production.ics`): Fyne's file dialog
- Import as Markdown (`file.import`): Fyne's file dialog
- Import with pandoc (`file.importPandoc`): Fyne's file dialog (pandoc)
- Join a Shared Script (`file.join`): joins over the network
- Merge Changes (`file.mergeChanges`): Fyne's file dialog first; the merge itself is audited as "merge view"
- Open (`file.open`): Fyne's file dialog
- Open Bundle Folder (`file.openFolder`): Fyne's folder dialog
- Print (`file.print`): the system's print dialog
- Save PDF (`preview.savePDF`): Fyne's file dialog
- Save Source As (`source.saveAs`): Fyne's file dialog
- Send Feedback (`help.feedback`): the web browser
