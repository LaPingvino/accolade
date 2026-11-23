# Layers: lock what came before, add on top

LOOP.md, Joop 2026-11-20: "lock the writer's source, start adding on top". Each stage locks what came before
and adds layers over it, never editing it:

1. **the source story**, locked when adaptation starts;
2. **the writer's draft**, locked at Production's Lock (the White issue);
3. **the shooting script**: the writer's draft plus production's layers (scene numbers, OMITTED, revisions as
   issued drafts, tags, breakdown and schedule in sidecars).

Either the clean writer's version or the shooting version is exported from the same bundle. The drafts store keeps
every locked stage.

## The audit (2026-11-23): what was there

| | What is locked | Where it is kept | "Edit" after the lock | Was missing |
|---|---|---|---|---|
| Source | nothing: the source was a live file | the bundle's `assets/source.md`, a kept copy, the file beside the script, or the script's Source block (source_file.go) | edits the source in place; the links follow it (relink.go writes a link again only when that is certain) | a locked copy; "changed since linked" lasted one session only (`linkedWords`) |
| Writer's draft | the White issue: the text in the drafts store, and the scene numbers locked first (revisions.go `issueDraft`) | `assets/drafts/NN-white/text.fountain`, kind `issue`, in `drafts.json` (drafts.go) | an edit is the open revision (Blue next), marked by diffing the issues (revisions.go) | a name for the stage; an export of the writer's draft as locked |
| Shooting script | each later issue (Blue, Pink …), the scene lock, the page lock (lockedpages.go) | the drafts store; the production block at the script's end (`scene numbers:`, `revision:`, `pages:`); `assets/production/breakdown.json`, `schedule.json`, `revisions.json` | Issue Draft… makes the next set; Undo or Withdraw Last Issue… takes the newest back (issue_undo.go) | nothing to do: the layers exist |
| Exports | — | Export… (export_dialog.go), Export Clean Fountain… (clean_fountain.go: Accolade's data and the writer's choice of notes, boneyard, numbers, OMITTED, title page, outline stripped, `fountain.Strip`) | — | an explicit choice between the writer's version and the shooting version |

Also there: Compare (drafts_view.go `showCompare`, the Compare middle tab: any draft or issue with another or the
script now, Script or Source); Drafts… (browse, compare, restore as a new draft); flatten.go (a bundle saved plain
keeps the extras in a kept copy or `Name.production/`); the Merge Changes branch (merge.go, on its own branch) finds
its base among the drafts, so every lock kept there is a base it can merge against.

## Built (branch lockwriter)

### 1. The source, locked (stages.go)

- **What is locked**: the source as adapted, kept in the drafts store as a draft of the kind `source`, with the
  script as it was then (every draft holds a text; Compare and Restore need one). `drafts.json` gains no field:
  `"kind": "source"`. A source in the script's Source block is read from that text.
- **When**: the first time the script links a passage in this session, as the source pane sees it (none linked,
  then one: Next starting a scene, Adapt…, Link, Start a Screenplay from This Source) and no source is locked yet. Not for a viewer or a
  writer who joined another's shared script, nor for a `.highland` not yet warned about Accolade's files (its
  warning would pop up unasked): the status line offers Lock Source there. A script that was linked before this
  branch has no lock; Adaptation > Lock Source (or View > Stages…) makes it.
- **Never a surprise**: the status line says it ("The source is locked as adapted: Source 1 …"); Adaptation >
  Unlock Source takes the newest lock out of the store (kept for the session: Lock Source then puts that same one
  back, not the source as it is now).
- **Edit after the lock**: the source is edited as before (it is the next source draft, in the making); the locked
  one stays as it was. Relink's certain fixes still run (Joop 2026-11-22/23, later than this item): they keep the
  links finding their passages and do not move the lock. The lock is what "changed since the script was linked"
  is measured from.
- **The notice**: the source's passages are aligned with the locked one's (relink.go `alignPassagesSure`, in the
  background); a linked passage whose words differ from its locked one is changed: its status bar in the warning
  colour, and its row says "Changed since the script was linked" with **Relink From Here**. The count says "· 3
  changed since linked", the cursor's scene's line names them. Without a lock the source as the pane first read it
  this session is the base (as `linkedWords` was, which it replaces).
- **Relink From Here** (the row's button, Adaptation > Relink From Here): asks first, naming how many passages
  changed; then the links are looked at from the locked source to the source now (relink, written only where
  certain) and the source as it is now is locked as the next source draft ("Source 2"): the notices go. The
  earlier lock stays in the store, to compare.

### 2. The stage (stages.go)

`stageOf` is pure: a Markdown document has none; a script with a source, nothing linked and no lock is at
**Source**; without an issue it is **Writing**; with only the White issue and the text the same as it, **Locked**;
with a later issue or changes after one (the open revision), **Shooting**. The window's title says it, and the
header bar and the ribbon's title (`story.accolade · Writing`; on a phone the stage first, `Writing · story.…`,
since the bar cuts the name short). A script with no source and nothing issued shows none (always Writing).

**View > Stages…** lists the locked stages, oldest first: each source lock, the White (the writer's draft), each
later issue, with its date and **Compare** (the Compare middle tab, that stage with the script now; Source for a
source lock). Under them what can be done next: Lock Source / Unlock Source, Lock the Writer's Draft (Issue Draft…
as White), Export Writer's Version… and Export Shooting Version….

### 3. The two versions (export_dialog.go, stages.go)

- **Writer's Version**: the writer's draft as locked: the White issue's text, or before any issue the script now;
  without scene numbers, OMITTED scenes, revision marks, locked pages, the breakdown, and Accolade's blocks and
  links (`fountain.Strip`, as Clean Fountain does); the writer's notes, boneyard, title page and outline kept.
- **Shooting Version**: the script now with every layer: scene numbers, OMITTED, the revision marks (as Page Setup
  shows them, once issued), the locked pages, the FDX's breakdown: what Export… made before.
- File > Export Writer's Version… and Export Shooting Version… open Export… with the version chosen; the dialog
  has the choice itself ("Version:"), saying what each is, and the writer's file is named so (`story - writer.pdf`;
  the shooting version keeps the plain name). Plain Export… opens on the shooting version, what it always made; Revised Pages PDF… is
  always the shooting version. The ribbon's File tab has Writer's Version beside Export (the shooting version is
  Export's; a third button did not fit 1024 px); Lock / Unlock Source live in the Adaptation menu and View > Stages…
  (the ribbon's View tab), Relink From Here on the changed passage's row.

## Decisions for Joop

1. The source locks at the first link of a session with none locked; older linked scripts lock on Lock Source,
   not on their own on opening (no write the writer did not make).
2. Relink's certain fixes stay; the lock is the measure for "changed since", not a freeze of the links.
3. Relink From Here relocks the whole source (a lock is a whole source in the store), not one passage: accepting
   one passage at a time would need a per-passage record, which nothing has asked for yet.
4. Writer's Version before any issue is the script now, stripped of production's layers.
5. The stage is shown once there is something to tell: a source, a lock or an issue (a plain script is Writing,
   unsaid). A source lock taken out by Unlock Source is kept for the session only, as a withdrawn issue is.

## Not done

- The Merge Changes branch could take a returning file's base from the writer's draft or a source lock by kind;
  nothing here depends on it.
- The sample (examples/sample.accolade) was made again: it now shows a source lock (Source 1) too.
- A source draft per source edit (each Done): the store keeps the source with every Save Draft and each lock, which
  seemed enough.
