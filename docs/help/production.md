# Production

When a script goes into production, its scene numbers and pages must stop moving, changes must be marked, and the scenes broken down and scheduled. Accolade does this in the Production menu and the Production pane, on the same Fountain file.

The Production menu is there for screenplays (not for Markdown documents), when you work as Production (Preferences > General > I am working as, or View > Working As). Working as a writer puts it away, with the pane and the production settings; the script keeps its locks, marks and tags, and prints the same.

## The Production pane

View > Production opens a tab in the side panes (see [Help Contents](index.md), Side panes). It walks through the stages, one button each:

1. **Lock** — check the script, then lock the scene numbers and the pages, and issue the White draft.
2. **Revise** — the open revision: its colour, how many pages changed, which scenes.
3. **Break down** — tag what each scene needs.
4. **Schedule** — put the scenes on shoot days.

The pane opens at the stage the script is in. A "Day pack" stage is shown as coming: it is not there yet.

### The check before the lock

Before anything is locked, the Lock stage lists what to look at, each row going to its line when tapped:

- a heading without INT./EXT. or a time of day;
- scene numbers that are doubled, missing or left inside a heading;
- a `/*` or `[[` that is not closed;
- a cue spelled two ways, or a name the adaptation learned as another name for a character;
- scenes over about three pages;
- no title page.

## Scene numbers

### Locking them

Production > Lock Scene Numbers numbers every scene heading (`#1#`, `#2#` …) and records the lock in the script's `/* Production` block:

```
/* Production
scene numbers: locked, 10A
*/
```

From then on a scene you write between 10 and 11 gets 10A once you leave its heading and stop typing. The next one after it is 10B; one between 10A and 10B is 10AA, one between 10 and 10A is 10aA. Preferences > Script > Production can give new scenes A11 instead of 10A, for scripts locked from then on.

Numbers are labels. Accolade numbers only labels like 10, 10A or A11, and never changes any other (`#I-1-A#`, `#PROLOGUE#`).

### Unlocking

Undo right after locking takes the numbers away again. Later, Production > Unlock Scene Numbers… asks:

- **Remove the Numbers** takes out the numbers the lock wrote and keeps those you typed yourself. This is the default while nothing has been issued.
- **Keep the Numbers** ends the lock but leaves every number. This is the default once a draft has been issued: the numbers have gone out.

### Renumbering

Production > Renumber Scenes… numbers the scenes again, with a preview of old and new numbers before anything changes:

- **Preset:** "Clean 1…n (before anything went out)", "After an issue: keep sent scenes" (every number in an issued draft stays), or "Last used".
- **Scenes:** all, the selected ones, from one scene to another, or from the cursor on.
- **Numbers:** keep a list of numbers as they are (`5, 12A, 20-24`), or renumber only the listed ones.
- **A-scenes:** fold them into whole numbers, or keep the letters.
- **OMITTED:** drop them and close the gaps, keep them with their numbers, or keep them and print each run as one line ("12-14 OMITTED").

After an issue, the dialog warns that sides, schedules and pages that went out use the old numbers, and offers "Save a Draft, Then Renumber". The breakdown, the schedule and locked pages follow the scenes to their new numbers. One undo step.

### Omitting and restoring

In a locked script, a cut scene keeps its number. Production > Omit Scene, with the cursor in the scene, turns its heading into `.OMITTED #12#` and keeps the scene in a boneyard under it. Production > Restore Scene brings it back.

## Drafts

Collaboration > Save Draft… keeps the script as it is now under a name, at any time. Collaboration > Drafts… lists the drafts:

- **Compare** two drafts, or one with the script now, scene by scene and word by word, in a middle tab of its own beside the script's. Collaboration > Compare… goes straight there, the newest draft with the script now. The ribbon's Compare tab goes from one changed scene to the next (Next and Previous Change) and closes it. Compare only shows: restoring a whole draft is in Drafts….
- **Restore…** makes the script that draft's text again. The script as it was is saved as a draft first, and Undo takes it back.

A text or source that is the same as one already stored is not stored twice. The drafts are kept with the script; see [where it is all kept](#where-it-is-all-kept).

## Issues and revisions

Production > Issue Draft… issues the script as it is now, with a date, as the next colour in the usual order: White, Blue, Pink, Yellow, Green, Goldenrod, Buff, Salmon, Cherry, then Second Blue and on. The colours are in the [Revision Colours](cheat-revisions.md) cheat sheet.

- **White** is the first issue. It locks the scene numbers first, if they are not locked yet.
- Every issue is kept in the drafts, and the production block notes the current one: `revision: Blue, 2026-11-20`.
- **Changes are never marked in the text.** Accolade compares each issue with the one before. A changed line gets the colour of the issue it changed in; the changes since the last issue are the open revision, in the next colour.
- The editor shows a changed line as a coloured bar in the margin. Preferences > Script > Production "Revision marks since" shows the last issue's changes only, or every issue's, each in its colour.

Once issued, File > Page Setup… has three options for a PDF (Export, Preview PDF) that use the revisions:

- **Revision marks:** a `*` at the right of every changed line, and the revision's name and date at the top of each changed page.
- **Revised pages only:** just the pages with a change.
- **Changed pages on their colour's paper.**

Production's Revised Pages PDF… exports the revised pages only, for that export, whatever Page Setup says.

The header text is set in Preferences > Script > Production ("Revised page header", with `{set}` and `{date}`). FDX export can carry the revision marks too. File > Print…, which prints through a web page, has no marks.

In the pane's Revise stage, Revised Pages PDF… sets this up, and Compare with the Last Issue shows what changed.

### Stages and the two versions

Each stage locks what came before and adds on top of it: the source (locked when you start adapting: [Adapting a Story](adapting.md#the-source-locked)), the **writer's draft** (locked by the White issue), and the **shooting script**, the writer's draft with production's layers: scene numbers, OMITTED scenes, the later issues, the breakdown and the schedule. The window's title says where the script is: **Source**, **Writing**, **Locked** (White issued, nothing changed since) or **Shooting** (a revision after it). A script without a source and with nothing issued shows none.

View > Stages… lists what is locked, oldest first, each with **Compare** (with the script now), and what can be done next.

One script exports both versions. File > Export Writer's Version… is the writer's draft as White locked it (before any issue, the script now), without scene numbers, OMITTED scenes, revision marks, locked pages or Accolade's data; the file's name ends in "- writer". File > Export Shooting Version… is the script now with all of it, as Export… always made. The Export dialog has the same choice (Version), and says what each one is.

### Taking an issue back

- **Undo** right after Issue Draft… takes the whole issue back in one step: its entry in the drafts, the revision line, the page lock it wrote, and the scene lock the White issue made. **Redo** issues it again, as it was.
- **Production > Withdraw Last Issue…** does the same later, for the newest issue only (an older one has a later issue built on it). It asks first. It takes back only what the issue made and is still there: a page lock or revision line you changed since stays, and the dialog says so.
- Withdraw only an issue nobody has received. If its PDF or FDX has already gone out, those copies still say its colour, and the next issue gets that colour again with other changes.
- A script imported from Final Draft with its revisions keeps them as its first issue: that one cannot be withdrawn.

## Locked pages

Production > Lock Pages keeps the pages as the PDF prints them now. From then on:

- text that grows goes on to an A-page (12A, 12B);
- a page whose text is gone gives its number to the page before (12-13);
- a page that gets shorter stays short.

Each page is found by its words, so an edit in another program does not move it. The lock is kept in the production block, with the paper and layout it was made in; the export and preview use that layout while the pages are locked. Every Issue locks the pages again with their A-pages. The first revision's Issue offers to lock them as White printed them.

Production > Unlock Pages… takes the lock out. Preferences > Script > Production "CONTINUED where a scene breaks over pages (PDF)" prints (CONTINUED) and CONTINUED: where a scene goes over a page.

## Breakdown

The breakdown tags what each scene needs: cast, props, vehicles, wardrobe and so on. The categories are Final Draft's (the same as Movie Magic's), with their colours; you can add your own. They are in the [Breakdown Categories](cheat-breakdown.md) cheat sheet.

- **Tag:** select words and choose Production > Tag Selection… (Ctrl+Shift+B), or right-click. Pick a category and an element, or type a new one.
- **Untag** takes the tags off the selected words, or the words at the cursor.
- **Show Tagged Words** colours and underlines the tagged words in the editor.

Production > Break Down… opens the pane's Break down stage: how much is done ("33% broken down · 2 scenes to go"), and for the cursor's scene its length in eighths of a page, its location, its cast (from the cues, numbered) and its elements by category. Next Scene goes to the next scene not done, and Mark Scene Done marks this one.

Tags are not written in the script. Each tag is found again by its words and its scene, so editing elsewhere, even in another program, does not move it. A tag whose words are gone is listed as lost, never dropped.

## Schedule

### The stripboard

Production > Stripboard shows the scenes as strips, in a tab beside the script's. While it is in front the ribbon has a Stripboard tab with its commands (in the Menus look, the Production menu has them). A strip's colour says interior or exterior, day or night (white, yellow, blue, green). It shows the scene's number, location, length in eighths and cast numbers.

- **Shoot days** each have a date, a call time and notes, and end with a bar like "End of Day 1 — 4 6/8 pages". Add Day makes one.
- **Moving strips:** drag them, or select one and use Move Up / Move Down (Alt+Up / Alt+Down) or Move To. Unschedule puts it back in the pool of unscheduled scenes. On a phone, drag a strip by its grip (≡) at its end; a long press shows what you can do with it.
- **Folding days:** the ▾ by a day's head folds it to one band, with its scenes and length; Fold Days folds them all.
- **Group by Location** and **Group by Cast** sort the strips once.
- **Undo** on the board takes back the last move.
- A scene changed in a revision since it was scheduled is flagged on its strip. Checked clears the flag.
- The Script tab returns to the editor; Close closes the board.

Production > Schedule… opens the pane's Schedule stage: how many scenes are on a day, and buttons for the board, the reports and the calendar.

### Calendar

Production > Export Shoot Days (.ics)… writes the dated shoot days as a calendar file, for any calendar app. Days without a date are left out.

## Reports

Production > Reports… makes, from the script as Export would print it:

- **Scenes**, **Locations**, **Characters** and **Cast** (numbered by size);
- **Breakdown sheets**, one per scene, and **Elements**, every tagged element;
- **One-liner**, the schedule day by day;
- **Day out of days**, which actor works which day, with the usual codes ([Day Out of Days Codes](cheat-dood.md)).

Each can be sorted, saved as CSV, saved as PDF or printed.

## Sides

File > Export… can print one character's sides: choose the character, then Full scenes, Cue lines or Lines only. The same dialog can print only the scenes with a tag or colour on their heading.

## Where it is all kept

- **In the script,** in the `/* Production` block: the scene number lock, the current revision, the page lock and the OMITTED print setting. Undo works on them like on any text, and they survive any other Fountain app.
- **In a bundle** (.accolade, .highland, .textbundle, .textpack): the drafts, issues, breakdown and schedule, inside it under `assets/`.
- **For a plain .fountain:** the same files in a copy Accolade keeps in its data folder for that script, which follows a rename or Save As.
- **In a folder beside it:** when you save a bundle as a plain .fountain on the desktop, you can have them in `Name.production/` next to it, to take along or put in version control.

File > Save As… offers .accolade once a script has drafts or a lock. More in [File Formats](formats.md#bundles).
