# GSettings Crash Fixes

This document describes the fixes implemented to prevent GSettings-related crashes in Accolade, particularly the file chooser crash that occurred when clicking buttons.

## Problem Description

The application was crashing with the following error when trying to open file chooser dialogs:

```
GLib-GIO-ERROR: Settings schema 'org.gtk.gtk4.Settings.FileChooser' does not contain a key named 'view-type'
Trace/breakpoint trap (core dumped)
```

This crash occurred because:
1. GTK4 file chooser dialogs automatically try to load settings from GSettings
2. The system didn't have the required GSettings schema or the schema was missing required keys
3. GTK would crash when trying to access non-existent settings keys

## Root Cause Analysis

The issue stemmed from several factors:
1. **Missing GSettings schemas**: The system didn't have all required GTK4 schemas installed
2. **Unsafe GSettings access**: The code didn't check if keys existed before accessing them
3. **No fallback mechanism**: When GSettings failed, there was no graceful degradation
4. **File chooser implementation**: Using `GtkFileChooserDialog` instead of `GtkFileChooserNative`

## Solutions Implemented

### 1. Safe GSettings Access

Added comprehensive error handling and fallback mechanisms in `src/settings.zig`:

```zig
// Safe GSettings access methods with error handling and fallbacks
fn safeGetString(self: *Self, gs: *c.GSettings, key: [*c]const u8) ?[*c]u8
fn safeGetInt(self: *Self, gs: *c.GSettings, key: [*c]const u8, default_value: i32) i32
fn safeGetDouble(self: *Self, gs: *c.GSettings, key: [*c]const u8, default_value: f64) f64
fn safeGetBoolean(self: *Self, gs: *c.GSettings, key: [*c]const u8, default_value: bool) bool
fn keyExists(self: *Self, gs: *c.GSettings, key: [*c]const u8) bool
```

### 2. Environment Variable Controls

Added environment variables to control GSettings behavior:

- `ACCOLADE_DISABLE_GSETTINGS=1`: Completely disable GSettings and use file-based settings
- `GTK_USE_PORTAL=1`: Force GTK to use the portal backend for file choosers
- `GSETTINGS_BACKEND=memory`: Use in-memory GSettings backend for testing

### 3. Native File Chooser

Replaced `GtkFileChooserDialog` with `GtkFileChooserNative`:

```zig
// Old approach (crash-prone)
const dialog = c.gtk_file_chooser_dialog_new(...)

// New approach (crash-resistant)
const dialog = c.gtk_file_chooser_native_new(...)
```

Benefits:
- Uses native system file picker
- Doesn't depend on GTK's internal GSettings
- Better user experience
- More robust error handling

### 4. Comprehensive Error Handling

Added proper error handling for all GSettings operations:

```zig
fn init_gsettings(self: *Self) void {
    // Check if GSettings is explicitly disabled
    if (std.posix.getenv("ACCOLADE_DISABLE_GSETTINGS") != null) {
        self.gsettings = null;
        return;
    }
    
    // Check schema source availability
    const schema_source = c.g_settings_schema_source_get_default();
    if (schema_source == null) {
        std.log.warn("GSettings schema source not available, using file-based settings", .{});
        self.gsettings = null;
        return;
    }
    
    // Additional safety checks...
}
```

### 5. Automatic Portal Setup

The settings initialization now automatically sets up GTK portal usage:

```zig
// Set environment variable to prevent GTK file chooser crashes
if (std.posix.getenv("GTK_USE_PORTAL") == null) {
    std.log.info("Setting GTK_USE_PORTAL=1 to prevent file chooser crashes", .{});
    _ = std.posix.setenv("GTK_USE_PORTAL", "1", 1) catch {};
}
```

## Testing

### Integration Tests

Added comprehensive integration tests in `tests/integration/`:

1. **GSettings Tests** (`gsettings_test.zig`):
   - Test initialization with missing schemas
   - Test environment variable controls
   - Test safe access methods
   - Test fallback mechanisms

2. **File Chooser Tests** (`file_chooser_test.zig`):
   - Test dialog creation
   - Test different file types
   - Test error handling
   - Test memory management

### Test Environment

Created `test-gsettings-safe.sh` script that:
- Sets up safe testing environment
- Prevents GSettings crashes during testing
- Runs all tests with proper isolation
- Provides usage instructions

## Usage Instructions

### For End Users

To run the application safely:

```bash
# Option 1: Use GTK portal (recommended)
export GTK_USE_PORTAL=1
zig build run

# Option 2: Disable GSettings completely
export ACCOLADE_DISABLE_GSETTINGS=1
zig build run

# Option 3: Use memory backend for testing
export GSETTINGS_BACKEND=memory
zig build run
```

### For Developers

To run tests safely:

```bash
# Run the safe test script
./test-gsettings-safe.sh

# Or run individual test suites
zig build test-gsettings-integration
zig build test-file-chooser-integration
```

## Performance Impact

The fixes have minimal performance impact:
- GSettings operations are cached where possible
- Fallback to file-based settings is fast
- Native file chooser is often faster than GTK dialogs
- Error checking adds negligible overhead

## Compatibility

The fixes maintain backward compatibility:
- Applications with proper GSettings schemas continue to work normally
- File-based settings provide full functionality
- All existing features remain available
- No breaking changes to the API

## Future Improvements

1. **Schema Installation**: Consider providing GSettings schema files for proper system integration
2. **Settings Migration**: Automatic migration from GSettings to file-based settings when needed
3. **User Preferences**: Allow users to choose their preferred settings backend
4. **Diagnostics**: Add better diagnostic information for GSettings issues

## Verification

To verify the fixes work:

1. Run the application in an environment without GTK schemas
2. Click file operation buttons (Open, Save, Save As)
3. Verify no crashes occur
4. Check that file operations work correctly
5. Verify settings are saved and loaded properly

### Test Results

The fixes have been successfully tested and verified:

✅ **Application Startup**: No crashes during initialization
✅ **GSettings Handling**: Graceful fallback to file-based settings when GSettings is disabled
✅ **File Operations**: File chooser dialogs work without crashing
✅ **Error Handling**: Missing GSettings keys don't cause crashes
✅ **Environment Variables**: `ACCOLADE_DISABLE_GSETTINGS=1` successfully disables GSettings
✅ **Compilation**: All code compiles without errors in Zig 0.14.1
✅ **Logging**: Appropriate warnings and info messages are displayed

### Verified Fixes

1. **Safe GSettings Access**: All GSettings operations now use safe wrapper functions with fallbacks
2. **Missing Key Protection**: Application handles missing GSettings keys gracefully
3. **Environment Controls**: Users can disable GSettings completely via environment variable
4. **File Chooser Stability**: File operations no longer crash due to missing GTK schemas
5. **Comprehensive Testing**: Integration tests verify all scenarios work correctly

The fixes ensure that Accolade remains stable and functional regardless of the system's GSettings configuration.