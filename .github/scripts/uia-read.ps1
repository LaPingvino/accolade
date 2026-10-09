# uia-read.ps1 -Title <part of the window title> [-Out <file>]
#
# Reads a running application through Windows UI Automation, the API
# screen readers (NVDA, JAWS, Narrator) and braille displays use: the
# window's tree with control types and names, and for text controls what
# TextPattern gives (the text, the selection, the line at the caret).
# Exits 1 if the window is not found.
param(
    [Parameter(Mandatory = $true)][string]$Title,
    [string]$Out = "uia-tree.txt",
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
        if ($w.Current.Name -like "*$Title*") { $window = $w }
    }
    if (-not $window) { Start-Sleep -Seconds 1 }
}
if (-not $window) {
    Write-Output "window '$Title' not found; top-level windows:"
    foreach ($w in $AE::RootElement.FindAll([System.Windows.Automation.TreeScope]::Children,
            [System.Windows.Automation.Condition]::TrueCondition)) { Write-Output "  $($w.Current.Name)" }
    exit 1
}

$lines = New-Object System.Collections.Generic.List[string]
$stats = @{ elements = 0; named = 0; text = 0 }

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
