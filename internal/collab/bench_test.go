package collab

// Benchmarks of a Doc on a novel-sized text (docs/COLLAB.md, What is left
// 2): ~100k words, ~600k runes, as a source shared beside the script.
// Run them with
//
//	go test -run '^$' -bench Novel -benchmem ./internal/collab
//
// "pasted" is the novel set in one go; "edited" is that after 3000
// scattered edits by two replicas (words typed key by key, words
// deleted), which is what fragments a Doc kept as runs.

import (
	"fmt"
	"math/rand"
	"runtime"
	"strings"
	"testing"
)

// novelText is ~100k words in paragraphs and chapters.
func novelText() string {
	r := rand.New(rand.NewSource(2))
	syl := []string{"ka", "lo", "mi", "ra", "ten", "sor", "el", "an", "du", "vi", "ne", "sha", "tor", "be", "lin"}
	var b strings.Builder
	words := 0
	for p := 0; words < 100000; p++ {
		if p%85 == 0 {
			fmt.Fprintf(&b, "## Chapter %d\n\n", p/85+1)
		}
		n := 5 + r.Intn(70)
		for i := 0; i < n; i++ {
			if i > 0 {
				b.WriteByte(' ')
			}
			for k := 1 + r.Intn(3); k > 0; k-- {
				b.WriteString(syl[r.Intn(len(syl))])
			}
		}
		b.WriteString(".\n\n")
		words += n
	}
	return b.String()
}

// editedNovel is a Doc of the novel, after edits scattered over it by a
// and a peer b (each key its own op, as typing makes them).
func editedNovel(edits int) (*Doc, *Doc) {
	a, b := New("a"), New("b")
	for _, op := range a.SetText(novelText()) {
		b.Apply(op)
	}
	r := rand.New(rand.NewSource(3))
	for i := 0; i < edits; i++ {
		d, o := a, b
		if i%2 == 1 {
			d, o = b, a
		}
		n := d.Len()
		p := r.Intn(n)
		var ops []Op
		if i%3 == 2 {
			ops = d.Delete(p, 1+r.Intn(6))
		} else {
			for j, c := range []rune(" word") {
				ops = append(ops, d.Insert(p+j, string(c))...)
			}
		}
		for _, op := range ops {
			o.Apply(op)
		}
	}
	return a, b
}

func heapNow() uint64 {
	runtime.GC()
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapAlloc
}

func novelDoc(b *testing.B, kind string) *Doc {
	if kind == "pasted" {
		d := New("a")
		d.SetText(novelText())
		return d
	}
	d, _ := editedNovel(3000)
	return d
}

// BenchmarkNovelMemory: the heap a Doc of the novel holds, per rune.
func BenchmarkNovelMemory(b *testing.B) {
	for _, kind := range []string{"pasted", "edited"} {
		b.Run(kind, func(b *testing.B) {
			var d *Doc
			var per float64
			for i := 0; i < b.N; i++ {
				before := heapNow()
				d = novelDoc(b, kind)
				per = float64(heapNow()-before) / float64(d.Len())
			}
			b.ReportMetric(per, "B/rune")
			b.ReportMetric(float64(d.Len()), "runes")
			runtime.KeepAlive(d)
		})
	}
}

// BenchmarkNovelJoin: a new peer catching up on the novel through a
// session (hello, the ops it lacks), and a session taken up from the kept
// state.
func BenchmarkNovelJoin(b *testing.B) {
	key, _ := NewKey()
	for _, kind := range []string{"pasted", "edited"} {
		d := novelDoc(b, kind)
		want := d.Text()
		b.Run(kind+"/catch-up", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				host, joiner := NewSession(key, d), NewSession(key, New("j"))
				link(host, joiner, "h", "j")
				if text(joiner) != want {
					b.Fatal("not caught up")
				}
			}
		})
		b.Run(kind+"/kept-state", func(b *testing.B) {
			st, _ := NewSession(key, d).State()
			b.ReportMetric(float64(len(st)), "state-bytes")
			for i := 0; i < b.N; i++ {
				s, err := Restore(key, "k", st)
				if err != nil || text(s) != want {
					b.Fatal("not restored", err)
				}
			}
		})
	}
}

// BenchmarkNovelKey: a key typed in the novel's middle, as a window does
// it (Doc.SetText with the editor's text), and as Insert alone.
func BenchmarkNovelKey(b *testing.B) {
	for _, kind := range []string{"pasted", "edited"} {
		d := novelDoc(b, kind)
		text := d.Text()
		mid := len([]rune(text)) / 2
		at := len(string([]rune(text)[:mid]))
		b.Run(kind+"/SetText", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				text = text[:at] + "x" + text[at:] // the editor's new text, as its Text gives it
				d.SetText(text)
			}
		})
		b.Run(kind+"/Insert", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				d.Insert(mid, "x")
			}
		})
	}
}
