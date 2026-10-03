[![Please do not theme this app](https://stopthemingmy.app/badge.svg)](https://stopthemingmy.app)

# Accolade

![](screenshots/main.png)

## About

Accolade is a distraction-free Fountain editor for screenwriters, built with [Fyne](https://fyne.io) in Go. Originally forked from a GTK-based application, Accolade has been completely rewritten to provide faster compilation, better cross-platform support, and easier deployment. It uses lexington as back-end for parsing Fountain and offers a clean, modern user interface optimized for distraction-free writing.

Accolade was migrated from GTK4 to Fyne; see
[FYNE_TRANSITION.md](FYNE_TRANSITION.md) for the background and
[BUILD_STATUS.md](BUILD_STATUS.md) for what works today, how to test it,
and what is still missing.

## Building

### Prerequisites

- Go 1.21 or later
- GCC compiler
- X11 development libraries (Linux only)

On Ubuntu/Debian:
```bash
sudo apt-get install libx11-dev libxrandr-dev libxcursor-dev \
                     libxinerama-dev libxi-dev libgl1-mesa-dev \
                     build-essential pkg-config
```

Until the lexington fixes Accolade uses are tagged, `go.mod` builds
against a lexington checkout next to this one:

```bash
git clone https://github.com/LaPingvino/lexington ../lexington
```

### Quick Start

The easiest way to build Accolade:

```bash
./build-fyne.sh
```

### Manual Build

```bash
export CGO_ENABLED=1
go mod tidy
go build -v -o accolade .
```

### Using Nix

For development with the included Nix flake:

```bash
nix develop
go build -v -o accolade .
```

### Legacy GTK4 Build

The previous GTK4-based version has been replaced. To build the legacy version, check out commit `b6150ed` or earlier:

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
$ sudo cp data/eu.kiefte.Accolade.gschema.xml /usr/share/glib-2.0/schemas/eu.kiefte.Accolade.gschema.xml
$ sudo glib-compile-schemas /usr/share/glib-2.0/schemas
```

Once all dependencies are installed you can build accolade using the following commands:

```bash
$ git clone https://github.com/LaPingvino/accolade.git
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
