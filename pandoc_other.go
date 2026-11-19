//go:build !js

package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// pandocPath is the pandoc program on the computer, "" if there is none:
// with it, more formats can be imported and exported (pandoc.go). A
// variable, so tests can do without it.
var pandocPath = func() string {
	p, err := exec.LookPath("pandoc")
	if err != nil {
		return ""
	}
	return p
}

// runPandoc runs pandoc with args (paths as arguments, never through a
// shell), for at most two minutes.
func runPandoc(args ...string) error {
	p := pandocPath()
	if p == "" {
		return fmt.Errorf("pandoc is not installed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, p, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("pandoc: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
