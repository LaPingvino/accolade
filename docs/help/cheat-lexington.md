# Lexington Flags

`lexington -i in -o out [flags]`: formats from the file names. Full page: [Lexington on the Command Line](lexington.md).

<!-- flags -->

### Input and output

- `-i` — Input from provided filename.
- `-o` — Output to provided filename.
- `-from` — Input file type …
- `-to` — Output file type.
- `-template` — Path to a custom template file (e.g., for HTML, FDX, or LaTeX output).

### Format, paper and pagination

- `-e` — Element settings to use …
- `-presets` — List the presets for -e (stage play, radio, ...).
- `-formats` — Folders of format files of your own (name.toml, as Lexington's presets), as a path list, to use with -e.
- `-page` — Paper size for pdf, docx and odt: letter, a4 or a5 (default: the preset's suggestion, else letter).
- `-pagination` — How pages break in pdf, docx, odt and printed html …
- `-config` — Configuration file to use.
- `-dumpconfig` — Dump the default configuration to the location of --config to be adapted manually.
- `-scenein` — Configuration to use for scene header detection on input.
- `-sceneout` — Configuration to use for scene header detection on output.

### Scene numbers, pages and reports

- `-number` — Number the scenes without a number …
- `-mergeomitted` — Print a run of OMITTED scenes as one heading, 12-14 OMITTED (not in FDX, Fountain or lex, which keep one a number).
- `-continueds` — Print the scenes' CONTINUEDs in the PDF …
- `-lockpages` — Write the lock of the script's pages as the PDF prints them (with -lock: locked again, A-pages kept), instead of the script.
- `-lock` — Keep the locked pages in this file (as -lockpages writes them) …
- `-lengths` — Write each scene's page and length in eighths into the FDX (SceneProperties Page and Length), measured as -e and -page print them.
- `-report` — Write a report as CSV instead of the script …

### Colours

- `-colors` — Print scene headings and sections in the colors of their notes ([[red]], [[COLOR BLUE]], [[#3366ff]]).
- `-colorstyle` — How -colors shows a color …
- `-colorscope` — What a marker, background or text color covers …
- `-colortint` — With the line style: a light tint under the block too.
- `-colorpatterns` — With the line style …

### Parts of the script and sides

- `-tag` — Keep only the scenes and sections with this note ([[London]]) or color, with all under them …
- `-character` — Keep this character's sides; needs -sides.
- `-sides` — With -character …
- `-cuewords` — With -sides cue: the cue's last words only, this many (0: the whole speech before).
- `-strip` — Take out of the Fountain input, as a comma list …

### Other

- `-lint` — Run the Fountain linter on the input file
- `-help` — Show this help message
- `-version` — Show version information
- `-footer` — The page's footer for a script with no {{FOOTER: …}} of its own, as -header.
- `-form` — The form of a bundle written …
- `-header` — The page's header for a script with no {{HEADER …
- `-keeploose` — Write the Fountain input so that it reads under the spec as Lexington before 1.19 read it (blank lines and force marks added), instead of converting it …
- `-loose` — Read Fountain as Lexington before 1.19 did …
- `-osfversion` — The Open Screenplay Format version osf and fadein are written in …
- `-titlepage` — The title page template …
- `-titlepages` — List the title page templates for -titlepage.

<!-- /flags -->

## Often used

- `lexington -i script.fountain -o script.pdf`
- `lexington -i script.fdx -o script.fountain`
- `lexington -i scan.pdf -o script.fountain`
- `lexington -i script.fountain -report scenes -o scenes.csv`
