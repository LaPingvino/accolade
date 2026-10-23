package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/collab"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// Editing together: a script shared with a key (collab), its peers
// found through WebTorrent trackers and connected over WebRTC. The
// editor's changes become the document's ops; the peers' ops change
// the editor's text where they apply, keeping the cursor.

// shareLink is the web version's link for a key; the desktop app takes
// it too.
func shareLink(k collab.Key) string { return "https://lapingvino.github.io/accolade/#k=" + k.String() }

// shared is a window's script shared with peers.
type shared struct {
	session *collab.Session
	network *collab.Network
	key     collab.Key
	stop    context.CancelFunc
	peers   int
	other   bool // a peer runs another version of Accolade
}

func newReplica() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// startSharing shares the window's script with the peers that have key;
// joining, the script comes from them (the window's text is replaced).
func (w *MainWindow) startSharing(key collab.Key, joining bool) {
	w.stopSharing()
	doc := collab.New(newReplica())
	if !joining {
		doc.SetText(w.textEditor.Text())
	}
	s := &shared{session: collab.NewSession(key, doc), key: key}
	s.session.OnRemote = func() { fyne.Do(w.applyRemote) }
	s.session.OnMismatch = func(int) {
		fyne.Do(func() {
			s.other = true
			w.updateStatusBar()
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.stop = cancel
	w.shared = s
	if joining {
		w.textEditor.SetText("")
		w.filledWith = ""
	}
	s.network = collab.Join(ctx, s.session, key, collab.Trackers, func(n int) {
		fyne.Do(func() {
			s.peers = n
			w.updateStatusBar()
		})
	})
	w.updateStatusBar()
}

// stopSharing leaves the peers (the script stays as it is).
func (w *MainWindow) stopSharing() {
	if w.shared != nil {
		w.shared.stop()
		w.shared = nil
		w.updateStatusBar()
	}
}

// shareEdit sends the editor's change to the peers.
func (w *MainWindow) shareEdit(text string) {
	if w.shared == nil {
		return
	}
	var ops []collab.Op
	w.shared.session.Do(func(d *collab.Doc) { ops = d.SetText(text) })
	w.shared.session.Local(ops)
}

// applyRemote puts the peers' changes in the editor: only the part that
// changed is replaced, and the cursor and selection move with the text.
func (w *MainWindow) applyRemote() {
	if w.shared == nil {
		return
	}
	var text string
	w.shared.session.Do(func(d *collab.Doc) { text = d.Text() })
	old, nw := []rune(w.textEditor.Text()), []rune(text)
	start := 0
	for start < len(old) && start < len(nw) && old[start] == nw[start] {
		start++
	}
	end := 0
	for end < len(old)-start && end < len(nw)-start && old[len(old)-1-end] == nw[len(nw)-1-end] {
		end++
	}
	if start == len(old) && start == len(nw) {
		return
	}
	oldEnd, newEnd := len(old)-end, len(nw)-end
	move := func(p int) int {
		switch {
		case p <= start:
			return p
		case p >= oldEnd:
			return p + newEnd - oldEnd
		}
		return newEnd
	}
	w.textEditor.Edit(func(b *buffer.Buffer) {
		from, to := b.Selection()
		cursorAtStart := b.Cursor() == from && from != to
		b.Replace(start, oldEnd, string(nw[start:newEnd]))
		from, to = move(from), move(to)
		if cursorAtStart {
			b.Select(to, from)
		} else {
			b.Select(from, to)
		}
	})
}

// shareDialog shares the script: a new key, its link to send.
func (w *MainWindow) shareDialog() {
	key := w.sharedKey()
	link := widget.NewEntry()
	link.SetText(shareLink(key))
	copyBtn := widget.NewButton("Copy the link", func() {
		fyne.CurrentApp().Clipboard().SetContent(link.Text)
		w.setStatus("Link copied")
	})
	msg := widget.NewLabel("Whoever has this link can read and edit the script with you, in the web version or in Accolade (File > Join a Shared Script…).\nThe peers connect directly; trackers only help them find each other and see nothing of the script.")
	msg.Wrapping = fyne.TextWrapWord
	stop := widget.NewButton("Stop sharing", func() { w.stopSharing() })
	if w.shared == nil {
		stop.Disable()
	}
	d := dialog.NewCustom("Edit Together", "Close", container.NewVBox(msg, link, container.NewHBox(copyBtn, stop)), w.fyneWindow)
	d.Resize(fyne.NewSize(560, 0))
	d.Show()
}

// sharedKey is the key the script is shared with, sharing it first.
func (w *MainWindow) sharedKey() collab.Key {
	if w.shared != nil {
		return w.shared.key
	}
	key, err := collab.NewKey()
	if err != nil {
		dialog.ShowError(err, w.fyneWindow)
		return key
	}
	w.startSharing(key, false)
	return key
}

// joinDialog joins a shared script by its link or key.
func (w *MainWindow) joinDialog() {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("https://lapingvino.github.io/accolade/#k=…")
	dialog.ShowForm("Join a Shared Script", "Join", "Cancel",
		[]*widget.FormItem{widget.NewFormItem("Link or key", entry)}, func(ok bool) {
			if ok {
				w.join(entry.Text)
			}
		}, w.fyneWindow)
}

// join joins a shared script: in this window if it holds nothing of its
// own (see openPath), else in a new one.
func (w *MainWindow) join(linkOrKey string) {
	key, err := collab.ParseKey(strings.TrimSpace(linkOrKey))
	if err != nil {
		dialog.ShowError(err, w.fyneWindow)
		return
	}
	w.withTarget(func(target *MainWindow) {
		target.startSharing(key, true)
		target.setStatus("Joining the shared script…")
	})
}

// sharingStatus is the status bar's word on sharing ("" when not shared).
func (w *MainWindow) sharingStatus() string {
	if w.shared == nil {
		return ""
	}
	if w.shared.other {
		return "Shared · another peer runs another Accolade version: update both"
	}
	switch w.shared.peers {
	case 0:
		return "Shared · looking for others"
	case 1:
		return "Shared · 1 other"
	}
	return fmt.Sprintf("Shared · %d others", w.shared.peers)
}
