package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// LightTheme provides a light color scheme for Accolade
type LightTheme struct{}

func (t *LightTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.RGBA{R: 255, G: 255, B: 255, A: 255} // White
	case theme.ColorNameForeground:
		return color.RGBA{R: 0, G: 0, B: 0, A: 255} // Black
	case theme.ColorNamePrimary:
		return color.RGBA{R: 52, G: 96, B: 140, A: 255} // Muted ink blue
	case theme.ColorNameFocus:
		return color.RGBA{R: 52, G: 96, B: 140, A: 255} // Muted ink blue
	case theme.ColorNameHover:
		return color.RGBA{R: 240, G: 240, B: 240, A: 255} // Light gray
	case theme.ColorNamePressed:
		return color.RGBA{R: 220, G: 220, B: 220, A: 255} // Darker gray
	case theme.ColorNameSelection:
		return color.RGBA{R: 52, G: 96, B: 140, A: 60} // Semi-transparent ink blue
	case theme.ColorNameSeparator:
		return color.RGBA{R: 200, G: 200, B: 200, A: 255} // Light gray
	case theme.ColorNameError:
		return color.RGBA{R: 220, G: 53, B: 69, A: 255} // Red
	case theme.ColorNameWarning:
		return color.RGBA{R: 255, G: 193, B: 7, A: 255} // Yellow
	case theme.ColorNameSuccess:
		return color.RGBA{R: 40, G: 167, B: 69, A: 255} // Green
	case theme.ColorNameButton:
		return color.RGBA{R: 248, G: 249, B: 250, A: 255} // Very light gray
	case theme.ColorNameDisabled:
		return color.RGBA{R: 150, G: 150, B: 150, A: 255} // Gray
	case theme.ColorNamePlaceHolder:
		return color.RGBA{R: 120, G: 120, B: 120, A: 255} // Medium gray
	case theme.ColorNameScrollBar:
		return color.RGBA{R: 200, G: 200, B: 200, A: 255} // Light gray
	case theme.ColorNameShadow:
		return color.RGBA{R: 0, G: 0, B: 0, A: 50} // Semi-transparent black
	case theme.ColorNameInputBackground:
		return color.RGBA{R: 255, G: 255, B: 255, A: 255} // White
	case theme.ColorNameInputBorder:
		return color.RGBA{R: 200, G: 200, B: 200, A: 255} // Light gray
	case theme.ColorNameMenuBackground:
		return color.RGBA{R: 255, G: 255, B: 255, A: 255} // White
	case theme.ColorNameOverlayBackground:
		return color.RGBA{R: 0, G: 0, B: 0, A: 128} // Semi-transparent black
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (t *LightTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *LightTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *LightTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

// DarkTheme provides a dark color scheme for Accolade
type DarkTheme struct{}

func (t *DarkTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.RGBA{R: 35, G: 35, B: 35, A: 255} // Dark gray
	case theme.ColorNameForeground:
		return color.RGBA{R: 240, G: 240, B: 240, A: 255} // Light gray
	case theme.ColorNamePrimary:
		return color.RGBA{R: 140, G: 176, B: 214, A: 255} // Soft blue
	case theme.ColorNameFocus:
		return color.RGBA{R: 140, G: 176, B: 214, A: 255} // Soft blue
	case theme.ColorNameHover:
		return color.RGBA{R: 60, G: 60, B: 60, A: 255} // Medium dark gray
	case theme.ColorNamePressed:
		return color.RGBA{R: 80, G: 80, B: 80, A: 255} // Lighter dark gray
	case theme.ColorNameSelection:
		return color.RGBA{R: 140, G: 176, B: 214, A: 70} // Semi-transparent soft blue
	case theme.ColorNameSeparator:
		return color.RGBA{R: 80, G: 80, B: 80, A: 255} // Dark gray
	case theme.ColorNameError:
		return color.RGBA{R: 255, G: 100, B: 120, A: 255} // Light red
	case theme.ColorNameWarning:
		return color.RGBA{R: 255, G: 220, B: 100, A: 255} // Light yellow
	case theme.ColorNameSuccess:
		return color.RGBA{R: 100, G: 220, B: 130, A: 255} // Light green
	case theme.ColorNameButton:
		return color.RGBA{R: 50, G: 50, B: 50, A: 255} // Dark gray
	case theme.ColorNameDisabled:
		return color.RGBA{R: 100, G: 100, B: 100, A: 255} // Medium gray
	case theme.ColorNamePlaceHolder:
		return color.RGBA{R: 140, G: 140, B: 140, A: 255} // Light gray
	case theme.ColorNameScrollBar:
		return color.RGBA{R: 80, G: 80, B: 80, A: 255} // Dark gray
	case theme.ColorNameShadow:
		return color.RGBA{R: 0, G: 0, B: 0, A: 80} // Semi-transparent black
	case theme.ColorNameInputBackground:
		return color.RGBA{R: 45, G: 45, B: 45, A: 255} // Slightly lighter dark gray
	case theme.ColorNameInputBorder:
		return color.RGBA{R: 80, G: 80, B: 80, A: 255} // Dark gray
	case theme.ColorNameMenuBackground:
		return color.RGBA{R: 40, G: 40, B: 40, A: 255} // Dark gray
	case theme.ColorNameOverlayBackground:
		return color.RGBA{R: 0, G: 0, B: 0, A: 180} // Semi-transparent black
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (t *DarkTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *DarkTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *DarkTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

// SepiaTheme provides a sepia/warm color scheme for distraction-free writing
type SepiaTheme struct{}

func (t *SepiaTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.RGBA{R: 252, G: 248, B: 227, A: 255} // Warm off-white
	case theme.ColorNameForeground:
		return color.RGBA{R: 92, G: 80, B: 65, A: 255} // Dark brown
	case theme.ColorNamePrimary:
		return color.RGBA{R: 140, G: 90, B: 40, A: 255} // Warm brown
	case theme.ColorNameFocus:
		return color.RGBA{R: 140, G: 90, B: 40, A: 255} // Warm brown
	case theme.ColorNameHover:
		return color.RGBA{R: 245, G: 240, B: 215, A: 255} // Slightly darker sepia
	case theme.ColorNamePressed:
		return color.RGBA{R: 235, G: 225, B: 195, A: 255} // Darker sepia
	case theme.ColorNameSelection:
		return color.RGBA{R: 184, G: 134, B: 11, A: 80} // Semi-transparent golden brown
	case theme.ColorNameSeparator:
		return color.RGBA{R: 220, G: 200, B: 160, A: 255} // Light brown
	case theme.ColorNameError:
		return color.RGBA{R: 180, G: 65, B: 47, A: 255} // Warm red
	case theme.ColorNameWarning:
		return color.RGBA{R: 218, G: 165, B: 32, A: 255} // Golden rod
	case theme.ColorNameSuccess:
		return color.RGBA{R: 107, G: 142, B: 35, A: 255} // Olive green
	case theme.ColorNameButton:
		return color.RGBA{R: 248, G: 242, B: 220, A: 255} // Light sepia
	case theme.ColorNameDisabled:
		return color.RGBA{R: 150, G: 140, B: 120, A: 255} // Muted brown
	case theme.ColorNamePlaceHolder:
		return color.RGBA{R: 140, G: 125, B: 100, A: 255} // Medium brown
	case theme.ColorNameScrollBar:
		return color.RGBA{R: 220, G: 200, B: 160, A: 255} // Light brown
	case theme.ColorNameShadow:
		return color.RGBA{R: 92, G: 80, B: 65, A: 50} // Semi-transparent dark brown
	case theme.ColorNameInputBackground:
		return color.RGBA{R: 255, G: 252, B: 235, A: 255} // Very light sepia
	case theme.ColorNameInputBorder:
		return color.RGBA{R: 220, G: 200, B: 160, A: 255} // Light brown
	case theme.ColorNameMenuBackground:
		return color.RGBA{R: 250, G: 245, B: 225, A: 255} // Light sepia
	case theme.ColorNameOverlayBackground:
		return color.RGBA{R: 92, G: 80, B: 65, A: 128} // Semi-transparent dark brown
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (t *SepiaTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *SepiaTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *SepiaTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

// WriterTheme is an alias for SepiaTheme - optimized for long writing sessions
type WriterTheme = SepiaTheme

// Helper function to get the appropriate theme based on string name
func GetThemeByName(name string) fyne.Theme {
	switch name {
	case "light":
		return &LightTheme{}
	case "dark":
		return &DarkTheme{}
	case "sepia", "writer":
		return &SepiaTheme{}
	default:
		return theme.DefaultTheme()
	}
}

// Helper function to get theme names
func GetAvailableThemes() []string {
	return []string{"light", "dark", "sepia", "system"}
}