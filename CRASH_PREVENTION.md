# Crash Prevention Guide

This guide helps you prevent crashes when using Accolade, particularly the GSettings-related crashes that can occur on some systems.

## Quick Fix

If Accolade crashes when clicking buttons (especially Open, Save, Save As), run it with:

```bash
export ACCOLADE_DISABLE_GSETTINGS=1
export GTK_USE_PORTAL=1
zig build run
```

## Understanding the Issue

The crashes typically happen because:
- Your system is missing GTK4 GSettings schemas
- The file chooser tries to access non-existent configuration keys
- GSettings backend is not properly configured

## Prevention Methods

### Method 1: Disable GSettings (Recommended)
```bash
export ACCOLADE_DISABLE_GSETTINGS=1
zig build run
```

This makes Accolade use file-based settings instead of GSettings.

### Method 2: Use GTK Portal
```bash
export GTK_USE_PORTAL=1
zig build run
```

This forces GTK to use the system's native file chooser.

### Method 3: Use Memory Backend
```bash
export GSETTINGS_BACKEND=memory
zig build run
```

This uses an in-memory GSettings backend that doesn't depend on system schemas.

### Method 4: Combine All Methods
```bash
export ACCOLADE_DISABLE_GSETTINGS=1
export GTK_USE_PORTAL=1
export GSETTINGS_BACKEND=memory
zig build run
```

## Permanent Solution

Add these lines to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.):

```bash
# Prevent Accolade crashes
export ACCOLADE_DISABLE_GSETTINGS=1
export GTK_USE_PORTAL=1
```

## Testing the Fix

After applying any method:

1. Start Accolade
2. Click the "Open" button - should not crash
3. Click the "Save As" button - should not crash
4. Try opening and saving files - should work normally

## System Requirements

To avoid these issues entirely, ensure your system has:
- Complete GTK4 development packages
- GSettings schemas for GTK4
- Portal support (xdg-desktop-portal)

On most Linux distributions:
```bash
# Ubuntu/Debian
sudo apt install gtk4-dev glib2.0-dev

# Fedora
sudo dnf install gtk4-devel glib2-devel

# Arch Linux
sudo pacman -S gtk4 glib2
```

## Troubleshooting

### Still Crashing?
1. Check the console output for error messages
2. Try running with all environment variables set
3. Ensure you're using the latest version of Accolade
4. Check if your display server supports portals

### Error Messages to Look For
- `GLib-GIO-ERROR`: GSettings schema issues
- `Settings schema does not contain a key`: Missing GSettings keys
- `Trace/breakpoint trap`: Fatal GSettings error

### Getting Help
If none of these methods work:
1. Note your Linux distribution and version
2. Note your GTK4 version (`gtk4-demo --version`)
3. Include the full error message
4. Report the issue with this information

## What Changed

The fixes include:
- Safe GSettings access with fallback values
- Environment variable controls for disabling problematic features
- Comprehensive error handling in file operations
- Graceful degradation when system components are missing

All functionality remains available even with GSettings disabled.