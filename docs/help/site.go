package help

import (
	"os"
	"path/filepath"
)

// WriteSite writes every page and cheat sheet as a web page into dir
// (<slug>.html; index.html the contents), for the given version of
// Accolade: what the browser version publishes at /help/.
func WriteSite(dir, version string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, p := range All() {
		page, err := HTML(p, version, false)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, p.Slug+".html"), page, 0o644); err != nil {
			return err
		}
	}
	return nil
}
