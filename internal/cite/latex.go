// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package cite

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// accents are LaTeX's accent commands as Unicode's combining marks.
var accents = map[string]rune{
	"\"": '̈', "'": '́', "`": '̀', "^": '̂', "~": '̃', "=": '̄',
	".": '̇', "u": '̆', "v": '̌', "H": '̋', "c": '̧', "k": '̨',
	"r": '̊', "d": '̣', "b": '̱',
}

// symbols are LaTeX's letters and signs without an argument.
var symbols = map[string]string{
	"ss": "ß", "o": "ø", "O": "Ø", "ae": "æ", "AE": "Æ", "oe": "œ", "OE": "Œ", "aa": "å", "AA": "Å",
	"l": "ł", "L": "Ł", "i": "ı", "j": "ȷ", "dh": "ð", "DH": "Ð", "th": "þ", "TH": "Þ", "ng": "ŋ", "NG": "Ŋ",
	"&": "&", "%": "%", "$": "$", "#": "#", "_": "_", "{": "{", "}": "}", " ": " ", ",": " ",
	"textendash": "–", "textemdash": "—", "textquoteleft": "‘", "textquoteright": "’",
	"textquotedblleft": "“", "textquotedblright": "”", "dots": "…", "ldots": "…", "textellipsis": "…",
	"S": "§", "P": "¶", "copyright": "©", "textregistered": "®", "texttrademark": "™", "pounds": "£",
	"euro": "€", "textdegree": "°", "LaTeX": "LaTeX", "TeX": "TeX", "-": "", "/": "", "@": "",
}

// decodeLaTeX turns a field's LaTeX into plain text: accents to their
// letters, \ss and the like, \emph{x} and other commands to their
// argument, -- and --- to dashes, ~ to a space, braces dropped,
// whitespace collapsed.
func decodeLaTeX(s string) string {
	if !strings.ContainsAny(s, "\\{}~-`'") {
		return strings.Join(strings.Fields(s), " ")
	}
	r := []rune(s)
	var b strings.Builder
	// arg reads a command's argument at i: {x} or one letter; it gives
	// the argument and where it ends
	arg := func(i int) (string, int) {
		for i < len(r) && r[i] == ' ' {
			i++
		}
		if i >= len(r) {
			return "", i
		}
		if r[i] == '{' {
			depth := 0
			for j := i; j < len(r); j++ {
				switch r[j] {
				case '{':
					depth++
				case '}':
					depth--
					if depth == 0 {
						return string(r[i+1 : j]), j + 1
					}
				}
			}
			return string(r[i+1:]), len(r)
		}
		if r[i] == '\\' { // \"\i
			j := i + 1
			for j < len(r) && unicode.IsLetter(r[j]) {
				j++
			}
			return decodeLaTeX(string(r[i:j])), j
		}
		return string(r[i]), i + 1
	}
	for i := 0; i < len(r); i++ {
		c := r[i]
		switch c {
		case '{', '}':
			continue
		case '~':
			b.WriteRune(' ')
			continue
		case '-':
			if i+2 < len(r) && r[i+1] == '-' && r[i+2] == '-' {
				b.WriteString("—")
				i += 2
			} else if i+1 < len(r) && r[i+1] == '-' {
				b.WriteString("–")
				i++
			} else {
				b.WriteRune(c)
			}
			continue
		case '`':
			if i+1 < len(r) && r[i+1] == '`' {
				b.WriteString("“")
				i++
				continue
			}
		case '\'':
			if i+1 < len(r) && r[i+1] == '\'' {
				b.WriteString("”")
				i++
				continue
			}
		case '\\':
		default:
			b.WriteRune(c)
			continue
		}
		if c != '\\' {
			b.WriteRune(c)
			continue
		}
		// a command: \x (one sign) or \name
		j := i + 1
		if j >= len(r) {
			break
		}
		var name string
		if unicode.IsLetter(r[j]) {
			for j < len(r) && unicode.IsLetter(r[j]) {
				j++
			}
			name = string(r[i+1 : j])
		} else {
			name = string(r[j])
			j++
		}
		if mark, ok := accents[name]; ok {
			a, end := arg(j)
			a = strings.Trim(a, "{}")
			if a == "i" || a == "\\i" || a == "ı" {
				a = "i"
			}
			if a == "j" || a == "ȷ" {
				a = "j"
			}
			ar := []rune(a)
			if len(ar) > 0 {
				b.WriteString(norm.NFC.String(string(ar[0]) + string(mark) + string(ar[1:])))
			}
			i = end - 1
			continue
		}
		if sym, ok := symbols[name]; ok {
			b.WriteString(sym)
			// a letter command ends at the space after it
			if unicode.IsLetter([]rune(name)[0]) && j < len(r) && r[j] == ' ' {
				j++
			}
			i = j - 1
			continue
		}
		// \emph{x}, \textit{x}, \url{x}, an unknown command: its argument
		if j < len(r) && r[j] == '{' {
			a, end := arg(j)
			b.WriteString(decodeLaTeX(a))
			i = end - 1
			continue
		}
		if j < len(r) && r[j] == ' ' {
			j++
		}
		i = j - 1
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
