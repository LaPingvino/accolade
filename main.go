// Go rewrite of Accolade, a fork of Apostrophe, a Gnome Markdown editor originally written in Python.
// Accolade relies on lexington instead and is a Fountain editor, not a Markdown editor.
package main

// We rely on GTK4 just like Apostrophe did.
import (
	"os"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
)

func main() {
	app := gtk.NewApplication("com.github.diamondburned.gotk4-examples.gtk4.simple", gio.ApplicationFlagsNone)
	app.ConnectActivate(func() { activate(app) })

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func activate(app *gtk.Application) {
	window := gtk.NewApplicationWindow(app)
	window.SetTitle("gotk4 Example")
	window.SetChild(gtk.NewLabel("Hello from Go!"))
	window.SetDefaultSize(400, 300)
	window.Show()
}
