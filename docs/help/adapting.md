# Adapting a Story

Accolade can keep a story beside the script you make from it: a novel, a short story, a play in prose. You work through it passage by passage. The script links each scene to the passages it comes from, and the source shows what is adapted, what you cut, and what is still to do.

Everything Accolade writes for this is ordinary Fountain: a note in the script for each link, and a boneyard block at the end for cuts and what it learned. Other Fountain apps open the script and print it without them.

The commands are in the Adaptation menu, which is there for screenplays (not for Markdown documents).

## The source

The source is a story in Markdown. Its paragraphs, headings, quotes and list items are the passages you adapt.

### Choosing a source

Adaptation > Choose Source… takes a story as Markdown (.md, .markdown), plain text (.txt), Word (.docx), OpenDocument (.odt) or a web page (.html). Accolade converts it to Markdown and keeps that copy for the script, so the script finds it again after a rename or Save As.

A story you have as PDF, EPUB or another format: open it with File > Import as Markdown… (or Import with pandoc… on the desktop), save it as .md, then choose that.

### Where the source is kept

Accolade looks for a script's source in this order and uses the first it finds. The pane names the one in use.

1. **In the script,** in a `/* Source` block at its end. Adaptation > Keep Source in the Script moves it in, and again to move it out. Off by default.
2. **In the bundle,** for a script saved as .accolade, .highland or another bundle: the source travels inside the file.
3. **Kept by Accolade,** the copy Choose Source… made, in Accolade's data folder (in the browser, its kept settings).
4. **Beside the script:** `story.md` next to `story.fountain`.

Adaptation > Save Source As… writes the source out as a file of its own (the browser downloads it).

### Starting from a story

With a story open as a Markdown document, File > Start a Screenplay from This Source (also a button in the toolbar) makes a new script for it:

- a title page that names the story (`Source: Based on …`);
- the story's chapters as sections;
- the first scene started for the first passage, with its link and the heading the story suggests (its words after INT. selected, to type over).

A saved .md on the desktop stays the source itself, so you can go on writing the story. A script that has a source but nothing written yet gets the same offer in the source pane.

## The source pane

View > Source, or Adaptation > Source Pane, shows the source as a tab in the side panes (at the bottom at first, beside Notes if it is shown; right-click the tab to move it to the top). Each passage is a row.

- **The cursor's scene lights its passages,** and the pane scrolls to them.
- **Tapping a passage picks it,** for Adapt…, Mark as Cut and Link to Source Passage, and makes it where Next goes on from. The cursor stays where it is.
- **A passage a scene adapts shows it under it:** "→ Scene 4" (the scene's number, or its locked one). Tap it to go to the scene, which flashes for a moment. Several scenes: "→ 2 scenes" lists them to choose from. Adaptation > Go to the Passage's Scene does the same from the keyboard, the next scene each time.
- **The picked passage, if still to do,** shows **Adapt…** and **Mark as Cut** under it (Unmark Cut for a cut one).
- **Edit** opens the source as text in the pane; **Done** saves it where it is kept. If another program changes the source file, the pane reads it again.
- **What the source shows,** at the bottom, is how Accolade reads the picked passage: who says what, the scene heading it suggests, what a camera cannot see, the names in it.

On a narrow screen (a phone), the side panes go under the script as one row of tabs, and a toolbar button shows or hides the source.

## Links

A link is a Fountain note in the scene, in the browsers' Text Fragment form. It holds as few of the passage's own words as find it and nothing else in the story: its first words, then after a comma (the range's comma) its last words. Often one or two of each are enough; a short passage is quoted whole when that is no longer. A comma in the story's own words is written `%2C`, so it is not read as the range's.

```
INT. KITCHEN - NIGHT

[[from: #:~:text=She opened,rain.]]
```

The editor draws a link as a small chip, **↗ source** (or **↗ 3 passages** for a link to several; **↗ 2 places** or **↗ source?** when it cannot find its passage alone, see below). Point at it, or press and hold on a touch screen, to read its words as the story has them: "She opened the door … into the rain." Click it, or move the cursor into it, and the link's own text shows, to edit; it is a chip again when the cursor leaves.

Because a link holds words, not a line number, the story can change around it and the link still finds its passage.

### Making a link

Pick the passage in the pane, put the cursor in the scene, and choose Adaptation > Link to Source Passage. The link goes in at the cursor: on an empty line, after a line of dialogue, or on the line under a heading.

When the same words are in the story more than once, Accolade quotes more of the passage, or the words just before or after it (`text=said Anna.-,Yes.`). If two passages and their neighbours are exactly alike, the link cannot tell them apart and the status bar says so.

Links written before read as they are; one is written again only when the source changes under it, as below.

### When the source changes

Because links are short, a change to the story can break one: a new paragraph elsewhere with the same few words makes it fit two places, and editing the words it quotes loses it. So Accolade looks at every link again whenever the source changes: when you edit it in Accolade (as the cursor leaves the passage, at Done, or at Save), when its file changed and is read again, when you open a script whose source changed since it was last saved, and at Adaptation > Update Source Links. Accolade writes a link again by itself only when it is sure which passage it is: the passage is unchanged (the link is lengthened), or its words changed but it is the only passage left between two that did not and still shares a good part of its words (the link is re-anchored to its new words). That is one undo step, and the status bar says how many, for example "2 links lengthened".

When Accolade is not sure (several passages next to each other reworded, a passage moved and edited), it leaves the link as it is and its chip says **↗ check?**. Point at it to read which passage it seems to be now; press and hold it (or right-click) for **Move It There**, one undo step, or **Keep As It Is**.

Some links cannot be fixed. When two passages and the words around them are exactly alike, the chip says **↗ 2 places**; when the passage was taken out of the story, it says **↗ source?**. Point at the chip to read why. The Source pane lists them too: the scene's links under its heading, and the scenes with such links in the line about the whole script.

### Another version of the source

Choose a new version of the story with Adaptation > Choose Source… (a new draft, a heavily edited copy, a translation), and Accolade lines the whole old version up with the new one, passage by passage in order. Links whose passage it is sure of move at once, in one undo step; the rest get **↗ check?** as above. The status bar says, for example, "7 links moved, 3 to check, 1 not found".

Within a session, the pane still shows a passage whose words changed after it was linked in the warning colour, and the count says how many are "changed since linked": check those scenes against the story.

### Old links

Earlier versions of Accolade wrote the first words in quotes: `[[from: "She opened the door to the…"]]`. These are still read. Adaptation > Update Source Links rewrites every link it can find in the new form, in one undo step. Links it cannot place are left as they are.

### Linking up a script written without links

A script written before it had a source has no links. Accolade reads its scenes against the source, in order (the rarer words they share, and where the scenes around them are), and guesses where each comes from. A guess is never written into the script:

- A scene it is fairly sure of gets a faint **↗ link?** after its heading (just **↗?** where the line has no room). It is not text: the cursor passes it. Tap it (or Adaptation > Link Guess for This Scene…, Alt+L; in vim mode `gs`) for **Accept** (the link goes in, one undo step), **Adjust…** (the guess picked in the source pane, to pick the right passage and link it) or **Not This** (not offered for that scene again). A scene it is not sure of, or one invented for the script, gets nothing.
- The source pane labels its chapters (or, without chapters, the stretches the scenes come from): "adapted (scenes 3–9)", "adapted? (scene 4)" when only guessed, and "not adapted yet" after the last of them. Next goes on from there, not from the top.
- A part of the source skipped before or between adapted ones is offered, one at a time, on a quiet line in the pane: "Prologue (9 paragraphs) isn't in the script. Mark as Cut?"
- Adaptation > Link Up the Script… lists all the guesses and skipped parts at once, to check and write in one go (one undo step).

## Statuses

Every passage that is not a heading or a break has one of three statuses, shown as a mark at its left:

- **adapted** (a solid green mark): a scene links to it;
- **cut** (a dashed red mark): you cut it on purpose;
- **to do** (a thin grey line): neither yet.

The count at the top of the pane reads like "50% adapted · 1 cut · 3 to go". The percentage is of the passages you did not cut.

### Cutting a passage

Pick it and press Mark as Cut in the pane, or choose Adaptation > Mark Source Passage as Cut. The same command on a cut passage takes the cut back. Cuts are kept in the script's `/* Adaptation` block under `cut:`, so they travel with it. Linking a cut passage takes its cut away.

## Next, Previous and Adapt…

Next and Previous find what to adapt; Adapt… writes it. Nothing goes into the script until you say how and where.

### Next and Previous

Adaptation > Next Source Passage (Ctrl+Alt+J), or Next in the pane, picks the next passage still to do (neither adapted nor cut), in the story's order, and scrolls to it. Previous Source Passage (Ctrl+Alt+K), or ◀ in the pane, picks the one before. Neither writes anything.

Where they go on from is what you are looking at. The line under the count says which ("after: “…” · picked"):

1. **The passage you picked** (or Next went to), as long as the cursor has not moved since. Pressing Next again goes on from there, so you can step through the passages still to do.
2. **Else the cursor's scene:** the last passage it links to.
3. **Else the nearest scene above the cursor** that links any: its last passage.
4. **Else the passage at the top of the list.**

So you can start in the middle of the story, adapt scenes out of order, or come back to a script half done: Next goes on from where you are, not from the start. When none is left that way, the pane says so and offers to go round to the first (or last) still to do; Next (or Previous) again does the same.

### Adapt…

Adapt… under the picked passage, or Adaptation > Adapt Source Passage… (Ctrl+Alt+A), opens a dialog (on a phone, a sheet at the bottom). It suggests how and where, with the likely choice of each already chosen and why:

- **How:** Start a new scene (a heading and the link); Draft a scene from the passage (also its lines under their cues, the rest as action, which you can edit before it goes in); Add to the current scene (the link at the cursor); Mark as cut; Skip for now (it stays to do, and the next one is picked). Adding to the current scene is the likely one when the story's scene goes on from the passage that scene adapts, or the scene adapts nothing yet; otherwise a new scene, and the dialog says why (for example, the story starts a new scene there: a chapter heading, a break, "The next morning").
- **Heading:** the place and time the story suggests for the passage (`EXT. GARDEN - MORNING`), with the other likely ones in its list; edit it as you like.
- **Where** a new scene goes: in the story's order (after the scene adapting the passage before this one, and inside its chapter's section if the script has it), at the cursor, after the cursor's scene, or at the end. The likely one is the story's order, with the scene it goes after named.
- **What goes in:** exactly the text that will be written, to edit. If the story enters a new chapter the script has no section for, the chapter's section comes first.

Enter takes the choices as they are. Esc cancels. Up and Down change a choice; Tab goes on to the next part. What goes in is one undo step, and only added: a co-writer editing the same script at the time loses nothing. Adapting a passage you had cut takes its cut away in the same step.

A heading you write for a new scene teaches Accolade that place (see What Accolade learned), the same as one you write in the script.

## Sections from the source

Adaptation > Source Headings as Sections writes the story's headings into the script as Fountain sections (`## Chapter 3`), at the same levels. If the script has none of them yet, they all go in as an outline at the cursor's line. Otherwise each missing one goes in its place among those already there. A heading is never written twice. One undo step.

## Suggestions while writing

With a source, Accolade offers the next piece of the story in the status bar as you write, as "Tab: …". Tab takes it, in one undo step; Tab again offers the next one; Esc puts it away for that line.

- **After a cue** (or on the empty line under it): that character's next line in the passage, with how they say it as a parenthetical.
- **Typing a cue:** the names of the passage's speakers first.
- **On an empty line:** the next piece not yet in the scene, a line of dialogue with its cue, or the narration as action.
- **Typing a scene heading:** INT. or EXT., the place and the time, as read from the passage.

The suggestions come from the passages the scene links to, else the one Next picked. On a phone, a small "Tab" button before the hint does the same as Tab. Preferences > Script > Adapting "Suggest from the source while writing" turns them off.

Accolade reads the story with word lists for English, Dutch, Portuguese, Spanish, French, German and Esperanto. It is a heuristic reader, with no AI: it gets most lines right, and some speakers wrong. It needs no network.

## What Accolade learned

As you adapt, Accolade learns from what you do, quietly, as part of your own undo step:

- **A name:** you rename a cue Tab put in (TOM to THOMAS), so the story's name becomes another name for that character.
- **Not a name:** you throw away a cue Tab put in twice (Monday, Lord), so the word is no character.
- **A place:** you write your own heading for a linked scene, so that place gets your heading from then on.
- **A skipped link:** a guessed link you turned down (Not This) is not offered to that scene again.

It keeps them in the script's `/* Adaptation` block:

```
/* Adaptation
characters: ANNA = Anna, Annie, Miss Kovač
not characters: Monday, Lord
places: the house = INT. KOVAČ HOUSE
cut:
  #:~:text=It was late.
*/
```

Adaptation > What Accolade Learned… opens the list in the pane. Each lesson has a button that takes it away, and it is not learned again in that session. You can edit the block by hand too. Nothing is learned while you edit the script together with others.

See also: [Fountain Reference](fountain.md#source-links), [Markdown Syntax](cheat-markdown.md), [Accolade Blocks & Links](cheat-blocks.md).
