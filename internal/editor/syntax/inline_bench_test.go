// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package syntax

import (
	"testing"
)

// BenchmarkMarkdownInlineDoc is the editor's per-key inline pass over a
// long document (every line, as a layout does): prose with emphasis,
// links, notes and, in some paragraphs, citations.
func BenchmarkMarkdownInlineDoc(b *testing.B) {
	para := []string{
		"The keeper climbed the *hundred* steps as he had every night for **forty years**, and the lamp",
		"was lit before the [harbour](https://example.org) lights came on below him.[^1] Nothing moved.",
		"As the records show [@doe99, p. 12; see also -@roe2001], and as @smith04 [ch. 3] argues, the",
		"light was never late: me@example.org kept the `log` for it.",
		"",
	}
	var lines []string
	for len(lines) < 2000 {
		lines = append(lines, para...)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		n := 0
		for _, l := range lines {
			n += len(MarkdownInline(l))
		}
		if n == 0 {
			b.Fatal("no spans")
		}
	}
}
