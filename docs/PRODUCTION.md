# From screenplay to shoot: the shooting script and pre-production (research and plan)

Research, 2026-11-20, for Joop. The adaptation plan (docs/ADAPTATION.md) brings a story into a screenplay. This is
the other direction, from a finished screenplay to the day it is shot:

- lock the script;
- revise it in coloured sets;
- break it down;
- schedule it;
- make each day's sides and call sheet.

Nothing here is built. Sources for claims about other products are linked inline. Where something is marked
*unverified*, I looked for it and didn't find it.

## 1. What Accolade and Lexington already have for production

**Scene numbers**

- Fountain's `#12A#` is read by `lex.SceneNumber` (lexington lex/type.go:65-71) and printed in both margins of the
  PDF (pdf/create.go:128-159).
- DOCX and ODT put it at the right margin (office/docx.go:243, office/odt.go:246). FDX writes it as
  `Paragraph Number` (fdx/write.go:188), and the layout package has it as `Line.SceneNumber`.
- The PDF reader finds numbers in the margins (pdfin/classify.go:260-278).
- Accolade numbers the unnumbered scenes at export: the Export dialog's "Number the scenes (those without a
  number)" (export_dialog.go:128, Accolade's own `numberScenes` in export_lexington.go:121). It counts on from the last explicit
  number, so after 7 comes 8. Nothing locks the numbers, makes A-scenes or writes OMITTED.

**Cards, outline and title page**

- View > Cards (cards.go) and View > Outline (outline.go, outline_move.go): drag to reorder, colours, folding.
  This is Accolade's counterpart to Final Draft's Beat Board and Story Map, and the base for a stripboard.
- File > Title Page… (title_page_dialog.go): where issued revision sets would be listed.

**Pagination**

- Lexington paginates as screenplay software does: keep-with-next, widows and orphans, and `(MORE)` / `NAME
  (CONT'D)` in the PDF. This is Lexington 1.6.0 (CHANGELOG) and the Export dialog's page-break choice
  (export_pagination.go).
- Missing: the scene `CONTINUED:` / `(CONTINUED)`, locked pages, A-pages, and running headers other than the page
  number (pdf/create.go:483-487 prints only `n.`).

**Colours and tags**

- A note on a heading or section, `[[red]]` or `[[#3366ff]]` as Beat writes it, colours the outline, the cards and
  the export.
  - Accolade: Format > Scene or Section Color… (Ctrl+Shift+L, scene_colors.go).
  - Lexington: `-colors`, `-colorstyle`, `-colorscope` and patterns.
  - FDX keeps the colour as SceneProperties Color (fdx/colors.go).
- Any note on a heading or section is a tag. The Export dialog's "Only scenes and sections tagged" exports just
  those scenes (export_filters.go:18-60, 133-189; Lexington's `lex.FilterTags`, `-tag`).
- These are scene-level tags only. There are no breakdown categories and no tagged words in the text.

**Sides**

- Export > "Sides for character" has three modes: full scenes, cue lines (the last N words of the cue) or lines only
  (export_filters.go:113-153; Lexington's `lex.Sides`, `-character -sides -cuewords`).
- The file name gets " - EDWARD sides" (export_filters.go:93).
- Sides by scene list or by shoot day are missing, and so are the revision header and the "sides" cover.

**Reports**

- View > Statistics… (details.go:20-248) counts:
  - words, pages and printed lines;
  - duration at a page a minute (all, action, dialogue);
  - scenes, interior and exterior;
  - scenes by heading start;
  - each role's speeches and words.
- The status bar shows "Scene 2 of 3 · 412 words · ~2 pages" (stats.go).
- Missing reports:
  - a scene report: number, heading, page, length in eighths, day or night;
  - a location report;
  - a character report: a character's scenes and lines;
  - all of these as CSV.

**Notes and collaboration**

- `[[notes]]` survive every writer. FDX `ScriptNote` is read and written (fdx/write.go:130-151).
- The notes block is `/* Notes */` (notes.go). The source, adaptation and notes blocks are boneyard at the end of
  the script.
- View > Changes (changes.go) is the history of a shared script (Edit Together), with who and when for each
  change. It is not a revision set.

**Export formats**

- Through Lexington (export_formats.go; cli.Formats): PDF, HTML, DOCX, ODT, FDX, Fountain, LaTeX and lex. With
  external tools: EPUB and the e-book formats, RTF and Markdown.
- FDX keeps the title page, dual dialogue, styles, scene numbers, scene colours and script notes.
- FDX writes `RevisionID="0"` on every text run (fdx/write.go:232) and reads `RevisionID` without using it
  (fdx/parse.go:64, 188).
- FDX has no `<Revisions>`, `<LockedPages>`, `<TagData>` or `<SceneProperties Length=…>`.

**Bundles**

- Accolade opens and saves .highland, .textpack and .textbundle (bundle.go, docs/HIGHLAND.md).
- It copies every unknown entry raw, keeps its own state under info.json's `"eu.kiefte.Accolade"` and keeps the
  adaptation source at `assets/source.md`. This is the natural container for production data (§4).

**Not handled: Beat's data block**

Accolade doesn't read Beat's data block, the trailing `/* If you're seeing this … BEAT: {…} END_BEAT */`. It stays in
the text as boneyard. Its offsets go stale as soon as Accolade edits the text (§3).

## 2. What the field does

### Production revisions

**Convention**

- The page colours run white, blue, pink, yellow, green, goldenrod, buff, salmon, cherry, tan, ivory, then
  "double white", "double blue" and so on (https://en.wikipedia.org/wiki/Shooting_script; Final Draft cites WGA
  West for the same order: https://kb.finaldraft.com/hc/en-us/articles/15575314119316).
- Each changed line gets an asterisk in the right margin.
- Each revised page has a header slug with the date and the colour, and the title page lists the slugs issued
  (Wikipedia, above).

**Final Draft**

- Revision Mode, editable Revision Sets, "Show Page Colors", and printing revised pages only
  (https://kb.finaldraft.com/hc/en-us/articles/27813588722708-How-do-I-use-Revision-Mode,
  https://kb.finaldraft.com/hc/en-us/articles/27815245815060-How-do-I-select-modify-or-create-a-Revision-Set).
- Real FD files show:
  - `<Revisions ActiveSet= RevisionMode= ShowPageColor=>` with `<Revision ID Name Color PageColor Mark="*"
    FullRevision Style>` entries;
  - `RevisionID` on each `<Text>` run;
  - `<DeletedText><DeletedTextLocation Offset RevisionID>` for deletions;
  - a "Last Revised" DynamicLabel in `<HeaderAndFooter>`.
  - Sample: https://github.com/michaelforrest/ShootingChecklist/blob/HEAD/ShootingChecklist/Reference/EuroMechanical.fdx.
    FDX has no official spec (https://www.nationalarchives.gov.uk/PRONOM/fmt/1845).

**Fade In** (https://www.fadeinpro.com/page.pl?content=features)

- Revisions, page locking, scene locking, omitted scenes, coloured pages, tags, and reports as HTML and CSV.
- Its format is the **Open Screenplay Format (OSF)**, open and published with an SDK
  (https://github.com/OpenScreenplayFormat/osf-sdk, v4/OSF-4.0.xml).
  - Settings carry `pages_locked`, `pagenumber_mode="1AB"` and `revision`.
  - The scene-heading style carries `number_locked`, `number_mode` and `omitted_text="OMITTED"`.
  - Text carries `<text revision="2">`.
  - It also has `<revision_colors>` (White–Tan, mark `*`) and `<tag_categories>`, the same list as FDX.

**WriterDuet**

- Production revisions with presets "Blue [MM/DD/YY]…", asterisks, and text and page colours
  (https://www.writerduet.com/article/109-view-and-track-revisions-edits).
- Lock Pages: overflow goes to 2A, and a repeat lock gives 1AA (https://www.writerduet.com/article/112-lock-pages).
- Locked scene numbers give 2A, 2B (https://www.writerduet.com/article/162-enable-disable-line-numbering).
- Omit Scene leaves a grey OMITTED (https://www.writerduet.com/article/382-omit-scene).

**Highland**

- Says it leaves this out: "Locked pages and colored revisions are specialized production features that are not
  well-suited to Highland's plain text philosophy… you're generally better off jumping to Fade In or Final Draft";
  and A/B pages: "Not presently"
  (https://quoteunquoteapps.com/highland-pro/library-files/Switching-from-Final-Draft-to-Highland2.pdf).
- Highland 2.5 added a Revision Mode, coloured tracked changes with stars
  (https://quoteunquoteapps.com/highland-2/). I found no locked pages since.

**Beat** (open source, https://github.com/lmparppei/Beat)

- Eight generations: blue `*`, pink `**`, yellow `+`, green `++`, goldenrod `@`, buff `@@`, rose `#`, cherry
  `##` (Frameworks/BeatCore/…/Revisions/BeatRevisions.m:79-87).
- They are stored as character ranges in the trailing JSON block (§3).

**Fountain itself**

- Leaves revisions out on purpose: "Not included are production features such as MOREs, CONTINUEDs, revision
  marks, locked pages, or colored pages… For this reason, Fountain does support scene numbers"
  (https://github.com/nyousefi/Fountain/blob/master/README.markdown;
  https://johnaugust.com/?p=8070).
- A 2018 fountain-dev proposal for `{{if revision==green}}` went nowhere. Rankin preferred a diff built into the
  app (https://groups.google.com/g/fountain-dev/c/6Bt2155VpuI).

### Locked pages, A-pages, A-scenes, OMITTED

**Convention**

- Once the script circulates, page numbers are fixed. A page that grows continues on 45A, 45B.
- Scene numbers are fixed too. A scene inserted between 10 and 11 is A11 in the US and 10A in the UK.
- A cut scene keeps its number as "OMITTED" and can come back
  (https://en.wikipedia.org/wiki/Shooting_script).

**Who has it**

- Final Draft locks pages, has "unlock A pages" and "unlock all"
  (https://kb.finaldraft.com/hc/en-us/articles/27817054455828-How-do-I-lock-or-unlock-script-pages-and-use-the-locking-tools,
  https://kb.finaldraft.com/hc/en-us/articles/15573887116948-Final-Draft-Locked-Pages-and-Page-Numbering).
  - FDX: `<LockedPages><LockedPage PageNumber Position LockLevel LevelIndex>`, where Position is a character
    offset.
  - FDX: `<SceneNumberOptions NumberScheme="1A">`.
  - *Unverified:* how Final Draft itself writes an omitted scene or `Number="12A"`. Only third-party fixtures show
    it, for example OpenDraft's golden file
    https://github.com/Proteus-Technologies-Private-Limited/OpenDraft/blob/HEAD/frontend/src/utils/__snapshots__/exportGolden/locked-omitted-scenes.fdx,
    which writes OMITTED as the heading with its number kept.
- STARC (GPL, https://github.com/story-apps/starc) gives each scene `Number{value, isLocked, followNumber}`. A new
  scene takes an A-number from the last locked one (src/corelib/business_layer/model/text/text_model_group_item.h).
- Fountain has `#12A#`. A fountain-dev thread agreed in principle on forced page numbers, `===17`, `===84A`
  (https://groups.google.com/g/fountain-dev/c/W_cAivCNhIk/m/ELnwExyTBAAJ). Only that thread proposes it, and no
  reader I know of implements it.

### Breakdown (tagging)

- **Final Draft's Tagger:** in FDX, `<TagData><TagCategories>` holds about 30 fixed categories with UUIDs (Cast
  Members, Extras, Props, Wardrobe, Makeup, Vehicles, Animals, Stunts, SFX, VFX, Sound, Music, Set Dressing,
  Location, Script Day, Unit …). Then come `<TagDefinitions CatId Id Label Number>`, `<Tags>`, and
  `TagNumber` on `<Text>`.
  - Beat's FDX writer is the readable reference:
    https://github.com/lmparppei/Beat/blob/main/Frameworks/BeatFileExport/BeatFileExport/Export/Modules/Final%20Draft/BeatFDXExport.m.
  - OSF has the same list.
- **Movie Magic Scheduling** imports .fdx directly and makes breakdown sheets from its scenes and tags, adding
  categories it lacks (https://mms-docs.ep.com/Breakdown/ImportingBreakdownSheets.html).
  - Its other route, `.sex`, is closed. Fade In: "a closed, proprietary format", so it doesn't write it
    (https://fadeinpro.com/kb/content/1/85/en/export-to-movie-magic-scheduling-or-movie-magic-budgeting.html).
  - Final Draft seems to say tags map fully only through .sex. This comes from a search snippet of
    https://kb.finaldraft.com/hc/en-us/articles/15575252743316-Can-Final-Draft-export-to-Movie-Magic-Scheduling-and-Budgeting;
    the page itself returned 403. *Test* tagged FDX against MMS.
- **StudioBinder** imports FDX, Fountain and PDF and tags in its own app
  (https://intercom.help/studiobinder/en/articles/7156315-importing-your-screenplay).
- **Yamdu** imports FDX, PDF and Fountain (https://yamdu.com/hu/blog/a-blog-article-heading/).
- **Gorilla** takes tagged FDX or .sex; I found no CSV import of its own.
- **Celtx** tags in its own app and makes breakdown and one-line schedule reports for distribution
  (https://support.celtx.com/hc/en-us/articles/360059004274). I found no documented import or export of breakdowns.
- **SetHero** imports CSV with a column mapping (https://help.sethero.com/en/articles/1513681-can-i-import-data-from-movie-magic-scheduling).
- **Filmustage** exports .sex, MMSX, XLSX and CSV
  (https://help.filmustage.com/en/articles/4802061-export-to-movie-magic-and-gorilla-software).
- **No published CSV layout:** each tool maps columns.
- **Page eighths:** each scene's length is counted in eighths of a page. FDX has `<SceneProperties
  Length="5 3/8" Page="1">`.

### Reports

- **Final Draft** has Scene, Location, Character, Cast and Script reports (Tools > Reports). Its Scene report
  lists number, intro, location, time, page and length, with tags
  (https://kb.finaldraft.com/hc/en-us/articles/27782286973716-How-do-I-create-reports-on-my-scenes-locations-characters-and-other-script-statistics).
- **Trelby** (open source) has Script, Location, Scene and Character reports and a Dialogue chart
  (https://github.com/trelby/trelby/blob/master/trelby/trelbyframe.py).
- **STARC** has cast, dialogue, gender, location, scene and summary reports.
- **WriterDuet** has reports and a locations/scenes report (https://www.writerduet.com/article/215-reports-overview).

### Scheduling, call sheets, DOOD, shot lists, storyboards

**StudioBinder**

- Script to stripboard: strips by cast, location, pages, script day, D/N and INT/EXT.
- Day breaks by page count, auto-sort, and duplicate schedules to compare.
- One-liner, DOOD, breakdown sheets, daily sides, and call sheets made from a day break.
- Source: https://www.studiobinder.com/film-scheduling-software.

**Open formats**

- **Call sheets:** none. Everyone sends PDFs.
- **Calendar:** StudioBinder has "no way to sync your shooting schedule with the Calendar"
  (https://intercom.help/studiobinder/en/articles/1408719-how-to-create-calendar-events). Celtx pushes shoot days to
  Google Calendar (https://support.celtx.com/hc/en-us/articles/360059004274). iCalendar (RFC 5545) is unclaimed but
  fits.
- **DOOD:** computed, as PDF or XLSX (https://help.filmustage.com/en/articles/8278194-view-and-export-the-dood-report).
  It is a grid of element × day with SW/W/H/WF codes.

**Shot lists and editorial**

- OpenTimelineIO (Apache-2.0, ASWF) has free metadata on every clip and marker. Its integrator's guide names
  "metadata from a screenplay (scene, location, characters)" as a use
  (https://github.com/AcademySoftwareFoundation/OpenTimelineIO/wiki/OpenTimelineIO-Application-Integrator's-Guide).
  Its adapters reach EDL, FCP XML, AAF and Premiere.
- Avid ALE (tab-separated, Scene and Take columns, custom columns allowed;
  https://pomfort.com/article/ale-avid-log-exchange-files-what-they-are-and-why-you-should-understand-them/) is the
  script-supervisor-to-edit log.
- ScriptE and Scriptation hand lined scripts to editorial as PDF only
  (https://help.scriptation.com/en/article/how-do-i-export-my-lining-as-a-pdf-for-my-editor-lbt324/).

**Storyboards**

- Storyboarder (Wonder Unit, https://github.com/wonderunit/storyboarder) opens Fountain.
- It rewrites each heading's scene number to `#<n>-<5-char id>#`. That is valid Fountain.
- It keeps its boards as plain-JSON `.storyboarder` files in `storyboards/Scene-<n>-…/` beside the script.
- It is not OSI-licensed. Its last release was in 2020, and the last push to it in March 2024.

## 3. Existing formats, and what Accolade should do with each

| Format | Covers | Open? | Read/write by | Accolade |
|---|---|---|---|---|
| Fountain `#12A#` | scene numbers | spec | everyone | **both**: the numbers stay in the text |
| Fountain `===` | page break | spec | everyone | both (already) |
| `===17A` | forced page number | proposal only | nobody known | no: other apps print it as action |
| Highland `{{HEADER: …}}` | running header, `%p` | Highland's docs (https://highland-kb.quoteunquoteapps.com/kb/fountain-syntax) | Highland | **read** later in Lexington. Writing it puts directives other apps print |
| Beat's trailing JSON block | revisions, tags, tag definitions, locked, scene numbering start, colour-coded pages (BeatDocumentSettings.m:26-76) | readable source, undocumented | Beat | **read on open** (import its ranges while `Text Length` matches the file, BeatDocumentSettings.m:72) and **keep on save**. Don't write: character offsets break under any other editor, which is why Accolade's anchors are words |
| FDX `<Revisions>`, `RevisionID`, `<DeletedText>`, `<LockedPages>`, `<SceneNumberOptions>`, `<TagData>`, `<SceneProperties Length Page>`, `<HeaderAndFooter>` "Last Revised" | all production data | undocumented XML, well reverse-engineered | Final Draft, Beat, Fade In, MMS, Gorilla, StudioBinder, Yamdu | **both**, the main exchange (Lexington) |
| OSF 4.0 | the same, in Fade In | **open, published** | Fade In | **both** (Lexington), second after FDX |
| .sex / .mmsx / .msd | MMS scheduling | closed | Final Draft, Filmustage, MMS | no. Tagged FDX instead |
| CSV breakdown/stripboard | per-scene rows | no fixed layout | SetHero, Filmustage, StudioBinder (export) | **write** a documented layout, read best effort |
| iCalendar (RFC 5545) | shoot days | standard | every calendar | **write** |
| vCard (RFC 6350) | cast and crew contacts | standard | every address book | **both**, for the call sheet's contacts |
| OpenTimelineIO `.otio` | shot list as a timeline, animatic | open | Resolve, Premiere, Avid via adapters | **write**, read later |
| Avid ALE | scene/take log | de-facto, plain text | Avid, Resolve, Silverstack | write, later |
| CMX3600 EDL | cuts | SMPTE 258M | everyone | no; through OTIO if wanted |
| Fountain issue #50 `[[SHOT …]]` | shot markers in the text (https://github.com/nyousefi/Fountain/issues/50) | proposal | nobody known | **read** as a shot's anchor in its scene. The shot's data (size, lens, board) goes in shots.otio: a note per shot would crowd the script other people read |
| `.storyboarder` | boards per scene | plain JSON, undocumented, not OSI | Storyboarder | **keep its `#n-xxxxx#` numbers intact**. Read its boards as a shot list later |
| Trelby `.trelby`, KIT `.kitsp`, STARC `.starc` | own scripts (SQLite for KIT and STARC) | source only | themselves | no: nothing to exchange that FDX or OSF doesn't carry |

## 4. Where production data lives

The bundle is the container. A `.highland` stays a Highland file when saved. **`.accolade`** is the same zipped
TextBundle with Accolade's production files added. Save As offers both, plus plain `.fountain`. Everything else is an
export.

**In the Fountain text**, readable by every app:

- Scene numbers `#12A#`, once locked.
- OMITTED scenes: `OMITTED #12#` as a forced heading (`.OMITTED #12#`). The old scene goes into a
  `/* Omitted 12 … */` boneyard right under it, so it can come back. Other apps print OMITTED with its number, as the
  convention wants.
- Notes, colours, the Notes, Source and Adaptation blocks, as now.
- A revision is never marked in the text. No asterisks in Fountain and no ranges: they would be noise to other
  apps, and they break.

**In the bundle**, under `assets/production/`, next to the source's `assets/source.md`:

| File | Holds | Format |
|---|---|---|
| `revisions/01-white.fountain`, `02-blue.fountain`, … | each issued draft's text, as sent out | Fountain, whole |
| `revisions.json` | the sets: name, colour, page colour, mark, date, issued or open; FDX `<Revision>`'s fields | JSON (own, after FDX and OSF) |
| `pages.json` | locked pages: page number to its first words in the locked draft, with A-pages | JSON (own, after FDX `<LockedPage>`, anchored by words) |
| `breakdown.json` | tag definitions in FDX's categories (with their UUIDs and numbers), and each tag's scene number and quoted words | JSON (own; FDX/OSF categories) |
| `schedule.json` | shoot days, strips in order, day breaks, banners, units | JSON (own) |
| `cast.vcf`, `crew.vcf` | contacts, with the role or department | vCard |
| `shots.otio` | shot list per scene, with boards as media references | OpenTimelineIO |
| `boards/` | images | PNG/JPEG |

**Why the own files.** No open format exists for a schedule, a breakdown or a lock. FDX's are offsets into its own
XML, and Beat's are offsets into the text. Accolade's own rule, from the adaptation links, is to anchor by words: a
tag is `{scene: "12", category: "Props", item: "revolver", at: "the revolver on the table"}`. An edit elsewhere doesn't
move it, and a broken anchor is shown as missing, the way the source pane shows it.

**Revisions are snapshots, not marks.** Issuing a set saves the text. A line's colour is the last set it changed in,
found by diffing the issued drafts in turn. This is Rankin's "diff tool built into the app", and it survives edits
in any other app between issues.

**info.json.** `"eu.kiefte.Accolade"` gets `production: {version, stage, activeSet}`. The stage drives §6's guide.

**Risk: Highland may drop unknown files when it saves.** This is untested. Highland's own page says it keeps
"revisions" metadata in subdirectories (https://quoteunquoteapps.com/highland-pro/highland-format), and assets/ is
the likeliest kept. Test with a real Highland: open an `.accolade` renamed `.highland`, edit, save, and list the zip.
Until then, Save as .highland warns that production data may not survive Highland.

**Flattening to plain `.fountain`.** What is in the text stays: the numbers, OMITTED and the blocks. The rest follows
the source's layers:

1. a copy Accolade keeps in its data folder, per script, following renames, as `sources` does;
2. or, on choice, a `Name.production/` folder beside the file with the same files. That folder is the bundle
   unzipped, and the browser downloads it as a zip.

Save As plain says which of the two it uses. A plain file opened again finds its data that way.

## 5. The gaps, ranked for an independent filmmaker or small team

Effort is rough: S is a day or two of ticks, M a week, L more.

1. **Locked scene numbers, A-scenes, OMITTED.**
   - What: number once, then lock. A new scene between 10 and 11 gets A11 (US) or 10A (UK, a preference). Omit
     keeps the number.
   - Who has it: Final Draft, Fade In, WriterDuet, STARC.
   - Fountain: `#n#` in the text, the omitted scene in boneyard.
   - FDX: `Number` round-trips already. Add `<SceneNumberOptions>`, and write OMITTED as OpenDraft does.
     OSF's `number_locked`.
   - Effort: S. Where: the numbering moves from Accolade's export-time `numberScenes` (export_lexington.go:121) into
     Lexington, as `lex.NumberScenes` with locked and A-number rules, so its CLI, the FDX and OSF writers and
     Accolade share it. Accolade gets the menu items: Format > Number and Lock Scenes, Omit Scene, Restore Scene.
2. **Production revisions: issue, colours, asterisks, headers, revised pages only.**
   - What: Issue White (the first lock), then Blue and on, in the WGA order with "double" sets.
     - The PDF puts a `*` at the right margin of each changed line.
     - The header reads "Blue Rev. 11/20/26" with the page number.
     - The title page lists the issued sets.
     - Export revised pages only, optionally on coloured paper.
   - Who has it: Final Draft, Fade In, WriterDuet, Beat, Highland (marks only).
   - Representation: snapshots and revisions.json (§4).
   - FDX: write `<Revisions>` and `RevisionID` per run from the diff, and `<DeletedText>`. Reading is lossy:
     `<DeletedTextLocation>` records where text went, not the text, so earlier drafts can't be rebuilt. An FDX that
     arrives with Blue and Pink marks keeps them as imported marks, one set per `RevisionID`, until the next
     Issue. From then on the snapshots take over.
   - Effort: M. Where: Lexington (marks, header, revised-pages filter, colour band, FDX); Accolade (Issue, the
     diff, View > Revisions showing colour bars in the margin as the source pane's marks).
3. **Locked pages and A-pages.**
   - What: after lock, page breaks stay. Overflow goes to 12A, a shrunk page gets "12-13" or a blank, and the
     scene's `CONTINUED:` and `(CONTINUED)` are added.
   - Who has it: Final Draft, Fade In, WriterDuet.
   - Representation: pages.json by words.
   - FDX: `<LockedPages>`, its Position computed when writing. OSF: `pages_locked`, `page_number`.
   - Effort: M–L, the hardest part being Lexington's paginator laying out to anchors. Where: mostly Lexington
     (`rules.Pagination` with locks), Accolade for Lock Pages and Unlock.
4. **Breakdown tagging.**
   - What: select words and tag them in a category (Cast fills itself from the cues, Location from the heading).
     Each scene gets its eighths, D/N, INT/EXT and script day, and a breakdown sheet.
   - Who has it: Final Draft Tagger, Fade In, WriterDuet, Beat, StudioBinder, Celtx.
   - Representation: breakdown.json by words.
   - FDX: write `<TagData>` and `TagNumber` as Beat does, and `<SceneProperties Length Page>`. That is the route
     into Movie Magic, Gorilla, StudioBinder and Yamdu. Read FDX and Beat tags.
   - Effort: M. Where: Accolade (a Breakdown pane, the editor's tint for tagged words); Lexington (eighths from the
     layout, FDX TagData).
5. **Reports.**
   - What:
     - scene report (number, heading, page, eighths, D/N, cast);
     - location report;
     - character report (their scenes, lines, monologues);
     - cast report;
     - breakdown sheets per scene and element report per category;
     - all as PDF, CSV and HTML.
   - Who has it: Final Draft, Fade In, Trelby, STARC, WriterDuet.
   - Representation: computed, nothing stored.
   - Effort: S–M. Where: Lexington (a report package, as `layout`); Accolade (Statistics grows into View > Reports).
     The basic reports need no tagging and come early (Order step 2); breakdown sheets follow tagging.
6. **Stripboard and shooting schedule.**
   - What: strips, one per scene, coloured INT/EXT × D/N. Drag them into days and add day breaks with totals in
     eighths. Sort by location, cast or D/N. Make a one-liner and a shooting schedule. DOOD comes from it.
   - Who has it: MMS, StudioBinder, Gorilla, Celtx, Yamdu.
   - Representation: schedule.json.
   - Exports: iCalendar per day, CSV per strip.
   - FDX: none. FDX carries tags, not schedules.
   - Effort: M–L. Where: Accolade (a Schedule view like the cards), Lexington (one-liner, DOOD and schedule
     PDFs).
7. **Day packs: sides and call sheets.**
   - What: for a shoot day, the day's scenes as sides (Lexington has `lex.Sides`; add by scene list, two-up, the
     revision header, a cover) and a call sheet. The call sheet has date, call times, location, scenes, cast with
     their numbers, contacts from vCard, the advance schedule and weather left blank.
   - Who has it: StudioBinder, SetHero, Celtx, Yamdu.
   - Exports: PDF and .ics.
   - Effort: M. Where: Lexington (sides by scene list, call sheet PDF), Accolade (the day's pack in one export).
8. **FDX and OSF round-trip of all of this.** This is the exchange each step above needs: it lets the team hand the
   script to a production office using Final Draft or MMS, and take it back.
   - Read and write FDX's revisions, locks, tags and scene properties.
   - Add an OSF reader and writer.
   - Effort: M, spread over steps 1–4. Where: Lexington.
9. **Shot list and storyboards.**
   - What: shots per scene (number, size, angle, movement, lens, notes, board image).
   - Who has it: StudioBinder, Celtx, Storyboarder.
   - Representation: shots.otio.
   - Also: keep Storyboarder's scene ids.
   - Effort: M. Where: Accolade.
10. **On set and after.**
    - Lined script, script supervisor notes, ALE out.
    - Who has it: ScriptE, Scriptation.
    - Effort: L. Later, and only if wanted.

## 6. The guided workflow

The adaptation workflow's strength is one pane that says where you are and what is next: "50% adapted · 1 cut · 3 to
go", and Next. Production gets the same.

The **Production pane** (View > Production, a tab beside Notes and Source) shows the stage and one count per stage:

1. **Lock.** A check before the lock lists the scenes without a heading time, duplicate numbers and an unsaved
   title page. Then Number and Lock Scenes, and Lock Pages. Issuing White is the lock. From then on the pane says
   "Locked 11/20 · 112 pages · 87 scenes".
2. **Revise.** Writing goes on as usual. The pane shows "Blue (open) · 6 pages changed · scenes 12, A14, 30". Issue
   Blue asks for the date and gives the export of revised pages and the full script. The margin shows each line's
   colour.
3. **Break down.** "62% broken down · 33 scenes to go". **Next scene** goes to the first scene not marked done, the
   way Next passage does, with:
   - Cast filled from its cues and the location from its heading;
   - the eighths shown;
   - suggestions for the rest, from words found in earlier scenes' tags.

   Done marks it done.
4. **Schedule.** "18 of 87 scenes on a day · 3 days". The stripboard, with the unscheduled strips at the end and
   Next Day.
5. **Day pack.** Per day: sides, call sheet and .ics, as one export. FDX or CSV goes out to an office using other
   software.

Each stage can be returned to. A revision after scheduling marks the changed scenes on the stripboard and in the
breakdown as "changed in Blue: check", which the stages before can't show.

## Order

1. **Lock scene numbers:** A-scenes, OMITTED and Restore. Accolade's `numberScenes` moves into Lexington as
   `lex.NumberScenes`, with A-numbers. In the text only, so nothing waits on the bundle.
2. **Basic reports:** scene (with eighths, D/N), location, character and cast, as PDF and CSV. They store nothing,
   need no bundle, and pay off at once. Statistics grows into View > Reports.
3. **The `.accolade` bundle and the production files' home:** `assets/production/`, the flattening layers and the
   Highland warning. Test with a real Highland.
4. **Issue and revision sets:** snapshots, the diff, margin colours, Lexington's asterisks, headers and revised pages
   only. FDX `<Revisions>` and `RevisionID` out, and in as marks.
5. **Locked pages and A-pages** in Lexington's paginator, with `CONTINUED`. FDX `<LockedPages>`.
6. **The Production pane** with Lock and Revise, so the workflow stands before breakdown starts.
7. **Breakdown:** the categories, tagging by words, Next scene, eighths, breakdown sheets and element reports. FDX
   `<TagData>` and `<SceneProperties>` out (test against MMS), Beat's and FDX's tags in.
8. **Schedule:** the stripboard (built on the cards), days, one-liner, DOOD, .ics and CSV.
9. **Day packs:** sides by scene list and the call sheet with vCard contacts.
10. **OSF** reader and writer.
11. **Shot list** in OTIO, keeping Storyboarder's ids and reading `[[SHOT]]` markers.
12. Later, and only if wanted: the lined script and ALE.

Open questions for Joop:

- US or UK A-numbering by default: A11 or 10A?
- Should `.accolade` be the default Save for a script that has production data, or should `.highland` stay the
  default with the warning?
