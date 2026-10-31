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
//   - translate: the headings' words become the new language's (INT. to
//     BIN.), as Lexington's matchups pair them.
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
// heading that could not be changed (no word for it in to).
func changedLine(line, from, to string, how languageChange) (out string, changed, kept bool) {
	marks, forced, bare, ok := headingLine(line, from)
	if !ok || how == markLanguage {
		return line, false, false
	}
	known := isSceneHeadingIn(bare, sceneStarts(to))
	switch how {
	case fountainForm:
		switch {
		case forced && known:
			return marks + bare, true, false
		case !forced && !known:
			return marks + "." + bare, true, false
		}
		return line, false, false
	case translateHeadings:
		dot := ""
		if forced {
			dot = "."
		}
		if tag := headingTag(bare, sceneHeaders(from)); tag != "" {
			if word, found := lexSettings.Translate(tag, from, to); found {
				t := marks + dot + withPlace(bare, word, sceneHeaders(from))
				return t, t != line, false
			}
		}
		if !forced && !known { // no word for it there: dotted, it stays a heading
			return marks + "." + bare, true, true
		}
		return line, false, true
	}
	return line, false, false
}

// headingChanges counts a text's headings in language from that changing
// to language to would change, and those it would leave as they are.
func headingChanges(text, from, to string, how languageChange) (changed, kept int) {
	for _, line := range strings.Split(text, "\n") {
		_, c, k := changedLine(line, from, to, how)
		if c {
			changed++
		}
		if k {
			kept++
		}
	}
	return changed, kept
}

// changeHeadings rewrites the script's headings for language lang, as one
// undo step, and counts them.
func (w *MainWindow) changeHeadings(lang string, how languageChange) int {
	from, n := w.language, 0
	w.textEditor.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			for i := 0; i < b.LineCount(); i++ {
				if c, changed, _ := changedLine(b.Line(i), from, lang, how); changed {
					b.Replace(b.LineStart(i), b.LineEnd(i), c)
					n++
				}
			}
		})
	})
	return n
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
// for it as how says.
func (w *MainWindow) changeLanguage(lang string, how languageChange) {
	n := w.changeHeadings(lang, how)
	w.setLanguage(lang)
	if err := w.nameLanguage(lang); err != nil {
		dialog.ShowError(fmt.Errorf("the script is now in %s, but its file could not be renamed: %v", languageLabel(lang), err), w.fyneWindow)
		return
	}
	status := "Script language: " + languageLabel(lang)
	switch how {
	case fountainForm:
		status += fmt.Sprintf("; %d headings dotted or undotted", n)
	case translateHeadings:
		status += fmt.Sprintf("; %d headings translated", n)
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
		f, _ := headingChanges(text, w.language, lang, fountainForm)
		t, k := headingChanges(text, w.language, lang, translateHeadings)
		s := "Only mark: the text stays, its headings read in " + languageLabel(lang) + ".\n" +
			fmt.Sprintf("Keep words: %d headings get or lose a dot so they stay headings (EN. KUIRO in English is .EN. KUIRO).\n", f) +
			fmt.Sprintf("Translate: %d headings get %s's words", t, languageLabel(lang))
		if k > 0 {
			s += fmt.Sprintf("; %d have none there and stay as they are", k)
		}
		info.SetText(s + ".")
	}
	var d dialog.Dialog
	do := func(how languageChange) func() { return func() { d.Hide(); w.changeLanguage(chosen(), how) } }
	mark.OnTapped, form.OnTapped, translate.OnTapped = do(markLanguage), do(fountainForm), do(translateHeadings)
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
