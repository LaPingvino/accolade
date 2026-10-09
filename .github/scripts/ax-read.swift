// ax-read.swift <pid> (swift .github/scripts/ax-read.swift <pid>, on a Mac
// that allows the terminal accessibility access): reads a running application through the macOS
// accessibility API (what VoiceOver uses): its windows' trees with roles
// and labels, and for text areas the text, the selected range and the line
// at the caret. Exits 0 without reading when this process is not trusted
// for accessibility (a CI runner may not allow it), 1 when it can read but
// finds no text area.
import ApplicationServices
import Foundation

guard CommandLine.arguments.count > 1, let pid = Int32(CommandLine.arguments[1]) else {
    print("usage: ax-read <pid>")
    exit(2)
}
if !AXIsProcessTrusted() {
    print("not trusted for accessibility here: not read")
    exit(0)
}

func attr(_ e: AXUIElement, _ name: String) -> AnyObject? {
    var v: AnyObject?
    return AXUIElementCopyAttributeValue(e, name as CFString, &v) == .success ? v : nil
}

func param(_ e: AXUIElement, _ name: String, _ arg: AnyObject) -> AnyObject? {
    var v: AnyObject?
    return AXUIElementCopyParameterizedAttributeValue(e, name as CFString, arg, &v) == .success ? v : nil
}

var textAreas = 0
func walk(_ e: AXUIElement, _ depth: Int) {
    if depth > 25 { return }
    let role = attr(e, kAXRoleAttribute) as? String ?? "?"
    let label = attr(e, kAXDescriptionAttribute) as? String ?? (attr(e, kAXTitleAttribute) as? String ?? "")
    var line = String(repeating: "  ", count: depth) + "\(role) '\(label)'"
    if role == kAXTextAreaRole || role == kAXTextFieldRole {
        if role == kAXTextAreaRole { textAreas += 1 }
        let value = (attr(e, kAXValueAttribute) as? String ?? "").replacingOccurrences(of: "\n", with: "\\n")
        line += " VALUE='\(value.prefix(80))'"
        if let sel = attr(e, kAXSelectedTextRangeAttribute) {
            var range = CFRange()
            AXValueGetValue(sel as! AXValue, .cfRange, &range)
            line += " CARET=\(range.location)"
            if let n = param(e, kAXLineForIndexParameterizedAttribute, range.location as AnyObject) as? Int,
               let r = param(e, kAXRangeForLineParameterizedAttribute, n as AnyObject),
               let s = param(e, kAXStringForRangeParameterizedAttribute, r) as? String {
                line += " LINE@CARET='\(s.replacingOccurrences(of: "\n", with: "\\n"))'"
            }
        }
    }
    print(line)
    for c in attr(e, kAXChildrenAttribute) as? [AXUIElement] ?? [] { walk(c, depth + 1) }
}

let app = AXUIElementCreateApplication(pid)
for _ in 0..<30 {
    if let windows = attr(app, kAXWindowsAttribute) as? [AXUIElement], !windows.isEmpty {
        for w in windows { walk(w, 0) }
        if textAreas == 0 {
            print("no text area read")
            exit(1)
        }
        exit(0)
    }
    Thread.sleep(forTimeInterval: 1)
}
print("no window of process \(pid)")
exit(1)
