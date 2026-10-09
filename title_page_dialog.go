package main

import (
	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type TitlePageDialog struct {
	window *MainWindow
	dialog dialog.Dialog

	// Title page fields
	titleEntry     *widget.Entry
	creditEntry    *widget.Entry
	authorEntry    *widget.Entry
	basedOnEntry   *widget.Entry
	draftDateEntry *widget.Entry
	contactEntry   *widget.Entry

	// Contact details
	nameEntry    *widget.Entry
	addressEntry *widget.Entry
	phoneEntry   *widget.Entry
	emailEntry   *widget.Entry

	// Options
	includeTitlePage *widget.Check
	includeContact   *widget.Check
	includeDate      *widget.Check

	// Buttons
	insertButton  *widget.Button
	previewButton *widget.Button
	cancelButton  *widget.Button
}

func NewTitlePageDialog(window *MainWindow) *TitlePageDialog {
	tpd := &TitlePageDialog{
		window: window,
	}

	tpd.createWidgets()
	tpd.loadDefaults()
	content := tpd.createContent()

	// its own buttons (NewCustom would add an empty dismiss button)
	d := dialog.NewCustomWithoutButtons(
		"Title Page Configuration",
		content,
		window.fyneWindow,
	)
	d.SetOnEscape(d.Hide) // as its Cancel button
	tpd.dialog = d

	tpd.dialog.Resize(fyne.NewSize(500, 600))

	return tpd
}

func (tpd *TitlePageDialog) createWidgets() {
	// Main title page fields
	tpd.titleEntry = widget.NewEntry()
	tpd.titleEntry.SetPlaceHolder("Your Screenplay Title")

	tpd.creditEntry = widget.NewEntry()
	tpd.creditEntry.SetText("Written by")

	tpd.authorEntry = widget.NewEntry()
	tpd.authorEntry.SetPlaceHolder("Your Name")

	tpd.basedOnEntry = widget.NewEntry()
	tpd.basedOnEntry.SetPlaceHolder("Based on... (optional)")

	tpd.draftDateEntry = widget.NewEntry()
	tpd.draftDateEntry.SetText(time.Now().Format("January 2, 2006"))

	// Contact information
	tpd.nameEntry = widget.NewEntry()
	tpd.nameEntry.SetPlaceHolder("Your Name")

	tpd.addressEntry = widget.NewEntry()
	tpd.addressEntry.SetPlaceHolder("Your Address\nCity, State ZIP")

	tpd.phoneEntry = widget.NewEntry()
	tpd.phoneEntry.SetPlaceHolder("(555) 123-4567")

	tpd.emailEntry = widget.NewEntry()
	tpd.emailEntry.SetPlaceHolder("your.email@example.com")

	// Options
	tpd.includeTitlePage = widget.NewCheck("Include title page", nil)
	tpd.includeTitlePage.SetChecked(true)

	tpd.includeContact = widget.NewCheck("Include contact information", nil)
	tpd.includeContact.SetChecked(true)

	tpd.includeDate = widget.NewCheck("Include draft date", nil)
	tpd.includeDate.SetChecked(true)

	// Buttons
	tpd.insertButton = widget.NewButton("Insert Title Page", tpd.insertTitlePage)
	tpd.previewButton = widget.NewButton("Preview", tpd.previewTitlePage)
	tpd.cancelButton = widget.NewButton("Cancel", func() {
		tpd.dialog.Hide()
	})
}

func (tpd *TitlePageDialog) loadDefaults() {
	// Try to load from settings or use sensible defaults
	settings := tpd.window.settings

	if settings != nil {
		tpd.titleEntry.SetText(settings.GetString("title-page-title"))
		tpd.authorEntry.SetText(settings.GetString("title-page-author"))
		tpd.nameEntry.SetText(settings.GetString("contact-name"))
		tpd.addressEntry.SetText(settings.GetString("contact-address"))
		tpd.phoneEntry.SetText(settings.GetString("contact-phone"))
		tpd.emailEntry.SetText(settings.GetString("contact-email"))
	}

	// the script's own title page, if it has one, before the settings'
	tpd.loadFromScript()

	// Set default title if empty
	if tpd.titleEntry.Text == "" {
		tpd.titleEntry.SetText("UNTITLED SCREENPLAY")
	}
}

func (tpd *TitlePageDialog) createContent() fyne.CanvasObject {
	// Title information section
	titleSection := widget.NewCard("Title Information", "",
		container.NewVBox(
			container.NewBorder(nil, nil, widget.NewLabel("Title:"), nil, tpd.titleEntry),
			container.NewBorder(nil, nil, widget.NewLabel("Credit:"), nil, tpd.creditEntry),
			container.NewBorder(nil, nil, widget.NewLabel("Author:"), nil, tpd.authorEntry),
			container.NewBorder(nil, nil, widget.NewLabel("Based on:"), nil, tpd.basedOnEntry),
			container.NewBorder(nil, nil, widget.NewLabel("Draft date:"), nil, tpd.draftDateEntry),
		),
	)

	// Contact information section
	contactSection := widget.NewCard("Contact Information", "",
		container.NewVBox(
			container.NewBorder(nil, nil, widget.NewLabel("Name:"), nil, tpd.nameEntry),
			container.NewBorder(nil, nil, widget.NewLabel("Address:"), nil, tpd.addressEntry),
			container.NewBorder(nil, nil, widget.NewLabel("Phone:"), nil, tpd.phoneEntry),
			container.NewBorder(nil, nil, widget.NewLabel("Email:"), nil, tpd.emailEntry),
		),
	)

	// Options section
	optionsSection := widget.NewCard("Options", "",
		container.NewVBox(
			tpd.includeTitlePage,
			tpd.includeContact,
			tpd.includeDate,
		),
	)

	// the buttons stay below the form, which scrolls
	buttonContainer := container.NewHBox(
		tpd.previewButton,
		layout.NewSpacer(),
		tpd.cancelButton,
		tpd.insertButton,
	)
	scrollContent := container.NewVScroll(container.NewVBox(titleSection, contactSection, optionsSection))
	scrollContent.SetMinSize(fyne.NewSize(480, 480))
	return container.NewBorder(nil, container.NewVBox(widget.NewSeparator(), buttonContainer), nil, nil, scrollContent)
}

func (tpd *TitlePageDialog) insertTitlePage() {
	if !tpd.includeTitlePage.Checked {
		tpd.dialog.Hide()
		return
	}

	titlePageText := tpd.generateTitlePageText()
	currentText := tpd.window.textEditor.Text()
	lines := strings.Split(currentText, "\n")
	end := titlePageEnd(lines)
	// fields the dialog does not edit (Copyright, Notes, ...) stay
	for _, f := range titleFields(lines[:end]) {
		if !dialogField[strings.ToLower(f.key)] {
			titlePageText += "\n" + strings.Join(f.raw, "\n")
		}
	}
	// the old title page (and the blank lines after it) replaced, as
	// one step that undo takes back
	oldEnd := len([]rune(strings.Join(lines[:end], "\n")))
	if end > 0 && end < len(lines) {
		oldEnd++ // the line break before the script
	}
	if end == 0 { // no title page: the blank lines before the script go
		oldEnd = len(currentText) - len(strings.TrimLeft(currentText, "\n"))
	}
	tpd.window.textEditor.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			b.Replace(0, oldEnd, titlePageText+"\n\n")
			b.SetCursor(0, false)
		})
	})

	// Save settings
	tpd.saveSettings()

	// Mark as changed
	tpd.window.hasChanges = true
	tpd.window.updateTitle()

	tpd.dialog.Hide()
}

func (tpd *TitlePageDialog) previewTitlePage() {
	titlePageText := tpd.generateTitlePageText()

	// Create preview dialog
	previewEntry := widget.NewMultiLineEntry()
	previewEntry.SetText(titlePageText)
	previewEntry.Disable() // Read-only

	previewContent := container.NewScroll(previewEntry)
	previewContent.Resize(fyne.NewSize(500, 400))

	previewDialog := dialog.NewCustom(
		"Title Page Preview",
		"Close",
		previewContent,
		tpd.window.fyneWindow,
	)

	previewDialog.Resize(fyne.NewSize(550, 450))
	previewDialog.Show()
}

func (tpd *TitlePageDialog) generateTitlePageText() string {
	var lines []string

	// Title (will be centered in formatted output)
	if tpd.titleEntry.Text != "" {
		lines = append(lines, "Title: "+tpd.titleEntry.Text)
	}

	// Credit line
	if tpd.creditEntry.Text != "" {
		lines = append(lines, "Credit: "+tpd.creditEntry.Text)
	}

	// Author
	if tpd.authorEntry.Text != "" {
		lines = append(lines, "Author: "+tpd.authorEntry.Text)
	}

	// Based on (source)
	if tpd.basedOnEntry.Text != "" {
		lines = append(lines, "Source: "+tpd.basedOnEntry.Text)
	}

	// Draft date
	if tpd.includeDate.Checked && tpd.draftDateEntry.Text != "" {
		lines = append(lines, "Draft date: "+tpd.draftDateEntry.Text)
	}

	// Contact information (will be positioned at bottom left in formatted output)
	if tpd.includeContact.Checked {
		var contactLines []string
		if tpd.nameEntry.Text != "" {
			contactLines = append(contactLines, tpd.nameEntry.Text)
		}
		if tpd.addressEntry.Text != "" {
			// Handle multi-line addresses
			addressLines := strings.Split(tpd.addressEntry.Text, "\n")
			for _, addressLine := range addressLines {
				if strings.TrimSpace(addressLine) != "" {
					contactLines = append(contactLines, strings.TrimSpace(addressLine))
				}
			}
		}
		if tpd.phoneEntry.Text != "" {
			contactLines = append(contactLines, tpd.phoneEntry.Text)
		}
		if tpd.emailEntry.Text != "" {
			contactLines = append(contactLines, tpd.emailEntry.Text)
		}

		if len(contactLines) > 0 {
			lines = append(lines, "Contact:")
			for _, contactLine := range contactLines {
				lines = append(lines, "    "+contactLine)
			}
		}
	}

	return strings.Join(lines, "\n")
}

func (tpd *TitlePageDialog) hasExistingTitlePage(text string) bool {
	return titlePageEnd(strings.Split(text, "\n")) > 0
}

// removeExistingTitlePage returns text without its leading title page.
func (tpd *TitlePageDialog) removeExistingTitlePage(text string) string {
	lines := strings.Split(text, "\n")
	return strings.Join(lines[titlePageEnd(lines):], "\n")
}

// titlePageEnd returns the index of the first script line after a leading
// Fountain title page, or 0 if there is none. The title page is the run of
// "Key: value" fields, their indented continuation lines and the blank
// lines between them, read the way lexington's parser reads it.
func titlePageEnd(lines []string) int {
	end, sawField := 0, false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			continue
		case sawField && (strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "   ")):
		case isTitleField(line, lines, i):
			sawField = true
		default:
			if !sawField {
				return 0
			}
			return i
		}
		end = i + 1
	}
	if !sawField {
		return 0
	}
	for end < len(lines) && strings.TrimSpace(lines[end]) == "" {
		end++
	}
	return end
}

// isTitleField reports whether lines[i] is a "Key: value" field rather than
// script text such as a scene heading or a "FADE IN:" cue.
func isTitleField(line string, lines []string, i int) bool {
	if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
		return false
	}
	key, value, ok := strings.Cut(line, ":")
	if !ok || strings.TrimSpace(key) == "" || isSceneHeadingIn(line, nil) {
		return false
	}
	if strings.TrimSpace(value) != "" || key != strings.ToUpper(key) {
		return true
	}
	return i+1 < len(lines) && strings.TrimSpace(lines[i+1]) != "" &&
		(strings.HasPrefix(lines[i+1], "   ") || strings.HasPrefix(lines[i+1], "\t"))
}

func (tpd *TitlePageDialog) saveSettings() {
	if tpd.window.settings == nil {
		return
	}

	settings := tpd.window.settings
	settings.SetString("title-page-title", tpd.titleEntry.Text)
	settings.SetString("title-page-author", tpd.authorEntry.Text)
	settings.SetString("contact-name", tpd.nameEntry.Text)
	settings.SetString("contact-address", tpd.addressEntry.Text)
	settings.SetString("contact-phone", tpd.phoneEntry.Text)
	settings.SetString("contact-email", tpd.emailEntry.Text)
}

func (tpd *TitlePageDialog) Show() {
	tpd.dialog.Show()
}

// titleField is a field of a title page: its key, value and the lines it
// takes (with its indented continuation lines).
type titleField struct {
	key, value string
	more       []string // continuation lines, trimmed
	raw        []string
}

// dialogField are the fields the dialog edits.
var dialogField = map[string]bool{"title": true, "credit": true, "author": true, "authors": true,
	"source": true, "draft date": true, "contact": true}

// titleFields are the fields of a title page's lines.
func titleFields(lines []string) []titleField {
	var out []titleField
	for i, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		if (strings.HasPrefix(l, "   ") || strings.HasPrefix(l, "\t")) && len(out) > 0 {
			f := &out[len(out)-1]
			f.more = append(f.more, strings.TrimSpace(l))
			f.raw = append(f.raw, l)
			continue
		}
		if key, value, ok := strings.Cut(l, ":"); ok && isTitleField(l, lines, i) {
			out = append(out, titleField{key: strings.TrimSpace(key), value: strings.TrimSpace(value), raw: []string{l}})
		}
	}
	return out
}

// loadFromScript fills the dialog from the script's title page.
func (tpd *TitlePageDialog) loadFromScript() {
	lines := strings.Split(tpd.window.textEditor.Text(), "\n")
	fields := titleFields(lines[:titlePageEnd(lines)])
	if len(fields) == 0 {
		return
	}
	for _, e := range []*widget.Entry{tpd.titleEntry, tpd.authorEntry, tpd.basedOnEntry, tpd.nameEntry,
		tpd.addressEntry, tpd.phoneEntry, tpd.emailEntry} {
		e.SetText("")
	}
	tpd.creditEntry.SetText("")
	tpd.draftDateEntry.SetText("")
	tpd.includeDate.SetChecked(false)
	tpd.includeContact.SetChecked(false)
	for _, f := range fields {
		value := strings.TrimSpace(strings.Join(append([]string{f.value}, f.more...), " "))
		switch strings.ToLower(f.key) {
		case "title":
			tpd.titleEntry.SetText(value)
		case "credit":
			tpd.creditEntry.SetText(value)
		case "author", "authors":
			tpd.authorEntry.SetText(value)
		case "source":
			tpd.basedOnEntry.SetText(value)
		case "draft date":
			tpd.draftDateEntry.SetText(value)
			tpd.includeDate.SetChecked(true)
		case "contact":
			tpd.includeContact.SetChecked(true)
			var address []string
			for i, c := range append([]string{f.value}, f.more...) {
				switch {
				case c == "":
				case strings.Contains(c, "@"):
					tpd.emailEntry.SetText(c)
				case isPhone(c):
					tpd.phoneEntry.SetText(c)
				case i <= 1 && tpd.nameEntry.Text == "":
					tpd.nameEntry.SetText(c)
				default:
					address = append(address, c)
				}
			}
			tpd.addressEntry.SetText(strings.Join(address, "\n"))
		}
	}
}

// isPhone reports whether a contact line is a telephone number.
func isPhone(s string) bool {
	digits := 0
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case strings.ContainsRune(" +-().", r):
		default:
			return false
		}
	}
	return digits >= 6
}
