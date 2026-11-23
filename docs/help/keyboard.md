# Keyboard Only

Everything Accolade does can be done without a mouse. This page walks through a day's work by the keys alone; its keys are looked up in Accolade's own list of commands as the page is made, so they are the ones the app has. Ctrl is Cmd on a Mac; a few keys are others there and in the browser ([Keyboard Shortcuts](shortcuts.md) says which).

## Finding anything

- **Ctrl+K** or **F1** — the command palette: type a few letters of any command (the letters in order: “nsp” is Next Source Passage), **Enter** runs it. Every command of the menus and the ribbon is there.
- **F10**, or **Alt** pressed and let go (not on a Mac) — the ribbon's key tips: a letter on each tab, then on each button.
- **F6** and **Shift+F6** — from the script to each pane in turn and back; **Esc** in a pane goes back to the script.
- **Ctrl+PgDn** and **Ctrl+PgUp** — the middle tabs (the script, the cards, the stripboard, the Table Read).
- **Tab** and **Shift+Tab** go through every dialog's controls (a dialog's list scrolls to the one with the focus); **Enter** presses the button in blue, **Esc** closes. A row of tabs is one stop: the arrow keys go along it.
- **Shift+F10** on a pane's tab or a middle tab opens its menu (Move to Top, Open in the Middle, Close).

## Writing

- **Tab** on an empty line starts a character cue; **Enter** after it is the dialogue; **Tab** in dialogue a parenthetical; **Enter** on an empty dialogue line is action. [Coming from Final Draft](final-draft.md) has the whole flow.
- **Ctrl+1** … **Ctrl+8** — the line's element (Scene Heading to Lyrics); Format > New Scene — a new scene, field by field.
- While “Tab: …” shows on the status line, **Tab** takes the name or heading offered, again for the next; **Esc** puts it away.
- **Ctrl+F**, **Ctrl+G**, **Ctrl+H** — find, find next, replace.
- **Ctrl+Shift+P** — the pages beside the script; **Ctrl+Shift+O** — the outline; **Ctrl+Shift+D** — the cards.

## Moving scenes and sections

- **Alt+Up** and **Alt+Down** move the scene, section, speech or paragraph at the cursor past the next one; **Alt+Left** and **Alt+Right** promote and demote it (View > Structure has the rest).
- View > Structure > Move and View > Structure > Swap — a run of scenes or a section moved or swapped by a dialog: pick the first, the last, where.
- With vim's or helix's keys: `]]` and `[[` from scene to scene, `das` deletes a scene and `p` puts it after the next, `mas` then `after 12` moves it; `:move 3..5 after 9` and `:swap 3 with #Act Two` do the same by the scenes' labels ([Vim and Helix Modes](vim.md)).

## Adapting

- **Ctrl+Alt+J** and **Ctrl+Alt+K** — the next and previous passage still to adapt; **Ctrl+Alt+A** — Adapt…: how and where it goes in, **Enter** takes the likely choice.
- **Ctrl+Alt+G** — the scene's guessed link to the source; Adaptation > Link to Source Passage — link the selection to the source passage.
- **F6** reaches the Source pane beside the script, Tab its passages' buttons.

## Production

- Production > Lock Scene Numbers, then Production > Renumber Scenes or `:renumber keep 5,12A` — the numbers locked, then renumbered by a strategy; Production > Omit Scene and Production > Restore Scene (`:omit 4..6`, `:restore 5`).
- **Ctrl+Shift+B** — tag the selection for the breakdown (`:tag props`); Production > Break Down — the breakdown; Production > Reports — reports.
- Production > Stripboard — the stripboard: **Up**/**Down** choose a strip, **Alt+Up**/**Alt+Down** move it, Production > Move Strip To moves it to a day.
- Production > Issue Draft (`:issue`) — issue the draft; Collaboration > Save Draft (`:draft name`) — keep a draft; Collaboration > Drafts — the drafts, to compare.
- View > Table Read — the Table Read: **Right**, **Space** or **PgDn** the next scene, **Left** or **PgUp** the one before, **F** full screen, **Esc** back.
