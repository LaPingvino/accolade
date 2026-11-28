// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2/dialog"
)

// The storyboard as an OpenTimelineIO timeline (docs/research/
// storyboard-formats.md: the one open, documented format for frames,
// timing and text; OTIO's adapters reach EDL, FCP XML, AAF and Resolve
// from it). Plain JSON, written here (OTIO's file format, schemas
// Timeline.1, Stack.1, Track.1, Clip.2, ExternalReference.1, Marker.2):
// one video track, a clip per frame in script order, its picture
// referenced, its time the note's, else its file's (an animation's),
// else otioDefaultSecs; a marker where each scene starts; the scene, its
// heading and the lines the frame illustrates in metadata["accolade"].

// otioRate is the timeline's frames per second.
const otioRate = 24.0

// otioDefaultSecs is an untimed frame's time (Storyboarder's and Boords'
// default).
const otioDefaultSecs = 2.0

type otioObj = map[string]any

func otioTime(frames float64) otioObj {
	return otioObj{"OTIO_SCHEMA": "RationalTime.1", "rate": otioRate, "value": frames}
}

func otioRange(start, dur float64) otioObj {
	return otioObj{"OTIO_SCHEMA": "TimeRange.1", "start_time": otioTime(start), "duration": otioTime(dur)}
}

// boardTimeline is the script's frames (s, the table read's reading) as
// an .otio file named name; ref is the address a frame's picture is
// referenced by.
func boardTimeline(s *trScript, name string, ref func(src string) string) ([]byte, int) {
	var clips, markers []any
	at, count := 0.0, 0
	for _, sc := range s.Scenes {
		var said []string
		marked := false
		for _, l := range sc.Lines {
			if l.Kind != trFrame {
				if t := strings.TrimSpace(l.Text); t != "" && l.Kind != trHeading {
					if l.Kind == trCue {
						t += ":"
					}
					said = append(said, t)
				}
				continue
			}
			if !marked && sc.Heading != "" {
				markers = append(markers, otioObj{"OTIO_SCHEMA": "Marker.2", "name": strings.TrimSpace(sc.Number + " " + sc.Heading),
					"marked_range": otioRange(at, 0), "color": "GREEN", "comment": "",
					"metadata": otioObj{"accolade": otioObj{"scene": sc.Number, "heading": sc.Heading, "line": sc.Line + 1}}})
				marked = true
			}
			secs := l.Secs
			if secs <= 0 {
				secs = otioDefaultSecs
			}
			dur := math.Round(secs * otioRate)
			meta := otioObj{"scene": sc.Number, "heading": sc.Heading, "line": l.Line + 1, "file": l.Frame, "timed": l.Secs > 0}
			if len(said) > 0 {
				meta["text"] = strings.Join(said, "\n")
			}
			said = nil
			clips = append(clips, otioObj{
				"OTIO_SCHEMA": "Clip.2", "name": path.Base(l.Frame), "enabled": true,
				"source_range": otioRange(0, dur),
				"media_references": otioObj{"DEFAULT_MEDIA": otioObj{"OTIO_SCHEMA": "ExternalReference.1",
					"name": path.Base(l.Frame), "target_url": ref(l.Frame), "available_range": nil, "available_image_bounds": nil, "metadata": otioObj{}}},
				"active_media_reference_key": "DEFAULT_MEDIA",
				"effects":                    []any{}, "markers": []any{},
				"metadata": otioObj{"accolade": meta},
			})
			at += dur
			count++
		}
	}
	if clips == nil {
		clips = []any{}
	}
	if markers == nil {
		markers = []any{}
	}
	track := otioObj{"OTIO_SCHEMA": "Track.1", "name": "Storyboard", "kind": "Video", "children": clips,
		"markers": markers, "effects": []any{}, "metadata": otioObj{}, "source_range": nil, "enabled": true}
	tl := otioObj{"OTIO_SCHEMA": "Timeline.1", "name": name, "global_start_time": nil,
		"metadata": otioObj{"accolade": otioObj{"rate": otioRate, "default_frame_secs": otioDefaultSecs}},
		"tracks": otioObj{"OTIO_SCHEMA": "Stack.1", "name": "tracks", "children": []any{track},
			"markers": []any{}, "effects": []any{}, "metadata": otioObj{}, "source_range": nil, "enabled": true}}
	data, _ := json.MarshalIndent(tl, "", "    ")
	return append(data, '\n'), count
}

// frameURL is the address an exported timeline names a frame's picture
// by: its file's on the desktop (file:///…), else as the script names it.
func (w *MainWindow) frameURL(src string) string {
	if abs, entry, err := w.frameOnDisk(src); err == nil && entry == "" && fileExists(abs) {
		p := filepath.ToSlash(abs)
		if !strings.HasPrefix(p, "/") {
			p = "/" + p // C:/… → file:///C:/…
		}
		return (&url.URL{Scheme: "file", Path: p}).String()
	}
	return src
}

// exportBoardTimeline is File › Export Storyboard Timeline (OTIO)….
func (w *MainWindow) exportBoardTimeline() {
	text := w.textEditor.Text()
	s := readTableWith(text, w.sceneStarts(), w.dialect(), w.storyboarder(), w.frameFileSecs)
	title := cmpOr(strings.TrimSuffix(filepath.Base(w.currentFile), filepath.Ext(w.currentFile)), "Storyboard")
	data, n := boardTimeline(s, title, w.frameURL)
	if n == 0 {
		dialog.ShowInformation("No Storyboard Frames", "The script has no frames to put on a timeline: add one with Insert › Storyboard Frame… or New Frame.", w.fyneWindow)
		return
	}
	w.saveNamedFile(title+" - storyboard.otio", func(p string) error {
		if err := writeFileAtomic(p, data); err != nil {
			return err
		}
		w.setStatus(fmt.Sprintf("Storyboard timeline saved as %s: %s", filepath.Base(p), plural(n, "frame")))
		return nil
	})
}
