package prose

import "testing"

// Only web and mail addresses become a PDF link; javascript: and the like
// never do (a PDF reader might act on them).
func TestSafeAddress(t *testing.T) {
	for href, want := range map[string]bool{
		"https://example.org/x": true, "http://example.org": true, "mailto:a@b.org": true, "HTTPS://EXAMPLE.ORG": true,
		"javascript:alert(1)": false, " javascript:alert(1)": false, "JavaScript:x": false, "file:///etc/passwd": false,
		"data:text/html,x": false, "http:/nohost": false, "": false, "mailto:": false, "vbscript:x": false,
	} {
		if got := safeAddress(href); got != want {
			t.Errorf("%q: %v, want %v", href, got, want)
		}
	}
}
