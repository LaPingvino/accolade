# Editing together: review and design (2026-11-21)

Joop asked after "the edit together link fix", and LOOP's item "review collaboration" (2026-11-20) asks four
questions of it. This file says what works today, what was broken, each question answered with a decision and
its reason, and what is left. Overnight 21 Nov Joop said not to wait for him before code ("we can revert if
necessary"), so the decisions below were made, and the first part built, without him.

## How it works

- **internal/collab** is the whole of it, the same code on the desktop and in the browser.
  - `Doc`: an RGA text CRDT of runes. Each rune has an ID (a Lamport counter and the replica that made it) and goes
    after the rune it was typed after. Deleted runes stay as tombstones.
  - `Session`: shares one Doc with peers, plus a few last-writer-wins settings ("meta": language, format,
    markdown, each peer's name). A hello carries the sender's version; the other side sends the ops it lacks.
    New ops are passed on to the other peers, so a chain of peers works.
  - `Key`: 32 random bytes. The room (a tracker info hash) and the AES-GCM key are derived from it. Every message
    and every WebRTC offer/answer is sealed with it, so a tracker sees an opaque hash and opaque data.
  - `Network`: peers find each other through public WebTorrent trackers and talk over WebRTC data channels
    (pion on the desktop, the browser's own in the web version).
- **collab_window.go** ties a window to a Session. Collaboration > Edit Together… makes a key and shows the link
  `https://lapingvino.github.io/accolade/#k=<key>`. The fragment never goes to a server. Join a Shared Script…
  takes the link or the bare key. The web version opened with `#k=` joins by itself.
- The editor's change goes to the Doc as ops (`shareEdit`). A peer's ops change the editor where they apply, and the
  cursor moves with the text (`applyRemote`). Each change goes into the window's Changes history under the
  peer's name.
- A hidden browser page lets go of its peers after two minutes and finds them again when it shows. The hello's
  catch-up brings both sides up to date.

Both trackers answered a WebSocket upgrade (101) when probed by hand on 2026-11-21. Whether they still relay offers
was not checked: the tests use a fake tracker and pion's virtual network, never the internet.

## What was broken (found 2026-11-21), worst first

1. **Opening, New or Import in a shared window replaced everyone's script.** File > Open/New/New Markdown,
   Import, Start a Screenplay from This Source and the browser's draft restore all call the editor's `SetText`.
   That goes through `shareEdit` as one big edit: every peer's text was deleted and replaced by the new file. The
   document type and the language went with it too (meta). In the browser, Open into a joined window only asked
   "Replace the script?". It did not say that this replaces it for everyone.
2. **Typing during a peer's catch-up could delete the peer's edit.** `Receive` changed the Doc on the network
   goroutine, and the editor caught up later in `fyne.Do`. A keystroke in that gap diffed the old editor text
   against the newer Doc, and so deleted what had just come in, for everyone. It was worst while joining, which
   takes many messages.
3. **Undo undid the co-writer's typing.** A peer's change was an ordinary undo step in this window. Ctrl+Z took the
   last one away, whoever made it, and that deletion then went to everyone. Older local steps kept their old
   positions, so after a peer's edit above them they undid the wrong text.
4. **Joining in the browser overwrote your own kept script.** The web version keeps one script per browser
   (draft.json). Opening a share link skipped restoring it, emptied the editor to join, and a second later kept
   the shared script in its place, so your own unsaved script was gone. A reload of a joined page with nobody else
   online (or the watchdog's automatic restart) came back empty. The script lived only in the peers' memory.
5. **A whole-text action was one edit from the first change to the last.** `SetText` kept only the common start
   and end. Renumber, Lock, Update Source Links, reloading a bundle or a learned-notes write that touched lines
   far apart deleted and re-inserted everything between them. A co-writer's concurrent typing in that stretch
   then landed beside text it was no longer part of, and their deletes there were undone.
6. **Catch-up could leave a gap for good.** The version a hello sends is each replica's highest insert counter
   seen. Ops can be applied out of a replica's order (an op waits for the rune it goes after). A link that broke
   mid-catch-up could therefore leave a peer with a replica's op 52 but not its op 40. Every later hello then
   said "I have up to 52", so op 40 never came. The texts diverged silently.
7. **A peer whose connection failed before it opened was never tried again.** `answer` reserves a peer, and
   nothing released the reservation when ICE failed. The status bar also counted reserved peers as connected
   ("1 other" with nobody there).
8. **The link only worked from GitHub Pages, and only once.**
   - A web version served elsewhere (a local test deploy, a fork's Pages) still made github.io links.
   - The host's own page never had `#k=` in its address, so a reload left the session.
   - A link pasted into a tab that already had Accolade open changed only the fragment, and nothing listened for
     that.
   - A key with a chat app's trailing `.` or `)`, or wrapped onto two lines, was refused.
   - On the desktop, a link given as an argument (`accolade 'https://…#k=…'`) was opened as a file name.
9. **"Another version" did not say whose.** A peer of another protocol was told apart (good), but the status bar
   could not say which side should update.

## Fixed in the first part (branch collab)

- **(1)** A shared window that gets another document leaves the session first: Open, New, New Markdown, Import,
  Start from Source and restore. The peers keep the script, and this window says it left. A shared window is never
  "replaceable": on the desktop, Open goes to a new window. In the browser, the question says the shared script
  stays with the others.
- **(2)** The Doc changes only on the UI goroutine. `Session.Run` is the executor (the window passes `fyne.Do`),
  and everything a received message does to the Doc runs in it, in order. In that same step the editor is brought
  up to date, so no keystroke can fall between a peer's edit reaching the Doc and reaching the editor.
- **(3)** A peer's change is applied outside the undo history (`Buffer.Remote`). The local undo and redo steps are
  moved past it, so Undo undoes only your own steps, at the right place. A local step that a peer's change
  overlaps cannot be undone faithfully, so it is dropped from the history, together with the steps before it.
- **(4)** While shared, the browser keeps the shared script as its CRDT state (`shared/<room>.json`). A script
  you joined is kept only there, not in draft.json: your own kept script stays as it was, and comes back without
  the link (Stop sharing takes `#k=` out of the address, so the next reload restores it). A script you shared is
  yours, so draft.json follows it too, and Stop sharing writes it at once; a mark beside the state
  (`<room>.own`) says so across a reload. The browser's local storage is a few megabytes for everything, so it
  keeps one room's state (starting another removes the others) and none above 1 MiB. Joining a
  room this browser (or this desktop) was in before starts from the kept state: the text shows at once, even
  with nobody online, and merges with the peers without doubling anything. Each run uses a new replica name, so
  two tabs never share one.
- **(5)** `Doc.SetText` makes the minimal edit. It trims the common ends, then diffs the middle by lines and the
  changed lines by runes (Myers, bounded). Every whole-text action is therefore minimal without changing any of
  them. A peer's change goes into the editor the same way, hunk by hunk, so cursors in between stay put.
- **(6)** A hello also carries how many inserts of each replica the peer has. Where that differs from ours, all of
  that replica's inserts are sent; the receiver drops the ones it already has. This is protocol 4.
- **(7)** A reservation is released when its connection fails or closes before its channel opens. The status bar
  counts open channels only.
- **(8)** The link is the page's own address in the browser (origin and path, plus `#k=`), and github.io on the
  desktop. While shared, the page's address carries `#k=`, so a reload rejoins (from the kept state). Leaving
  takes it off. A `hashchange` to another key joins it (after the usual question). `ParseKey` finds the 43 key
  characters after `k=`, ignoring what surrounds them and any white space inside. A desktop argument with
  `#k=` joins.
- **(9)** The hello carries the app's version. The status bar then names the other side's version and says which
  one is older.

## The four questions

### (1) Markdown documents on their own

**Decision: they work with what is there, and a test proves it.** The document type is a shared setting
("markdown"), so a joiner's window switches mode. The preview, the outline and the cheat sheet are local views of
the shared text. The Markdown commands (headings, lists, links, tables) are ordinary edits, and with (5) each one is
a minimal edit. Each peer saves the document under its own name. The per-file "document-types" record is written on
that peer's save, as for any file.

### (2) A script and its source, together

**Decision: one session holds several named Docs, "script" and "source", over one connection. Each has its own
RGA and version.** Each message names its Doc, and a hello carries a version per Doc.

Reasons:
- The source lives in a different place for every peer (a kept copy, a file beside the script, inside a bundle, or
  in the script). Only its text can be shared, not where it is. Each peer keeps the shared source where its own
  window keeps one, and a joiner with none gets a kept copy.
- Forcing the source into the script (`/* Source */`) would change the writer's file. It would also make every
  passage edit part of the script's history and undo.
- With named Docs the source pane's Edit/Done becomes live editing of the source Doc. Links stay anchored by
  words, as now.

Needed first: a novel-sized source (≈500K runes) in today's Doc costs a node, a map entry and an op per rune,
roughly 150 bytes per rune, about 75 MB. That is too much for the browser. So the Doc first gets run-length nodes:
a run of consecutive IDs is one node, split when someone types inside it, and kept ops are runs. Benchmark it on
the PERF-SOURCE fixture before building the source sharing.

**Built (step 2).** `Op` is now a run (the wire's and the kept state's format already were, so neither the protocol
nor the state's version changed): an insert of consecutive IDs, each rune after the one before, or a delete of
consecutive IDs. A node is such a run, all deleted or none; it is split where someone types inside it or deletes
part of it, and it grows in place when its writer types on at its end. Each replica's nodes are found by counter
in a sorted index (no map entry per rune); the kept ops are runs that grow in place too. `Apply` takes a run that
is here in part (always its first runes: each waits for the one before) and keeps the part that is not, and
returns what was new, to pass on. A delete waits for those of its runes not here yet. `Missing` cuts a run at the
peer's version. The Doc keeps the text it last made or set (`SetText` knows it), so a key is compared with it
without the novel made again, and `Edits` finds the common ends by bytes before it turns the middle into runes.
The old Doc of a node per rune is kept in oracle_test.go: `TestRunsAgreeWithRunes` gives both the same ops, cut
at random, shuffled, repeated and out of order, and their text, version, counts and missing runes agree at every
step (300 seeds; 5000 were run once). It found a delete waiting for its first rune while a later one was there.

**Built (step 3, protocol 5).** A session holds Docs by name: `"script"` and any other (`DoIn`, `LocalIn`,
`Has`). A message names its Doc (`"d"`, none: the script's); a hello carries each Doc's version and counts
(`"docs"`), and the peer answers with each Doc's missing ops, a Doc it lacks whole. `OnRemote` says which Doc
changed. The kept state keeps every Doc (`"docs"` beside `"runs"`, which old readers ignore, so its version
stays); in the browser a state over 1 MiB is kept with the script's Doc alone (`StateOf`), and a reload then gets
the source from the peers.

The source (collab_source.go): a source kept outside the script goes in a Doc of its own, `source:<id>`, which the
shared setting `source` names (last writer wins). Only the peer who shares the script makes it, from its source as
it is, or a peer who picks another source while shared (Choose Source, Keep Source in the Script turned off); Keep
Source in the Script turned on clears the setting, the block then travelling in the script. A peer joining, or
taking up a kept state, takes the Doc as it is and never sets it from its own copy, so nothing doubles. A peer
with no source of its own keeps the shared one for the room (`sources/shared-<room>/source.md`, one copy however
often it rejoins), written as the Doc is whenever editing pauses (with the kept state); a peer with one (the sharer:
in its bundle, a kept copy, beside the script) keeps it there. A source Doc the setting no longer names (another
source picked, or two at once) is let go on every peer that hears of it (`Session.Drop`): not kept, not sent to a
joiner, and its ops are not taken again from a peer that has it still. The
window's source document (document.go, MIDDLE.md) is the Doc's editor: `sharedAs` names the Doc, `changed` sends
each key (`Doc.SetText`, 3 ms a key on the novel), `followShared` brings the peers' edits in outside its undo
history, and the Source pane reads its passages from it. The pane's Edit/Done and the source in the middle
are live for the peers; the sharer's own home is written on its Save or Done, as for its own edits, so until
then its source is marked unsaved (•). The script's Characters and Notes are blocks in the script's text
(characters_block.go, notes.go): they travel in the script's Doc already, nothing to add.

Old peers: protocol 4 unmarshals the hello into its own message before it looks at `p`, so the fields it knows
keep their JSON types and the new ones have keys of their own; it reads `p: 5` and says "another peer's Accolade
(…) is newer than yours: update yours"; this side says the other is older and needs updating. Nothing either sends
is taken by the other. Tests: `TestNamedDocs` (a chain of three, a joiner gets each Doc, edits say their Doc),
`TestJoinMidEdit` (a peer catching up while the other types in a long source), `TestStateDocs`, `TestMixedProtocols`
(a frozen v4 message unmarshals the hello and sees p=5; a v4 hello and its edits are not taken nor relayed; a
session of protocol 4 and one of 5 tell each other apart); in windows `TestSourceTogether` (the sharer's source
beside the script reaches the joiner as the room's kept copy; edits both ways; Undo only one's own; the pane's
passages; the sharer's Save writes the file beside the script with the joiner's edit), `TestSourceJoinMidEdit` (a
third window joins through the second while the first types in the source key by key), `TestSourceRejoin` (a peer
takes the session up from its kept state, the source with it, and meets the other: not doubled), `TestDrop`, and
`TestSourceOldPeer` (a v4 hello: the status bar says older, its edit not taken). `TestSourceTogether` also goes
through the joiner's Source pane: its passages, Edit typed in and seen by the sharer, Done, then the sharer's edit
in the pane's passages; and the room's copy as the Doc is. A source Doc with no op yet is one whose ops have not come: the document neither follows it nor
sends to it, so a browser that kept its state without the source (over 1 MiB) and reloads with nobody online keeps
the room's copy, and an edit made then stays its own and is replaced when the peers' Doc comes; an empty source is
therefore not shared (the status line says "yours only"). Not done: a peer's own different source when it
joins is replaced by the shared one for that session (its file is written only if it saves). Two peers picking a
source at the same moment (built 2026-11-23): the setting's last writer wins on both; the winner hears of the other
pick as a setting that lost (`Session.OnLost`, a peer's value older than its own) and lets that Doc go, so neither
keeps or sends it; the loser (the window that made the Doc, `shared.madeSource`) is told on the status line who
picked theirs and that Choose Source… again shares its own instead. The file it picked is never touched (Choose
Source keeps a copy); its window follows the shared one from then on. `TestSourcePickRace` (it failed before: the
winner kept the lost Doc). What a
peer's key costs the window it reaches is measured in step 8 below: each message used to read the source pane's
passages again; now the pane reads them once the peer pauses. The breakdown and the schedule are step 4.

Benchmarks (internal/collab/bench_test.go, ~100k words, 572k runes; "edited" is that after 3000 scattered edits by
two replicas, key by key; 3 runs each, this VPS):

| | before | after |
|---|---|---|
| heap per rune, pasted / edited | 178 / 191 B (≈100 MB) | 6.5 / 7.9 B (≈4 MB, the kept text included) |
| a peer joining, catch-up through a session, pasted / edited | 1.26 s / 1.21 s | 44 ms / 66 ms |
| a session taken up from the kept state, pasted / edited | 0.76 s / 0.80 s | 39 ms / 39 ms |
| a key in the middle, `SetText` with the editor's text | 21 ms / 33 ms | 3.0 ms / 3.0 ms |
| a key in the middle, `Insert` alone | 2.5 ms / 4.5 ms | 18 µs / 130 µs |

The "before" `SetText` loop also turned the novel into runes and back each key (≈2 ms of it); the "after" one makes
the editor's new text as a string, which is most of the 3 ms that is left. Building the edited novel itself (3000
edits) took 49 s before and 0.2 s after.

### (3) What lives outside the text

Decided per item, with its merge rule:

| What | Decision | Merge rule |
|---|---|---|
| Language, script format, document type, names | shared setting (meta) | last writer wins (clock, then replica) |
| Breakdown tags (breakdown.json) | shared CRDT map (built: step 4 below) | per key (a category, an element, a tag, a scene done): last writer wins; a removal is a tombstone, so two people tagging different things keep both |
| Schedule (schedule.json: days, strip order) | shared CRDT map (built: step 4 below) | per strip: (day, fractional position), last writer wins, so a strip is never lost or doubled by concurrent moves; day definitions per day ID, last writer wins |
| Locked pages, issues' data in the production block | in the text already, or a snapshot sent on change | a single production act (Lock, Issue) by one person, last writer wins as a whole value |
| Drafts store (assets/drafts/) | local | each peer's own history; Save Draft and Compare work on what that peer has. Too big to send live, and a draft is a moment of one person's copy |
| The kept source copy, the source choice | local, the content shared by (2) | — |
| Per-file settings (document type, source file) | local, written on each peer's own save | — |
| Learned notes, adaptation and notes blocks | in the text | as the text |

Sidecar maps do not go in meta. Meta is sent whole in a hello, which a browser drops above ~16 KiB. They get
messages of their own, batched like the text's runs.

### (4) Actions on the whole text, and the "not in a shared session" guards

- **User actions** (Renumber, Lock, Unlock, Update Source Links, Source Headings as Sections, the keyboard flow, a
  suggestion taken, a bundle re-read): no change is needed in each of them. Fix (5) makes every change minimal
  where it reaches the Doc, so one rule covers them all. Each stays one undo step for the peer who did it, and (3)
  keeps the other peers' Undo out of it. The Characters pane's Rename Everywhere and Merge Into… make the least
  edits themselves (each name's runes, the lines and words of the list and the learned notes that changed) and
  give them to the session as such before the editor makes them (`shareEditsFirst`): `Doc.SetText`'s own diff
  gives up past 2000 changed lines (scriptdiff's `maxEdits`) and then replaces the whole stretch between, which
  a lead's rename in a feature reaches. TestRenameTogether renames 1200 places while a peer types beside a cue
  and in the list (it fails without `shareEditsFirst`). Other whole-text actions spread that wide would need the
  same.
- **Automatic writes** (auto A-numbers in a locked script, learned notes) run on every peer that sees the trigger,
  so two peers would write the same thing twice, however small the edit. **Decision: a single-writer rule, not
  a permanent guard.** Only the peer who wrote the text an automatic write follows from may make it. The Doc knows
  who wrote every rune (`Doc.Writer`): the new heading's runes say whose scene it is to number, and the
  learned-from line says whose lesson it is to write. **Built** (`wroteLine`, collab_window.go): a line is the
  writer's whose replica wrote its first rune, the same answer on every peer. `numberNewScenes` numbers only the
  headings this peer wrote; `learnFromWriting` learns only from a heading or a cue line this peer wrote (a
  co-writer's is marked known, without a lesson; a cue Tab put in is watched only where Tab was pressed). The two
  guards are gone. A line whose writer has left, or wrote it in an earlier run (each run is a new replica), is
  nobody's: it is not numbered until Lock Scene Numbers, nor learned from. Two peers adding a scene at one place at
  the same moment, before either hears of the other's, both give theirs the same A-number (each numbers its own
  as the only new one). Built 2026-11-23: after a peer's edit the numbering runs again (`applyRemote` schedules
  it), and of two headings with one number the later in the text is numbered anew, by its writer only
  (`wroteLine`): the same answer on every peer, so one of them writes it, and the order comes out right (12A, then
  12B). "The later writer" by clock was not taken: it could renumber the earlier heading into a number that sorts
  after the later one. `TestSceneNumberCollision` (two windows apart, each numbering its own 1A, then meeting; it
  failed before). TestSceneNumbersTogether and TestLearnTogether fail when every peer may write (checked). The other "not shared" checks were looked at: the breakdown, the
  schedule, the revision marks and the scene-number follow write sidecars or the margin, never the text (sharing
  them is step 4); `shareEditsFirst` (Characters' Rename/Merge) and the middle tabs' "yours only" are not guards.
- **Undo**: only your own steps (fix 3).

### Protocol versioning

- Protocol 6 (2026-11-23): shared maps (below). A peer of 5 would read the hello and the maps' messages without
  error and silently miss every breakdown and schedule change, so by the rule below 6 is a new protocol: 5 and 6
  tell each other apart and say which side is older. Presence (`"h"`, and its anchors `"a"`, `"f"`) is ignorable
  and changed nothing.
- Every hello carries `p` (the protocol) and `app` (Accolade's version). A peer of another protocol is listened to
  for nothing more, and the status bar says which side runs the older Accolade.
- Peers before protocol 4 show "another version" as before. They read the v4 hello, see p=4, and stop. Protocol 5
  (named Docs) is the same for protocol 4 peers: the hello's old fields keep their types, the new ones have new
  keys, so a v4 peer reads it, sees p=5 and says to update.
- **Rule**: any change to what goes over the wire (a field whose meaning changes, a new message kind, the named
  Docs) bumps `Protocol`. A field that old peers may simply ignore does not.
- The kept state files carry their own `"v"`. A file of another version is not read, and the session joins from
  the peers as before.

## What is tested, and what is not

- Tested in-process: two windows over an async loopback (each side's messages delivered in order on a goroutine of
  its own, the UI turn held by the test): joining, typing both ways, the language, pausing, typing during a
  catch-up, Undo, a whole-text action against concurrent typing, another document leaving the session, a rejoin
  from the kept state, Markdown together, the link, the host's draft. The collab package has the gap, the
  executor, the kept state, the minimal edits and mangled links; the buffer has Remote's rebase (random too).
- `TestTypingDuringCatchUp` and `TestCatchUpGap` were checked to fail on the old code. The others test the new
  behaviour only.
- **Not verified in a browser**: `setPageKey`, `pageBase`, the `hashchange` join and the draft guard are only
  compiled for js/wasm (no headless-browser run of the web version was made); nor the shared source there (the
  room's copy in the browser's storage, the state kept without the source past 1 MiB).
- **Not tested**: the released reservation of fix (7) (no vnet test of a peer failing before its channel
  opens), the status text of a version mismatch, and the desktop `#k=` argument.
- `TestLongScript` (internal/collab) was seen to fail under `-race` on this machine (it times 70 000 inserts); it
  passed in the race gates of 21 Nov evening, before runs and after.

## Step 4: the breakdown and the schedule as shared maps (built 2026-11-23, protocol 6)

- **internal/collab maps.go:** a session holds maps by name, each of string keys to values, the last writer winning
  per key (a Lamport clock over the map, then the replica), a key taken away kept as gone. `MapSet` sets and takes
  away keys here and sends them; a peer's entries come in messages of their own (`"x"` the map, `"e"` the entries,
  batched under 8 KiB as the text's runs) and are passed on along a chain; `OnMap` says which map changed. A hello
  carries each map's version (`"xs"`: the highest clock seen of each replica), never the map, and the other side
  sends the entries past it, oldest first. The kept state keeps the maps (`"maps"`, which an older reader ignores).
  `TestSharedMaps`: a chain of three, two writing one key apart and meeting, different keys both kept, the kept
  state, entries shuffled and doubled (the same map), `OnMap`.
- **The breakdown (collab_extras.go):** a key per category (`c <id>`), element (`e <id>`), tag (`t <the tag>`: its
  element, scene and words) and scene marked done (`d <scene>`). Every change of the breakdown goes through
  `saveBreakdown`, which gives the map the keys that changed (`shareBreakdownChange`); a peer's change makes the map
  the window's breakdown and writes it where its extras are (`takeSharedBreakdown`, through `storeBreakdown`, which
  never writes over a file that could not be read). One element made by two at once (the same category and name,
  each its own random id) is read as the one with the first id, the other's tags joined to it. The sharer puts its
  breakdown in as it starts; a joiner, or a kept state taken up, takes the map and never puts its own in.
- **The schedule:** a key per day (`y <id>`: date, call, notes and its place among the days), per strip (`s <the
  scene>`: its day and place there, its start, sum and set), and the pool's order (`p`). A place is a number between
  its neighbours' (`placesBetween`: a place still in order is kept, a new or moved one goes halfway), so a move
  changes the strip moved only. Every change goes through `writeSchedule`; a peer's change replaces the window's
  schedule and forgets the board's Undo (it undoes one's own last change only). A strip whose day was taken away
  goes to the pool.
- Tests: `TestBreakdownTogether` (a's tag to b, who has none; b's Untag back; tags made apart, one element made on
  both sides at once: all kept, one element, the files written), `TestScheduleTogether` (days and strips to b; a
  strip moved on each side apart: both kept; a day taken away), `TestPlacesBetween`.
- Not done: the drafts store, the locked pages and the issues stay as the table says (local, or in the text). Two
  people adding a shoot day at once get two days. Untag and a tag of the same words at once: the tag's key is its
  words, so the last writer wins. A map's catch-up goes by the highest clock seen of each replica, without the
  counts the Docs have (found item 6): a link that breaks mid catch-up while a replica's entries also reach the
  peer another way could leave an entry out until that key is written again. A joining window starts its breakdown
  and schedule from the maps (empty while they have nothing, `joinExtras`), so what it had of its own is never put
  in; it is written over where that window keeps them.

## Presence: each other's places (step 6, built 2026-11-23)

- Presence (`collab.Presence`, presence.go) carries the cursor and the selection's other end as anchors in the
  script's Doc (`"a"`, `"f"`: the ID of the rune each comes after, `Doc.Anchor`, placed with `Doc.Place`; a deleted
  rune still places, where it was). A peer whose runes are not here yet is drawn at its line, as before. A window
  says its place when its cursor's line changes (at once), when its selection changes (once it has rested 150 ms),
  and every 30 s.
- The editor (internal/editor peers.go) draws each co-writer as a bar in their colour in the gutter's last column
  (the revision mark's, which theirs covers while they are there), on every row of their line (a folded line's: its
  heading's), and their selection tinted. Their places are buffer marks (`Buffer.SetMarks`), so this side's typing
  moves them with the text at once; they are placed anew from the anchors whenever presence comes and after each
  peer's edit (`showPeers`). The pointer over a bar (or a tap on it) names them in a tip; the tap leaves the cursor
  and the focus as they are. The outline and the cards (the pane and the full view) show a dot in their colour on
  their scene. Nothing is drawn per key that was not before: with nobody here the editor's drawing is unchanged.
- Tests: `TestPresenceAnchors` (collab), `TestMarks` (buffer), `TestPeers` (editor: the bar, its line after typing
  above, the tap, a fold), `TestCursorsTogether` (two windows: line, selection, typing above, the dots, leaving);
  shots `TestCursorsScreenshots`.

## Trying it on two devices (by hand)

The tests run two windows in one process over a fake connection; the real trackers, WebRTC and the browser build
were never run together here. A script for two people (or one person with two devices), about fifteen minutes:

1. **Desktop and browser.** On the desktop open a screenplay with a source beside it (or File > Open Sample).
   Collaboration > Edit Together…, Copy the link. Open the link on a phone or in another browser. Within half a
   minute both status bars say "Shared · 1 other", and the Session tab lists the other.
2. **Typing.** Type on each side in turn, then both at once in the same scene. Both texts stay the same; each sees
   the other's bar in the margin on their line, their selection tinted, and a dot on their scene in the outline.
   Point at (or tap) the bar: their name. Undo on one side takes back only its own typing.
3. **Follow.** Session > Follow on the phone; move the desktop's cursor through the script: the phone follows. Type
   on the phone: it stops.
4. **The source.** Open the Source pane on both; Edit on the phone, type a sentence, Done: it shows on the desktop.
   On the desktop, Save: the file beside the script has the phone's sentence.
5. **Production.** Tag a word on the desktop (Production > Tag Selection…): it is tagged on the phone. Put a scene
   on a shoot day on the phone's stripboard: the desktop's board shows it.
6. **Leaving and coming back.** Close the desktop's window, then reload the browser tab with nobody else
   connected: the script comes back from what the browser kept. Open the link on the desktop again: both meet,
   nothing doubled.
7. **An old peer.** With an Accolade from before protocol 6 at hand, join with it: both say which side is older,
   and neither takes the other's text.
8. **A hidden tab.** Leave the browser tab in the background for over two minutes while typing on the desktop;
   come back: it catches up.

Note what differs (with the device and browser) in LOOP.md.

## What is left, in order

1. ~~The single-writer rule for automatic writes (auto A-numbers, learned notes). Then the two guards go:
   production.go `numberNewScenes`, adaptation_learn.go `learnFromWriting`.~~ Done (question 4 above).
2. ~~Run-length Doc nodes and kept ops, benchmarked on a novel-sized text.~~ Done (question 2 above).
3. ~~Named Docs in a session (protocol 5). The source shared through them, and the source pane's editing live.~~
   Done (question 2 above).
4. ~~The breakdown and the schedule as shared CRDT maps (their own message kind, batched).~~ Done (step 4 above).
5. ~~A help page on editing together: what is shared and what stays yours, Undo, and leaving.~~ Done:
   docs/help/together.md ("Writing Together").
6. ~~Presence: each peer's cursor shown in the others' editors, with their name.~~ Done (presence above).
7. The two-device run above, by Joop, and what it finds.
8. ~~Measure a peer's edit's cost in a window on the novel (each message re-reads the source pane's passages).~~
   Done (step 8 below).

## Step 8: what a peer's key costs a window (measured 2026-11-24)

`BenchmarkPeerKey` (collab_cost_bench_test.go): two windows sharing the PERF-SOURCE bench script (200 scenes
linking a ~100k-word novel, the fixture of question 2's benchmark) with the novel beside it, their sessions linked
in-process by queues (as presence_test.go links two sessions, but delivered one message at a time). The receiving
window has the Source pane shown and its passages read. A peer types a key (one message); what is timed is the
receiving window taking it in: `Receive` (unsealing, the Doc's apply), the source document following the Doc, the
pane, and the layout and refreshes they make (Fyne's test driver lays out but draws nothing, so "redraws" here is
the editors' relayout and the widgets' refreshes). Background work runs at once and is counted (in the browser it
shares the one thread); what waits for a pause (the pane's follow, the kept state, the rows' measuring after its
first stretch) is cancelled. 10 runs each, this VPS:

| a key, taken in by the window | before | after |
|---|---|---|
| the window's own key in its source (the Doc's `SetText`, the editor) | 88 ms, 39 MB, 31k allocs | 77 ms, 39 MB, 31k allocs |
| a peer's key in the source | **600 ms, 113 MB, 489k allocs** | 89 ms, 40 MB, 32k allocs |
| the pane reading the source again, once the peer pauses | (each message) | 511 ms, 74 MB, 456k allocs |
| the window's own key in the script (the pane shown) | 62 ms, 6.7 MB, 23k allocs | 34 ms, 6.7 MB, 23k allocs |
| a peer's key in the script (the pane shown) | 28 ms, 6.8 MB, 18k allocs | 24 ms, 6.8 MB, 18k allocs |

(The own-key rows changed nothing in the code; they vary between runs.) A first run with every put-off thing run
at once too, the rows all measured, gave 2.3 s and 285 MB for a peer's source key: that is the upper bound of the
old per-message cost, the 600 ms its share before the rows' measuring.

Why (pprof): `bindSource` called the pane's `loadSource` on every message of the source's Doc. On the novel that is
the whole pane anew, as when a source is opened: the passages parted again (`passagesOf`, ≈100 ms), the rows
estimated and measured again, the links' model made again, then `follow`, whose scene guesses (`guessScenes` and
`alignScenes`, ≈330 ms) are most of it. The writer's own typing in the source never paid this per key: the pane
follows the text after a pause (`followPause`).

The fix: a peer's message brings the source document up to the Doc at once, as before, and the pane reads the
source again once the peer's messages pause (`followPause`, 200 ms; `sourcePane.peerRead`), as it follows the
writer's own typing. Re-reading only the passages an edit touched was not taken: the parting is a fifth of the
cost, and the guesses and the links' model would still be made again over the whole novel. A peer's key in the
source now costs what this window's own key does; what is left of it is the source editor's relayout of the novel
(`ScriptEditor.relayout`, `wrap.WrapSpans`), which the own key pays too, even with the editor not in sight: a
place to look next, for both. A peer's key in the script needed nothing (it is debounced as one's own).

### The source editor out of sight (2026-11-24)

Why (pprof): the window's source document, opened to follow the source's Doc, had its own editor laid out at every
key, its own or a peer's, though it was nowhere on the screen. Worse than one in sight: never shown, it was never
given a width, and wrapped the novel to 10 columns (its MinSize; 74 in the middle), seven times the rows, and with
no scroll's view its grid is built for all of them (`viewRows`): 39 MB a key, against 16 MB for the same editor in
the middle. The narrow width hit `wrap.WrapSpans` hardest: each row of a line asked how many columns the whole rest
of the line takes, quadratic in a paragraph's length (that alone: 93 → 59 ms out of sight, one run).

The fix, two parts:

- **Lazy, out of sight.** The documents' editors (source, notes) are offstage unless in sight, with the mechanism the
  middle's tabs already used for the script (`SetOffstage`: an edit marks the layout stale; it is made when the
  editor comes back in sight or when something reads it). `docsOnStage` (document.go) decides: in front of the
  middle, or the source edited in the Source pane with that pane in front of its place and not folded under a
  phone's board. It runs wherever what is shown changes: the window laid out again, a middle tab to the front, a
  dock pane to the front of its place (`showFront`), the pane's Edit/Done. A new document starts offstage.
- **Wrap in one pass per line.** `fits` stops counting at the row's room; the spaces kept at a break count on; rows
  are appended in place. The rows are the same: `TestWrapAsBefore` checks them against the old code (kept in the
  test) on random texts, plain and with clusters, wide characters, right-to-left letters, spans, indents past half
  the width and widths down to 1.

Incremental relayout (re-wrapping only the lines an edit touched) was not taken. A key in the source in sight
(profiled alone, after the two fixes): ≈50 ms, of it the editor's relayout ≈29 ms and the status bar's Markdown
counts (`markdownStats`, outside the editor) ≈13 ms. In the relayout, `ClassifyMarkdown` was ≈15 ms, nearly all of
it `rule` copying every line without its spaces to see whether it is a thematic break; it now looks at the line in
place (`TestRuleAsBefore` against the old one), which took 8k of the key's 32k allocations. What is left of the
relayout is the wrap (≈8 ms), the inline emphasis (≈7 ms), and the text as a string and its lines; a re-wrap of the
touched lines alone would save the 8 ms, and the formats depend on the kinds, which depend on the cursor's line. Each
of those passes would have to be made incremental together; that, and the status bar's counts, are the next step if
a key in a shown novel must fit in a frame.

The readers of a hidden editor's layout, audited: every exported one (cursor position, rows, chips, tags, fold
marks, peers' bars and lines, accessibility's lines, `MinSize`, a click's offset, motion keys) lays the editor out
first if it is stale (`fresh`). `TypingKinds`/`MarkdownKinds` report "not laid out" instead: the status line's
element already classifies itself then; cite completion now does too (stale, it would have offered a key inside a code block).
Collab cursors are the script editor's only (buffer marks, drawn at layout). Find highlights the editor in front.
The source pane's row estimates measure its passages, not the editor. The page view reads the text.
`TestDocsOffstage` walks the window's tree through the pane's Edit, panes put in front of it and away, the middle,
back to the side, and a phone's sheet folded down: an editor in sight is never offstage, one out of sight always is,
and one shown again shows the text typed while it was away.

`BenchmarkPeerKey` gains the source in front of the middle (`-shown`), where it is laid out at each key. 3 runs
each, the medians, before and after back to back, this VPS:

| a key, taken in by the window | before | after |
|---|---|---|
| own key in the source, out of sight | 88 ms, 39 MB, 31k allocs | 8.4 ms, 2.8 MB, 0.8k allocs |
| a peer's key in the source, out of sight | 92 ms, 40 MB, 32k allocs | 12 ms, 3.6 MB, 2.3k allocs |
| own key in the source, in sight (the middle) | 51 ms, 16 MB, 42k allocs | 47 ms, 13 MB, 32k allocs |
| a peer's key in the source, in sight | 53 ms, 17 MB, 42k allocs | 53 ms, 14 MB, 32k allocs |

With `rule` in place too (a run of its own afterwards, the machine busier, load 4–5): own key in sight 45 ms (35,
45, 87), 11.6 MB, 24k allocs; a peer's 38 ms, 12.2 MB, 24k allocs. The allocations are the steady measure here.
| own key in the script (the pane shown) | 33 ms, 6.7 MB, 23k allocs | 29 ms, 5.5 MB, 16k allocs |
| a peer's key in the script | 27 ms, 6.8 MB, 18k allocs | 27 ms, 5.6 MB, 11k allocs |

`BenchmarkScriptKey` (3 runs, medians): the novel's screenplay with the novel kept in it 86 → 78 ms (23 → 18 MB,
36k → 19k allocs); sample 3.6 / 3.8 ms, sample-strip 3.6 / 3.3, hi-Deva 6.8 / 6.3, hi-Deva-drawn 7.7 / 7.2,
novel-move 20 / 16 (its allocations the same: run-to-run spread). In sight, a key in the novel source is now the
dozen whole-text passes above, the status bar's counts among them (~5 ms).
