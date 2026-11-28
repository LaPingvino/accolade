// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package cite

import (
	"strings"
	"testing"

	"github.com/LaPingvino/accolade/internal/editor/syntax"
)

// Odd brackets the author may be typing: no panic, every key cited.
func TestStyleOddBrackets(t *testing.T) {
	lib := styleLib(t)
	for _, in := range []string{"[see; @roe01]", "[@{a;b}]", "[@roe01; x; @moe15]", "[;@roe01]", "[@roe01;;@moe15]", "[@roe01, p. 3; and passim]", "[@roe01;", "[@", "@"} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%q panics: %v", in, r)
				}
			}()
			out := Render(in, lib, nil)
			for _, c := range syntax.Citations(in) {
				for _, k := range c.Keys {
					if strings.Contains(out, "@"+k.Key) {
						t.Errorf("%q: %s left as written: %q", in, k.Key, out)
					}
				}
			}
		}()
	}
	if got := FrontOf("---\nnocite: |\n  @*\n---\n").Nocite; len(got) != 1 || got[0] != "*" {
		t.Errorf("nocite @*: %q", got)
	}
}
