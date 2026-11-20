// Package pace keeps long work from freezing the web version.
//
// In the browser Go runs on the page's one thread, and the page gets it
// back only when every goroutine is blocked: work in a goroutine of its
// own still freezes the page, and runtime.Gosched does not help. So a
// long loop calls Yield as it goes; every ~20 ms of work it blocks for a
// millisecond, which lets the browser draw and take input. Elsewhere
// (goroutines run on threads of their own) Yield does nothing.
package pace

import (
	"runtime"
	"sync/atomic"
	"time"
)

// Work is how long a stretch of work runs before Yield blocks.
const Work = 20 * time.Millisecond

// On is whether Yield blocks: in the browser (tests may turn it on).
var On atomic.Bool

func init() { On.Store(runtime.GOOS == "js") }

var (
	last  atomic.Int64  // when Yield last blocked (UnixNano)
	calls atomic.Uint32 // the clock is looked at every few calls
)

// Yield blocks for a millisecond if ~20 ms of work went by since it last
// did; cheap to call often (a loop's every step).
func Yield() {
	if !On.Load() || calls.Add(1)%16 != 0 {
		return
	}
	YieldNow()
}

// YieldNow is Yield for a loop whose steps are long (a PDF's pages): the
// clock is looked at every call.
func YieldNow() {
	if !On.Load() {
		return
	}
	now := time.Now().UnixNano()
	if l := last.Load(); l != 0 && now-l < int64(Work) {
		return
	}
	if last.Load() == 0 {
		last.Store(now) // the first call starts the clock
		return
	}
	time.Sleep(time.Millisecond)
	last.Store(time.Now().UnixNano())
}
