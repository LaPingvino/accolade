package main

import (
	"log"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type HeaderBar struct {
	*adw.HeaderBar
	
	// Core components
	window       *MainWindow
	
	// Title and subtitle
	titleLabel   *gtk.Label
	subtitleLabel *gtk.Label
	
	// Left side buttons
	newButton    *gtk.Button
	openButton   *gtk.Button
	saveButton   *gtk.Button
	
	// Right side buttons
	previewButton *gtk.Button
	exportButton  *gtk.Button
	menuButton    *gtk.Button
	
	// Menu popover
	menuPopover  *gtk.PopoverMenu
	
	// State
	isFullscreen bool
	title        string
	subtitle     string
}

func NewHeaderBar(window *MainWindow) *HeaderBar {
	hb := &HeaderBar{
		HeaderBar:    adw.NewHeaderBar(),
		window:       window,
		isFullscreen: false,
		title:        "Accolade",
		subtitle:     "",
	}
	
	hb.setupUI()
	hb.setupActions()
	hb.setupSignals()
	
	return hb
}

func (hb *HeaderBar) setupUI() {
	// Set up basic properties
	hb.SetShowEndTitleButtons(true)
	hb.SetShowStartTitleButtons(true)
	hb.AddCSSClass("accolade-headerbar")
	
	// Create title box
	titleBox := gtk.NewBox(gtk.OrientationVertical, 0)
	titleBox.SetVAlign(gtk.AlignCenter)
	
	hb.titleLabel = gtk.NewLabel(hb.title)
	hb.titleLabel.AddCSSClass("title")
	titleBox.Append(hb.titleLabel)
	
	hb.subtitleLabel = gtk.NewLabel(hb.subtitle)
	hb.subtitleLabel.AddCSSClass("subtitle")
	hb.subtitleLabel.SetVisible(false)
	titleBox.Append(hb.subtitleLabel)
	
	hb.SetTitleWidget(titleBox)
	
	// Create left side buttons
	hb.setupLeftSide()
	
	// Create right side buttons
	hb.setupRightSide()
	
	// Create menu
	hb.setupMenu()
}

func (hb *HeaderBar) setupLeftSide() {
	leftBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	
	// New button
	hb.newButton = gtk.NewButtonFromIconName("document-new-symbolic")
	hb.newButton.SetTooltipText("New document")
	hb.newButton.SetActionName("app.new_window")
	leftBox.Append(hb.newButton)
	
	// Open button
	hb.openButton = gtk.NewButtonFromIconName("document-open-symbolic")
	hb.openButton.SetTooltipText("Open document")
	hb.openButton.SetActionName("win.open")
	leftBox.Append(hb.openButton)
	
	// Save button
	hb.saveButton = gtk.NewButtonFromIconName("document-save-symbolic")
	hb.saveButton.SetTooltipText("Save document")
	hb.saveButton.SetActionName("win.save")
	leftBox.Append(hb.saveButton)
	
	hb.PackStart(leftBox)
}

func (hb *HeaderBar) setupRightSide() {
	rightBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	
	// Preview button
	hb.previewButton = gtk.NewButtonFromIconName("view-paged-symbolic")
	hb.previewButton.SetTooltipText("Toggle preview")
	hb.previewButton.SetActionName("win.preview")
	rightBox.Append(hb.previewButton)
	
	// Export button
	hb.exportButton = gtk.NewButtonFromIconName("document-send-symbolic")
	hb.exportButton.SetTooltipText("Export document")
	hb.exportButton.SetActionName("win.export")
	rightBox.Append(hb.exportButton)
	
	// Menu button
	hb.menuButton = gtk.NewButtonFromIconName("open-menu-symbolic")
	hb.menuButton.SetTooltipText("Main menu")
	rightBox.Append(hb.menuButton)
	
	hb.PackEnd(rightBox)
}

func (hb *HeaderBar) setupMenu() {
	// Create menu model
	menu := gio.NewMenu()
	
	// File section
	fileSection := gio.NewMenu()
	fileSection.AppendItem(gio.NewMenuItem("New Window", "app.new_window"))
	fileSection.AppendItem(gio.NewMenuItem("Open", "win.open"))
	fileSection.AppendItem(gio.NewMenuItem("Save", "win.save"))
	fileSection.AppendItem(gio.NewMenuItem("Save As", "win.save_as"))
	fileSection.AppendItem(gio.NewMenuItem("Export", "win.export"))
	menu.AppendSection("", fileSection)
	
	// Edit section
	editSection := gio.NewMenu()
	editSection.AppendItem(gio.NewMenuItem("Find", "win.find"))
	editSection.AppendItem(gio.NewMenuItem("Find & Replace", "win.find_replace"))
	menu.AppendSection("", editSection)
	
	// View section
	viewSection := gio.NewMenu()
	viewSection.AppendItem(gio.NewMenuItem("Preview", "win.preview"))
	viewSection.AppendItem(gio.NewMenuItem("Focus Mode", "win.focus_mode"))
	viewSection.AppendItem(gio.NewMenuItem("Fullscreen", "win.fullscreen"))
	menu.AppendSection("", viewSection)
	
	// App section
	appSection := gio.NewMenu()
	appSection.AppendItem(gio.NewMenuItem("Preferences", "app.preferences"))
	appSection.AppendItem(gio.NewMenuItem("About", "app.about"))
	menu.AppendSection("", appSection)
	
	// Create popover
	hb.menuPopover = gtk.NewPopoverMenuFromModel(menu)
	hb.menuPopover.SetParent(hb.menuButton)
	
	// Connect menu button
	hb.menuButton.ConnectClicked(func() {
		hb.menuPopover.Popup()
	})
}

func (hb *HeaderBar) setupActions() {
	// Actions are handled by the main window
	log.Println("HeaderBar actions set up")
}

func (hb *HeaderBar) setupSignals() {
	// Connect button signals if needed
	log.Println("HeaderBar signals set up")
}

func (hb *HeaderBar) SetTitle(title string) {
	hb.title = title
	hb.titleLabel.SetText(title)
}

func (hb *HeaderBar) GetTitle() string {
	return hb.title
}

func (hb *HeaderBar) SetSubtitle(subtitle string) {
	hb.subtitle = subtitle
	hb.subtitleLabel.SetText(subtitle)
	hb.subtitleLabel.SetVisible(subtitle != "")
}

func (hb *HeaderBar) GetSubtitle() string {
	return hb.subtitle
}

func (hb *HeaderBar) SetFullscreen(fullscreen bool) {
	hb.isFullscreen = fullscreen
	
	if fullscreen {
		hb.SetShowEndTitleButtons(false)
		hb.SetShowStartTitleButtons(false)
	} else {
		hb.SetShowEndTitleButtons(true)
		hb.SetShowStartTitleButtons(true)
	}
}

func (hb *HeaderBar) IsFullscreen() bool {
	return hb.isFullscreen
}

func (hb *HeaderBar) SetSaveButtonSensitive(sensitive bool) {
	hb.saveButton.SetSensitive(sensitive)
}

func (hb *HeaderBar) SetPreviewButtonActive(active bool) {
	if active {
		hb.previewButton.AddCSSClass("suggested-action")
	} else {
		hb.previewButton.RemoveCSSClass("suggested-action")
	}
}

func (hb *HeaderBar) UpdateForFile(filename string, hasChanges bool) {
	if filename != "" {
		if hasChanges {
			hb.SetTitle("• " + filename)
		} else {
			hb.SetTitle(filename)
		}
	} else {
		if hasChanges {
			hb.SetTitle("• Untitled")
		} else {
			hb.SetTitle("Accolade")
		}
	}
}

func (hb *HeaderBar) ShowProgress(show bool) {
	// TODO: Add progress indicator to headerbar
	log.Printf("HeaderBar progress: %v", show)
}

func (hb *HeaderBar) SetProgress(progress float64) {
	// TODO: Update progress indicator
	log.Printf("HeaderBar progress: %.2f", progress)
}

func (hb *HeaderBar) AddWidget(widget gtk.Widgetter, position HeaderBarPosition) {
	switch position {
	case HeaderBarPositionStart:
		hb.PackStart(widget)
	case HeaderBarPositionEnd:
		hb.PackEnd(widget)
	}
}

func (hb *HeaderBar) RemoveWidget(widget gtk.Widgetter) {
	hb.Remove(widget)
}

func (hb *HeaderBar) SetTitleWidgetVisible(visible bool) {
	titleWidget := hb.TitleWidget()
	if titleWidget != nil {
		titleWidget.SetVisible(visible)
	}
}

func (hb *HeaderBar) GetMenuButton() *gtk.Button {
	return hb.menuButton
}

func (hb *HeaderBar) GetMenuPopover() *gtk.PopoverMenu {
	return hb.menuPopover
}

type HeaderBarPosition int

const (
	HeaderBarPositionStart HeaderBarPosition = iota
	HeaderBarPositionEnd
)