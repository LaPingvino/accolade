# Accolade in the hourly loop

Joop, 2026-11-19: Accolade is in the hourly tick, next to goloco, after Rebekah's WEquilibrium notes.

## What the tick works on

- The adaptation plan, `docs/ADAPTATION.md`, in its Order. Each step is one slice, or one background agent in a worktree.
- Only one Accolade agent runs at a time.

## Rules

- **Never cd:** use `git -C` and `go -C`.
- **Tests:** `go test ./...` and `go test -race .` must pass. Commit only inside an `if` on success.
- **Pushing to main** redeploys the public browser version (pages.yml), so main must always work in the browser too.
  Anything that needs the desktop (pandoc, `exec`) has a browser fallback, or is left out there with a clear message.
- **Lexington:** changes it needs go into `~/Projects/lexington` first and are tagged there. Accolade then updates
  its go.mod, as for the earlier export work.
- **Screenshots:** anything visible is checked with a screenshot, as `screenshot_test.go` makes them.

## Tick log

- 2026-11-19: added to the loop. Plan in docs/ADAPTATION.md. Next: step 1, the Markdown mode.
- 2026-11-19: step 1, Markdown mode (branch markdown, not merged). A .md/.markdown file opens with Markdown colouring
  (syntax/markdown.go: line kinds and inline emphasis), an outline of its headings, a live preview beside the editor
  (Fyne's RichText, goldmark inside; Ctrl+Shift+P), notes in `<!-- Notes -->`, list continuation on Enter, Print as a
  goldmark web page. Elements, completion, title page, export, cards and colours are put away in Markdown. File > New
  Markdown Document; Open offers .md (browser too). Next: step 2, import and export.
