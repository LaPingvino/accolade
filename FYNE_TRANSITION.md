# Accolade Fyne Transition

This document describes the transition of Accolade from a GTK4-based application to a Fyne-based Go application.

## Overview

Accolade has been successfully converted from:
- **From**: Python/GTK4 with gotk4 Go bindings 
- **To**: Pure Go with Fyne UI framework

This transition was motivated by:
1. **Faster compilation**: Fyne compiles much faster than GTK4 bindings
2. **Better cross-platform support**: Fyne works on Windows, macOS, and Linux
3. **Simpler deployment**: Single binary with no external dependencies
4. **LLM-friendly**: Pure Go code is easier to work with than GTK4 bindings

## What's Been Implemented

### Core Application Structure
- ✅ Main application framework with Fyne
- ✅ Window management system
- ✅ Settings system with JSON persistence
- ✅ Theme system (Light, Dark, Sepia themes)
- ✅ Menu system and application lifecycle

### UI Components
- ✅ **HeaderBar**: Toolbar with file operations, view controls, and settings
- ✅ **SearchBar**: Find and replace (case, whole word, regex)
- ✅ **MainWindow**: Split-pane layout with editor and preview
- ✅ **PreferencesDialog**: Comprehensive settings dialog with tabs
- ✅ **ExportDialog**: Export to PDF, HTML, FDX, TXT and Fountain

### Editor Features
- ✅ Multi-line text editor with word wrap
- ✅ File operations (New, Open, Save, Save As)
- ✅ Screenplay formatting on Enter
- ✅ Auto-save
- ✅ Recent files management

### Fountain Support
- ✅ **FountainFormatter**: Basic Fountain parsing and formatting
- ✅ **Lexington library**: parsing and PDF/HTML/FDX output
- ✅ **Export**: PDF, HTML and FDX via lexington
- ✅ **Validation**: Framework for Fountain syntax validation

### Settings & Preferences
- ✅ Comprehensive settings system
- ✅ Theme switching (Light/Dark/Sepia)
- ✅ Editor preferences (font, word wrap, etc.)
- ✅ Window state persistence
- ✅ Export settings
- ✅ Fountain-specific options

## Architecture

### File Structure
```
accolade/
├── main.go                 # Application entry point
├── window.go              # Main window implementation
├── header_bar.go          # Toolbar implementation
├── search_bar.go          # Find/replace functionality
├── preferences_dialog.go   # Settings dialog
├── export_dialog.go       # Export functionality
├── settings.go            # Configuration management
├── themes.go              # Custom theme definitions
├── fountain_formatter.go  # Fountain syntax handling
├── lexington_converter.go # External tool integration
├── spell_checker.go       # Spell checking (stub)
├── go.mod                 # Go module definition
├── flake.nix             # Nix development environment
└── build-fyne.sh         # Build script
```

### Key Design Decisions

1. **Pure Go Implementation**: No CGO dependencies for GTK4, only for Fyne's OpenGL/X11 requirements
2. **Component-based Architecture**: Each UI component is self-contained with its own file
3. **Settings Persistence**: JSON-based configuration instead of GSettings
4. **Theme System**: Custom Fyne themes for better writing experience
5. **Split Architecture**: Clear separation between UI and business logic

## Building and Running

### Prerequisites

#### System Dependencies
- Go 1.21 or later
- GCC compiler
- X11 development libraries:
  - `libx11-dev`
  - `libxrandr-dev` 
  - `libxcursor-dev`
  - `libxinerama-dev`
  - `libxi-dev`
  - `libgl1-mesa-dev`
  - `pkg-config`

#### Ubuntu/Debian Installation
```bash
sudo apt-get update
sudo apt-get install libx11-dev libxrandr-dev libxcursor-dev \
                     libxinerama-dev libxi-dev libgl1-mesa-dev \
                     build-essential pkg-config
```

#### Fedora/RHEL Installation
```bash
sudo dnf install libX11-devel libXrandr-devel libXcursor-devel \
                 libXinerama-devel libXi-devel mesa-libGL-devel \
                 gcc pkg-config
```

### Building

#### Using the Build Script (Recommended)
```bash
./build-fyne.sh
```

#### Manual Build
```bash
export CGO_ENABLED=1
go mod tidy
go build -v -o accolade .
```

#### Using Nix (Development Environment)
```bash
nix develop
go build -v -o accolade .
```

### Running
```bash
./accolade
```

## Current Status

The feature-by-feature status, build instructions and known limitations
now live in [BUILD_STATUS.md](BUILD_STATUS.md). In short: editing,
formatting on Enter, find/replace, undo/redo, keyboard shortcuts,
auto-save, the title page dialog, themes, and PDF/HTML/FDX export work;
spell checking, DOCX export, focus mode and syntax highlighting do not
yet.

## Migration Notes

### For Users
- **Settings**: Previous GTK settings won't carry over (different storage format)
- **Themes**: New custom themes available (Light, Dark, Sepia)
- **Performance**: Should be faster to start and use less memory
- **Dependencies**: Fewer system dependencies required

### For Developers
- **Build Time**: Significantly faster compilation
- **Code Style**: Pure Go instead of GTK4 bindings
- **Testing**: Easier to unit test UI components
- **Cross-platform**: Can build for Windows and macOS now
- **Deployment**: Single binary deployment

## Development

### Adding New Features
1. Create component in separate file (e.g., `new_feature.go`)
2. Add to main window in `window.go`
3. Update settings if needed in `settings.go`
4. Add to preferences dialog if user-configurable

### Testing
```bash
go test ./...
```

### Code Style
- Follow standard Go conventions
- Use Fyne widgets and containers appropriately
- Keep UI logic separate from business logic
- Document public APIs

## Future Improvements

1. A dedicated editor widget (or upstream Fyne support) for selection,
   highlighting, Fountain syntax colouring and line numbers
2. Spell checking
3. DOCX export
4. Focus mode, file drop

## Conclusion

The Fyne transition successfully modernizes Accolade while maintaining its core functionality. The application is now:

- **Faster to build and deploy**
- **More cross-platform friendly**
- **Easier to maintain and extend**
- **LLM development friendly**

While some advanced features from GTK4 are not immediately available in Fyne, the core writing experience remains excellent, and the foundation is solid for future enhancements.

The transition demonstrates that Fyne is a viable alternative to GTK4 for desktop applications, especially when targeting multiple platforms and prioritizing development velocity.