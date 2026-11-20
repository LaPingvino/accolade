package main

import (
	"github.com/LaPingvino/lexington/fountain"
)

// A script in another language than the standard one is a dialect of
// Fountain: INT. and EXT. are part of Fountain, so "EN. DOMO - TAGO" is a
// scene heading only to Accolade and Lexington, which read the script with
// Esperanto's words (Joop, 2026-11-20). Where another Fountain app reads
// the text — a plain .fountain, a bundle's text.fountain — it is written
// as standard Fountain: Lexington's fountain.Force adds a force mark to
// each line the standard words would read as another element, and
// fountain.Unforce takes exactly those back when Accolade reads it again.
// Both decide by parsing with the languages' words as Lexington's
// configuration has them (its Scenes, and Standard for the language
// other apps read: English by default); nothing here knows a language.

// standardLanguage is the language whose scene words are standard
// Fountain's.
func standardLanguage() string { return lexSettings.StandardLanguage() }

func (c lexConf) standard() string { return c.StandardLanguage() }

// standardFountain is lang's text as standard Fountain.
func standardFountain(text, lang string) string { return machineLex().standardFountain(text, lang) }

func (c lexConf) standardFountain(text, lang string) string {
	if !c.dialect(lang) {
		return text
	}
	return fountain.Force(text, c.headers(lang), c.headers(c.standard()))
}

// dialectFountain is standard Fountain read as lang's text: the marks
// standardFountain gives taken back.
func dialectFountain(text, lang string) string { return machineLex().dialectFountain(text, lang) }

func (c lexConf) dialectFountain(text, lang string) string {
	if !c.dialect(lang) {
		return text
	}
	return fountain.Unforce(text, c.headers(lang), c.headers(c.standard()))
}

// dialect tells whether lang is a language whose scripts are a dialect
// of Fountain: a known one other than the standard one.
func dialect(lang string) bool { return machineLex().dialect(lang) }

func (c lexConf) dialect(lang string) bool {
	return lang != "" && lang != c.standard() && c.knows(lang)
}
