# Storyboard formats: research notes (2026-11-26)

Research only, nothing implemented. Sources are the repos themselves (cloned
2026-11-26) and vendor docs. Where a fact could not be checked from a
primary source it says so.

## 1. Storyboarder (wonderunit/storyboarder)

### Licence: not open source (verified)

- The repo has **no LICENSE file**, now or ever: the GitHub API's commit
  history for `LICENSE`, `LICENSE.md`, `LICENSE.txt` is empty, and the API
  reports `"license": null`. `package.json` has no `license` field.
- The only licence text is `build/license_en.txt`, the **EULA** the
  installers show (`package.json` → `nsis.license`, `appImage.license`):
  "a non-exclusive, non-transferable, limited, revocable license to Use
  the Software … Wonder Unit reserves all rights not expressly granted".
  You may not "rent, lease, lend, sell, redistribute, sublicense" it, nor
  "modify the code for the purpose of enabling features of the Upgraded
  Version". It refers to a Free and a paid "Upgraded version". Some
  bundled components are open source under their own licences.
- README, section "Notes on Open Source License", only links to
  wonderunit.com/thoughts-on-free-and-open-source/.
- Verdict: source-available, all rights reserved. No code may be copied
  into Accolade. Reading and writing its *file format* is interoperability
  and needs none of its code. That is what storyboarder.go already does.
- Activity: last commit on master 2022-06-30 (v3.0.0); last push to any
  branch 2024-03-17. Not archived, but effectively dormant.
- Its predecessor **Storyboard Fountain** (setpixel/storyboard-fountain,
  2014–2017, by the same author, Charles Forman) is **MIT** licensed.

### What it reads and writes (from src/js/main.js, exporters/, importers/, main/menu.js)

| Direction | Format | Notes |
|---|---|---|
| read/write | `.storyboarder` | JSON: `version, aspectRatio, fps, defaultBoardTiming, boards[]`. Each board: `uid, url (board-<n>-<uid>.png), newShot, lastEdited, layers{name:{url,opacity}}, number, shot ("1A"), time, duration (ms), dialogue, action, notes, audio{filename}, link (PSD), sg (Shot Generator 3D data)`. Pictures in `images/` beside it (`-posterframe.jpg`, `-thumbnail.png`, one PNG per layer). Undocumented, but plain and stable since 2017 (fixtures at 1.3.0 still load). |
| read | `.fountain` | Opens a script as a "project": parses it (vendored fountain.js), and **writes scene IDs into the script**: `INT. HOUSE - DAY #3-AB12C#` (scene count + 5-char uid, `fountain-scene-id-util.js`), then shows a dialog asking the user not to change them. Boards go to `storyboards/Scene-<n>-<slug or synopsis>-<id>/…storyboarder`; the folder is found again by the id after its last `-`. Uses headings, synopsis (names the folder), sections (depth 1 split acts), dialogue/action (word count → estimated timing). **Notes are ignored.** It never writes boards back into the Fountain. It **watches the script** (chokidar) and reloads on change. |
| read | `.fdx` | Same flow. Scene IDs go into the FDX XML. |
| read | images, PSD | Import images to new boards / replace a board. PSD import. "Edit in Photoshop" watches the PSD for saves. |
| read | worksheets | Scans printed worksheets back in (QR-coded paper). |
| write | PDF | Print / Export to PDF, configurable grid. |
| write | Animated GIF | |
| write | Video | via bundled ffmpeg (`exporters/ffmpeg.js`) |
| write | Images, ZIP, Web | Scene as images; project ZIP; "Export to Web" (upload to their service) |
| write | **FCP 7 XML (xmeml v4)** | `exporters/final-cut-pro.js`, also imports into Premiere and Resolve |
| write | **FCPXML 1.6** | `exporters/final-cut-pro-x.js` |
| write | glTF | Shot Generator 3D scenes |

There is no CSV, no EDL, no OTIO and no Fountain export.

### Fountain conventions for boards

- Storyboarder: only the `#n-XXXXX#` scene number. That is legal Fountain
  scene-number syntax, so other apps see it as a scene number.
- Storyboard Fountain (MIT, `node-webkit-src/js/fountainmanager.js`)
  *did* put boards in the script as notes. This is what Accolade's
  `[[board:]]` follows:
  - `[[Storyboard Fountain: 0.0.2]]`, `[[aspectRatio: 2.35]]`,
    `[[dataPath: robotdawn-data]]` at the top;
  - `[[board: <file>(, duration: <seconds, 2dp>)(, continued: true)]]` on
    its own line, followed by a blank line; duration only when not
    calculated;
  - `[[duration: <s>]]` for timing a text atom.
  This is the only Fountain board convention found in a primary source. It
  is not part of the Fountain spec (fountain.io has no image syntax).

### What is worth supporting

- `.storyboarder` **read**: done (storyboarder.go). **Write** is feasible:
  the JSON is simple and we know every field. Write only `boards[]` fields
  we understand, and keep unknown fields (`sg`, `layers`, `audio`,
  `lineMileage`) by round-tripping them as raw JSON. Format risk is low
  because the app is dormant.
- Do not use Storyboarder's FCP exports as a reference
  implementation (EULA). Write our own from Apple's docs.

## 2. Other storyboard software

| App | Open? | Exchange (from vendor docs) |
|---|---|---|
| **Toon Boom Storyboard Pro** | proprietary | Imports scripts as .txt/.rtf/**.fdx** (FD elements mapped to panel caption fields). Exports **EDL, AAF, FCP XML** (sequence plus rendered media plus sound), PDF, movies, image sequences, **CSV** of captions (and re-imports its own CSV: column names, scene, panel, captions). "Conformation" round-trips an XML/AAF edit back into the project. No OTIO found in its docs. |
| **Krita storyboard docker** | GPL (KDE) | Panels are animation keyframes. Per scene: name, duration (s+frames), any number of named comment fields. Exports **PDF and SVG** only. Saved inside the `.kra` (a zip) as `<image>/storyboard/index.xml` (`storyboard-info` 1.1: `StoryboardItemList`, `StoryboardCommentList`, export-settings; `plugins/impex/libkra/kis_kra_saver.cpp`). Readable from Go with archive/zip + encoding/xml. |
| **Blender** (grease pencil, Storypencil add-on, VSE) | GPL | No storyboard file format of its own. Shots are VSE strips/markers in a `.blend`. Fully scriptable from Python (`bpy`), so any format can be got in or out with a script. OTIO only through third-party add-ons (not checked). |
| **Boords** | SaaS | PDF, xlsx, docx (text only), PowerPoint/Slides, MP4 animatic, and an **XML for Premiere/Resolve** (timed frames, soundtrack, frame notes; frames without duration get 2 s). Most of it needs a paid plan. No CSV listed. |
| **StudioBinder** | SaaS | Imports scripts (.fdx, **.fountain**, .docx, .pdf, .txt), making a shot list per scene. Exports storyboard **CSV** and PDF. Column set not documented. No re-import found. |
| **Final Draft 11–13** | proprietary | JPG/PNG/GIF images on script pages, title page and Beat Board. "FD10 and older do not support graphics." *How images sit in the .fdx (embedded or linked) is unverified*: the FD knowledge base returned 403 to an honest fetch, and Accolade's FDX code has no image handling. Needs a sample .fdx from FD 12/13. |
| **FrameForge, Celtx, Plot** | proprietary | Not verified: the searches returned no vendor documentation. Check vendor docs or sample files before claiming anything. |

The common pattern: everyone **imports FDX (some Fountain)** and **exports PDF
plus an NLE timeline (FCP XML / Premiere XML / AAF / EDL)**. Nobody
exchanges storyboards with each other in a shared format.

## 3. Standards for frames + timing + text

| Format | Open / documented | Carries | Go practicality |
|---|---|---|---|
| **OpenTimelineIO (.otio)** | Apache-2.0, ASWF (Pixar origin). Documented JSON file-format spec and serialized-schema doc | Timeline → Stack → Tracks → Clips (media refs: image files work, including image-sequence refs), Gaps, Transitions, **Markers**, free-form **metadata dicts** on every object, rational time | Plain JSON: easy to read and write with encoding/json. No official Go binding. Third-party pure-Go `Avalanche-io/gotio` exists (not vetted). Core ships only `.otio/.otiod/.otioz`. EDL, FCP XML, AAF and others are Python adapter plugins, so OTIO is also a bridge format. **Best fit**: frames + durations + per-clip metadata (dialogue, action, scene id) with no loss. Bundles (`.otioz`) carry the images. |
| **EDL (CMX3600)** | De facto; no free official spec (many public write-ups) | Events: reel, track, transition, source/record timecodes, `* FROM CLIP NAME:` comments | Trivial text. Very lossy: no images (reel/clip names only), 8-char reels, no metadata. Write-only, as a last resort. |
| **FCP 7 XML (xmeml)** | Apple-documented (legacy), DTD public | Sequence, clipitems with file refs, durations, names, descriptions, markers | encoding/xml, moderate. Imported by Premiere and Resolve, so it is the widest NLE handoff. Storyboarder and Boords both use it. **Write.** |
| **FCPXML (1.x)** | Apple-documented, DTD per version | Resources (assets), spine of clips, roles, markers, keywords, notes | encoding/xml. Version churn (1.6–1.11+). Write for FCP users; lower priority. |
| **AAF** | AMWA; SDK under its own public-source licence | Full edit + media metadata | Binary Microsoft structured-storage (CFB). Impractical in Go without cgo. Skip, or reach it via OTIO's Python adapter. |
| **CSV shot list** | No standard; every vendor has its own columns | scene, shot, description, duration, image filename | Trivial. **Write** (and lenient read with column mapping). Useful for spreadsheets and StudioBinder-style tools. |
| **Fountain extensions** | Fountain spec has no image or board syntax | Storyboard Fountain's `[[board: file, duration: s, continued: true]]` + `[[dataPath:]]` (MIT, 2014) | Already Accolade's main notation. Keep it. |
| **Pixar/ASWF** | OTIO is the relevant one. (USD covers 3D scenes, not boards; out of scope) | | |

## Recommendation

Keep Fountain `[[board:]]` notes as the main format. Next, in order:

1. **OTIO write (then read)**: the one open, documented, lossless-enough
   standard for frames + timing + text. Plain JSON in Go. Through OTIO's
   adapters a user reaches EDL, FCP XML, AAF and Resolve with no
   per-format code from us. Put scene id, dialogue and action in
   `metadata["accolade"]`. Write `.otioz` so the images travel along.
2. **FCP 7 XML (xmeml) write**: the direct, no-Python handoff to
   Premiere and Resolve (and FCP 7). It is what Storyboarder and Boords
   ship, so editors expect it.
3. **`.storyboarder` write** (read is done): enables round-trip with the
   one free drawing-first storyboard app (section 4). CSV shot-list export
   is a cheap extra if anyone asks.

## 4. Round-trip editing (shelling out to a drawing app)

Goal: the user draws in another app, saves, and Accolade merges the
result into `[[board:]]` notes with timings, without losing script edits
made in the meantime.

### Candidates

**Storyboarder**
- *Launch with a file*: **only on macOS** in release builds (Electron
  `open-file` event, i.e. `open -a Storyboarder x.fountain`). On
  Linux/Windows `main.js` opens `argv[0]` **only in dev mode**
  (`if (isDev)`, with the source comment "TODO why is loading via arg
  limited to dev mode only?"). A registered `storyboarder://` scheme has
  no handler. So on Linux and Windows the user must open the file from
  the app themselves. `xdg-open` on a `.storyboarder` file goes through
  the file association, which also ends in argv, so it probably fails
  too (not tested).
- *Watches/writes back*: it watches the **script** (reloads when Accolade
  saves it) and writes `.storyboarder` JSON + PNGs in `storyboards/…`
  on every save. It rewrites the script only once, to add `#n-XXXXX#` ids.
- *Hand over*: the `.fountain` file itself (headings carry ids; dialogue
  and action appear per scene). Or, for one scene, a `.storyboarder` we
  write ourselves, pre-filled with boards (pictures, `duration`,
  `dialogue`, `action`, `notes`).
- *Back*: boards per scene folder with ms durations and the poster
  frames. Accolade already polls the folder (newestIn, every few s).
- *Platforms*: AppImage / exe / dmg. No Flatpak.

**Krita**
- *Launch*: `krita file.kra` on all platforms. Also headless
  `krita --export …` / `--export-sequence` (check flags against the
  installed version's `krita --help`).
- *Watches*: no. Writes the same `.kra` on save (atomic replace via a
  temp file and backup `.kra~`).
- *Hand over*: a `.kra` with one animation keyframe per board and the
  storyboard `index.xml` (scene names = shot ids, durations, comment
  fields "Dialogue"/"Action"). Building a valid .kra from scratch is
  real work (maindoc.xml, layers, keyframes). Simpler: hand over an
  empty template .kra plus images to import.
- *Back*: `index.xml` gives names, durations and comments. Frames need
  rendering, either with `krita --export-sequence` (headless) or from
  the user's PDF/SVG export. Detect by mtime/size on the .kra.
- *Platforms*: Linux AppImage/Flatpak (org.kde.krita), Windows, macOS.
  Flatpak: run with `flatpak run org.kde.krita <file>`. Whether it can
  see the path depends on the sandbox's filesystem permissions; check
  `flatpak info --show-permissions org.kde.krita` (not verified here).

**Blender**
- *Launch*: `blender file.blend --python bridge.py` (or
  `--python-expr`) on all platforms. This is the most controllable
  option.
- *Watches/writes back*: Blender itself doesn't, but the bridge script
  can register `bpy.app.handlers.save_post`, so that on every Ctrl-S it
  writes a sidecar JSON (and renders each shot's key frame to PNG):
  shot name, frame start/end → duration, marker names, strip notes.
  No guessing at the .blend format.
- *Hand over*: the bridge script builds the scene on first launch: one
  VSE image strip or grease-pencil keyframe per board, timed, plus text
  strips or markers for dialogue. Storypencil can be used if installed.
- *Back*: the sidecar JSON + PNGs, written atomically
  (write temp + rename). Accolade watches that one file.
- *Platforms*: Linux (tarball, Snap, Flatpak org.blender.Blender: same
  sandbox caveat), Windows, macOS (`/Applications/Blender.app/Contents/MacOS/Blender`).

**Others**: Storyboard Pro has a conformation round-trip, but only through
its own XML/AAF and a paid app. Boords and StudioBinder are web apps with
no local file to watch. A plain image editor (GIMP, Krita in plain mode)
editing one frame's PNG is the trivial case: Accolade already reloads
pictures by mtime.

### Merging back safely (all candidates)

1. When handing over, record a **manifest** beside the board data: for
   each board, its stable id (Storyboarder `uid` / Krita scene name /
   Blender strip name = the same id), the scene id
   (`#n-XXXXX#` or Accolade's own), the picture's file and its hash.
2. Keep the id in the Fountain note so the merge never depends on line
   numbers, e.g. `[[board: 3A-K2P9Q.png, duration: 2.00]]`. The file name
   *is* the id, and that stays valid Storyboard Fountain syntax.
3. On a detected save (fsnotify where it works, plus the existing mtime
   poll as fallback, since fsnotify is unreliable on network shares and
   macOS rename-saves; debounce about 1 s and wait for size to settle):
   read the result and compute a per-board diff against the manifest:
   changed picture, changed duration, new board after id X, removed board.
4. Apply the diff to the **current buffer**, not the text as it was at
   handover. Locate each `[[board:]]` note by id. Rewrite only that note's
   line. Insert new boards after their predecessor's note, or at the end
   of the scene found by its scene id. Never touch other lines. The user's
   script edits survive because the merge is a set of edits keyed by
   board id, not a file replace. Make it one undo step.
5. Conflicts: if the user also edited the same note in Accolade
   (duration typed by hand), keep the user's text and offer the app's
   value. If a scene id is gone (scene deleted), park the boards in a
   "loose boards" list instead of dropping them.
6. Pictures are copied into the bundle's `assets/` (or the dataPath
   folder) under the id name. The external app's folder stays its own.

### Recommended flows

1. **Storyboarder, via the storyboards/ folder (do first: half is
   built).** Accolade already reads `storyboards/…/.storyboarder` and
   finds scenes by `#n-XXXXX#`. Add a menu item "Edit boards in
   Storyboarder" that, on macOS, runs `open -a Storyboarder <script>`.
   On Linux/Windows it tells the user to open the script from
   Storyboarder (argv is ignored in release builds). Then add
   "Adopt Storyboarder boards" (automatic on a folder change, or by
   command), which turns the scene's boards into `[[board:]]` notes via
   the id-keyed merge above, using `duration` (ms → s) and the
   poster frames. Storyboarder reloads the script itself when Accolade
   saves, so both stay live together.
2. **Blender with a bridge script (best automation on all three OSes).**
   Accolade writes `bridge.py` plus the board manifest and launches
   `blender <proj>.blend --python bridge.py`. The script builds or updates
   the timeline from the manifest and writes `boards.json` + PNGs on
   every save (`save_post`). Accolade watches `boards.json` and merges.
   This needs no format reverse-engineering and is fully open (GPL app,
   our own script). The cost: Blender is heavy, and the script must
   track bpy API changes across versions.

Krita is a good third choice for drawing-first users: read `index.xml`
from the .kra for names, timing and comments, and render frames headless.
But building the hand-over .kra is the hard part, so start with "pick an
existing .kra" (read-only adopt) before writing one.
