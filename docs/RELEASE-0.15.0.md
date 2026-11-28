# Accolade 0.15.0

## The ribbon: nothing is menu-only

Some commands could only be found in the menu bar. Now every one of them has a place on the ribbon as well: on a tab, in a group or at the end of the tab row (New from Structure, the PDF imports, the treatment, outline and storyboard exports, notes and markers, Include File, the structural moves, the pane moves, every Help page and more).

A group's **⋯** opens its **More** tab: a contextual tab, brought to the front with the focus on its first command, that holds the group's less common commands and settings as ribbon buttons.

- **Submenus are groups.** Each submenu (New from Structure, Insert Structure, Theme, Font, Language for New Scripts …) is a group of its own on the More tab, its items as buttons, the chosen one ✓. A menu a level deeper is a ▾ button with the whole menu.
- **Settings show their state:** an on/off setting is ✓ when on.
- **✕ closes it**, first on the tab and always in sight, however far the groups scroll; the tab itself has a ✕, and Esc on the ribbon closes it too. You are back on the tab you came from, on the group's ⋯. Choosing another tab closes it as well.
- **On a phone** the More tab is in the switcher's menu with the other contextual tabs, its Close a finger's size.

## Beats

Looking for beat sheets? Accolade calls them structures, and now they are easy to find.

- **Insert Beats / Structure** is a button in the **Revision** tab's new Beats group (and in the menu as before). Searching the command palette for "beats" or "beat sheet" finds every structure to insert or to start a script from.
- **Three new structures**, in Accolade's own words: **Freytag's Pyramid** (1863), **Seven Key Steps** (after John Truby) and **Six Stages** (after Michael Hauge), with the page ranges each beat usually falls in.
- **Each beat has a purpose and a tension** (1 to 5), in all ten structures: in the Beats readout's tooltip, as "tension n/5" in the list, on the status line and in Help > Structure.
- **The `[[beat]]` marker:** a section with `[[beat]]` on its heading is a beat while no scene is in it, as a synopsis on its own line has always been. The structures write the marker on their beats, and the Notes pane, the counts and the cards all read beats the same way.
- **Every Section Is a Beat:** a setting behind the Beats group's ⋯ that makes every section a beat. It is kept per script, in its `.accolade` file.

## Other formats

Accolade reads and writes other apps' files through Lexington 1.67.2:

- **Fade In in OSF 5.0.** `.fadein` and `.osf` files are written as Fade In 5 writes them (its version, settings and styles), so Fade In opens them as its own current format instead of an older one. `-osfversion 4.0`, `2.0` and `1.2` still write the older versions on the command line.
- **Clean `FADE IN:` and `FADE OUT.`** A script that goes to Fade In or Final Draft and back no longer comes back with force marks on every line that once needed one: a `!`, `@`, `.` or `>` is written only where the line would otherwise read as something else, judged by the script's own language. A script in another language (Esperanto, say) keeps the marks its scene words need, when it is imported and when it is exported as Fountain.
- **Notes stay where they were.** A line of notes before a centred line, a transition or a scene heading used to come back inside that line's text through Fade In or Final Draft. It now comes back as a line of notes before it; in a speech it stays at its line.
- **Trelby keeps centred lines.** Trelby cannot centre a line, so `> THE END <` is written with Fountain's markers and comes back centred.
- **"CUT TO:" needs no blank line after it.** A transition ending in `TO:` right above the next scene's text is a transition again, not a character cue.

## Also new

- **`accolade --version` says Accolade's version** ("Accolade 0.15.0", its commit and date, and the Lexington built in), not Lexington's "dev". Every release build stamps its version and checks it: the Linux tarball, the AppImage, Windows and both macOS builds.
- **The build workflows** use the current majors of GitHub's actions (checkout, setup-go, upload and download artifact, Pages), the ones on Node 24.
