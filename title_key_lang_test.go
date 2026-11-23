package main

import (
	"testing"

	"github.com/LaPingvino/lexington/rules"
)

// Plain hi writes the standard title keys (English first), never a mix of
// its scripts' words; hi-Latn and hi-Deva write their own.
func TestTitleKeyHindi(t *testing.T) {
	ls := rules.DefaultLanguages()
	for _, std := range titleKeys {
		if got := titleKeyIn(ls["hi"], std); got != std {
			t.Errorf("hi: %s written %q", std, got)
		}
	}
	if got := titleKeyIn(ls["hi-Deva"], "Author"); got == "Author" {
		t.Errorf("hi-Deva: Author written in English")
	}
	if got := titleKeyIn(ls["hi-Latn"], "Source"); got != "Srot" {
		t.Errorf("hi-Latn: Source written %q", got)
	}
}
