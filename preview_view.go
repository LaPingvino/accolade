package main

import (
	"log"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4-webkitgtk/pkg/webkit2/v4"
)

type PreviewView struct {
	*gtk.ScrolledWindow
	
	// Core components
	webView      *webkit2.WebView
	window       *MainWindow
	
	// Content management
	currentContent string
	restrictedMode bool
	
	// Settings
	settings     *Settings
	
	// Lexington integration
	lexington    *LexingtonConverter
}

func NewPreviewView(window *MainWindow) *PreviewView {
	pv := &PreviewView{
		ScrolledWindow: gtk.NewScrolledWindow(),
		window:         window,
		restrictedMode: false,
	}
	
	pv.setupUI()
	pv.setupWebView()
	pv.setupLexington()
	
	return pv
}

func (pv *PreviewView) setupUI() {
	pv.SetHExpand(true)
	pv.SetVExpand(true)
	pv.SetPolicy(gtk.PolicyAutomatic, gtk.PolicyAutomatic)
	pv.AddCSSClass("preview-view")
}

func (pv *PreviewView) setupWebView() {
	pv.webView = webkit2.NewWebView()
	pv.webView.SetHExpand(true)
	pv.webView.SetVExpand(true)
	
	// Configure web view settings
	webSettings := pv.webView.Settings()
	webSettings.SetEnableJavaScript(!pv.restrictedMode)
	webSettings.SetEnableHTML5Database(false)
	webSettings.SetEnableHTML5LocalStorage(false)
	webSettings.SetEnablePlugins(false)
	
	pv.SetChild(pv.webView)
	
	// Connect signals
	pv.webView.ConnectLoadChanged(func(loadEvent webkit2.LoadEvent) {
		pv.onLoadChanged(loadEvent)
	})
}

func (pv *PreviewView) setupLexington() {
	pv.lexington = NewLexingtonConverter()
}

func (pv *PreviewView) onLoadChanged(loadEvent webkit2.LoadEvent) {
	switch loadEvent {
	case webkit2.LoadEventStarted:
		log.Println("Preview: Load started")
	case webkit2.LoadEventCommitted:
		log.Println("Preview: Load committed")
	case webkit2.LoadEventFinished:
		log.Println("Preview: Load finished")
	}
}

func (pv *PreviewView) UpdateContent(fountainText string) {
	if pv.currentContent == fountainText {
		return
	}
	
	pv.currentContent = fountainText
	
	// Convert Fountain to HTML using Lexington
	htmlContent, err := pv.lexington.ConvertToHTML(fountainText, pv.restrictedMode)
	if err != nil {
		log.Printf("Error converting Fountain to HTML: %v", err)
		pv.showError("Error converting Fountain to HTML: " + err.Error())
		return
	}
	
	// Load HTML content
	pv.webView.LoadHTML(htmlContent, "file:///")
}

func (pv *PreviewView) showError(message string) {
	errorHTML := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Preview Error</title>
		<style>
			body { font-family: sans-serif; padding: 20px; }
			.error { color: red; border: 1px solid red; padding: 10px; border-radius: 5px; }
		</style>
	</head>
	<body>
		<div class="error">
			<h3>Preview Error</h3>
			<p>` + message + `</p>
		</div>
	</body>
	</html>`
	
	pv.webView.LoadHTML(errorHTML, "file:///")
}

func (pv *PreviewView) SetRestrictedMode(restricted bool) {
	pv.restrictedMode = restricted
	
	// Update web view settings
	webSettings := pv.webView.Settings()
	webSettings.SetEnableJavaScript(!restricted)
	
	// Refresh content if we have any
	if pv.currentContent != "" {
		pv.UpdateContent(pv.currentContent)
	}
}

func (pv *PreviewView) GetZoomLevel() float64 {
	return pv.webView.ZoomLevel()
}

func (pv *PreviewView) SetZoomLevel(level float64) {
	pv.webView.SetZoomLevel(level)
}

func (pv *PreviewView) ZoomIn() {
	currentZoom := pv.GetZoomLevel()
	pv.SetZoomLevel(currentZoom * 1.1)
}

func (pv *PreviewView) ZoomOut() {
	currentZoom := pv.GetZoomLevel()
	pv.SetZoomLevel(currentZoom / 1.1)
}

func (pv *PreviewView) ResetZoom() {
	pv.SetZoomLevel(1.0)
}

func (pv *PreviewView) Print() {
	printOperation := gtk.NewPrintOperation()
	printOperation.ConnectBeginPrint(func(context *gtk.PrintContext) {
		// TODO: Set up print operation
		log.Println("Print operation begun")
	})
	
	printOperation.ConnectDrawPage(func(context *gtk.PrintContext, pageNum int) {
		// TODO: Draw page content
		log.Printf("Drawing page %d", pageNum)
	})
	
	result := printOperation.Run(gtk.PrintOperationActionPrintDialog, &pv.window.Window)
	if result == gtk.PrintOperationResultError {
		log.Println("Print operation failed")
	}
}

func (pv *PreviewView) ExportToPDF(filename string) error {
	// TODO: Use Lexington to export to PDF
	return pv.lexington.ConvertToPDF(pv.currentContent, filename)
}

func (pv *PreviewView) ScrollToTop() {
	vadj := pv.VAdjustment()
	vadj.SetValue(0)
}

func (pv *PreviewView) ScrollToBottom() {
	vadj := pv.VAdjustment()
	vadj.SetValue(vadj.Upper() - vadj.PageSize())
}

func (pv *PreviewView) GetCurrentURL() string {
	return pv.webView.URI()
}

func (pv *PreviewView) Reload() {
	if pv.currentContent != "" {
		pv.UpdateContent(pv.currentContent)
	}
}

func (pv *PreviewView) GoBack() {
	if pv.webView.CanGoBack() {
		pv.webView.GoBack()
	}
}

func (pv *PreviewView) GoForward() {
	if pv.webView.CanGoForward() {
		pv.webView.GoForward()
	}
}

func (pv *PreviewView) FindText(text string, caseSensitive bool) {
	// TODO: Implement text search in preview
	log.Printf("Finding text in preview: %s", text)
}

func (pv *PreviewView) ClearFind() {
	// TODO: Clear text search highlighting
	log.Println("Clearing find in preview")
}

func (pv *PreviewView) GetSelectedText() string {
	// TODO: Get selected text from web view
	return ""
}

func (pv *PreviewView) SelectAll() {
	// TODO: Select all text in web view
	log.Println("Selecting all text in preview")
}

func (pv *PreviewView) Copy() {
	// TODO: Copy selected text to clipboard
	log.Println("Copying selected text from preview")
}

func (pv *PreviewView) applySettings(settings *Settings) {
	pv.settings = settings
	
	// Apply preview settings
	if settings.GetBoolean("sync-scroll") {
		pv.enableSyncScroll()
	} else {
		pv.disableSyncScroll()
	}
	
	// Apply theme settings
	pv.applyTheme()
}

func (pv *PreviewView) enableSyncScroll() {
	// TODO: Enable scroll synchronization with editor
	log.Println("Preview: Sync scroll enabled")
}

func (pv *PreviewView) disableSyncScroll() {
	// TODO: Disable scroll synchronization with editor
	log.Println("Preview: Sync scroll disabled")
}

func (pv *PreviewView) applyTheme() {
	// TODO: Apply theme to preview content
	log.Println("Preview: Applying theme")
}

func (pv *PreviewView) SyncScrollToEditor(editorPosition float64) {
	// TODO: Sync preview scroll position with editor
	log.Printf("Preview: Syncing scroll to editor position: %f", editorPosition)
}

func (pv *PreviewView) GetScrollPosition() float64 {
	vadj := pv.VAdjustment()
	return vadj.Value() / (vadj.Upper() - vadj.PageSize())
}

func (pv *PreviewView) SetScrollPosition(position float64) {
	vadj := pv.VAdjustment()
	value := position * (vadj.Upper() - vadj.PageSize())
	vadj.SetValue(value)
}

func (pv *PreviewView) IsRestrictedMode() bool {
	return pv.restrictedMode
}

func (pv *PreviewView) GetContent() string {
	return pv.currentContent
}

func (pv *PreviewView) IsLoading() bool {
	return pv.webView.IsLoading()
}

func (pv *PreviewView) Stop() {
	pv.webView.StopLoading()
}

func (pv *PreviewView) GetWebView() *webkit2.WebView {
	return pv.webView
}