package main

import (
	"strings"
)

// The script's text as it is on disk, which is not always the text in
// the editor: a screenplay in another language than the standard one
// (Lexington's configuration: English by default) is written
// in its own words only where its language says so (fountain_dialect.go).
//
//   - film.eo.fountain: the name says the language; the text is as
//     written (Lexington's convention, a dialect of Fountain).
//   - film.fountain: standard Fountain. A script in another language is
//     written with force marks, its language in a last line of the
//     boneyard, "/* Language: eo */", which no app prints; Accolade takes
//     both back when it reads the file. Only a language said (the name,
//     that line, the rename editor, Script Language, Save As) is written
//     so: one assumed from the preferences, for a file that says none,
//     leaves the text as it is.
//   - a bundle: text.eo.fountain as written, the one Accolade reads, and
//     text.fountain as standard Fountain beside it for other apps (an
//     .accolade may leave that out); the language in info.json
//     (bundle.go). If another app changed text.fountain, that change wins
//     when the bundle is opened, and the script before it is kept as a
//     draft.
//
// A Markdown document has no language here.

// languageLinePrefix starts the line keeping a plain file's language.
const languageLinePrefix = "/* Language: "

// withLanguageLine is a standard Fountain text with the line saying its
// language after it.
func withLanguageLine(text, lang string) string {
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return text + "\n" + languageLinePrefix + lang + " */\n"
}

// cutLanguageLine takes the language line off a text: the text before it
// and the language, ok false if it has none.
func cutLanguageLine(text string) (string, string, bool) {
	body := strings.TrimRight(text, "\r\n")
	nl := strings.LastIndexByte(body, '\n')
	last := body[nl+1:]
	rest, ok := strings.CutPrefix(strings.TrimSpace(last), languageLinePrefix)
	if !ok {
		return text, "", false
	}
	lang, ok := strings.CutSuffix(rest, " */")
	if lang = strings.ToLower(strings.TrimSpace(lang)); !ok || sceneConf[lang] == nil || nl < 0 {
		return text, "", false
	}
	before := body[:nl+1]
	if strings.HasSuffix(before, "\n\n") { // the blank line written before it
		before = before[:len(before)-1]
	}
	return before, lang, true
}

// plainFileText is a plain screenplay file's text as the editor has it,
// and its language: as the name says (film.eo.fountain), else the
// language line's, its force marks taken back; else "", the text as it
// is (the preferences' language assumed, which changes nothing written).
func plainFileText(path, text string) (string, string) {
	if l := languageOf(path); l != "" {
		return text, l
	}
	if before, l, ok := cutLanguageLine(text); ok {
		return dialectFountain(before, l), l
	}
	return text, ""
}

// plainFileForm is a plain screenplay file's text for the editor's text
// in language lang, as the file's name says to write it.
func plainFileForm(path, text, lang string) string {
	if languageOf(path) != "" || !dialect(lang) {
		return text
	}
	return withLanguageLine(standardFountain(text, lang), lang)
}

// writesBundle tells whether saving to path writes a bundle (a plain
// .highland stays plain).
func (w *MainWindow) writesBundle(path string) bool {
	return bundleKindOf(path) != notBundle && (w.bundle != nil || !w.isPlainBundle(path))
}

// scriptTexts are the texts saving to path writes for the editor's
// text: the file's (a bundle's text.fountain), and a bundle's text in
// the language ("" none, for the standard language and Markdown).
func (w *MainWindow) scriptTexts(path, text string) (main, native string, hasNative bool) {
	switch {
	case w.markdown:
		return text, "", false
	case w.writesBundle(path):
		if dialect(w.language) {
			return standardFountain(text, w.language), text, true
		}
		return text, "", false
	}
	if !w.languageSaid { // only assumed: written as it is
		return text, "", false
	}
	return plainFileForm(path, text, w.language), "", false
}

// keepsFountainCopy tells whether a bundle at path keeps text.fountain
// beside its language's text: always, but an .accolade as chosen.
func (w *MainWindow) keepsFountainCopy(path string) bool {
	return !w.noFountainCopy || !strings.EqualFold(typeOfName(path, false).id, typeAccolade.id)
}
