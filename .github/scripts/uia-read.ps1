# uia-read.ps1 -ProcessId <pid> [-Title <part of the window title>] [-Out <file>]
#
# Reads a running application through Windows UI Automation, the API
# screen readers (NVDA, JAWS, Narrator) and braille displays use: the
# window's tree with control types and names, and for text controls what
# TextPattern gives (the text, the selection, the line at the caret).
# Exits 1 if the window is not found.
param(
    [Parameter(Mandatory = $true)][int]$ProcessId,
    [string]$Title = "",
    [string]$Out = "uia-tree.txt",
    # fail unless an element's TextPattern holds this text
    [string]$ExpectText = "",
    # first move the focus to the element of this name (as a screen reader
    # can), then fail unless it has the focus and a line at its caret: the
    # line a braille display shows
    [string]$Focus = "",
    [string]$ExpectLine = "",
    [int]$WaitSeconds = 30
)

Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
$AE = [System.Windows.Automation.AutomationElement]
$TW = [System.Windows.Automation.TreeWalker]::RawViewWalker

$window = $null
for ($i = 0; $i -lt $WaitSeconds -and -not $window; $i++) {
    foreach ($w in $AE::RootElement.FindAll([System.Windows.Automation.TreeScope]::Children,
            [System.Windows.Automation.Condition]::TrueCondition)) {
        # the application's own window (a terminal's title can name it too)
        if ($w.Current.ProcessId -eq $ProcessId -and $w.Current.Name -like "*$Title*") { $window = $w }
    }
    if (-not $window) { Start-Sleep -Seconds 1 }
}
if (-not $window) {
    Write-Output "no window of process $ProcessId named '*$Title*'; top-level windows:"
    foreach ($w in $AE::RootElement.FindAll([System.Windows.Automation.TreeScope]::Children,
            [System.Windows.Automation.Condition]::TrueCondition)) { Write-Output "  [$($w.Current.ProcessId)] $($w.Current.Name)" }
    exit 1
}

if ($Focus) {
    $target = $window.FindFirst([System.Windows.Automation.TreeScope]::Descendants,
        (New-Object System.Windows.Automation.PropertyCondition($AE::NameProperty, $Focus)))
    if (-not $target) { Write-Output "nothing named '$Focus'"; exit 1 }
    $target.SetFocus()
    Start-Sleep -Seconds 2 # the next snapshots
    $f = $AE::FocusedElement
    Write-Output "focused: $($f.Current.ControlType.ProgrammaticName) '$($f.Current.Name)'"
    $tp = $null
    if (-not $target.Current.HasKeyboardFocus -or
        -not $target.TryGetCurrentPattern([System.Windows.Automation.TextPattern]::Pattern, [ref]$tp)) {
        Write-Output "'$Focus' did not get the focus, or has no TextPattern"
        exit 1
    }
    $sel = $tp.GetSelection()
    if ($sel.Length -lt 1) { Write-Output "no caret in '$Focus'"; exit 1 }
    $line = $sel[0].Clone()
    $line.ExpandToEnclosingUnit([System.Windows.Automation.Text.TextUnit]::Line)
    $lineText = $line.GetText(-1)
    # the next line, as a braille display's "next line" key reads it
    $next = $sel[0].Clone()
    [void]$next.Move([System.Windows.Automation.Text.TextUnit]::Line, 1)
    $next.ExpandToEnclosingUnit([System.Windows.Automation.Text.TextUnit]::Line)
    $word = $sel[0].Clone()
    $word.ExpandToEnclosingUnit([System.Windows.Automation.Text.TextUnit]::Word)
    Write-Output "line at the caret: '$($lineText -replace "`r?`n", "\n")'; next line: '$($next.GetText(-1) -replace "`r?`n", "\n")'; word: '$($word.GetText(-1))'"
    if ($ExpectLine -and -not $lineText.Contains($ExpectLine)) {
        Write-Output "the line at the caret is not '$ExpectLine'"
        exit 1
    }
}

$lines = New-Object System.Collections.Generic.List[string]
$stats = @{ elements = 0; named = 0; text = 0; expected = $false }

function Describe($el, $depth) {
    if ($depth -gt 30) { return }
    $c = $el.Current
    $stats.elements++
    if ($c.Name) { $stats.named++ }
    $type = $c.ControlType.ProgrammaticName -replace '^ControlType\.', ''
    $line = ("  " * $depth) + "$type '$($c.Name)'"
    if ($c.HasKeyboardFocus) { $line += " [focused]" }
    if ($c.IsKeyboardFocusable) { $line += " [focusable]" }
    $tp = $null
    if ($el.TryGetCurrentPattern([System.Windows.Automation.TextPattern]::Pattern, [ref]$tp)) {
        $stats.text++
        $whole = $tp.DocumentRange.GetText(-1)
        if ($ExpectText -and $whole.Contains($ExpectText)) { $stats.expected = $true }
        $doc = $tp.DocumentRange.GetText(200) -replace "`r?`n", "\n"
        $line += " TEXT='$doc'"
        $sel = $tp.GetSelection()
        if ($sel.Length -gt 0) {
            $r = $sel[0].Clone()
            $r.ExpandToEnclosingUnit([System.Windows.Automation.Text.TextUnit]::Line)
            $line += " LINE@CARET='$($r.GetText(200) -replace "`r?`n", "\n")'"
        }
    }
    $vp = $null
    if ($el.TryGetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern, [ref]$vp)) {
        $line += " VALUE='$($vp.Current.Value -replace "`r?`n", "\n")'"
    }
    $lines.Add($line)
    $child = $TW.GetFirstChild($el)
    while ($child) {
        Describe $child ($depth + 1)
        $child = $TW.GetNextSibling($child)
    }
}

Describe $window 0
$lines | Set-Content -Path $Out -Encoding utf8
$lines | Select-Object -First 120 | ForEach-Object { Write-Output $_ }
Write-Output "elements: $($stats.elements), named: $($stats.named), with TextPattern: $($stats.text)"
if ($ExpectText -and -not $stats.expected) {
    Write-Output "no TextPattern holds '$ExpectText'"
    exit 1
}
