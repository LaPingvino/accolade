package main

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/LaPingvino/accolade/docs/help"
)

// The ex commands' table: every command the ':' line knows, with a line
// saying what it does. ex.go runs the line commands (a range, then d, s,
// g, m…), ex_window.go the window's own (:e, :scene, :help, the moves)
// and every command of the registry (commands.go) by a name made from
// its label (Next Source Passage is :next-source-passage). The ':' line
// completes from it (exComplete), the help page is made from it
// (exHelpPage): one table, so neither can drift from what runs.

// exArg is what a command takes after its name, for completion.
type exArg int

const (
	argNone    exArg = iota
	argFile          // a file's path
	argScene         // a scene's or section's heading, or its number
	argElement       // a screenplay element
	argHeading       // a Markdown heading's level
	argExport        // an export format
	argHelp          // an ex command or a help page
	argItems         // a submenu's items (the registry command's)
	argAddress       // a line address (m, t)
	argPattern       // /re/… (s, g, v)
)

// exCmd is one ex command.
type exCmd struct {
	name string // its full name, what completion writes
	// short is the shortest abbreviation vim's way, s[ubstitute]: any
	// beginning of name at least this long (""; the name alone, or a
	// beginning no other command shares)
	short string
	canon string // ex.go's name for it (its switch), for a line command
	usage string // what follows the name, as the help writes it
	desc  string
	group string // its heading in the help and its kin in the list
	// keywords are matched too, less than the name: ":mo" finds every
	// move, ":go" every jump
	keywords string
	ranged   bool  // takes a range (offered after '<,'>)
	argReq   bool  // does nothing without its argument
	arg      exArg // what completes after it
	// id is the registry command it runs (or, for a line command, the
	// one it stands for: :undo is Edit > Undo)
	id string
	// run is a window command's (ex_window.go); nil for a line command
	// (ex.go) and a registry one (id)
	run func(w *MainWindow, arg string) error
}

// line reports whether ex.go runs c.
func (c *exCmd) line() bool { return c.canon != "" }

// The groups, in the help's order.
const (
	exGroupLines   = "Lines"
	exGroupMoves   = "Moving and going to"
	exGroupFiles   = "Files and the window"
	exGroupWriting = "Writing"
	exGroupHelp    = "Help"
	exGroupMenus   = "Accolade's commands"
)

// exLineCommands are ex.go's commands, vim's names and abbreviations.
var exLineCommands = []*exCmd{
	{name: "substitute", short: "s", canon: "s", usage: "/pattern/replacement/[gi]", ranged: true, arg: argPattern,
		desc: "Replace what matches in the lines (the cursor's; % all): g every match, i any case; & is the match, \\1 a group", group: exGroupLines, keywords: "replace"},
	{name: "global", short: "g", canon: "g", usage: "/pattern/command", ranged: true, arg: argPattern,
		desc: "Run d, s, j, m or t on every line that matches (all lines if no range); g! on the others", group: exGroupLines},
	{name: "vglobal", short: "v", canon: "v", usage: "/pattern/command", ranged: true, arg: argPattern,
		desc: "Run a command on every line that does not match", group: exGroupLines},
	{name: "delete", short: "d", canon: "d", ranged: true,
		desc: "Delete the lines (into the register)", group: exGroupLines, keywords: "remove cut"},
	{name: "yank", short: "y", canon: "y", ranged: true,
		desc: "Copy the lines into the register", group: exGroupLines, keywords: "copy"},
	{name: "join", short: "j", canon: "j", ranged: true,
		desc: "Join the lines into one (the cursor's and the next without a range)", group: exGroupLines},
	{name: "undo", short: "u", canon: "u", id: "edit.undo",
		desc: "Undo the last change", group: exGroupLines},
	{name: "move", short: "m", canon: "m", usage: "address", ranged: true, arg: argAddress, argReq: true,
		desc: "Move the lines to after a line: :m0 to the top, :m$ to the end, :m-2 up one, :m+1 down one", group: exGroupMoves, keywords: "move lines"},
	{name: "t", short: "t", canon: "t", usage: "address", ranged: true, arg: argAddress, argReq: true,
		desc: "Copy the lines to after a line (:t. doubles the line)", group: exGroupMoves, keywords: "copy duplicate move"},
	{name: "copy", short: "co", canon: "t", usage: "address", ranged: true, arg: argAddress, argReq: true,
		desc: "Copy the lines to after a line, as :t", group: exGroupMoves, keywords: "duplicate move"},
	{name: "write", short: "w", canon: "w", usage: "[file]", arg: argFile, id: "file.save",
		desc: "Save; with a file, save as that file", group: exGroupFiles, keywords: "save"},
	{name: "wq", short: "wq", canon: "wq",
		desc: "Save and close the window", group: exGroupFiles, keywords: "save quit"},
	{name: "xit", short: "x", canon: "x",
		desc: "Save and close the window, as :wq", group: exGroupFiles, keywords: "save quit exit"},
	{name: "exit", short: "exi", canon: "x",
		desc: "Save and close the window, as :wq", group: exGroupFiles, keywords: "save quit"},
	{name: "quit", short: "q", canon: "q",
		desc: "Close the window (asks to save first); :q! closes it without saving", group: exGroupFiles, keywords: "close"},
}

// exWindowCommands are the window's own, written in ex_window.go.
func exWindowCommands() []*exCmd {
	return []*exCmd{
		{name: "edit", short: "e", usage: "[file]", arg: argFile, run: (*MainWindow).exEdit,
			desc: "Open a file (in a new window if this one has work of its own); without one, the Open dialog", group: exGroupFiles, keywords: "open"},
		{name: "help", short: "h", usage: "[command or page]", arg: argHelp, run: (*MainWindow).exHelp,
			desc: "This list in the Help tab; :h scene the line on :scene; :h fountain a help page", group: exGroupHelp, keywords: "commands"},
		{name: "element", short: "el", usage: "name", arg: argElement, argReq: true, run: (*MainWindow).exElement,
			desc: "Make the cursor's line an element: :element action, :el character…", group: exGroupWriting, keywords: "format"},
		{name: "heading", usage: "[0-6]", arg: argHeading, run: (*MainWindow).exHeading,
			desc: "A Markdown line's heading level (0: normal text); without one, the picker", group: exGroupWriting},

		// Moving and going to. Moves are named move-<what>-up and
		// move-<what>-down, so that :mo lists them all; jumps go-…,
		// next-… and previous-…, with "go" among their keywords.
		{name: "scene", short: "sc", usage: "heading or number", arg: argScene, argReq: true, run: (*MainWindow).exScene,
			desc: "Go to a scene or section: its number, or its heading (Tab lists them)", group: exGroupMoves, keywords: "go move jump"},
		{name: "move-scene-up", run: exOrg("M-k"),
			desc: "Move the scene or section under the cursor, with all it holds, above the one before (Alt+K in vim)", group: exGroupMoves, keywords: "section"},
		{name: "move-scene-down", run: exOrg("M-j"),
			desc: "Move the scene or section under the cursor, with all it holds, below the one after (Alt+J in vim)", group: exGroupMoves, keywords: "section"},
		// STRUCTURAL MOVES (queued, a branch of its own): move-speech-up
		// and -down, move-beat-…, and the like go here, beside
		// move-scene-*, named the same way.
		{name: "promote-section", run: exOrg("M-h"),
			desc: "Make the section under the cursor one level higher (one # fewer; Alt+H in vim)", group: exGroupMoves, keywords: "move level outdent"},
		{name: "demote-section", run: exOrg("M-l"),
			desc: "Make the section under the cursor one level deeper (one # more; Alt+L in vim)", group: exGroupMoves, keywords: "move level indent"},
		{name: "next-heading", run: exOrg("gj"),
			desc: "Go to the next scene or section heading (gj in vim)", group: exGroupMoves, keywords: "go move jump scene"},
		{name: "previous-heading", run: exOrg("gk"),
			desc: "Go to this scene's or section's heading, or the one before (gk in vim)", group: exGroupMoves, keywords: "go move jump scene"},
		{name: "go-parent-section", run: exOrg("gh"),
			desc: "Go to the section the cursor's heading is in (gh in vim)", group: exGroupMoves, keywords: "move jump up"},
		{name: "go-first-inside", run: exOrg("gl"),
			desc: "Go to the first heading inside the cursor's section (gl in vim)", group: exGroupMoves, keywords: "move jump down"},
	}
}

// exNames are the registry commands' names where the label's would not
// do: shorter for the common ones, other where a line command has the
// label's (Copy is :copy's), a page's where two labels are one.
var exNames = map[string]string{
	"view.preview":      "preview",
	"view.foldHere":     "fold",
	"view.foldSections": "fold-sections",
	"view.unfoldAll":    "unfold-all",
	"view.headingsOnly": "headings-only",
	"view.statistics":   "statistics",
	"view.help":         "help-pane",
	"view.menu":         "menus",
	"view.documentType": "document-type",
	"edit.copy":         "copy-selection",
	"edit.cut":          "cut-selection",
	"edit.paste":        "paste",
	"edit.redo":         "redo",
	"edit.findReplace":  "find-replace",
	"file.join":         "join-shared-script",
	"file.newEmpty":     "new-empty",
	"file.recent":       "recent",
	"file.export":       "export",
	"file.saveAs":       "save-as",
	"app.palette":       "palette",
	"app.preferences":   "preferences",
	"preview.pageSetup": "preview-page-setup",
	"preview.text":      "preview-text",
	"preview.pages":     "preview-pages",
	"preview.refresh":   "preview-refresh",
	"preview.follow":    "preview-follow-edits",
	"preview.close":     "close-preview",
	"preview.savePDF":   "save-pdf",
	"preview.fit":       "fit-width",
	"help.about":        "about",
	"help.cheats":       "cheat-sheet",
	"md.headings":       "", // :heading (a window command)
	"source.middle":     "source-in-middle",
	"source.goScene":    "go-passage-scene",
	"md.noteJump":       "go-note",
	"write.elementKind": "elements-list-or-buttons",
	"write.place":       "inside-outside",
	"format.lineUp":     "move-line-up",
	"format.lineDown":   "move-line-down",
	"view.paneTop":      "move-pane-top",
	"view.paneBottom":   "move-pane-bottom",
	"view.paneMiddle":   "move-pane-middle",
	"view.paneSide":     "move-pane-side",
}

// exKeywords are registry commands' keywords (the moves').
var exKeywords = map[string]string{
	"format.lineUp": "move", "format.lineDown": "move",
	"source.next": "go move jump", "source.previous": "go move jump", "source.goScene": "move jump",
	"preview.nextPage": "go move", "preview.previousPage": "go move", "md.noteJump": "move jump",
	"edit.findNext": "go move", "edit.findPrevious": "go move",
	"view.nextPane": "go", "view.previousPane": "go", "view.midNext": "go", "view.midPrev": "go",
}

// exByCommand: commands reached by a window command's argument, not a
// name of their own (an element is :element's, a heading level
// :heading's, a help page :help's), and what to type for them.
func exByCommand(c *command) (string, bool) {
	switch {
	case strings.HasPrefix(c.id, "format.element."):
		return "element " + exElementName(strings.TrimPrefix(c.id, "format.element.")), true
	case strings.HasPrefix(c.id, "md.heading") && c.id != "md.headings":
		return "heading " + strings.TrimPrefix(c.id, "md.heading"), true
	case c.id == "md.headings":
		return "heading", true
	case strings.HasPrefix(c.id, "help.page."):
		return "help " + strings.TrimPrefix(c.id, "help.page."), true
	case strings.HasPrefix(c.id, "help.cheat-"):
		return "help " + strings.TrimPrefix(c.id, "help."), true
	}
	return "", false
}

// exElementName is an element's name as :element takes it: scene-heading.
func exElementName(el string) string { return strings.ReplaceAll(strings.ToLower(el), " ", "-") }

// exLabelName is a label as an ex name: lower case, words joined by -,
// the articles and what is not a letter or digit left out ("Go to the
// Passage's Scene": go-to-passages-scene).
func exLabelName(label string) string {
	var words []string
	for _, f := range strings.Fields(strings.ToLower(label)) {
		var b strings.Builder
		for _, r := range f {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				b.WriteRune(r)
			}
		}
		if w := b.String(); w != "" && w != "the" && w != "a" && w != "an" {
			words = append(words, w)
		}
	}
	return strings.Join(words, "-")
}

// exTable is every ex command, line commands first (their
// abbreviations come first, as in vim), then the window's, then the
// registry's in its order.
type exTable struct {
	cmds   []*exCmd
	byName map[string]*exCmd
}

var (
	exTableOnce sync.Once
	exTab       *exTable
)

func exCommands() *exTable {
	exTableOnce.Do(func() { exTab = newExTable() })
	return exTab
}

func newExTable() *exTable {
	t := &exTable{byName: map[string]*exCmd{}}
	add := func(c *exCmd) {
		if _, dup := t.byName[c.name]; dup {
			panic("ex command named twice: " + c.name)
		}
		t.byName[c.name] = c
		t.cmds = append(t.cmds, c)
	}
	for _, c := range exLineCommands {
		add(c)
	}
	for _, c := range exWindowCommands() {
		add(c)
	}
	for _, c := range commands().commands {
		if _, by := exByCommand(c); by {
			continue
		}
		name, ok := exNames[c.id]
		if !ok {
			name = exLabelName(c.label)
		}
		if name == "" || c.id == "edit.undo" { // :undo is the line command
			continue
		}
		ec := &exCmd{name: name, desc: exDesc(c), group: exGroupMenus, keywords: exKeywords[c.id], id: c.id}
		if c.submenu != nil {
			ec.arg, ec.argReq, ec.usage = argItems, true, "item"
		}
		if c.id == "file.export" {
			ec.arg, ec.usage = argExport, "[format]"
		}
		if strings.HasPrefix(name, "move-") {
			ec.group = exGroupMoves
		}
		add(ec)
	}
	return t
}

// exDesc is a registry command's line: its label (the menu's words),
// and where it lives.
func exDesc(c *command) string {
	label := strings.TrimSuffix(c.label, "…")
	hs := homesOf(c.id)
	if len(hs) == 0 {
		return label
	}
	home := strings.TrimSuffix(hs[0], "…")
	if i := strings.LastIndex(home, " > "); i >= 0 && home[i+3:] == label {
		return home // Edit > Move Line Up: the label is its end
	}
	return label + " — " + home
}

// lookup is the command a name names, vim's way: a line or window
// command by its abbreviation (the first in the table's order), then
// any by its full name, then by a beginning only one name has.
// Ambiguous: the names it begins, if more than one.
func (t *exTable) lookup(name string) (c *exCmd, ambiguous []string) {
	if name == "" {
		return nil, nil
	}
	for _, c := range t.cmds {
		if c.short != "" && len(name) >= len(c.short) && strings.HasPrefix(c.name, name) && strings.HasPrefix(name, c.short) {
			return c, nil
		}
	}
	if c, ok := t.byName[name]; ok {
		return c, nil
	}
	var found []*exCmd
	for _, c := range t.cmds {
		if strings.HasPrefix(c.name, name) {
			found = append(found, c)
		}
	}
	if len(found) == 1 {
		return found[0], nil
	}
	for _, c := range found {
		ambiguous = append(ambiguous, c.name)
	}
	return nil, ambiguous
}

// exNameOf is the name a line names in s (after its range) and where it
// starts and ends: letters, and "-" with a letter after it
// (move-scene-up; m-2 is :m with -2).
func exNameOf(s string, i int) (name string, end int) {
	j := i
	for j < len(s) {
		c := s[j]
		switch {
		case c < 128 && unicode.IsLetter(rune(c)):
			j++
		case c == '-' && j > i && j+1 < len(s) && s[j+1] < 128 && unicode.IsLetter(rune(s[j+1])):
			j++
		default:
			return s[i:j], j
		}
	}
	return s[i:j], j
}

// exSkipRange is where a line's range ends, read without running it (a
// /pattern/ is skipped, not searched), and whether it has one; ok is
// false inside an unclosed /pattern/.
func exSkipRange(s string) (end int, given, ok bool) {
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case c == '%' || c == '.' || c == '$' || c == ',' || c == ';' || c == '+' || c == '-' || c >= '0' && c <= '9':
			i++
		case c == '\'' && i+1 < len(s):
			i += 2
		case c == '\'':
			return i, true, false
		case c == '/' || c == '?':
			j := strings.IndexByte(s[i+1:], c)
			if j < 0 {
				return len(s), true, false
			}
			i += j + 2
		case c == ' ' && i > 0:
			i++
		default:
			return i, i > 0, true
		}
	}
	return i, i > 0, true
}

// exCtx is what completion looks at: the window's state, gathered once
// when the line opens.
type exCtx struct {
	env      cmdEnv
	headings []outlineItem // the script's (or document's) scenes and sections
	dir      string        // where a relative path is (the file's folder)
	files    bool          // paths complete (not in the browser)
	// items are a submenu command's items' labels
	items func(id string) []string
}

// exChoice is one line of the list.
type exChoice struct {
	text  string // what replaces the word completed
	label string // what the list shows: the name and its usage
	desc  string
	cmd   *exCmd // the command it names (nil for an argument)
	// more: it wants an argument still (Enter or a tap goes on to its
	// list rather than running it)
	more bool
	off  bool // cannot be given now (greyed)
}

// exCompletion is what can complete a line.
type exCompletion struct {
	start   int // where the word completed starts in the line
	choices []exChoice
	arg     bool // the choices are an argument's
}

// exFavourites are the commands shown on an empty line.
var exFavourites = []string{"help", "write", "substitute", "global", "scene", "move-scene-up", "move-scene-down",
	"preview", "export", "outline", "cards", "find", "fold", "palette", "quit"}

// complete is what completes line in ctx (the cursor at its end).
func (t *exTable) complete(ctx exCtx, line string) exCompletion {
	if strings.TrimSpace(line) == "" {
		var out []exChoice
		for _, n := range exFavourites {
			if c := t.byName[n]; c != nil && t.available(ctx, c) {
				out = append(out, t.nameChoice(ctx, c))
			}
		}
		return exCompletion{start: len(line), choices: out}
	}
	end, ranged, ok := exSkipRange(line)
	if !ok {
		return exCompletion{start: len(line)}
	}
	name, nameEnd := exNameOf(line, end)
	if nameEnd == len(line) { // the name is being typed
		return exCompletion{start: end, choices: t.names(ctx, name, ranged)}
	}
	if line[nameEnd] != ' ' && line[nameEnd] != '!' {
		return exCompletion{start: len(line)}
	}
	c, _ := t.lookup(name)
	if c == nil {
		return exCompletion{start: len(line)}
	}
	argStart := nameEnd
	for argStart < len(line) && (line[argStart] == ' ' || line[argStart] == '!') {
		argStart++
	}
	return exCompletion{start: argStart, choices: t.args(ctx, c, line[argStart:]), arg: true}
}

// available reports whether c can be offered in ctx.
func (t *exTable) available(ctx exCtx, c *exCmd) bool {
	if c.id != "" && !c.line() {
		rc := commands().byID[c.id]
		return rc != nil && rc.appliesForRole(ctx.env) && (c.id != "view.keyTips" || ctx.env.ribbon)
	}
	switch c.name {
	case "element":
		return !ctx.env.markdown
	case "heading":
		return ctx.env.markdown || ctx.env.midDoc
	case "edit":
		return true
	}
	return true
}

// nameChoice is c as a choice.
func (t *exTable) nameChoice(ctx exCtx, c *exCmd) exChoice {
	label := c.name
	if c.short != "" && c.short != c.name {
		label = c.short + "[" + strings.TrimPrefix(c.name, c.short) + "]"
	}
	if c.usage != "" {
		label += " " + c.usage
	}
	ch := exChoice{text: c.name, label: label, desc: c.desc, cmd: c}
	if c.arg != argNone {
		ch.text += " "
		ch.more = c.argReq
	}
	if c.id != "" && !c.line() {
		rc := commands().byID[c.id]
		e := ctx.env
		e.writer = false
		ch.off = !rc.canRun(e)
		if !rc.applies(ctx.env) { // put away by the role: asked for by name
			ch.desc += " — switches to Production"
		}
	}
	return ch
}

// names are the commands name may be the beginning of (any, typed
// nothing), best first: the one it names (s is substitute), then those
// it begins, then the fuzzy matches; after a range only those taking one.
func (t *exTable) names(ctx exCtx, name string, ranged bool) []exChoice {
	exact, _ := t.lookup(name)
	type scored struct {
		c     *exCmd
		score int
		i     int
	}
	var ss []scored
	for i, c := range t.cmds {
		if ranged && !c.ranged || !t.available(ctx, c) {
			continue
		}
		if name == "" && c.id != "" && !c.line() && !commands().byID[c.id].applies(ctx.env) {
			continue // put away by the role: found once asked for
		}
		s := 0
		switch {
		case c == exact:
			s = 1 << 20
		case strings.HasPrefix(c.name, name):
			// the shortest first word first (pre: preview, then
			// previous-…), then the table's order (a move's up and down
			// together)
			word, _, _ := strings.Cut(c.name, "-")
			s = 1<<19 - len(word)
		case len(name) < 3: // two letters: a word they begin, no more
			if exWordStarts(name, c.name) || exWordStarts(name, c.keywords) {
				s = 1 << 10
			}
		default:
			s = fuzzyScore(name, c.name) * 2
			if k := fuzzyScore(name, c.keywords); c.keywords != "" && k > 0 && k/2 > s {
				s = k / 2
			}
			if l := fuzzyScore(name, c.desc); l/4 > s {
				s = l / 4
			}
		}
		if name == "" {
			s = 1
		}
		if s > 0 {
			ss = append(ss, scored{c, s, i})
		}
	}
	sort.SliceStable(ss, func(a, b int) bool {
		if ss[a].score != ss[b].score {
			return ss[a].score > ss[b].score
		}
		return ss[a].i < ss[b].i
	})
	out := make([]exChoice, len(ss))
	for i, s := range ss {
		out[i] = t.nameChoice(ctx, s.c)
	}
	return out
}

// exWordStarts reports whether a word of s (its words split at - and
// spaces) begins with p.
func exWordStarts(p, s string) bool {
	for _, w := range strings.FieldsFunc(s, func(r rune) bool { return r == '-' || r == ' ' }) {
		if strings.HasPrefix(w, p) {
			return true
		}
	}
	return false
}

// args are what completes c's argument arg.
func (t *exTable) args(ctx exCtx, c *exCmd, arg string) []exChoice {
	var all []exChoice
	switch c.arg {
	case argFile:
		return exFiles(ctx, arg)
	case argScene:
		n := 0
		for _, h := range ctx.headings {
			label := h.Text
			if h.Scene {
				n++
				label = strconv.Itoa(n) + "  " + h.Text
			} else {
				label = strings.Repeat("#", max(h.Level, 1)) + " " + h.Text
			}
			all = append(all, exChoice{text: h.Text, label: label, desc: h.Synopsis})
		}
		if num, err := strconv.Atoi(strings.TrimSpace(arg)); err == nil { // a number: that scene
			n = 0
			for _, h := range ctx.headings {
				if h.Scene {
					n++
					if n == num {
						return []exChoice{{text: arg, label: strconv.Itoa(n) + "  " + h.Text}}
					}
				}
			}
			return nil
		}
	case argElement:
		for _, el := range toolbarElements {
			all = append(all, exChoice{text: exElementName(el), label: exElementName(el), desc: el})
		}
	case argHeading:
		for l := 0; l <= 6; l++ {
			rc := commands().byID["md.heading"+strconv.Itoa(l)]
			all = append(all, exChoice{text: strconv.Itoa(l), label: strconv.Itoa(l), desc: rc.label})
		}
	case argExport:
		if ctx.env.markdown {
			for _, f := range markdownExports() {
				all = append(all, exChoice{text: f.id, label: f.id, desc: f.label})
			}
		} else {
			for _, f := range exportFormatList() {
				all = append(all, exChoice{text: f.id, label: f.id, desc: f.label})
			}
		}
	case argHelp:
		for _, p := range append(help.Pages(), help.Cheats()...) {
			all = append(all, exChoice{text: p.Slug, label: p.Slug, desc: "Help: " + p.Title})
		}
		for _, x := range t.cmds {
			if t.available(ctx, x) {
				all = append(all, exChoice{text: x.name, label: x.name, desc: x.desc})
			}
		}
	case argItems:
		if ctx.items != nil {
			for _, it := range ctx.items(c.id) {
				all = append(all, exChoice{text: it, label: it})
			}
		}
	default:
		return nil
	}
	return exFilter(all, arg)
}

// exFilter are the choices arg begins (any case) first, then the fuzzy
// matches on their text or label.
func exFilter(all []exChoice, arg string) []exChoice {
	if strings.TrimSpace(arg) == "" {
		return all
	}
	low := strings.ToLower(arg)
	var pre, fuzzy []exChoice
	type scored struct {
		c exChoice
		s int
	}
	var ss []scored
	for _, c := range all {
		if strings.HasPrefix(strings.ToLower(c.text), low) {
			pre = append(pre, c)
			continue
		}
		s := max(fuzzyScore(arg, c.text), fuzzyScore(arg, c.label))
		if s > 0 {
			ss = append(ss, scored{c, s})
		}
	}
	sort.SliceStable(ss, func(a, b int) bool { return ss[a].s > ss[b].s })
	for _, s := range ss {
		fuzzy = append(fuzzy, s.c)
	}
	return append(pre, fuzzy...)
}

// exFiles are the files and folders a path begins, a folder with its /
// (Tab goes on into it).
func exFiles(ctx exCtx, arg string) []exChoice {
	if !ctx.files {
		return nil
	}
	dir, base := filepath.Split(arg)
	look := dir
	if strings.HasPrefix(look, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			look = filepath.Join(home, look[2:])
		}
	}
	if !filepath.IsAbs(look) {
		look = filepath.Join(ctx.dir, look)
	}
	ents, err := os.ReadDir(look)
	if err != nil {
		return nil
	}
	var out []exChoice
	for _, e := range ents {
		n := e.Name()
		if strings.HasPrefix(n, ".") && !strings.HasPrefix(base, ".") {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(n), strings.ToLower(base)) {
			continue
		}
		ch := exChoice{text: dir + n, label: n}
		if e.IsDir() {
			ch.text += "/"
			ch.label += "/"
			ch.more = true
		}
		out = append(out, ch)
	}
	return out
}
