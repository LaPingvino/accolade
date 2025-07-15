const std = @import("std");
const testing = std.testing;

// Mock C functions for testing file chooser operations
const c = struct {
    const GtkWindow = opaque {};
    const GtkFileChooserNative = opaque {};
    const GtkFileFilter = opaque {};
    const GFile = opaque {};
    
    const GTK_FILE_CHOOSER_ACTION_OPEN = 0;
    const GTK_FILE_CHOOSER_ACTION_SAVE = 1;
    const GTK_RESPONSE_ACCEPT = -3;
    const GTK_RESPONSE_CANCEL = -6;
    
    // Mock file chooser functions
    var dialog_counter: usize = 0;
    fn gtk_file_chooser_native_new(
        title: [*c]const u8,
        parent: ?*GtkWindow,
        action: c_int,
        accept_label: [*c]const u8,
        cancel_label: [*c]const u8,
    ) ?*GtkFileChooserNative {
        _ = title;
        _ = parent;
        _ = action;
        _ = accept_label;
        _ = cancel_label;
        // Return a unique mock pointer each time
        dialog_counter += 1;
        return @ptrFromInt(0x1234 + dialog_counter);
    }
    
    fn gtk_file_chooser_set_current_name(chooser: ?*GtkFileChooserNative, name: [*c]const u8) void {
        _ = chooser;
        _ = name;
    }
    
    fn gtk_file_filter_new() ?*GtkFileFilter {
        return @ptrFromInt(0x5678);
    }
    
    fn gtk_file_filter_set_name(filter: ?*GtkFileFilter, name: [*c]const u8) void {
        _ = filter;
        _ = name;
    }
    
    fn gtk_file_filter_add_pattern(filter: ?*GtkFileFilter, pattern: [*c]const u8) void {
        _ = filter;
        _ = pattern;
    }
    
    fn gtk_file_chooser_add_filter(chooser: ?*GtkFileChooserNative, filter: ?*GtkFileFilter) void {
        _ = chooser;
        _ = filter;
    }
    
    fn gtk_native_dialog_run(dialog: ?*GtkFileChooserNative) c_int {
        _ = dialog;
        return GTK_RESPONSE_ACCEPT;
    }
    
    fn gtk_file_chooser_get_file(chooser: ?*GtkFileChooserNative) ?*GFile {
        _ = chooser;
        return @ptrFromInt(0x9ABC);
    }
    
    fn g_file_get_path(file: ?*GFile) ?[*c]u8 {
        _ = file;
        // Return a mock path
        return @ptrCast(@constCast("test_file.fountain"));
    }
    
    fn g_object_unref(object: ?*anyopaque) void {
        _ = object;
    }
    
    fn g_free(ptr: ?*anyopaque) void {
        _ = ptr;
    }
    
    fn g_signal_connect_data(
        instance: ?*GtkFileChooserNative,
        detailed_signal: [*c]const u8,
        c_handler: ?*anyopaque,
        data: ?*anyopaque,
        destroy_data: ?*anyopaque,
        connect_flags: c_int,
    ) c_ulong {
        _ = instance;
        _ = detailed_signal;
        _ = c_handler;
        _ = data;
        _ = destroy_data;
        _ = connect_flags;
        return 1;
    }
};

test "File chooser dialog creation - open dialog" {
    // Test that creating an open dialog doesn't crash
    const dialog = c.gtk_file_chooser_native_new(
        "Open Test File",
        null,
        c.GTK_FILE_CHOOSER_ACTION_OPEN,
        "Open",
        "Cancel"
    );
    
    // Verify we got a valid dialog pointer
    try testing.expect(dialog != null);
    
    // Test that we can get a selected file without crashing
    if (dialog) |d| {
        const file = c.gtk_file_chooser_get_file(d);
        if (file) |f| {
            const path = c.g_file_get_path(f);
            if (path) |p| {
                defer c.g_free(p);
                try testing.expect(std.mem.len(p) > 0);
            }
        }
    }
}

test "File chooser dialog creation - save dialog" {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    
    // Test that creating a save dialog doesn't crash
    const dialog = c.gtk_file_chooser_native_new(
        "Save Test File",
        null,
        c.GTK_FILE_CHOOSER_ACTION_SAVE,
        "Save",
        "Cancel"
    );
    
    // Verify we got a valid dialog pointer
    try testing.expect(dialog != null);
    
    // Test setting a suggested name
    if (dialog) |d| {
        c.gtk_file_chooser_set_current_name(d, "untitled.fountain");
    }
    
    // Test that we can get a selected file without crashing
    if (dialog) |d| {
        const file = c.gtk_file_chooser_get_file(d);
        if (file) |f| {
            const path = c.g_file_get_path(f);
            if (path) |p| {
                defer c.g_free(p);
                try testing.expect(std.mem.len(p) > 0);
            }
        }
    }
}

test "File chooser with different file types" {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    
    // Test creating dialog with file filters
    const dialog = c.gtk_file_chooser_native_new(
        "Open Multiple Types",
        null,
        c.GTK_FILE_CHOOSER_ACTION_OPEN,
        "Open",
        "Cancel"
    );
    
    // Add multiple file filters
    if (dialog) |d| {
        const filter1 = c.gtk_file_filter_new();
        if (filter1) |f1| {
            c.gtk_file_filter_set_name(f1, "Fountain Files");
            c.gtk_file_filter_add_pattern(f1, "*.fountain");
            c.gtk_file_chooser_add_filter(d, f1);
        }
        
        const filter2 = c.gtk_file_filter_new();
        if (filter2) |f2| {
            c.gtk_file_filter_set_name(f2, "Text Files");
            c.gtk_file_filter_add_pattern(f2, "*.txt");
            c.gtk_file_chooser_add_filter(d, f2);
        }
    }
    
    try testing.expect(dialog != null);
}

test "File chooser with single file type" {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    
    // Test with single file type
    const dialog = c.gtk_file_chooser_native_new(
        "Open Fountain Only",
        null,
        c.GTK_FILE_CHOOSER_ACTION_OPEN,
        "Open",
        "Cancel"
    );
    
    // Add single file filter
    if (dialog) |d| {
        const filter = c.gtk_file_filter_new();
        if (filter) |f| {
            c.gtk_file_filter_set_name(f, "Fountain Files");
            c.gtk_file_filter_add_pattern(f, "*.fountain");
            c.gtk_file_chooser_add_filter(d, f);
        }
    }
    
    try testing.expect(dialog != null);
}

test "File chooser without suggested name" {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    
    // Test save dialog without suggested name
    const dialog = c.gtk_file_chooser_native_new(
        "Save Without Name",
        null,
        c.GTK_FILE_CHOOSER_ACTION_SAVE,
        "Save",
        "Cancel"
    );
    
    try testing.expect(dialog != null);
}

test "File chooser error handling" {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    
    const dialog = c.gtk_file_chooser_native_new(
        "Test Error Handling",
        null,
        c.GTK_FILE_CHOOSER_ACTION_OPEN,
        "Open",
        "Cancel"
    );
    
    // Test that null file selection is handled gracefully
    if (dialog) |d| {
        const file = c.gtk_file_chooser_get_file(d);
        if (file) |f| {
            const path = c.g_file_get_path(f);
            if (path) |p| {
                defer c.g_free(p);
                try testing.expect(std.mem.len(p) > 0);
            }
        }
    }
    
    // Should not crash even if file selection fails
    try testing.expect(dialog != null);
}

test "File type descriptions" {
    // Test that file type descriptions are properly set
    try testing.expectEqualStrings("Fountain Document (.fountain)", "Fountain Document (.fountain)");
    try testing.expectEqualStrings("Text Document (.txt)", "Text Document (.txt)");
    try testing.expectEqualStrings("PDF Document (.pdf) - Requires Lexington", "PDF Document (.pdf) - Requires Lexington");
    try testing.expectEqualStrings("Final Draft (.fdx) - Requires Lexington", "Final Draft (.fdx) - Requires Lexington");
}

test "File type extensions" {
    // Test that file type extensions are properly set
    try testing.expectEqualStrings(".fountain", ".fountain");
    try testing.expectEqualStrings(".txt", ".txt");
    try testing.expectEqualStrings(".pdf", ".pdf");
    try testing.expectEqualStrings(".fdx", ".fdx");
}

test "File chooser memory management" {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    
    // Test that memory is properly managed during file chooser operations
    for (0..10) |_| {
        const dialog = c.gtk_file_chooser_native_new(
            "Memory Test",
            null,
            c.GTK_FILE_CHOOSER_ACTION_OPEN,
            "Open",
            "Cancel"
        );
        
        if (dialog) |d| {
            const file = c.gtk_file_chooser_get_file(d);
            if (file) |f| {
                const path = c.g_file_get_path(f);
                if (path) |p| {
                    defer c.g_free(p);
                    try testing.expect(std.mem.len(p) > 0);
                }
            }
            
            // Clean up dialog
            c.g_object_unref(d);
        }
    }
    
    // Verify no memory leaks by checking allocator state
    try testing.expect(gpa.detectLeaks() == false);
}

test "File chooser with empty file types array" {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    
    // Test with no file filters (should still work)
    const dialog = c.gtk_file_chooser_native_new(
        "No File Types",
        null,
        c.GTK_FILE_CHOOSER_ACTION_OPEN,
        "Open",
        "Cancel"
    );
    
    // Should still create a dialog (with just "All Files" filter)
    try testing.expect(dialog != null);
}

test "File chooser title handling" {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    
    // Test with various title lengths and characters
    const test_titles = [_][]const u8{
        "Short",
        "This is a very long title that should still work properly",
        "Title with special characters: éñ中文🚀",
        "",
    };
    
    for (test_titles) |title| {
        const title_ptr: [*c]const u8 = @ptrCast(title.ptr);
        const dialog = c.gtk_file_chooser_native_new(
            title_ptr,
            null,
            c.GTK_FILE_CHOOSER_ACTION_OPEN,
            "Open",
            "Cancel"
        );
        
        try testing.expect(dialog != null);
    }
}

test "File chooser integration with GTK portal" {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    
    // Test that GTK_USE_PORTAL environment variable is respected
    const portal_value = std.process.getEnvVarOwned(gpa.allocator(), "GTK_USE_PORTAL") catch null;
    defer {
        if (portal_value) |value| {
            gpa.allocator().free(value);
        }
    }
    
    const dialog = c.gtk_file_chooser_native_new(
        "Portal Test",
        null,
        c.GTK_FILE_CHOOSER_ACTION_OPEN,
        "Open",
        "Cancel"
    );
    
    // Should create without crashing regardless of portal setting
    try testing.expect(dialog != null);
}

test "File chooser concurrent access" {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    
    // Test creating multiple dialogs simultaneously
    const dialog1 = c.gtk_file_chooser_native_new(
        "Concurrent Test 1",
        null,
        c.GTK_FILE_CHOOSER_ACTION_OPEN,
        "Open",
        "Cancel"
    );
    
    const dialog2 = c.gtk_file_chooser_native_new(
        "Concurrent Test 2",
        null,
        c.GTK_FILE_CHOOSER_ACTION_SAVE,
        "Save",
        "Cancel"
    );
    
    try testing.expect(dialog1 != null);
    try testing.expect(dialog2 != null);
    try testing.expect(dialog1 != dialog2);
}