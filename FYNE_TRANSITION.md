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
- ✅ **SearchBar**: Find and replace functionality (UI complete, search logic TODO)
- ✅ **MainWindow**: Split-pane layout with editor and preview
- ✅ **PreferencesDialog**: Comprehensive settings dialog with tabs
- ✅ **ExportDialog**: Export to PDF, HTML, DOCX, TXT, and Fountain formats

### Editor Features
- ✅ Multi-line text editor with word wrap
- ✅ File operations (New, Open, Save, Save As)
- ✅ Basic Fountain syntax highlighting preparation
- ✅ Auto-save capability (framework in place)
- ✅ Recent files management

### Fountain Support
- ✅ **FountainFormatter**: Basic Fountain parsing and formatting
- ✅ **LexingtonConverter**: Integration with Lexington tool for advanced processing
- ✅ **Basic export**: Fountain to HTML conversion
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

## Features Comparison

| Feature | GTK4 Version | Fyne Version | Status |
|---------|-------------|--------------|--------|
| Text Editor | ✅ GtkSourceView | ✅ Fyne Entry | Complete |
| File Operations | ✅ | ✅ | Complete |
| Find/Replace | ✅ | 🔄 | UI Complete, Logic TODO |
| Preferences | ✅ | ✅ | Complete |
| Themes | ✅ | ✅ | Complete + Custom |
| Preview | ✅ WebKit | ✅ HTML Preview | Basic Complete |
| Export | ✅ | ✅ | Framework Complete |
| Spell Check | ✅ | ⏳ | Stub Only |
| Auto-save | ✅ | ⏳ | Framework Ready |
| Session Restore | ✅ | ⏳ | TODO |

Legend: ✅ Complete, 🔄 Partial, ⏳ TODO, ❌ Not Planned

## Known Limitations

### Fyne Framework Limitations
1. **Text Editor**: Less sophisticated than GtkSourceView
   - No syntax highlighting built-in
   - Limited text manipulation APIs
   - No line numbers display
   - Basic find/replace functionality

2. **File Dialogs**: Simpler than GTK file dialogs
   - No custom file filters (commented out)
   - Limited file type handling

3. **Themes**: Less extensive theming than GTK
   - Custom themes implemented for key colors
   - Font customization is limited

### TODO Items
1. **Search Functionality**: Complete find/replace implementation
2. **Syntax Highlighting**: Custom Fountain syntax highlighting
3. **Spell Checking**: Integration with system spell checker
4. **Auto-save**: Implement debounced auto-save
5. **Session Restore**: Restore open files on startup
6. **Advanced Preview**: Better Fountain to HTML conversion
7. **Keyboard Shortcuts**: Fyne has limited shortcut support
8. **Drag & Drop**: File drop support
9. **Line Numbers**: Custom text widget with line numbers

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

### Short Term
1. Complete find/replace functionality
2. Implement basic syntax highlighting
3. Add spell checking integration
4. Improve export formats

### Medium Term
1. Plugin system for export formats
2. Custom text editor widget with line numbers
3. Advanced Fountain features (dual dialogue, etc.)
4. Collaborative editing features

### Long Term
1. Mobile app versions (Fyne supports mobile)
2. Web version (Fyne can compile to WebAssembly)
3. Advanced screenplay formatting
4. Integration with screenplay databases

## Conclusion

The Fyne transition successfully modernizes Accolade while maintaining its core functionality. The application is now:

- **Faster to build and deploy**
- **More cross-platform friendly**
- **Easier to maintain and extend**
- **LLM development friendly**

While some advanced features from GTK4 are not immediately available in Fyne, the core writing experience remains excellent, and the foundation is solid for future enhancements.

The transition demonstrates that Fyne is a viable alternative to GTK4 for desktop applications, especially when targeting multiple platforms and prioritizing development velocity.