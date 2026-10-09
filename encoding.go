package main

import (
	"bytes"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// A script's file format: how its text was stored, so that saving it
// writes it back the same way instead of silently changing it. Fountain
// files are UTF-8; older ones (and those from Windows tools) can be
// Windows-1252, start with a byte order mark, or end lines with CR LF.

type fileFormat struct {
	Windows1252 bool // not UTF-8: Windows-1252 (Latin-1 and more)
	BOM         bool // a UTF-8 byte order mark at the start
	CRLF        bool // lines end with CR LF
}

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// decodeScript is a file's text, and how it was stored.
func decodeScript(b []byte) (string, fileFormat) {
	var f fileFormat
	if bytes.HasPrefix(b, utf8BOM) {
		f.BOM = true
		b = b[len(utf8BOM):]
	}
	text := string(b)
	if !utf8.Valid(b) {
		if dec, err := charmap.Windows1252.NewDecoder().Bytes(b); err == nil {
			text, f.Windows1252 = string(dec), true
		}
	}
	if n := strings.Count(text, "\r\n"); n > 0 && n >= strings.Count(text, "\n")/2 {
		f.CRLF = true
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return text, f
}

// encodeScript is text stored as f; if Windows-1252 cannot hold it (a
// character typed since), it is UTF-8, and utf8 says so.
func encodeScript(text string, f fileFormat) (data []byte, utf8Instead bool) {
	if f.CRLF {
		text = strings.ReplaceAll(text, "\n", "\r\n")
	}
	if f.Windows1252 {
		if enc, err := charmap.Windows1252.NewEncoder().Bytes([]byte(text)); err == nil {
			return enc, false
		}
		utf8Instead = true
	}
	if f.BOM {
		return append(append([]byte(nil), utf8BOM...), text...), utf8Instead
	}
	return []byte(text), utf8Instead
}
