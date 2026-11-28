// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

//go:build js

package main

import "errors"

// pandocPath: the browser runs no programs, so no pandoc.
var pandocPath = func() string { return "" }

func runPandoc(...string) error { return errors.New("pandoc runs only in Accolade on your computer") }
