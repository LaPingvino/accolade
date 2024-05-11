[![Please do not theme this app](https://stopthemingmy.app/badge.svg)](https://stopthemingmy.app)

# Accolade

![](screenshots/main.png)

## About

Accolade is a [GTK+](https://www.gtk.org) based distraction free Fountain editor, forked from Accolade which was originally developed by Wolf Vollprecht and currently developed and maintained by Manuel Genovés. Accolade is being developed independently by Joop Kiefte. It uses lexington as back-end for parsing Fountain and offers a very clean and sleek user interface.

## Building

### Building using GNOME Builder

GNOME Builder offers the easiest method to build accolade. Just follow [this guide](https://welcome.gnome.org/app/accolade/#getting-the-app-to-build) and you'll be up and running in a minute.

### Building from Git

To build accolade from source you need to have the following dependencies installed:

- Build system: `meson ninja-build`
- Lexington, a Go tool to convert Fountain to PDF (you will probably need to build this from source)
- GTK4 and GLib development packages: `libgtk-4-dev libglib2.0-dev`
- Rendering the preview panel: `libwebkit2gtk`

- Python dependencies: `python3 python3-regex python3-setuptools python3-levenshtein python3-enchant python3-gi python3-cairo python3-pypandoc`
- Patched libspelling and sourceview. The patches are in `build-aux/flatpak/sourceview_text_commits.patch` and `build-aux/flatpak/libspelling_text_commits.patch`
- FiraSans-Regular, FiraMono-Regular, FiraMono-Bold, FiraMono-Medium
- *optional:* AppStream utility: `appstreamcli`
- *optional:* pdftex module: `texlive texlive-latex-extra`

Depending on your setup you may need to install these schemas before building:

```bash
$ sudo cp data/org.codeberg.lapingvino.Accolade.gschema.xml /usr/share/glib-2.0/schemas/org.codeberg.lapingvino.Accolade.gschema.xml
$ sudo glib-compile-schemas /usr/share/glib-2.0/schemas
```

Once all dependencies are installed you can build accolade using the following commands:

```bash
$ git clone https://codeberg.org/lapingvino/accolade/
$ cd accolade
$ meson builddir --prefix=/usr -Dprofile=development
$ sudo ninja -C builddir install
```

Then you can run the installed package:

```bash
$ accolade
```

Or a local version which runs from the source tree
```bash
$ ./builddir/local-accolade
```
