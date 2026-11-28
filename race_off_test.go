// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

//go:build !race

package main

// raceDetector: the tests run under -race (some pixel-capture checks are skipped there).
const raceDetector = false
