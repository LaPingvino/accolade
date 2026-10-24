//go:build !js

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// On the desktop, a script the PDF writer cannot print as written
// (Hebrew, CJK, right to left) is printed by a browser on the computer
// from Lexington's HTML: its fonts and layout have every script. Print
// opens the browser's print dialog; a PDF is made by a Chromium browser
// (Chrome, Chromium, Edge, Brave) without showing it.

func init() { printFile = printInBrowser }

// printInBrowser opens an HTML file in the system's browser, which opens
// its print dialog when the page has loaded.
func printInBrowser(path string) {
	if b, err := os.ReadFile(path); err == nil {
		page := strings.Replace(string(b), "</body>", `<script>addEventListener("load", () => print())</script></body>`, 1)
		_ = os.WriteFile(path, []byte(page), 0o644)
	}
	openFile(path)
}

// localBrowser is a Chromium browser on the computer that can make PDFs,
// "" if there is none.
func localBrowser() string {
	names := []string{"google-chrome-stable", "google-chrome", "chromium", "chromium-browser", "microsoft-edge", "microsoft-edge-stable", "brave-browser", "brave"}
	for _, n := range names {
		if p, err := exec.LookPath(n); err == nil {
			return p
		}
	}
	var paths []string
	switch runtime.GOOS {
	case "darwin":
		paths = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
		}
	case "windows":
		for _, dir := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("LocalAppData")} {
			if dir != "" {
				paths = append(paths, filepath.Join(dir, `Google\Chrome\Application\chrome.exe`),
					filepath.Join(dir, `Microsoft\Edge\Application\msedge.exe`),
					filepath.Join(dir, `BraveSoftware\Brave-Browser\Application\brave.exe`))
			}
		}
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// browserPDF prints an HTML file to a PDF with a Chromium browser.
func browserPDF(browser, htmlPath, pdfPath string) error {
	profile, err := os.MkdirTemp("", "accolade-browser-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(profile)
	abs, err := filepath.Abs(htmlPath)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, browser, "--headless=new", "--disable-gpu", "--no-pdf-header-footer",
		"--user-data-dir="+profile, "--print-to-pdf="+pdfPath, "file://"+filepath.ToSlash(abs))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s could not print the script: %v: %s", filepath.Base(browser), err, strings.TrimSpace(string(out)))
	}
	if st, err := os.Stat(pdfPath); err != nil || st.Size() == 0 {
		return fmt.Errorf("%s made no PDF", filepath.Base(browser))
	}
	return nil
}
