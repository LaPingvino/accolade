// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package help

import (
	"fmt"
	"strings"

	"github.com/LaPingvino/lexington/rules"
)

// settingsPage: what every setting of lexington.toml and of a format file
// does, made from Lexington's own descriptions (rules.Describe), so it
// never drifts from what Lexington reads.
func settingsPage() string {
	var b strings.Builder
	b.WriteString("# Settings Files\n\n")
	b.WriteString("Lexington, the converter inside Accolade, reads two kinds of TOML file: its **settings** (`lexington.toml`) and **format files** (one per script format). " +
		"Accolade opens any `.toml` in an editor made for it — File > Open, a file dropped on the window, `accolade path/file.toml` on the command line, or Open With on the desktop: " +
		"a Lexington settings file in the settings editor, a format file in the format editor, any other TOML as text with a note. It edits the file where it is.\n\n")
	b.WriteString("Every editor has a **TOML** tab with the file's text itself. The pages and the text are the same file: a change on a page is one change in the text, and your comments, " +
		"the order of things and keys Accolade does not know stay as you wrote them. A setting the file does not have shows Lexington's default, greyed, with **Override** to set it; " +
		"**Use the Default** takes it out again. What you type is checked as you type: a change that would make the file wrong is not made, and the line under the pages says why.\n\n")

	b.WriteString("## Accolade and the lexington command\n\n")
	b.WriteString("- **Accolade** reads its own settings file, in its settings folder (Preferences > Script > Edit Lexington Settings… opens it). What the file leaves out is Lexington's default: every language Lexington knows, the screenplay and the presets.\n")
	b.WriteString("- **The lexington command** reads `lexington.toml` in the folder it runs in, or the file `-config` names: `lexington -config ~/.config/accolade/lexington.toml -i film.fountain -o film.pdf` prints with Accolade's settings. " +
		"It adds the presets to the file's element sets; a language the file leaves out is still read with Lexington's rules for it.\n")
	b.WriteString("- **Format files** are read from folders: Accolade's own and yours (Preferences > Script > Edit Format…, New Format from This…). A format saved into your formats folder is a script format at once. The lexington command reads a folder with `-formats folder`, and a format by its file's name with `-e name`.\n\n")

	b.WriteString("## lexington.toml\n\n")
	b.WriteString("The settings editor has these pages: **General** (the standard language and file extensions), **Languages** (one language at a time) and **Element Sets** (one set at a time, with a sample Lexington draws).\n\n")
	for _, f := range rules.ConfFields() {
		fmt.Fprintf(&b, "- **`%s`** — %s\n", f.Key, f.Doc)
	}
	b.WriteString("\n### A language\n\n")
	b.WriteString("`[Languages.eo]` and the like. Hekaya (`.hekaya` files) is fixed, as its spec says; Arabic (`ar`) starts as the same and is yours to change.\n\n")
	for _, f := range rules.LanguageFields() {
		fmt.Fprintf(&b, "- **`%s`** — %s", f.Key, f.Doc)
		if len(f.Choices) > 0 {
			fmt.Fprintf(&b, " (%s)", strings.Join(f.Choices, ", "))
		}
		b.WriteString("\n")
	}
	b.WriteString("\nThe shape rule's `Shape`:\n\n")
	for _, f := range rules.ShapeFields() {
		fmt.Fprintf(&b, "- **`%s`** — %s\n", f.Key, f.Doc)
	}
	b.WriteString("\nThe cue rules (`Cues`):\n\n")
	for _, f := range rules.CueRuleFields() {
		fmt.Fprintf(&b, "- **`%s`** — %s\n", f.Key, f.Doc)
	}

	b.WriteString("\n## Elements\n\n")
	b.WriteString("An element set (`[Elements.name]` in lexington.toml) and a format file's `[Elements]` say how each kind of line is printed. " +
		"In lexington.toml a set is whole: an element it has no rule for is not printed. In a format file an element is whole too, and one it does not have is its Base's; " +
		"changing one copies Base's to start from. The elements:\n\n")
	for _, f := range rules.ElementKeys() {
		fmt.Fprintf(&b, "- **`%s`** — %s\n", f.Key, f.Doc)
	}
	b.WriteString("\nEach element's fields:\n\n")
	for _, f := range rules.FormatFields() {
		fmt.Fprintf(&b, "- **`%s`** — %s\n", f.Key, f.Doc)
	}
	b.WriteString("\nIn Accolade the editor and the page view follow the script format's margins and styles; the PDF, Word, ODT and HTML exports print with all of it. The sample in the editors is Lexington's own printing.\n")

	b.WriteString("\n## Format files\n\n")
	b.WriteString("The format editor has these pages: **Format** (name, description, paper, Base), **Elements** (one at a time, with a sample) and **Buttons**.\n\n")
	for _, f := range rules.FormatFileFields() {
		fmt.Fprintf(&b, "- **`%s`** — %s\n", f.Key, f.Doc)
	}
	b.WriteString("- **`[[Accolade.Buttons]]`** — Accolade's toolbar buttons for scripts in the format, in order (Lexington skips them): `Label`, `Insert` (what it inserts; `{n}` numbers it, `{N}` in words), " +
		"`Restart` (counting starts again after a line starting with this) and `Line = true` (on the next line, not a paragraph of its own). The Buttons page moves them up and down; a button's comments move with it.\n")
	b.WriteString("\n## In the browser\n\n")
	b.WriteString("The browser version opens a `.toml` you pick with File > Open the same way, in a dialog over the script. Its settings file and formats live in the browser's kept settings folder; a file you opened from your computer is offered back as a download when you save it.\n")
	return b.String()
}
