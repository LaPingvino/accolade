package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// A script's language can change after it was opened: an import may have
// guessed wrong, the script should be standard Fountain, or it is being
// translated. Script Language (the place button's menu, Format > Script
// Language…) does one of three:
//
//   - mark: the text stays; its headings are read in the new language.
//   - Fountain form: the words stay, and the headings stay headings: those
//     the new language does not know get a dot (EN. KUIRO in an English
//     script is .EN. KUIRO, standard Fountain), and dotted ones it knows
//     lose theirs.
//   - translate: the headings' words become what is chosen for each, from
//     a proposal by Lexington's matchups (INT. to BIN.).
//
// Each way the file's name takes the language (scene.nl.fountain, an
// English one none), which is where it is kept.

// A languageChange is what changing the script's language does to its
// text.
type languageChange int

const (
	markLanguage languageChange = iota
	fountainForm
	translateHeadings
)

// headingLine splits a line that may be a scene heading in language lang
// into its direction marks, whether it is forced (a dot) and the rest;
// ok false for a line that is no heading there, forced or not.
func headingLine(line, lang string) (marks string, forced bool, bare string, ok bool) {
	bare = strings.TrimLeft(line, directionMarks)
	marks = line[:len(line)-len(bare)]
	if r := []rune(bare); len(r) > 1 && r[0] == '.' && (unicode.IsLetter(r[1]) || unicode.IsDigit(r[1])) {
		return marks, true, bare[1:], true // .EN. KUIRO; not ...and so
	}
	return marks, false, bare, isSceneHeadingIn(bare, sceneStarts(lang))
}

// changedLine is a line of a script in language from as it is after the
// change to language to, and whether that changed a heading; kept is a
// heading that stays as it is. Translating, words are what each heading
// word (as headingTag has it: INT, I/E) becomes: "" or none keeps it.
func changedLine(line, from, to string, how languageChange, words map[string]string) (out string, changed, kept bool) {
	marks, forced, bare, ok := headingLine(line, from)
	if !ok || how == markLanguage {
		return line, false, false
	}
	heading := func(s string) bool { return isSceneHeadingIn(s, sceneStarts(to)) }
	switch how {
	case fountainForm:
		switch known := heading(bare); {
		case forced && known:
			return marks + bare, true, false
		case !forced && !known:
			return marks + "." + bare, true, false
		}
		return line, false, false
	case translateHeadings:
		t, tag := bare, headingTag(bare, sceneHeaders(from))
		if word := strings.TrimSpace(words[tag]); tag != "" && word != "" {
			t = withPlace(bare, word, sceneHeaders(from))
		} else {
			kept = true
		}
		if forced || !heading(t) { // not a heading there by itself: dotted, it stays one
			t = "." + t
		}
		return marks + t, marks+t != line, kept
	}
	return line, false, false
}

// A wordProposal is a heading word a script uses, how often, and what
// translating proposes it becomes ("" for none: it stays).
type wordProposal struct {
	tag   string
	count int
	word  string
}

// proposedWords are the heading words a text in language from uses, in
// the order they first come, each with Lexington's matchup in language to.
func proposedWords(text, from, to string) []wordProposal {
	var out []wordProposal
	at := map[string]int{}
	for _, line := range strings.Split(text, "\n") {
		_, _, bare, ok := headingLine(line, from)
		if !ok {
			continue
		}
		tag := headingTag(bare, sceneHeaders(from))
		if tag == "" {
			continue
		}
		if i, seen := at[tag]; seen {
			out[i].count++
			continue
		}
		word, _ := lexSettings.Translate(tag, from, to)
		at[tag] = len(out)
		out = append(out, wordProposal{tag: tag, count: 1, word: word})
	}
	return out
}

// changedLines are a script's lines in language from as changing to
// language to makes them, and how many headings and action lines that
// changed. Headings change as how says; an action line the new language
// would read as a heading ("En la domo..." in Esperanto) gets a ! to stay
// action, and a ! only the old language needed is dropped.
func changedLines(lines []string, from, to string, how languageChange, words map[string]string) (out []string, headings, kept, actions int) {
	out = make([]string, len(lines))
	for i, line := range lines {
		c, changed, k := changedLine(line, from, to, how, words)
		if k {
			kept++
		}
		if changed {
			headings++
		}
		if _, _, _, heading := headingLine(line, from); !heading && (i == 0 || strings.TrimSpace(lines[i-1]) == "") {
			c = actionLine(line, from, to)
			if c != line {
				actions++
			}
		}
		out[i] = c
	}
	return out, headings, kept, actions
}

// actionLine is an action line after a blank one, as it is in language
// to: with a ! if to would read it as a scene heading, without the one it
// had only because from would have.
func actionLine(line, from, to string) string {
	bare := strings.TrimLeft(line, directionMarks)
	marks := line[:len(line)-len(bare)]
	if rest, bang := strings.CutPrefix(bare, "!"); bang {
		if isSceneHeadingIn(rest, sceneStarts(from)) && !isSceneHeadingIn(rest, sceneStarts(to)) {
			return marks + rest
		}
		return line
	}
	if strings.TrimSpace(bare) != "" && isSceneHeadingIn(bare, sceneStarts(to)) {
		return marks + "!" + bare
	}
	return line
}

// headingChanges counts a text's headings in language from that changing
// to language to would change, those it would leave as they are, and the
// action lines that would get or lose a !.
func headingChanges(text, from, to string, how languageChange, words map[string]string) (changed, kept, actions int) {
	_, changed, kept, actions = changedLines(strings.Split(text, "\n"), from, to, how, words)
	return changed, kept, actions
}

// changeHeadings rewrites the script's lines for language lang, as one
// undo step, and counts the headings and action lines it changed.
func (w *MainWindow) changeHeadings(lang string, how languageChange, words map[string]string) (headings, actions int) {
	from := w.language
	w.textEditor.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			lines := make([]string, b.LineCount())
			for i := range lines {
				lines[i] = b.Line(i)
			}
			out, h, _, a := changedLines(lines, from, lang, how, words)
			headings, actions = h, a
			for i := len(out) - 1; i >= 0; i-- { // from the end: earlier offsets stay
				if out[i] != lines[i] {
					b.Replace(b.LineStart(i), b.LineEnd(i), out[i])
				}
			}
		})
	})
	return headings, actions
}

// withLanguageName is a file's name saying it is in lang: its language
// part changed (scene.eo.fountain) or put before its extension; English
// drops it (scene.fountain).
func withLanguageName(path, lang string) string {
	dir, base := filepath.Split(path)
	parts := strings.Split(base, ".")
	switch {
	case len(parts) >= 3 && sceneConf[strings.ToLower(parts[len(parts)-2])] != nil:
		if lang == "en" {
			parts = append(parts[:len(parts)-2], parts[len(parts)-1])
		} else {
			parts[len(parts)-2] = lang
		}
	case lang == "en":
	case len(parts) >= 2:
		parts = append(parts[:len(parts)-1], lang, parts[len(parts)-1])
	default:
		parts = append(parts, lang)
	}
	return dir + strings.Join(parts, ".")
}

// nameLanguage makes the script's file (or the name it will be saved as)
// say lang, renaming the file on disk.
func (w *MainWindow) nameLanguage(lang string) error {
	if w.currentFile == "" {
		name := w.suggestedName
		if name == "" {
			name = "untitled.fountain"
		}
		if n := filepath.Base(withLanguageName(name, lang)); n != name || w.suggestedName != "" {
			w.suggestedName = n
		}
		return nil
	}
	to := withLanguageName(w.currentFile, lang)
	if to == w.currentFile {
		return nil
	}
	if _, err := os.Stat(to); err == nil {
		return fmt.Errorf("%s already exists", filepath.Base(to))
	}
	if err := os.Rename(w.currentFile, to); err != nil {
		return err
	}
	w.currentFile = to
	w.rememberRecent(to)
	w.updateTitle()
	return nil
}

// changeLanguage makes the script's language lang, its headings changed
// for it as how says (translating: into words).
func (w *MainWindow) changeLanguage(lang string, how languageChange, words map[string]string) {
	n, actions := w.changeHeadings(lang, how, words)
	w.setLanguage(lang)
	if err := w.nameLanguage(lang); err != nil {
		dialog.ShowError(fmt.Errorf("the script is now in %s, but its file could not be renamed: %v", languageLabel(lang), err), w.fyneWindow)
		return
	}
	w.updateTitle() // the name says the language, saved or not
	status := "Script language: " + languageLabel(lang)
	switch how {
	case fountainForm:
		status += fmt.Sprintf("; %d headings dotted or undotted", n)
	case translateHeadings:
		status += fmt.Sprintf("; %d headings translated", n)
	}
	if actions > 0 {
		status += fmt.Sprintf("; %d action lines got or lost a ! (to stay action)", actions)
	}
	w.setStatus(status)
}

// pickLanguage asks for the script's language and what changing to it
// does: only mark it, keep the words in Fountain form, or translate.
func (w *MainWindow) pickLanguage() {
	langs := scriptLanguages()
	labels := make([]string, len(langs))
	for i, l := range langs {
		labels[i] = languageLabel(l)
	}
	choose := widget.NewSelect(labels, nil)
	info := widget.NewLabel("")
	info.Wrapping = fyne.TextWrapWord
	mark := widget.NewButton("Only mark", nil)
	form := widget.NewButton("Keep words", nil)
	translate := widget.NewButton("Translate", nil)
	translate.Importance = widget.HighImportance
	chosen := func() string {
		for i, l := range labels {
			if l == choose.Selected {
				return langs[i]
			}
		}
		return ""
	}
	choose.OnChanged = func(string) {
		lang, text := chosen(), w.textEditor.Text()
		if lang == "" || lang == w.language {
			info.SetText("The script is in " + languageLabel(w.language) + ".")
			mark.Disable()
			form.Disable()
			translate.Disable()
			return
		}
		mark.Enable()
		form.Enable()
		translate.Enable()
		f, _, a := headingChanges(text, w.language, lang, fountainForm, nil)
		_, _, ma := headingChanges(text, w.language, lang, markLanguage, nil)
		bangs := func(n int) string {
			if n == 0 {
				return ""
			}
			return fmt.Sprintf(" %d action lines get or lose a ! so they stay action.", n)
		}
		info.SetText("Only mark: the headings are read in " + languageLabel(lang) + "." + bangs(ma) + "\n" +
			fmt.Sprintf("Keep words: %d headings get or lose a dot so they stay headings (EN. KUIRO in English is .EN. KUIRO).", f) + bangs(a) + "\n" +
			"Translate: choose what each heading word becomes, from a proposal; action lines get or lose a ! as needed.")
	}
	var d dialog.Dialog
	do := func(how languageChange) func() { return func() { d.Hide(); w.changeLanguage(chosen(), how, nil) } }
	mark.OnTapped, form.OnTapped = do(markLanguage), do(fountainForm)
	translate.OnTapped = func() { d.Hide(); w.pickTranslation(chosen()) }
	cancel := widget.NewButton("Cancel", func() { d.Hide() })
	content := container.NewVBox(
		widget.NewLabel("The script is in "+languageLabel(w.language)+". Change it to:"),
		choose, info,
		container.NewHBox(cancel, mark, form, translate),
	)
	d = dialog.NewCustomWithoutButtons("Script Language", content, w.fyneWindow)
	choose.SetSelected(languageLabel(w.language))
	d.Resize(fyne.NewSize(520, 300))
	d.Show()
}

// pickTranslation shows the heading words the script uses with what
// translating to lang proposes for each, to change or clear (cleared, a
// word stays as it is), and then translates.
func (w *MainWindow) pickTranslation(lang string) {
	props := proposedWords(w.textEditor.Text(), w.language, lang)
	if len(props) == 0 {
		dialog.ShowInformation("Translate Headings", "The script has no scene headings with words to translate; marking it as "+languageLabel(lang)+" is enough.", w.fyneWindow)
		return
	}
	form := container.New(layout.NewFormLayout())
	entries := make([]*widget.Entry, len(props))
	for i, p := range props {
		e := widget.NewEntry()
		e.SetText(p.word)
		e.SetPlaceHolder("stays " + p.tag + ".")
		entries[i] = e
		form.Add(widget.NewLabel(fmt.Sprintf("%s. (%d×)", p.tag, p.count)))
		form.Add(e)
	}
	note := widget.NewLabel("Proposed from Lexington's matchups; change any, or clear one to keep it. A heading that is no heading in " + languageLabel(lang) + " gets a dot.")
	note.Wrapping = fyne.TextWrapWord
	d := dialog.NewCustomConfirm("Translate Headings to "+languageLabel(lang), "Translate", "Cancel",
		container.NewBorder(note, nil, nil, nil, container.NewVScroll(form)), func(ok bool) {
			if !ok {
				return
			}
			words := map[string]string{}
			for i, p := range props {
				words[p.tag] = entries[i].Text
			}
			w.changeLanguage(lang, translateHeadings, words)
		}, w.fyneWindow)
	d.Resize(fyne.NewSize(480, 200+float32(min(len(props), 8))*44))
	d.Show()
}
