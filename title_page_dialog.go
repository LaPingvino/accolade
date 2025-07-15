package main

import (
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type TitlePageDialog struct {
	window   *MainWindow
	dialog   dialog.Dialog
	
	// Title page fields
	titleEntry       *widget.Entry
	creditEntry      *widget.Entry
	authorEntry      *widget.Entry
	basedOnEntry     *widget.Entry
	draftDateEntry   *widget.Entry
	contactEntry     *widget.Entry
	
	// Contact details
	nameEntry        *widget.Entry
	addressEntry     *widget.Entry
	phoneEntry       *widget.Entry
	emailEntry       *widget.Entry
	
	// Options
	includeTitlePage *widget.Check
	includeContact   *widget.Check
	includeDate      *widget.Check
	
	// Buttons
	insertButton     *widget.Button
	previewButton    *widget.Button
	cancelButton     *widget.Button
}

func NewTitlePageDialog(window *MainWindow) *TitlePageDialog {
	tpd := &TitlePageDialog{
		window: window,
	}
	
	tpd.createWidgets()
	tpd.loadDefaults()
	content := tpd.createContent()
	
	tpd.dialog = dialog.NewCustom(
		"Title Page Configuration",
		"",
		content,
		window.fyneWindow,
	)
	
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
	
	// Button container
	buttonContainer := container.NewHBox(
		tpd.previewButton,
		widget.NewLabel(""), // Spacer
		tpd.cancelButton,
		tpd.insertButton,
	)
	
	// Main content
	content := container.NewVBox(
		titleSection,
		contactSection,
		optionsSection,
		widget.NewSeparator(),
		buttonContainer,
	)
	
	// Wrap in scroll container
	scrollContent := container.NewScroll(content)
	scrollContent.SetMinSize(fyne.NewSize(480, 580))
	
	return scrollContent
}

func (tpd *TitlePageDialog) insertTitlePage() {
	if !tpd.includeTitlePage.Checked {
		tpd.dialog.Hide()
		return
	}
	
	titlePageText := tpd.generateTitlePageText()
	
	// Insert at the beginning of the document
	currentText := tpd.window.textEditor.Text
	
	// If document already starts with a title page, replace it
	if tpd.hasExistingTitlePage(currentText) {
		currentText = tpd.removeExistingTitlePage(currentText)
	}
	
	// Combine title page with content
	newText := titlePageText + "\n\n" + currentText
	tpd.window.textEditor.SetText(newText)
	
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
	
	// Add two blank lines after title page to separate from script content
	lines = append(lines, "")
	lines = append(lines, "")
	
	return strings.Join(lines, "\n")
}

func (tpd *TitlePageDialog) hasExistingTitlePage(text string) bool {
	// Check for title page markers: key-value pairs like Title:, Author:, etc.
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if i > 20 { // Only check first part of document
			break
		}
		trimmed := strings.TrimSpace(line)
		// Look for title page key-value pairs
		if strings.HasPrefix(trimmed, "Title:") ||
		   strings.HasPrefix(trimmed, "Author:") ||
		   strings.HasPrefix(trimmed, "Credit:") ||
		   strings.HasPrefix(trimmed, "Contact:") ||
		   strings.HasPrefix(trimmed, "Draft date:") {
			return true
		}
	}
	return false
}

func (tpd *TitlePageDialog) removeExistingTitlePage(text string) string {
	lines := strings.Split(text, "\n")
	
	// Find where the title page ends (look for script start)
	inTitlePage := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		trimmedUpper := strings.ToUpper(trimmed)
		
		// Check if we're in title page content
		if strings.Contains(trimmed, ":") && (strings.HasPrefix(trimmed, "Title:") ||
		   strings.HasPrefix(trimmed, "Author:") ||
		   strings.HasPrefix(trimmed, "Credit:") ||
		   strings.HasPrefix(trimmed, "Contact:") ||
		   strings.HasPrefix(trimmed, "Draft date:") ||
		   strings.HasPrefix(trimmed, "Source:")) {
			inTitlePage = true
			continue
		}
		
		// Skip indented contact info lines
		if inTitlePage && strings.HasPrefix(line, "    ") {
			continue
		}
		
		// Look for script start
		if trimmedUpper == "FADE IN:" || 
		   strings.HasPrefix(trimmedUpper, "INT.") || 
		   strings.HasPrefix(trimmedUpper, "EXT.") ||
		   strings.HasPrefix(trimmedUpper, "EST.") {
			// Found start of script, return everything from here
			return strings.Join(lines[i:], "\n")
		}
		
		// If we hit non-empty, non-title-page content, start from here
		if inTitlePage && trimmed != "" && !strings.Contains(trimmed, ":") {
			return strings.Join(lines[i:], "\n")
		}
	}
	
	// If no script start found, return empty (was all title page)
	return ""
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